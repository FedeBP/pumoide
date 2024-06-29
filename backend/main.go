package main

import (
	"log"
)

func main() {
	config := LoadConfig()
	pumoide, err := NewPumoide(config)
	if err != nil {
		log.Fatalf("Failed to start Pumoide service: %v", err)
	}

	pumoide.InitRoutes()

	if err := pumoide.Start(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
