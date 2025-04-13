// Package middleware provides HTTP middleware for intercepting and monitoring AI endpoints.
package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rohanadwankar/guardian/pkg/analyzer"
	"github.com/rohanadwankar/guardian/pkg/monitoring"
	"github.com/rohanadwankar/guardian/pkg/telemetry"
)

// Config holds configuration for the middleware component.
type Config struct {
	Analyzer          *analyzer.Analyzer
	Telemetry         *telemetry.Client
	Monitor           *monitoring.Monitor
	StandardPrePrompt string
}

// Middleware provides HTTP interception capabilities for AI endpoints.
type Middleware struct {
	config             Config
	aiEndpointPatterns []*regexp.Regexp
}

// New creates a new Middleware instance.
func New(config Config) *Middleware {
	// Compile patterns for detecting AI endpoints
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)/v\d+/chat/completions`),
		regexp.MustCompile(`(?i)/v\d+/completions`),
		regexp.MustCompile(`(?i)/v\d+/generate`),
		regexp.MustCompile(`(?i)/generate`),
		regexp.MustCompile(`(?i)/chat`),
	}

	return &Middleware{
		config:             config,
		aiEndpointPatterns: patterns,
	}
}

// HTTPHandler wraps an HTTP handler with Guardian monitoring.
func (m *Middleware) HTTPHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is an AI endpoint
		if !m.isAIEndpoint(r) {
			next.ServeHTTP(w, r)
			return
		}

		fmt.Printf("[%s] Guardian intercepted AI endpoint: %s\n",
			time.Now().Format(time.RFC3339), r.URL.Path)

		startTime := time.Now()
		clientIP := getClientIP(r)
		userID := getUserID(r)

		// Check if IP is blocked
		if m.config.Monitor.IsIPBlocked(clientIP) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		// Create a context and span for telemetry
		ctx, span := m.config.Telemetry.StartSpan(r.Context(), "ai_request")
		defer m.config.Telemetry.EndSpan(span)

		// Read and modify the request body
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body.Close()
		}

		// Parse the request body
		var bodyMap map[string]interface{}
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, &bodyMap); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)
				return
			}
		}

		// Apply standard pre-prompt if specified
		originalBody := make(map[string]interface{})
		for k, v := range bodyMap {
			originalBody[k] = v
		}

		if m.config.StandardPrePrompt != "" {
			m.applyPrePrompt(bodyMap)
		}

		// Extract text to analyze
		textToAnalyze := extractTextToAnalyze(bodyMap)

		// Analyze for security concerns
		var analysisResult analyzer.Result
		if textToAnalyze != "" {
			analysisResult = m.config.Analyzer.AnalyzeText(textToAnalyze)
		}

		// Check if should block based on analysis
		if m.config.Analyzer.ShouldBlock(analysisResult) {
			// Flag user
			m.config.Monitor.FlagUser(userID, analysisResult.Score, analysisResult.Reasons)

			// Record telemetry
			m.config.Telemetry.RecordEvent(ctx, telemetry.Event{
				UserID:      userID,
				IP:          clientIP,
				Endpoint:    r.URL.Path,
				RequestData: originalBody,
				Duration:    time.Since(startTime),
				Score:       analysisResult.Score,
				Reasons:     analysisResult.Reasons,
				Timestamp:   time.Now(),
			})

			http.Error(w, "Request blocked for security reasons", http.StatusForbidden)
			return
		}

		// Update request body with modified content
		modifiedBody, _ := json.Marshal(bodyMap)
		r.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
		r.ContentLength = int64(len(modifiedBody))

		// Create response interceptor
		rw := newResponseWriter(w)

		// Process the request
		next.ServeHTTP(rw, r)

		// Record telemetry after the request
		var responseData map[string]interface{}
		if rw.body != nil && len(rw.body) > 0 {
			_ = json.Unmarshal(rw.body, &responseData)
		}

		m.config.Telemetry.RecordEvent(ctx, telemetry.Event{
			UserID:       userID,
			IP:           clientIP,
			Endpoint:     r.URL.Path,
			RequestData:  originalBody,
			ResponseData: responseData,
			Duration:     time.Since(startTime),
			Score:        analysisResult.Score,
			Reasons:      analysisResult.Reasons,
			Timestamp:    time.Now(),
		})
	})
}

// isAIEndpoint determines if the request is to an AI endpoint.
func (m *Middleware) isAIEndpoint(r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}

	for _, pattern := range m.aiEndpointPatterns {
		if pattern.MatchString(r.URL.Path) {
			return true
		}
	}

	return false
}

// applyPrePrompt adds the standard pre-prompt to the request.
func (m *Middleware) applyPrePrompt(body map[string]interface{}) {
	// Handle completions format
	if prompt, ok := body["prompt"].(string); ok {
		body["prompt"] = m.config.StandardPrePrompt + "\n\n" + prompt
		return
	}

	// Handle chat completions format
	if messages, ok := body["messages"].([]interface{}); ok {
		// Look for a system message
		hasSystem := false
		for i, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok && role == "system" {
					if content, ok := msgMap["content"].(string); ok {
						msgMap["content"] = m.config.StandardPrePrompt + " " + content
						messages[i] = msgMap
						hasSystem = true
						break
					}
				}
			}
		}

		// Add a system message if none exists
		if !hasSystem {
			systemMsg := map[string]interface{}{
				"role":    "system",
				"content": m.config.StandardPrePrompt,
			}
			body["messages"] = append([]interface{}{systemMsg}, messages...)
		}
	}
}

// extractTextToAnalyze gets the text to analyze from various request formats.
func extractTextToAnalyze(body map[string]interface{}) string {
	var texts []string

	// Handle completions format
	if prompt, ok := body["prompt"].(string); ok {
		texts = append(texts, prompt)
	}

	// Handle chat completions format
	if messages, ok := body["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok && role != "system" {
					if content, ok := msgMap["content"].(string); ok {
						texts = append(texts, content)
					}
				}
			}
		}
	}

	return strings.Join(texts, "\n")
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}

	// Fall back to remote address
	if r.RemoteAddr != "" {
		ip := strings.Split(r.RemoteAddr, ":")
		return ip[0]
	}

	return "unknown"
}

// getUserID extracts the user ID from the request.
func getUserID(r *http.Request) string {
	// Check various headers for user identification
	if userID := r.Header.Get("User-ID"); userID != "" {
		return userID
	}

	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		// In a real implementation, this could parse JWT tokens, etc.
		// For now, just use a simplified version
		if strings.HasPrefix(authHeader, "Bearer ") {
			return "auth_" + authHeader[7:15] // Take a portion of the token as ID
		}
	}

	return "anonymous"
}

// responseWriter is a wrapper for http.ResponseWriter that captures the response body.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

// newResponseWriter creates a new response writer wrapper.
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body.
func (rw *responseWriter) Write(b []byte) (int, error) {
	rw.body = append(rw.body, b...)
	return rw.ResponseWriter.Write(b)
}
