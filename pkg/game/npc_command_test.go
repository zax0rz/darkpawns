package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func newNPCCommandWorld(t *testing.T) (*World, *Player, *MobInstance, *strings.Builder) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1000, Name: "The Void", Zone: 1}, // real room 0
			{VNum: 1001, Name: "Temple", Zone: 1},
		},
		Mobs: []parser.Mob{{
			VNum: 2001, Keywords: "healer", ShortDesc: "the healer",
			Level: 20, Alignment: -350, Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	player := NewPlayer(1, "Zach", 1001)
	player.Level = 10
	player.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetPosition(combat.PosStanding)
	out := &strings.Builder{}
	w.MessageSink = func(name string, message []byte) {
		if name == "Zach" {
			out.Write(message)
		}
	}
	return w, player, mob, out
}

// command_interpreter draws number(0,3) for AFF_HIDE before anything else,
// for every line a mobile runs, known command or not (R3).
func TestNPCCommandDrawsHideRollFirst(t *testing.T) {
	for _, line := range []string{"say hello", "xyzzy", ""} {
		w, _, mob, _ := newNPCCommandWorld(t)
		dprng.ResetStream(4242)
		dprng.Number(0, 3)
		want := dprng.Number(0, 100)

		dprng.ResetStream(4242)
		w.NPCCommand(mob, line)
		if got := dprng.Number(0, 100); got != want {
			t.Errorf("NPCCommand(%q) consumed the wrong number of draws: next draw %d, want %d", line, got, want)
		}
	}
}

// do_say picks its verb from the line's last character.
func TestNPCCommandSay(t *testing.T) {
	cases := map[string]string{
		"say I cannot let you pass.": "states, 'I cannot let you pass.'",
		"say Welcome!":               "exclaims, 'Welcome!'",
		"' Who goes there?":          "asks, 'Who goes there?'",
		"say hello":                  "says, 'hello'",
	}
	for line, want := range cases {
		w, _, mob, out := newNPCCommandWorld(t)
		w.NPCCommand(mob, line)
		if !strings.Contains(out.String(), "healer "+want) {
			t.Errorf("NPCCommand(%q) showed %q, want it to contain %q", line, out.String(), want)
		}
	}
}

func TestNPCCommandDirectRunsBodyWithoutInterpreterPreamble(t *testing.T) {
	w, _, mob, out := newNPCCommandWorld(t)
	mob.SetPosition(combat.PosSleeping)
	mob.SetAffected(affHide)
	dprng.ResetStream(8417)
	wantNext := dprng.Number(0, 100)
	dprng.ResetStream(8417)

	if got := w.NPCCommandDirect(mob, "say", "   hush now!"); got != CharacterCommandHandled {
		t.Fatalf("direct say result = %v, want handled", got)
	}
	if got := dprng.Number(0, 100); got != wantNext {
		t.Fatalf("direct command consumed an interpreter draw: next=%d want=%d", got, wantNext)
	}
	if mob.IsAffected(affHide) != true {
		t.Fatal("direct command cleared AFF_HIDE; C's direct ACMD call skips command_interpreter")
	}
	if got, want := out.String(), "The healer exclaims, 'hush now!'\r\n"; got != want {
		t.Fatalf("direct say bytes = %q, want %q", got, want)
	}
}

func TestNPCCommandDirectPassesExactKnownBodyAndArgument(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	var gotCommand, gotArgument string
	w.CharacterCommandDirectExecFunc = func(body combat.Combatant, commandName, argument string) CharacterCommandResult {
		if body != mob {
			t.Fatalf("direct command body = %p, want mobile %p", body, mob)
		}
		gotCommand, gotArgument = commandName, argument
		return CharacterCommandHandled
	}
	const argument = "  2.gate iron north  "
	if got := w.NPCCommandDirect(mob, "OPEN", argument); got != CharacterCommandHandled {
		t.Fatalf("direct command result = %v, want handled", got)
	}
	if gotCommand != "open" || gotArgument != argument {
		t.Fatalf("direct dispatch = (%q, %q), want (%q, %q)", gotCommand, gotArgument, "open", argument)
	}
}

func TestTellFromMobUsesTellDeliveryGates(t *testing.T) {
	w, target, mob, out := newNPCCommandWorld(t)
	target.SetPlrFlag(PrfNotell, true)

	tellFromMob(w, mob, target, "private message")

	if got := out.String(); got != "" {
		t.Fatalf("do_tell bypassed the target's NOTELL gate: %q", got)
	}
}

func TestTellFromMobUsesDirectTellArgument(t *testing.T) {
	w, player, transcript := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
	var commandName, commandArg string
	w.CharacterCommandDirectExecFunc = func(_ combat.Combatant, command, argument string) CharacterCommandResult {
		commandName, commandArg = command, argument
		return CharacterCommandUnsupported
	}

	tellFromMob(w, mob, player, "The C buffer payload.")
	if commandName != "tell" || commandArg != player.GetName()+" The C buffer payload." {
		t.Fatalf("direct command = %q %q, want tell %q", commandName, commandArg, player.GetName()+" The C buffer payload.")
	}
	if got := transcript(); !strings.Contains(got, "tells you, 'The C buffer payload.'") {
		t.Fatalf("tell output = %q, want the command body output", got)
	}
}

// A command not ported for mobiles does nothing visible.
func TestNPCCommandUnportedIsSilent(t *testing.T) {
	w, _, mob, out := newNPCCommandWorld(t)
	w.NPCCommand(mob, "cast 'heal' zach")
	if out.Len() != 0 {
		t.Fatalf("unported command showed %q", out.String())
	}
}

// char_to_table's fields for a mobile and a player, through the bridge.
func TestWorldBridgeCharFields(t *testing.T) {
	w, player, mob, _ := newNPCCommandWorld(t)
	b := NewWorldScriptableAdapter(w)
	mobRef := scripting.CharRef{NPC: true, ID: mob.GetID()}
	f, ok := b.CharFields(mobRef)
	if !ok {
		t.Fatal("mobile not resolved")
	}
	if f.VNum != 2001 || !f.NPC || !f.Evil || f.Name != "the healer" || f.Alias != "healer" {
		t.Fatalf("mobile fields %+v", f)
	}
	pf, ok := b.CharFields(scripting.CharRef{ID: player.ID})
	if !ok || pf.NPC || pf.Name != "Zach" || pf.Level != 10 {
		t.Fatalf("player fields %+v ok=%v", pf, ok)
	}
	if !b.InRoomAboveZero(mobRef) {
		t.Fatal("mobile in real room 1 should pass run_script's in_room > 0 check")
	}
	mob.SetRoom(1000)
	if b.InRoomAboveZero(mobRef) {
		t.Fatal("real room 0 fails run_script's in_room > 0 check")
	}
	if _, ok := b.CharFields(scripting.CharRef{NPC: true, ID: 999999}); ok {
		t.Fatal("unknown mobile resolved")
	}
}

// The pulse triggers' people scans (mobact.c:148-192).
func TestRoomHasOtherOccupant(t *testing.T) {
	w, player, mob, _ := newNPCCommandWorld(t)
	if !w.roomHasOtherOccupant(mob, true) || !w.roomHasOtherOccupant(mob, false) {
		t.Fatal("a player in the room should satisfy both scans")
	}
	player.SetRoom(0)
	if w.roomHasOtherOccupant(mob, true) || w.roomHasOtherOccupant(mob, false) {
		t.Fatal("an empty room should satisfy neither scan")
	}
	if _, err := w.SpawnMob(2001, 1001); err != nil {
		t.Fatal(err)
	}
	if w.roomHasOtherOccupant(mob, true) || !w.roomHasOtherOccupant(mob, false) {
		t.Fatal("another mobile satisfies onpulse_all's scan but not the player-only one")
	}
}

// table_to_char writes a mobile's timer to GET_MOB_WAIT, the field combat
// reads (the port once kept a separate copy only scripts saw).
func TestWorldBridgeTimerIsMobWait(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	b := NewWorldScriptableAdapter(w)
	ref := scripting.CharRef{NPC: true, ID: mob.GetID()}
	b.ApplyChar(ref, scripting.CharWrite{Level: mob.GetLevel(), HP: mob.GetHP(), Pos: mob.GetPosition(), Timer: 3, Align: mob.GetAlignment()})
	if got := mob.GetWaitState(); got != 3 {
		t.Fatalf("GET_MOB_WAIT after a script's timer write = %d, want 3", got)
	}
	if f, _ := b.CharFields(ref); f.Timer != 3 {
		t.Fatalf("me.timer reads %d, want 3", f.Timer)
	}
}
