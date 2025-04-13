package dashboard

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestDashboardWithMissingFiles tests how the dashboard handles missing files
func TestDashboardWithMissingFiles(t *testing.T) {
	// Create a temporary directory with different working directory
	tmpDir := t.TempDir()

	// Store current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to the temporary directory to simulate running from a different location
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Restore working directory when test finishes
	defer os.Chdir(originalWd)

	// Create dashboard with test configuration
	config := Config{
		Port:            8999,
		EnableWebsocket: true,
		RefreshInterval: 5,
		ServiceName:     "test-service",
		Environment:     "test-env",
	}

	dash := New(config)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Call dashboard handler
	dash.dashboardHandler(rec, req)

	// Verify that it returns a 500 error when template is not found
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 when template not found, got %d", rec.Code)
	}

	if rec.Body.String() != "Dashboard template not found" {
		t.Errorf("Expected error message 'Dashboard template not found', got '%s'", rec.Body.String())
	}
}

// TestDashboardWithEmbeddedFiles tests that the dashboard can serve embedded templates
// even when running from a directory that doesn't contain the files
func TestDashboardWithEmbeddedFiles(t *testing.T) {
	// Create a temporary directory with different working directory
	tmpDir := t.TempDir()

	// Store current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to the temporary directory to simulate running from a different location
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Restore working directory when test finishes
	defer os.Chdir(originalWd)

	// Create dashboard with test configuration
	config := Config{
		Port:            8999,
		EnableWebsocket: true,
		RefreshInterval: 5,
		ServiceName:     "test-service",
		Environment:     "test-env",
	}

	dash := New(config)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Call dashboard handler
	dash.dashboardHandler(rec, req)

	// Verify that it returns a 200 OK (embedded files should work)
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK with embedded files, got %d", rec.Code)
	}

	// Check that template variables were replaced correctly
	body := rec.Body.String()

	if !strings.Contains(body, `useWebSocket = true`) {
		t.Errorf("WebSocketEnabled not rendered correctly in template")
	}

	if !strings.Contains(body, `refreshInterval = 5`) {
		t.Errorf("RefreshInterval not rendered correctly in template, expected '5', got %s",
			func() string {
				start := strings.Index(body, "refreshInterval = ")
				if start < 0 {
					return "not found"
				}
				start += len("refreshInterval = ")
				end := strings.Index(body[start:], ";")
				if end < 0 {
					return body[start:]
				}
				return body[start : start+end]
			}())
	}

	// Test static file serving
	t.Run("Serve static CSS file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/css/styles.css", nil)
		rec := httptest.NewRecorder()

		dash.staticHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for CSS file, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/css" {
			t.Errorf("Expected Content-Type 'text/css', got '%s'", contentType)
		}
	})

	t.Run("Serve static JS file", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static/js/dashboard.js", nil)
		rec := httptest.NewRecorder()

		dash.staticHandler(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for JS file, got %d", rec.Code)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/javascript" {
			t.Errorf("Expected Content-Type 'application/javascript', got '%s'", contentType)
		}
	})
}
