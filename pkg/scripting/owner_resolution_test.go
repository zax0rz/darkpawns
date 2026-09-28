package scripting

import (
	"os"
	"path/filepath"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// TestRunOwnerTypedScriptResolution proves run_script's owner-typed lookup
// (scripts.c:1748-1775): a room owner loads scripts/room/<name> and an object
// owner loads scripts/obj/<name>, strictly — never the mob/room/obj search,
// and never a flat or mob file with the same name. C builds the path as
// SCRIPT_DIR/type/script_name with no fallback.
func TestRunOwnerTypedScriptResolution(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"mob", "room", "obj"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
		// Every owner directory carries a same-named script that records
		// which one ran in a global the test reads back (test_-prefixed so
		// the engine's post-run cleanup keeps it).
		src := "test_owner = \"" + sub + "\"\nfunction oncmd()\n\treturn 1\nend\n"
		if err := os.WriteFile(filepath.Join(dir, sub, "shared.lua"), []byte(src), 0o644); err != nil {
			t.Fatalf("write %s/shared.lua: %v", sub, err)
		}
	}
	// A flat same-named script must not shadow the typed lookup either.
	if err := os.WriteFile(filepath.Join(dir, "shared.lua"), []byte("test_owner = \"flat\"\nfunction oncmd()\n\treturn 1\nend\n"), 0o644); err != nil {
		t.Fatalf("write flat shared.lua: %v", err)
	}

	engine := NewEngine(dir, nil)
	t.Cleanup(engine.Close)

	run := func(owner string) string {
		t.Helper()
		ctx := &ScriptContext{OwnerType: owner, Argument: "look"}
		handled, err := engine.RunScript(ctx, "shared.lua", "oncmd")
		if err != nil {
			t.Fatalf("room-owner run: %v", err)
		}
		if !handled {
			t.Fatalf("owner=%s: script did not handle", owner)
		}
		got := engine.l.GetGlobal("test_owner")
		if got.Type() != lua.LTString {
			t.Fatalf("owner global type = %v, want string", got.Type())
		}
		return got.String()
	}

	if got := run("room"); got != "room" {
		t.Fatalf("room owner ran %q, want the scripts/room copy", got)
	}
	if got := run("obj"); got != "obj" {
		t.Fatalf("obj owner ran %q, want the scripts/obj copy", got)
	}

	// The mob call sites keep the legacy search (brief: do not change mob
	// resolution): with no OwnerType, the flat file wins.
	ctx := &ScriptContext{Argument: "look"}
	handled, err := engine.RunScript(ctx, "shared.lua", "oncmd")
	if err != nil || !handled {
		t.Fatalf("legacy run: handled=%v err=%v", handled, err)
	}
	if got := engine.l.GetGlobal("test_owner").String(); got != "flat" {
		t.Fatalf("legacy lookup ran %q, want the flat copy", got)
	}
}

// TestOwnerTypeMissingScriptIsolatedFromMobCopy proves the failure cache is
// owner-scoped: a room owner whose script file is missing must fail (and
// cache) without being satisfied by a mob copy of the same name, and without
// poisoning a later obj-owner run of a script that does exist.
func TestOwnerTypeMissingScriptIsolatedFromMobCopy(t *testing.T) {
	dir := t.TempDir()
	for _, sub := range []string{"mob", "room", "obj"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", sub, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "mob", "only-mob.lua"), []byte("function oncmd()\n\treturn 1\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "obj", "real.lua"), []byte("function oncmd()\n\treturn 1\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(dir, nil)
	t.Cleanup(engine.Close)

	// scripts/room/only-mob.lua does not exist; the mob copy must not satisfy
	// the room owner.
	if _, err := engine.RunScript(&ScriptContext{OwnerType: "room"}, "only-mob.lua", "oncmd"); err == nil {
		t.Fatal("room owner found a script; want not-found (no mob fallback)")
	}
	// The obj owner's own script still loads after that failure.
	handled, err := engine.RunScript(&ScriptContext{OwnerType: "obj"}, "real.lua", "oncmd")
	if err != nil || !handled {
		t.Fatalf("obj owner run after failure: handled=%v err=%v", handled, err)
	}
}

// TestDofileScriptsPrefixResolvesInsideSandbox proves dofile("scripts/room/
// x.lua") loads scriptsDir/room/x.lua — the way C's working directory makes
// the literal path work (pattern_3065.lua dofiles pattern_tport.lua this
// way) — and that the traversal guard still rejects escapes.
func TestDofileScriptsPrefixResolvesInsideSandbox(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "room"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "room", "tport.lua"), []byte("tport_loaded = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "caller.lua"), []byte(
		"function oncmd()\n\tdofile(\"scripts/room/tport.lua\")\n\treturn tport_loaded\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	engine := NewEngine(dir, nil)
	t.Cleanup(engine.Close)

	handled, err := engine.RunScript(&ScriptContext{Argument: "say haven"}, "caller.lua", "oncmd")
	if err != nil {
		t.Fatalf("caller run: %v", err)
	}
	// tport_loaded is 1 only if the dofile'd file ran; returning it makes the
	// trigger's own return value the proof.
	if !handled {
		t.Fatal("oncmd did not return tport_loaded=1; dofile of scripts/room/tport.lua did not load")
	}

	// A dofile that tries to escape scriptsDir must not read outside it.
	secret := filepath.Join(dir, "secret.lua")
	if err := os.WriteFile(secret, []byte("test_leaked = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "escape.lua"), []byte(
		"function oncmd()\n\tdofile(\"../secret.lua\")\n\treturn 0\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.RunScript(&ScriptContext{Argument: "x"}, "escape.lua", "oncmd"); err != nil {
		t.Fatalf("escape run: %v", err)
	}
	if got := engine.l.GetGlobal("test_leaked"); got != lua.LNil {
		t.Fatalf("dofile escaped the sandbox; leaked = %v", got)
	}
}
