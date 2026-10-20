package main

import "net/http"

// The SPA navigates to /login. Serve its existing public entry on direct loads
// without changing any authenticated API route or rewriting the browser URL.
func adminLoginPage(admin http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		entry := r.Clone(r.Context())
		entry.URL.Path = "/admin/"
		admin.ServeHTTP(w, entry)
	})
}
