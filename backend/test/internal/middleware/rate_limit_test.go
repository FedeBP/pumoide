package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FedeBP/pumoide/backend/internal/middleware"
	"golang.org/x/time/rate"
)

func TestRateLimitMiddleware(t *testing.T) {
	limiter := middleware.NewIPRateLimiter(rate.Limit(1), 3)
	handler := middleware.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), limiter)

	makeRequest := func() int {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = "127.0.0.1:12345" // Set a consistent IP
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr.Code
	}

	for i := 0; i < 3; i++ {
		if code := makeRequest(); code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i+1, code)
		}
	}

	if code := makeRequest(); code != http.StatusTooManyRequests {
		t.Errorf("Expected status 429, got %d", code)
	}

	time.Sleep(1 * time.Second)
	if code := makeRequest(); code != http.StatusOK {
		t.Errorf("After waiting, expected status 200, got %d", code)
	}
}
