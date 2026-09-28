package apidoc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesMarshaledDocument(t *testing.T) {
	d := New()

	rec := httptest.NewRecorder()
	d.Handler()(rec, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("body is empty, want the marshaled document")
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got["openapi"] == "" {
		t.Fatalf("marshaled document missing openapi version: %v", got)
	}
}

func TestHandlerReturns500WhenMarshalFails(t *testing.T) {
	d := New()
	d.oapi.Extensions = map[string]any{"unsupported": make(chan int)}

	rec := httptest.NewRecorder()
	d.Handler()(rec, httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("body is empty, want a diagnostic")
	}
	if !strings.Contains(rec.Body.String(), "error") {
		t.Fatalf("body = %q, want an error diagnostic", rec.Body.String())
	}
}

func TestJSONReturnsMarshalError(t *testing.T) {
	d := New()
	d.oapi.Extensions = map[string]any{"unsupported": make(chan int)}

	body, err := d.JSON()
	if err == nil {
		t.Fatal("JSON() error = nil, want a marshal error")
	}
	if body != nil {
		t.Fatalf("JSON() body = %q, want nil", body)
	}

	if _, err := d.JSON(); err == nil {
		t.Fatal("second JSON() error = nil, want the cached marshal error")
	}
}
