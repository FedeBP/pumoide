package api

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/FedeBP/pumoide/backend/apperrors"
	"github.com/FedeBP/pumoide/backend/models"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RequestHandler struct {
	Client          *http.Client
	EnvironmentPath string
	Logger          *log.Logger
}

func NewRequestHandler(environmentPath string, logger *log.Logger) *RequestHandler {
	return &RequestHandler{
		Client: &http.Client{
			Timeout: time.Second * 30,
		},
		EnvironmentPath: environmentPath,
		Logger:          logger,
	}
}

func (h *RequestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req models.Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusBadRequest, "Invalid request body", err, h.Logger)
		return
	}

	envID := r.URL.Query().Get("env")
	var env *models.Environment
	if envID != "" {
		env, err = models.LoadEnvironment(h.EnvironmentPath, envID)
		if err != nil {
			apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to load environment", err, h.Logger)
			return
		}
	}

	resp, err := h.ExecuteRequest(req, env)
	if err != nil {
		if strings.HasPrefix(err.Error(), "Invalid HTTP method:") {
			apperrors.RespondWithError(w, http.StatusBadRequest, err.Error(), nil, h.Logger)
		} else {
			apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to execute request", err, h.Logger)
		}
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			apperrors.RespondWithError(w, http.StatusInternalServerError, "Error closing the body", err, h.Logger)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to read response body", err, h.Logger)
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

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apperrors.RespondWithError(w, http.StatusInternalServerError, "Failed to encode response", err, h.Logger)
		return
	}
}

func (h *RequestHandler) ExecuteRequest(req models.Request, env *models.Environment) (*http.Response, error) {
	if !req.Method.IsValid() {
		return nil, apperrors.NewAppError(http.StatusBadRequest, "Invalid HTTP method", fmt.Errorf("%s", req.Method))
	}

	parsedURL, err := url.Parse(h.substituteVariables(req.URL, env))
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusBadRequest, "Invalid URL", err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, h.substituteVariables(value, env))
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(string(req.Method), parsedURL.String(), bytes.NewBufferString(h.substituteVariables(req.Body, env)))
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusInternalServerError, "Failed to create request", err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, h.substituteVariables(header.Value, env))
	}

	if req.Auth != nil {
		err = h.applyAuthentication(httpReq, req.Auth, env)
		if err != nil {
			return nil, apperrors.NewAppError(http.StatusInternalServerError, "Failed to apply authentication", err)
		}
	}

	resp, err := h.Client.Do(httpReq)
	if err != nil {
		return nil, apperrors.NewAppError(http.StatusInternalServerError, "Failed to execute request", err)
	}

	return resp, nil
}

func (h *RequestHandler) applyAuthentication(req *http.Request, auth *models.Auth, env *models.Environment) error {
	if auth == nil || auth.Type == models.AuthNone {
		return nil
	}

	switch auth.Type {
	case models.AuthBasic:
		username := h.substituteVariables(auth.Params["username"], env)
		password := h.substituteVariables(auth.Params["password"], env)
		req.SetBasicAuth(username, password)
	case models.AuthBearer:
		token := h.substituteVariables(auth.Params["token"], env)
		req.Header.Set("Authorization", "Bearer "+token)
	case models.AuthAPIKey:
		key := h.substituteVariables(auth.Params["key"], env)
		value := h.substituteVariables(auth.Params["value"], env)
		if auth.Params["in"] == "header" {
			req.Header.Set(key, value)
		} else if auth.Params["in"] == "query" {
			q := req.URL.Query()
			q.Add(key, value)
			req.URL.RawQuery = q.Encode()
		}
	case models.AuthOAuth2:
		token := h.substituteVariables(auth.Params["access_token"], env)
		req.Header.Set("Authorization", "Bearer "+token)
	case models.AuthAWSSigV4:
		accessKey := h.substituteVariables(auth.Params["access_key"], env)
		secretKey := h.substituteVariables(auth.Params["secret_key"], env)
		sessionToken := h.substituteVariables(auth.Params["session_token"], env)
		region := h.substituteVariables(auth.Params["region"], env)
		service := h.substituteVariables(auth.Params["service"], env)

		creds := credentials.NewStaticCredentials(accessKey, secretKey, sessionToken)
		signer := v4.NewSigner(creds)

		_, err := signer.Sign(req, nil, service, region, time.Now())
		if err != nil {
			return apperrors.NewAppError(http.StatusInternalServerError, "Failed to sign request with AWS SigV4", err)
		}
	case models.AuthDigest:
		username := h.substituteVariables(auth.Params["username"], env)
		password := h.substituteVariables(auth.Params["password"], env)
		realm := h.substituteVariables(auth.Params["realm"], env)
		nonce := h.substituteVariables(auth.Params["nonce"], env)
		qop := h.substituteVariables(auth.Params["qop"], env)
		nc := h.substituteVariables(auth.Params["nc"], env)
		cnonce := h.substituteVariables(auth.Params["cnonce"], env)

		ha1 := md5.Sum([]byte(username + ":" + realm + ":" + password))
		ha2 := md5.Sum([]byte(req.Method + ":" + req.URL.Path))
		response := md5.Sum([]byte(fmt.Sprintf("%x:%s:%s:%s:%s:%x", ha1, nonce, nc, cnonce, qop, ha2)))

		auth := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", qop=%s, nc=%s, cnonce="%s", response="%x"`,
			username, realm, nonce, req.URL.Path, qop, nc, cnonce, response)
		req.Header.Set("Authorization", auth)
	case models.AuthNTLM:
		// NTLM authentication is complex and typically requires multiple requests
		// This is a placeholder for NTLM authentication
		return apperrors.NewAppError(http.StatusNotImplemented, "NTLM authentication not implemented", nil)
	default:
		return apperrors.NewAppError(http.StatusBadRequest, "Unknown authentication type", fmt.Errorf("%s", auth.Type))
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
