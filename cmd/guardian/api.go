package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rohanadwankar/guardian"
)

// FlaggedRequest represents a request that was flagged by Guardian rules
type FlaggedRequest struct {
	Timestamp    time.Time `json:"timestamp"`
	IP           string    `json:"ip"`
	Endpoint     string    `json:"endpoint"`
	Method       string    `json:"method"`
	Score        float64   `json:"score"`
	MatchedRules []string  `json:"matchedRules"`
	InputTokens  int       `json:"inputTokens"`
	OutputTokens int       `json:"outputTokens"`
	Cost         float64   `json:"cost"`
}

// getFlaggedRequests retrieves flagged requests from the Guardian instance
// This is a simple implementation that should be enhanced based on how you store flagged requests
func getFlaggedRequests(g *guardian.Guardian) []FlaggedRequest {
	// This is a placeholder implementation
	// You need to implement this based on how your Guardian stores flagged requests

	// If Guardian has a method to get flagged requests, use it
	// Example: return g.GetFlaggedRequests()

	// For now, return mock data for testing
	mockData := []FlaggedRequest{
		{
			Timestamp:    time.Now().Add(-5 * time.Minute),
			IP:           "192.168.1.1",
			Endpoint:     "/v1/chat/completions",
			Method:       "POST",
			Score:        0.85,
			MatchedRules: []string{"Ignore Instructions Pattern"},
			InputTokens:  150,
			OutputTokens: 50,
			Cost:         0.002,
		},
		{
			Timestamp:    time.Now().Add(-10 * time.Minute),
			IP:           "192.168.1.2",
			Endpoint:     "/v1/chat/completions",
			Method:       "POST",
			Score:        0.95,
			MatchedRules: []string{"Attempted Hacking Keywords"},
			InputTokens:  200,
			OutputTokens: 0, // Blocked, no output
			Cost:         0.0005,
		},
	}

	return mockData
}

// Add API endpoints to the handler
func addAPIEndpoints(handler *http.ServeMux, g *guardian.Guardian) {
	// API endpoint for flagged requests
	handler.HandleFunc("/_guardian/api/flagged", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Get flagged requests from the middleware
		flaggedRequests := getFlaggedRequests(g)
		json.NewEncoder(w).Encode(flaggedRequests)
	})

	// Add more API endpoints as needed...
}
