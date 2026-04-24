package privacy

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestClient_FilterText(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"filtered_text": "Hello [REDACTED]", "detected_categories": ["person"]}`))
	}))
	defer server.Close()

	config := DefaultFilterConfig()
	client := NewClient(server.URL, config)

	filtered, detected, err := client.FilterText("Hello John Doe")
	if err != nil {
		t.Fatalf("FilterText failed: %v", err)
	}

	if filtered != "Hello [REDACTED]" {
		t.Errorf("Expected filtered text 'Hello [REDACTED]', got '%s'", filtered)
	}

	if len(detected) != 1 || detected[0] != "person" {
		t.Errorf("Expected detected categories ['person'], got %v", detected)
	}
}

func TestClient_FilterText_Fallback(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultFilterConfig()
	client := NewClient(server.URL, config)

	filtered, detected, err := client.FilterText("Hello John Doe")
	if err != nil {
		t.Fatalf("FilterText should not return error on fallback: %v", err)
	}

	if !contains(detected, "fallback") {
		t.Errorf("Expected fallback detection, got %v", detected)
	}

	if filtered == "Hello John Doe" {
		t.Error("Expected filtered text to be modified on fallback")
	}
}

func TestClient_FilterText_Disabled(t *testing.T) {
	config := DefaultFilterConfig()
	client := NewClient("disabled", config)

	filtered, detected, err := client.FilterText("Hello John Doe")
	if err != nil {
		t.Fatalf("FilterText failed: %v", err)
	}

	if !contains(detected, "fallback") {
		t.Errorf("Expected fallback detection, got %v", detected)
	}

	if filtered != "[FILTERED]" {
		t.Errorf("Expected [FILTERED], got %q", filtered)
	}
}

func TestBatchFilter(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"filtered_text": "[REDACTED]", "detected_categories": ["person"]}`))
	}))
	defer server.Close()

	config := DefaultFilterConfig()
	client := NewClient(server.URL, config)

	texts := []string{"John Doe", "Jane Smith"}
	filtered, detected, err := client.BatchFilter(texts)
	if err != nil {
		t.Fatalf("BatchFilter failed: %v", err)
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered texts, got %d", len(filtered))
	}

	if len(detected) != 2 {
		t.Errorf("Expected 2 detection lists, got %d", len(detected))
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("PRIVACY_FILTER_URL", "http://test:8000")
	os.Setenv("PRIVACY_FILTER_ENABLED", "false")
	os.Setenv("PRIVACY_FILTER_CATEGORIES", "email,phone")
	os.Setenv("PRIVACY_FILTER_REPLACEMENT", "***")
	os.Setenv("PRIVACY_FILTER_KEEP_LENGTH", "true")
	os.Setenv("FILTER_PLAYER_NAMES", "false")

	defer func() {
		os.Unsetenv("PRIVACY_FILTER_URL")
		os.Unsetenv("PRIVACY_FILTER_ENABLED")
		os.Unsetenv("PRIVACY_FILTER_CATEGORIES")
		os.Unsetenv("PRIVACY_FILTER_REPLACEMENT")
		os.Unsetenv("PRIVACY_FILTER_KEEP_LENGTH")
		os.Unsetenv("FILTER_PLAYER_NAMES")
	}()

	config := LoadConfig()

	if config.URL != "http://test:8000" {
		t.Errorf("Expected URL 'http://test:8000', got '%s'", config.URL)
	}

	if config.Enabled {
		t.Error("Expected Enabled false")
	}

	if len(config.Categories) != 2 || config.Categories[0] != "email" || config.Categories[1] != "phone" {
		t.Errorf("Expected categories ['email', 'phone'], got %v", config.Categories)
	}

	if config.Replacement != "***" {
		t.Errorf("Expected replacement '***', got '%s'", config.Replacement)
	}

	if !config.KeepLength {
		t.Error("Expected KeepLength true")
	}

	if config.FilterPlayerNames {
		t.Error("Expected FilterPlayerNames false")
	}
}

func TestPrivacyLogger(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"filtered_text": "Hello [REDACTED]", "detected_categories": ["person"]}`))
	}))
	defer server.Close()

	config := DefaultFilterConfig()
	client := NewClient(server.URL, config)

	// Capture log output
	var buf bytes.Buffer
	logger := &PrivacyLogger{
		client:  client,
		stdLog:  log.New(&buf, "", 0),
		enabled: true,
	}

	logger.Print("Hello John Doe")
	output := buf.String()

	if !strings.Contains(output, "[REDACTED]") {
		t.Errorf("Expected filtered output, got: %s", output)
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function for string operations
var stringHelpers = struct {
	Contains func(string, string) bool
}{
	Contains: func(s, substr string) bool {
		return strings.Contains(s, substr)
	},
}
