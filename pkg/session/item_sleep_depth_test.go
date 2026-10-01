package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestItemSleepEntryDepth enters the real potion handler. C authority:
// src/spell_parser.c:699-718; src/magic.c:1199-1248,1406-1421.
func TestItemSleepEntryDepth(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3, 5, 8} {
		for _, outlaw := range []bool{false, true} {
			for _, sand := range []bool{false, true} {
				for _, resist := range []bool{false, true} {
					t.Run(fmt.Sprintf("seed%d/outlaw%t/sand%t/resist%t", seed, outlaw, sand, resist), func(t *testing.T) {
						w, err := game.NewWorld(&parser.World{
							Rooms: []parser.Room{{VNum: 1001, Name: "Sleep lab"}},
							Objs: []parser.Obj{
								{VNum: 9001, Keywords: "dose", ShortDesc: "a sleeping potion", TypeFlag: 10, WearFlags: [4]int{1}, Values: [4]int{40, 38, -1, -1}},
								{VNum: 1226, Keywords: "sand", ShortDesc: "a pinch of sand", TypeFlag: 13, WearFlags: [4]int{1}},
								{VNum: 1227, Keywords: "sand", ShortDesc: "counterfeit sand", TypeFlag: 13, WearFlags: [4]int{1}},
							},
						})
						if err != nil {
							t.Fatal(err)
						}
						t.Cleanup(w.StopAITicker)
						m := newTestManager(t, w, nil)
						s := makeTestSession(t, m, "Drinker", 1001, true)
						s.player.SetLevel(8)
						s.player.SetClass(combat.ClassMage)
						s.player.SetPosition(combat.PosStanding)
						s.player.SetPlrFlag(game.PlrOutlaw, outlaw)
						mod := 1000
						if resist {
							mod = -1000
						}
						s.player.SetSavingThrow(int(spells.SaveRodStaff), mod)
						// Opposite spell-save modifier makes using SAVING_SPELL detectable.
						s.player.SetSavingThrow(int(spells.SaveSpell), -mod)
						if err := w.AddPlayer(s.player); err != nil {
							t.Fatal(err)
						}
						observer := game.NewPlayer(2, "Observer", 1001)
						if err := w.AddPlayer(observer); err != nil {
							t.Fatal(err)
						}
						messages := map[string]string{}
						w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
						add := func(vnum int) {
							o, err := w.SpawnObject(vnum, -1)
							if err != nil {
								t.Fatal(err)
							}
							if err := w.MoveObjectToPlayerInventory(o, s.player); err != nil {
								t.Fatal(err)
							}
						}
						add(1227)
						if sand {
							add(1226)
						}
						add(9001)
						dprng.ResetStream(seed)
						expected := dprng.New(seed)
						saved := false
						if outlaw {
							roll := expected.Number(0, 99)
							saved = resist && roll > 1
						}
						if err := cmdQuaff(s, []string{"dose"}); err != nil {
							t.Fatal(err)
						}
						if got, want := dprng.Next(), expected.Next(); got != want {
							t.Fatalf("save draw boundary: next=%d want %d", got, want)
						}
						if !strings.Contains(drainSendChannel(t, s), "You quaff a sleeping potion.") {
							t.Fatal("potion entry bytes missing")
						}
						want := "You attempt the spell without the components...\r\n"
						if sand {
							want = "Pulling a bit of sand from a pocket, you cast it about the room...\r\n"
						}
						if !outlaw {
							want += "Your spell fails to affect them because you are not an Outlaw!\r\nDrinker tried to cast a spell on you but failed because Drinker is not an Outlaw!\r\n"
						}
						slept := outlaw && !saved
						if slept {
							want += "You feel very sleepy...  Zzzz......\r\n"
						}
						if got := messages["Drinker"]; got != want {
							t.Fatalf("self bytes=%q want %q", got, want)
						}
						wantObserver := ""
						if sand {
							wantObserver = "Drinker pulls a bit of sand out of a pocket and casts it about the room.\r\n"
						}
						if slept {
							wantObserver += "Drinker goes to sleep.\r\n"
						}
						if outlaw && saved {
							wantObserver += "Drinker shakes his head wearily, but then snaps out of it!\r\n"
						}
						if got := messages["Observer"]; got != wantObserver {
							t.Fatalf("observer bytes=%q want %q", got, wantObserver)
						}
						if s.player.GetWaitState() != engine.PULSE_VIOLENCE {
							t.Fatalf("wait=%d want %d", s.player.GetWaitState(), engine.PULSE_VIOLENCE)
						}
						if s.player.Inventory.GetItemCount() != 1 || len(s.player.Inventory.FindItems("dose")) != 0 {
							t.Fatal("potion/component not consumed")
						}
						if s.player.Inventory.Items[0].VNum != 1227 {
							t.Fatal("keyword counterfeit consumed instead of exact reagent")
						}
						wantPos := combat.PosStanding
						wantAffects := 0
						if slept {
							wantPos = combat.PosSleeping
							wantAffects = 1
						}
						if s.player.GetPosition() != wantPos || len(s.player.ActiveAffects) != wantAffects {
							t.Fatalf("sleep state position=%d affects=%v", s.player.GetPosition(), s.player.ActiveAffects)
						}
						if slept {
							af := s.player.ActiveAffects[0]
							duration := 6
							if sand {
								duration++
							}
							if af.SpellID != spells.SpellSleep || af.Duration != duration || af.Flags != engine.AFFSleep {
								t.Fatalf("sleep affect=%+v", af)
							}
						}
					})
				}
			}
		}
	}
}
