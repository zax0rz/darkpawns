package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestParseSyslogLevelMatchesCPrefixSearch(t *testing.T) {
	tests := []struct {
		input string
		want  int
		ok    bool
	}{
		{input: "o", want: 0, ok: true},
		{input: "BR", want: 1, ok: true},
		{input: "nor", want: 2, ok: true},
		{input: "complete", want: 3, ok: true},
		{input: "", want: 0, ok: false},
		{input: "briefly", want: 0, ok: false},
		{input: "unknown", want: 0, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := parseSyslogLevel(tt.input)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseSyslogLevel(%q) = (%d, %t), want (%d, %t)", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestSetSyslogFlagsClearsPriorLevel(t *testing.T) {
	p := game.NewPlayer(1, "Syslogtest", game.MortalStartRoom)
	setSyslogFlags(p, 1)
	setSyslogFlags(p, 2)
	if got := syslogLevel(p.GetFlags()); got != "normal" {
		t.Fatalf("syslog after Brief to Normal = %q, want normal", got)
	}
	setSyslogFlags(p, 3)
	if got := syslogLevel(p.GetFlags()); got != "complete" {
		t.Fatalf("syslog after Complete = %q, want complete", got)
	}
	setSyslogFlags(p, 0)
	if got := syslogLevel(p.GetFlags()); got != "off" {
		t.Fatalf("syslog after Off = %q, want off", got)
	}
}
