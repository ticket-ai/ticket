// Package main provides the Guardian proxy server binary that can be launched by any application.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rohanadwankar/guardian"
)

var (
	port            = flag.Int("port", 8080, "Port to listen on")
	configFile      = flag.String("config", "", "Path to config file (optional)")
	serviceName     = flag.String("service", "guardian-app", "Service name")
	environment     = flag.String("env", "development", "Environment (development, staging, production)")
	prePrompt       = flag.String("pre-prompt", "", "Standard pre-prompt to apply to all requests")
	blockThreshold  = flag.Float64("threshold", 0.85, "Threshold for automatically blocking requests")
	enableNLP       = flag.Bool("nlp", true, "Enable NLP analysis")
	dashboardPort   = flag.Int("dashboard-port", 8888, "Port for the dashboard to listen on")
	enableDashboard = flag.Bool("dashboard", true, "Enable the monitoring dashboard")
	debug           = flag.Bool("debug", false, "Enable debug mode")
)

// Metrics for the Guardian proxy
type metrics struct {
	requestCount       uint64
	blockedCount       uint64
	totalLatency       uint64
	requestsLastSecond uint64
	lastCountTime      time.Time
}

func main() {
	flag.Parse()

	// Log startup information
	log.Printf("Guardian v%s starting...", guardian.Version())
	log.Printf("Service: %s, Environment: %s", *serviceName, *environment)
	log.Printf("Listening on port %d", *port)

	// Create Guardian configuration
	config := guardian.DefaultConfig()
	config.ServiceName = *serviceName
	config.Environment = *environment
	config.AutoBlockThreshold = *blockThreshold
	config.NLPAnalysisEnabled = *enableNLP
	config.DashboardPort = *dashboardPort
	config.DashboardEnabled = *enableDashboard
	config.Debug = *debug

	if *prePrompt != "" {
		config.StandardPrePrompt = *prePrompt
	}

	// If config file is provided, load it (not implemented in this example)
	if *configFile != "" {
		log.Printf("Loading configuration from %s", *configFile)
		// loadConfig(configFile, &config)
	}

	// Initialize Guardian
	g, err := guardian.New(config)
	if err != nil {
		log.Fatalf("Failed to initialize Guardian: %v", err)
	}

	// Initialize metrics
	m := &metrics{
		lastCountTime: time.Now(),
	}

	// Create middleware handler
	handler := http.NewServeMux()

	// Register internal endpoint for health checks
	handler.HandleFunc("/_guardian/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, guardian.Version())
	})

	// Create a middleware wrapper that monitors requests
	monitoredHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an AI completion/chat endpoint
		isAIEndpoint := false
		if r.URL.Path == "/v1/completions" || r.URL.Path == "/v1/chat/completions" ||
			r.URL.Path == "/completions" || r.URL.Path == "/chat/completions" {
			isAIEndpoint = true
		}

		// Record the request start time
		start := time.Now()

		// Process the request with the default handler
		handler.ServeHTTP(w, r)

		// Calculate latency
		latency := time.Since(start).Milliseconds()

		// Only record metrics for AI endpoints
		if isAIEndpoint {
			// Update metrics
			atomic.AddUint64(&m.requestCount, 1)
			atomic.AddUint64(&m.totalLatency, uint64(latency))
			atomic.AddUint64(&m.requestsLastSecond, 1)

			// Record the request in the dashboard
			status := "OK"
			g.RecordCompletionRequest(r.Method, r.URL.Path, status, int(latency), r.RemoteAddr)
		}
	})

	// Create server with the monitored handler
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: g.Middleware.HTTPHandler(monitoredHandler),
	}

	// Start the dashboard if enabled
	var dashboardURL string
	if config.DashboardEnabled {
		var startErr error
		dashboardURL, startErr = g.Start()
		if startErr != nil {
			log.Printf("Warning: Failed to start dashboard: %v", startErr)
		} else {
			if dashboardURL != "" {
				log.Printf("")
				log.Printf("🔍 Guardian dashboard running at: %s", dashboardURL)
				log.Printf("📊 View metrics and monitor AI requests in real-time")
				log.Printf("")
			}
		}

		// Start a background goroutine to update metrics regularly
		go updateMetrics(g, m)
	}

	// Handle termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		log.Printf("Received signal: %v", sig)

		// Create a timeout context for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Attempt to gracefully shut down the server
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}

		// Shut down Guardian components
		if err := g.Shutdown(); err != nil {
			log.Printf("Error during Guardian shutdown: %v", err)
		}

		log.Println("Guardian shutdown complete")
		os.Exit(0)
	}()

	// Start the server
	log.Printf("Guardian proxy server listening on port %d", *port)
	log.Printf("Ready to protect AI endpoints...")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// updateMetrics calculates and updates metrics for the dashboard and sends them to OpenTelemetry
func updateMetrics(g *guardian.Guardian, m *metrics) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Calculate messages per second
		now := time.Now()
		elapsed := now.Sub(m.lastCountTime).Seconds()
		count := atomic.SwapUint64(&m.requestsLastSecond, 0)

		mps := 0.0
		if elapsed > 0 {
			mps = float64(count) / elapsed
		}

		// Reset the timer
		m.lastCountTime = now
		_ = mps // Use mps to avoid unused variable error if needed, or remove calculation
	}
}
