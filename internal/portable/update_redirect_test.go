package portable

import (
	"net/http"
	"testing"
)

// L1-011: release redirects may stay on the original host or move to
// GitHub's asset CDN, never to an arbitrary HTTPS server.
func TestUpdateClientRedirectAllowlist(t *testing.T) {
	client := updateClient()
	check := func(via, to string) bool {
		req, _ := http.NewRequest(http.MethodGet, to, nil)
		earlier, _ := http.NewRequest(http.MethodGet, via, nil)
		return client.CheckRedirect(req, []*http.Request{earlier}) == nil
	}
	if !check("https://github.com/a", "https://github.com/b") {
		t.Fatal("same-host redirect refused")
	}
	if !check("https://github.com/a", "https://objects.githubusercontent.com/b") {
		t.Fatal("asset-CDN redirect refused")
	}
	if check("https://github.com/a", "https://attacker.example/b") {
		t.Fatal("cross-host redirect to an attacker host allowed")
	}
}
