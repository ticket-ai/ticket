// Package middleware provides HTTP request interception for Guardian
package middleware

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Middleware handles HTTP request interception for Guardian
type Middleware struct {
	debug bool
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(debug bool) *Middleware {
	return &Middleware{
		debug: debug,
	}
}

// log prints debug messages if debug mode is enabled
func (m *Middleware) log(format string, args ...interface{}) {
	if m.debug {
		log.Printf("[Guardian Middleware] "+format, args...)
	}
}

// HTTPHandler wraps an HTTP handler with Guardian functionality
func (m *Middleware) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an AI endpoint that we should process
		if isAIEndpoint(r.URL.Path) {
			m.log("Intercepted AI endpoint: %s", r.URL.Path)
			
			// Add timestamp header to help with tracking
			timestamp := time.Now().Format(time.RFC3339)
			r.Header.Set("X-Guardian-Timestamp", timestamp)

			// Get the original destination from headers if it exists
			originalDestination := r.Header.Get("X-Guardian-Original-Destination")
			
			if originalDestination != "" {
				m.log("Found original destination header: %s", originalDestination)
				
				// Parse the original URL
				targetURL, err := url.Parse(originalDestination)
				if err != nil {
					m.log("Error parsing original URL: %v", err)
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}

				// Create a reverse proxy
				proxy := httputil.NewSingleHostReverseProxy(targetURL)
				
				// Modify the outgoing request
				originalDirector := proxy.Director
				proxy.Director = func(req *http.Request) {
					originalDirector(req)
					
					// Update the host and scheme
					req.Host = targetURL.Host
					req.URL.Host = targetURL.Host
					req.URL.Scheme = targetURL.Scheme
					
					// Keep the original path
					req.URL.Path = targetURL.Path
					
					// Remove Guardian-specific headers for the outgoing request
					req.Header.Del("X-Guardian-Original-Destination")
					req.Header.Del("X-Guardian-Timestamp")
					
					m.log("Forwarding to: %s", req.URL.String())
				}
				
				// Call the proxy handler
				proxy.ServeHTTP(w, r)
				return
			}
			
			// If no original destination, this is not a proxied request
			// So we continue with regular processing
			m.log("Processing AI request normally: %s", r.URL.Path)
		}
		
		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

// isAIEndpoint determines if a given path is an AI completion/chat endpoint
func isAIEndpoint(path string) bool {
	path = strings.ToLower(path)
	return strings.Contains(path, "/v1/completions") ||
		strings.Contains(path, "/v1/chat/completions") ||
		strings.Contains(path, "/completions") ||
		strings.Contains(path, "/chat/completions")
}
