package game

import (
	"strings"
	"testing"
)

// TestLookGates proves do_look's early gates (act.item.c:1104-1113): a
// sub-sleeping position and blindness each short-circuit the look.
func TestLookGates(t *testing.T) {
	t.Run("position", func(t *testing.T) {
		w, ch, _ := newDonateTestWorld(t)
		ch.SetPosition(posSleeping - 1)
		result := w.DoLook(ch, "look", "")
		if len(result.Messages) == 0 || !strings.Contains(result.Messages[0].Format, "You can't see anything but stars!") {
			t.Fatalf("position gate: got %+v", result.Messages)
		}
	})
	t.Run("blind", func(t *testing.T) {
		w, ch, _ := newDonateTestWorld(t)
		ch.SetAffect(affBlind, true)
		result := w.DoLook(ch, "look", "")
		if len(result.Messages) == 0 || !strings.Contains(result.Messages[0].Format, "you're blind!") {
			t.Fatalf("blind gate: got %+v", result.Messages)
		}
	})
}

func TestDoExitsBlindnessGate(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	ch.SetAffect(affBlind, true)

	result := w.DoExits(ch)
	if len(result.Messages) != 1 || result.Messages[0].Format != "You can't see a damned thing, you're blind!" {
		t.Fatalf("blind exits gate: got %+v", result.Messages)
	}
}

func TestDoExitsInfravisionSeesDarkDestination(t *testing.T) {
	w, ch := newMovementTestWorld(t)
	destination, ok := w.GetRoom(1002)
	if !ok {
		t.Fatal("destination room missing")
	}
	destination.Flags = []string{"1"} // C ROOM_DARK
	ch.SetAffect(affInfravision, true)

	result := w.DoExits(ch)
	if len(result.Messages) != 2 || result.Messages[0].Format != "Obvious exits:" || result.Messages[1].Format != "North - Room South" {
		t.Fatalf("infravision exits rendering: got %+v", result.Messages)
	}
}

func TestImmortalWithoutHolyLightCannotSeeDarkRoom(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	ch.SetLevel(LVL_IMMORT)
	ch.SetHolyLight(false)
	dark := w.GetRoomInWorld(ch.GetRoom())
	if dark == nil {
		t.Fatal("current room missing")
	}
	dark.Flags = []string{"1", "0", "0", "0"} // C ROOM_DARK

	result := w.DoLook(ch, "look", "")
	if len(result.Messages) == 0 || !strings.Contains(result.Messages[0].Format, "Darkness") || !strings.Contains(result.Messages[0].Format, "too dark") {
		t.Fatalf("immortal without holy light should not see dark room: got %+v", result.Messages)
	}
}
