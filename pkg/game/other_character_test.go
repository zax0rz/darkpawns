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
			"unrelated source text",
		))

		w.doVisible(player, nil, "visible", "ignored")
		if got, want := out.String(), "You cannot become visible until your zai ends!\r\n"; got != want {
			t.Fatalf("zai output = %q, want %q", got, want)
		}
	})

	t.Run("other affect text containing zai does not block", func(t *testing.T) {
		w := &World{}
		var out strings.Builder
		w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
		player := NewPlayer(1, "Visible", 8162)
		player.worldRef = w
		player.AddAffect(engine.NewAffectDirect(
			skillNumKkZhen,
			engine.ApplyNone,
			3,
			0,
			engine.AFFKujiKiri,
			"unrelated zai-like source text",
		))

		w.doVisible(player, nil, "visible", "ignored")
		if got, want := out.String(), "You are already visible.\r\n"; got != want {
			t.Fatalf("other affect output = %q, want %q", got, want)
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

// TestDoVisibleStripsInvisSpellAffect is the regression for the player report
// (BRIEF-2026-10-24 Fix 2): a mortal under the invisibility spell ran
// `visible`, was told "You fade into view.", yet remained invisible to every
// shopkeeper for the spell's whole duration. C do_visible routes through
// appear(ch), which removes the SPELL_INVISIBLE affect record (src/fight.c:
// 108-121); the port cleared only the raw AFF bit, and IsAffected reads both
// sources (player_social.go isAffectedLocked), so the surviving record kept
// reporting the player invisible.
func TestDoVisibleStripsInvisSpellAffect(t *testing.T) {
	w := &World{}
	delivered := map[string]string{}
	w.MessageSink = func(playerName string, msg []byte) {
		delivered[playerName] += string(msg)
	}
	player := NewPlayer(1, "Egon", 8162)
	player.worldRef = w
	// The exact spell record the invisibility spell applies
	// (pkg/spells/affect_spells.go SPELL_INVISIBLE branch).
	player.AddAffect(engine.NewAffectDirect(
		spellInvisible, engine.ApplyNone, 12, 0, engine.AFFInvisible, "invisibility"))
	if !player.IsAffected(affInvisible) {
		t.Fatal("precondition: spell record must make IsAffected(affInvisible) true")
	}

	// A shopkeeper-style observer: a mortal without detect-invisible.
	keeper := NewPlayer(2, "Keeper", 8162)
	keeper.worldRef = w
	if canSee(keeper, player) {
		t.Fatal("precondition: observer must not see the invisible player")
	}

	w.doVisible(player, nil, "visible", "ignored")

	if got, want := delivered["Egon"], "You fade into view.\r\n"; got != want {
		t.Fatalf("player output = %q, want %q", got, want)
	}
	if player.IsAffected(affInvisible) {
		t.Fatal("visible left the player invisible: AFF_INVISIBLE still reports set")
	}
	if player.HasSpellAffect(spellInvisible) {
		t.Fatal("visible left the SPELL_INVISIBLE affect record in place")
	}
	if !canSee(keeper, player) {
		t.Fatal("shopkeeper-style observer still cannot see the player after visible")
	}
}

// TestDoVisibleInvisAndSneakTogether covers C do_visible's two sequential if
// arms (src/act.other.c:575-588): a character under both the invisibility
// spell and sneak gets both removals and both messages in C's order, and the
// room sees appear()'s "$n slowly fades into existence." line.
func TestDoVisibleInvisAndSneakTogether(t *testing.T) {
	w := &World{}
	delivered := map[string]string{}
	w.MessageSink = func(playerName string, msg []byte) {
		delivered[playerName] += string(msg)
	}
	player := NewPlayer(1, "Egon", 8162)
	player.worldRef = w
	player.AddAffect(engine.NewAffectDirect(
		spellInvisible, engine.ApplyNone, 12, 0, engine.AFFInvisible, "invisibility"))
	player.SetAffect(affSneak, true)
	player.ActiveAffects = append(player.ActiveAffects,
		engine.NewAffectDirect(skillNumSneak, engine.ApplyNone, 6, 0, engine.AFFSneak, SkillSneak),
		engine.NewAffectDirect(skillNumStealth, engine.ApplyNone, 6, 0, engine.AFFSneak, SkillStealth),
	)
	bystander := NewPlayer(2, "Bystander", 8162)
	bystander.worldRef = w
	// A bare &World{} has no initialized player index (AddPlayer needs the
	// full constructor); seat the bystander directly so Act's TO_ROOM
	// audience scan finds them, as AddPlayer would on a constructed world.
	w.mu.Lock()
	w.players = map[string]*Player{"Bystander": bystander}
	w.mu.Unlock()

	w.doVisible(player, nil, "visible", "ignored")

	if got, want := delivered["Egon"], "You fade into view.\r\nYou stop sneaking.\r\n"; got != want {
		t.Fatalf("player output = %q, want %q", got, want)
	}
	if got, want := delivered["Bystander"], "Egon slowly fades into existence.\r\n"; got != want {
		t.Fatalf("room output = %q, want %q", got, want)
	}
	if player.IsAffected(affInvisible) || player.IsAffected(affSneak) {
		t.Fatal("visible left invis or sneak reported set")
	}
	if player.HasSpellAffect(spellInvisible) || player.HasSpellAffect(skillNumSneak) {
		t.Fatal("visible left the invis or sneak spell records in place")
	}
}
