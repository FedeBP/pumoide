package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/internal/validators"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	"github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/oliveagle/jsonpath"
	"github.com/sirupsen/logrus"
)

type RequestHandler struct {
	Client          *http.Client
	EnvironmentPath string
	Logger          *logrus.Logger
	WorkerCount     int
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadRequestBody, err, h.Logger)
		return
	}
	err = r.Body.Close()
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToCloseBody, err, h.Logger)
		return
	}

	var requests []models.Request
	err = json.Unmarshal(bodyBytes, &requests)
	if err != nil {
		var singleRequest models.Request
		err = json.Unmarshal(bodyBytes, &singleRequest)
		if err != nil {
			errors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequestBody, err, h.Logger)
			return
		}
		requests = []models.Request{singleRequest}
	}

	envID := r.URL.Query().Get(constants.Env)
	var env *models.Environment
	if envID != constants.EmptyString {
		env, err = models.LoadEnvironment(h.EnvironmentPath, envID)
		if err != nil {
			errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToLoadEnvironment, err, h.Logger)
			return
		}
	} else {
		env = &models.Environment{Variables: make(map[string]string)}
	}

	results, err := h.ExecuteRequests(requests, env)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToExecuteRequest, err, h.Logger)
		return
	}

	w.Header().Set(constants.ContentType, constants.AppJson)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToWriteResponse, err, h.Logger)
	}
}

func (h *RequestHandler) ExecuteRequests(requests []models.Request, env *models.Environment) ([]models.RequestResult, error) {
	results := make([]models.RequestResult, len(requests))
	completedRequests := make(map[string]bool)
	allFailed := true

	for i, req := range requests {
		for _, depID := range req.DependsOn {
			if !completedRequests[depID] {
				return results[:i], fmt.Errorf("dependency %s not completed for request %s", depID, req.ID)
			}
		}

		result := h.ExecuteRequest(req, env, results[:i])
		results[i] = result

		completedRequests[req.ID] = true

		if result.Error == "" && len(result.ValidationErrors) == 0 {
			allFailed = false
		}

		if result.Error != "" {
			return results[:i+1], fmt.Errorf("error in request %s: %s", req.ID, result.Error)
		}
	}

	if allFailed {
		return results, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedAllRequests, nil)
	}

	return results, nil
}

func (h *RequestHandler) ExecuteRequest(req models.Request, env *models.Environment, previousResults []models.RequestResult) models.RequestResult {
	result := models.RequestResult{
		Request: req,
	}

	req = h.substituteVariables(req, env, previousResults)

	httpReq, err := h.prepareRequest(req, env)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	resp, clientErr := h.Client.Do(httpReq)
	if clientErr != nil {
		result.Error = fmt.Sprintf("%s: %v", constants.ErrFailedToExecuteRequest, clientErr)
		return result
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			h.Logger.Warnf("%s: %v", constants.ErrFailedToCloseBody, err)
		}
	}()

	body, clientErr := io.ReadAll(resp.Body)
	if clientErr != nil {
		result.Error = fmt.Sprintf("%s: %v", constants.ErrFailedToReadResponse, clientErr)
		return result
	}

	result.Response = models.Response{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
		Body:       string(body),
	}

	for k, v := range resp.Header {
		result.Response.Headers[k] = v[0]
	}

	if len(req.ExtractVariables) > 0 {
		for varName, extractPath := range req.ExtractVariables {
			extractedValue, err := extractValueFromResponse(result.Response, extractPath)
			if err == nil {
				env.Variables[varName] = extractedValue
			} else {
				h.Logger.Warnf("Failed to extract variable %s: %v", varName, err)
			}
		}
	}

	if req.ResponseValidation != nil {
		validationErrors := validators.ValidateResponse(result.Response, req.ResponseValidation)
		if len(validationErrors) > 0 {
			result.ValidationErrors = validationErrors
		}
	}

	return result
}

func (h *RequestHandler) substituteVariables(req models.Request, env *models.Environment, previousResults []models.RequestResult) models.Request {
	if env == nil {
		env = &models.Environment{Variables: make(map[string]string)}
	}

	combinedEnv := &models.Environment{
		Variables: make(map[string]string),
	}

	for k, v := range env.Variables {
		combinedEnv.Variables[k] = v
	}

	for _, prevResult := range previousResults {
		if prevResult.Request.ExtractVariables != nil {
			for varName, extractPath := range prevResult.Request.ExtractVariables {
				extractedValue, err := extractValueFromResponse(prevResult.Response, extractPath)
				if err == nil {
					combinedEnv.Variables[varName] = extractedValue
				}
			}
		}
	}

	substituteFunc := func(input string) string {
		return utils.SubstituteVariables(input, combinedEnv)
	}

	req.URL = substituteFunc(req.URL)
	req.Body = substituteFunc(req.Body)
	for i, header := range req.Headers {
		req.Headers[i].Value = substituteFunc(header.Value)
	}
	for key, value := range req.QueryParams {
		req.QueryParams[key] = substituteFunc(value)
	}

	return req
}

func (h *RequestHandler) prepareRequest(req models.Request, env *models.Environment) (*http.Request, *errors.AppError) {
	if !req.Method.IsValid() {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidHTTPMethod, nil)
	}

	parsedURL, err := url.Parse(utils.SubstituteVariables(req.URL, env))
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, constants.ErrInvalidURL, err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, utils.SubstituteVariables(value, env))
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(string(req.Method), parsedURL.String(), bytes.NewBufferString(utils.SubstituteVariables(req.Body, env)))
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, constants.ErrFailedToCreateRequest, err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, utils.SubstituteVariables(header.Value, env))
	}

	if req.Auth != nil {
		err = validators.ApplyAuthentication(httpReq, req.Auth, env)
		if err != nil {
			return nil, errors.NewAppError(http.StatusForbidden, constants.ErrFailedToAuthenticate, err)
		}
	}

	return httpReq, nil
}

func extractValueFromResponse(response models.Response, extractPath string) (string, error) {
	if strings.HasPrefix(extractPath, "$") {
		return extractJSONValue(response.Body, extractPath)
	} else {
		return extractRegexValue(response.Body, extractPath)
	}
}

func extractJSONValue(responseBody string, jsonPath string) (string, error) {
	var data interface{}
	err := json.Unmarshal([]byte(responseBody), &data)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON: %v", err)
	}

	result, err := jsonpath.JsonPathLookup(data, jsonPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract value using JSON path: %v", err)
	}

	switch v := result.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func extractRegexValue(responseBody string, regexPattern string) (string, error) {
	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %v", err)
	}

	matches := re.FindStringSubmatch(responseBody)
	if len(matches) > 1 {
		return matches[1], nil
	} else if len(matches) == 1 {
		return matches[0], nil
	}

	return "", fmt.Errorf("no match found for regex pattern")
}
