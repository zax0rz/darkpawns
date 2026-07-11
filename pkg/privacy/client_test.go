package privacy

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_FilterText(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"filtered_text": "Hello [REDACTED]", "detected_categories": ["person"]}`))
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
		_, _ = w.Write([]byte(`{"filtered_text": "[REDACTED]", "detected_categories": ["person"]}`))
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

func TestBatchFilter_PartialError(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if requestCount == 2 {
			// Malformed JSON triggers FilterText error for the second element.
			_, _ = w.Write([]byte(`{not valid json`))
			return
		}
		_, _ = w.Write([]byte(`{"filtered_text": "[REDACTED]", "detected_categories": ["person"]}`))
	}))
	defer server.Close()

	config := DefaultFilterConfig()
	client := NewClient(server.URL, config)

	texts := []string{"first", "second", "third"}
	filtered, detected, err := client.BatchFilter(texts)
	if err == nil {
		t.Fatal("BatchFilter expected error for malformed element, got nil")
	}

	if len(filtered) != 3 {
		t.Errorf("Expected 3 filtered texts, got %d", len(filtered))
	}
	if len(detected) != 3 {
		t.Errorf("Expected 3 detection lists, got %d", len(detected))
	}

	if filtered[0] != "[REDACTED]" || filtered[2] != "[REDACTED]" {
		t.Errorf("Expected successful elements to be filtered, got %v", filtered)
	}
	if filtered[1] != "[FILTERED]" {
		t.Errorf("Expected failed element to use fallback, got %q", filtered[1])
	}
	if !contains(detected[1], "fallback") {
		t.Errorf("Expected failed element to report fallback, got %v", detected[1])
	}
}

func TestConfig_LoadFromEnv(t *testing.T) {
	// t.Setenv auto-restores the prior value on test completion, so the env
	// no longer leaks to other tests (the old os.Setenv/Unsetenv deferred
	// cleanup raced with parallel tests).
	t.Setenv("PRIVACY_FILTER_URL", "http://test:8000")
	t.Setenv("PRIVACY_FILTER_ENABLED", "false")
	t.Setenv("PRIVACY_FILTER_CATEGORIES", "email,phone")
	t.Setenv("PRIVACY_FILTER_REPLACEMENT", "***")
	t.Setenv("PRIVACY_FILTER_KEEP_LENGTH", "true")
	t.Setenv("FILTER_PLAYER_NAMES", "false")
	t.Setenv("PRIVACY_FILTER_TIMEOUT", "7")

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

	// Timeout must be loaded and flow through to FilterConfig for the Client.
	if config.Timeout != 7 {
		t.Errorf("Expected Timeout 7, got %d", config.Timeout)
	}
	fc := config.ToFilterConfig()
	if fc.TimeoutSeconds != 7 {
		t.Errorf("Expected FilterConfig.TimeoutSeconds 7, got %d", fc.TimeoutSeconds)
	}
}

func TestPrivacyLogger(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"filtered_text": "Hello [REDACTED]", "detected_categories": ["person"]}`))
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
