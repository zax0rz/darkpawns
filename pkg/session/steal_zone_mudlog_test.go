package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Real command/editor dispatch and manager delivery. Defensive editor states
// are injected explicitly; these tests make no valid-play reachability claim.
func testStealZoneMudlog(t *testing.T, kind string) {
	t.Helper()
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	a := makeZeditTestSession(t, m, "Steallog", 40)
	v := makeCommandTestSession(t, m, "Marklog", 10, 3000)
	typ := game.MudlogComplete
	if strings.HasPrefix(kind, "zone") {
		typ = game.MudlogBrief
	}
	watch := makeCommandTestSession(t, m, "Stealwatch", 31, 3001)
	below := makeCommandTestSession(t, m, "Stealbelow", 30, 3001)
	less := makeCommandTestSession(t, m, "Stealless", 40, 3001)
	watch.player.SetPlrFlag(game.PrfLog1, typ&1 != 0)
	watch.player.SetPlrFlag(game.PrfLog2, typ&2 != 0)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	less.player.SetPlrFlag(game.PrfLog1, (typ-1)&1 != 0)
	less.player.SetPlrFlag(game.PrfLog2, (typ-1)&2 != 0)
	for _, s := range []*Session{a, v, watch, below, less} {
		registerTestSession(t, m, s, s.player.GetName())
	}
	run := func(cmd string, args ...string) {
		t.Helper()
		if err := ExecuteCommand(a, cmd, args); err != nil {
			t.Fatal(err)
		}
	}
	payload := ""
	var invoke, at func()
	item := game.NewObjectInstance(&parser.Obj{VNum: 3008, Keywords: "proof ring", ShortDesc: "a proof ring", Weight: 1}, 3000)
	switch kind {
	case "inventory", "equipment", "failure":
		a.player.SetInvisLevel(40) // success producers ignore invisibility
		a.player.SetPlrFlag(game.PlrOutlaw, true)
		a.player.SetSkill(game.SkillSteal, 100)
		a.player.Stats.Str = 25
		a.player.Stats.Dex = 25
		a.player.CopyBaseAttributes()
		if kind == "equipment" {
			v.player.SetPosition(combat.PosSleeping)
			if err := v.player.Equipment.SetSlot(game.SlotFingerR, item); err != nil {
				t.Fatal(err)
			}
			item.Location = game.LocEquippedPlayer(v.player.Name, game.SlotFingerR)
		} else {
			if err := v.player.Inventory.AddItem(item); err != nil {
				t.Fatal(err)
			}
			item.Location = game.LocInventoryPlayer(v.player.Name)
		}
		payload = "(PS) Steallog stole a proof ring from Marklog."
		if kind == "failure" {
			a.player.SetLevel(30)
			a.player.SetSkill(game.SkillSteal, 0)
			a.player.SetInvisLevel(0)
			v.player.SetLevel(40)
			w.MovePlayerToRoom(watch.player, 3000)
			payload = "(PS) Steallog unsuccessfuly tried to steal from Marklog."
		}
		invoke = func() { dprng.ResetStream(1); run("steal", "proof", "Marklog") }
		at = func() {
			if a.player.WaitState != 0 {
				t.Error("steal log must precede WAIT_STATE")
			}
			if kind == "failure" {
				if got := strings.Join(drainSessionText(t, a), ""); !strings.Contains(got, "catches you trying to steal") {
					t.Errorf("actor caught message must precede log: %q", got)
				}
				if got := strings.Join(drainSessionText(t, v), ""); !strings.Contains(got, "tried to steal something from you") {
					t.Errorf("victim caught message must precede log: %q", got)
				}
				if got := strings.Join(drainSessionText(t, watch), ""); !strings.Contains(got, "tries to steal something from Marklog") {
					t.Errorf("room caught message must precede log: %q", got)
				}
				if a.player.GetFlags()&(1<<uint(game.PlrOutlaw)) == 0 {
					t.Error("outlaw must precede log")
				}
			} else {
				if _, ok := a.player.Inventory.FindItem("proof"); !ok {
					t.Error("transfer must precede log")
				}
				if v.player.IsAffected(38) {
					t.Error("robbed affect must follow log")
				} // AFF_ROBBED, src/structs.h
				if len(a.send) != 0 {
					t.Error("success acknowledgement must follow log")
				}
			}
		}
	case "zonearg3", "zonedefault":
		a.player.SetInvisLevel(40)
		run("zedit", "3000")
		if a.zedit == nil {
			t.Fatal("editor did not open")
		}
		a.zedit.zone.Commands[0].Arg2 = 999
		a.zedit.position = 0
		if kind == "zonearg3" {
			a.zedit.mode = zeditArg3
			payload = "SYSERR: OLC: zedit_parse(): case ARG3: Ack!"
		} else {
			a.zedit.mode = zeditMode(255)
			payload = "SYSERR: OLC: zedit_parse(): Reached default case!"
		}
		invoke = func() { a.handleZeditInput("1") }
		at = func() {
			if a.zedit != nil || a.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
				t.Error("zone log must follow full cleanup")
			}
			if m.zoneEditHolder(3000) != "" {
				t.Error("zone reservation must be released before log")
			}
			if z, _ := w.GetZone(30); z.Commands[0].Arg2 == 999 {
				t.Error("invalid editor must discard changes")
			}
		}
	default:
		t.Fatal("unknown case")
	}
	for _, s := range []*Session{a, v, watch, below, less} {
		drainSessionText(t, s)
	}
	file := captureMudlogFile(t)
	sink := w.MessageSink
	called := false
	w.MessageSink = func(name string, b []byte) {
		if name == watch.player.Name && strings.Contains(string(b), "[ "+payload) {
			called = true
			at()
		}
		sink(name, b)
	}
	invoke()
	if !called {
		t.Fatalf("missing %s producer; actor=%q victim=%q", kind, drainSessionText(t, a), drainSessionText(t, v))
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+payload+" ]\r\n" {
		t.Fatalf("producer bytes=%q", got)
	}
	if !strings.Contains(file.String(), payload) {
		t.Errorf("missing file payload: %q", file.String())
	}
	if len(below.send) != 0 || len(less.send) != 0 {
		t.Error("minimum/type filter leak")
	}
	if strings.HasPrefix(kind, "zone") {
		if a.zedit != nil {
			t.Fatal("editor retained")
		}
		return
	}
	if a.player.WaitState == 0 {
		t.Error("steal cooldown absent")
	}
	// NPC paths must not emit any PC-steal producer. This direct game
	// boundary complements, rather than replaces, the live PC command proof.
	file.Reset()
	mob := game.NewMob(&parser.Mob{VNum: 3010, Keywords: "npctarget", ShortDesc: "an NPC target", Level: 10}, 3000)
	npcItem := game.NewObjectInstance(&parser.Obj{VNum: 3011, Keywords: "proof ring", ShortDesc: "a proof ring", Weight: 1}, 3000)
	if kind == "equipment" {
		mob.SetPosition(combat.PosSleeping)
		npcItem.Location = game.LocNowhere()
		if err := w.EquipMobileObject(mob, npcItem, 1); err != nil {
			t.Fatal(err)
		} // C WEAR_FINGER_R
	} else {
		mob.AddToInventory(npcItem)
	}
	if kind == "failure" {
		mob.SetLevel(40)
	}
	dprng.ResetStream(1)
	npcResult := game.DoSteal(a.player, mob, "proof", w)
	if npcResult.Success != (kind != "failure") {
		t.Fatalf("NPC classifier missed intended branch: %+v", npcResult)
	}
	if strings.Contains(file.String(), "(PS)") || len(watch.send) != 0 {
		t.Error("NPC steal path logged")
	}
	// Early refusal must never reuse the failure tail marker or log.
	file.Reset()
	run("steal", "proof", "Nobody")
	if strings.Contains(file.String(), "(PS)") || len(watch.send) != 0 {
		t.Error("missing-target refusal logged")
	}
}
