package api

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/FedeBP/pumoide/backend/utils"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
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
		apperrors.RespondWithError(w, http.StatusBadRequest, utils.InvalidRequestBodyErr, err, h.Logger)
		return
	}

	envID := r.URL.Query().Get(utils.Env)
	var env *models.Environment
	if envID != utils.EmptyString {
		env, err = models.LoadEnvironment(h.EnvironmentPath, envID)
		if err != nil {
			apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToLoadEnvironmentErr, err, h.Logger)
			return
		}
	}

	results := h.ExecuteRequests(requests, env)

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(results); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
	}
}

func (h *RequestHandler) ExecuteRequests(requests []models.Request, env *models.Environment) []models.RequestResult {
	resultChan := make(chan models.RequestResult, len(requests))
	var wg sync.WaitGroup

	semaphore := make(chan struct{}, h.WorkerCount)

	for _, req := range requests {
		wg.Add(1)
		go func(r models.Request) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := h.executeRequest(r, env)
			resultChan <- result
		}(req)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var results []models.RequestResult
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

func (h *RequestHandler) executeRequest(req models.Request, env *models.Environment) models.RequestResult {
	result := models.RequestResult{
		Request: req,
	}

	httpReq, err := h.prepareRequest(req, env)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	resp, intError := h.Client.Do(httpReq)
	if intError != nil {
		result.Error = fmt.Sprintf("%s: %v", utils.FailedToExecuteRequestErr, intError)
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

	body, intError := io.ReadAll(resp.Body)
	if intError != nil {
		result.Error = fmt.Sprintf("%s: %v", utils.FailedToReadResponseErr, intError)
	} else {
		result.Response.Body = string(body)
	}

	return result
}

func (h *RequestHandler) prepareRequest(req models.Request, env *models.Environment) (*http.Request, *apperrors.AppError) {
	if !req.Method.IsValid() {
		return nil, apperrors.NewAppError(http.StatusBadRequest, utils.InvalidHTTPMethodErr, nil)
	}

	parsedURL, err := url.Parse(substituteVariables(req.URL, env))
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusBadRequest, utils.InvalidURLErr, err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, substituteVariables(value, env))
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(string(req.Method), parsedURL.String(), bytes.NewBufferString(substituteVariables(req.Body, env)))
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusInternalServerError, utils.FailedToCreateRequestErr, err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, substituteVariables(header.Value, env))
	}

	if req.Auth != nil {
		err = applyAuthentication(httpReq, req.Auth, env)
		if err != nil {
			return nil, apperrors.NewAppError(http.StatusForbidden, utils.FailedToAuthenticateErr, err)
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
			return apperrors.NewAppError(http.StatusInternalServerError, utils.FailedAwsSigV4Err, err)
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
		return apperrors.NewAppError(http.StatusBadRequest, utils.UnknownAuthErr, fmt.Errorf("%s", auth.Type))
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
