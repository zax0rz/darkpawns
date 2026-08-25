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
