package api

import (
	"github.com/FedeBP/pumoide/backend/utils"
	"log"
	"net/http"
)

func InitRoutes(logger *log.Logger) {
	defaultCollectionsPath := utils.GetDefaultCollectionsPath()
	collectionHandler := NewCollectionHandler(defaultCollectionsPath, logger)
	http.HandleFunc("/pumoide-api/collections", collectionHandler.Handle)

	defaultEnvironmentsPath := utils.GetDefaultEnvironmentsPath()
	requestHandler := NewRequestHandler(defaultEnvironmentsPath, logger)
	http.HandleFunc("/pumoide-api/execute", requestHandler.Handle)

	environmentHandler := NewEnvironmentHandler(defaultEnvironmentsPath, logger)
	http.HandleFunc("/pumoide-api/environments", environmentHandler.Handle)

	methodHandler := NewMethodHandler(logger)
	http.HandleFunc("/pumoide-api/methods", methodHandler.Handle)
}
