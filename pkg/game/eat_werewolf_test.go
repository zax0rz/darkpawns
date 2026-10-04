package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func werewolfEatWorld(t *testing.T) (*World, *Player, *ObjectInstance, map[string]string) {
	t.Helper()
	w, p, _ := newSpecProcTestWorld(t)
	w.mu.Lock()
	w.objs[19] = &parser.Obj{VNum: 19, Keywords: "mangled flesh meat", ShortDesc: "some mangled flesh", LongDesc: "Some mangled flesh is here.", TypeFlag: ITEM_CONTAINER, Values: [4]int{3, 2, 1, 0}}
	w.mu.Unlock()
	corpse := registerBeheadObject(t, w, &parser.Obj{VNum: 6010, Keywords: "corpse trainee", ShortDesc: "the corpse of a trainee", TypeFlag: ITEM_CONTAINER, Values: [4]int{1000, 0, -1, 1}})
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	peer := NewPlayer(2, "Watcher", 1001)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatal(err)
	}
	p.SetAffect(affWerewolf, true)
	p.SetLevel(8)
	return w, p, corpse, out
}

func TestWerewolfCorpseConsumeStateAndAudience(t *testing.T) {
	w, p, corpse, out := werewolfEatWorld(t)
	p.SetCondition(CondFull, 39)
	first := registerBeheadObject(t, w, &parser.Obj{VNum: 6011, Keywords: "first ring", ShortDesc: "a first ring", TypeFlag: ITEM_OTHER})
	second := registerBeheadObject(t, w, &parser.Obj{VNum: 6012, Keywords: "second ring", ShortDesc: "a second ring", TypeFlag: ITEM_OTHER})
	for _, item := range []*ObjectInstance{first, second} {
		if err := w.MoveObjectToContainer(item, corpse); err != nil {
			t.Fatal(err)
		}
	}
	order := append([]*ObjectInstance(nil), corpse.Contains...)
	w.DoEat(p, nil, "eat", "cor", scmdEat)
	if got, want := out[p.Name], "You savagely rip into the corpse of a trainee, feeding your insatiable appetite.\r\n"; got != want {
		t.Fatalf("actor: %q want %q", got, want)
	}
	if got, want := out["Watcher"], "Tester savagely rips into the corpse of a trainee, crunching through flesh and bone alike.\r\n"; got != want {
		t.Fatalf("observer: %q want %q", got, want)
	}
	if p.GetCondition(CondFull) != 43 || p.Hunger != 43 {
		t.Fatalf("FULL/legacy hunger = %d/%d, want 43", p.GetCondition(CondFull), p.Hunger)
	}
	if corpse.Location != LocNowhere() || len(corpse.Contains) != 0 {
		t.Fatal("corpse not extracted")
	}
	if _, exists := w.objectInstances[corpse.ID]; exists {
		t.Fatal("corpse registry survived extraction")
	}
	floor := w.GetItemsInRoom(1001)
	if len(floor) != 3 || floor[0].GetVNum() != 19 || floor[1] != order[1] || floor[2] != order[0] {
		t.Fatalf("wrong C prepend order: %#v", floor)
	}
	flesh := floor[0]
	if flesh.GetValue(0) != 0 || flesh.GetValue(1) != 2 || flesh.GetValue(2) != 1 || flesh.GetValue(3) != 1 || flesh.GetTimer() != MaxNPCCorpseTime || flesh.Location != LocRoom(1001) {
		t.Fatalf("wrong flesh state: values=%v timer=%d location=%+v", [4]int{flesh.GetValue(0), flesh.GetValue(1), flesh.GetValue(2), flesh.GetValue(3)}, flesh.GetTimer(), flesh.Location)
	}
	if flesh.Prototype.Values != ([4]int{3, 2, 1, 0}) {
		t.Fatal("flesh instance modified prototype")
	}
	for _, child := range order {
		if child.Location != LocRoom(1001) || w.objectInstances[child.ID] != child {
			t.Fatal("spill extracted or mislocated a child")
		}
	}
}

func TestWerewolfCorpseFullBoundariesAndTaste(t *testing.T) {
	for _, tc := range []struct{ full, level, want int }{{-1, 40, -1}, {0, 9, 4}, {39, 40, 59}, {40, 8, 40}, {48, 8, 48}} {
		t.Run(fmt.Sprint(tc), func(t *testing.T) {
			w, p, _, _ := werewolfEatWorld(t)
			p.SetLevel(tc.level)
			p.SetCondition(CondFull, tc.full)
			// C's floor corpse branch precedes subcmd dispatch: taste also consumes it.
			w.DoEat(p, nil, "taste", "corpse", scmdTaste)
			if got := p.GetCondition(CondFull); got != tc.want {
				t.Fatalf("FULL=%d want %d", got, tc.want)
			}
			floor := w.GetItemsInRoom(1001)
			if len(floor) != 1 || floor[0].GetVNum() != 19 {
				t.Fatal("taste did not replace the corpse with flesh")
			}
		})
	}
}

func TestWerewolfCorpseEntryGates(t *testing.T) {
	for _, gate := range []string{"ordinary-player", "invisible", "ordinary-container", "floor-food", "carried-corpse"} {
		t.Run(gate, func(t *testing.T) {
			w, p, corpse, out := werewolfEatWorld(t)
			want := "Eat what?!?\r\n"
			switch gate {
			case "ordinary-player":
				p.SetAffect(affWerewolf, false)
				want = "You don't seem to have a corpse.\r\n"
			case "invisible":
				corpse.SetExtraFlag(0, extraFlagInvisible)
				want = "You don't seem to have a corpse.\r\n"
			case "ordinary-container":
				corpse.SetValue(3, 0)
			case "floor-food":
				corpse.Prototype.TypeFlag = ITEM_FOOD
			case "carried-corpse":
				if err := w.MoveObjectToPlayerInventory(corpse, p); err != nil {
					t.Fatal(err)
				}
				want = "You can't eat THAT!\r\n"
			}
			w.DoEat(p, nil, "eat", "corpse", scmdEat)
			if got := out[p.Name]; got != want {
				t.Fatalf("gate %s: %q want %q", gate, got, want)
			}
			if strings.Contains(out["Watcher"], "savagely") || corpse.Location == LocNowhere() {
				t.Fatal("refused corpse consumed")
			}
		})
	}
}
