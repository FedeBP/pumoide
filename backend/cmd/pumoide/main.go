package main

import "github.com/FedeBP/pumoide/backend/internal/app"

func main() {
	pumoide, err := app.NewPumoide()
	if err != nil {
		panic(err)
	}

	pumoide.InitRoutes()

	if err := pumoide.Start(); err != nil {
		pumoide.Logger.Fatalf("Server failed to start: %v", err)
	}
}
