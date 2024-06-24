package main

import (
	"github.com/FedeBP/pumoide/backend/api"
	"github.com/FedeBP/pumoide/backend/utils"
	"log"
	"net/http"
)

func main() {
	log.Printf("Storage location: %s", utils.GetCurrentStorageLocation())

	err := utils.EnsureDir(utils.GetDefaultCollectionsPath())
	if err != nil {
		log.Fatalf("Failed to create collections directory: %v", err)
	}

	err = utils.EnsureDir(utils.GetDefaultEnvironmentsPath())
	if err != nil {
		log.Fatalf("Failed to create environments directory: %v", err)
	}

	api.InitRoutes()

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
