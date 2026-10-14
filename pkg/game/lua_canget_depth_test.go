package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func TestLuaCanGetUsesMeAndLiveObjectVisibility(t *testing.T) {
	for _, tc := range []struct {
		name        string
		dark        bool
		invisible   bool
		detect      bool
		infravision bool
		want        bool
	}{
		{name: "plain visible", want: true},
		{name: "invisible to mobile", invisible: true},
		{name: "detect invisible", invisible: true, detect: true, want: true},
		{name: "dark to mobile", dark: true},
		{name: "infravision", dark: true, infravision: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flags := []string{"0", "0", "0", "0"}
			if tc.dark {
				flags[0] = "1" // C ROOM_DARK
			}
			w, err := NewWorld(&parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "Canget test room", Flags: flags}},
				Mobs: []parser.Mob{{
					VNum: 3000, Keywords: "probe mobile", ShortDesc: "a probe mobile",
					Str: 18, Dex: 18, Int: 18, Wis: 18, Con: 18, Cha: 18,
				}},
				Objs: []parser.Obj{{
					VNum: 3001, Keywords: "probe gem", ShortDesc: "a probe gem",
					WearFlags: [4]int{1}, Weight: 1, Cost: 10,
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)

			trigger := NewPlayer(1, "Cangettrigger", 1001)
			trigger.SetHolyLight(true)
			if err := w.AddPlayer(trigger); err != nil {
				t.Fatal(err)
			}
			mob, err := w.SpawnMobQuiet(3000, 1001)
			if err != nil {
				t.Fatal(err)
			}
			if tc.detect {
				mob.SetAffected(affDetectInvisible)
			}
			if tc.infravision {
				mob.SetAffected(affInfravision)
			}
			obj, err := w.SpawnObject(3001, -1)
			if err != nil {
				t.Fatal(err)
			}
			if tc.invisible {
				obj.SetExtraFlag(0, extraFlagInvisible)
			}
			if err := w.MoveObjectToMobInventoryFront(obj, mob); err != nil {
				t.Fatal(err)
			}

			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "canget.lua"), []byte(
				"function oncmd()\n"+
					"local item = me.objs[1]\n"+
					"assert((canget(item) ~= nil) == "+boolLua(tc.want)+", 'canget result')\n"+
					"return TRUE\nend\n",
			), 0o600); err != nil {
				t.Fatal(err)
			}
			a := NewWorldScriptableAdapter(w)
			e := scripting.NewEngine(dir, a)
			t.Cleanup(e.Close)
			chRef := scripting.CharRef{ID: trigger.ID}
			meRef := scripting.CharRef{NPC: true, ID: mob.GetID()}
			ctx := &scripting.ScriptContext{World: a, ChRef: &chRef, MeRef: &meRef, RoomVNum: 1001}
			if handled, err := e.RunScript(ctx, "canget.lua", "oncmd"); err != nil || !handled {
				t.Fatalf("script contract: handled=%t err=%v", handled, err)
			}
		})
	}
}

func boolLua(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
