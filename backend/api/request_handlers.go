package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/FedeBP/pumoide/backend/models"
)

type RequestHandler struct {
	Client *http.Client
}

func NewRequestHandler() *RequestHandler {
	return &RequestHandler{
		Client: &http.Client{
			Timeout: time.Second * 30,
		},
	}
}

func (h *RequestHandler) ExecuteRequest(req models.Request) (*http.Response, error) {
	parsedURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	q := parsedURL.Query()
	for key, value := range req.QueryParams {
		q.Add(key, value)
	}
	parsedURL.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(req.Method, parsedURL.String(), bytes.NewBufferString(req.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for _, header := range req.Headers {
		httpReq.Header.Set(header.Key, header.Value)
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

	resp, err := h.ExecuteRequest(req)
	if err != nil {
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
