package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ticket-ai/ticket/pkg/telemetry"
	"github.com/ticket-ai/ticket/pkg/budget"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCostAnalyticsMiddleware(t *testing.T) {
	telemetryConfig := telemetry.Config{
		ServiceName:    "test-service",
		Environment:    "test",
		OTelEndpoint:   "localhost:4317",
		MetricsEnabled: true,
		TracingEnabled: true,
	}

	telemetryClient, err := telemetry.NewClient(telemetryConfig)
	require.NoError(t, err)
	defer telemetryClient.Shutdown()

	budgetConfig := budget.Config{
		DefaultBudget: 100.0,
		AlertThreshold: 0.8,
		Models: map[string]budget.ModelConfig{
			"gpt-3.5-turbo": {
				MaxTokens: 4000,
				CostPer1KTokens: 0.002,
			},
			"gpt-4": {
				MaxTokens: 8000,
				CostPer1KTokens: 0.03,
			},
		},
		DatabasePath: ":memory:",
	}
	budgetManager, err := budget.NewManager(budgetConfig)
	require.NoError(t, err)
	defer budgetManager.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("X-Cost", "0.0025")
		w.Header().Set("X-Input-Tokens", "50")
		w.Header().Set("X-Output-Tokens", "100")
		w.Header().Set("X-Model", "gpt-3.5-turbo")
		w.WriteHeader(http.StatusOK)
	})

	middleware := CostAnalyticsMiddleware(telemetryClient, budgetManager)
	wrappedHandler := middleware(handler)

	t.Run("Successful Request", func(t *testing.T) 
		req := httptest.NewRequest("POST", "/test", nil)
		ctx := context.WithValue(req.Context(), "user_id", "test-user")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		assert.Equal(t, "0.0025", rr.Header().Get("X-Cost"))
		assert.Equal(t, "50", rr.Header().Get("X-Input-Tokens"))
		assert.Equal(t, "100", rr.Header().Get("X-Output-Tokens"))
		assert.Equal(t, "gpt-3.5-turbo", rr.Header().Get("X-Model"))
	})

	t.Run("Unauthorized Request", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", nil)
		rr := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Failed Request", func(t *testing.T) {
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		errorMiddleware := CostAnalyticsMiddleware(telemetryClient, budgetManager)
		wrappedErrorHandler := errorMiddleware(errorHandler)

		req := httptest.NewRequest("POST", "/test", nil)
		ctx := context.WithValue(req.Context(), "user_id", "test-user")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		wrappedErrorHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
	})

	t.Run("Throttled Request", func(t *testing.T) {
		throttledHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		})

		throttledMiddleware := CostAnalyticsMiddleware(telemetryClient, budgetManager)
		wrappedThrottledHandler := throttledMiddleware(throttledHandler)

		req := httptest.NewRequest("POST", "/test", nil)
		ctx := context.WithValue(req.Context(), "user_id", "test-user")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()

		wrappedThrottledHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusTooManyRequests, rr.Code)
	})
} 