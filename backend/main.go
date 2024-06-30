package main

func main() {
	pumoide, err := InitPumoide()
	if err != nil {
		pumoide.logger.Fatalf("Failed to start Pumoide service: %v", err)
	}

	pumoide.InitRoutes()

	if err := pumoide.Start(); err != nil {
		pumoide.logger.Fatalf("Server failed to start: %v", err)
	}
}
