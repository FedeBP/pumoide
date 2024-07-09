package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type RequestService struct {
	Logger          *logrus.Logger
	HistoryManager  *models.History
	EnvironmentPath string
}

func NewRequestService(logger *logrus.Logger, historyManager *models.History, environmentPath string) *RequestService {
	return &RequestService{
		Logger:          logger,
		HistoryManager:  historyManager,
		EnvironmentPath: environmentPath,
	}
}

func (s *RequestService) ExecuteRequests(requests []domain.Request, envID string) ([]domain.RequestResult, error) {
	env := s.loadEnvironment(envID)
	results := make([]domain.RequestResult, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex

	completed := make(map[string]bool)

	for {
		readyRequests := s.getReadyRequests(requests, completed)
		if len(readyRequests) == 0 {
			break
		}

		for _, req := range readyRequests {
			wg.Add(1)
			go func(r domain.Request) {
				defer wg.Done()

				s.PreprocessRequest(r, env)

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
					s.PostprocessRequest(r, response, env)
				}

				mu.Lock()
				results[s.findRequestIndex(requests, r.GetID())] = result
				completed[r.GetID()] = true
				mu.Unlock()

				s.saveToHistory(r, response, executionTime)
			}(req)
		}

		wg.Wait()
	}

	return results, nil
}

func (s *RequestService) loadEnvironment(envID string) *domain.Environment {
	if envID == "" {
		return &domain.Environment{Variables: make(map[string]string)}
	}

	env, err := domain.LoadEnvironment(s.EnvironmentPath, envID)
	if err != nil {
		s.Logger.Warnf("Failed to load environment %s: %v", envID, err)
		return &domain.Environment{Variables: make(map[string]string)}
	}

	return env
}

func (s *RequestService) getReadyRequests(requests []domain.Request, completed map[string]bool) []domain.Request {
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

func (s *RequestService) findRequestIndex(requests []domain.Request, id string) int {
	for i, req := range requests {
		if req.GetID() == id {
			return i
		}
	}
	return -1
}

func (s *RequestService) PreprocessRequest(req domain.Request, env *domain.Environment) {
	utils.SubstituteRequestVariables(req, env)
}

func (s *RequestService) PostprocessRequest(req domain.Request, resp domain.Response, env *domain.Environment) {
	for varName, extractPath := range req.GetExtractVariables() {
		extractedValue, err := utils.ExtractValueFromResponse(resp, extractPath)
		if err == nil {
			env.Variables[varName] = extractedValue
		} else {
			s.Logger.Warnf("Failed to extract variable %s: %v", varName, err)
		}
	}
}

func (s *RequestService) saveToHistory(req domain.Request, resp domain.Response, executionTime time.Duration) {
	if s.HistoryManager == nil {
		return
	}

	requestMap, err := convertToMap(req)
	if err != nil {
		s.Logger.Warnf("Failed to convert request to map: %v", err)
		return
	}

	responseMap, err := convertToMap(resp)
	if err != nil {
		s.Logger.Warnf("Failed to convert response to map: %v", err)
		return
	}

	entry := domain.HistoryEntry{
		ID:            uuid.New().String(),
		Timestamp:     time.Now(),
		Request:       requestMap,
		Response:      responseMap,
		ExecutionTime: executionTime,
	}

	if err := s.HistoryManager.AddEntry(entry); err != nil {
		s.Logger.Warnf("Failed to add history entry: %v", err)
	}
}

func convertToMap(v interface{}) (map[string]interface{}, error) {
	var result map[string]interface{}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
}
