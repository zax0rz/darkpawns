// Package scripting provides Lua scripting support for Dark Pawns MUD.
// Based on original C code from scripts.c.
//
//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package scripting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	lua "github.com/yuin/gopher-lua"
	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Engine manages the Lua VM.
// Based on boot_lua() in scripts.c lines 1703-1716.
type Engine struct {
	scriptsDir string
	l          *lua.LState
	mu         sync.Mutex
	world      ScriptableWorld
	// activeBridge is the game bridge for the script now running, set by
	// RunScript for the length of a bridged run (see bridge.go).
	activeBridge Bridge
	// owner is the goroutine now running a script (0 when none), so a nested
	// run from inside a binding is recognised instead of deadlocking on mu.
	owner atomic.Uint64
	// recreatePending is set when a nested run crashes the Lua state; the
	// outermost run replaces the state once the outer script has returned.
	recreatePending bool
	transitItems    map[int]*transitEntry // in-flight items moved by objfrom/objto (key = instance ID)
	// failedScripts is a negative cache of scripts that failed to load
	// (file-not-found, Lua parse/compile errors, load timeouts). The first
	// failure is logged at slog.Error; subsequent RunScript calls for the
	// same script return immediately with a cheap error and no log line,
	// so a missing script can't flood the log every pulse. Path-traversal
	// rejections are intentionally NOT cached — those should keep logging.
	// Accessed only under e.mu.
	failedScripts map[string]struct{}
	// failedScriptKinds keeps C's open_lua_file kind for each cached load
	// failure that has a C counterpart (timeouts are absent). C retries and
	// logs on every run_script call; the cache skips the retry but still
	// emits the producers (src/scripts.c:1674-1694, 1777-1779).
	failedScriptKinds map[string]string
	done              chan struct{}
	closeOnce         sync.Once

	// Script execution budget. Scripting is single-threaded (e.mu is held for
	// the whole of RunScript) to match the C game loop, so one script blocks all
	// others for its entire duration. scriptTimeout is the hard context deadline
	// that interrupts a runaway script; slowScriptThreshold is the hold time
	// above which a completed script is logged (so slow/abusive scripts are
	// visible — DP-702). Tunable via SetScriptBudget; read under e.mu.
	scriptTimeout       time.Duration
	slowScriptThreshold time.Duration
}

const (
	// defaultScriptTimeout is the hard per-script execution budget. A tight loop
	// with no yield is interrupted at ~this via the LState context deadline.
	defaultScriptTimeout = 5 * time.Second
	// defaultSlowScriptThreshold is well above a normal trigger (microseconds to
	// low-ms) but far below the hard timeout, so it flags genuinely slow scripts
	// without spamming on legitimate ones.
	defaultSlowScriptThreshold = 250 * time.Millisecond
)

// LState returns the underlying Lua state. The caller must NOT hold the engine mutex
// when calling into the LState — use RunScript or the lua* methods instead.
// This accessor exists only for code that needs read-only access outside of script execution.
func (e *Engine) LState() *lua.LState {
	return e.l
}

// newSafeLState creates a fresh LState with all sandboxing applied:
// standard libraries opened, dangerous functions removed, and custom
// API functions registered. Used both for initial engine creation and
// for state recreation after a script timeout or crash.
// Lua Script Security Model
//
// Scripts are operator-authored (immortal/builder), loaded from the server
// filesystem. Players cannot inject or upload scripts. The sandbox prevents
// buggy or malicious operator scripts from:
//
//   - Accessing the filesystem (io, os.execute, etc.)
//   - Loading arbitrary code (dofile, load, require)
//   - Exporting bytecode (string.dump)
//   - Escaping the Lua VM (package, debug)
//   - Forcing garbage collection cycles (collectgarbage)
//
// Scripts CAN access: math, string, table, coroutine, os.time/os.date, and
// ~60 registered game API functions (act, say, spell, oload, steal, etc.).
//
// NOTE: gopher-lua v1.1.2 does not support SetAllowance (memory ceiling) or
// SetInstructionLimit. Memory is bounded by the OS and the 5s wall-clock
// timeout prevents runaway scripts in practice. Both limits should be added
// when gopher-lua is upgraded.
func (e *Engine) newSafeLState() *lua.LState {
	L := lua.NewState()

	// Open standard libraries
	L.OpenLibs()

	// Sandbox dangerous functions — replace with stubs rather than nil.
	// Nilling a table field allows Lua __index metamethods to intercept the
	// nil lookup and supply a replacement function, bypassing the sandbox.
	// Stubs return a string error message (like collectgarbage) so scripts
	// that try to use them get a clear "disabled" response instead of a crash.
	// Global-level removals (whole libraries) use nil since metatables on
	// globals are not a standard Lua pattern.

	disabledFunc := func(name string) *lua.LFunction {
		return L.NewFunction(func(L *lua.LState) int {
			L.Push(lua.LString(name + " is disabled in this sandbox"))
			return 1
		})
	}

	// File system access — load arbitrary code
	// dofile is re-registered as a sandboxed version in registerFunctionsOn()
	// (restricted to engine scriptsDir), so we nil it here to let the
	// sandboxed version take over. The others have no safe variant.
	L.SetGlobal("loadfile", disabledFunc("loadfile"))
	L.SetGlobal("load", disabledFunc("load"))
	L.SetGlobal("loadstring", disabledFunc("loadstring"))

	// OS access — filesystem, process control, environment
	if osTable := L.GetGlobal("os"); osTable.Type() == lua.LTTable {
		tb := osTable.(*lua.LTable)
		tb.RawSetString("clock", disabledFunc("os.clock"))     // DoS: timing-detection busy loop
		tb.RawSetString("execute", disabledFunc("os.execute")) // arbitrary command execution
		tb.RawSetString("exit", disabledFunc("os.exit"))       // crash the server
		tb.RawSetString("getenv", disabledFunc("os.getenv"))   // information disclosure
		tb.RawSetString("remove", disabledFunc("os.remove"))   // file deletion
		tb.RawSetString("rename", disabledFunc("os.rename"))   // file manipulation
		tb.RawSetString("setenv", disabledFunc("os.setenv"))   // affect other processes
		tb.RawSetString("setlocale", disabledFunc("os.setlocale"))
		tb.RawSetString("tmpname", disabledFunc("os.tmpname")) // temp file creation
	}

	// string.dump — produces bytecode that can exploit VM bugs
	if stringTable := L.GetGlobal("string"); stringTable.Type() == lua.LTTable {
		if tb, ok := stringTable.(*lua.LTable); ok {
			tb.RawSetString("dump", disabledFunc("string.dump"))
		}
	}

	// math.randomseed — with a known seed a script can predict or
	// break randomness for all subsequent scripts sharing the LState.
	if mathTable := L.GetGlobal("math"); mathTable.Type() == lua.LTTable {
		if tb, ok := mathTable.(*lua.LTable); ok {
			tb.RawSetString("randomseed", disabledFunc("math.randomseed"))
		}
	}

	// Remove package library (can load arbitrary code)
	L.SetGlobal("package", lua.LNil)
	// Also clear the package loaders from the registry so require() cannot work.
	L.SetField(L.Get(lua.RegistryIndex), "_LOADED", lua.LNil)
	L.SetField(L.Get(lua.RegistryIndex), "_PRELOAD", lua.LNil)

	// Remove debug library
	L.SetGlobal("debug", lua.LNil)

	// Remove io library
	L.SetGlobal("io", lua.LNil)

	// Block collectgarbage to prevent GC abuse (DoS vector — forced GC cycles)
	L.SetGlobal("collectgarbage", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString("collectgarbage is disabled in this sandbox"))
		return 1
	}))

	// Register our custom functions on the fresh state
	e.registerFunctionsOn(L)

	// Load globals.lua
	e.loadGlobalsOn(L)

	return L
}

// matchKeyword checks if a search string matches any keyword in a space-separated keyword list.
// Mirrors C's isname_with_abbrevs() behavior: case-insensitive prefix match.
//
//nolint:unused // Reserved for inworld() mob search when implemented
func matchKeyword(keywords, search string) bool {
	search = strings.ToLower(strings.TrimSpace(search))
	if search == "" {
		return false
	}
	for _, kw := range strings.Fields(keywords) {
		if strings.HasPrefix(strings.ToLower(kw), search) {
			return true
		}
	}
	return false
}

// NewEngine creates a new Lua scripting engine.
func NewEngine(scriptsDir string, world ScriptableWorld) *Engine {
	engine := &Engine{
		scriptsDir:          scriptsDir,
		transitItems:        make(map[int]*transitEntry),
		failedScripts:       make(map[string]struct{}),
		failedScriptKinds:   make(map[string]string),
		world:               world,
		done:                make(chan struct{}),
		scriptTimeout:       defaultScriptTimeout,
		slowScriptThreshold: defaultSlowScriptThreshold,
	}

	// Create a properly sandboxed LState
	engine.l = engine.newSafeLState()

	// Start transitItems cleanup goroutine — items orphaned for >5s are logged and removed.
	go engine.cleanTransitItems()

	return engine
}

// SetScriptBudget tunes the per-script execution budget. timeout is the hard
// context deadline after which a running script is interrupted and its LState
// recreated; slowThreshold is the wall-clock hold time above which a completed
// script is logged as slow. Because scripting is single-threaded (to match the
// C game loop), a script blocks all others for its whole duration — lowering
// the timeout bounds a runaway script's impact; the slow log surfaces offenders.
// Non-positive values leave the corresponding setting unchanged.
// ForgetFailures clears the negative cache of failed script loads. The cache
// assumes failures are stable per file (DP-903); an in-game save under the
// scripts tree invalidates that assumption wholesale, so luaedit clears it
// after every successful write or delete.
func (e *Engine) ForgetFailures() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.failedScripts = make(map[string]struct{})
	e.failedScriptKinds = make(map[string]string)
}

func (e *Engine) SetScriptBudget(timeout, slowThreshold time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if timeout > 0 {
		e.scriptTimeout = timeout
	}
	if slowThreshold > 0 {
		e.slowScriptThreshold = slowThreshold
	}
}

const transitItemTTL = 30 * time.Second

type transitEntry struct {
	obj      ScriptableObject
	placedAt time.Time
}

// cleanTransitItems periodically removes orphaned items from the transit map.
func (e *Engine) cleanTransitItems() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			e.mu.Lock()
			for id, entry := range e.transitItems {
				if time.Since(entry.placedAt) > transitItemTTL {
					slog.Warn("transitItem orphaned, discarding", "instanceID", id, "vnum", entry.obj.GetVNum())
					delete(e.transitItems, id)
				}
			}
			e.mu.Unlock()
		case <-e.done:
			return
		}
	}
}

// Close shuts down the background cleanup goroutine and releases the Lua VM.
// Safe to call multiple times.
func (e *Engine) Close() {
	e.closeOnce.Do(func() {
		close(e.done)
		if e.l != nil {
			e.l.Close()
		}
	})
}

// cleanupScriptGlobalsLocked removes any global keys introduced during script execution
// that were not present in knownGlobals. This prevents script-defined functions, variables,
// and tables from leaking between RunScript calls. Caller must hold e.mu.
func (e *Engine) cleanupScriptGlobalsLocked(L *lua.LState, knownGlobals map[string]struct{}) {
	gt := L.Get(lua.GlobalsIndex)
	if tbl, ok := gt.(*lua.LTable); ok {
		var cleanupKeys []string
		tbl.ForEach(func(key, _ lua.LValue) {
			if k, ok := key.(lua.LString); ok {
				sk := string(k)
				if _, found := knownGlobals[sk]; !found {
					// Keep per-run context globals set by this RunScript — they
					// will be cleared at the top of the next run anyway.
					// Also keep globals prefixed with "test_" for integration tests
					// that check script results via GetGlobal after RunScript returns.
					if sk != "ch" && sk != "me" && sk != "obj" && sk != "argument" && sk != "room" &&
						!strings.HasPrefix(sk, "test_") {
						cleanupKeys = append(cleanupKeys, sk)
					}
				}
			}
		})
		for _, k := range cleanupKeys {
			L.SetGlobal(k, lua.LNil)
		}
	}
}

// resolveScriptPath looks for a script file in scriptsDir, then in known
// subdirectories (mob/, room/, obj/). Matches C's SCRIPT_DIR/type/script_name
// pattern from scripts.c:1775.
func (e *Engine) resolveScriptPath(cleanName string) string {
	return ResolveScriptPath(e.scriptsDir, cleanName)
}

// ResolveScriptPath is the engine's script_name lookup, exported so webOLC's
// usage lookup and "edit script" link resolve a name to the same file the
// next trigger will run. It returns "" when nothing matches.
func ResolveScriptPath(scriptsDir, cleanName string) string {
	// 1. Direct lookup (flat — globals.lua lives here)
	direct := filepath.Join(scriptsDir, cleanName)
	if _, err := os.Stat(direct); err == nil {
		return direct
	}
	// 2. Search known subdirectories (matches C's type parameter)
	for _, sub := range []string{"mob", "room", "obj"} {
		candidate := filepath.Join(scriptsDir, sub, cleanName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "" // not found
}

// ResolveOwnerScriptPath is run_script's strict owner-typed lookup
// (scripts.c:1775): C builds SCRIPT_DIR/type/script_name, so a room owner
// loads only scripts/room/<name> and an object owner only scripts/obj/<name>,
// never the mob/room/obj search the legacy mob call sites use. It returns ""
// when the file does not exist.
func ResolveOwnerScriptPath(scriptsDir, ownerType, cleanName string) string {
	if ownerType != "room" && ownerType != "obj" {
		return ""
	}
	path := filepath.Join(scriptsDir, ownerType, cleanName)
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}

// RunScript loads and executes a named trigger function in a script file.
// fname is relative to scriptsDir (e.g. "mob/144/hisc.lua").
// triggerName is the function to call (e.g. "oncmd", "sound", "fight").
// Returns true if the script handled the event (returned TRUE), false otherwise.
// Based on run_script() in scripts.c lines 1718-1810.
// mudlog types and the immortal level for the script producers (utils.h:114-117
// for BRF 1 / NRM 2 / CMP 3, structs.h:620 for LVL_IMMORT 31).
const (
	scriptMudlogBrief    = 1
	scriptMudlogComplete = 3
	scriptMudlogImmortal = 31
)

// scriptMudLogBrief emits a C script diagnostic at BRF/LVL_IMMORT/file FALSE
// through the active bridge, which is Bridge.Log's contract
// (pkg/game/world_bridge.go:470-473). Every production run is bridged; a
// bridge-less run (an engine-only test) emits nothing.
func (e *Engine) scriptMudLogBrief(msg string) {
	if b := e.activeBridge; b != nil {
		b.Log(msg)
	}
}

// scriptMudLogFile emits a C script producer that also writes the file. The
// adapter's MudLog is a one-line forward to game.MudLog
// (pkg/game/world_bridge.go); the engine reaches it by inline assertion, so the
// Bridge interface and every other implementation are unchanged. Every
// production run has an adapter; a run without one emits nothing.
//
// Called from RunScript while it holds the engine mutex (e.mu) and no world,
// player, manager or lifecycle lock: the producers run before any bridge
// write-back, and the emit itself is a stateless forward.
func scriptMudLogFile(bridge Bridge, msg string, typ int) {
	logger, ok := bridge.(interface{ MudLog(string, int, int, bool) })
	if !ok {
		return
	}
	logger.MudLog(msg, typ, scriptMudlogImmortal, true)
}

// luaLoadErrorKind maps a gopher-lua load error onto the semantic kinds C's
// open_lua_file names (src/scripts.c:1674-1692). C's numbers are Lua 4's
// lua_dofile codes; the port maps kinds, not numbers. It returns "" for a
// Go-only failure or one C has no message for, so no payload is invented.
func luaLoadErrorKind(err error) string {
	var apiErr *lua.ApiError
	if !errors.As(err, &apiErr) {
		return ""
	}
	switch apiErr.Type {
	case lua.ApiErrorFile:
		return "No such file."
	case lua.ApiErrorSyntax:
		return "Syntax error."
	case lua.ApiErrorRun:
		return "Execution failed."
	case lua.ApiErrorError:
		return "Generic Error."
	default:
		return ""
	}
}

// scriptMudLogLoadFailure emits C's two load-failure producers in C's order:
// open_lua_file's classified line (src/scripts.c:1674-1694, CMP/31/file TRUE),
// then run_script's own line (src/scripts.c:1777-1779, BRF/31/file TRUE).
func scriptMudLogLoadFailure(bridge Bridge, name, kind string) {
	if kind != "" {
		scriptMudLogFile(bridge, fmt.Sprintf("[Lua] Could not call script %s: %s", name, kind), scriptMudlogComplete)
	}
	scriptMudLogFile(bridge, fmt.Sprintf("SYSERR: Error opening lua script %s.", name), scriptMudlogBrief)
}

// scriptMudLogCallFailure emits run_script's failed-call producer
// (src/scripts.c:1798-1800, BRF/31/file TRUE).
func scriptMudLogCallFailure(bridge Bridge, name, triggerName string) {
	scriptMudLogFile(bridge, fmt.Sprintf("[Lua] Script %s being called with an error, function '%s'.", name, triggerName), scriptMudlogBrief)
}

func (e *Engine) RunScript(ctx *ScriptContext, fname string, triggerName string) (handled bool, err error) {
	// C's run_script nests: a script's action(), raw_kill() or give can reach
	// another script (ongive, death) while the first is still running, on the
	// same Lua state. A RunScript from the goroutine already running a script
	// is that nested call. It runs on the outer run's lock, deadline and
	// state, and leaves the globals it set behind for the outer script, as
	// C does.
	gid := goroutineID()
	nested := e.owner.Load() == gid
	if !nested {
		e.mu.Lock()
		e.owner.Store(gid)
		defer func() {
			e.owner.Store(0)
			e.mu.Unlock()
		}()
	}

	// Measure how long this script holds the engine. Because scripting is
	// single-threaded, this duration is exactly how long every other script was
	// blocked; log it when it exceeds the threshold so slow/abusive scripts are
	// visible (DP-702). Registered before the panic defer so it still runs (and
	// includes any LState recreation) but inside the unlock defer.
	scriptStart := time.Now()
	defer func() {
		if d := time.Since(scriptStart); d > e.slowScriptThreshold {
			slog.Warn("slow script held the scripting engine",
				"file", fname, "trigger", triggerName,
				"duration", d, "budget", e.scriptTimeout)
		}
	}()

	// Recover from Lua panics (instruction limit, context timeout, Go triggers, etc.)
	// and recreate the LState so a single poisoned script doesn't corrupt the engine.
	var needsRecreate bool
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("lua script panic, recreating LState", "reason", r, "file", fname, "trigger", triggerName)
			needsRecreate = true
			err = fmt.Errorf("lua script panic: %v", r)
		}
		if needsRecreate && nested {
			// The outer script is still running on this state; the outermost
			// run replaces it when it returns.
			e.recreatePending = true
			needsRecreate = false
		}
		if !nested && e.recreatePending {
			e.recreatePending = false
			needsRecreate = true
		}
		if needsRecreate {
			slog.Info("recreating Lua state after script crash", "file", fname)
			e.l.Close()
			e.l = e.newSafeLState()
		}
	}()

	L := e.l

	// A run with C-level references goes through the game bridge: C's tables
	// for ch, me, room and obj (run_script, scripts.c:1727-1746) and C's
	// write-back afterwards. Owner-typed runs bridge too: an object's onpulse
	// passes ch = me = NULL (comm.c:789-793), so the bridge cannot hinge on
	// MeRef alone.
	var bridge Bridge
	if ctx.MeRef != nil || ctx.OwnerType != "" {
		if b, ok := ctx.World.(Bridge); ok && ctx.World != nil {
			bridge = b
		} else if b, ok := e.world.(Bridge); ok {
			bridge = b
		}
	}
	// Clear any per-run globals left over from a previous script execution so
	// stale context and trigger functions cannot leak into this run. C's
	// run_script sets ch, me, room, obj and argument only when it has them
	// and never clears them, so a bridged run leaves them alone.
	clearGlobals := []string{"ch", "me", "obj", "argument", "room"}
	if bridge != nil {
		clearGlobals = nil
	}
	if triggerName != "" {
		clearGlobals = append(clearGlobals, triggerName)
	}
	for _, name := range clearGlobals {
		L.SetGlobal(name, lua.LNil)
	}

	outerBridge := e.activeBridge
	e.activeBridge = bridge
	defer func() { e.activeBridge = outerBridge }()
	if bridge != nil {
		if ctx.ChRef != nil {
			L.SetGlobal("ch", e.charToTable(bridge, *ctx.ChRef))
		}
		// run_script sets me only when non-NULL (scripts.c:1733-1736); the
		// object onpulse site passes NULL and C leaves the previous run's me
		// global in place.
		if ctx.MeRef != nil {
			L.SetGlobal("me", e.charToTable(bridge, *ctx.MeRef))
		}
		if ctx.RoomVNum > 0 {
			L.SetGlobal("room", e.roomToTable(bridge, ctx.RoomVNum, ctx.MeRef))
		}
		if ctx.ObjRef != nil {
			L.SetGlobal("obj", e.cObjToTable(bridge, *ctx.ObjRef))
		}
		if ctx.Argument != "" {
			L.SetGlobal("argument", lua.LString(ctx.Argument))
		}
	}

	// Set globals based on context
	// Based on run_script() lines 1732-1761
	if bridge == nil && ctx.Ch != nil {
		e.charToTableLocked(ctx.Ch, "ch")
		slog.Debug("set ch global", "player", ctx.Ch.GetName())
	} else {
		slog.Debug("ctx.Ch is nil")
	}
	if bridge == nil && ctx.Me != nil {
		e.mobToTableLocked(ctx.Me, "me")
	}
	if bridge == nil && ctx.Obj != nil {
		e.objToTableLocked(ctx.Obj, "obj")
	}
	if bridge == nil && ctx.Argument != "" {
		L.SetGlobal("argument", lua.LString(ctx.Argument))
	}

	// Set room global if we have room vnum
	if bridge == nil && ctx.RoomVNum > 0 {
		// Create a room table with vnum and char array
		roomTbl := e.l.NewTable()
		roomTbl.RawSetString("vnum", lua.LNumber(ctx.RoomVNum))

		charTbl := L.NewTable()
		idx := 1
		if e.world != nil {
			for _, p := range e.world.GetPlayersInRoom(ctx.RoomVNum) {
				pt := L.NewTable()
				pt.RawSetString("name", lua.LString(p.GetName()))
				pt.RawSetString("level", lua.LNumber(p.GetLevel()))
				pt.RawSetString("hp", lua.LNumber(p.GetHealth()))
				pt.RawSetString("maxhp", lua.LNumber(p.GetMaxHealth()))
				pt.RawSetString("evil", lua.LBool(p.GetAlignment() < -350))
				pt.RawSetString("pos", lua.LNumber(combat.PosStanding)) // POS_STANDING
				charTbl.RawSetInt(idx, pt)
				idx++
			}
			for _, m := range e.world.GetMobsInRoom(ctx.RoomVNum) {
				mt := L.NewTable()
				mt.RawSetString("name", lua.LString(m.GetName()))
				mt.RawSetString("level", lua.LNumber(m.GetLevel()))
				mt.RawSetString("hp", lua.LNumber(m.GetHealth()))
				mt.RawSetString("maxhp", lua.LNumber(m.GetMaxHealth()))
				// Check mob alignment (negative = evil)
				alignment := m.GetPrototype().GetAlignment()
				mt.RawSetString("evil", lua.LBool(alignment < 0))
				mt.RawSetString("pos", lua.LNumber(combat.PosStanding)) // POS_STANDING
				mt.RawSetString("vnum", lua.LNumber(m.GetVNum()))
				mt.RawSetString("room", lua.LNumber(ctx.RoomVNum))
				charTbl.RawSetInt(idx, mt)
				idx++
			}
		}
		roomTbl.RawSetString("char", charTbl)

		L.SetGlobal("room", roomTbl)
	}

	// Validate path: sanitize fname, reject traversal attempts.
	// Subdirectory resolution below only searches known folders under scriptsDir.
	cleanName := filepath.Clean(fname)
	if strings.Contains(cleanName, "..") || strings.HasPrefix(cleanName, "/") {
		slog.Error("path traversal blocked in RunScript", "fname", fname)
		return false, fmt.Errorf("path traversal blocked: %s", fname)
	}

	// Owner-typed runs resolve strictly as SCRIPT_DIR/type/script_name
	// (scripts.c:1775); the mob-era search stays below for mob call sites.
	// The negative-cache key carries the owner type because one name can
	// exist for one owner and be missing for another.
	cacheKey := cleanName
	var scriptPath string
	if ctx.OwnerType == "room" || ctx.OwnerType == "obj" {
		cacheKey = ctx.OwnerType + "/" + cleanName
		scriptPath = ResolveOwnerScriptPath(e.scriptsDir, ctx.OwnerType, cleanName)
	} else {
		scriptPath = e.resolveScriptPath(cleanName)
	}

	// Negative cache: if this script previously failed to load (file-not-found,
	// parse error, load timeout), skip the disk hit and log entirely. The first
	// failure was already logged when it was added to failedScripts. Per-pulse
	// retries on a missing script would otherwise flood the log (DP-903).
	if _, failed := e.failedScripts[cacheKey]; failed {
		if kind, logged := e.failedScriptKinds[cacheKey]; logged {
			scriptMudLogLoadFailure(bridge, fname, kind)
		}
		return false, fmt.Errorf("script %s previously failed to load", fname)
	}

	if scriptPath == "" {
		e.failedScripts[cacheKey] = struct{}{}
		slog.Error("error loading script", "file", fname, "error", "script not found")
		// C reaches the same case inside lua_dofile (src/scripts.c:1677-1678
		// "No such file.") and logs both producers before returning.
		e.failedScriptKinds[cacheKey] = "No such file."
		scriptMudLogLoadFailure(bridge, fname, "No such file.")
		return false, fmt.Errorf("script not found: %s", fname)
	}

	// Snapshot known global keys before loading the script so we can remove
	// script-defined globals after execution, preventing context leaks between runs.
	knownGlobals := make(map[string]struct{})
	gt := L.Get(lua.GlobalsIndex)
	if tbl, ok := gt.(*lua.LTable); ok {
		tbl.ForEach(func(key, _ lua.LValue) {
			if k, ok := key.(lua.LString); ok {
				knownGlobals[string(k)] = struct{}{}
			}
		})
	}

	// Load and execute the script file
	// Based on open_lua_file() in scripts.c lines 1666-1699
	// Execution timeout prevents tight loops from hanging the server indefinitely.

	// A nested run keeps the outer run's deadline and stack frame.
	scriptCancel := context.CancelFunc(func() {})
	if !nested {
		var scriptCtx context.Context
		scriptCtx, scriptCancel = context.WithTimeout(context.Background(), e.scriptTimeout)
		L.SetContext(scriptCtx)
	}
	defer scriptCancel()
	baseTop := L.GetTop()
	releaseContext := func() {
		L.SetTop(baseTop)
		if !nested {
			L.RemoveContext()
		}
		scriptCancel()
	}

	if err := L.DoFile(scriptPath); err != nil {
		// Negative-cache the script so subsequent pulses skip the disk hit
		// and the error log. Both timeout and file-not-found/parse errors are
		// stable per file — they won't fix themselves between pulses (DP-903).
		e.failedScripts[cacheKey] = struct{}{}
		timeout := errors.Is(err, context.DeadlineExceeded)
		if timeout {
			slog.Error("script timed out during load", "file", fname, "error", err)
			needsRecreate = true
		} else {
			slog.Error("error loading script", "file", fname, "error", err)
			// scripts.c:1674-1694 then 1777-1779. A Go-only timeout has no C
			// kind, so it emits neither line rather than an invented one.
			kind := luaLoadErrorKind(err)
			e.failedScriptKinds[cacheKey] = kind
			scriptMudLogLoadFailure(bridge, fname, kind)
		}
		e.cleanupScriptGlobalsLocked(L, knownGlobals)
		releaseContext()
		return false, err
	}

	// Call the trigger function
	// Based on run_script() lines 1780-1795
	L.SetTop(baseTop) // lua_settop(L, top_of_stack): drop what the file returned
	fn := L.GetGlobal(triggerName)
	slog.Debug("calling function", "trigger", triggerName, "type", fn.Type())
	L.Push(fn)
	if fn.Type() == lua.LTNil {
		// Function doesn't exist
		L.Pop(1)
		// scripts.c:1791-1800: C calls the global, the call fails, and the
		// producer fires before the write-back below.
		scriptMudLogCallFailure(bridge, fname, triggerName)
		e.cleanupScriptGlobalsLocked(L, knownGlobals)
		releaseContext()
		slog.Debug("function not found in script", "trigger", triggerName, "file", fname)
		return false, nil
	}

	if err := L.PCall(0, 1, nil); err != nil {
		// scripts.c:1798-1800 logs the failed call; C writes back afterwards.
		// A Go-only timeout is not a C call failure and emits no C payload.
		if !errors.Is(err, context.DeadlineExceeded) {
			scriptMudLogCallFailure(bridge, fname, triggerName)
		}
		if bridge != nil {
			// C logs the failed call and still writes back (scripts.c:1788-1816).
			e.bridgeWriteBack(bridge, ctx)
		}
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("script timed out during execution", "trigger", triggerName, "file", fname, "error", err)
			needsRecreate = true
		} else {
			slog.Error("error calling function", "trigger", triggerName, "file", fname, "error", err)
		}
		e.cleanupScriptGlobalsLocked(L, knownGlobals)
		releaseContext()
		return false, err
	}

	// Get return value
	stackTop := L.GetTop()
	slog.Debug("stack top after PCall", "top", stackTop)

	var ret lua.LValue = lua.LFalse
	if stackTop > baseTop {
		ret = L.Get(-1)
		slog.Debug("function returned", "type", ret.Type(), "value", ret)
		L.Pop(1)
	}

	if bridge != nil {
		e.bridgeWriteBack(bridge, ctx)
		e.cleanupScriptGlobalsLocked(L, knownGlobals)
		releaseContext()
		// retval = (int)lua_tonumber(L, -1): any nonzero number is handled.
		return int(lua.LVAsNumber(ret)) != 0, nil
	}

	// Legacy write-back, reached only without a bridge: engine tests that build
	// a world without one. Every server run bridges (cmd/server passes the
	// WorldScriptableAdapter), and C's write-back is bridgeWriteBack's (ch
	// only). Do not route a live path here.
	if ctx.Ch != nil {
		slog.Debug("reading back ch changes", "stack_top", L.GetTop())
		chVal := L.GetGlobal("ch")
		slog.Debug("ch global type", "type", chVal.Type())
		if chVal.Type() != lua.LTNil {
			L.Push(chVal)
			e.tableToCharLocked(ctx.Ch)
			slog.Debug("after tableToChar", "stack_top", L.GetTop())
			if L.GetTop() > 0 {
				L.Pop(1)
			}
		}
	}

	if ctx.Me != nil {
		meVal := L.GetGlobal("me")
		slog.Debug("me global type", "type", meVal.Type())
		if meVal.Type() != lua.LTNil {
			L.Push(meVal)
			e.tableToMobLocked(ctx.Me)
			if L.GetTop() > 0 {
				L.Pop(1)
			}
		}
	}

	e.cleanupScriptGlobalsLocked(L, knownGlobals)

	releaseContext()

	// Check return value. Both numeric 1 and boolean true mean "handled".
	if ret.Type() == lua.LTNumber {
		return lua.LVAsNumber(ret) == 1, nil
	}
	if ret == lua.LTrue {
		return true, nil
	}
	return false, nil
}

// registerFunctions registers all Lua API functions.
// Based on cmdlib array in scripts.c lines 1609-1668.
// registerFunctionsOn registers all Lua API functions on the given LState.
// This is separate from the state setup so it can be called on a fresh state
// after a timeout-induced recreation.
func (e *Engine) registerFunctionsOn(L *lua.LState) {
	// Core functions mentioned in the task
	L.SetGlobal("act", L.NewFunction(e.bridged(e.bridgeAct)))
	L.SetGlobal("say", L.NewFunction(e.bridged(e.bridgeSay)))
	L.SetGlobal("gossip", L.NewFunction(e.bridged(e.bridgeGossip)))
	L.SetGlobal("emote", L.NewFunction(e.bridged(e.bridgeEmote)))
	L.SetGlobal("action", L.NewFunction(e.bridged(e.bridgeAction)))
	L.SetGlobal("oload", L.NewFunction(e.bridged(e.bridgeOLoad)))
	L.SetGlobal("mload", L.NewFunction(e.bridged(e.bridgeMLoad)))
	L.SetGlobal("extobj", L.NewFunction(e.bridged(e.bridgeExtObj)))
	L.SetGlobal("extchar", L.NewFunction(e.bridged(e.bridgeExtChar)))
	L.SetGlobal("number", L.NewFunction(e.luaNumber))
	L.SetGlobal("strlower", L.NewFunction(e.luaStrlower))
	// Lua 4's strfind, strsub and gsub are Lua 5.1's string.find, string.sub
	// and string.gsub: same arguments, same (multiple) results. tonumber is
	// the base library's own.
	stringLib, _ := L.GetGlobal("string").(*lua.LTable)
	for global, field := range map[string]string{"strfind": "find", "strsub": "sub", "gsub": "gsub"} {
		if stringLib != nil {
			L.SetGlobal(global, stringLib.RawGetString(field))
		}
	}
	L.SetGlobal("format", L.NewFunction(luaFormat))
	L.SetGlobal("getn", L.NewFunction(e.luaGetn))
	// Don't override tostring - it's a Lua built-in
	// L.SetGlobal("tostring", L.NewFunction(e.luaTostring))

	// Additional functions from cmdlib that might be needed
	L.SetGlobal("log", L.NewFunction(e.bridged(e.bridgeLog)))
	L.SetGlobal("raw_kill", L.NewFunction(e.bridged(e.bridgeRawKill)))
	L.SetGlobal("save_char", L.NewFunction(e.bridged(e.bridgeSaveChar)))
	L.SetGlobal("save_obj", L.NewFunction(e.bridged(e.bridgeSaveObj)))
	// dofile/call: shared-script delegation pattern used by cityguard, breed_killer, etc.
	// NOTE: dofile is re-registered below after being nilled in newSafeLState().
	// This is intentional — each script file is loaded into its own sandboxed Lua state,
	// so re-registration is expected behavior.
	L.SetGlobal("dofile", L.NewFunction(e.luaDofile))
	L.SetGlobal("call", L.NewFunction(e.luaCall))
	L.SetGlobal("save_room", L.NewFunction(e.bridged(e.bridgeSaveRoom)))
	L.SetGlobal("set_skill", L.NewFunction(e.bridged(e.bridgeSetSkill)))
	L.SetGlobal("spell", L.NewFunction(e.bridged(e.bridgeSpell)))
	L.SetGlobal("tport", L.NewFunction(e.bridged(e.bridgeTport)))

	// Additional functions needed for combat AI scripts
	L.SetGlobal("isfighting", L.NewFunction(e.luaIsFighting))
	L.SetGlobal("round", L.NewFunction(luaRound4))

	// Functions needed for RESTORE scripts
	L.SetGlobal("objfrom", L.NewFunction(e.bridged(e.bridgeObjFrom)))
	L.SetGlobal("objto", L.NewFunction(e.bridged(e.bridgeObjTo)))
	L.SetGlobal("obj_extra", L.NewFunction(e.bridged(e.bridgeObjExtra)))
	L.SetGlobal("tell", L.NewFunction(e.bridged(e.bridgeTell)))
	L.SetGlobal("plr_flagged", L.NewFunction(e.bridged(e.bridgePlrFlagged)))
	L.SetGlobal("cansee", L.NewFunction(e.bridged(e.bridgeCanSee)))
	L.SetGlobal("isnpc", L.NewFunction(e.bridged(e.bridgeIsNPC)))
	L.SetGlobal("aff_flagged", L.NewFunction(e.bridged(e.bridgeAffFlagged)))
	L.SetGlobal("plr_flags", L.NewFunction(e.bridged(e.actFlagsBinding("plr_flags"))))
	L.SetGlobal("obj_list", L.NewFunction(e.bridged(e.bridgeObjList)))

	// Stubs needed by Tier 3 Economy scripts
	L.SetGlobal("item_check", L.NewFunction(e.luaItemCheck))
	L.SetGlobal("load_room", L.NewFunction(e.bridged(e.bridgeLoadRoom)))
	L.SetGlobal("inworld", L.NewFunction(e.bridged(e.bridgeInWorld)))
	L.SetGlobal("mob_flagged", L.NewFunction(e.bridged(e.bridgeMobFlagged)))
	L.SetGlobal("aff_flags", L.NewFunction(e.bridged(e.bridgeAffFlags)))
	L.SetGlobal("follow", L.NewFunction(e.bridged(e.bridgeFollow)))
	L.SetGlobal("mount", L.NewFunction(e.bridged(e.bridgeMount)))
	L.SetGlobal("direction", L.NewFunction(e.luaDirection))
	L.SetGlobal("set_hunt", L.NewFunction(e.bridged(e.bridgeSetHunt)))
	L.SetGlobal("ishunt", L.NewFunction(e.bridged(e.bridgeIsHunt)))
	L.SetGlobal("skip_spaces", L.NewFunction(e.luaSkipSpaces))
	L.SetGlobal("social", L.NewFunction(e.bridged(e.bridgeSocial)))
	L.SetGlobal("obj_flagged", L.NewFunction(e.bridged(e.bridgeObjFlagged)))
	L.SetGlobal("mob_flags", L.NewFunction(e.bridged(e.actFlagsBinding("mob_flags"))))
	L.SetGlobal("exit_flagged", L.NewFunction(e.bridged(e.bridgeExitFlagged)))
	L.SetGlobal("exit_flags", L.NewFunction(e.bridged(e.bridgeExitFlags)))
	L.SetGlobal("unaffect", L.NewFunction(e.bridged(e.bridgeUnaffect)))
	L.SetGlobal("equip_char", L.NewFunction(e.bridged(e.bridgeEquipChar)))
	// echo(ch, type, msg) — zone-wide sound broadcast. Used by werewolf.lua.
	L.SetGlobal("echo", L.NewFunction(e.bridged(e.bridgeEcho)))

	// Stubs needed by Batch C Quest/Mechanic NPC scripts
	L.SetGlobal("extra", L.NewFunction(e.bridged(e.bridgeExtra)))
	L.SetGlobal("strlen", L.NewFunction(e.luaStrlen))
	L.SetGlobal("iscorpse", L.NewFunction(e.luaIsCorpse))
	L.SetGlobal("canget", L.NewFunction(e.luaCanGet))
	L.SetGlobal("steal", L.NewFunction(e.bridged(e.bridgeSteal)))
}

// loadGlobals loads the globals.lua file.
// Based on boot_lua() lines 1711-1714.
// loadGlobalsOn loads the globals.lua file onto the given LState.
func (e *Engine) loadGlobalsOn(L *lua.LState) {
	globalsPath := e.scriptsDir + "/globals.lua"
	slog.Debug("loading globals", "path", globalsPath)
	if err := L.DoFile(globalsPath); err != nil {
		slog.Warn("could not load globals.lua", "error", err)
	} else {
		slog.Debug("globals.lua loaded successfully")
	}
	e.setupBasicConstantsOn(L)
	// boot_lua() then calls default() (scripts.c:1713-1714), which defines
	// the constants and dofile()s scripts/mob/no_move.lua. It runs after the
	// engine's own constants so C's definitions are the ones scripts see. The
	// call happens after DoFile returns, so the dofile inside it is not the
	// re-entrant DoFile that crashed (DP-600).
	if fn, ok := L.GetGlobal("default").(*lua.LFunction); ok {
		if err := L.CallByParam(lua.P{Fn: fn, NRet: 0, Protect: true}); err != nil {
			slog.Warn("globals.lua default() failed", "error", err)
		}
	}
}

// setupBasicConstantsOn sets up essential constants on the given LState.
func (e *Engine) setupBasicConstantsOn(L *lua.LState) {
	// Direction constants
	L.SetGlobal("NORTH", lua.LNumber(0))
	L.SetGlobal("EAST", lua.LNumber(1))
	L.SetGlobal("SOUTH", lua.LNumber(2))
	L.SetGlobal("WEST", lua.LNumber(3))
	L.SetGlobal("UP", lua.LNumber(4))
	L.SetGlobal("DOWN", lua.LNumber(5))

	// Message types for act()
	L.SetGlobal("TO_ROOM", lua.LNumber(1))
	L.SetGlobal("TO_VICT", lua.LNumber(2))
	L.SetGlobal("TO_NOTVICT", lua.LNumber(3))
	L.SetGlobal("TO_CHAR", lua.LNumber(4))

	// Boolean constants
	L.SetGlobal("TRUE", lua.LNumber(1))
	L.SetGlobal("FALSE", lua.LNumber(0))
	L.SetGlobal("NIL", lua.LNil)

	// Level constants
	L.SetGlobal("LVL_IMMORT", lua.LNumber(combat.LVL_IMMORT))
	L.SetGlobal("LVL_IMPL", lua.LNumber(40))

	// Player flags
	L.SetGlobal("PLR_OUTLAW", lua.LNumber(0))
	L.SetGlobal("PLR_WEREWOLF", lua.LNumber(16))
	L.SetGlobal("PLR_VAMPIRE", lua.LNumber(17))

	// Mob flags
	L.SetGlobal("MOB_SENTINEL", lua.LNumber(1))
	L.SetGlobal("MOB_HUNTER", lua.LNumber(18))
	L.SetGlobal("MOB_MOUNTABLE", lua.LNumber(21))

	// Affect flags
	L.SetGlobal("AFF_DETECT_MAGIC", lua.LNumber(4))
	L.SetGlobal("AFF_GROUP", lua.LNumber(8))
	L.SetGlobal("AFF_POISON", lua.LNumber(11))
	L.SetGlobal("AFF_CHARM", lua.LNumber(21))
	L.SetGlobal("AFF_FLY", lua.LNumber(26))
	L.SetGlobal("AFF_WEREWOLF", lua.LNumber(27))
	L.SetGlobal("AFF_VAMPIRE", lua.LNumber(28))
	L.SetGlobal("AFF_MOUNT", lua.LNumber(29))

	// Position constants
	L.SetGlobal("POS_DEAD", lua.LNumber(combat.PosDead))
	L.SetGlobal("POS_MORTALLYW", lua.LNumber(1))
	L.SetGlobal("POS_INCAP", lua.LNumber(combat.PosIncap))
	L.SetGlobal("POS_STUNNED", lua.LNumber(combat.PosStunned))
	L.SetGlobal("POS_SLEEPING", lua.LNumber(combat.PosSleeping))
	L.SetGlobal("POS_RESTING", lua.LNumber(combat.PosResting))
	L.SetGlobal("POS_SITTING", lua.LNumber(combat.PosSitting))
	L.SetGlobal("POS_STANDING", lua.LNumber(combat.PosStanding))

	// Item type constants
	L.SetGlobal("ITEM_STAFF", lua.LNumber(4))
	L.SetGlobal("ITEM_WEAPON", lua.LNumber(5))
	L.SetGlobal("ITEM_ARMOR", lua.LNumber(9))
	L.SetGlobal("ITEM_WORN", lua.LNumber(11))
	L.SetGlobal("ITEM_TRASH", lua.LNumber(13))
	L.SetGlobal("ITEM_NOTE", lua.LNumber(16))
	L.SetGlobal("ITEM_DRINKCON", lua.LNumber(17))
	L.SetGlobal("ITEM_KEY", lua.LNumber(18))
	L.SetGlobal("ITEM_FOOD", lua.LNumber(19))
	L.SetGlobal("ITEM_PEN", lua.LNumber(21))

	// Object extra flags
	L.SetGlobal("ITEM_GLOW", lua.LNumber(0))
	L.SetGlobal("ITEM_MAGIC", lua.LNumber(6))
	L.SetGlobal("ITEM_NODROP", lua.LNumber(7))
	L.SetGlobal("ITEM_NOSELL", lua.LNumber(16))

	// Item wear positions
	L.SetGlobal("ITEM_WEAR_TAKE", lua.LNumber(0))

	// Spell constants (from spells.h and globals.lua)
	L.SetGlobal("SPELL_TELEPORT", lua.LNumber(2))
	L.SetGlobal("SPELL_BLINDNESS", lua.LNumber(4))
	L.SetGlobal("SPELL_BURNING_HANDS", lua.LNumber(5))
	L.SetGlobal("SPELL_CHARM", lua.LNumber(7))
	L.SetGlobal("SPELL_COLOR_SPRAY", lua.LNumber(10))
	L.SetGlobal("SPELL_CURE_LIGHT", lua.LNumber(16))
	L.SetGlobal("SPELL_CURSE", lua.LNumber(17))
	L.SetGlobal("SPELL_DISPEL_EVIL", lua.LNumber(22))
	L.SetGlobal("SPELL_EARTHQUAKE", lua.LNumber(23))
	L.SetGlobal("SPELL_ENCHANT_WEAPON", lua.LNumber(24))
	L.SetGlobal("SPELL_FIREBALL", lua.LNumber(26))
	L.SetGlobal("SPELL_HARM", lua.LNumber(27))
	L.SetGlobal("SPELL_HEAL", lua.LNumber(28))
	L.SetGlobal("SPELL_LIGHTNING_BOLT", lua.LNumber(30))
	L.SetGlobal("SPELL_MAGIC_MISSILE", lua.LNumber(32))
	L.SetGlobal("SPELL_POISON", lua.LNumber(33))
	L.SetGlobal("SPELL_SANCTUARY", lua.LNumber(36))
	L.SetGlobal("SPELL_SHOCKING_GRASP", lua.LNumber(37))
	L.SetGlobal("SPELL_SLEEP", lua.LNumber(38))
	L.SetGlobal("SPELL_METEOR_SWARM", lua.LNumber(41))
	L.SetGlobal("SPELL_WORD_OF_RECALL", lua.LNumber(42))
	L.SetGlobal("SPELL_REMOVE_POISON", lua.LNumber(43))
	L.SetGlobal("SPELL_DISPEL_GOOD", lua.LNumber(46))
	L.SetGlobal("SPELL_HELLFIRE", lua.LNumber(58))
	L.SetGlobal("SPELL_ENCHANT_ARMOR", lua.LNumber(59))
	L.SetGlobal("SPELL_IDENTIFY", lua.LNumber(60))
	L.SetGlobal("SPELL_MINDBLAST", lua.LNumber(62))
	L.SetGlobal("SPELL_INVULNERABILITY", lua.LNumber(66))
	L.SetGlobal("SPELL_VITALITY", lua.LNumber(67))
	L.SetGlobal("SPELL_ACID_BLAST", lua.LNumber(75))
	L.SetGlobal("SPELL_DIVINE_INT", lua.LNumber(81))
	L.SetGlobal("SPELL_MIND_BAR", lua.LNumber(82))
	L.SetGlobal("SPELL_SOUL_LEECH", lua.LNumber(83))
	L.SetGlobal("SPELL_DISRUPT", lua.LNumber(92))
	L.SetGlobal("SPELL_DISINTEGRATE", lua.LNumber(93))
	L.SetGlobal("SPELL_FLAMESTRIKE", lua.LNumber(96))
	L.SetGlobal("SPELL_PSIBLAST", lua.LNumber(100))
	L.SetGlobal("SPELL_PETRIFY", lua.LNumber(104))

	// Dragon Breath spells
	L.SetGlobal("SPELL_FIRE_BREATH", lua.LNumber(202))
	L.SetGlobal("SPELL_GAS_BREATH", lua.LNumber(203))
	L.SetGlobal("SPELL_FROST_BREATH", lua.LNumber(204))
	L.SetGlobal("SPELL_ACID_BREATH", lua.LNumber(205))
	L.SetGlobal("SPELL_LIGHTNING_BREATH", lua.LNumber(206))

	// Skill constants
	L.SetGlobal("SKILL_BASH", lua.LNumber(132))
	L.SetGlobal("SKILL_HEADBUTT", lua.LNumber(141))
	L.SetGlobal("SKILL_BERSERK", lua.LNumber(171))
	L.SetGlobal("SKILL_PARRY", lua.LNumber(172))

	// Raw kill types
	L.SetGlobal("TYPE_UNDEFINED", lua.LNumber(-1))

	// Sector types
	L.SetGlobal("SECT_FOREST", lua.LNumber(3))
	L.SetGlobal("SECT_UNDERWATER", lua.LNumber(8))
	L.SetGlobal("SECT_FIRE", lua.LNumber(11))
	L.SetGlobal("SECT_EARTH", lua.LNumber(12))
	L.SetGlobal("SECT_WIND", lua.LNumber(13))
	L.SetGlobal("SECT_WATER", lua.LNumber(14))

	// Exit flags
	L.SetGlobal("EX_ISDOOR", lua.LNumber(0))
	L.SetGlobal("EX_CLOSED", lua.LNumber(1))
	L.SetGlobal("EX_LOCKED", lua.LNumber(2))
	L.SetGlobal("EX_PICKPROOF", lua.LNumber(3))

	// Lua script flags
	L.SetGlobal("LT_MOB", lua.LString("mob"))
	L.SetGlobal("LT_OBJ", lua.LString("obj"))
	L.SetGlobal("LT_ROOM", lua.LString("room"))
}

// charToTableLocked converts a ScriptablePlayer to a Lua table. Caller must hold e.mu.
// Based on char_to_table() in scripts.c lines 1812-1916.
// charToTableLocked converts a ScriptablePlayer to a Lua table. Caller must hold e.mu.
func (e *Engine) charToTableLocked(player ScriptablePlayer, globalName string) {
	L := e.l
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("name", lua.LString(player.GetName()))
	tbl.RawSetString("level", lua.LNumber(player.GetLevel()))
	tbl.RawSetString("hp", lua.LNumber(player.GetHealth()))
	tbl.RawSetString("maxhp", lua.LNumber(player.GetMaxHealth()))
	tbl.RawSetString("gold", lua.LNumber(player.GetGold()))
	tbl.RawSetString("race", lua.LNumber(player.GetRace()))
	tbl.RawSetString("class", lua.LNumber(player.GetClass()))
	tbl.RawSetString("alignment", lua.LNumber(player.GetAlignment()))
	tbl.RawSetString("room", lua.LNumber(player.GetRoomVNum()))
	// move/maxmove not yet on ScriptablePlayer interface — skip for now

	// Evil property (based on alignment - negative = evil)
	alignment := player.GetAlignment()
	evil := 0 // FALSE
	if alignment < 0 {
		evil = 1 // TRUE
	}
	tbl.RawSetString("evil", lua.LNumber(evil))

	// is_npc: false for players. Source: utils.h IS_NPC() macro.
	tbl.RawSetString("is_npc", lua.LBool(false))

	// Expose raw PLR flags bitmask so plr_flagged() can check individual bits.
	// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
	tbl.RawSetString("plr_flags_raw", lua.LNumber(float64(player.GetFlags())))

	// Skills table (stub for now)
	skillsTbl := L.NewTable()
	tbl.RawSetString("skills", skillsTbl)

	// Objs — inventory items. Source: scripts.c char_to_table() lines 1842-1849.
	if items := player.GetInventoryItems(); len(items) > 0 {
		objsTbl := L.NewTable()
		for i, obj := range items {
			objsTbl.RawSetInt(i+1, e.objToTable(obj))
		}
		tbl.RawSetString("objs", objsTbl)
	}

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(player.GetID()))

	L.SetGlobal(globalName, tbl)
	slog.Debug("set global", "name", globalName, "level", player.GetLevel())
}

// mobToTableLocked converts a ScriptableMob to a Lua table. Caller must hold e.mu.
// Based on char_to_table() for NPCs in scripts.c lines 1904-1910.
func (e *Engine) mobToTableLocked(mob ScriptableMob, globalName string) {
	L := e.l
	tbl := L.NewTable()

	proto := mob.GetPrototype()

	// Basic fields
	tbl.RawSetString("name", lua.LString(proto.GetShortDesc()))
	tbl.RawSetString("level", lua.LNumber(proto.GetLevel()))
	tbl.RawSetString("hp", lua.LNumber(mob.GetHealth()))
	tbl.RawSetString("maxhp", lua.LNumber(mob.GetMaxHealth()))
	tbl.RawSetString("vnum", lua.LNumber(mob.GetVNum()))
	tbl.RawSetString("gold", lua.LNumber(proto.GetGold()))
	tbl.RawSetString("room", lua.LNumber(mob.GetRoomVNum()))

	// Evil property (based on alignment - negative = evil)
	// In Dark Pawns, alignment ranges from -1000 to +1000
	// Negative alignment = evil (TRUE), positive = good (FALSE)
	alignment := proto.GetAlignment()
	evil := 0 // FALSE
	if alignment < 0 {
		evil = 1 // TRUE
	}
	tbl.RawSetString("evil", lua.LNumber(evil))

	// is_npc: true for mobs. Source: utils.h IS_NPC() macro — MOB_ISNPC flag always set.
	tbl.RawSetString("is_npc", lua.LBool(true))

	// Wear property (array of worn items) - placeholder empty table
	wearTbl := L.NewTable()
	tbl.RawSetString("wear", wearTbl)

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(mob.GetVNum()))
	// Retain the actual body for combat reads; other legacy bindings keep their
	// separately owned numeric struct contract until their C1 port.
	combatBody := L.NewUserData()
	combatBody.Value = mob
	tbl.RawSetString("__combat_body", combatBody)

	L.SetGlobal(globalName, tbl)
}

// objToTable creates a Lua table from a ScriptableObject, suitable for nesting
// or as a standalone value. Caller must hold e.mu.
// Based on obj_to_table() in scripts.c lines 1918-2016.
func (e *Engine) objToTable(obj ScriptableObject) *lua.LTable {
	L := e.l
	tbl := L.NewTable()

	// Basic fields — matching C obj_to_table
	tbl.RawSetString("name", lua.LString(obj.GetShortDesc()))
	tbl.RawSetString("alias", lua.LString(obj.GetKeywords()))
	tbl.RawSetString("vnum", lua.LNumber(obj.GetVNum()))
	tbl.RawSetString("cost", lua.LNumber(obj.GetCost()))
	tbl.RawSetString("type", lua.LNumber(obj.GetTypeFlag()))
	tbl.RawSetString("timer", lua.LNumber(obj.GetTimer()))
	tbl.RawSetString("perc_load", lua.LNumber(0)) // Default 0% load chance

	// obj_id — unique instance ID for runtime reference (steal, etc.)
	// The C version stores a raw struct pointer via lua_pushuserdata.
	// We store the integer ID and look up via World.GetObjByInstanceID().
	tbl.RawSetString("obj_id", lua.LNumber(obj.GetInstanceID()))

	// struct field — for C compatibility, store vnum (legacy scripts may reference it)
	tbl.RawSetString("struct", lua.LNumber(obj.GetVNum()))

	return tbl
}

// objToTableLocked converts a ScriptableObject to a Lua table and sets it as a global.
// Caller must hold e.mu.
func (e *Engine) objToTableLocked(obj ScriptableObject, globalName string) {
	tbl := e.objToTable(obj)
	e.l.SetGlobal(globalName, tbl)
}

// tableToCharLocked reads back changes from the ch table. Caller must hold e.mu.
func (e *Engine) tableToCharLocked(player ScriptablePlayer) {
	L := e.l
	slog.Debug("tableToChar", "stack_top", L.GetTop())
	tbl := L.Get(-1)
	slog.Debug("tableToChar tbl type", "type", tbl.Type())

	if tbl.Type() != lua.LTTable {
		slog.Debug("tableToChar: not a table, returning")
		return
	}

	// Read hp changes
	hpVal := L.GetField(tbl, "hp")
	slog.Debug("tableToChar hp field", "value", hpVal, "type", hpVal.Type())
	if hpVal.Type() == lua.LTNumber {
		player.SetHealth(int(hpVal.(lua.LNumber)))
	}

	// Read gold changes
	goldVal := L.GetField(tbl, "gold")
	slog.Debug("tableToChar gold field", "value", goldVal, "type", goldVal.Type())
	if goldVal.Type() == lua.LTNumber {
		player.SetGold(int(goldVal.(lua.LNumber)))
	}
}

// tableToMobLocked reads back changes from the me table. Caller must hold e.mu.
func (e *Engine) tableToMobLocked(mob ScriptableMob) {
	L := e.l
	tbl := L.Get(-1)

	if tbl.Type() != lua.LTTable {
		return
	}

	// Read hp changes
	L.GetField(tbl, "hp")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		mob.SetHealth(int(lua.LVAsNumber(val)))
	}
	L.Pop(1)
}

// Lua function implementations
// Based on corresponding functions in scripts.c

func (e *Engine) luaNumber(L *lua.LState) int {
	// number(low, high)
	// Based on lua_number() in scripts.c lines 817-830
	low := L.ToInt(1)
	high := L.ToInt(2)

	if low > high {
		low, high = high, low
	}

	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	result := dprng.Number(low, high)
	L.Push(lua.LNumber(result))
	return 1
}

func (e *Engine) luaStrlower(L *lua.LState) int {
	// strlower(s)
	// Lua 4 compat function
	s := L.ToString(1)
	L.Push(lua.LString(strings.ToLower(s)))
	return 1
}

func (e *Engine) luaGetn(L *lua.LState) int {
	// Lua 4.0's getn (lbuiltin.c luaB_getn, ltable.c luaA_getn): the table's
	// "n" field when it is a number, otherwise its largest numeric key. A
	// non-table argument is an error, as in Lua 4 ("table expected"), which
	// ends the script: C's assembler.lua relies on getn(me.objs) raising
	// when the mobile carries nothing and me.objs is nil.
	tbl := L.CheckTable(1)
	if n, ok := tbl.RawGetString("n").(lua.LNumber); ok {
		L.Push(lua.LNumber(int(n)))
		return 1
	}
	maxKey := 0.0
	tbl.ForEach(func(k, _ lua.LValue) {
		if n, ok := k.(lua.LNumber); ok && float64(n) > maxKey {
			maxKey = float64(n)
		}
	})
	L.Push(lua.LNumber(int(maxKey)))
	return 1
}

func (e *Engine) luaIsFighting(L *lua.LState) int {
	// isfighting(mob) - returns the mob's current combat target as a table, or nil
	// Based on lua_isfighting() in scripts.c — checks mob's fighting pointer
	mobTbl := L.Get(1)
	if mobTbl.Type() != lua.LTTable {
		// scripts.c:670-673: a non-table argument logs at BRF/LVL_IMMORT/file
		// FALSE and returns no values; the port returned 1 with a nil.
		e.scriptMudLogBrief("[Lua] Invalid argument passed to lua_isfighting.")
		return 0
	}
	tbl := mobTbl.(*lua.LTable)
	if ref, ok := charRefOf(tbl); ok {
		if bridge, ok := e.world.(interface{ CombatTargetName(CharRef) (string, bool) }); ok {
			if name, found := bridge.CombatTargetName(ref); found {
				tgt := L.NewTable()
				tgt.RawSetString("name", lua.LString(name))
				tgt.RawSetString("level", lua.LNumber(1))
				L.Push(tgt)
				return 1
			}
		}
	}

	if handle, ok := tbl.RawGetString("__combat_body").(*lua.LUserData); ok {
		if body, ok := handle.Value.(interface{ GetFightingBody() combat.Combatant }); ok {
			if target := body.GetFightingBody(); target != nil {
				tgt := L.NewTable()
				tgt.RawSetString("name", lua.LString(target.GetName()))
				tgt.RawSetString("level", lua.LNumber(1)) // Preserve the existing unsupported target-table shape.
				L.Push(tgt)
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaDofile(L *lua.LState) int {
	// dofile(path) - load and execute a Lua file, used for shared script delegation
	// Source: cityguard.lua, breed_killer.lua — dofile+call pattern for shared AI
	path := L.ToString(1)
	if path == "" {
		return 0
	}
	// C runs with lib/ as its working directory and SCRIPT_DIR "scripts"
	// (db.h:79), so C scripts name shared files "scripts/mob/no_move.lua".
	path = strings.TrimPrefix(filepath.ToSlash(path), "scripts/")
	fullPath := filepath.Clean(filepath.Join(e.scriptsDir, path))
	rel, err := filepath.Rel(e.scriptsDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/") {
		slog.Warn("dofile: path traversal blocked", "path", path)
		return 0
	}
	if err := L.DoFile(fullPath); err != nil {
		slog.Debug("dofile error", "path", path, "error", err)
	}
	return 0
}

func (e *Engine) luaCall(L *lua.LState) int {
	// call(fn, arg1, arg2) - call a function loaded via dofile.
	// fn may be a function reference (most common: call(fight, ch, "x"))
	// or a string global name.
	// Source: dracula.lua, pyros.lua, breed_killer.lua — dofile+call delegation pattern.
	nArgs := L.GetTop() - 1
	arg1 := L.Get(1)
	var fn lua.LValue
	if arg1.Type() == lua.LTFunction {
		// Direct function reference — the common case after dofile redefines the global
		fn = arg1
	} else {
		fnName := L.ToString(1)
		if fnName == "" {
			return 0
		}
		fn = L.GetGlobal(fnName)
		if fn.Type() == lua.LTNil {
			slog.Debug("call: function not found", "name", fnName)
			return 0
		}
	}
	// Push function then remaining arguments
	L.Push(fn)
	for i := 2; i <= nArgs+1; i++ {
		L.Push(L.Get(i))
	}
	if err := L.PCall(nArgs, 0, nil); err != nil {
		slog.Debug("call error", "error", err)
	}
	return 0
}

func (e *Engine) luaDirection(L *lua.LState) int {
	// direction(from_vnum, to_vnum) - returns direction (0-5) from one room to
	// another.
	// Source: scripts.c lua_direction() lines 317-340.
	// find_first_step returns -1 on error, -2 if already there and -3 if no
	// path was found; invalid arguments and unknown rooms return no values.
	//
	// C gates the two failure arms (src/scripts.c:326-331 and :336-339) and
	// returns 0 from both: the invalid-room arm pushes a nil first, which Lua
	// callers cannot observe because the function declares no results, so the
	// port logs and returns 0 without touching the stack.
	if L.Get(1).Type() != lua.LTNumber || L.Get(2).Type() != lua.LTNumber {
		e.scriptMudLogBrief("[Lua] Invalid arguments passed to lua_direction.")
		return 0
	}
	fromVNum := L.ToInt(1)
	toVNum := L.ToInt(2)
	if e.world == nil || e.world.GetRoomInWorld(fromVNum) == nil || e.world.GetRoomInWorld(toVNum) == nil {
		e.scriptMudLogBrief("[Lua] Invalid room specified in lua_direction.")
		return 0
	}
	if fromVNum == toVNum {
		L.Push(lua.LNumber(-2))
		return 1
	}
	result := e.world.FindFirstStep(fromVNum, toVNum)
	L.Push(lua.LNumber(result))
	return 1
}

func (e *Engine) luaSkipSpaces(L *lua.LState) int {
	// skip_spaces(s) - trim leading spaces from a string.
	// Source: scripts.c lua_skip_spaces() lines 1385-1397.
	//
	// C's else arm logs "[Lua] Invalid argument passed to lua_skip_spaces."
	// (BRF/LVL_IMMORT/file FALSE, src/scripts.c:1393-1394) and still returns 1.
	// The port returns 1 on both arms; only the producer is missing.
	if L.Get(1).Type() != lua.LTString {
		if b := e.activeBridge; b != nil {
			b.Log("[Lua] Invalid argument passed to lua_skip_spaces.")
		}
	}
	s := L.ToString(1)
	L.Push(lua.LString(strings.TrimLeft(s, " ")))
	return 1
}

// --- Batch C Quest/Mechanic NPC stubs ---

func (e *Engine) luaStrlen(L *lua.LState) int {
	// strlen(s) - Lua 4 compat string length function.
	// Source: head_shrinker.lua make_necklace() — pads owner name to 29 chars.
	s := L.ToString(1)
	L.Push(lua.LNumber(len(s)))
	return 1
}

func (e *Engine) luaIsCorpse(L *lua.LState) int {
	// iscorpse(obj) - returns true if the object is a player or mob corpse.
	// Source: scripts.c lua_iscorpse() lines 636-654.
	// Checks OBJ_TYPE_CORPSE (ITEM_CORPSE) — type flag 18 in the original C.
	// Engine gap: object type flag comparison needs GetTypeFlag() >= 18 check.

	if tbl, ok := L.Get(1).(*lua.LTable); ok {
		typeFlagL := tbl.RawGetString("type")
		typeFlag, ok := typeFlagL.(lua.LNumber)
		if ok && int(typeFlag) == 18 { // ITEM_CORPSE
			L.Push(lua.LNumber(1))
			return 1
		}
	} else {
		// scripts.c:650-653: a non-table argument logs at BRF/LVL_IMMORT/file
		// FALSE and returns no values; only a table reaches the corpse test.
		e.scriptMudLogBrief("[Lua] Invalid argument passed to lua_iscorpse.")
		return 0
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaCanGet(L *lua.LState) int {
	// canget(obj) - returns true if the mob is permitted to pick up the object.
	// Source: scripts.c lua_canget() lines 193-219.
	// Checks CAN_GET_OBJ(ch, obj) — verifies ITEM_WEAR_TAKE flag and weight limits.
	if e.world == nil {
		L.Push(lua.LNumber(1))
		return 1
	}
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		// scripts.c:215-217: a non-table argument logs at BRF/LVL_IMMORT/file
		// FALSE and returns no values (C pushes nothing and returns 0). The
		// port returned 1 with a truthy value.
		e.scriptMudLogBrief("[Lua] Invalid argument to lua_canget.")
		return 0
	}
	// Get object vnum
	vnumVal := tbl.RawGetString("vnum")
	if vnumVal.Type() != lua.LTNumber {
		L.Push(lua.LNumber(1))
		return 1
	}
	objVNum := int(vnumVal.(lua.LNumber))
	// Get character name from the global 'ch' table (set by script context)
	chTbl := L.GetGlobal("ch")
	if chTbl == lua.LNil {
		L.Push(lua.LNumber(1))
		return 1
	}
	chTblT, ok := chTbl.(*lua.LTable)
	if !ok {
		L.Push(lua.LNumber(1))
		return 1
	}
	nameVal := chTblT.RawGetString("name")
	if nameVal.Type() != lua.LTString {
		L.Push(lua.LNumber(1))
		return 1
	}
	charName := nameVal.String()
	if e.world.CanCarryObject(charName, objVNum) {
		L.Push(lua.LNumber(1))
	} else {
		// scripts.c:211-212: CAN_GET_OBJ failing pushes nil, not a number.
		// Lua treats 0 as true, so the port's 0 made a script's
		// "if canget(obj) then" take the wrong branch.
		L.Push(lua.LNil)
	}
	return 1
}

func (e *Engine) luaItemCheck(L *lua.LState) int {
	// item_check(obj) - checks whether the shopkeeper's shop buys items of this type.
	// Source: scripts.c lua_item_check() lines 717-753.
	// Gets the shopkeeper mob from the Lua global `me`, the item from arg 1,
	// then calls World.ShopBuysType(mobVNum, objType).

	if L.GetTop() < 1 || L.Get(1).Type() != lua.LTTable {
		// scripts.c:752-755: a non-table argument logs at BRF/LVL_IMMORT/file
		// FALSE and returns no values.
		e.scriptMudLogBrief("[Lua] Invalid argument to lua_item_check.")
		return 0
	}

	// Get the item table (arg 1)
	objTbl := L.Get(1).(*lua.LTable)
	objTypeVal := objTbl.RawGetString("type")
	if objTypeVal.Type() == lua.LTNil {
		L.Push(lua.LNil)
		return 1
	}
	objType := int(objTypeVal.(lua.LNumber))

	// Get the shopkeeper mob from global `me`
	meVal := L.GetGlobal("me")
	if meVal.Type() != lua.LTTable {
		slog.Warn("[Lua] item_check: 'me' global is not a table")
		L.Push(lua.LNil)
		return 1
	}
	meTbl := meVal.(*lua.LTable)
	mobVNumVal := meTbl.RawGetString("vnum")
	if mobVNumVal.Type() == lua.LTNil {
		L.Push(lua.LNil)
		return 1
	}
	mobVNum := int(mobVNumVal.(lua.LNumber))

	// scripts.c:734-742: C scans the shop table for a keeper whose nr matches
	// this mob; no match logs "Unable to determine shop" and returns a nil,
	// which is also what a shop that does not buy the type returns. The port's
	// ShopBuysType bool collapses those two cases, so the keeper lookup is
	// asked separately through the adapter.
	if keeper, ok := e.activeBridge.(interface{ IsShopKeeper(int) bool }); ok && !keeper.IsShopKeeper(mobVNum) {
		e.scriptMudLogBrief("[Lua] Unable to determine shop in lua_item_check.")
		L.Push(lua.LNil)
		return 1
	}

	if e.world == nil {
		L.Push(lua.LNil)
		return 1
	}

	if e.world.ShopBuysType(mobVNum, objType) {
		L.Push(lua.LNumber(1)) // TRUE
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// Flag check/set functions (ported from src/scripts.c).
// luaMobFlags, luaObjExtra, luaExitFlagged, luaExitFlags, and luaExtra
// are fully implemented below.

// --- Skill group stubs (archived teacher.lua, system not ported) ---
