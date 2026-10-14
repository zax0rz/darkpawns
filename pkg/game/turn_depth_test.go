package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func turnDepthWorld(t *testing.T, exits bool) (*World, *Player, *Player) {
	t.Helper()
	room := parser.Room{VNum: 1001, Name: "Turn proof", Zone: 1}
	destination := parser.Room{VNum: 1002, Name: "Flee destination", Zone: 1}
	if exits {
		room.Exits = map[string]parser.Exit{}
		for _, direction := range dirs {
			room.Exits[direction] = parser.Exit{Direction: direction, ToRoom: destination.VNum}
		}
	}
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{room, destination},
		Mobs: []parser.Mob{
			{VNum: 99001, Keywords: "lich", ShortDesc: "a pale lich", Level: 30, Race: 9},
			{VNum: 99002, Keywords: "vampire", ShortDesc: "a vampire", Level: 30, Race: 8},
			{VNum: 99003, Keywords: "zombie", ShortDesc: "a zombie", Level: 30, Race: 0},
			{VNum: 99004, Keywords: "undead", ShortDesc: "a restless undead", Level: 17, Race: 9},
			{VNum: 99005, Keywords: "wraith", ShortDesc: "a quiet wraith", Level: 22, Race: 9},
			{VNum: 99006, Keywords: "young lich", ShortDesc: "a young lich", Level: 7, Race: 9},
			{VNum: 99007, Keywords: "young vampire", ShortDesc: "a young vampire", Level: 7, Race: 8},
		},
		Objs: []parser.Obj{{VNum: 18, Keywords: "dust ash", ShortDesc: "a pile of dust", LongDesc: "A pile of dust is here."}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	actor := NewPlayer(1, "Turner", 1001)
	actor.SetLevel(22)
	actor.SetAlignment(1000)
	actor.SetSkill(SkillTurn, 100)
	actor.SetMana(1000)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatal(err)
	}
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	return w, actor, observer
}

func capturePlayerMessages(w *World) map[string]*strings.Builder {
	messages := make(map[string]*strings.Builder)
	w.MessageSink = func(name string, message []byte) {
		if messages[name] == nil {
			messages[name] = &strings.Builder{}
		}
		messages[name].Write(message)
	}
	return messages
}

func TestDoTurnScansCOrderedRoomBodiesByNumericRace(t *testing.T) {
	w, actor, observer := turnDepthWorld(t, false)
	vampirePC := NewPlayer(3, "VampirePC", 1001)
	vampirePC.Race = 8
	vampirePC.SetLevel(30)
	if err := w.AddPlayer(vampirePC); err != nil {
		t.Fatal(err)
	}
	private := make(map[*MobInstance]string)
	spawned := make(map[int]*MobInstance)
	for _, vnum := range []int{99001, 99002, 99003, 99005} {
		mob, err := w.SpawnMobQuiet(vnum, 1001)
		if err != nil {
			t.Fatal(err)
		}
		spawned[vnum] = mob
	}
	w.MobileMessageSink = func(body *MobInstance, message []byte) { private[body] += string(message) }
	seed := uint32(2)
	dprng.ResetStream(seed)
	wantCost := actor.GetLevel()*3 + dprng.Number(1, 55)
	dprng.ResetStream(seed)
	messages := capturePlayerMessages(w)

	w.DoTurn(actor)

	if actor.GetMana() != 1000-wantCost {
		t.Fatalf("mana = %d, want %d after one cost draw", actor.GetMana(), 1000-wantCost)
	}
	if got, want := messages[actor.Name].String(), "You attempt to turn away the unholy presence in this room.\r\nA vampire shivers uncomfortably.\r\nA pale lich shivers uncomfortably.\r\nVampirePC shivers uncomfortably.\r\n"; got != want {
		t.Fatalf("actor output = %q, want %q", got, want)
	}
	if got, want := messages[observer.Name].String(), "Turner's eyes are suddenly filled with blinding white light!\r\nA vampire shivers uncomfortably.\r\nA pale lich shivers uncomfortably.\r\nVampirePC shivers uncomfortably.\r\n"; got != want {
		t.Fatalf("observer output = %q, want %q", got, want)
	}
	if got, want := messages[vampirePC.Name].String(), "Turner's eyes are suddenly filled with blinding white light!\r\nA vampire shivers uncomfortably.\r\nA pale lich shivers uncomfortably.\r\nA disturbing feeling washes over your body.\r\n"; got != want {
		t.Fatalf("player target private output = %q, want %q", got, want)
	}
	for _, vnum := range []int{99001, 99002} {
		if got, want := private[spawned[vnum]], "A disturbing feeling washes over your body.\r\n"; got != want {
			t.Errorf("vnum %d private shiver output = %q, want %q", vnum, got, want)
		}
	}
}

func TestDoTurnLowManaSubtractsQuarterCostAfterRoomActs(t *testing.T) {
	for _, test := range []struct {
		name     string
		mana     func(int) int
		wantMana func(int, int) int
	}{
		{name: "quarter deduction", mana: func(cost int) int { return cost - 1 }, wantMana: func(cost, before int) int { return before - cost/4 }},
		{name: "clamps at zero", mana: func(int) int { return 0 }, wantMana: func(_, _ int) int { return 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, actor, observer := turnDepthWorld(t, false)
			if _, err := w.SpawnMobQuiet(99004, 1001); err != nil {
				t.Fatal(err)
			}
			seed := uint32(73)
			dprng.ResetStream(seed)
			needMana := actor.GetLevel()*3 + dprng.Number(1, 55)
			dprng.ResetStream(seed)
			before := test.mana(needMana)
			actor.SetMana(before)
			messages := capturePlayerMessages(w)

			w.DoTurn(actor)

			if wantMana := test.wantMana(needMana, before); actor.GetMana() != wantMana {
				t.Fatalf("mana after failed summon = %d, want %d", actor.GetMana(), wantMana)
			}
			if got, want := messages[actor.Name].String(), "You attempt to turn away the unholy presence in this room.\r\nYou fail to summon enough energy to dispel the darkness!\r\n"; got != want {
				t.Fatalf("actor output = %q, want %q", got, want)
			}
			if got, want := messages[observer.Name].String(), "Turner's eyes are suddenly filled with blinding white light!\r\n"; got != want {
				t.Fatalf("observer output = %q, want %q", got, want)
			}
			if mobs := w.GetMobsInRoom(1001); len(mobs) != 1 || mobs[0].GetVNum() != 99004 {
				t.Fatalf("low-mana target state = %#v, want unchanged target in origin", mobs)
			}
		})
	}
}

func TestDoTurnDisintegratesEligibleUndeadAtFifteenLevelAdvantage(t *testing.T) {
	w, actor, observer := turnDepthWorld(t, false)
	oldCallbacks := combat.GetCallbacks()
	callbacks := w.WireCombatCallbacks()
	callbacks.Broadcast = w.WoundBroadcast
	combat.SetCallbacks(callbacks)
	t.Cleanup(func() { combat.SetCallbacks(oldCallbacks) })
	for _, vnum := range []int{99006, 99007} {
		if _, err := w.SpawnMobQuiet(vnum, 1001); err != nil {
			t.Fatal(err)
		}
	}
	messages := capturePlayerMessages(w)

	w.DoTurn(actor)

	if mobs := w.GetMobsInRoom(1001); len(mobs) != 0 {
		t.Fatalf("disintegrated mob remains in origin: %#v", mobs)
	}
	if objects := w.GetItemsInRoom(1001); len(objects) != 2 || objects[0].GetVNum() != 18 || objects[1].GetVNum() != 18 {
		t.Fatalf("disintegration dust = %#v, want one standard dust object per target", objects)
	}
	for _, name := range []string{actor.Name, observer.Name} {
		got := messages[name].String()
		for _, bodyName := range []string{"A young lich", "A young vampire"} {
			if !strings.Contains(got, bodyName+" grimaces and then explodes into a cloud of dust!\r\n") {
				t.Errorf("%s did not see %s disintegrate: %q", name, bodyName, got)
			}
		}
		if !strings.Contains(got, "Your blood freezes as you hear a young lich's death cry.\r\n") {
			t.Errorf("%s did not see the raw_kill death cry: %q", name, got)
		}
		if !strings.Contains(got, "Your blood freezes as you hear a young vampire's death cry.\r\n") {
			t.Errorf("%s did not see the vampire raw_kill death cry: %q", name, got)
		}
	}
}

func TestDoTurnFleeUsesNpcDirectMovementAndPreservesDirectionDraw(t *testing.T) {
	w, actor, observer := turnDepthWorld(t, true)
	mob, err := w.SpawnMobQuiet(99004, 1001)
	if err != nil {
		t.Fatal(err)
	}
	var private strings.Builder
	w.MobileMessageSink = func(body *MobInstance, message []byte) {
		if body == mob {
			private.Write(message)
		}
	}
	destination := NewPlayer(3, "Destination", 1002)
	if err := w.AddPlayer(destination); err != nil {
		t.Fatal(err)
	}
	seed := uint32(119)
	dprng.ResetStream(seed)
	_ = dprng.Number(1, 55)
	wantDirection := dirs[dprng.Number(0, len(dirs)-1)]
	wantFrom := dirs[revDir[searchBlock(wantDirection, dirs, false)]]
	dprng.ResetStream(seed)
	messages := capturePlayerMessages(w)

	w.DoTurn(actor)

	if got := w.GetMobsInRoom(1002); len(got) != 1 || got[0].GetVNum() != 99004 {
		t.Fatalf("flee destination occupants = %#v, want the turned undead mob", got)
	}
	if got, want := messages[observer.Name].String(), "Turner's eyes are suddenly filled with blinding white light!\r\nA restless undead shrieks in terror!\r\nA restless undead panics, and attempts to flee!\r\nA restless undead leaves "+wantDirection+".\r\n"; got != want {
		t.Fatalf("origin output = %q, want %q", got, want)
	}
	if got, want := messages[destination.Name].String(), "A restless undead arrives from the "+wantFrom+".\r\n"; got != want {
		t.Fatalf("destination output = %q, want %q", got, want)
	}
	if got, want := private.String(), "You are suddenly terrified!\r\nYou flee head over heels.\r\n"; got != want {
		t.Fatalf("NPC private output = %q, want %q", got, want)
	}
}

func TestDoTurnFleesPlayerByNumericRaceThroughPlayerMovement(t *testing.T) {
	w, actor, observer := turnDepthWorld(t, true)
	vampire := NewPlayer(3, "VampireVictim", 1001)
	vampire.Race = 8
	vampire.SetLevel(17)
	vampire.SetMove(100)
	vampire.SetExp(5000)
	if err := w.AddPlayer(vampire); err != nil {
		t.Fatal(err)
	}
	destination := NewPlayer(4, "Destination", 1002)
	if err := w.AddPlayer(destination); err != nil {
		t.Fatal(err)
	}
	w.MovementLook = func(*Player) {}
	seed := uint32(193)
	dprng.ResetStream(seed)
	_ = dprng.Number(1, 55)
	wantDirection := dirs[dprng.Number(0, len(dirs)-1)]
	dprng.ResetStream(seed)
	messages := capturePlayerMessages(w)

	w.DoTurn(actor)

	if vampire.GetRoom() != 1002 {
		t.Fatalf("turned player room = %d, want destination 1002", vampire.GetRoom())
	}
	// C computes 500*(level/2.6) in floating point, then truncates.
	wantExp := 5000 - int(500*(float64(vampire.GetLevel())/2.6))
	if got := vampire.GetExp(); got != wantExp {
		t.Fatalf("turned player experience = %d, want %d", got, wantExp)
	}
	if got, want := messages[vampire.Name].String(), "Turner's eyes are suddenly filled with blinding white light!\r\nYou are suddenly terrified!\r\nYou flee head over heels.\r\n"; got != want {
		t.Fatalf("turned player output = %q, want %q", got, want)
	}
	if got, want := messages[observer.Name].String(), "Turner's eyes are suddenly filled with blinding white light!\r\nVampireVictim shrieks in terror!\r\nVampireVictim panics, and attempts to flee!\r\nVampireVictim leaves "+wantDirection+".\r\n"; got != want {
		t.Fatalf("origin output = %q, want %q", got, want)
	}
}

func TestDoFleeNpcWaitGateDoesNotDrawRandomness(t *testing.T) {
	w, _, _ := turnDepthWorld(t, true)
	mob, err := w.SpawnMobQuiet(99004, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetWaitState(1)
	var private strings.Builder
	w.MobileMessageSink = func(body *MobInstance, message []byte) {
		if body == mob {
			private.Write(message)
		}
	}
	seed := uint32(271)
	dprng.ResetStream(seed)
	wantNext := dprng.Number(0, len(dirs)-1)
	dprng.ResetStream(seed)

	w.DoFlee(mob)

	gotNext := dprng.Number(0, len(dirs)-1)
	if gotNext != wantNext {
		t.Fatalf("next random direction = %d, want %d after NPC wait gate", gotNext, wantNext)
	}
	if got, want := private.String(), "You can't flee yet!\r\n"; got != want {
		t.Fatalf("NPC wait-gate output = %q, want %q", got, want)
	}
	if mob.GetRoom() != 1001 {
		t.Fatalf("NPC moved despite wait gate: room %d", mob.GetRoom())
	}
}

func TestDoTurnEarlyGatesDoNotDrawRandomness(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*Player)
		want  string
	}{
		{name: "unlearned", setup: func(p *Player) { p.SetSkill(SkillTurn, 0) }, want: "Huh?!?\r\n"},
		{name: "not holy", setup: func(p *Player) { p.SetSkill(SkillTurn, 100) }, want: "You are not holy enough to turn away the Undead!\r\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, actor, _ := turnDepthWorld(t, false)
			actor.SetAlignment(0)
			test.setup(actor)
			messages := capturePlayerMessages(w)
			seed := uint32(163)
			dprng.ResetStream(seed)
			wantNext := dprng.Number(1, 55)
			dprng.ResetStream(seed)
			w.DoTurn(actor)
			gotNext := dprng.Number(1, 55)
			if gotNext != wantNext {
				t.Fatalf("next random draw = %d, want %d after early return", gotNext, wantNext)
			}
			if got := messages[actor.Name].String(); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}
