package api

import "net/http"

func InitRoutes() {
	http.HandleFunc("/pumoide-api/collections", handleCollections)
}
