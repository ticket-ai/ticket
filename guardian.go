// Package guardian is the main entry point for the Guardian ethical telemetry and governance platform for AI applications.
package guardian

import (
	"fmt"
	"log"
	"time"

	"github.com/rohanadwankar/guardian/pkg/analyzer"
	"github.com/rohanadwankar/guardian/pkg/dashboard"
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

	// Dashboard configuration
	DashboardPort    int
	DashboardEnabled bool

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
		StaticAnalysisRules: []string{
			`\b(system prompt|ignore previous instructions|my previous instructions|my prior instructions)\b`,
			`\b(pretend|imagine|role-play|simulation).*?(ignore|forget|disregard).*?(instruction|prompt|rule)\b`,
			`\b(let's play a game|hypothetically speaking|in a fictional scenario)\b`,
		},
		AutoBlockThreshold: 0.85,
		ReviewAgentEnabled: false,
		StandardPrePrompt:  "Always adhere to ethical guidelines and refuse harmful requests.",
		DashboardPort:      8888,
		DashboardEnabled:   true,
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
	Dashboard  *dashboard.Dashboard
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
	if config.DashboardPort == 0 {
		config.DashboardPort = DefaultConfig().DashboardPort
	}

	// Debug logging
	if config.Debug {
		log.Println("Guardian debug mode enabled")
		log.Printf("Configuration: %+v", config)
	}

	// Initialize analyzer component
	analyzerInstance, err := analyzer.New(analyzer.Config{
		NLPEnabled:         config.NLPAnalysisEnabled,
		StaticRules:        config.StaticAnalysisRules,
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
	middlewareInstance := middleware.New(middleware.Config{
		Analyzer:          analyzerInstance,
		Telemetry:         telemetryClient,
		Monitor:           monitorInstance,
		StandardPrePrompt: config.StandardPrePrompt,
	})

	// Initialize dashboard component if enabled
	var dashboardInstance *dashboard.Dashboard
	if config.DashboardEnabled {
		dashboardInstance = dashboard.New(dashboard.Config{
			Port:            config.DashboardPort,
			EnableWebsocket: true,
			RefreshInterval: 5 * time.Second,
		})
	}

	g := &Guardian{
		Config:     config,
		Analyzer:   analyzerInstance,
		Telemetry:  telemetryClient,
		Monitoring: monitorInstance,
		Middleware: middlewareInstance,
		Dashboard:  dashboardInstance,
		StartTime:  time.Now(),
	}

	return g, nil
}

// Start initializes and starts all Guardian components.
func (g *Guardian) Start() (string, error) {
	// Start the dashboard if it's enabled
	var dashboardURL string
	if g.Config.DashboardEnabled && g.Dashboard != nil {
		dashboardURL = g.Dashboard.Start()
		if g.Config.Debug {
			log.Printf("Guardian dashboard started at %s", dashboardURL)
		}
	}

	return dashboardURL, nil
}

// Shutdown gracefully shuts down all Guardian components.
func (g *Guardian) Shutdown() error {
	// Shutdown dashboard if it exists
	if g.Dashboard != nil {
		if err := g.Dashboard.Stop(); err != nil {
			log.Printf("Error shutting down dashboard: %v", err)
		}
	}

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

// GetDashboardURL returns the URL for the Guardian dashboard.
func (g *Guardian) GetDashboardURL() string {
	if !g.Config.DashboardEnabled || g.Dashboard == nil {
		return ""
	}

	return g.Dashboard.GetDashboardURL()
}

// RecordCompletionRequest records a completion request for monitoring.
// This updates the dashboard with the latest metrics.
func (g *Guardian) RecordCompletionRequest(method, endpoint, status string, latencyMs int, ip string) {
	// Update dashboard if enabled
	if g.Dashboard != nil {
		g.Dashboard.RecordRequest(method, endpoint, latencyMs, status, ip)
	}

	// Update other monitoring systems as needed
	// ...
}

// UpdateMessagesPerSecond updates the messages per second metric on the dashboard.
func (g *Guardian) UpdateMessagesPerSecond(value float64) {
	if g.Dashboard != nil {
		g.Dashboard.IncrementMessagesPerSecond(value)
	}
}
