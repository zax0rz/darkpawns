package privacy

import (
	"bytes"
	"io"
	"log"
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

// silence unused imports in case helpers are not needed.
var _ = log.New(bytes.NewBuffer(nil), "", 0)
