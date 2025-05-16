package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yourusername/guardian/pkg/telemetry"
	"github.com/yourusername/guardian/pkg/budget"
)

func CostAnalyticsMiddleware(telemetryClient *telemetry.Client, budgetManager *budget.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			userID, ok := r.Context().Value("user_id").(string)
			if !ok || userID == "" {
				log.Printf("CostAnalyticsMiddleware: unauthorized request from %s", r.RemoteAddr)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			rw := newResponseWriter(w)

			model := r.Header.Get("X-Model")
			if model == "" {
				model = "unknown"
				log.Printf("CostAnalyticsMiddleware: no model specified for request from user %s", userID)
			}

			next.ServeHTTP(rw, r)

			if rw.statusCode >= 200 && rw.statusCode < 300 {
				cost := getFloat64Header(r, "X-Cost", 0.0)
				inputTokens := int(getFloat64Header(r, "X-Input-Tokens", 0))
				outputTokens := int(getFloat64Header(r, "X-Output-Tokens", 0))
				throttled := rw.statusCode == http.StatusTooManyRequests

				event := telemetry.CostEvent{
					Model:        model,
					UserID:       userID,
					Cost:         cost,
					InputTokens:  inputTokens,
					OutputTokens: outputTokens,
					Throttled:    throttled,
				}

				if err := telemetryClient.RecordCostEvent(r.Context(), event); err != nil {
					log.Printf("CostAnalyticsMiddleware: failed to record cost event: %v", err)
				}

				if err := telemetryClient.UpdateBalance(r.Context(), userID, -cost); err != nil {
					log.Printf("CostAnalyticsMiddleware: failed to update balance: %v", err)
				}

				duration := time.Since(start)
				log.Printf("CostAnalyticsMiddleware: tracked cost %.4f USD for user %s using model %s (tokens: %d/%d) in %v",
					cost, userID, model, inputTokens, outputTokens, duration)
			} else {
				log.Printf("CostAnalyticsMiddleware: request failed with status %d for user %s using model %s",
					rw.statusCode, userID, model)
			}
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getFloat64Header(r *http.Request, key string, defaultValue float64) float64 {
	value := r.Header.Get(key)
	if value == "" {
		return defaultValue
	}

	var result float64
	_, err := fmt.Sscanf(value, "%f", &result)
	if err != nil {
		log.Printf("CostAnalyticsMiddleware: failed to parse header %s: %v", key, err)
		return defaultValue
	}
	return result
} 