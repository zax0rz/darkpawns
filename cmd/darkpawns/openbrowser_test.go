package main

import (
	"context"
	"testing"
)

// L1-010: only plain http(s) to a loopback host is ever handed to the browser.
// The launcher is replaced for the test, so no real browser opens.
func TestOpenBrowserRefusesNonLocalURLs(t *testing.T) {
	var launched []string
	saved := launchBrowser
	launchBrowser = func(_ context.Context, _ string, args ...string) error {
		launched = append(launched, args[len(args)-1])
		return nil
	}
	t.Cleanup(func() { launchBrowser = saved })

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
	if len(launched) != 0 {
		t.Fatalf("refused addresses reached the launcher: %q", launched)
	}

	const loopback = "http://127.0.0.1:4350/"
	if err := openBrowser(context.Background(), loopback); err != nil {
		t.Fatalf("openBrowser refused loopback %q: %v", loopback, err)
	}
	if len(launched) != 1 || launched[0] != loopback {
		t.Fatalf("launcher got %q, want [%q]", launched, loopback)
	}
}
