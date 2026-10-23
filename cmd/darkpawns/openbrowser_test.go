package main

import (
	"context"
	"testing"
)

// L1-010: only plain http(s) to a loopback host is ever handed to the browser.
func TestOpenBrowserRefusesNonLocalURLs(t *testing.T) {
	for _, url := range []string{
		"https://attacker.example/",
		"http://192.168.1.5:4350/",
		"file:///etc/passwd",
		"not a url",
	} {
		if err := openBrowser(context.Background(), url); err == nil {
			t.Fatalf("openBrowser accepted %q", url)
		}
	}
	if err := openBrowser(context.Background(), "http://127.0.0.1:4350/"); err == nil {
		// Reaching a real browser opener in CI must still not happen for the
		// refusal cases; for the loopback case an error here is acceptable
		// (no browser), only acceptance is the failure.
		t.Log("loopback open attempted (no browser in CI is fine)")
	}
}
