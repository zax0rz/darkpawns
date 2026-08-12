package privacy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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

// failingFilterServer returns a privacy-filter stub that always replies with
// malformed JSON, forcing Client.FilterText to return a real error instead of
// a silent fallback.
func failingFilterServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
}

// TestPIIHandler_FilterErrorPreservesMessage is the regression guard for the
// silent data-loss bug: when FilterText fails (e.g. the filter service returns
// malformed JSON), the original message must survive with a pii_filter_error
// annotation instead of being silently replaced by "" or "[FILTERED]".
func TestPIIHandler_FilterErrorPreservesMessage(t *testing.T) {
	server := failingFilterServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.Info("login attempt")

	out := buf.String()
	if !strings.Contains(out, "login attempt") {
		t.Errorf("filter error destroyed the original message, got: %q", out)
	}
	if !strings.Contains(out, "pii_filter_error") {
		t.Errorf("expected pii_filter_error annotation on filter error, got: %q", out)
	}
	if strings.Contains(out, "[FILTERED]") {
		t.Errorf("message must not be replaced by the fallback sentinel, got: %q", out)
	}
}

// TestPIIHandler_FilterErrorPreservesAttr is the regression guard for attr
// data loss: when FilterText fails, the original string attr value must be kept
// (not silently replaced by "[FILTERED]") and the record must carry a
// pii_filter_error annotation.
func TestPIIHandler_FilterErrorPreservesAttr(t *testing.T) {
	server := failingFilterServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.Info("login attempt", slog.String("email", "user@example.com"))

	out := buf.String()
	if !strings.Contains(out, "user@example.com") {
		t.Errorf("filter error destroyed the string attr, got: %q", out)
	}
	if !strings.Contains(out, "pii_filter_error") {
		t.Errorf("expected pii_filter_error annotation on filter error, got: %q", out)
	}
	if strings.Contains(out, "[FILTERED]") {
		t.Errorf("attr must not be replaced by the fallback sentinel, got: %q", out)
	}
}

// TestPIIHandler_FilterErrorWithAttrs is the regression guard for the
// WithAttrs path: handler-level attrs that fail filtering must be preserved
// and the error must be annotated on subsequently handled records.
func TestPIIHandler_FilterErrorWithAttrs(t *testing.T) {
	server := failingFilterServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.With("email", "user@example.com").Info("login attempt")

	out := buf.String()
	if !strings.Contains(out, "user@example.com") {
		t.Errorf("filter error destroyed the WithAttrs attr, got: %q", out)
	}
	if !strings.Contains(out, "pii_filter_error") {
		t.Errorf("expected pii_filter_error annotation on filter error, got: %q", out)
	}
	if strings.Contains(out, "[FILTERED]") {
		t.Errorf("WithAttrs attr must not be replaced by the fallback sentinel, got: %q", out)
	}
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

// TestPIIHandler_HandleFiltersMixedKindsAndGroups drives a record containing
// string attrs (PII), non-string attrs, and nested groups through the handler
// chain. It asserts the email is masked, the integer passes through, the group
// payload is filtered, and the record reaches the sink (DP-871).
func TestPIIHandler_HandleFiltersMixedKindsAndGroups(t *testing.T) {
	server := redactingServer(t)
	defer server.Close()

	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	client := NewClient(server.URL, DefaultFilterConfig())

	logger := slog.New(NewPIIHandler(inner, client))
	logger.Info(
		"login user@example.com",
		slog.String("email", "user@example.com"),
		slog.Int("level", 42),
		slog.Group(
			"account",
			slog.String("owner", "user@example.com"),
			slog.Bool("active", true),
		),
	)

	out := buf.String()
	if strings.Contains(out, "user@example.com") {
		t.Errorf("Handle leaked unfiltered email: %q", out)
	}
	if !strings.Contains(out, "[REDACTED-EMAIL]") {
		t.Errorf("expected redacted email in output, got: %q", out)
	}
	if !strings.Contains(out, "level=42") {
		t.Errorf("expected non-string attr to pass through, got: %q", out)
	}
	if !strings.Contains(out, "active=true") {
		t.Errorf("expected group bool attr to pass through, got: %q", out)
	}
}

// TestInitSlogPII drives a log record through the global slog handler installed
// by InitSlogPII and asserts PII is masked and the record reaches stdout
// (DP-871).
func TestInitSlogPII(t *testing.T) {
	server := redactingServer(t)
	defer server.Close()

	oldDefault := slog.Default()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w

	InitSlogPII(server.URL)
	slog.Info("contact user@example.com")

	_ = w.Close()
	os.Stdout = oldStdout
	slog.SetDefault(oldDefault)

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	output := string(out)
	if strings.Contains(output, "user@example.com") {
		t.Errorf("InitSlogPII leaked unfiltered email: %q", output)
	}
	if !strings.Contains(output, "[REDACTED-EMAIL]") {
		t.Errorf("expected redacted email in InitSlogPII output, got: %q", output)
	}
}
