package app_test

import (
	"testing"

	"github.com/FedeBP/pumoide/backend/internal/app"
)

func TestNewPumoide(t *testing.T) {
	pumoide, err := app.NewPumoide()
	if err != nil {
		t.Fatalf("Failed to create new Pumoide instance: %v", err)
	}
	if pumoide == nil {
		t.Fatal("NewPumoide returned nil")
	}
}

func TestInitRoutes(t *testing.T) {
	pumoide, _ := app.NewPumoide()
	pumoide.InitRoutes()
}
