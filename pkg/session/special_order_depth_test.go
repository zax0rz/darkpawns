package session

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type orderScriptRecorder struct{ visit func(string) bool }

func (orderScriptRecorder) ForgetFailures() {}
func (r orderScriptRecorder) RunScript(_ *game.ScriptContext, filename, trigger string) (bool, error) {
	return r.visit(filename), nil
}

// TestSpecialOrderDepth covers every adjacent native/Lua boundary and each
// first-TRUE exit in src/interpreter.c:1414-1476 through ExecuteCommand (R5e).
func TestSpecialOrderDepth(t *testing.T) {
	order := []string{"room-native", "room-lua", "worn-native", "worn-lua", "carried-native", "carried-lua", "mob-native", "mob-lua", "floor-native", "floor-lua"}
	for stop := -1; stop < len(order); stop++ {
		t.Run(fmt.Sprintf("consume-%d", stop), func(t *testing.T) {
			var calls []string
			visit := func(name string) bool { calls = append(calls, name); return stop >= 0 && name == order[stop] }
			installRoomSpec(t, 1001, "depth_room", func(_ *game.World, _ *game.Player, _ *game.MobInstance, _, _ string) bool {
				return visit("room-native")
			})
			installMobSpec(t, 2001, "depth_mob", func(_ *game.World, _ *game.Player, _ *game.MobInstance, _, _ string) bool { return visit("mob-native") })
			for i, tier := range []string{"worn", "carried", "floor"} {
				installObjSpec(t, 7001+i, "depth_"+tier, func(_ *game.World, _ *game.Player, _ *game.ObjectInstance, _, _ string) bool {
					return visit(tier + "-native")
				})
			}
			obj := func(vnum int, tier string) parser.Obj {
				return parser.Obj{VNum: vnum, Keywords: "token", ShortDesc: "a token", ScriptName: tier + "-lua", LuaFunctions: 2, WearFlags: [4]int{1 | 1<<13}}
			}
			w := mustWorld(t, &parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "Order room", Zone: 1, ScriptName: "room-lua", ScriptFunctions: 1 << 5}},
				Objs:  []parser.Obj{obj(7001, "worn"), obj(7002, "carried"), obj(7003, "floor")},
				Mobs:  []parser.Mob{{VNum: 2001, Keywords: "probe", ShortDesc: "a probe", ScriptName: "mob-lua", LuaFunctions: 512}},
			})
			m := newTestManager(t, w, nil)
			s := makeCommandTestSession(t, m, "Orderprobe", 1, 1001)
			s.player.SetPosition(combat.PosStanding)
			if err := w.AddPlayer(s.player); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 3; i++ {
				item, err := w.SpawnObject(7001+i, -1)
				if err != nil {
					t.Fatal(err)
				}
				switch i {
				case 0:
					err = s.player.Equipment.Equip(item, s.player.Inventory)
				case 1:
					err = w.MoveObjectToPlayerInventory(item, s.player)
				case 2:
					err = w.MoveObjectToRoom(item, 1001)
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if _, err := w.SpawnMob(2001, 1001); err != nil {
				t.Fatal(err)
			}
			previous := game.ScriptEngine
			game.ScriptEngine = orderScriptRecorder{visit: visit}
			t.Cleanup(func() { game.ScriptEngine = previous })
			if err := ExecuteCommand(s, "sit", nil); err != nil {
				t.Fatal(err)
			}
			wantPosition := combat.PosSitting
			if stop >= 0 {
				wantPosition = combat.PosStanding
			}
			if s.player.GetPosition() != wantPosition {
				t.Fatalf("handler position = %d, want %d", s.player.GetPosition(), wantPosition)
			}
			want := order
			if stop >= 0 {
				want = order[:stop+1]
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("dispatch calls = %v, want %v", calls, want)
			}
		})
	}
}

// C resolves and rejects position before special(), src/interpreter.c:910-948.
func TestSpecialOrderEntryGates(t *testing.T) {
	for _, tc := range []struct {
		name, command string
		position      int
		want          []string
	}{
		{"unknown", "zzinvalid", combat.PosStanding, nil},
		{"position", "look", combat.PosSleeping, nil},
		{"resolved", "s", combat.PosStanding, []string{"south"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			installRoomSpec(t, specProbeRoom, "depth_gate", func(_ *game.World, _ *game.Player, _ *game.MobInstance, cmd, _ string) bool {
				calls = append(calls, cmd)
				return true
			})
			_, s := specProbeSession(t, nil)
			s.player.SetPosition(tc.position)
			if err := ExecuteCommand(s, tc.command, nil); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, tc.want) {
				t.Fatalf("special calls = %v, want %v", calls, tc.want)
			}
		})
	}
}
