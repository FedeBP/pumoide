package main

import (
	"github.com/FedeBP/pumoide/backend/api"
	"github.com/FedeBP/pumoide/backend/utils"
	"log"
	"net/http"
	"os"
)

var logger *log.Logger

func init() {
	logFile, err := os.OpenFile("pumoide.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	logger = log.New(logFile, "API: ", log.Ldate|log.Ltime|log.Lshortfile)
}

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

	api.InitRoutes(logger)

	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
