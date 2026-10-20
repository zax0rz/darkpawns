package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminLoginDirectLoadUsesOnlyPublicSPAEntry(t *testing.T) {
	admin := http.NewServeMux()
	admin.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/" {
			t.Fatalf("unexpected admin route: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	})
	admin.HandleFunc("/admin/players", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	root := http.NewServeMux()
	root.Handle("/login", adminLoginPage(admin))
	root.Handle("/admin/", admin)
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPost} {
		req := httptest.NewRequest(method, "/login?next=staff", nil)
		res := httptest.NewRecorder()
		root.ServeHTTP(res, req)
		want := http.StatusOK
		if method == http.MethodPost {
			want = http.StatusMethodNotAllowed
		}
		if res.Code != want {
			t.Fatalf("%s: %d", method, res.Code)
		}
		if req.URL.Path != "/login" {
			t.Fatal("original request changed")
		}
	}
	denied := httptest.NewRecorder()
	root.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/admin/players", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatal("protected route bypassed")
	}
}
