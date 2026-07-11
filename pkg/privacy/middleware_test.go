package privacy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestHTTPMiddleware_UsesRequestLocalClient(t *testing.T) {
	// Two filter services that return distinct markers so we can verify each
	// middleware uses its own client instead of mutating a shared logger.
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"filtered_text": "CLIENT-A", "detected_categories": []}`))
	}))
	defer serverA.Close()

	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"filtered_text": "CLIENT-B", "detected_categories": []}`))
	}))
	defer serverB.Close()

	clientA := NewClient(serverA.URL, DefaultFilterConfig())
	clientB := NewClient(serverB.URL, DefaultFilterConfig())

	// Capture stdout while the middlewares run concurrently.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w

	var wg sync.WaitGroup
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("response body"))
	})

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/a", nil)
			rec := httptest.NewRecorder()
			HTTPMiddleware(handler, clientA).ServeHTTP(rec, req)
		}()
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/b", nil)
			rec := httptest.NewRecorder()
			HTTPMiddleware(handler, clientB).ServeHTTP(rec, req)
		}()
	}

	wg.Wait()
	_ = w.Close()
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	output := string(out)

	countA := strings.Count(output, "CLIENT-A")
	countB := strings.Count(output, "CLIENT-B")
	if countA != 50 {
		t.Errorf("expected 50 CLIENT-A outputs, got %d", countA)
	}
	if countB != 50 {
		t.Errorf("expected 50 CLIENT-B outputs, got %d", countB)
	}
}

func TestWebSocketLogger_UsesInstanceClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"filtered_text": "WS-FILTERED", "detected_categories": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, DefaultFilterConfig())

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w

	logger := NewWebSocketLogger(client, "[ws]")
	logger.LogIncoming("session-1", "hello")

	_ = w.Close()
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	if !strings.Contains(string(out), "WS-FILTERED") {
		t.Errorf("expected WebSocket logger output to use its own client, got:\n%s", string(out))
	}
}

// TestWebSocketLogger_LogOutgoingAndEvent drives outgoing and event records
// through the WebSocketLogger and asserts they reach the sink after PII
// filtering (DP-871).
func TestWebSocketLogger_LogOutgoingAndEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"filtered_text": "WS-OUT-FILTERED", "detected_categories": []}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, DefaultFilterConfig())

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w

	logger := NewWebSocketLogger(client, "[ws]")
	logger.LogOutgoing("session-2", "reply to user@example.com")
	logger.LogEvent("session-2", "disconnect", "user@example.com left")

	_ = w.Close()
	os.Stdout = oldStdout

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	output := string(out)
	if strings.Contains(output, "user@example.com") {
		t.Errorf("WebSocket logger leaked unfiltered email: %q", output)
	}
	if !strings.Contains(output, "WS-OUT-FILTERED") {
		t.Errorf("expected filtered output in WebSocket logs, got:\n%s", output)
	}
}

type failingBody struct{}

func (failingBody) Read(p []byte) (int, error) {
	return 0, errors.New("simulated body read failure")
}

func (failingBody) Close() error { return nil }

func TestHTTPMiddleware_BodyReadFailure(t *testing.T) {
	var buf bytes.Buffer
	textHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(textHandler))
	defer slog.SetDefault(oldLogger)

	var receivedBody []byte
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/test", failingBody{})
	rec := httptest.NewRecorder()
	HTTPMiddleware(handler, nil).ServeHTTP(rec, req)

	if len(receivedBody) != 0 {
		t.Errorf("expected downstream handler to receive empty body, got %q", receivedBody)
	}

	logs := buf.String()
	if !strings.Contains(logs, "failed to read request body in privacy middleware") {
		t.Errorf("expected warning log for body read failure, got:\n%s", logs)
	}
}

// TestLoggingResponseWriter_BoundedCapture verifies that a large response
// is written through to the client in full while the buffer kept around
// for logging is capped at maxLoggedBodyBytes (DP-787).
func TestLoggingResponseWriter_BoundedCapture(t *testing.T) {
	const bigSize = 5 * 1024 * 1024 // 5MB, well over maxLoggedBodyBytes
	payload := bytes.Repeat([]byte("B"), bigSize)

	rec := httptest.NewRecorder()
	lrw := NewLoggingResponseWriter(rec)

	n, err := lrw.Write(payload)
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if n != bigSize {
		t.Errorf("expected Write to report %d bytes written, got %d", bigSize, n)
	}

	if rec.Body.Len() != bigSize {
		t.Errorf("expected underlying ResponseWriter to receive %d bytes, got %d", bigSize, rec.Body.Len())
	}
	if !bytes.Equal(rec.Body.Bytes(), payload) {
		t.Errorf("underlying response body content does not match what was written")
	}

	if lrw.body.Len() != maxLoggedBodyBytes {
		t.Errorf("expected captured log buffer to be capped at %d bytes, got %d", maxLoggedBodyBytes, lrw.body.Len())
	}
	if !lrw.truncated {
		t.Errorf("expected truncated flag to be set for a response larger than the cap")
	}
}

// TestCaptureRequestBody_BoundedAndFullPassthrough verifies that a large
// request body is captured only up to maxLoggedBodyBytes for logging, while
// downstream handlers still see the complete, unaltered body (DP-787).
func TestCaptureRequestBody_BoundedAndFullPassthrough(t *testing.T) {
	const bigSize = 5 * 1024 * 1024 // 5MB, well over maxLoggedBodyBytes
	body := bytes.Repeat([]byte("A"), bigSize)
	req := httptest.NewRequest(http.MethodPost, "/big", bytes.NewReader(body))

	captured, truncated := captureRequestBody(req)

	if !truncated {
		t.Errorf("expected truncated=true for a body larger than maxLoggedBodyBytes")
	}
	if len(captured) != maxLoggedBodyBytes {
		t.Errorf("expected captured length %d, got %d", maxLoggedBodyBytes, len(captured))
	}

	full, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading reconstructed body: %v", err)
	}
	if len(full) != bigSize {
		t.Errorf("expected downstream handler to see %d bytes, got %d", bigSize, len(full))
	}
	if !bytes.Equal(full, body) {
		t.Errorf("reconstructed body content does not match original")
	}
}

// TestHTTPMiddleware_LargeBodiesFullyPassedThroughButBoundedInLogs is an
// end-to-end regression test for DP-787: previously the middleware buffered
// entire request/response bodies just to log a 1000-char truncation,
// causing memory spikes on multi-MB uploads/responses. This confirms the
// client still receives the full response, the downstream handler still
// receives the full request body, and the text handed to the PII filter
// (i.e. what gets captured/logged) stays bounded regardless of body size.
func TestHTTPMiddleware_LargeBodiesFullyPassedThroughButBoundedInLogs(t *testing.T) {
	const bigSize = 5 * 1024 * 1024 // 5MB per body

	var mu sync.Mutex
	var loggedTextLen int
	filterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var freq FilterRequest
		if err := json.NewDecoder(r.Body).Decode(&freq); err != nil {
			t.Errorf("filter server failed to decode request: %v", err)
		}
		mu.Lock()
		loggedTextLen = len(freq.Text)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FilterResponse{FilteredText: freq.Text, Detected: []string{}})
	}))
	defer filterServer.Close()

	client := NewClient(filterServer.URL, DefaultFilterConfig())

	reqBody := bytes.Repeat([]byte("A"), bigSize)
	respBody := bytes.Repeat([]byte("B"), bigSize)

	var receivedReqLen int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("downstream handler failed reading body: %v", err)
		}
		receivedReqLen = len(b)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(respBody); err != nil {
			t.Fatalf("handler failed writing response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/big", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()

	HTTPMiddleware(handler, client).ServeHTTP(rec, req)

	// (a) client receives the full response.
	if rec.Body.Len() != bigSize {
		t.Errorf("expected client to receive %d response bytes, got %d", bigSize, rec.Body.Len())
	}
	if !bytes.Equal(rec.Body.Bytes(), respBody) {
		t.Errorf("client response body content does not match what the handler wrote")
	}

	// (b) downstream handler receives the full request body.
	if receivedReqLen != bigSize {
		t.Errorf("expected downstream handler to see %d request bytes, got %d", bigSize, receivedReqLen)
	}

	// (c) the captured/logged text (sent to the PII filter) stays bounded,
	// nowhere near the multi-MB bodies that flowed through the middleware.
	mu.Lock()
	defer mu.Unlock()
	maxExpectedLoggedText := 2*(maxLoggedBodyBytes+len("... [truncated]")) + 512
	if loggedTextLen == 0 {
		t.Errorf("expected non-empty logged text")
	}
	if loggedTextLen > maxExpectedLoggedText {
		t.Errorf("expected logged text to stay bounded (<= %d bytes), got %d bytes", maxExpectedLoggedText, loggedTextLen)
	}
}

// silence unused imports in case helpers are not needed.
var _ = log.New(bytes.NewBuffer(nil), "", 0)
