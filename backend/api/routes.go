package api

import (
	"github.com/FedeBP/pumoide/backend/utils"
	"net/http"
)

func InitRoutes() {
	defaultCollectionsPath := utils.GetDefaultCollectionsPath()
	collectionHandler := NewCollectionHandler(defaultCollectionsPath)
	http.HandleFunc("/pumoide-api/collections", collectionHandler.Handle)

	defaultEnvironmentsPath := utils.GetDefaultEnvironmentsPath()
	requestHandler := NewRequestHandler(defaultEnvironmentsPath)
	http.HandleFunc("/pumoide-api/execute", requestHandler.Handle)

	environmentHandler := NewEnvironmentHandler(defaultEnvironmentsPath)
	http.HandleFunc("/pumoide-api/environments", environmentHandler.Handle)

	methodHandler := NewMethodHandler()
	http.HandleFunc("/pumoide-api/methods", methodHandler.Handle)
}
