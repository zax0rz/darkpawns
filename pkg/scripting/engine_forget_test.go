package scripting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestForgetFailuresAllowsFixedScriptToLoad reproduces the luaedit flow: a
// script that fails to load is negative-cached (DP-903); after an in-game
// save fixes the file, ForgetFailures must let the next run load it instead
// of answering from the cache until reboot.
func TestForgetFailuresAllowsFixedScriptToLoad(t *testing.T) {
	dir := t.TempDir()
	name := "broken-then-fixed.lua"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("this is not lua(\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e := NewEngine(dir, nil)

	if _, err := e.RunScript(&ScriptContext{}, name, ""); err == nil {
		t.Fatal("broken script unexpectedly loaded")
	}
	if _, err := e.RunScript(&ScriptContext{}, name, ""); err == nil || !strings.Contains(err.Error(), "previously failed") {
		t.Fatalf("second run should hit the negative cache, got %v", err)
	}

	if err := os.WriteFile(path, []byte("function fix() end\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e.ForgetFailures()
	if _, err := e.RunScript(&ScriptContext{}, name, ""); err != nil {
		t.Fatalf("fixed script should load after ForgetFailures, got %v", err)
	}
}

// TestForgetFailuresClearsCache asserts the map is actually emptied.
func TestForgetFailuresClearsCache(t *testing.T) {
	dir := t.TempDir()
	e := NewEngine(dir, nil)
	e.mu.Lock()
	e.failedScripts["anything.lua"] = struct{}{}
	e.mu.Unlock()
	e.ForgetFailures()
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.failedScripts) != 0 {
		t.Fatalf("cache not cleared: %v", e.failedScripts)
	}
}
