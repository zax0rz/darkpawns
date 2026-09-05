package session

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdAliasDepthManagerBranches(t *testing.T) {
	t.Chdir(t.TempDir())
	m := makeTestManager(t)
	s := makeTestSession(t, m, "AliasDepthTest", 1001, true)

	call := func(args ...string) string {
		t.Helper()
		if err := cmdAlias(s, args); err != nil {
			t.Fatalf("cmdAlias(%v): %v", args, err)
		}
		return readSessionText(t, s)
	}
	callMany := func(args ...string) string {
		t.Helper()
		return call(args...) + readSessionText(t, s)
	}

	if got, want := callMany(), "Currently defined aliases:\r\n None.\r\n"; got != want {
		t.Fatalf("empty list = %q, want %q", got, want)
	}
	if got, want := call("ghost"), "No such alias.\r\n"; got != want {
		t.Fatalf("missing delete = %q, want %q", got, want)
	}
	if got, want := call("g", "get", "all"), "Alias added."; got != want {
		t.Fatalf("add = %q, want %q", got, want)
	}
	if got, want := callMany(), fmt.Sprintf("Currently defined aliases:\r\n%-15s %s\r\n", "g", " get all"); got != want {
		t.Fatalf("formatted list = %q, want %q", got, want)
	}
	if got, want := call("g", "get", "all", "corpse"), "Alias added."; got != want {
		t.Fatalf("redefine = %q, want %q", got, want)
	}
	if len(s.player.Aliases) != 1 || s.player.Aliases[0].Replacement != " get all corpse" || s.player.Aliases[0].Type != game.AliasSimple {
		t.Fatalf("redefine state = %#v", s.player.Aliases)
	}
	if got, want := call("alias", "look"), "You can't alias 'alias'.\r\n"; got != want {
		t.Fatalf("protected name = %q, want %q", got, want)
	}
	if got, want := call("c", "look;score"), "Alias added."; got != want {
		t.Fatalf("complex add = %q, want %q", got, want)
	}
	if len(s.player.Aliases) != 2 || s.player.Aliases[0].Type != game.AliasComplex {
		t.Fatalf("complex alias state = %#v", s.player.Aliases)
	}
	if got, want := call("d", "say", "$$$$"), "Alias added."; got != want {
		t.Fatalf("doubled-dollar add = %q, want %q", got, want)
	}
	if got := s.player.Aliases[0]; got.Replacement != " say $$" || got.Type != game.AliasComplex {
		t.Fatalf("doubled-dollar state = %#v", got)
	}
	if got, want := callMany("z", strings.Repeat("x", 120)), "Maximum alias length is 120 characters. Yours has been truncated.\r\nAlias added."; got != want {
		t.Fatalf("truncated add = %q, want %q", got, want)
	}
	if got, want := call("g"), "Alias deleted."; got != want {
		t.Fatalf("delete = %q, want %q", got, want)
	}
	if got, want := call("g"), "No such alias.\r\n"; got != want {
		t.Fatalf("delete missing after delete = %q, want %q", got, want)
	}

	if _, err := os.Stat("data/aliases/a/aliasdepthtest.alias"); err != nil {
		t.Fatalf("alias file was not persisted: %v", err)
	}
}
