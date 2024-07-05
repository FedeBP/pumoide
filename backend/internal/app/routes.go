package app

import (
	"net/http"

	"github.com/FedeBP/pumoide/backend/internal/api"
	"github.com/FedeBP/pumoide/backend/internal/middleware"
)

func (a *Pumoide) InitRoutes() {
	limiter := middleware.NewIPRateLimiter(a.Config.RateLimit, a.Config.RateLimitBurst)

	a.Router.Handle("/pumoide-api/collections", middleware.RateLimitMiddleware(
		&api.CollectionHandler{DefaultPath: a.Config.DefaultCollectionsPath, Logger: a.Logger},
		limiter,
	))

	a.Router.Handle("/pumoide-api/execute", middleware.RateLimitMiddleware(
		&api.RequestHandler{
			Client:          &http.Client{Timeout: a.Config.ClientTimeout},
			EnvironmentPath: a.Config.DefaultEnvironmentsPath,
			Logger:          a.Logger,
			WorkerCount:     a.Config.Workers,
		},
		limiter,
	))

	a.Router.Handle("/pumoide-api/environments", middleware.RateLimitMiddleware(
		&api.EnvironmentHandler{DefaultPath: a.Config.DefaultEnvironmentsPath, Logger: a.Logger},
		limiter,
	))

	a.Router.Handle("/pumoide-api/methods", middleware.RateLimitMiddleware(
		&api.MethodHandler{Logger: a.Logger},
		limiter,
	))
}
