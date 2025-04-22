package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/rohanadwankar/guardian"
	"github.com/rohanadwankar/guardian/pkg/telemetry"
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

// TelemetryStats represents stats from the telemetry system
type TelemetryStats struct {
	TotalRequests    int            `json:"totalRequests"`
	FlaggedRequests  int            `json:"flaggedRequests"`
	BlockedRequests  int            `json:"blockedRequests"`
	AverageScore     float64        `json:"averageScore"`
	EstimatedCost    float64        `json:"estimatedCost"`
	ToxicityScores   []float64      `json:"toxicityScores"`
	ProfanityScores  []float64      `json:"profanityScores"`
	PIIScores        []float64      `json:"piiScores"`
	BiasScores       []float64      `json:"biasScores"`
	RequestsPerModel map[string]int `json:"requestsPerModel"`
}

// In-memory cache of flagged requests
var (
	flaggedRequests = make([]FlaggedRequest, 0, 100)
	flaggedMutex    = sync.RWMutex{}
)

// Add API endpoints to the handler
func addAPIEndpoints(handler *http.ServeMux, g *guardian.Guardian) {
	// API endpoint for flagged requests
	handler.HandleFunc("/_guardian/api/flagged", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		flaggedMutex.RLock()
		defer flaggedMutex.RUnlock()
		json.NewEncoder(w).Encode(flaggedRequests)
	})

	// API endpoint for telemetry statistics
	handler.HandleFunc("/_guardian/api/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Access the telemetry client from guardian
		telemetryClient := g.GetTelemetryClient()
		if telemetryClient == nil {
			http.Error(w, "Telemetry client not available", http.StatusInternalServerError)
			return
		}

		// Get stats from telemetry
		stats := getTelemetryStats(telemetryClient)
		json.NewEncoder(w).Encode(stats)
	})

	// API endpoint for real-time metrics
	handler.HandleFunc("/_guardian/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Get metrics data from telemetry or guardian instance
		metrics := getMetricsData(g)
		json.NewEncoder(w).Encode(metrics)
	})
}

// Get telemetry stats from the telemetry client
// In your api.go file or a new file specifically for telemetry API
func getTelemetryStats(client *telemetry.Client) TelemetryStats {
	// Create context for metrics retrieval
	ctx := context.Background()

	// This is where you'd pull actual data from your telemetry system
	// For example:
	totalRequests := client.GetTotalRequests(ctx)
	flaggedRequests := client.GetFlaggedRequests(ctx)
	blockCount := client.GetBlockedRequests(ctx)
	avgScore := client.GetAverageScore(ctx)
	estimatedCost := client.GetEstimatedCost(ctx)

	// Get NLP scores (depending on how they're stored in your telemetry system)
	toxicityScores := client.GetRecentToxicityScores(ctx, 100)
	profanityScores := client.GetRecentProfanityScores(ctx, 100)
	piiScores := client.GetRecentPIIScores(ctx, 100)
	biasScores := client.GetRecentBiasScores(ctx, 100)

	// Get model usage
	modelUsage := client.GetRequestsPerModel(ctx)

	return TelemetryStats{
		TotalRequests:    totalRequests,
		FlaggedRequests:  flaggedRequests,
		BlockedRequests:  blockCount,
		AverageScore:     avgScore,
		EstimatedCost:    estimatedCost,
		ToxicityScores:   toxicityScores,
		ProfanityScores:  profanityScores,
		PIIScores:        piiScores,
		BiasScores:       biasScores,
		RequestsPerModel: modelUsage,
	}
}

// Get real-time metrics data
func getMetricsData(g *guardian.Guardian) map[string]interface{} {
	// Placeholder for real metrics data
	return map[string]interface{}{
		"requestsPerSecond": 2.5,
		"latencyMs":         245,
		"cpuUsage":          12.3,
		"memoryUsageMB":     156,
	}
}

// Record a flagged request for the API
func recordFlaggedRequest(req FlaggedRequest) {
	flaggedMutex.Lock()
	defer flaggedMutex.Unlock()

	// Keep only the newest 100 requests
	if len(flaggedRequests) >= 100 {
		flaggedRequests = flaggedRequests[1:]
	}
	flaggedRequests = append(flaggedRequests, req)
}
