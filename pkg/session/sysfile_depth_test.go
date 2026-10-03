package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSysfileNameMirrorsCIsAbbrev(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		arg  string
		want string
		ok   bool
	}{
		{name: "empty", arg: "", ok: false},
		{name: "bugs prefix", arg: "b", want: "bugs", ok: true},
		{name: "ideas prefix", arg: "i", want: "ideas", ok: true},
		{name: "todo prefix", arg: "t", want: "todo", ok: true},
		{name: "typos prefix", arg: "ty", want: "typos", ok: true},
		{name: "case insensitive", arg: "BUGS", want: "bugs", ok: true},
		{name: "unknown", arg: "nope", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sysfileName(tt.arg)
			if got != tt.want || ok != tt.ok {
				t.Errorf("sysfileName(%q) = (%q, %t), want (%q, %t)", tt.arg, got, ok, tt.want, tt.ok)
			}
		})
	}
}

// src/db.c:2896-2932: fgets reads 255 bytes, drops the last byte of
// each chunk and appends CRLF, even when a long line has not reached LF.
func TestSysfileReadCChunkBoundary(t *testing.T) {
	s := makeCharSession(t, makeTestManager(t))
	s.player = game.NewPlayer(1, "Sysfile", 8004)
	s.player.SetLevel(40)
	root := t.TempDir()
	s.GetWorld().WorldPath = filepath.Join(root, "world")
	if err := os.Mkdir(filepath.Join(root, "misc"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "misc", "bugs"), []byte(strings.Repeat("a", 255)+"Z\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmdSysfile(s, []string{"bugs"}); err != nil {
		t.Fatal(err)
	}
	got := entryMenuText(t, s)
	want := strings.Repeat("a", 254) + "\r\nZ\r\n"
	if got != want {
		t.Fatalf("C file chunk bytes: got %q, want %q", got, want)
	}
}
