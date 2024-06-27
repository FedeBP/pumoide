package api

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
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
}

func NewRequestHandler(environmentPath string) *RequestHandler {
	return &RequestHandler{
		Client: &http.Client{
			Timeout: time.Second * 30,
		},
		EnvironmentPath: environmentPath,
	}
}

func (h *RequestHandler) ExecuteRequest(req models.Request, env *models.Environment) (*http.Response, error) {
	if !req.Method.IsValid() {
		return nil, fmt.Errorf("invalid HTTP method: %s", req.Method)
	}

	parsedURL, err := url.Parse(h.substituteVariables(req.URL, env))
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, h.substituteVariables(value, env))
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(string(req.Method), parsedURL.String(), bytes.NewBufferString(h.substituteVariables(req.Body, env)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, h.substituteVariables(header.Value, env))
	}

	if req.Auth != nil {
		err = h.applyAuthentication(httpReq, req.Auth, env)
		if err != nil {
			return nil, fmt.Errorf("failed to apply authentication: %w", err)
		}
	}

	resp, err := h.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	return resp, nil
}

func (h *RequestHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var req models.Request
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	envID := r.URL.Query().Get("env")
	var env *models.Environment
	if envID != "" {
		env, err = models.LoadEnvironment(h.EnvironmentPath, envID)
		if err != nil {
			http.Error(w, "Failed to load environment: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	resp, err := h.ExecuteRequest(req, env)
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid HTTP method:") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Failed to execute request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Error closing response body: %v", closeErr)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response body: "+err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
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
			return fmt.Errorf("failed to sign request with AWS SigV4: %w", err)
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
		return fmt.Errorf("NTLM authentication not implemented")
	default:
		return fmt.Errorf("unknown authentication type: %s", auth.Type)
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
