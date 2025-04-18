// Package guardian is the main entry point for the Guardian ethical telemetry and governance platform for AI applications.
package guardian

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rohanadwankar/guardian/pkg/analyzer"
	"github.com/rohanadwankar/guardian/pkg/middleware"
	"github.com/rohanadwankar/guardian/pkg/monitoring"
	"github.com/rohanadwankar/guardian/pkg/telemetry"
)

// Config contains the configuration options for Guardian.
type Config struct {
	// Basic configuration
	ServiceName string
	Environment string

	// Telemetry configuration
	OTelEndpoint   string
	MetricsEnabled bool
	TracingEnabled bool

	// Security features
	NLPAnalysisEnabled bool
	Rules              []analyzer.Rule // Use Rule struct directly

	// Governance options
	AutoBlockThreshold float64
	ReviewAgentEnabled bool

	// Pre-prompting management
	StandardPrePrompt string

	// Debug mode (enables verbose logging)
	Debug bool
}

// DefaultConfig returns a default configuration for Guardian.
func DefaultConfig() Config {
	return Config{
		ServiceName:        "guardian-app",
		Environment:        "development",
		OTelEndpoint:       "localhost:4317",
		MetricsEnabled:     true,
		TracingEnabled:     true,
		NLPAnalysisEnabled: true,
		AutoBlockThreshold: 0.85,
		ReviewAgentEnabled: false,
		StandardPrePrompt:  "Always adhere to ethical guidelines and refuse harmful requests.",
		Debug:              false,
	}
}

// Guardian is the main struct that coordinates all the components of the Guardian platform.
type Guardian struct {
	Config     Config
	Analyzer   *analyzer.Analyzer
	Middleware *middleware.Middleware
	Telemetry  *telemetry.Client
	Monitoring *monitoring.Monitor
	StartTime  time.Time
}

// New creates a new Guardian instance with the provided configuration.
func New(config Config) (*Guardian, error) {
	// Apply default values for any unspecified fields
	if config.ServiceName == "" {
		config.ServiceName = DefaultConfig().ServiceName
	}
	if config.Environment == "" {
		config.Environment = DefaultConfig().Environment
	}

	// Debug logging
	if config.Debug {
		log.Println("Guardian debug mode enabled")
		log.Printf("Configuration: %+v", config)
	}

	// Initialize analyzer component
	analyzerInstance, err := analyzer.New(analyzer.Config{
		NLPEnabled:         config.NLPAnalysisEnabled,
		AutoBlockThreshold: config.AutoBlockThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize analyzer: %w", err)
	}

	// Initialize telemetry component
	telemetryClient, err := telemetry.New(telemetry.Config{
		ServiceName:    config.ServiceName,
		Environment:    config.Environment,
		OTelEndpoint:   config.OTelEndpoint,
		MetricsEnabled: config.MetricsEnabled,
		TracingEnabled: config.TracingEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telemetry client: %w", err)
	}

	// Initialize monitoring component
	monitorInstance := monitoring.New(monitoring.Config{
		ServiceName: config.ServiceName,
		Environment: config.Environment,
	})

	// Initialize middleware component
	middlewareInstance := middleware.NewMiddleware(config.Debug, analyzerInstance, telemetryClient) // Corrected argument order

	g := &Guardian{
		Config:     config,
		Analyzer:   analyzerInstance,
		Telemetry:  telemetryClient,
		Monitoring: monitorInstance,
		Middleware: middlewareInstance,
		StartTime:  time.Now(),
	}

	return g, nil
}

// Start initializes and starts all Guardian components.
func (g *Guardian) Start() (string, error) {
	if g.Config.Debug {
		log.Println("Guardian started.") // Simplified log message
	}

	return "", nil // Return empty string and nil error as dashboard URL is removed
}

// Shutdown gracefully shuts down all Guardian components.
func (g *Guardian) Shutdown() error {
	// Shutdown telemetry
	if err := g.Telemetry.Shutdown(); err != nil {
		return fmt.Errorf("error shutting down telemetry: %w", err)
	}

	return nil
}

// Version returns the current version of the Guardian platform.
func Version() string {
	return "0.1.0-alpha"
}

// RecordCompletionRequest records a completion request for monitoring.
// This sends data to OpenTelemetry.
func (g *Guardian) RecordCompletionRequest(method, endpoint, status string, latencyMs int, ip string) {
	// Send telemetry if enabled
	if g.Telemetry != nil && g.Config.MetricsEnabled {
		ctx := context.Background()

		// Create a telemetry event
		event := telemetry.Event{
			Endpoint:  endpoint,
			IP:        ip,
			Duration:  time.Duration(latencyMs) * time.Millisecond,
			Score:     0.0, // Set appropriate score if available
			Timestamp: time.Now(),
			RequestData: map[string]interface{}{
				"method": method,
				"status": status,
			},
		}

		// Record the event
		if err := g.Telemetry.RecordEvent(ctx, event); err != nil {
			// Just log the error but don't fail the request
			if g.Config.Debug {
				log.Printf("Failed to record telemetry event: %v", err)
			}
		}
	}
}
