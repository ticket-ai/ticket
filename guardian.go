// Package guardian is the main entry point for the Guardian ethical telemetry and governance platform for AI applications.
package guardian

import (
	"fmt"
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
	NLPAnalysisEnabled  bool
	StaticAnalysisRules []string

	// Governance options
	AutoBlockThreshold float64
	ReviewAgentEnabled bool

	// Pre-prompting management
	StandardPrePrompt string
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
		StaticAnalysisRules: []string{
			`\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b`,
			`\b(pretend|imagine|role-play|simulation).*?(ignore|forget|disregard).*?(instruction|prompt|rule)\b`,
			`\b(let's play a game|hypothetically speaking|in a fictional scenario)\b`,
		},
		AutoBlockThreshold: 0.85,
		ReviewAgentEnabled: false,
		StandardPrePrompt:  "Always adhere to ethical guidelines and refuse harmful requests.",
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
	analyzerInstance, err := analyzer.New(analyzer.Config{
		NLPEnabled:         config.NLPAnalysisEnabled,
		StaticRules:        config.StaticAnalysisRules,
		AutoBlockThreshold: config.AutoBlockThreshold,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize analyzer: %w", err)
	}

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

	monitorInstance := monitoring.New(monitoring.Config{
		ServiceName: config.ServiceName,
		Environment: config.Environment,
	})

	middlewareInstance := middleware.New(middleware.Config{
		Analyzer:          analyzerInstance,
		Telemetry:         telemetryClient,
		Monitor:           monitorInstance,
		StandardPrePrompt: config.StandardPrePrompt,
	})

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

// Shutdown gracefully shuts down all Guardian components.
func (g *Guardian) Shutdown() error {
	if err := g.Telemetry.Shutdown(); err != nil {
		return fmt.Errorf("error shutting down telemetry: %w", err)
	}

	return nil
}

// Version returns the current version of the Guardian platform.
func Version() string {
	return "0.1.0-alpha"
}
