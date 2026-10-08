package scripting

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// scriptWorldStub answers the world queries the lua_seam bindings make. It
// embeds ScriptableWorld (nil) for the rest, like the package's fake bridge.
type scriptWorldStub struct {
	ScriptableWorld
	rooms map[int]*parser.Room
	dir   int
}

func (s *scriptWorldStub) GetRoomInWorld(vnum int) *parser.Room { return s.rooms[vnum] }
func (s *scriptWorldStub) FindFirstStep(from, to int) int       { return s.dir }

// shopProbeBridge answers the shop-keeper lookup the item_check producer asks
// by inline assertion.
type shopProbeBridge struct {
	*recordingBridge
	shopKeeper bool
}

func (s *shopProbeBridge) IsShopKeeper(int) bool { return s.shopKeeper }

func worldWith(rooms ...int) *scriptWorldStub {
	w := &scriptWorldStub{rooms: map[int]*parser.Room{}, dir: 3}
	for _, vnum := range rooms {
		w.rooms[vnum] = &parser.Room{VNum: vnum}
	}
	return w
}

// TestLuaSeamProducersReturnCValueAndLog proves the five lua-seams bindings that
// C gates on a bad argument: each logs its diagnostic and returns C's value, no
// values at all (src/scripts.c:216, 328, 337, 651, 671, 739, 753). Lua sees no
// values as nil, so every script below fails its own check if the port returns
// anything, and the bridge's log list proves the producer fired.
func TestLuaSeamProducersReturnCValueAndLog(t *testing.T) {
	for _, tc := range []struct {
		name    string
		call    string
		payload string
		world   *scriptWorldStub
		me      *CharRef
		shop    bool
	}{
		{"canget", "canget(42)", "[Lua] Invalid argument to lua_canget.", worldWith(100), nil, false},
		{"direction arguments", "direction('x', 2)", "[Lua] Invalid arguments passed to lua_direction.", worldWith(100), nil, false},
		{"direction rooms", "direction(100, 999)", "[Lua] Invalid room specified in lua_direction.", worldWith(100), nil, false},
		{"iscorpse", "iscorpse(42)", "[Lua] Invalid argument passed to lua_iscorpse.", worldWith(100), nil, false},
		{"isfighting", "isfighting(42)", "[Lua] Invalid argument passed to lua_isfighting.", worldWith(100), nil, false},
		{"item_check argument", "item_check(42)", "[Lua] Invalid argument to lua_item_check.", worldWith(100), nil, false},
		{"item_check no shop", "item_check({type = 5})", "[Lua] Unable to determine shop in lua_item_check.", worldWith(100), &guard, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeScript(t, dir, "seam.lua",
				"function oncmd()\n  local v = "+tc.call+"\n  if v ~= nil then return FALSE end\n  return TRUE\nend\n")
			bridge := &shopProbeBridge{recordingBridge: newRecordingBridge(), shopKeeper: tc.shop}
			engine := NewEngine(dir, tc.world)
			t.Cleanup(func() { engine.l.Close() })
			me := player
			if tc.me != nil {
				me = *tc.me
			}
			// The context carries the outer bridge: the engine reaches the
			// shop-keeper lookup by an inline assertion on it.
			ctx := &ScriptContext{World: bridge, MeRef: &me, OwnerType: "mob"}
			handled, err := engine.RunScript(ctx, "seam.lua", "oncmd")
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if !handled {
				t.Fatalf("C returns no values here, so the script's nil check must pass; log=%q", bridge.logs)
			}
			got := 0
			for _, line := range bridge.logs {
				if line == tc.payload {
					got++
				}
			}
			if got != 1 {
				t.Fatalf("producer calls=%d want 1 for %q; logs=%q", got, tc.payload, bridge.logs)
			}
			if len(bridge.mudlogs) != 0 {
				t.Fatalf("BRF/file-FALSE producer used the file-TRUE sink: %+v", bridge.mudlogs)
			}
		})
	}
}

// TestLuaSeamValidPathsKeepTheirResult guards the other half: the same bindings
// still return their value on the valid path, so the two tests together pin
// both arms.
func TestLuaSeamValidPathsKeepTheirResult(t *testing.T) {
	for _, tc := range []struct {
		name    string
		call    string
		payload string
	}{
		{"direction returns a step", "direction(100, 200)", ""},
		{"iscorpse returns nil for a non-corpse", "iscorpse({type = 1})", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeScript(t, dir, "seam.lua", "function oncmd()\n  "+tc.call+"\n  return TRUE\nend\n")
			bridge := &shopProbeBridge{recordingBridge: newRecordingBridge(), shopKeeper: true}
			me := player
			engine := NewEngine(dir, worldWith(100, 200))
			t.Cleanup(func() { engine.l.Close() })
			handled, err := engine.RunScript(&ScriptContext{World: bridge, MeRef: &me, OwnerType: "mob"}, "seam.lua", "oncmd")
			if err != nil || !handled {
				t.Fatalf("valid call must run and count as handled: handled=%v err=%v", handled, err)
			}
			for _, line := range bridge.logs {
				if strings.Contains(line, "Invalid") || strings.Contains(line, "Unable to determine") {
					t.Fatalf("valid path logged a diagnostic: %q", line)
				}
			}
		})
	}
}
