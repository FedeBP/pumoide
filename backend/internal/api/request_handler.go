package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/domain"
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
	defer r.Body.Close()

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
	results, err := h.Service.ExecuteRequests(requestDataList, envID)
	if err != nil && err.Error() == constants.ErrFailedAllRequests {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set(constants.ContentType, constants.AppJson)

	response := struct {
		Results []domain.RequestResult `json:"results"`
		Error   string                 `json:"error,omitempty"`
	}{
		Results: results,
	}

	if err != nil {
		response.Error = err.Error()
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Logger.Errorf(constants.ErrFailedToWriteResponse+": %v", err)
	}
}

func (h *RequestHandler) writeResponse(w http.ResponseWriter, results []domain.RequestResult) {
	w.Header().Set(constants.ContentType, constants.AppJson)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		h.Logger.Errorf(constants.ErrFailedToWriteResponse+": %v", err)
	}
}
