package apidoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestJSONHappyPath pins the success path: the document marshals, is valid
// JSON, and the cached second call returns the same bytes without error.
func TestJSONHappyPath(t *testing.T) {
	doc := New()

	body, err := doc.JSON()
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
		t.Fatalf("JSON() produced invalid JSON: %v", err)
	}

	again, err := doc.JSON()
	if err != nil {
		t.Fatalf("second JSON() error = %v", err)
	}
	if string(again) != string(body) {
		t.Errorf("cached JSON() differs from first call")
	}
}

// TestHandlerReportsMarshalFailure guards against serving a 200 with an empty
// body when the document cannot be encoded. The marshal error must surface as
// a 500 with a non-empty diagnostic on every request, not be cached as a
// successful nil document.
func TestHandlerReportsMarshalFailure(t *testing.T) {
	doc := New()
	doc.oapi.Extensions = map[string]any{"unmarshalable": make(chan int)}

	if _, err := doc.JSON(); err == nil {
		t.Fatal("JSON() = nil error, want marshal failure")
	}

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		doc.Handler()(rec, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("attempt %d: status = %d, want 500; body: %q", i, rec.Code, rec.Body.String())
		}
		if rec.Body.Len() == 0 {
			t.Fatalf("attempt %d: empty diagnostic body", i)
		}
	}
}
