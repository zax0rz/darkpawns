// Package scripting provides Lua scripting support for Dark Pawns MUD.
// Based on original C code from scripts.c.
package scripting

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	lua "github.com/yuin/gopher-lua"
)

// Engine manages the Lua VM.
// Based on boot_lua() in scripts.c lines 1703-1716.
type Engine struct {
	scriptsDir   string
	l            *lua.LState
	mu           sync.Mutex
	world        ScriptableWorld
	transitItems map[int]*transitEntry // in-flight items moved by objfrom/objto
}

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
func (e *Engine) newSafeLState() *lua.LState {
	L := lua.NewState()

	// Open standard libraries
	L.OpenLibs()

	// Remove dangerous functions for security
	// Remove file system access (load arbitrary code)
	// These are intentionally nilled here and re-registered in registerFunctionsOn()
	// with sandboxed implementations, since each script runs in its own Lua state.
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("load", lua.LNil)
	L.SetGlobal("loadstring", lua.LNil)

	// Remove OS access — filesystem, process control, environment
	if osTable := L.GetGlobal("os"); osTable.Type() == lua.LTTable {
		tb := osTable.(*lua.LTable)
		tb.RawSetString("clock", lua.LNil)   // DoS: timing-detection busy loop
		tb.RawSetString("execute", lua.LNil)  // arbitrary command execution
		tb.RawSetString("exit", lua.LNil)     // crash the server
		tb.RawSetString("getenv", lua.LNil)   // information disclosure
		tb.RawSetString("remove", lua.LNil)   // file deletion
		tb.RawSetString("rename", lua.LNil)   // file manipulation
		tb.RawSetString("setenv", lua.LNil)   // affect other processes
		tb.RawSetString("setlocale", lua.LNil)
		tb.RawSetString("tmpname", lua.LNil)  // temp file creation
	}

	// Remove string.dump — produces bytecode that can exploit VM bugs
	if stringTable := L.GetGlobal("string"); stringTable.Type() == lua.LTTable {
		if tb, ok := stringTable.(*lua.LTable); ok {
			tb.RawSetString("dump", lua.LNil)
		}
	}

	// Remove math.randomseed — with a known seed a script can predict or
	// break randomness for all subsequent scripts sharing the LState.
	if mathTable := L.GetGlobal("math"); mathTable.Type() == lua.LTTable {
		if tb, ok := mathTable.(*lua.LTable); ok {
			tb.RawSetString("randomseed", lua.LNil)
		}
	}

	// Remove package library (can load arbitrary code)
	L.SetGlobal("package", lua.LNil)

	// Remove debug library
	L.SetGlobal("debug", lua.LNil)

	// Remove io library
	L.SetGlobal("io", lua.LNil)

	// Register our custom functions on the fresh state
	e.registerFunctionsOn(L)

	// Load globals.lua
	e.loadGlobalsOn(L)

	return L
}

// matchKeyword checks if a search string matches any keyword in a space-separated keyword list.
// Mirrors C's isname_with_abbrevs() behavior: case-insensitive prefix match.
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
		scriptsDir:   scriptsDir,
		transitItems: make(map[int]*transitEntry),
		world:        world,
	}

	// Create a properly sandboxed LState
	engine.l = engine.newSafeLState()

	// Start transitItems cleanup goroutine — items orphaned for >5s are logged and removed.
	go engine.cleanTransitItems()

	return engine
}

const transitItemTTL = 30 * time.Second

type transitEntry struct {
	obj       ScriptableObject
	placedAt  time.Time
}

// cleanTransitItems periodically removes orphaned items from the transit map.
func (e *Engine) cleanTransitItems() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		e.mu.Lock()
		for vnum, entry := range e.transitItems {
			if time.Since(entry.placedAt) > transitItemTTL {
				slog.Warn("transitItem orphaned, discarding", "vnum", vnum)
				delete(e.transitItems, vnum)
			}
		}
		e.mu.Unlock()
	}
}

// RunScript loads and executes a named trigger function in a script file.
// fname is relative to scriptsDir (e.g. "mob/144/hisc.lua").
// triggerName is the function to call (e.g. "oncmd", "sound", "fight").
// Returns true if the script handled the event (returned TRUE), false otherwise.
// Based on run_script() in scripts.c lines 1718-1810.
func (e *Engine) RunScript(ctx *ScriptContext, fname string, triggerName string) (bool, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Recover from Lua panics (instruction limit, context timeout, Go triggers, etc.)
	// and recreate the LState so a single poisoned script doesn't corrupt the engine.
	var needsRecreate bool
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("lua script panic, recreating LState", "reason", r, "file", fname, "trigger", triggerName)
			needsRecreate = true
		}
		if needsRecreate {
			slog.Info("recreating Lua state after script crash", "file", fname)
			e.l.Close()
			e.l = e.newSafeLState()
		}
	}()

	L := e.l

	// Set globals based on context
	// Based on run_script() lines 1732-1761
	if ctx.Ch != nil {
		e.charToTableLocked(ctx.Ch, "ch")
		slog.Debug("set ch global", "player", ctx.Ch.GetName())
	} else {
		slog.Debug("ctx.Ch is nil")
	}
	if ctx.Me != nil {
		e.mobToTableLocked(ctx.Me, "me")
	}
	if ctx.Obj != nil {
		e.objToTableLocked(ctx.Obj, "obj")
	}
	if ctx.Argument != "" {
		L.SetGlobal("argument", lua.LString(ctx.Argument))
	}

	// Set room global if we have room vnum
	if ctx.RoomVNum > 0 {
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

	// Load and execute the script file
	// Based on open_lua_file() in scripts.c lines 1641-1701
	// Execution timeout prevents tight loops from hanging the server indefinitely.
	scriptPath := e.scriptsDir + "/" + fname

	scriptCtx, scriptCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer scriptCancel()
	L.SetContext(scriptCtx)

	if err := L.DoFile(scriptPath); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("script timed out during load", "file", fname, "error", err)
			needsRecreate = true
		} else {
			slog.Error("error loading script", "file", fname, "error", err)
		}
		L.RemoveContext()
		scriptCancel()
		return false, err
	}

	// Call the trigger function
	// Based on run_script() lines 1780-1795
	fn := L.GetGlobal(triggerName)
	slog.Debug("calling function", "trigger", triggerName, "type", fn.Type())
	L.Push(fn)
	if fn.Type() == lua.LTNil {
		// Function doesn't exist
		L.Pop(1)
		L.RemoveContext()
		scriptCancel()
		slog.Debug("function not found in script", "trigger", triggerName, "file", fname)
		return false, nil
	}

	if err := L.PCall(0, 1, nil); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			slog.Error("script timed out during execution", "trigger", triggerName, "file", fname, "error", err)
			needsRecreate = true
		} else {
			slog.Error("error calling function", "trigger", triggerName, "file", fname, "error", err)
		}
		if L.GetTop() > 0 {
			L.Pop(1)
		}
		L.RemoveContext()
		scriptCancel()
		return false, err
	}

	// Get return value
	stackTop := L.GetTop()
	slog.Debug("stack top after PCall", "top", stackTop)

	var ret lua.LValue = lua.LFalse
	if stackTop > 0 {
		ret = L.Get(-1)
		slog.Debug("function returned", "type", ret.Type(), "value", ret)
		L.Pop(1)
	}

	// Read back changes from tables
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

	L.RemoveContext()
	scriptCancel()

	// Check return value
	if ret.Type() == lua.LTNumber {
		return lua.LVAsNumber(ret) == 1, nil
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
	L.SetGlobal("act", L.NewFunction(e.luaAct))
	L.SetGlobal("do_damage", L.NewFunction(e.luaDoDamage))
	L.SetGlobal("say", L.NewFunction(e.luaSay))
	L.SetGlobal("gossip", L.NewFunction(e.luaGossip))
	L.SetGlobal("emote", L.NewFunction(e.luaEmote))
	L.SetGlobal("action", L.NewFunction(e.luaAction))
	L.SetGlobal("oload", L.NewFunction(e.luaOload))
	L.SetGlobal("mload", L.NewFunction(e.luaMload))
	L.SetGlobal("extobj", L.NewFunction(e.luaExtobj))
	L.SetGlobal("extchar", L.NewFunction(e.luaExtchar))
	L.SetGlobal("number", L.NewFunction(e.luaNumber))
	L.SetGlobal("send_to_room", L.NewFunction(e.luaSendToRoom))
	L.SetGlobal("strlower", L.NewFunction(e.luaStrlower))
	L.SetGlobal("strfind", L.NewFunction(e.luaStrfind))
	L.SetGlobal("strsub", L.NewFunction(e.luaStrsub))
	L.SetGlobal("gsub", L.NewFunction(e.luaGsub))
	L.SetGlobal("getn", L.NewFunction(e.luaGetn))
	L.SetGlobal("tonumber", L.NewFunction(e.luaTonumber))
	// Don't override tostring - it's a Lua built-in
	// L.SetGlobal("tostring", L.NewFunction(e.luaTostring))

	// Additional functions from cmdlib that might be needed
	L.SetGlobal("log", L.NewFunction(e.luaLog))
	L.SetGlobal("raw_kill", L.NewFunction(e.luaRawKill))
	L.SetGlobal("save_char", L.NewFunction(e.luaSaveChar))
	L.SetGlobal("save_obj", L.NewFunction(e.luaSaveObj))
	// dofile/call: shared-script delegation pattern used by cityguard, breed_killer, etc.
	// NOTE: dofile is re-registered below after being nilled in newSafeLState().
	// This is intentional — each script file is loaded into its own sandboxed Lua state,
	// so re-registration is expected behavior.
	L.SetGlobal("dofile", L.NewFunction(e.luaDofile))
	L.SetGlobal("call", L.NewFunction(e.luaCall))
	L.SetGlobal("save_room", L.NewFunction(e.luaSaveRoom))
	L.SetGlobal("set_skill", L.NewFunction(e.luaSetSkill))
	L.SetGlobal("spell", L.NewFunction(e.luaSpell))
	L.SetGlobal("tport", L.NewFunction(e.luaTport))

	// Additional functions needed for combat AI scripts
	L.SetGlobal("isfighting", L.NewFunction(e.luaIsFighting))
	L.SetGlobal("round", L.NewFunction(e.luaRound))

	// Functions needed for RESTORE scripts
	L.SetGlobal("has_item", L.NewFunction(e.luaHasItem))
	L.SetGlobal("obj_in_room", L.NewFunction(e.luaObjInRoom))
	L.SetGlobal("objfrom", L.NewFunction(e.luaObjFrom))
	L.SetGlobal("objto", L.NewFunction(e.luaObjTo))
	L.SetGlobal("obj_extra", L.NewFunction(e.luaObjExtra))
	L.SetGlobal("create_event", L.NewFunction(e.luaCreateEvent))
	L.SetGlobal("tell", L.NewFunction(e.luaTell))
	L.SetGlobal("plr_flagged", L.NewFunction(e.luaPlrFlagged))
	L.SetGlobal("cansee", L.NewFunction(e.luaCanSee))
	L.SetGlobal("isnpc", L.NewFunction(e.luaIsNPC))
	L.SetGlobal("aff_flagged", L.NewFunction(e.luaAffFlagged))
	L.SetGlobal("plr_flags", L.NewFunction(e.luaPlrFlags))
	L.SetGlobal("obj_list", L.NewFunction(e.luaObjList))

	// Stubs needed by Tier 3 Economy scripts
	L.SetGlobal("item_check", L.NewFunction(e.luaItemCheck))
	L.SetGlobal("load_room", L.NewFunction(e.luaLoadRoom))
	L.SetGlobal("inworld", L.NewFunction(e.luaInworld))
	L.SetGlobal("mob_flagged", L.NewFunction(e.luaMobFlagged))
	L.SetGlobal("aff_flags", L.NewFunction(e.luaAffFlags))
	L.SetGlobal("follow", L.NewFunction(e.luaFollow))
	L.SetGlobal("mount", L.NewFunction(e.luaMount))
	L.SetGlobal("direction", L.NewFunction(e.luaDirection))
	L.SetGlobal("set_hunt", L.NewFunction(e.luaSetHunt))
	L.SetGlobal("ishunt", L.NewFunction(e.luaIshunt))
	L.SetGlobal("mxp", L.NewFunction(e.luaMxp))
	L.SetGlobal("skip_spaces", L.NewFunction(e.luaSkipSpaces))
	L.SetGlobal("social", L.NewFunction(e.luaSocial))
	L.SetGlobal("obj_flagged", L.NewFunction(e.luaObjFlagged))
	L.SetGlobal("mob_flags", L.NewFunction(e.luaMobFlags))
	L.SetGlobal("exit_flagged", L.NewFunction(e.luaExitFlagged))
	L.SetGlobal("exit_flags", L.NewFunction(e.luaExitFlags))
	L.SetGlobal("get_group_lvl", L.NewFunction(e.luaGetGroupLvl))
	L.SetGlobal("get_group_pts", L.NewFunction(e.luaGetGroupPts))
	L.SetGlobal("skill_group", L.NewFunction(e.luaSkillGroup))
	L.SetGlobal("unaffect", L.NewFunction(e.luaUnaffect))
	L.SetGlobal("equip_char", L.NewFunction(e.luaEquipChar))
	// echo(ch, type, msg) — zone-wide sound broadcast. Used by werewolf.lua.
	L.SetGlobal("echo", L.NewFunction(e.luaEcho))

	// Stubs needed by Batch C Quest/Mechanic NPC scripts
	L.SetGlobal("extra", L.NewFunction(e.luaExtra))
	L.SetGlobal("strlen", L.NewFunction(e.luaStrlen))
	L.SetGlobal("iscorpse", L.NewFunction(e.luaIsCorpse))
	L.SetGlobal("canget", L.NewFunction(e.luaCanGet))
	L.SetGlobal("steal", L.NewFunction(e.luaSteal))
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
	// Always set up basic constants
	e.setupBasicConstantsOn(L)
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
	L.SetGlobal("LVL_IMMORT", lua.LNumber(31))
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
	// SPELL_PARALYSE: not in original globals.lua; assigned 105 as next available value.
	// SPELL_PARALYSE: Dark Pawns custom spell, not in original C spells.h.
	// Used by paralyse.lua and head_shrinker.lua. Verified: no C source equivalent.
	L.SetGlobal("SPELL_PARALYSE", lua.LNumber(105))

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
	L.SetGlobal("SKILL_KICK", lua.LNumber(134))
	L.SetGlobal("SKILL_TRIP", lua.LNumber(144))

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

	L.SetGlobal(globalName, tbl)
}

// objToTableLocked converts a ScriptableObject to a Lua table. Caller must hold e.mu.
// Based on obj_to_table() in scripts.c lines 1918-2016.
func (e *Engine) objToTableLocked(obj ScriptableObject, globalName string) {
	L := e.l
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("vnum", lua.LNumber(obj.GetVNum()))
	tbl.RawSetString("alias", lua.LString(obj.GetKeywords()))
	tbl.RawSetString("name", lua.LString(obj.GetShortDesc()))
	tbl.RawSetString("cost", lua.LNumber(obj.GetCost()))
	tbl.RawSetString("timer", lua.LNumber(obj.GetTimer()))

	// Object prototype fields (stubbed)
	tbl.RawSetString("perc_load", lua.LNumber(0)) // Default 0% load chance

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(obj.GetVNum()))

	L.SetGlobal(globalName, tbl)
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

func (e *Engine) luaAct(L *lua.LState) int {
	// act(msg, visible, ch, obj, vict, type)
	// Based on lua_act() in scripts.c lines 79-124
	msg := L.ToString(1)
	_ = L.ToBool(2) // visible - unused for now
	where := L.ToInt(6)

	// Get room vnum from context
	var roomVNum int
	roomVal := L.GetGlobal("room")
	if roomVal.Type() == lua.LTNumber {
		roomVNum = int(roomVal.(lua.LNumber))
	}

	// Get me from global for room context
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "room")
		if L.Get(-1).Type() == lua.LTNumber {
			roomVNum = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}
	L.Pop(1)

	// Get ch from global for TO_VICT/TO_CHAR
	L.GetGlobal("ch")
	var ch ScriptablePlayer = nil
	// In real implementation, we'd get the actual player pointer from the Lua table.
	// For now, we'll use the world to find players.
	_ = ch // placeholder until Lua player lookup is implemented
	L.Pop(1)

	if e.world == nil || roomVNum == 0 {
		slog.Debug("act: no world or room context", "msg", msg, "where", where)
		return 0
	}

	players := e.world.GetPlayersInRoom(roomVNum)

	switch where {
	case 1: // TO_ROOM
		for _, player := range players {
			player.SendMessage(msg + "\r\n")
		}
	case 2: // TO_VICT
		if ch != nil {
			ch.SendMessage(msg + "\r\n")
		}
	case 3: // TO_NOTVICT
		for _, player := range players {
			if ch == nil || player.GetID() != ch.GetID() {
				player.SendMessage(msg + "\r\n")
			}
		}
	case 4: // TO_CHAR
		if ch != nil {
			ch.SendMessage(msg + "\r\n")
		}
	}

	return 0
}

func (e *Engine) luaDoDamage(L *lua.LState) int {
	// do_damage(amount)
	// Based on pattern_dmg.lua example
	amount := L.ToInt(1)

	// Get ch from global
	L.GetGlobal("ch")
	if L.Get(-1).Type() == lua.LTTable {
		// Apply damage to ch
		L.GetField(L.Get(-1), "hp")
		if L.Get(-1).Type() == lua.LTNumber {
			currentHP := int(L.ToNumber(-1))
			newHP := currentHP - amount
			if newHP < 0 {
				newHP = 0
			}
			L.Pop(1) // pop hp value

			// Update hp in table
			L.Push(lua.LNumber(newHP))
			L.SetField(L.Get(-2), "hp", lua.LNumber(newHP))

			// Check for death
			if newHP <= 0 && e.world != nil {
				// Get the actual player object from the world
				// For now, just log
				slog.Debug("do_damage: player would die", "damage", amount)
			}
		} else {
			L.Pop(1)
		}
	}
	L.Pop(1)

	return 0
}

func (e *Engine) luaSay(L *lua.LState) int {
	// say(msg)
	// Based on lua_say() (not shown in snippets but referenced)
	msg := L.ToString(1)

	// Get me from global for room context
	var roomVNum int
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "room")
		if L.Get(-1).Type() == lua.LTNumber {
			roomVNum = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}
	L.Pop(1)

	if e.world == nil || roomVNum == 0 {
		slog.Debug("say: no world or room context", "msg", msg)
		return 0
	}

	// Format message: "mob says 'message'"
	L.GetGlobal("me")
	var mobName = "someone"
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "name")
		if L.Get(-1).Type() == lua.LTString {
			mobName = L.ToString(-1)
		}
		L.Pop(1)
	}
	L.Pop(1)

	formattedMsg := mobName + " says '" + msg + "'\r\n"
	players := e.world.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.SendMessage(formattedMsg)
	}

	return 0
}

// luaGossip broadcasts a mob's message to the room (gossip channel stub).
// gossip(msg) — in the original, this goes to all players world-wide.
// Since we have no global channel yet, we send to the mob's current room.
// Based on gossip channel in comm.c; used by quanlo.lua.
func (e *Engine) luaGossip(L *lua.LState) int {
	msg := L.ToString(1)

	if e.world == nil {
		slog.Debug("gossip", "msg", msg)
		return 0
	}

	formatted := "[gossip] " + msg + "\r\n"
	e.world.SendToAll(formatted)
	return 0
}

func (e *Engine) luaEmote(L *lua.LState) int {
	// emote(msg)
	// Based on lua_emote() in scripts.c lines 291-306
	msg := L.ToString(1)

	// Get me from global for room context
	var roomVNum int
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "room")
		if L.Get(-1).Type() == lua.LTNumber {
			roomVNum = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}
	L.Pop(1)

	if e.world == nil || roomVNum == 0 {
		slog.Debug("emote: no world or room context", "msg", msg)
		return 0
	}

	// Format message: "mob message"
	L.GetGlobal("me")
	var mobName = "someone"
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "name")
		if L.Get(-1).Type() == lua.LTString {
			mobName = L.ToString(-1)
		}
		L.Pop(1)
	}
	L.Pop(1)

	formattedMsg := mobName + " " + msg + "\r\n"
	players := e.world.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.SendMessage(formattedMsg)
	}

	return 0
}

func (e *Engine) luaAction(L *lua.LState) int {
	// action(mob, cmdstr)
	// Based on lua_action() in scripts.c lines 126-144
	// mob would be table, cmdstr is string
	// In real implementation: mob executes command

	// Get parameters
	mobTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		slog.Warn("action: first arg is not a table")
		return 0
	}
	cmdStr := L.ToString(2)

	if e.world == nil {
		slog.Debug("action: no world context", "command", cmdStr)
		return 0
	}

	// Get mob VNum from the table
	vnumL := mobTbl.RawGetString("vnum")
	vnum, ok := vnumL.(lua.LNumber)
	if !ok || vnum <= 0 {
		slog.Debug("action: no vnum in mob table")
		return 0
	}

	e.world.ExecuteMobCommand(int(vnum), cmdStr)
	return 0
}

func (e *Engine) luaOload(L *lua.LState) int {
	// oload(target, vnum, location)
	// Based on lua_oload() in scripts.c lines 1047-1090
	// target is ch table, vnum is number, location is string
	targetTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		slog.Warn("oload: first arg is not a table")
		L.Push(lua.LNil)
		return 1
	}
	vnum := L.ToInt(2)
	location := L.ToString(3)

	if e.world == nil {
		slog.Debug("oload: no world context", "vnum", vnum, "location", location)
		L.Push(lua.LNil)
		return 1
	}

	// Get object prototype
	objProto := e.world.GetObjPrototype(vnum)
	if objProto == nil {
		slog.Debug("oload: object prototype not found", "vnum", vnum)
		L.Push(lua.LNil)
		return 1
	}

	// Create object instance from prototype
	tbl := L.NewTable()
	tbl.RawSetString("vnum", lua.LNumber(objProto.GetVNum()))
	tbl.RawSetString("alias", lua.LString(objProto.GetKeywords()))
	tbl.RawSetString("name", lua.LString(objProto.GetShortDesc()))
	tbl.RawSetString("cost", lua.LNumber(objProto.GetCost()))
	tbl.RawSetString("timer", lua.LNumber(objProto.GetTimer()))

	// Based on C lua_oload(): if location is "room", add object to room via ch->in_room.
	// If location is "char", add to the character's inventory.
	switch location {
	case "room":
		roomVNumL := targetTbl.RawGetString("room")
		roomVNum, ok := roomVNumL.(lua.LNumber)
		if !ok || roomVNum <= 0 {
			slog.Debug("oload: no room in target table")
		} else {
			if err := e.world.AddItemToRoom(objProto, int(roomVNum)); err != nil {
				slog.Warn("oload: failed to add item to room", "vnum", vnum, "room", int(roomVNum), "error", err)
			} else {
				slog.Debug("oload: added item to room", "vnum", vnum, "room", int(roomVNum))
			}
		}
	case "char":
		charNameL := targetTbl.RawGetString("name")
		charName, ok := charNameL.(lua.LString)
		if !ok || charName == "" {
			slog.Debug("oload: no name in target table")
		} else {
			if err := e.world.GiveItemToChar(string(charName), objProto); err != nil {
				slog.Warn("oload: failed to give item to char", "vnum", vnum, "char", string(charName), "error", err)
			} else {
				slog.Debug("oload: gave item to char", "vnum", vnum, "char", string(charName))
			}
		}
	default:
		slog.Warn("oload: unknown location", "location", location)
	}

	L.Push(tbl)
	return 1
}

func (e *Engine) luaMload(L *lua.LState) int {
	// mload(room_vnum, mob_vnum)
	// Based on lua_mload() in scripts.c lines 661-688
	return 0
}

func (e *Engine) luaExtobj(L *lua.LState) int {
	// extobj(obj)
	// Based on lua_extobj() in scripts.c lines 443-456
	return 0
}

func (e *Engine) luaExtchar(L *lua.LState) int {
	// extchar(ch)
	// Based on lua_extchar() in scripts.c lines 430-441
	return 0
}

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
	result := low + rand.Intn(high-low+1)
	L.Push(lua.LNumber(result))
	return 1
}

func (e *Engine) luaSendToRoom(L *lua.LState) int {
	// send_to_room(msg, room_vnum)
	// Based on lua_echo() with type="room" in scripts.c lines 308-345
	msg := L.ToString(1)
	roomVNum := L.ToInt(2)

	if e.world == nil {
		slog.Debug("send_to_room: no world context", "room_vnum", roomVNum, "msg", msg)
		return 0
	}

	players := e.world.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.SendMessage(msg + "\r\n")
	}

	return 0
}

func (e *Engine) luaStrlower(L *lua.LState) int {
	// strlower(s)
	// Lua 4 compat function
	s := L.ToString(1)
	L.Push(lua.LString(strings.ToLower(s)))
	return 1
}

func (e *Engine) luaStrfind(L *lua.LState) int {
	// strfind(s, pattern)
	// Already in Lua 5.1 as string.find, expose as global
	// Just call the built-in
	L.GetGlobal("string")
	L.GetField(L.Get(-1), "find")
	L.Push(L.Get(1))
	L.Push(L.Get(2))
	L.Call(2, 1)
	return 1
}

func (e *Engine) luaStrsub(L *lua.LState) int {
	// strsub(s, i, j)
	// Already in Lua 5.1 as string.sub, expose as global
	L.GetGlobal("string")
	L.GetField(L.Get(-1), "sub")
	L.Push(L.Get(1))
	L.Push(L.Get(2))
	L.Push(L.Get(3))
	L.Call(3, 1)
	return 1
}

func (e *Engine) luaGsub(L *lua.LState) int {
	// gsub(s, pattern, repl)
	// Already in Lua 5.1 as string.gsub, expose as global
	L.GetGlobal("string")
	L.GetField(L.Get(-1), "gsub")
	L.Push(L.Get(1))
	L.Push(L.Get(2))
	L.Push(L.Get(3))
	L.Call(3, 1)
	return 1
}

func (e *Engine) luaGetn(L *lua.LState) int {
	// getn(t)
	// Lua 4 compat for table length
	tbl := L.Get(1)
	if tbl.Type() != lua.LTTable {
		L.Push(lua.LNumber(0))
		return 1
	}

	L.Push(lua.LNumber(L.ObjLen(tbl)))
	return 1
}

func (e *Engine) luaTonumber(L *lua.LState) int {
	// tonumber(s)
	// Already in Lua 5.1, expose as global
	L.GetGlobal("tonumber")
	L.Push(L.Get(1))
	L.Call(1, 1)
	return 1
}

func (e *Engine) luaLog(L *lua.LState) int {
	// log(txt)
	// Based on lua_log() in scripts.c lines 690-703
	txt := L.ToString(1)
	slog.Debug("lua log", "text", txt)
	return 0
}

func (e *Engine) luaRawKill(L *lua.LState) int {
	// raw_kill(vict, killer, type)
	// Based on lua_raw_kill() (not shown in snippets)
	return 0
}

func (e *Engine) luaSaveChar(L *lua.LState) int {
	// save_char() - calls table_to_char
	// Based on lua_save_char() (not shown in snippets)
	return 0
}

func (e *Engine) luaSaveObj(L *lua.LState) int {
	// save_obj() - calls table_to_obj
	// Based on lua_save_obj() (not shown in snippets)
	return 0
}

func (e *Engine) luaSaveRoom(L *lua.LState) int {
	// save_room() - calls table_to_room
	// Based on lua_save_room() (not shown in snippets)
	return 0
}

func (e *Engine) luaSetSkill(L *lua.LState) int {
	// set_skill(ch, skill, value)
	// Based on lua_set_skill() (not shown in snippets)
	return 0
}

func (e *Engine) luaSpell(L *lua.LState) int {
	// spell(caster, target, spellnum, aggressive)
	// Based on lua_spell() function called from Lua scripts
	// caster: table representing mob/player casting spell
	// target: table representing target (can be NIL)
	// spellnum: spell number constant
	// aggressive: boolean indicating if spell is offensive

	// Get parameters
	casterTbl := L.Get(1)
	targetTbl := L.Get(2)
	spellNum := L.ToInt(3)
	aggressive := L.ToBool(4)

	// Get caster level
	casterLevel := 1
	if casterTbl.Type() == lua.LTTable {
		L.GetField(casterTbl, "level")
		if L.Get(-1).Type() == lua.LTNumber {
			casterLevel = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}

	// Helper function for dice rolls
	dice := func(num, sides int) int {
		total := 0
		for i := 0; i < num; i++ {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			total += rand.Intn(sides) + 1
		}
		return total
	}

	// Handle different spell types
	switch spellNum {
	case 2: // SPELL_TELEPORT
		// Move target to a random room
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Get a random room from world (for now, just room 8004 - death room)
			newRoom := 8004
			L.SetField(targetTbl, "room", lua.LNumber(newRoom))
			// Send message to target if it's a player
			L.GetField(targetTbl, "name")
			targetName := L.ToString(-1)
			L.Pop(1)
			slog.Debug("spell teleport", "target", targetName, "room", newRoom)
		}
		return 0

	case 16: // SPELL_CURE_LIGHT
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Healing spell: dice(2,8) + 1 + (level >> 2) - magic.c mag_points() line ~1765
			healAmount := dice(2, 8) + 1 + (casterLevel >> 2)
			L.GetField(targetTbl, "hp")
			currentHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxhp")
			maxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newHP := currentHP + healAmount
			if newHP > maxHP {
				newHP = maxHP
			}
			L.SetField(targetTbl, "hp", lua.LNumber(newHP))
			slog.Debug("spell cure light", "heal_amount", healAmount, "new_hp", newHP)
		}
		return 0

	case 28: // SPELL_HEAL
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Healing spell: 100 + dice(3,8) - magic.c mag_points() line ~1783
			healAmount := 100 + dice(3, 8)
			L.GetField(targetTbl, "hp")
			currentHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxhp")
			maxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newHP := currentHP + healAmount
			if newHP > maxHP {
				newHP = maxHP
			}
			L.SetField(targetTbl, "hp", lua.LNumber(newHP))
			slog.Debug("spell heal", "heal_amount", healAmount, "new_hp", newHP)
		}
		return 0

	case 67: // SPELL_VITALITY
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Healing spell: dice(5,10) HP + dice(10,10) move - magic.c mag_points() line ~1800
			healAmount := dice(5, 10)
			L.GetField(targetTbl, "hp")
			currentHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxhp")
			maxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newHP := currentHP + healAmount
			if newHP > maxHP {
				newHP = maxHP
			}
			L.SetField(targetTbl, "hp", lua.LNumber(newHP))
			// Restore move points (always 10-100 move restored by vitality)
			moveAmount := dice(10, 10)
			L.GetField(targetTbl, "move")
			currentMove := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxmove")
			maxMove := int(L.ToNumber(-1))
			L.Pop(1)
			newMove := currentMove + moveAmount
			if newMove > maxMove && maxMove > 0 {
				newMove = maxMove
			}
			L.SetField(targetTbl, "move", lua.LNumber(newMove))
			slog.Debug("spell vitality", "heal_amount", healAmount, "new_hp", newHP, "move_restore", moveAmount, "new_move", newMove)
		}
		return 0
	}

	// For offensive spells (aggressive=true)
	if aggressive && targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
		// Calculate damage based on spell type and caster level
		// Formulas from Dark Pawns magic.c
		damage := 0
		switch spellNum {
		case 32: // SPELL_MAGIC_MISSILE
			damage = dice(4, 3) + casterLevel
		case 5: // SPELL_BURNING_HANDS
			damage = dice(4, 5) + casterLevel
		case 37: // SPELL_SHOCKING_GRASP
			damage = dice(4, 7) + casterLevel
		case 30: // SPELL_LIGHTNING_BOLT
			damage = dice(9, 4) + casterLevel
		case 10: // SPELL_COLOR_SPRAY
			damage = dice(9, 7) + casterLevel
		case 26: // SPELL_FIREBALL
			// magic.c mag_damage() line ~690 - non-mage formula (we don't track reagents)
			damage = dice(12, 8) + casterLevel*2
		case 58: // SPELL_HELLFIRE
			// SPELL_HELLFIRE: disabled in original source (magic.c spell_hellfire - dummy) line ~1583
			slog.Debug("spell hellfire: disabled in original source")
			return 0
		case 92: // SPELL_DISRUPT
			// magic.c mag_damage() line ~720 - non-mage formula
			damage = dice(20, 7) + casterLevel
		case 93: // SPELL_DISINTEGRATE
			// magic.c mag_damage() line ~700 - non-mage formula
			damage = dice(18, 8) + casterLevel
		case 96: // SPELL_FLAMESTRIKE
			// SPELL_FLAMESTRIKE: NOT a direct damage spell in mag_damage() - it's an outdoor AFF_FLAMING affect
			slog.Debug("spell flamestrike: outdoor affect spell, not direct damage")
			return 0
		case 75: // SPELL_ACID_BLAST
			// magic.c mag_damage() line ~790
			damage = dice(4, 3) + casterLevel
		case 22: // SPELL_DISPEL_EVIL
			// magic.c mag_damage() line ~730
			damage = dice(9, 5) + casterLevel + 5 + casterLevel/2
		case 46: // SPELL_DISPEL_GOOD
			// magic.c mag_damage() line ~740
			damage = dice(9, 5) + casterLevel + 5
		case 27: // SPELL_HARM
			// magic.c mag_damage() line ~750
			damage = dice(12, 8) + casterLevel*2
		case 4: // SPELL_BLINDNESS
			// SPELL_BLINDNESS: affect only, not damage
			slog.Debug("spell blindness: affect only, not damage")
			return 0
		case 17: // SPELL_CURSE
			// SPELL_CURSE: affect only, not damage
			slog.Debug("spell curse: affect only, not damage")
			return 0
		case 33: // SPELL_POISON
			// SPELL_POISON: affect only, not damage
			slog.Debug("spell poison: affect only, not damage")
			return 0
		case 23: // SPELL_EARTHQUAKE
			// magic.c mag_damage() line ~785
			damage = dice(7, 7) + casterLevel
		case 81: // SPELL_DIVINE_INT
			// SPELL_DIVINE_INT: NOT a damage spell — it summons an angel
			slog.Debug("spell divine int: summon spell, not damage")
			return 0
		case 82: // SPELL_MIND_BAR
			// SPELL_MIND_BAR: NOT a damage spell — it's an INT debuff affect
			slog.Debug("spell mind bar: INT debuff affect, not damage")
			return 0
		case 41: // SPELL_METEOR_SWARM
			// SPELL_METEOR_SWARM: Area spell that calls damage() per person in room — not single-target damage
			slog.Debug("spell meteor swarm: area spell handled separately")
			return 0
		case 100: // SPELL_PSIBLAST
			// magic.c mag_damage() line ~805
			damage = dice(15, 13) + 3*casterLevel
		case 104: // SPELL_PETRIFY
			// SPELL_PETRIFY: raw_kill mechanic, not mag_damage
			slog.Debug("spell petrify: raw_kill mechanic, not mag_damage")
			return 0
		case 8: // SPELL_CHILL_TOUCH
			// magic.c mag_damage() line ~636
			damage = dice(5, 3) + casterLevel
		case 6: // SPELL_CALL_LIGHTNING
			// magic.c mag_damage() line ~748
			damage = dice(10, 8) + casterLevel + 5
		case 25: // SPELL_ENERGY_DRAIN
		case 83: // SPELL_SOUL_LEECH
			// magic.c mag_damage() line ~778
			damage = dice(10, 6) + casterLevel
			// Soul leech heals caster by dam/3 — handled inline above
		case 62: // SPELL_MINDBLAST
			// magic.c mag_damage() line ~800
			damage = dice(9, 7) + casterLevel + casterLevel/2
		default:
			// Default formula for unknown offensive spells
			minDamage := casterLevel
			maxDamage := casterLevel * 3
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			damage = casterLevel*2 + rand.Intn(maxDamage-minDamage+1) + minDamage
		}

		// Get current HP
		L.GetField(targetTbl, "hp")
		currentHP := int(L.ToNumber(-1))
		L.Pop(1)

		// Apply damage
		newHP := currentHP - damage
		if newHP < 0 {
			newHP = 0
		}
		L.SetField(targetTbl, "hp", lua.LNumber(newHP))

		// Soul leech healing: caster heals by dam/3
		// Based on magic.c mag_damage() — soul leech applies vampiric healing
		if spellNum == 83 && casterTbl.Type() == lua.LTTable && casterTbl != lua.LNil {
			soulLeechHeal := damage / 3
			if soulLeechHeal < 1 {
				soulLeechHeal = 1
			}
			L.GetField(casterTbl, "hp")
			casterHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(casterTbl, "maxhp")
			casterMaxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newCasterHP := casterHP + soulLeechHeal
			if newCasterHP > casterMaxHP && casterMaxHP > 0 {
				newCasterHP = casterMaxHP
			}
			L.SetField(casterTbl, "hp", lua.LNumber(newCasterHP))
			slog.Debug("soul leech heal", "caster_hp_before", casterHP, "heal", soulLeechHeal, "caster_hp_after", newCasterHP)
		}

		// Check for death
		if newHP == 0 && e.world != nil {
			// Get target name and room
			L.GetField(targetTbl, "name")
			targetName := L.ToString(-1)
			L.Pop(1)
			L.GetField(targetTbl, "room")
			roomVNum := int(L.ToNumber(-1))
			L.Pop(1)

			// Notify world of spell death
			e.world.HandleSpellDeath(targetName, spellNum, roomVNum)
		}

		// Log
		L.GetField(casterTbl, "name")
		casterName := L.ToString(-1)
		L.Pop(1)
		L.GetField(targetTbl, "name")
		targetName := L.ToString(-1)
		L.Pop(1)

		// Convert spell number to name for logging
		spellName := fmt.Sprintf("SPELL_%d", spellNum)
		switch spellNum {
		case 32:
			spellName = "MAGIC_MISSILE"
		case 5:
			spellName = "BURNING_HANDS"
		case 30:
			spellName = "LIGHTNING_BOLT"
		case 26:
			spellName = "FIREBALL"
		case 58:
			spellName = "HELLFIRE"
		case 10:
			spellName = "COLOR_SPRAY"
		case 92:
			spellName = "DISRUPT"
		case 93:
			spellName = "DISINTEGRATE"
		case 96:
			spellName = "FLAMESTRIKE"
		case 75:
			spellName = "ACID_BLAST"
		case 22:
			spellName = "DISPEL_EVIL"
		case 46:
			spellName = "DISPEL_GOOD"
		case 27:
			spellName = "HARM"
		case 4:
			spellName = "BLINDNESS"
		case 17:
			spellName = "CURSE"
		case 33:
			spellName = "POISON"
		case 23:
			spellName = "EARTHQUAKE"
		case 81:
			spellName = "DIVINE_INT"
		case 82:
			spellName = "MIND_BAR"
		case 8:
			spellName = "CHILL_TOUCH"
		case 6:
			spellName = "CALL_LIGHTNING"
		case 25:
			spellName = "ENERGY_DRAIN"
		case 83:
			spellName = "SOUL_LEECH"
		case 62:
			spellName = "MINDBLAST"
		case 100:
			spellName = "PSIBLAST"
		case 37:
			spellName = "SHOCKING_GRASP"
		}

		slog.Debug("spell cast",
			"caster", casterName,
			"spell", spellName,
			"target", targetName,
			"damage", damage,
			"hp_before", currentHP,
			"hp_after", newHP,
		)

	} else {
		// Non-aggressive or no target
		L.GetField(casterTbl, "name")
		casterName := L.ToString(-1)
		L.Pop(1)

		spellName := fmt.Sprintf("SPELL_%d", spellNum)
		slog.Debug("spell cast (non-aggressive)", "caster", casterName, "spell", spellName)
	}

	return 0
}

func (e *Engine) luaTport(L *lua.LState) int {
	// tport(ch, room)
	// Based on lua_tport() (not shown in snippets)
	return 0
}

func (e *Engine) luaIsFighting(L *lua.LState) int {
	// isfighting(mob) - returns the mob's current combat target as a table, or nil
	// Based on lua_isfighting() in scripts.c — checks mob's fighting pointer
	mobTbl := L.Get(1)
	if mobTbl.Type() != lua.LTTable {
		L.Push(lua.LNil)
		return 1
	}
	// Get the mob's vnum from the table to look it up in world
	L.GetField(mobTbl, "vnum")
	vnum := int(L.ToNumber(-1))
	L.Pop(1)

	if e.world != nil && vnum > 0 {
		// Look up the mob's fighting target via ScriptableWorld
		// GetMobByVNumAndRoom is approximate — use vnum to find the mob
		L.GetField(mobTbl, "room")
		roomVNum := int(L.ToNumber(-1))
		L.Pop(1)

		mob := e.world.GetMobByVNumAndRoom(vnum, roomVNum)
		if mob != nil {
			targetName := mob.GetFighting()
			if targetName != "" {
				// Return a minimal table with .name so scripts can do fighting.name
				tgt := L.NewTable()
				tgt.RawSetString("name", lua.LString(targetName))
				tgt.RawSetString("level", lua.LNumber(1)) // Unknown — fighting target level
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
	fullPath := filepath.Clean(filepath.Join(e.scriptsDir, path))
	rel, err := filepath.Rel(e.scriptsDir, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") || strings.HasPrefix(rel, "/") {
		slog.Warn("dofile: path traversal blocked", "path", path)
		return 0
	}
	if err := e.l.DoFile(fullPath); err != nil {
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

func (e *Engine) luaRound(L *lua.LState) int {
	// round(n) - rounds a number to nearest integer
	// Lua 4 compat function used in combat AI scripts
	n := L.ToNumber(1)
	rounded := int(n + 0.5)
	L.Push(lua.LNumber(rounded))
	return 1
}

func (e *Engine) luaHasItem(L *lua.LState) int {
	// has_item(ch, vnum) - returns true if ch has an item with vnum in inventory.
	// Source: scripts.c lua_has_item() — searches char inventory for matching vnum.
	chTbl := L.Get(1)
	vnum := L.ToInt(2)

	if e.world == nil || chTbl.Type() != lua.LTTable {
		L.Push(lua.LBool(false))
		return 1
	}

	nameVal := L.GetField(chTbl, "name")
	if nameVal.Type() != lua.LTString {
		L.Push(lua.LBool(false))
		return 1
	}
	charName := string(nameVal.(lua.LString))

	L.Push(lua.LBool(e.world.HasItemByVNum(charName, vnum)))
	return 1
}

func (e *Engine) luaObjInRoom(L *lua.LState) int {
	// obj_in_room(room_vnum, obj_vnum) - returns item table if obj_vnum is in room, else nil.
	// Source: scripts.c lua_obj_in_room().
	roomVNum := L.ToInt(1)
	objVNum := L.ToInt(2)

	if e.world == nil {
		L.Push(lua.LNil)
		return 1
	}

	for _, item := range e.world.GetItemsInRoom(roomVNum) {
		if item.GetVNum() == objVNum {
			tbl := L.NewTable()
			tbl.RawSetString("vnum", lua.LNumber(item.GetVNum()))
			tbl.RawSetString("name", lua.LString(item.GetShortDesc()))
			tbl.RawSetString("alias", lua.LString(item.GetKeywords()))
			tbl.RawSetString("cost", lua.LNumber(item.GetCost()))
			tbl.RawSetString("timer", lua.LNumber(item.GetTimer()))
			// _src_room lets objfrom know which room to remove from
			tbl.RawSetString("_src_room", lua.LNumber(roomVNum))
			L.Push(tbl)
			return 1
		}
	}

	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaObjFrom(L *lua.LState) int {
	// objfrom(item, location) - remove item from location ('char' or 'room').
	// Removed item is held in e.transitItems until objto places it.
	// Source: scripts.c lua_objfrom() — calls obj_from_char() or obj_from_room().
	itemTbl := L.Get(1)
	location := L.ToString(2)

	if e.world == nil || itemTbl.Type() != lua.LTTable {
		return 0
	}

	vnumVal := L.GetField(itemTbl, "vnum")
	if vnumVal.Type() != lua.LTNumber {
		return 0
	}
	vnum := int(vnumVal.(lua.LNumber))

	var removed ScriptableObject

	switch location {
	case "room":
		// Determine source room: prefer _src_room field, fall back to me.room
		roomVNum := 0
		srcRoomVal := L.GetField(itemTbl, "_src_room")
		if srcRoomVal.Type() == lua.LTNumber {
			roomVNum = int(srcRoomVal.(lua.LNumber))
		} else {
			meVal := L.GetGlobal("me")
			if meVal.Type() == lua.LTTable {
				roomField := L.GetField(meVal, "room")
				if roomField.Type() == lua.LTNumber {
					roomVNum = int(roomField.(lua.LNumber))
				}
			}
		}
		if roomVNum > 0 {
			removed = e.world.RemoveItemFromRoom(vnum, roomVNum)
		}

	case "char":
		// Remove from mob's own inventory (me)
		meVal := L.GetGlobal("me")
		if meVal.Type() == lua.LTTable {
			nameField := L.GetField(meVal, "name")
			if nameField.Type() == lua.LTString {
				removed = e.world.RemoveItemFromChar(string(nameField.(lua.LString)), vnum)
			}
		}
	}

	if removed != nil {
		e.transitItems[vnum] = &transitEntry{obj: removed, placedAt: time.Now()}
	} else {
		slog.Debug("objfrom: item not found", "vnum", vnum, "location", location)
	}
	return 0
}

func (e *Engine) luaObjTo(L *lua.LState) int {
	// objto(item, location, target) - place item at location ('char' or 'room').
	// Retrieves the in-transit item placed by objfrom.
	// Source: scripts.c lua_objto() — calls obj_to_char() or obj_to_room().
	itemTbl := L.Get(1)
	location := L.ToString(2)
	targetVal := L.Get(3)

	if e.world == nil || itemTbl.Type() != lua.LTTable {
		return 0
	}

	vnumVal := L.GetField(itemTbl, "vnum")
	if vnumVal.Type() != lua.LTNumber {
		return 0
	}
	vnum := int(vnumVal.(lua.LNumber))

	entry, ok := e.transitItems[vnum]
	if !ok {
		slog.Debug("objto: no in-transit item", "vnum", vnum)
		return 0
	}

	switch location {
	case "char":
		charName := ""
		if targetVal.Type() == lua.LTTable {
			nameField := L.GetField(targetVal, "name")
			if nameField.Type() == lua.LTString {
				charName = string(nameField.(lua.LString))
			}
		}
		if charName == "" {
			slog.Debug("objto: char target has no name")
			return 0
		}
		if err := e.world.GiveItemToChar(charName, entry.obj); err != nil {
			slog.Debug("objto: GiveItemToChar error", "char", charName, "vnum", vnum, "error", err)
		} else {
			delete(e.transitItems, vnum)
		}

	case "room":
		roomVNum := 0
		if targetVal.Type() == lua.LTNumber {
			roomVNum = int(targetVal.(lua.LNumber))
		} else {
			// Default: me's room
			meVal := L.GetGlobal("me")
			if meVal.Type() == lua.LTTable {
				roomField := L.GetField(meVal, "room")
				if roomField.Type() == lua.LTNumber {
					roomVNum = int(roomField.(lua.LNumber))
				}
			}
		}
		if roomVNum == 0 {
			slog.Debug("objto: room target vnum is 0")
			return 0
		}
		if err := e.world.AddItemToRoom(entry.obj, roomVNum); err != nil {
			slog.Debug("objto: AddItemToRoom error", "room_vnum", roomVNum, "vnum", vnum, "error", err)
		} else {
			delete(e.transitItems, vnum)
		}
	}

	return 0
}

func (e *Engine) luaCreateEvent(L *lua.LState) int {
	// create_event(source, target, obj, argument, trigger, delay, type)
	// Source: scripts.c lua_create_event() lines 247-316 (commented out in original)
	//
	// Arguments:
	//   source  — me (mob table) or NIL
	//   target  — ch (player/mob table) or NIL
	//   obj     — obj (object table) or NIL
	//   argument — numeric argument or string (stored as int if numeric)
	//   trigger — Lua function name to call when event fires (e.g., "port", "jail")
	//   delay   — delay in PULSE_VIOLENCE units (1 = 2 seconds, 6 = 12 seconds)
	//   type    — event type: LT_MOB (1), LT_OBJ (2), LT_ROOM (3)
	//
	// In the original C code, delay was multiplied by PULSE_VIOLENCE (20 pulses)
	// to get the actual pulse count. The Go implementation does the same
	// conversion in WorldScriptableAdapter.CreateEvent().

	if e.world == nil {
		slog.Debug("create_event: no world available")
		return 0
	}

	// Parse source (arg 1) — extract mob instance ID from table
	sourceID := 0
	if srcTbl, ok := L.Get(1).(*lua.LTable); ok {
		// Try to get the "id" field (instance ID) or "vnum" field
		if idVal := srcTbl.RawGetString("id"); idVal.Type() == lua.LTNumber {
			sourceID = int(lua.LVAsNumber(idVal))
		} else if vnumVal := srcTbl.RawGetString("vnum"); vnumVal.Type() == lua.LTNumber {
			// Fallback: use vnum (less precise but works for simple cases)
			sourceID = int(lua.LVAsNumber(vnumVal))
		}
	}

	// Parse target (arg 2) — extract target ID from table
	targetID := 0
	if tgtTbl, ok := L.Get(2).(*lua.LTable); ok {
		if idVal := tgtTbl.RawGetString("id"); idVal.Type() == lua.LTNumber {
			targetID = int(lua.LVAsNumber(idVal))
		} else if vnumVal := tgtTbl.RawGetString("vnum"); vnumVal.Type() == lua.LTNumber {
			targetID = int(lua.LVAsNumber(vnumVal))
		}
	}

	// Parse obj (arg 3) — extract object vnum from table
	objVNum := 0
	if objTbl, ok := L.Get(3).(*lua.LTable); ok {
		if vnumVal := objTbl.RawGetString("vnum"); vnumVal.Type() == lua.LTNumber {
			objVNum = int(lua.LVAsNumber(vnumVal))
		}
	}

	// Parse argument (arg 4) — can be number or string
	argValue := 0
	if L.Get(4).Type() == lua.LTNumber {
		argValue = int(lua.LVAsNumber(L.Get(4)))
	}

	// Parse trigger (arg 5)
	trigger := ""
	if L.Get(5).Type() == lua.LTString {
		trigger = L.Get(5).String()
	}
	if trigger == "" {
		slog.Debug("create_event: no trigger specified")
		return 0
	}

	// Parse delay (arg 6) — in PULSE_VIOLENCE units
	delay := 1
	if L.Get(6).Type() == lua.LTNumber {
		delay = int(lua.LVAsNumber(L.Get(6)))
	}
	if delay < 1 {
		delay = 1 // events.c: "make sure its in the future"
	}

	// Parse event type (arg 7) — LT_MOB (1), LT_OBJ (2), LT_ROOM (3)
	eventType := 1 // default to LT_MOB
	if L.Get(7).Type() == lua.LTNumber {
		eventType = int(lua.LVAsNumber(L.Get(7)))
	}

	// Schedule the event
	eventID := e.world.CreateEvent(delay, sourceID, targetID, objVNum, argValue, trigger, eventType)
	if eventID > 0 {
		slog.Debug("created event", "event_id", eventID, "trigger", trigger, "delay", delay, "event_type", eventType, "source_id", sourceID)
	} else {
		slog.Debug("failed to create event", "trigger", trigger, "delay", delay, "event_type", eventType, "source_id", sourceID)
	}

	// Return the event ID to Lua (allows scripts to cancel events if needed)
	L.Push(lua.LNumber(eventID))
	return 1
}

// luaTell sends a private message to a named player.
// tell(player_name, message)
// Source: act.comm.c do_tell().
func (e *Engine) luaTell(L *lua.LState) int {
	targetName := L.CheckString(1)
	message := L.CheckString(2)
	if e.world != nil && targetName != "" {
		e.world.SendTell(targetName, message)
	}
	return 0
}

// luaPlrFlagged checks whether a player character has a given PLR_* flag set.
// plr_flagged(ch, flag) → bool
// Returns false for NPCs (they use mob flags, not PLR flags).
// Source: utils.h PLR_FLAGGED(ch, flag) macro.
func (e *Engine) luaPlrFlagged(L *lua.LState) int {
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}
	flagNum, ok := L.Get(2).(lua.LNumber)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}

	// NPCs never have PLR flags.
	if isNPC, ok := chTbl.RawGetString("is_npc").(lua.LBool); ok && bool(isNPC) {
		L.Push(lua.LBool(false))
		return 1
	}

	// Read raw flags bitmask serialised into the ch table by charToTable().
	rawFlags, ok := chTbl.RawGetString("plr_flags_raw").(lua.LNumber)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}

	bit := int(flagNum)
	if bit < 0 || bit >= 64 {
		L.Push(lua.LBool(false))
		return 1
	}
	result := (uint64(rawFlags)>>uint(bit))&1 == 1
	L.Push(lua.LBool(result))
	return 1
}

// luaCanSee checks whether the mob (me) can see character ch.
// cansee(ch) → bool
// Based on utils.h CAN_SEE() macro:
//   CAN_SEE(sub, obj) = SELF || ((GET_REAL_LEVEL(sub) >= GET_INVIS_LEV(obj)) && IMM_CAN_SEE(sub, obj))
//   IMM_CAN_SEE = MORT_CAN_SEE || PRF_HOLYLIGHT
//   MORT_CAN_SEE = LIGHT_OK && INVIS_OK
//   LIGHT_OK = !AFF_BLIND && (IS_LIGHT(room) || AFF_INFRAVISION)
//   INVIS_OK = !AFF_INVISIBLE(obj) || AFF_DETECT_INVIS(sub)
func (e *Engine) luaCanSee(L *lua.LState) int {
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}

	// Get the observer's level from the 'me' table (the mob casting the spell / running the script)
	var observerLevel = 0
	L.GetGlobal("me")
	if meTbl, meOk := L.Get(-1).(*lua.LTable); meOk {
		lvlL := meTbl.RawGetString("level")
		if levelNum, lvlOk := lvlL.(lua.LNumber); lvlOk {
			observerLevel = int(levelNum)
		}
	}
	L.Pop(1)

	// Get target's level as invise level proxy
	lvl, ok := chTbl.RawGetString("level").(lua.LNumber)
	if !ok || lvl <= 0 {
		L.Push(lua.LBool(false))
		return 1
	}
	objLevel := int(lvl)

	// Check invisibility: observer level must be >= target invis level
	// GET_INVIS_LEV in C; Lua tables don't carry invise_level separately,
	// so we use the target's level as a proxy for wizinvis.
	// PLR_INVSTART(14) would need the PLR flags check through plr_flagged
	if observerLevel < objLevel {
		// Can't see higher-level characters (invisibility approximation)
		L.Push(lua.LBool(false))
		return 1
	}

	// Check PLR_INVISIBLE (wizinvis) via plr_flags_raw
	// PLR_INVSTART = 14 in structs.h
	if isNPCL, npcOk := chTbl.RawGetString("is_npc").(lua.LBool); npcOk && !bool(isNPCL) {
		if rawFlagsL, flagsOk := chTbl.RawGetString("plr_flags_raw").(lua.LNumber); flagsOk {
			rawFlags := uint64(rawFlagsL)
			// PLR_INVSTART(14): wizinvis flag
			if (rawFlags>>14)&1 == 1 {
				// Observer also needs PLR_HOLYLIGHT or similar to see through
				// For now, only see-through if observer level is much higher
				if observerLevel < objLevel+10 {
					L.Push(lua.LBool(false))
					return 1
				}
			}
		}
	}

	// Check room darkness — requires world context and observer's room
	// IS_DARK checks ROOM_DARK flag, SECT_INSIDE, and sunlight
	L.GetGlobal("me")
	if meTbl, meOk := L.Get(-1).(*lua.LTable); meOk {
		roomL := meTbl.RawGetString("room")
		if observerRoom, roomOk := roomL.(lua.LNumber); roomOk && e.world != nil && int(observerRoom) > 0 {
			roomVNum := int(observerRoom)
			if e.world.IsRoomDark(roomVNum) {
				// Room is dark — check if observer can see in dark (AFF_INFRAVISION)
				// Not tracked in Lua tables yet; for now, dark rooms block sight
				// unless world says otherwise (handled by higher-level logic)
				L.Push(lua.LBool(false))
				L.Pop(1)
				return 1
			}
		}
	}
	L.Pop(1)

	L.Push(lua.LBool(true))
	return 1
}

// luaIsNPC returns true if ch is an NPC/mob, false if a player.
// isnpc(ch) → bool
// Reads the 'is_npc' field set by charToTable() (false) or mobToTable() (true).
// Source: utils.h IS_NPC(ch) macro — checks MOB_ISNPC in MOB_FLAGS(ch).
func (e *Engine) luaIsNPC(L *lua.LState) int {
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		// Unknown type — treat as non-NPC to be safe.
		L.Push(lua.LBool(false))
		return 1
	}
	isNPC, ok := chTbl.RawGetString("is_npc").(lua.LBool)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}
	L.Push(isNPC)
	return 1
}

func (e *Engine) luaFollow(L *lua.LState) int {
	// follow(ch, charm) - makes mob follow ch, optionally setting AFF_CHARM.
	// Source: scripts.c lua_follow() lines 542-569.
	// Engine gap: mob-to-player follow requires AddFollowerQuietMob which
	// needs World access not available through ScriptableWorld.
	// For now, the mob's follow state is set via ScriptableMob if available.
	slog.Debug("follow: not fully implemented — mob follow via scripting requires World access")
	return 0
}

func (e *Engine) luaMount(L *lua.LState) int {
	// mount(ch, mount_or_nil, position) - ride, dismount, or unmount.
	// Source: scripts.c lua_mount() lines 871-906.
	// position: "ride" = mount, "dismount" = dismount, "unmount" = force unmount.
	// Engine gap: mounting system requires World.doRide/doDismount which
	// needs actual Player/MobInstance objects, not available via ScriptableWorld.

	slog.Debug("mount: not fully implemented — requires World access for ride/dismount")
	return 0
}

func (e *Engine) luaDirection(L *lua.LState) int {
	// direction(from_vnum, to_vnum) - returns direction (0-5) from one room to another.
	// Source: scripts.c lua_direction() lines 317-340.
	// Returns -1 on error, -2 if already there, -3 if no path found.
	if e.world == nil {
		L.Push(lua.LNumber(-1))
		return 1
	}
	fromVNum := L.ToInt(1)
	toVNum := L.ToInt(2)
	if fromVNum == toVNum {
		L.Push(lua.LNumber(-2))
		return 1
	}
	result := e.world.FindFirstStep(fromVNum, toVNum)
	L.Push(lua.LNumber(result))
	return 1
}

func (e *Engine) luaSetHunt(L *lua.LState) int {
	// set_hunt(hunter, prey) - set mob to hunt a target.
	// Source: scripts.c lua_set_hunt() lines 1341-1363.
	if e.world == nil {
		return 0
	}
	hunterTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	victimName := L.ToString(2)

	// Get mob's vnum and room from table, look up via ScriptableWorld
	L.GetField(hunterTbl, "vnum")
	vnum := int(L.ToNumber(-1))
	L.Pop(1)
	L.GetField(hunterTbl, "room")
	roomVNum := int(L.ToNumber(-1))
	L.Pop(1)

	if vnum > 0 && roomVNum > 0 {
		mob := e.world.GetMobByVNumAndRoom(vnum, roomVNum)
		if mob != nil {
			mob.SetHunting(victimName)
		}
	}
	return 0
}

func (e *Engine) luaMxp(L *lua.LState) int {
	// mxp(text, command) - returns MXP-enabled link text. Falls back to plain text.
	// Source: merchant_inn.lua — creates clickable "interested?" link.
	if L.GetTop() >= 1 {
		L.Push(L.Get(1)) // Return the display text as-is
	} else {
		L.Push(lua.LString(""))
	}
	return 1
}

func (e *Engine) luaSkipSpaces(L *lua.LState) int {
	// skip_spaces(s) - trim leading spaces from a string.
	// Source: merchant_inn.lua — strips leading space from say argument.
	s := L.ToString(1)
	L.Push(lua.LString(strings.TrimLeft(s, " ")))
	return 1
}

func (e *Engine) luaUnaffect(L *lua.LState) int {
	// unaffect(ch) - remove all spell affections from character.
	// Source: scripts.c lua_unaffect() lines 1570-1607.
	if e.world == nil {
		return 0
	}
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	nameVal := chTbl.RawGetString("name")
	if nameVal.Type() != lua.LTString {
		return 0
	}
	charName := nameVal.String()
	// Check if mob (has vnum field) or player
	isMob := false
	vnumVal := chTbl.RawGetString("vnum")
	if vnumVal.Type() == lua.LTNumber {
		isMob = true
	}
	e.world.ClearAffects(charName, isMob)
	return 0
}

func (e *Engine) luaEquipChar(L *lua.LState) int {
	// equip_char(mob, obj) - equip a mob with an object.
	// Source: scripts.c lua_equip_char() lines 403-425.
	// Removes object from mob's inventory and equips in appropriate slot.
	if e.world == nil {
		return 0
	}
	mobTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	objTbl, ok := L.Get(2).(*lua.LTable)
	if !ok {
		return 0
	}

	// Get mob vnum and room
	L.GetField(mobTbl, "vnum")
	mobVNum := int(L.ToNumber(-1))
	L.Pop(1)
	L.GetField(mobTbl, "room")
	roomVNum := int(L.ToNumber(-1))
	L.Pop(1)

	// Get object vnum
	objVNumVal := objTbl.RawGetString("vnum")
	objVNum := int(0)
	if objVNumVal.Type() == lua.LTNumber {
		objVNum = int(objVNumVal.(lua.LNumber))
	}

	if mobVNum > 0 && roomVNum > 0 && objVNum > 0 {
		e.world.EquipMob(mobVNum, roomVNum, objVNum)
	}
	return 0
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
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaCanGet(L *lua.LState) int {
	// canget(obj) - returns true if the mob is permitted to pick up the object.
	// Source: scripts.c lua_canget() lines 193-219.
	// Checks CAN_GET_OBJ(ch, obj) — verifies ITEM_WEAR_TAKE flag and weight limits.
	// Engine gap: carry-weight system not yet implemented.
	// For now, return true (optimistic) since wear flags aren't fully mapped.
	L.Push(lua.LNumber(1))
	return 1
}

func (e *Engine) luaObjList(L *lua.LState) int {
	// obj_list(keyword, location) - search mob's inventory for item matching keyword
	// location: "char" = mob's inventory, "room" = room floor, "vict" = player inventory
	// Returns the object table if found, NIL otherwise
	// Based on lua_obj_list() pattern in scripts.c
	// obj_list: archived script dependency (no active callers)
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaItemCheck(L *lua.LState) int {
	// item_check(obj) - validates whether object is a production item for the shop.
	// Source: shop_give.lua — checks if given item is valid shop production.
	// Engine gap: always returns false — shop production tables not yet implemented.
	// item_check: shop production validation (shop system not yet wired)
	L.Push(lua.LBool(false))
	return 1
}

func (e *Engine) luaLoadRoom(L *lua.LState) int {
	// load_room(vnum) - returns a room table with vnum, char, exit, objs fields.
	// Source: pet_store.lua, merchant_inn.lua — loads adjacent room for pet listing.
	// load_room: adjacent room query (pet_store, merchant_inn archived scripts)
	vnum := L.ToInt(1)
	tbl := L.NewTable()
	tbl.RawSetString("vnum", lua.LNumber(vnum))
	tbl.RawSetString("char", L.NewTable())
	tbl.RawSetString("exit", L.NewTable())
	tbl.RawSetString("objs", L.NewTable())
	L.Push(tbl)
	return 1
}

func (e *Engine) luaInworld(L *lua.LState) int {
	// inworld(type, vnum) - check if a mob/obj with given vnum exists in the world.
	// Source: merchant_inn.lua — checks if travelling merchant (6805) already exists.
	// inworld: mob/obj existence check (merchant_inn archived script)
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaSocial(L *lua.LState) int {
	// social(mob, social_name) - perform a social command.
	// Source: remove_curse.lua — performs "cough" social.
	// social: perform social emote (remove_curse archived script)
	return 0
}

func (e *Engine) luaGetGroupLvl(L *lua.LState) int {
	// get_group_lvl(ch, group[, newval]) - get or set character's skill group level.
	// Source: teacher.lua — reads and writes group level for skill training.
	// get_group_lvl: skill group level (teacher.lua archived)
	if L.GetTop() >= 3 {
		return 0
	}
	L.Push(lua.LNumber(0))
	return 1
}

func (e *Engine) luaGetGroupPts(L *lua.LState) int {
	// get_group_pts(ch[, newval]) - get or set character's available group points.
	// Source: teacher.lua — reads and writes group points for skill training.
	// get_group_pts: skill group points (teacher.lua archived)
	if L.GetTop() >= 2 {
		return 0
	}
	L.Push(lua.LNumber(0))
	return 1
}

func (e *Engine) luaSkillGroup(L *lua.LState) int {
	// skill_group(name) - converts skill group name to numeric ID.
	// Source: teacher.lua — maps group names like "Rejuvenation" to IDs.
	// skill_group: group name to ID (teacher.lua archived)
	L.Push(lua.LNumber(0))
	return 1
}

// Flag check/set functions (ported from src/scripts.c) are in lua_aff_flags.go.
// Remaining stubs (luaMobFlags, luaObjExtra, luaExitFlagged, luaExitFlags, luaExtra)
// are defined below as placeholders until their table schemas are wired.

func (e *Engine) luaObjExtra(L *lua.LState) int {
	// obj_extra(obj, "set"|"remove", flag)
	// Engine gap: obj extra flags not yet on obj tables
	return 0
}

func (e *Engine) luaExtra(L *lua.LState) int {
	// extra(obj, text)
	// Engine gap: extra descriptions not yet mutable from Lua
	return 0
}

// --- Skill group stubs (archived teacher.lua, system not ported) ---

func (e *Engine) luaAffFlagged(L *lua.LState) int {
	// aff_flagged(ch, flag) → TRUE(1) or nil
	// Source: scripts.c lines 142-163
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	flag := L.ToInt(2)
	// Check affects_raw on mob tables, plr_flags_raw on player tables
	if val := tbl.RawGetString("affects_raw"); val.Type() == lua.LTNumber {
		if int(val.(lua.LNumber))&(1<<flag) != 0 {
			L.Push(lua.LNumber(1))
			return 1
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaAffFlags(L *lua.LState) int {
	// aff_flags(ch, "set"|"remove", flag)
	// Source: scripts.c lines 165-191
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	op := L.ToString(2)
	flag := L.ToInt(3)
	val := tbl.RawGetString("affects_raw")
	if val.Type() != lua.LTNumber {
		return 0
	}
	bits := int(val.(lua.LNumber))
	switch op {
	case "set":
		bits |= 1 << flag
	case "remove":
		bits &^= 1 << flag
	default:
		return 0
	}
	tbl.RawSetString("affects_raw", lua.LNumber(bits))
	return 0
}

func (e *Engine) luaPlrFlags(L *lua.LState) int {
	// plr_flags(ch, "set"|"remove", flag)
	// Source: scripts.c lines 1197-1223
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	op := L.ToString(2)
	flag := L.ToInt(3)
	val := tbl.RawGetString("plr_flags_raw")
	if val.Type() != lua.LTNumber {
		return 0
	}
	bits := int64(val.(lua.LNumber))
	switch op {
	case "set":
		bits |= 1 << flag
	case "remove":
		bits &^= 1 << flag
	default:
		return 0
	}
	tbl.RawSetString("plr_flags_raw", lua.LNumber(bits))
	return 0
}

func (e *Engine) luaMobFlagged(L *lua.LState) int {
	// mob_flagged(mob, flag) → TRUE(1) or nil
	// Source: scripts.c lines 820-841
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	flag := L.ToInt(2)
	val := tbl.RawGetString("mob_flags_raw")
	if val.Type() == lua.LTNumber {
		if int(val.(lua.LNumber))&(1<<flag) != 0 {
			L.Push(lua.LNumber(1))
			return 1
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaMobFlags(L *lua.LState) int {
	// mob_flags(mob, "set"|"remove", flag)
	// Source: scripts.c lines 843-869
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	op := L.ToString(2)
	flag := L.ToInt(3)
	val := tbl.RawGetString("mob_flags_raw")
	if val.Type() != lua.LTNumber {
		return 0
	}
	bits := int(val.(lua.LNumber))
	switch op {
	case "set":
		bits |= 1 << flag
	case "remove":
		bits &^= 1 << flag
	default:
		return 0
	}
	tbl.RawSetString("mob_flags_raw", lua.LNumber(bits))
	return 0
}

func (e *Engine) luaObjFlagged(L *lua.LState) int {
	// obj_flagged(obj, flag) → TRUE(1) or nil
	// Source: scripts.c lines 951-974
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	flag := L.ToInt(2)
	val := tbl.RawGetString("obj_flags_raw")
	if val.Type() == lua.LTNumber {
		if int(val.(lua.LNumber))&(1<<flag) != 0 {
			L.Push(lua.LNumber(1))
			return 1
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaExitFlagged(L *lua.LState) int {
	// exit_flagged(room, door, flag) → TRUE(1) or nil
	// Source: scripts.c lines 456-478
	// Engine gap: exit data not yet on room tables
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaExitFlags(L *lua.LState) int {
	// exit_flags(room, door, "set"|"remove", flag)
	// Source: scripts.c lines 427-454
	// Engine gap: exit data not yet on room tables
	return 0
}

func (e *Engine) luaIshunt(L *lua.LState) int {
	// ishunt(ch) → TRUE(1) if mob is hunting, else nil
	// Source: scripts.c lines 676-695
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LNil)
		return 1
	}
	val := tbl.RawGetString("hunting")
	if val.Type() == lua.LTString && val.String() != "" {
		L.Push(lua.LNumber(1))
		return 1
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaSteal(L *lua.LState) int {
	// steal(ch, obj) - steal an item from a character's inventory.
	// Simplified: transfers the object from the victim to the mob.
	if e.world == nil {
		return 0
	}
	// Arguments: mob_table, victim_name (or victim_table)
	if L.GetTop() < 2 {
		return 0
	}
	victimName := L.ToString(2)
	// Get mob room for the stolen item destination
	mobTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		return 0
	}
	L.GetField(mobTbl, "room")
	roomVNum := int(L.ToNumber(-1))
	L.Pop(1)

	// Try to remove a random item from victim and give to mob room
	if roomVNum > 0 && victimName != "" {
		obj := e.world.RemoveItemFromChar(victimName, 0) // vnum=0 means any item
		if obj != nil {
			e.world.AddItemToRoom(obj, roomVNum)
		}
	}
	return 0
}

func (e *Engine) luaEcho(L *lua.LState) int {
	// echo(ch, type, msg) — broadcast a message to a zone or room.
	// type="zone" sends to all players in the mob's zone.
	// type="room" sends to all players in the current room.
	// Based on lua_echo() in scripts.c lines 308-345.
	// Source: werewolf.lua — echo(me, "zone", "You hear a loud howling...")
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		slog.Warn("echo: first arg is not a table")
		return 0
	}
	echoType := L.ToString(2)
	msg := L.ToString(3)

	if e.world == nil {
		slog.Debug("echo: no world context", "echo_type", echoType, "msg", msg)
		return 0
	}

	switch echoType {
	case "zone":
		// Get the room VNum from the ch table
		roomVNumL := chTbl.RawGetString("room")
		roomVNum, ok := roomVNumL.(lua.LNumber)
		if !ok || roomVNum <= 0 {
			slog.Debug("echo: no room in ch table")
			return 0
		}
		e.world.SendToZone(int(roomVNum), msg+"\r\n")
		return 0
	case "room":
		roomVNumL := chTbl.RawGetString("room")
		roomVNum, ok := roomVNumL.(lua.LNumber)
		if !ok || roomVNum <= 0 {
			slog.Debug("echo: no room in ch table")
			return 0
		}
		for _, player := range e.world.GetPlayersInRoom(int(roomVNum)) {
			player.SendMessage(msg + "\r\n")
		}
		return 0
	default:
		slog.Warn("echo: unknown echo type", "echo_type", echoType)
		return 0
	}
}

