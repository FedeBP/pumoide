package api

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/factory"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/sirupsen/logrus"
)

type RequestHandler struct {
	Logger          *logrus.Logger
	HistoryManager  *models.History
	EnvironmentPath string
}

func (h *RequestHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(r.Body)

	var requestDataList []map[string]interface{}
	err = json.Unmarshal(body, &requestDataList)

	if err != nil {
		var singleRequest map[string]interface{}
		err = json.Unmarshal(body, &singleRequest)
		if err != nil {
			h.respondWithError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
			return
		}
		requestDataList = []map[string]interface{}{singleRequest}
	}

	env := h.loadEnvironment(r)
	requests := make([]domain.Request, len(requestDataList))
	for i, requestData := range requestDataList {
		request, err := factory.CreateRequest(requestData)
		if err != nil {
			h.respondWithError(w, http.StatusBadRequest, err.Error())
			return
		}
		requests[i] = request
	}

	results := h.executeRequests(requests, env)

	h.writeResponse(w, results)
}

func (h *RequestHandler) executeRequests(requests []domain.Request, env *domain.Environment) []domain.RequestResult {
	results := make([]domain.RequestResult, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	completed := make(map[string]bool)

	for {
		readyRequests := h.getReadyRequests(requests, completed)
		if len(readyRequests) == 0 {
			break
		}

		for _, req := range readyRequests {
			wg.Add(1)
			go func(r domain.Request) {
				defer wg.Done()

				h.preprocessRequest(r, env)

				startTime := time.Now()
				response, err := r.Execute(env)
				executionTime := time.Since(startTime)

				result := domain.RequestResult{
					Request:  r,
					Response: response,
					PerformanceMetrics: domain.PerformanceMetrics{
						Total: executionTime,
					},
				}

				if err != nil {
					result.Error = err.Error()
				} else {
					h.postprocessRequest(r, response, env)
				}

				mu.Lock()
				results[h.findRequestIndex(requests, r.GetID())] = result
				completed[r.GetID()] = true
				mu.Unlock()

				h.saveToHistory(r, response, executionTime)
			}(req)
		}

		wg.Wait()
	}

	return results
}

func (h *RequestHandler) getReadyRequests(requests []domain.Request, completed map[string]bool) []domain.Request {
	var readyRequests []domain.Request
	for _, req := range requests {
		if completed[req.GetID()] {
			continue
		}
		ready := true
		for _, depID := range req.GetDependsOn() {
			if !completed[depID] {
				ready = false
				break
			}
		}
		if ready {
			readyRequests = append(readyRequests, req)
		}
	}
	return readyRequests
}

func (h *RequestHandler) findRequestIndex(requests []domain.Request, id string) int {
	for i, req := range requests {
		if req.GetID() == id {
			return i
		}
	}
	return -1
}

func (h *RequestHandler) loadEnvironment(r *http.Request) *domain.Environment {
	envID := r.URL.Query().Get("env")
	if envID == "" {
		return &domain.Environment{Variables: make(map[string]string)}
	}

	env, err := domain.LoadEnvironment(h.EnvironmentPath, envID)
	if err != nil {
		h.Logger.Warnf("Failed to load environment %s: %v", envID, err)
		return &domain.Environment{Variables: make(map[string]string)}
	}

	return env
}

func (h *RequestHandler) preprocessRequest(req domain.Request, env *domain.Environment) {
	utils.SubstituteRequestVariables(req, env)
}

func (h *RequestHandler) postprocessRequest(req domain.Request, resp domain.Response, env *domain.Environment) {
	for varName, extractPath := range req.GetExtractVariables() {
		extractedValue, err := utils.ExtractValueFromResponse(resp, extractPath)
		if err == nil {
			env.Variables[varName] = extractedValue
		} else {
			h.Logger.Warnf("Failed to extract variable %s: %v", varName, err)
		}
	}
}

func (h *RequestHandler) saveToHistory(req domain.Request, resp domain.Response, executionTime time.Duration) {
	if h.HistoryManager == nil {
		return
	}

	entry := domain.HistoryEntry{
		Timestamp:     time.Now(),
		Request:       req,
		Response:      resp,
		ExecutionTime: executionTime,
	}

	if err := h.HistoryManager.AddEntry(entry); err != nil {
		h.Logger.Warnf("Failed to add history entry: %v", err)
	}
}

func (h *RequestHandler) writeResponse(w http.ResponseWriter, results []domain.RequestResult) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(results); err != nil {
		h.Logger.Errorf("Failed to write response: %v", err)
	}
}

func (h *RequestHandler) respondWithError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Errorf("Failed to write error response: %v", err)
	}
}
