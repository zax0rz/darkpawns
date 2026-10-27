package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRoomMessageUsesActRecipientGate(t *testing.T) {
	w, _, sleeping, _, output := newChannelWorld(t)
	awake := channelPlayer(t, w, 4, "Awake", 1001)
	writer := channelPlayer(t, w, 5, "Writer", 1001)
	sleeping.SetPosition(combat.PosSleeping)
	writer.SetPlrFlag(PlrWriting, true)

	w.roomMessage(1001, "A room act.")

	if got := channelOutput(output, sleeping.Name); got != "" {
		t.Fatalf("sleeping observer received room act %q", got)
	}
	if got := channelOutput(output, writer.Name); got != "" {
		t.Fatalf("writing observer received room act %q", got)
	}
	if got, want := channelOutput(output, awake.Name), "A room act.\r\n"; got != want {
		t.Fatalf("awake observer output = %q, want %q", got, want)
	}
}

func TestSendToRoomUsesCRecipientGate(t *testing.T) {
	w, _, sleeping, _, output := newChannelWorld(t)
	awake := channelPlayer(t, w, 4, "Awake", 1001)
	writer := channelPlayer(t, w, 5, "Writer", 1001)
	sleeping.SetPosition(combat.PosSleeping)
	writer.SetPlrFlag(PlrWriting, true)

	w.SendToRoom(1001, "A raw room line.\r\n")

	if got := channelOutput(output, sleeping.Name); got != "" {
		t.Fatalf("sleeping observer received send_to_room line %q", got)
	}
	if got, want := channelOutput(output, writer.Name), "A raw room line.\r\n"; got != want {
		t.Fatalf("writing observer output = %q, want %q (C send_to_room does not use SENDOK)", got, want)
	}
	if got, want := channelOutput(output, awake.Name), "A raw room line.\r\n"; got != want {
		t.Fatalf("awake observer output = %q, want %q", got, want)
	}
}

func TestAlienElevatorDirectSendReachesSleepingOccupant(t *testing.T) {
	w, _, _, _, output := newChannelWorld(t)
	const roomA = 19551
	const roomB = 19599
	actor := channelPlayer(t, w, 4, "Elevatoractor", roomA)
	sleeper := channelPlayer(t, w, 5, "Elevatorsleeper", roomA)
	awake := channelPlayer(t, w, 6, "Elevatorawake", roomB)
	sleeper.SetPosition(combat.PosSleeping)

	if !specAlienElevator(w, actor, nil, "close", "door") {
		t.Fatal("alien elevator did not handle close door")
	}
	if got, want := channelOutput(output, sleeper.Name), "The room starts to move!\r\n"; got != want {
		t.Fatalf("sleeping occupant output = %q, want direct C send_to_char bytes %q", got, want)
	}
	if got, want := channelOutput(output, awake.Name), "The room starts to move!\r\n"; got != want {
		t.Fatalf("awake occupant output = %q, want %q", got, want)
	}
	if got := sleeper.GetRoom(); got != roomB {
		t.Fatalf("sleeping occupant room = %d, want %d", got, roomB)
	}
}

func TestSpellCreatedObjectRoomActUsesActRecipientGate(t *testing.T) {
	w, caster, sleeping, _, output := newChannelWorld(t)
	awake := channelPlayer(t, w, 4, "Awake", 1001)
	writer := channelPlayer(t, w, 5, "Writer", 1001)
	sleeping.SetPosition(combat.PosSleeping)
	writer.SetPlrFlag(PlrWriting, true)
	obj := NewObjectInstance(&parser.Obj{Keywords: "mushroom", ShortDesc: "a Magic Mushroom"}, -1)

	if !w.SpellCreatedObjectAct(caster, obj) {
		t.Fatal("SpellCreatedObjectAct did not handle live player/object")
	}
	if got := channelOutput(output, sleeping.Name); got != "" {
		t.Fatalf("sleeping observer received Create Food room act %q", got)
	}
	if got := channelOutput(output, writer.Name); got != "" {
		t.Fatalf("writing observer received Create Food room act %q", got)
	}
	if got, want := channelOutput(output, awake.Name), "Actor creates a Magic Mushroom.\r\n"; got != want {
		t.Fatalf("awake observer output = %q, want %q", got, want)
	}
}

func TestSpellCreatedObjectRoomActUsesRecipientObjectVisibility(t *testing.T) {
	w, caster, _, _, output := newChannelWorld(t)
	blind := channelPlayer(t, w, 4, "Blind", 1001)
	blind.SetAffect(affBlind, true)
	obj := NewObjectInstance(&parser.Obj{Keywords: "mushroom", ShortDesc: "a Magic Mushroom"}, -1)

	w.SpellCreatedObjectAct(caster, obj)

	if got, want := channelOutput(output, blind.Name), "Someone creates something.\r\n"; got != want {
		t.Fatalf("blind observer output = %q, want %q", got, want)
	}
}

func TestRoomMessageRevertControl(t *testing.T) {
	w, _, sleeping, _, output := newChannelWorld(t)
	sleeping.SetPosition(combat.PosSleeping)
	w.roomMessage(1001, "revert control")
	if got := channelOutput(output, sleeping.Name); strings.Contains(got, "revert control") {
		t.Fatalf("sleeping observer received message after gate was reverted: %q", got)
	}
}
