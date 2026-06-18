package privacy

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// redactingServer returns a privacy-filter stub that replaces a sentinel email
// in whatever text it receives, echoing the rest back unchanged. This lets a
// test assert that a given value actually passed through the filter.
func redactingServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req FilterRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		filtered := strings.ReplaceAll(req.Text, "user@example.com", "[REDACTED-EMAIL]")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FilterResponse{
			FilteredText: filtered,
			Detected:     []string{"email"},
		})
	}))
}

// TestPIIHandler_WithAttrsFiltered is the regression guard for the WithAttrs
// PII bypass: handler-level attrs added via slog.With must be filtered, not
// passed through raw. Without the fix the email appears in plaintext.
func TestPIIHandler_WithAttrsFiltered(t *testing.T) {
	server := redactingServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.With("email", "user@example.com").Info("login attempt")

	out := buf.String()
	if strings.Contains(out, "user@example.com") {
		t.Errorf("WithAttrs leaked unfiltered email: %q", out)
	}
	if !strings.Contains(out, "[REDACTED-EMAIL]") {
		t.Errorf("expected redacted email in output, got: %q", out)
	}
}

// TestPIIHandler_HandleFiltered confirms per-call attrs are still filtered
// (guards against a regression in the existing Handle path).
func TestPIIHandler_HandleFiltered(t *testing.T) {
	server := redactingServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.Info("login attempt", slog.String("email", "user@example.com"))

	out := buf.String()
	if strings.Contains(out, "user@example.com") {
		t.Errorf("Handle leaked unfiltered email: %q", out)
	}
	if !strings.Contains(out, "[REDACTED-EMAIL]") {
		t.Errorf("expected redacted email in output, got: %q", out)
	}
}

// TestPIIHandler_WithGroupAttrsFiltered confirms attrs added after WithGroup
// are also filtered (WithGroup returns a PIIHandler whose WithAttrs filters).
func TestPIIHandler_WithGroupAttrsFiltered(t *testing.T) {
	server := redactingServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.WithGroup("auth").With("email", "user@example.com").Info("login attempt")

	if out := buf.String(); strings.Contains(out, "user@example.com") {
		t.Errorf("WithGroup+WithAttrs leaked unfiltered email: %q", out)
	}
}
