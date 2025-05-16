package telemetry

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/ticket-ai/ticket/pkg/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTelemetryIntegration(t *testing.T) {
	mockServer := llm.NewMockServer("8081")
	go func() {
		err := mockServer.Start()
		require.NoError(t, err)
	}()
	defer mockServer.Stop()

	config := Config{
		ServiceName:    "test-service",
		Environment:    "test",
		OTelEndpoint:   "localhost:4317",
		MetricsEnabled: true,
		TracingEnabled: true,
	}

	client, err := New(config)
	require.NoError(t, err)
	defer client.Shutdown()

	t.Run("Cost Tracking", func(t *testing.T) {
		event := CostEvent{
			Model:        "gpt-3.5-turbo",
			UserID:       "test-user",
			Cost:         0.0025,
			InputTokens:  50,
			OutputTokens: 100,
			Throttled:    false,
		}

		err := client.RecordCostEvent(context.Background(), event)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond) 

		totalRequests, err := client.GetTotalRequests(context.Background())
		require.NoError(t, err)
		assert.Greater(t, totalRequests, int64(0))

		modelUsage, err := client.GetRequestsPerModel(context.Background())
		require.NoError(t, err)
		assert.Contains(t, modelUsage, "gpt-3.5-turbo")
		assert.Greater(t, modelUsage["gpt-3.5-turbo"], int64(0))

		cost, err := client.GetEstimatedCost(context.Background())
		require.NoError(t, err)
		assert.Greater(t, cost, 0.0)
	})

	t.Run("NLP Metrics", func(t *testing.T) {
		event := Event{
			Timestamp: time.Now(),
			UserID:    "test-user",
			Endpoint:  "/v1/chat/completions",
			Method:    "POST",
			NLPMetrics: analyzer.NLPMetrics{
				Sentiment:        0.5,
				Toxicity:         0.1,
				PII:              0.0,
				Profanity:        0.0,
				Bias:             0.2,
				Emotional:        0.3,
				Manipulative:     0.1,
				JailbreakIntent:  0.0,
				Keywords:         map[string]float64{"test": 0.8},
			},
		}

		err := client.RecordEvent(context.Background(), event)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		toxicityScores, err := client.GetRecentToxicityScores(context.Background(), 5)
		require.NoError(t, err)
		assert.Greater(t, len(toxicityScores), 0)

		profanityScores, err := client.GetRecentProfanityScores(context.Background(), 5)
		require.NoError(t, err)
		assert.Greater(t, len(profanityScores), 0)

		piiScores, err := client.GetRecentPIIScores(context.Background(), 5)
		require.NoError(t, err)
		assert.Greater(t, len(piiScores), 0)

		biasScores, err := client.GetRecentBiasScores(context.Background(), 5)
		require.NoError(t, err)
		assert.Greater(t, len(biasScores), 0)
	})

	t.Run("Balance Tracking", func(t *testing.T) {
		err := client.UpdateBalance(context.Background(), "test-user", 10.0)
		require.NoError(t, err)

		event := CostEvent{
			Model:        "gpt-4",
			UserID:       "test-user",
			Cost:         0.05,
			InputTokens:  100,
			OutputTokens: 200,
			Throttled:    false,
		}

		err = client.RecordCostEvent(context.Background(), event)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		totalRequests, err := client.GetTotalRequests(context.Background())
		require.NoError(t, err)
		assert.Greater(t, totalRequests, int64(0))

		modelUsage, err := client.GetRequestsPerModel(context.Background())
		require.NoError(t, err)
		assert.Contains(t, modelUsage, "gpt-4")
		assert.Greater(t, modelUsage["gpt-4"], int64(0))
	})
} 