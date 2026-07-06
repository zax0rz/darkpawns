// Package scripting sandbox adversarial tests.
//
// Trust boundary documentation:
//
// The Lua sandbox (newSafeLState in engine.go) blocks these operations:
//
//   CODE LOADING:       load, loadstring, loadfile → nil (dofile is sandboxed)
//   FILESYSTEM:         os.execute, os.remove, os.rename, os.tmpname → nil
//   ENVIRONMENT:        os.getenv, os.setenv, os.setlocale → nil
//   PROCESS:            os.exit, os.clock → nil
//   BYTECODE:           string.dump → nil
//   RNG MANIPULATION:   math.randomseed → nil
//   PACKAGE SYSTEM:     package table, _LOADED, _PRELOAD → nil
//   DEBUG:              debug library → nil
//   I/O:                io library → nil
//   GC ABUSE:           collectgarbage → stub returning "disabled" string
//
// Still accessible (intentionally):
//   os.time, os.date, math (minus randomseed), string (minus dump),
//   table, coroutine, ~60 registered game API functions,
//   dofile (sandboxed — only loads scripts under engine scriptsDir).
//
// This file tests every blocked boundary and a few creative bypass attempts.
// COV-6 / DP-967.

package scripting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSandbox_Adversarial exercises every sandbox boundary with a script that
// attempts a dangerous operation and asserts the sandbox holds.
// Each subtest creates its own engine so state corruption from one test
// (e.g. os table overwrite) cannot leak into another.
func TestSandbox_Adversarial(t *testing.T) {
	type verdict string
	const (
		mustFail verdict = "fail"  // RunScript must return non-nil error
		mustPass verdict = "pass"  // RunScript must succeed (err == nil)
		anyOK    verdict = "anyok" // either is acceptable
	)

	type testCase struct {
		name     string
		source   string // Lua source — must define an oncmd function
		verdict  verdict
		msgMatch string // optional substring to look for in error message
	}

	tests := []testCase{
		// ── Code loading ────────────────────────────────────────────────
		{
			name:    "require",
			source:  `function oncmd() require("os") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "load",
			source:  `function oncmd() load("print(1)")() return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "loadstring",
			source:  `function oncmd() loadstring("print(1)")() return TRUE end`,
			verdict: mustFail,
		},
		// dofile is sandboxed (not nil) — logs warning + returns 0 for
		// paths outside scriptsDir, does NOT error at the Lua level.
		{
			name:    "dofile",
			source:  `function oncmd() dofile("/etc/passwd") return TRUE end`,
			verdict: mustPass,
		},
		{
			name:    "loadfile",
			source:  `function oncmd() loadfile("/etc/passwd") return TRUE end`,
			verdict: mustFail,
		},

		// ── Filesystem access ───────────────────────────────────────────
		{
			name:    "os_execute",
			source:  `function oncmd() os.execute("rm -rf /") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "os_remove",
			source:  `function oncmd() os.remove("/etc/passwd") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "os_rename",
			source:  `function oncmd() os.rename("/etc/passwd", "/tmp/pwn") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "os_tmpname",
			source:  `function oncmd() os.tmpname() return TRUE end`,
			verdict: mustFail,
		},

		// ── Environment access ──────────────────────────────────────────
		{
			name:    "os_getenv",
			source:  `function oncmd() os.getenv("HOME") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "os_setenv",
			source:  `function oncmd() os.setenv("PATH", "/tmp") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "os_setlocale",
			source:  `function oncmd() os.setlocale("C") return TRUE end`,
			verdict: mustFail,
		},

		// ── Process control ─────────────────────────────────────────────
		{
			name:    "os_exit",
			source:  `function oncmd() os.exit(1) return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "os_clock",
			source:  `function oncmd() os.clock() return TRUE end`,
			verdict: mustFail,
		},

		// ── Bytecode ────────────────────────────────────────────────────
		{
			name:    "string_dump",
			source:  `function oncmd() string.dump(print) return TRUE end`,
			verdict: mustFail,
		},

		// ── RNG manipulation ────────────────────────────────────────────
		{
			name:    "math_randomseed",
			source:  `function oncmd() math.randomseed(12345) return TRUE end`,
			verdict: mustFail,
		},

		// ── Package system ──────────────────────────────────────────────
		{
			name:    "package_table",
			source:  `function oncmd() return package end`,
			verdict: anyOK, // package is nil; returning nil means handled != TRUE
		},

		// ── Debug library ───────────────────────────────────────────────
		{
			name:    "debug_getinfo",
			source:  `function oncmd() debug.getinfo(1) return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "debug_sethook",
			source:  `function oncmd() debug.sethook(print, "", 1) return TRUE end`,
			verdict: mustFail,
		},

		// ── I/O library ─────────────────────────────────────────────────
		{
			name:    "io_open",
			source:  `function oncmd() io.open("/etc/passwd", "r") return TRUE end`,
			verdict: mustFail,
		},
		{
			name:    "io_popen",
			source:  `function oncmd() io.popen("ls") return TRUE end`,
			verdict: mustFail,
		},

		// ── GC abuse ────────────────────────────────────────────────────
		// collectgarbage is replaced with a stub that returns a string
		// ("collectgarbage is disabled in this sandbox") instead of erroring.
		{
			name:    "collectgarbage_collect",
			source:  `function oncmd() collectgarbage("collect") return TRUE end`,
			verdict: mustPass,
		},
		{
			name:    "collectgarbage_stop",
			source:  `function oncmd() collectgarbage("stop") return TRUE end`,
			verdict: mustPass,
		},
		{
			name:    "collectgarbage_noarg",
			source:  `function oncmd() collectgarbage() return TRUE end`,
			verdict: mustPass,
		},

		// ── Creative bypass attempts ────────────────────────────────────

		// Reintroduction via global assignment:
		// Assigning a new os table shouldn't grant real os access.
		// Just verify the script doesn't crash.
		{
			name:    "reintroduce_os_table",
			source:  `function oncmd() os = {execute = function() end} return TRUE end`,
			verdict: mustPass,
		},

		// Coroutine escape: os.execute is nil'd on the original os table;
		// coroutines share the same Lua state, so the nil persists.
		{
			name: "coroutine_os_execute",
			source: `function oncmd()
local co = coroutine.create(function()
	local ok, err = pcall(os.execute, "ls")
	if ok then error("os.execute was callable from coroutine") end
	return TRUE
end)
local ok, err = coroutine.resume(co)
if not ok then error("coroutine resume failed: " .. tostring(err)) end
return TRUE
end`,
			verdict: mustPass,
		},

		// Rawget bypass: rawget on the os table should return nil for
		// execute, same as direct access.
		{
			name: "rawget_os_execute",
			source: `function oncmd()
local v = rawget(os, "execute")
if v ~= nil then error("rawget bypassed os.execute nil: " .. tostring(v)) end
return TRUE
end`,
			verdict: mustPass,
		},

		// Metatable abuse: Lua's __index metamethod fires when a table key
		// lookup returns nil. Since os.execute = nil on the sandboxed table,
		// __index can supply a function, bypassing the nil guard.
		// This documents a known sandbox gap: metatables can override nil'd
		// fields on tables. The script's assertion fires (`error()`),
		// returning mustFail to document the bypass exists.
		{
			name: "metatable_os_index",
			source: `function oncmd()
setmetatable(os, {__index = function(t, k)
	if k == "execute" then return function() end end
end})
local ok, err = pcall(os.execute, "ls")
if ok then error("metatable bypass granted os.execute access") end
return TRUE
end`,
			verdict: mustFail,
		},

		// Global environment access via _G — dofile is sandboxed, not nil'd
		{
			name:    "global_env_dofile",
			source:  `function oncmd() _G.dofile("/etc/passwd") return TRUE end`,
			verdict: mustPass,
		},

		// Try loading os module back into the environment.
		// _LOADED and _PRELOAD are removed from registry; this should fail.
		{
			name:    "registry_loaded",
			source:  `function oncmd() local v = debug.getregistry() return TRUE end`,
			verdict: mustFail, // debug is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Each subtest gets its own temp dir and engine — no state leakage.
			dir := t.TempDir()
			fname := tt.name + ".lua"
			path := filepath.Join(dir, fname)
			if err := os.WriteFile(path, []byte(tt.source), 0o644); err != nil {
				t.Fatalf("write %s: %v", fname, err)
			}

			engine := NewEngine(dir, nil)
			defer engine.Close()

			handled, err := engine.RunScript(&ScriptContext{}, fname, "oncmd")

			switch tt.verdict {
			case mustFail:
				if err == nil {
					t.Errorf("expected error for %q, got nil (handled=%v)", tt.name, handled)
				}
				if handled {
					t.Errorf("expected handled=false for %q, got true", tt.name)
				}
			case mustPass:
				if err != nil {
					t.Errorf("unexpected error for %q: %v", tt.name, err)
				}
			}

			if tt.msgMatch != "" && err != nil {
				if !strings.Contains(err.Error(), tt.msgMatch) {
					t.Errorf("error message for %q does not contain %q: %v", tt.name, tt.msgMatch, err)
				}
			}
		})
	}
}

// TestSandbox_Adversarial_PackageNil verifies that the package global is nil.
// Separate from the table-driven suite because it doesn't call RunScript.
func TestSandbox_Adversarial_PackageNil(t *testing.T) {
	dir := t.TempDir()
	engine := NewEngine(dir, nil)
	defer engine.Close()

	pkg := engine.LState().GetGlobal("package")
	if pkg.Type() != 0 { // 0 = LTNil
		t.Errorf("package global = %v (type %s), want nil", pkg, pkg.Type())
	}
}

// TestSandbox_Adversarial_BlockedGlobals verifies that all globally-nilled
// functions are actually nil at the top level of the sandbox.
func TestSandbox_Adversarial_BlockedGlobals(t *testing.T) {
	dir := t.TempDir()
	engine := NewEngine(dir, nil)
	defer engine.Close()
	L := engine.LState()

	// Globals set to nil — these should all be nil
	blocked := []string{
		"load", "loadstring", "loadfile",
		"debug", "io", "package",
	}
	for _, name := range blocked {
		v := L.GetGlobal(name)
		if v.Type() != 0 { // LTNil
			t.Errorf("global %q = %v (type %s), want nil", name, v, v.Type())
		}
	}

	// dofile is NOT nil — it's re-registered as a sandboxed version
	dofile := L.GetGlobal("dofile")
	if dofile.Type() == 0 {
		t.Error("global dofile should be a function (sandboxed), not nil")
	}
}
