package privacy

import (
	"os"
	"strings"
	"testing"
)

// TestFilterText_LiveService exercises the client against a real
// privacy-filter service. It skips unless PRIVACY_FILTER_URL points at one,
// which is what `make privacy-test` sets up:
//
//	PRIVACY_FILTER_URL=http://localhost:8001 make privacy-test
//
// Everything else in this package tests against httptest stubs; this is the
// only check that the wire contract survives contact with the actual model
// service (deployment/privacy_filter_api.py).
func TestFilterText_LiveService(t *testing.T) {
	baseURL := os.Getenv("PRIVACY_FILTER_URL")
	if baseURL == "" {
		t.Skip("PRIVACY_FILTER_URL not set; pointing it at a live service enables this test")
	}

	client := NewClient(baseURL, DefaultFilterConfig())
	const input = "Contact alice@example.com or call 555-867-5309 for details."

	filtered, detected, err := client.FilterText(input)
	if err != nil {
		t.Fatalf("FilterText against live service failed: %v", err)
	}
	if filtered == input {
		t.Fatalf("input came back unfiltered; the service is not redacting: %q", filtered)
	}
	if strings.Contains(filtered, "alice@example.com") {
		t.Errorf("email survived filtering: %q", filtered)
	}
	if len(detected) == 0 {
		t.Errorf("no categories detected; got filtered=%q", filtered)
	}
	for _, category := range detected {
		if category == "fallback" {
			t.Errorf("client fell back while the service was reachable: %q", filtered)
		}
	}
}
