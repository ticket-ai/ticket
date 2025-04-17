// Package middleware provides HTTP request interception for Guardian
package middleware

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/rohanadwankar/guardian/pkg/analyzer"  // Corrected import path
	"github.com/rohanadwankar/guardian/pkg/telemetry" // Corrected import path
)

// Middleware handles HTTP request interception for Guardian
type Middleware struct {
	debug     bool
	analyzer  *analyzer.Analyzer
	telemetry *telemetry.Client // Corrected type
}

// NewMiddleware creates a new middleware instance
func NewMiddleware(debug bool, analyzer *analyzer.Analyzer, telemetry *telemetry.Client) *Middleware { // Corrected type
	return &Middleware{
		debug:     debug,
		analyzer:  analyzer,
		telemetry: telemetry,
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
		startTime := time.Now()
		ctx := r.Context()
		originalDestination := r.Header.Get("X-Guardian-Original-Destination")
		isAI := isAIEndpoint(r.URL.Path) && originalDestination != ""

		if !isAI {
			// Not an AI request proxied by Guardian, pass through
			next.ServeHTTP(w, r)
			return
		}

		m.log("Intercepted AI endpoint: %s for destination: %s", r.URL.Path, originalDestination)

		// --- Extract Client Info ---
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			// Fallback to RemoteAddr (might be the immediate client, e.g., server.js)
			// Consider splitting if RemoteAddr includes port
			remoteAddrParts := strings.Split(r.RemoteAddr, ":")
			if len(remoteAddrParts) > 0 {
				clientIP = remoteAddrParts[0]
			}
		} else {
			// If X-Forwarded-For has multiple IPs, take the first one
			ips := strings.Split(clientIP, ",")
			if len(ips) > 0 {
				clientIP = strings.TrimSpace(ips[0])
			}
		}
		userID := r.Header.Get("User-Id")

		// Read request body for analysis
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			m.log("Error reading request body: %v", err)
			http.Error(w, "Internal server error reading request", http.StatusInternalServerError)
			return
		}
		r.Body.Close()                                    // Close original body
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body for proxy

		// --- Analysis --- (Example: Analyze request body)
		analysisResult := analyzer.Result{Score: 0.0} // Default safe score
		if m.analyzer != nil {
			analysisResult = m.analyzer.AnalyzeText(string(bodyBytes))
			m.log("Analysis score for %s: %.2f", r.URL.Path, analysisResult.Score)
		}

		shouldBlock := m.analyzer != nil && m.analyzer.ShouldBlock(analysisResult)

		// --- Telemetry Event Data --- (Initialize)
		event := telemetry.Event{
			Timestamp:   startTime,
			IP:          clientIP, // Use extracted client IP
			UserID:      userID,   // Use extracted User ID
			Endpoint:    r.URL.Path,
			Method:      r.Method,
			Score:       analysisResult.Score,
			Reasons:     analysisResult.Reasons,
			Blocked:     shouldBlock,
			RequestData: map[string]interface{}{"body": string(bodyBytes)}, // Example
		}

		// --- Blocking --- (If analysis determines blocking)
		if shouldBlock {
			m.log("Blocking request to %s based on analysis score: %.2f", r.URL.Path, analysisResult.Score)
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Request blocked by Guardian policy."))

			// Record blocked event
			event.StatusCode = http.StatusForbidden
			event.Duration = time.Since(startTime)
			if m.telemetry != nil {
				m.telemetry.RecordEvent(ctx, event)
			}
			return
		}

		// --- Proxying --- (If not blocked)
		targetURL, err := url.Parse(originalDestination)
		if err != nil {
			m.log("Error parsing original URL '%s': %v", originalDestination, err)
			http.Error(w, "Internal server error parsing destination", http.StatusInternalServerError)
			return
		}
		event.Destination = targetURL.Host // Store host:port

		proxy := httputil.NewSingleHostReverseProxy(targetURL)

		// Modify the outgoing request
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Host = targetURL.Host
			req.URL.Host = targetURL.Host
			req.URL.Scheme = targetURL.Scheme
			req.URL.Path = targetURL.Path // Use original path from destination URL
			req.Header.Del("X-Guardian-Original-Destination")
			req.Header.Del("X-Guardian-Timestamp")
			m.log("Forwarding %s to: %s", req.Method, req.URL.String())
		}

		// Wrap response writer to capture status code
		rw := newResponseWriter(w)

		// Call the proxy handler
		proxy.ServeHTTP(rw, r)

		// --- Telemetry Recording --- (After proxying)
		event.StatusCode = rw.statusCode
		event.Duration = time.Since(startTime)
		// event.ResponseData = map[string]interface{}{"body": rw.body.String()} // Optionally capture response body

		if m.telemetry != nil {
			m.telemetry.RecordEvent(ctx, event)
		}

		m.log("Finished %s %s -> %s | Status: %d | Latency: %s", r.Method, r.URL.Path, targetURL.String(), rw.statusCode, event.Duration)
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

// responseWriter is a wrapper around http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
