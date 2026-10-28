package apidoc

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetupStatusFollowsSavedCharactersAndFailsUnavailable(t *testing.T) {
	for _, startedEmpty := range []bool{true, false} {
		mux := http.NewServeMux()
		doc := New()
		count := 0
		var err error
		RegisterSetupStatus(doc.NewAPI(mux), func() (int, error) { return count, err }, startedEmpty)
		for _, n := range []int{0, 1, 5} {
			count = n
			r := httptest.NewRecorder()
			mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/setup-status", nil))
			if r.Code != http.StatusOK {
				t.Fatalf("status %d", r.Code)
			}
			if r.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("setup state must not be cached")
			}
			var body SetupStatus
			if e := json.Unmarshal(r.Body.Bytes(), &body); e != nil {
				t.Fatal(e)
			}
			if body.FirstCharacterPending != (n == 0) || body.StartedEmpty != startedEmpty {
				t.Fatalf("status %+v", body)
			}
		}
		err = errors.New("private database path")
		r := httptest.NewRecorder()
		mux.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/setup-status", nil))
		if r.Code != http.StatusServiceUnavailable {
			t.Fatalf("failed read = %d", r.Code)
		}
		if strings.Contains(r.Body.String(), "private database path") {
			t.Fatal("leaked storage details")
		}
		if doc.OpenAPI().Paths["/setup-status"].Get == nil {
			t.Fatal("missing typed public operation")
		}
	}
}
