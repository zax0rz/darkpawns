package main

import "testing"

// The default must stay all-interfaces (":<port>") so existing self-hosted
// installs see no change; an explicit bind confines the listener, which is
// what the reverse-proxy topology wants for /ws, /api and /admin.
func TestListenAddr(t *testing.T) {
	cases := []struct{ bind, port, want string }{
		{"", "4350", ":4350"},
		{"127.0.0.1", "4350", "127.0.0.1:4350"},
		{"::1", "8080", "[::1]:8080"},
	}
	for _, tc := range cases {
		if got := listenAddr(tc.bind, tc.port); got != tc.want {
			t.Errorf("listenAddr(%q, %q) = %q, want %q", tc.bind, tc.port, got, tc.want)
		}
	}
}
