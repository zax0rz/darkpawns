package main

import "testing"

func TestIsLoopbackBind(t *testing.T) {
	cases := map[string]bool{
		"":            false, // all interfaces — public
		"0.0.0.0":     false,
		"::":          false,
		"192.168.1.5": false,
		"172.93.0.1":  false,
		"127.0.0.1":   true,
		"127.5.4.3":   true,
		"::1":         true,
		"localhost":   true,
		"LOCALHOST":   true,
	}
	for bind, want := range cases {
		if got := isLoopbackBind(bind); got != want {
			t.Errorf("isLoopbackBind(%q) = %v, want %v", bind, got, want)
		}
	}
}

func TestSanitizeBindsForEmptyWorld(t *testing.T) {
	t.Run("empty world downgrades every public bind", func(t *testing.T) {
		http, telnet, tls, whod := "", "0.0.0.0", "::", ""
		sanitizeBindsForEmptyWorld(0, &http, &telnet, &tls, &whod)
		for name, got := range map[string]string{"-http-bind": http, "-telnet-bind": telnet, "-telnet-tls-bind": tls, "-whod-bind": whod} {
			if got != "127.0.0.1" {
				t.Fatalf("%s = %q, want 127.0.0.1", name, got)
			}
		}
	})
	t.Run("populated world keeps binds", func(t *testing.T) {
		http, telnet, tls, whod := "", "0.0.0.0", "", ""
		sanitizeBindsForEmptyWorld(1, &http, &telnet, &tls, &whod)
		if http != "" || telnet != "0.0.0.0" || tls != "" || whod != "" {
			t.Fatalf("populated world altered binds: %q %q %q %q", http, telnet, tls, whod)
		}
	})
	t.Run("loopback binds untouched", func(t *testing.T) {
		http, telnet, tls, whod := "127.0.0.1", "localhost", "::1", "127.0.0.1"
		sanitizeBindsForEmptyWorld(0, &http, &telnet, &tls, &whod)
		if http != "127.0.0.1" || telnet != "localhost" || tls != "::1" || whod != "127.0.0.1" {
			t.Fatalf("loopback binds altered: %q %q %q %q", http, telnet, tls, whod)
		}
	})
}
