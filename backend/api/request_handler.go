package api

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FedeBP/pumoide/backend/errors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
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
	var requests []models.Request
	err := json.NewDecoder(r.Body).Decode(&requests)
	if err != nil {
		errors.RespondWithError(w, http.StatusBadRequest, utils.InvalidRequestBodyErr, err, h.Logger)
		return
	}

	envID := r.URL.Query().Get(utils.Env)
	var env *models.Environment
	if envID != utils.EmptyString {
		env, err = models.LoadEnvironment(h.EnvironmentPath, envID)
		if err != nil {
			errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToLoadEnvironmentErr, err, h.Logger)
			return
		}
	} else {
		env = &models.Environment{Variables: make(map[string]string)}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := h.ExecuteRequests(ctx, requests, env)
	if err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToExecuteRequestErr, err, h.Logger)
		return
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		errors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

func (h *RequestHandler) ExecuteRequests(ctx context.Context, requests []models.Request, env *models.Environment) ([]models.RequestResult, error) {
	results := make([]models.RequestResult, len(requests))
	completedRequests := make(map[string]bool)

	for i, req := range requests {
		select {
		case <-ctx.Done():
			return results[:i], ctx.Err()
		default:
			for _, depID := range req.DependsOn {
				if !completedRequests[depID] {
					return results[:i], fmt.Errorf("dependency %s not completed for request %s", depID, req.ID)
				}
			}

			result := h.executeRequest(ctx, req, env, results[:i])
			results[i] = result

			completedRequests[req.ID] = true

			if result.Error != "" {
				return results[:i+1], fmt.Errorf("error in request %s: %s", req.ID, result.Error)
			}
		}
	}

	return results, nil
}

func (h *RequestHandler) executeRequest(ctx context.Context, req models.Request, env *models.Environment, previousResults []models.RequestResult) models.RequestResult {
	result := models.RequestResult{
		Request: req,
	}

	req = h.substituteVariables(req, env, previousResults)

	httpReq, err := h.prepareRequest(req, env)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	httpReq = httpReq.WithContext(ctx)

	resp, clientErr := h.Client.Do(httpReq)
	if clientErr != nil {
		result.Error = fmt.Sprintf("%s: %v", utils.FailedToExecuteRequestErr, clientErr)
		return result
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			result.Error = fmt.Sprintf("%s: %v", utils.FailedToCloseBodyErr, err)
			return
		}
	}(resp.Body)

	result.Response.StatusCode = resp.StatusCode
	result.Response.Headers = make(map[string]string)
	for k, v := range resp.Header {
		result.Response.Headers[k] = v[0]
	}

	body, clientErr := io.ReadAll(resp.Body)
	if clientErr != nil {
		result.Error = fmt.Sprintf("%s: %v", utils.FailedToReadResponseErr, clientErr)
	} else {
		result.Response.Body = string(body)
	}

	if result.Error == "" && len(req.ExtractVariables) > 0 {
		for varName, extractPath := range req.ExtractVariables {
			extractedValue, err := extractValueFromResponse(result.Response, extractPath)
			if err == nil {
				env.Variables[varName] = extractedValue
			} else {
				h.Logger.Warnf("Failed to extract variable %s: %v", varName, err)
			}
		}
	}

	return result
}

func (h *RequestHandler) substituteVariables(req models.Request, env *models.Environment, previousResults []models.RequestResult) models.Request {
	substituteFunc := func(input string) string {
		for key, value := range env.Variables {
			input = strings.ReplaceAll(input, "{{"+key+"}}", value)
		}

		for _, prevResult := range previousResults {
			for varName, extractPath := range prevResult.Request.ExtractVariables {
				extractedValue, err := extractValueFromResponse(prevResult.Response, extractPath)
				if err == nil {
					input = strings.ReplaceAll(input, "{{"+varName+"}}", extractedValue)
				}
			}
		}

		return input
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
		return nil, errors.NewAppError(http.StatusBadRequest, utils.InvalidHTTPMethodErr, nil)
	}

	parsedURL, err := url.Parse(substituteVariables(req.URL, env))
	if err != nil {
		return nil, errors.NewAppError(http.StatusBadRequest, utils.InvalidURLErr, err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, substituteVariables(value, env))
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(string(req.Method), parsedURL.String(), bytes.NewBufferString(substituteVariables(req.Body, env)))
	if err != nil {
		return nil, errors.NewAppError(http.StatusInternalServerError, utils.FailedToCreateRequestErr, err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, substituteVariables(header.Value, env))
	}

	if req.Auth != nil {
		err = applyAuthentication(httpReq, req.Auth, env)
		if err != nil {
			return nil, errors.NewAppError(http.StatusForbidden, utils.FailedToAuthenticateErr, err)
		}
	}

	return httpReq, nil
}

func applyAuthentication(req *http.Request, auth *models.Auth, env *models.Environment) error {
	if auth == nil || auth.Type == models.AuthNone {
		return nil
	}

	switch auth.Type {
	case models.AuthBasic:
		username := substituteVariables(auth.Params[utils.Username], env)
		password := substituteVariables(auth.Params[utils.Password], env)
		req.SetBasicAuth(username, password)

	case models.AuthBearer:
		token := substituteVariables(auth.Params[utils.Token], env)
		req.Header.Set(utils.Authorization, utils.Bearer+token)

	case models.AuthAPIKey:
		key := substituteVariables(auth.Params[utils.Key], env)
		value := substituteVariables(auth.Params[utils.Value], env)
		if auth.Params[utils.In] == utils.Header {
			req.Header.Set(key, value)
		} else if auth.Params[utils.In] == utils.Query {
			q := req.URL.Query()
			q.Add(key, value)
			req.URL.RawQuery = q.Encode()
		}

	case models.AuthOAuth2:
		token := substituteVariables(auth.Params[utils.AccessToken], env)
		req.Header.Set(utils.Authorization, utils.Bearer+token)

	case models.AuthAWSSigV4:
		accessKey := substituteVariables(auth.Params[utils.AccessKey], env)
		secretKey := substituteVariables(auth.Params[utils.SecretKey], env)
		sessionToken := substituteVariables(auth.Params[utils.SessionToken], env)
		region := substituteVariables(auth.Params[utils.Region], env)
		service := substituteVariables(auth.Params[utils.Service], env)

		creds := credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)
		signer := v4.NewSigner(creds)

		_, err := signer.Sign(req, nil, service, region, time.Now())
		if err != nil {
			return errors.NewAppError(http.StatusInternalServerError, utils.FailedAwsSigV4Err, err)
		}

	case models.AuthDigest:
		username := substituteVariables(auth.Params[utils.Username], env)
		password := substituteVariables(auth.Params[utils.Password], env)
		realm := substituteVariables(auth.Params[utils.Realm], env)
		nonce := substituteVariables(auth.Params[utils.Nonce], env)
		qop := substituteVariables(auth.Params[utils.Qop], env)
		nc := substituteVariables(auth.Params[utils.NC], env)
		cnonce := substituteVariables(auth.Params[utils.Cnonce], env)

		ha1 := md5.Sum([]byte(username + ":" + realm + ":" + password))
		ha2 := md5.Sum([]byte(req.Method + ":" + req.URL.Path))
		response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%s:%s:%s:%x", ha1, nonce, nc, cnonce, qop, ha2)))

		auth := fmt.Sprintf(utils.DigestAuth, username, realm, nonce, req.URL.Path, qop, nc, cnonce, response)
		req.Header.Set(utils.Authorization, auth)

	default:
		return errors.NewAppError(http.StatusBadRequest, utils.UnknownAuthErr, fmt.Errorf("%s", auth.Type))
	}

	return nil
}

func substituteVariables(input string, env *models.Environment) string {
	if env == nil {
		return input
	}
	for key, value := range env.Variables {
		input = strings.ReplaceAll(input, "{{"+key+"}}", value)
	}
	return input
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
