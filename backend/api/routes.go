package api

import (
	"github.com/FedeBP/pumoide/backend/utils"
	"net/http"
)

func InitRoutes() {
	defaultPath := utils.GetDefaultCollectionsPath()
	collectionHandler := NewCollectionHandler(defaultPath)
	http.HandleFunc("/pumoide-api/collections", collectionHandler.Handle)
}
