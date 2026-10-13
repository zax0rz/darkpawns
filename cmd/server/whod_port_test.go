package main

import (
	"fmt"
	"net"
	"strings"
	"testing"
)

// resolveWhodPort is the guard for the port trap the 2026-10-13 deploy
// pre-flight caught: telnet on 7777, TLS telnet on 7778, no explicit
// -whod-port, so C's derive-telnet+1 rule hands WHOD the TLS port. WHOD binds
// first, the TLS bind then fails fatally, and the process would have
// crash-looped had the pre-flight not given the unit -whod-port 7779 before
// restarting. The collision rows must keep naming the flag whose listener
// owns the port: that refusal is what keeps any TLS-on-telnet+1 host bootable
// without an explicit -whod-port.
func TestResolveWhodPort(t *testing.T) {
	cases := []struct {
		name          string
		whod          int
		telnet        int
		telnetTLS     int
		wantPort      int
		wantCollision string
	}{
		{"C default with no TLS derives telnet+1", -1, 7777, 0, 7778, ""},
		{"production trap: derived telnet+1 is the TLS port", -1, 7777, 7778, 7778, "-telnet-tls-port"},
		{"production shape with an explicit WHOD port", 7779, 7777, 7778, 7779, ""},
		{"explicit WHOD on the TLS port", 7778, 7777, 7778, 7778, "-telnet-tls-port"},
		{"explicit WHOD on the telnet port", 7777, 7777, 0, 7777, "-telnet-port"},
		{"derived port free when TLS sits elsewhere", -1, 7777, 9999, 7778, ""},
		{"0 disables", 0, 7777, 7778, 0, ""},
		{"-1 with telnet disabled derives nothing", -1, 0, 0, 0, ""},
		{"-1 never derives from the TLS port", -1, 0, 7778, 0, ""},
		{"explicit port where a disabled telnet listened is free", 7777, 0, 0, 7777, ""},
	}
	for _, tc := range cases {
		gotPort, gotCollision := resolveWhodPort(tc.whod, tc.telnet, tc.telnetTLS)
		if gotPort != tc.wantPort || gotCollision != tc.wantCollision {
			t.Errorf("%s: resolveWhodPort(%d, %d, %d) = (%d, %q), want (%d, %q)",
				tc.name, tc.whod, tc.telnet, tc.telnetTLS,
				gotPort, gotCollision, tc.wantPort, tc.wantCollision)
		}
	}
}

// adjacentPortPair returns a telnet/TLS port pair (TLS = telnet+1) drawn
// from OS-assigned free ports, so the boot test below never squats on fixed
// ports. Binding both proves they were free; releasing them right before the
// child boots is the race any free-port picker has, and the assertions below
// do not depend on the child's binds succeeding — the WHOD refusal is flag
// arithmetic, so a stolen telnet port changes nothing the test reads.
func adjacentPortPair(t *testing.T) (int, int) {
	t.Helper()
	for i := 0; i < 10; i++ {
		low, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("pick a port: %v", err)
		}
		lowPort := low.Addr().(*net.TCPAddr).Port
		closeErr := low.Close()
		if closeErr != nil {
			t.Fatalf("release port %d: %v", lowPort, closeErr)
		}
		high, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", lowPort+1))
		if err != nil {
			continue // lowPort+1 is busy; take another pair
		}
		highPort := high.Addr().(*net.TCPAddr).Port
		closeErr = high.Close()
		if closeErr != nil {
			t.Fatalf("release port %d: %v", highPort, closeErr)
		}
		return lowPort, highPort
	}
	t.Fatal("no adjacent free port pair in 10 attempts")
	return 0, 0
}

// TestServerBootRefusesWhodOnTheTLSPort pins the wiring, not just the
// arithmetic: with the production port shape (TLS on telnet+1) and no explicit
// -whod-port, boot must log the refusal naming both flags and not start WHOD,
// leaving the TLS port alone. The boot then stops at the missing-certificate
// gate (TELNET_TLS_CERT_FILE unset), so the run is deterministic; without the
// guard it instead logs "WHOD listening" on the TLS port before dying there.
func TestServerBootRefusesWhodOnTheTLSPort(t *testing.T) {
	telnetP, tlsP := adjacentPortPair(t)
	code, out := bootServer(t,
		[]string{
			"-world", parseableWorld(t),
			"-port", "0",
			"-telnet-port", fmt.Sprintf("%d", telnetP),
			"-telnet-tls-port", fmt.Sprintf("%d", tlsP),
		},
		"ENVIRONMENT=development",
		"DATABASE_URL=",
		"TELNET_TLS_CERT_FILE=",
		"TELNET_TLS_KEY_FILE=",
	)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (missing TLS certificate stops the boot):\n%s", code, out)
	}
	if !strings.Contains(out, "WHOD disabled: -whod-port collides with -telnet-tls-port") {
		t.Errorf("expected the WHOD refusal naming both flags, got:\n%s", out)
	}
	if strings.Contains(out, "WHOD listening") {
		t.Errorf("WHOD started despite the collision, got:\n%s", out)
	}
}
