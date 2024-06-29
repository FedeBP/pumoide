package main

import (
	"github.com/FedeBP/pumoide/backend/api"
	"golang.org/x/time/rate"
	"net/http"
)

type RateLimitedHandler struct {
	handler http.Handler
	limiter *rate.Limiter
}

func (h *RateLimitedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.limiter.Allow() {
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}
	h.handler.ServeHTTP(w, r)
}

func (a *Pumoide) InitRoutes() {
	limiter := rate.NewLimiter(a.config.RateLimit, a.config.RateLimitBurst)

	a.router.Handle("/pumoide-api/collections", &RateLimitedHandler{
		handler: &api.CollectionHandler{DefaultPath: a.config.DefaultCollectionsPath, Logger: a.logger},
		limiter: limiter,
	})

	a.router.Handle("/pumoide-api/execute", &RateLimitedHandler{
		handler: &api.RequestHandler{
			Client:          &http.Client{Timeout: a.config.ClientTimeout},
			EnvironmentPath: a.config.DefaultEnvironmentsPath,
			Logger:          a.logger,
		},
		limiter: limiter,
	})

	a.router.Handle("/pumoide-api/environments", &RateLimitedHandler{
		handler: &api.EnvironmentHandler{DefaultPath: a.config.DefaultEnvironmentsPath, Logger: a.logger},
		limiter: limiter,
	})

	a.router.Handle("/pumoide-api/methods", &RateLimitedHandler{
		handler: &api.MethodHandler{Logger: a.logger},
		limiter: limiter,
	})
}
