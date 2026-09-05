package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

func TestDoVisibleRemovesSneakAndStealthAffects(t *testing.T) {
	w := &World{}
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	player := NewPlayer(1, "Visible", 8162)
	player.worldRef = w
	player.SetAffect(affSneak, true)
	player.ActiveAffects = append(player.ActiveAffects,
		engine.NewAffectDirect(skillNumSneak, engine.ApplyNone, 6, 0, engine.AFFSneak, SkillSneak),
		engine.NewAffectDirect(skillNumStealth, engine.ApplyNone, 6, 0, engine.AFFSneak, SkillStealth),
	)

	w.doVisible(player, nil, "visible", "ignored")

	if got, want := out.String(), "You stop sneaking.\r\n"; got != want {
		t.Fatalf("visible output = %q, want %q", got, want)
	}
	if player.IsAffected(affSneak) {
		t.Fatal("visible left AFF_SNEAK set")
	}
	if len(player.ActiveAffects) != 0 {
		t.Fatalf("visible left %d sneak affects, want none", len(player.ActiveAffects))
	}
}

func TestDoVisiblePreservesCEntryGateOrder(t *testing.T) {
	t.Run("zai gate", func(t *testing.T) {
		w := &World{}
		var out strings.Builder
		w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
		player := NewPlayer(1, "Visible", 8162)
		player.worldRef = w
		player.AddAffect(engine.NewAffectDirect(
			skillNumKkZai,
			engine.ApplyNone,
			3,
			0,
			engine.AFFKujiKiri,
			"kuji-kiri zai",
		))

		w.doVisible(player, nil, "visible", "ignored")
		if got, want := out.String(), "You cannot become visible until your zai ends!\r\n"; got != want {
			t.Fatalf("zai output = %q, want %q", got, want)
		}
	})

	t.Run("immortal path", func(t *testing.T) {
		w := &World{}
		var out strings.Builder
		w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
		player := NewPlayer(1, "Visible", 8162)
		player.worldRef = w
		player.SetLevel(LVL_IMMORT)

		w.doVisible(player, nil, "visible", "ignored")
		if got, want := out.String(), "You are already fully visible.\r\n"; got != want {
			t.Fatalf("immortal output = %q, want %q", got, want)
		}
	})

	t.Run("invisible path", func(t *testing.T) {
		w := &World{}
		var out strings.Builder
		w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
		player := NewPlayer(1, "Visible", 8162)
		player.worldRef = w
		player.SetAffect(affInvisible, true)

		w.doVisible(player, nil, "visible", "ignored")
		if got, want := out.String(), "You fade into view.\r\n"; got != want {
			t.Fatalf("invisible output = %q, want %q", got, want)
		}
		if player.IsAffected(affInvisible) {
			t.Fatal("invisible path left AFF_INVISIBLE set")
		}
	})
}
