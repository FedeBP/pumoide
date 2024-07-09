package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/domain"
	"github.com/FedeBP/pumoide/backend/internal/factory"
	"github.com/FedeBP/pumoide/backend/internal/services"
	"github.com/FedeBP/pumoide/backend/pkg/constants"
	customErrors "github.com/FedeBP/pumoide/backend/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RequestHandler struct {
	Service *services.RequestService
	Logger  *logrus.Logger
}

func NewRequestHandler(service *services.RequestService, logger *logrus.Logger) *RequestHandler {
	return &RequestHandler{
		Service: service,
		Logger:  logger,
	}
}

func (h *RequestHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrFailedToReadRequestBody, err, h.Logger)
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
			customErrors.RespondWithError(w, http.StatusBadRequest, constants.ErrInvalidRequestBody, err, h.Logger)
			return
		}
		requestDataList = []map[string]interface{}{singleRequest}
	}

	envID := r.URL.Query().Get(constants.Env)
	requests := make([]domain.Request, len(requestDataList))
	for i, requestData := range requestDataList {
		request, err := factory.CreateRequest(requestData)
		if err != nil {
			customErrors.RespondWithError(w, http.StatusBadRequest, err.Error(), nil, h.Logger)
			return
		}
		requests[i] = request
	}

	results, err := h.Service.ExecuteRequests(requests, envID)
	if err != nil {
		customErrors.RespondWithError(w, http.StatusInternalServerError, constants.ErrFailedToExecuteRequest, err, h.Logger)
		return
	}

	h.writeResponse(w, results)
}

func (h *RequestHandler) writeResponse(w http.ResponseWriter, results []domain.RequestResult) {
	w.Header().Set(constants.ContentType, constants.AppJson)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(results); err != nil {
		h.Logger.Errorf(constants.ErrFailedToWriteResponse+": %v", err)
	}
}
