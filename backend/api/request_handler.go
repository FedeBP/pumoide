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
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req models.Request
	err := json.NewDecoder(r.Body).Decode(&req)
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

	resp, err := h.ExecuteRequest(req, env)
	if err != nil {
		if strings.HasPrefix(err.Error(), utils.InvalidHTTPMethodErr) {
			apperrors.RespondWithError(w, http.StatusBadRequest, err.Error(), nil, h.Logger)
		} else {
			apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToExecuteRequestErr, err, h.Logger)
		}
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToCloseBodyErr, err, h.Logger)
			return
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToReadResponseErr, err, h.Logger)
		return
	}

	response := struct {
		StatusCode int               `json:"statusCode"`
		Headers    map[string]string `json:"headers"`
		Body       string            `json:"body"`
	}{
		StatusCode: resp.StatusCode,
		Headers:    make(map[string]string),
		Body:       string(body),
	}

	for k, v := range resp.Header {
		response.Headers[k] = v[0]
	}

	w.Header().Set(utils.ContentType, utils.AppJson)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, utils.FailedToWriteResponseErr, err, h.Logger)
		return
	}
}

func (h *RequestHandler) ExecuteRequest(req models.Request, env *models.Environment) (*http.Response, error) {
	if !req.Method.IsValid() {
		return nil, apperrors.NewAppError(http.StatusBadRequest, utils.InvalidHTTPMethodErr, fmt.Errorf("%s", req.Method))
	}

	parsedURL, err := url.Parse(h.substituteVariables(req.URL, env))
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusBadRequest, utils.InvalidURLErr, err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, h.substituteVariables(value, env))
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(string(req.Method), parsedURL.String(), bytes.NewBufferString(h.substituteVariables(req.Body, env)))
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusInternalServerError, utils.FailedToCreateRequestErr, err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, h.substituteVariables(header.Value, env))
	}

	if req.Auth != nil {
		err = h.applyAuthentication(httpReq, req.Auth, env)
		if err != nil {
			return nil, apperrors.NewAppError(http.StatusInternalServerError, utils.FailedToAuthenticateErr, err)
		}
	}

	resp, err := h.Client.Do(httpReq)
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusInternalServerError, utils.FailedToExecuteRequestErr, err)
	}

	return resp, nil
}

func (h *RequestHandler) applyAuthentication(req *http.Request, auth *models.Auth, env *models.Environment) error {
	if auth == nil || auth.Type == models.AuthNone {
		return nil
	}

	switch auth.Type {
	case models.AuthBasic:
		username := h.substituteVariables(auth.Params[utils.Username], env)
		password := h.substituteVariables(auth.Params[utils.Password], env)
		req.SetBasicAuth(username, password)

	case models.AuthBearer:
		token := h.substituteVariables(auth.Params[utils.Token], env)
		req.Header.Set(utils.Authorization, utils.Bearer+token)

	case models.AuthAPIKey:
		key := h.substituteVariables(auth.Params[utils.Key], env)
		value := h.substituteVariables(auth.Params[utils.Value], env)
		if auth.Params[utils.In] == utils.Header {
			req.Header.Set(key, value)
		} else if auth.Params[utils.In] == utils.Query {
			q := req.URL.Query()
			q.Add(key, value)
			req.URL.RawQuery = q.Encode()
		}

	case models.AuthOAuth2:
		token := h.substituteVariables(auth.Params[utils.AccessToken], env)
		req.Header.Set(utils.Authorization, utils.Bearer+token)

	case models.AuthAWSSigV4:
		accessKey := h.substituteVariables(auth.Params[utils.AccessKey], env)
		secretKey := h.substituteVariables(auth.Params[utils.SecretKey], env)
		sessionToken := h.substituteVariables(auth.Params[utils.SessionToken], env)
		region := h.substituteVariables(auth.Params[utils.Region], env)
		service := h.substituteVariables(auth.Params[utils.Service], env)

		creds := credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)
		signer := v4.NewSigner(creds)

		_, err := signer.Sign(req, nil, service, region, time.Now())
		if err != nil {
			return apperrors.NewAppError(http.StatusInternalServerError, utils.FailedAwsSigV4Err, err)
		}

	case models.AuthDigest:
		username := h.substituteVariables(auth.Params[utils.Username], env)
		password := h.substituteVariables(auth.Params[utils.Password], env)
		realm := h.substituteVariables(auth.Params[utils.Realm], env)
		nonce := h.substituteVariables(auth.Params[utils.Nonce], env)
		qop := h.substituteVariables(auth.Params[utils.Qop], env)
		nc := h.substituteVariables(auth.Params[utils.NC], env)
		cnonce := h.substituteVariables(auth.Params[utils.Cnonce], env)

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

func (h *RequestHandler) substituteVariables(input string, env *models.Environment) string {
	if env == nil {
		return input
	}
	for key, value := range env.Variables {
		input = strings.ReplaceAll(input, "{{"+key+"}}", value)
	}
	return input
}
