package app

import (
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/api"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"github.com/FedeBP/pumoide/backend/internal/services"
)

func (a *Pumoide) InitRoutes() {
	limiter := middleware.NewIPRateLimiter(a.Config.RateLimit, a.Config.RateLimitBurst)

	collectionService := services.NewCollectionService(a.Config.DefaultCollectionsPath, a.Logger)
	collectionHandler := api.NewCollectionHandler(collectionService, a.Logger)
	a.Router.Handle("/pumoide-api/collections", middleware.RateLimitMiddleware(collectionHandler, limiter))

	requestService := services.NewRequestService(a.Logger, a.HistoryManager, a.Config.DefaultEnvironmentsPath)
	requestHandler := api.NewRequestHandler(requestService, a.Logger)
	a.Router.Handle("/pumoide-api/execute", middleware.RateLimitMiddleware(http.HandlerFunc(requestHandler.HandleRequest), limiter))

	environmentService := services.NewEnvironmentService(a.Config.DefaultEnvironmentsPath, a.Logger)
	environmentHandler := api.NewEnvironmentHandler(environmentService, a.Logger)
	a.Router.Handle("/pumoide-api/environments", middleware.RateLimitMiddleware(environmentHandler, limiter))

	methodHandler := api.NewMethodHandler(a.Logger)
	a.Router.Handle("/pumoide-api/methods", middleware.RateLimitMiddleware(methodHandler, limiter))

	if a.HistoryManager != nil {
		historyHandler := api.NewHistoryHandler(a.HistoryManager, a.Logger)
		a.Router.Handle("/pumoide-api/history", middleware.RateLimitMiddleware(historyHandler, limiter))
	}
}
