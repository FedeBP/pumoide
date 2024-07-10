package services

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/models"
	"github.com/FedeBP/pumoide/backend/internal/utils"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
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

func (s *RequestService) ExecuteRequests(requestDataList []map[string]interface{}, envID string) ([]domain.RequestResult, error) {
	env := s.loadEnvironment(envID)
	requests := make([]domain.Request, len(requestDataList))
	results := make([]domain.RequestResult, len(requestDataList))
	var wg sync.WaitGroup
	var mu sync.Mutex

	allFailed := true

	for i, requestData := range requestDataList {
		request, err := models.CreateRequest(requestData)
		if err != nil {
			s.Logger.Errorf("Failed to create request: %v", err)
			results[i] = domain.RequestResult{Error: err.Error()}
			continue
		}
		requests[i] = request
	}

	for i, req := range requests {
		if req == nil {
			continue
		}

		wg.Add(1)
		go func(i int, r domain.Request) {
			defer wg.Done()

			result := s.executeRequest(r, env, results)

			mu.Lock()
			results[i] = result
			if result.Error == constants.EmptyString {
				allFailed = false
			}
			mu.Unlock()

			s.saveToHistory(r, result.Response, result.PerformanceMetrics.Total)
		}(i, req)
	}

	wg.Wait()

	var err error
	if allFailed {
		err = fmt.Errorf(constants.ErrFailedAllRequests)
	}

	return results, err
}

func (s *RequestService) executeRequest(req domain.Request, env *domain.Environment, results []domain.RequestResult) domain.RequestResult {
	for _, depID := range req.GetDependsOn() {
		for _, result := range results {
			if result.Request == nil {
				continue
			}

			resultID := result.Request.GetID()
			if resultID == constants.EmptyString {
				s.Logger.Warnf("Request in results doesn't have an ID. Can't match dependency: %s", depID)
				continue
			}

			if resultID == depID && result.Error != constants.EmptyString {
				return domain.RequestResult{
					Request: req,
					Error:   fmt.Sprintf("Dependent request %s failed", depID),
				}
			}
		}
	}

	s.PreprocessRequest(req, env)

	var result domain.RequestResult
	result.Request = req

	start := time.Now()
	var connect, dns, tlsHandshake time.Time
	var serverProcessing time.Duration

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	trace := &httptrace.ClientTrace{
		DNSStart: func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			result.PerformanceMetrics.DNSLookup = time.Since(dns)
		},
		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			result.PerformanceMetrics.TCPConnection = time.Since(connect)
		},
		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			result.PerformanceMetrics.TLSHandshake = time.Since(tlsHandshake)
		},
		GotFirstResponseByte: func() {
			serverProcessing = time.Since(start)
			result.PerformanceMetrics.ServerProcessing = serverProcessing
		},
	}

	ctx := httptrace.WithClientTrace(req.GetContext(), trace)

	response, err := req.Execute(ctx, env, transport)

	result.Response = response
	if err != nil {
		result.Error = err.Error()
	} else {
		s.PostprocessRequest(req, response, env)
	}

	result.PerformanceMetrics.Total = time.Since(start)
	result.PerformanceMetrics.ContentTransfer = result.PerformanceMetrics.Total -
		(result.PerformanceMetrics.DNSLookup +
			result.PerformanceMetrics.TCPConnection +
			result.PerformanceMetrics.TLSHandshake +
			result.PerformanceMetrics.ServerProcessing)

	return result
}

func (s *RequestService) getReadyRequests(requests []domain.Request, completed, failed map[string]bool) []domain.Request {
	var readyRequests []domain.Request
	for _, req := range requests {
		if completed[req.GetID()] {
			continue
		}
		ready := true
		for _, depID := range req.GetDependsOn() {
			if !completed[depID] || failed[depID] {
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

func (s *RequestService) loadEnvironment(envID string) *domain.Environment {
	if envID == constants.EmptyString {
		return &domain.Environment{Variables: make(map[string]string)}
	}

	env, err := domain.LoadEnvironment(s.EnvironmentPath, envID)
	if err != nil {
		s.Logger.Warnf("Failed to load environment %s: %v", envID, err)
		return &domain.Environment{Variables: make(map[string]string)}
	}

	return env
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

	if graphqlReq, ok := req.(*models.GraphQLRequest); ok {
		graphqlReq.Query = utils.SubstituteVariables(graphqlReq.Query, env)
		for key, value := range graphqlReq.Variables {
			if strValue, ok := value.(string); ok {
				graphqlReq.Variables[key] = utils.SubstituteVariables(strValue, env)
			}
		}
	}
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
