// Package main provides the Guardian proxy server binary that can be launched by any application.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rohanadwankar/guardian"
)

var (
	port           = flag.Int("port", 8080, "Port to listen on")
	configFile     = flag.String("config", "", "Path to config file (optional)")
	serviceName    = flag.String("service", "guardian-app", "Service name")
	environment    = flag.String("env", "development", "Environment (development, staging, production)")
	prePrompt      = flag.String("pre-prompt", "", "Standard pre-prompt to apply to all requests")
	blockThreshold = flag.Float64("threshold", 0.85, "Threshold for automatically blocking requests")
	enableNLP      = flag.Bool("nlp", true, "Enable NLP analysis")
)

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

	// Create server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: g.Middleware.HTTPHandler(http.DefaultServeMux),
	}

	// Register internal endpoint for health checks
	http.HandleFunc("/_guardian/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, guardian.Version())
	})

	// Handle termination signals
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		log.Printf("Received signal: %v", sig)

		if err := g.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}

		log.Println("Guardian shutdown complete")
		os.Exit(0)
	}()

	// Start the server
	log.Printf("Guardian proxy server listening on port %d", *port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
