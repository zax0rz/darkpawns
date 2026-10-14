package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// The real Lua binding must pass me.objs through lua_spell to both the vocal
// cast_spell path and the silent call_magic path. C authority:
// src/scripts.c:1428-1489; src/spell_parser.c:291-320,400-557,827-910;
// src/magic.c:1900-1945; src/structs.h:472-476.
func TestLuaSpellObjectTargetDispatchAndSpeechGate(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Spell room"}},
		Mobs:  []parser.Mob{{VNum: 3000, Keywords: "caster", ShortDesc: "a test caster", Level: 30, Int: 26, Wis: 26}},
		Objs:  []parser.Obj{{VNum: 3001, Keywords: "gem", ShortDesc: "a test gem", Weight: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	observer := NewPlayer(1, "Spellobserver", 1001)
	observer.SetHolyLight(true)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	output := captureOutput(w)
	mob, err := w.SpawnMob(3000, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetPosition(combat.PosStanding)
	adapter := NewWorldScriptableAdapter(w)
	mobRef := scripting.CharRef{NPC: true, ID: mob.GetID()}
	objRef, ok := adapter.LoadObjToChar(3001, mobRef)
	if !ok {
		t.Fatal("LoadObjToChar failed")
	}
	obj := adapter.resolveObj(objRef)
	if obj == nil {
		t.Fatal("loaded object did not resolve")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	script := `function oncmd()
local o = me.objs[1]
local visible = spell(nil, o, 29, 1)
assert(type(visible) == "userdata", "vocal object target return")
assert(obj_flagged(o, 5) == 1, "ITEM_INVISIBLE must be set")
say("object visible=" .. tostring(obj_flagged(o, 5)))
local blessed = spell(nil, o, 3, 0)
assert(type(blessed) == "userdata", "silent object target return")
assert(obj_flagged(o, 8) == 1, "ITEM_BLESS must be set")
say("object blessed=" .. tostring(obj_flagged(o, 8)))
return TRUE
end`
	if err := os.WriteFile(filepath.Join(dir, "spell.lua"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	engine := scripting.NewEngine(dir, adapter)
	t.Cleanup(engine.Close)
	playerRef := scripting.CharRef{ID: observer.ID}
	ctx := &scripting.ScriptContext{World: adapter, ChRef: &playerRef, MeRef: &mobRef, RoomVNum: 1001}
	handled, err := engine.RunScript(ctx, "spell.lua", "oncmd")
	if err != nil || !handled {
		t.Fatalf("Lua dispatch handled=%t err=%v", handled, err)
	}
	if !obj.HasExtraFlag(0, 5) || !obj.HasExtraFlag(0, 8) {
		t.Fatalf("object extra flags word0=%#x, want invisible bit 5 and bless bit 8", obj.ExtraFlagWord(0))
	}
	transcript := outputOf(output, observer.GetName())
	for _, want := range []string{
		"stares at a test gem",
		"A test gem vanishes.",
		"test gem glows briefly with a ethereal light.",
		"object visible=1",
		"object blessed=1",
	} {
		if !strings.Contains(transcript, want) {
			t.Errorf("room transcript missing %q: %q", want, transcript)
		}
	}
}
