package dashboard

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupTestEnvironment creates a temporary test directory with the necessary dashboard files
func setupTestEnvironment(t *testing.T) (string, func()) {
	// Create a temporary directory for our test files
	tempDir := t.TempDir()
	
	// Create the directory structure
	err := os.MkdirAll(filepath.Join(tempDir, "pkg", "dashboard", "templates"), 0755)
	if err != nil {
		t.Fatalf("Failed to create template directory: %v", err)
	}
	
	err = os.MkdirAll(filepath.Join(tempDir, "pkg", "dashboard", "static", "css"), 0755)
	if err != nil {
		t.Fatalf("Failed to create static/css directory: %v", err)
	}
	
	err = os.MkdirAll(filepath.Join(tempDir, "pkg", "dashboard", "static", "js"), 0755)
	if err != nil {
		t.Fatalf("Failed to create static/js directory: %v", err)
	}
	
	// Create test template with our test variables
	templateContent := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
</head>
<body>
    <div>{{.ServiceName}}</div>
    <div>{{.Environment}}</div>
    <script>
        const useWebSocket = {{.WebSocketEnabled}};
        const refreshInterval = {{.RefreshInterval}};
    </script>
</body>
</html>`
	
	err = os.WriteFile(
		filepath.Join(tempDir, "pkg", "dashboard", "templates", "dashboard.html"),
		[]byte(templateContent),
		0644,
	)
	if err != nil {
		t.Fatalf("Failed to write template file: %v", err)
	}
	
	// Create a simple CSS file
	cssContent := "body { font-family: sans-serif; }"
	err = os.WriteFile(
		filepath.Join(tempDir, "pkg", "dashboard", "static", "css", "styles.css"),
		[]byte(cssContent),
		0644,
	)
	if err != nil {
		t.Fatalf("Failed to write CSS file: %v", err)
	}
	
	// Create a simple JS file
	jsContent := "console.log('Dashboard loaded');"
	err = os.WriteFile(
		filepath.Join(tempDir, "pkg", "dashboard", "static", "js", "dashboard.js"),
		[]byte(jsContent),
		0644,
	)
	if err != nil {
		t.Fatalf("Failed to write JS file: %v", err)
	}
	
	// Store the original working directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	
	// Change to our temp directory
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}
	
	// Return cleanup function
	cleanup := func() {
		os.Chdir(oldWd)
	}
	
	return tempDir, cleanup
}

// TestDashboardTemplateRendering tests if template variables are correctly rendered
func TestDashboardTemplateRendering(t *testing.T) {
	// Setup test environment
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()
	
	// Create dashboard with test configuration
	config := Config{
		Port:            9999,
		EnableWebsocket: true,
		RefreshInterval: 10 * time.Second,
		ServiceName:     "test-service",
		Environment:     "test-env",
	}
	
	dash := New(config)
	
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(dash.dashboardHandler))
	defer server.Close()
	
	// Make request to the dashboard
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get dashboard: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	
	htmlContent := string(body)
	
	// Check if the template variables were properly replaced
	t.Run("ServiceName is rendered", func(t *testing.T) {
		if !strings.Contains(htmlContent, "<div>test-service</div>") {
			t.Errorf("ServiceName not rendered correctly")
			t.Logf("HTML content: %s", htmlContent)
		}
	})
	
	t.Run("Environment is rendered", func(t *testing.T) {
		if !strings.Contains(htmlContent, "<div>test-env</div>") {
			t.Errorf("Environment not rendered correctly")
			t.Logf("HTML content: %s", htmlContent)
		}
	})
	
	t.Run("WebSocketEnabled is rendered as boolean", func(t *testing.T) {
		expected := "const useWebSocket = true;"
		if !strings.Contains(htmlContent, expected) {
			t.Errorf("WebSocketEnabled not rendered correctly")
			t.Errorf("Expected to find: %s", expected)
			t.Logf("HTML content: %s", htmlContent)
		}
	})
	
	t.Run("RefreshInterval is rendered as number", func(t *testing.T) {
		expected := "const refreshInterval = 10;"
		if !strings.Contains(htmlContent, expected) {
			t.Errorf("RefreshInterval not rendered correctly")
			t.Errorf("Expected to find: %s", expected)
			t.Logf("HTML content: %s", htmlContent)
		}
	})
}

// TestStaticFileServing tests if static files are served correctly
func TestStaticFileServing(t *testing.T) {
	// Setup test environment
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()
	
	// Create dashboard with test configuration
	config := Config{
		Port:            9999,
		EnableWebsocket: true,
		RefreshInterval: 10 * time.Second,
	}
	
	dash := New(config)
	
	// Test CSS file serving
	t.Run("Serve CSS file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/css/styles.css", nil)
		rec := httptest.NewRecorder()
		
		dash.staticHandler(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		
		if contentType := rec.Header().Get("Content-Type"); contentType != "text/css" {
			t.Errorf("Expected Content-Type %s, got %s", "text/css", contentType)
		}
		
		expected := "body { font-family: sans-serif; }"
		if body := rec.Body.String(); body != expected {
			t.Errorf("Expected body %q, got %q", expected, body)
		}
	})
	
	// Test JS file serving
	t.Run("Serve JS file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/js/dashboard.js", nil)
		rec := httptest.NewRecorder()
		
		dash.staticHandler(rec, req)
		
		if rec.Code != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, rec.Code)
		}
		
		if contentType := rec.Header().Get("Content-Type"); contentType != "application/javascript" {
			t.Errorf("Expected Content-Type %s, got %s", "application/javascript", contentType)
		}
		
		expected := "console.log('Dashboard loaded');"
		if body := rec.Body.String(); body != expected {
			t.Errorf("Expected body %q, got %q", expected, body)
		}
	})
	
	// Test nonexistent file
	t.Run("Nonexistent file returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/nonexistent.js", nil)
		rec := httptest.NewRecorder()
		
		dash.staticHandler(rec, req)
		
		if rec.Code != http.StatusNotFound {
			t.Errorf("Expected status code %d, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

// TestDashboardIntegration tests the dashboard's HTTP endpoints
func TestDashboardIntegration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Setup test environment
	_, cleanup := setupTestEnvironment(t)
	defer cleanup()
	
	// Create dashboard with test configuration
	config := Config{
		Port:            0, // Use random port
		EnableWebsocket: true,
		RefreshInterval: 2 * time.Second,
		ServiceName:     "test-service",
		Environment:     "test-env",
	}
	
	dash := New(config)
	url := dash.Start()
	defer dash.Stop()
	
	// Wait for server to start
	time.Sleep(100 * time.Millisecond)
	
	// Test dashboard page
	t.Run("Dashboard page loads", func(t *testing.T) {
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed to get dashboard: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		// Read body to check for template variables
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Failed to read response body: %v", err)
		}
		
		htmlContent := string(body)
		
		// Check if the key template variables were properly replaced
		if !strings.Contains(htmlContent, fmt.Sprintf("%t", config.EnableWebsocket)) {
			t.Errorf("WebSocketEnabled not rendered correctly")
		}
		
		if !strings.Contains(htmlContent, fmt.Sprintf("%d", int(config.RefreshInterval.Seconds()))) {
			t.Errorf("RefreshInterval not rendered correctly")
		}
	})
	
	// Test metrics API
	t.Run("Metrics API returns JSON", func(t *testing.T) {
		resp, err := http.Get(url + "/api/metrics")
		if err != nil {
			t.Fatalf("Failed to get metrics: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain %s, got %s", "application/json", contentType)
		}
	})
	
	// Test activity API
	t.Run("Activity API returns JSON", func(t *testing.T) {
		resp, err := http.Get(url + "/api/activity")
		if err != nil {
			t.Fatalf("Failed to get activity: %v", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
		}
		
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/json") {
			t.Errorf("Expected Content-Type to contain %s, got %s", "application/json", contentType)
		}
	})
}