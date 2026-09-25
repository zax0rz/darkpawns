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
