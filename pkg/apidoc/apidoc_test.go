package apidoc

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONHappyPath(t *testing.T) {
	d := New()
	body, err := d.JSON()
	if err != nil {
		t.Fatalf("JSON() unexpected error: %v", err)
	}
	if len(body) == 0 {
		t.Fatal("JSON() returned empty body")
	}
}

func TestHandlerMarshalFailureIs500(t *testing.T) {
	d := New()
	d.oapi.Extensions = map[string]any{"bad": func() {}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
	d.Handler()(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if rec.Body.Len() == 0 {
		t.Fatal("expected a non-empty diagnostic body")
	}
}

func TestHandlerMarshalFailureIsNotCachedAsSuccess(t *testing.T) {
	d := New()
	d.oapi.Extensions = map[string]any{"bad": func() {}}

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/openapi.json", nil)
		d.Handler()(rec, req)
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("request %d status = %d, want %d", i, rec.Code, http.StatusInternalServerError)
		}
	}
}
