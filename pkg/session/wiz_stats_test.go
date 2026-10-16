package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestStatPlayer_DefaultPositionStanding is the DP-1332 regression: C's
// do_stat_character prints "Default position: Standing" for all characters
// (using mob_specials.default_pos, which clear_char initializes to POS_STANDING
// even for players). The port was using the player's current position instead.
func TestStatPlayer_DefaultPositionStanding(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.SetPosition(combat.PosResting)

	s.sendStatPlayerReport(s.player, 1001, true, 0)

	var texts []string
	for {
		select {
		case msg := <-s.send:
			if IsInputMarkFrame(msg) {
				continue
			}
			texts = append(texts, string(msg))
		default:
			goto drain
		}
	}
drain:

	found := false
	for _, raw := range texts {
		if strings.Contains(raw, "Default position:") {
			found = true
			if strings.Contains(raw, "Default position: Resting") {
				t.Errorf("stat used current position for Default position; want Standing (C's clear_char default).\n  raw=%s", raw)
			}
			if !strings.Contains(raw, "Default position: Standing") {
				t.Errorf("Default position line missing 'Standing':\n  raw=%s", raw)
			}
			break
		}
	}
	if !found {
		t.Fatal("stat output did not contain a 'Default position:' line")
	}
}

func TestStatPlayerPoofFieldsMatchImmortalGate(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.SetLevel(LVL_IMMORT)
	s.player.PoofIn = "a swirl of arrival"
	s.player.PoofOut = "a cloud of departure"

	s.sendStatPlayerReport(s.player, 1001, true, 0)
	out := drainSessionOutput(t, s)
	if want := "Poofin:a swirl of arrival\r\nPoofout:a cloud of departure\r\n"; !strings.Contains(out, want) {
		t.Fatalf("immortal stat poof fields missing: got %q, want substring %q", out, want)
	}

	s.player.PoofIn, s.player.PoofOut = "", ""
	s.sendStatPlayerReport(s.player, 1001, true, 0)
	out = drainSessionOutput(t, s)
	if want := "Poofin:None\r\nPoofout:None\r\n"; !strings.Contains(out, want) {
		t.Fatalf("immortal stat poof defaults missing: got %q, want substring %q", out, want)
	}

	s.player.SetLevel(LVL_IMMORT - 1)
	s.sendStatPlayerReport(s.player, 1001, true, 0)
	out = drainSessionOutput(t, s)
	if strings.Contains(out, "Poofin:") || strings.Contains(out, "Poofout:") {
		t.Fatalf("mortal stat report included immortal poof fields: %q", out)
	}
}

func TestStatMobileEquipmentPoints(t *testing.T) {
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 300, ShortDesc: "a guard", Keywords: "guard", AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11, Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 7}}}, Objs: []parser.Obj{{VNum: 200, TypeFlag: game.ITEM_ARMOR, Values: [4]int{10}, Affects: []parser.ObjAffect{{Location: 18, Modifier: 4}, {Location: 19, Modifier: 3}, {Location: 20, Modifier: -2}}}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)
	mob, err := w.SpawnMob(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := w.SpawnObject(200, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipMobileObject(mob, obj, 5); err != nil {
		t.Fatal(err)
	}
	s.sendStatMob(mob)
	var text strings.Builder
	for len(s.send) > 0 {
		text.WriteString(readSendText(t, s))
	}
	if !strings.Contains(text.String(), "AC: [70/10], Hitroll: [ 4], Damroll: [10], Saving throws: [-2/0/0/0/0]") {
		t.Fatalf("stat did not read effective mobile equipment points: %s", text.String())
	}
}

func TestStatMobNoiseColorAndHuntingMatchC(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room One"}},
		Mobs: []parser.Mob{{
			VNum: 300, ShortDesc: "a noisy guard", Keywords: "guard", Noise: "The guard rattles.",
			AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11,
			Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 1},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)
	mob, err := w.SpawnMob(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetHunting("Huntvictim")

	s.player.SetPlrFlag(game.PrfColor2, true)
	s.sendStatMob(mob)
	out := drainSessionOutput(t, s)
	if want := "Noise: \x1b[33mThe guard rattles.\x1b[0m\r\n"; !strings.Contains(out, want) {
		t.Fatalf("colored stat noise line = %q, want substring %q", out, want)
	}
	if want := "Hunting: Huntvictim\r\n"; !strings.Contains(out, want) {
		t.Fatalf("stat hunting line = %q, want substring %q", out, want)
	}

	s.player.SetPlrFlag(game.PrfColor2, false)
	mob.SetHunting("")
	s.sendStatMob(mob)
	out = drainSessionOutput(t, s)
	if want := "Noise: The guard rattles.\r\n"; !strings.Contains(out, want) {
		t.Fatalf("uncolored stat noise line = %q, want substring %q", out, want)
	}
	if strings.Contains(out, "\x1b[33m") || strings.Contains(out, "\x1b[0m") || strings.Contains(out, "Hunting:") {
		t.Fatalf("stat report invented color or empty hunting output: %q", out)
	}
}

func TestStatMobUsesPrototypeBareHandAttackType(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room One"}},
		Mobs: []parser.Mob{{
			VNum: 301, ShortDesc: "a whip guard", Keywords: "guard", BareHandAttack: 2,
			AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11,
			Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 1},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)
	mob, err := w.SpawnMob(301, 1001)
	if err != nil {
		t.Fatal(err)
	}

	s.sendStatMob(mob)
	out := drainSessionOutput(t, s)
	if want := "Pos: Standing, Fighting: Nobody, Attack type: whip\r\n"; !strings.Contains(out, want) {
		t.Fatalf("stat attack type line = %q, want substring %q", out, want)
	}
}

func drainSessionOutput(t *testing.T, s *Session) string {
	t.Helper()
	var text strings.Builder
	for len(s.send) > 0 {
		text.WriteString(readSendText(t, s))
	}
	return text.String()
}

func TestCmdStatFallbackChainMatchesC(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room One"},
			{VNum: 1002, Name: "Room Two"},
		},
		Mobs: []parser.Mob{
			{VNum: 300, ShortDesc: "the remote orc", Keywords: "orc", AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11, Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 1}},
		},
		Objs: []parser.Obj{
			{VNum: 200, ShortDesc: "a worn ring", Keywords: "ring", TypeFlag: game.ITEM_WORN},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)
	s.player.SetLevel(LVL_IMPL)

	// Spawn a mobile in room 1002 while Alice is in room 1001.
	if _, err := w.SpawnMob(300, 1002); err != nil {
		t.Fatal(err)
	}
	if err := cmdStat(s, []string{"orc"}); err != nil {
		t.Fatal(err)
	}
	out := drainSessionOutput(t, s)
	if !strings.Contains(out, "MOB 'the remote orc'") || !strings.Contains(out, "In room [ 1002]") {
		t.Fatalf("stat <mob> did not resolve remote world mobile via get_char_vis fallback:\n%s", out)
	}

	// Equip an object on Alice and verify step 1 (get_object_in_equip_vis) resolves it.
	ring, err := w.SpawnObject(200, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.player.Equipment.SetSlot(game.SlotFingerR, ring); err != nil {
		t.Fatal(err)
	}
	if err := cmdStat(s, []string{"ring"}); err != nil {
		t.Fatal(err)
	}
	out = drainSessionOutput(t, s)
	if !strings.Contains(out, "Name: 'a worn ring'") {
		t.Fatalf("stat <equip> did not resolve equipped item via get_object_in_equip_vis:\n%s", out)
	}
}

func TestCmdWhereSearchesMobsAndObjectsMatchC(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room One"},
			{VNum: 1002, Name: "Room Two"},
		},
		Mobs: []parser.Mob{
			{VNum: 300, ShortDesc: "a cityguard", Keywords: "cityguard guard", AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11},
		},
		Objs: []parser.Obj{
			{VNum: 200, ShortDesc: "a loaf of bread", Keywords: "bread loaf", TypeFlag: game.ITEM_FOOD},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)
	s.player.SetLevel(LVL_IMPL)

	if _, err := w.SpawnMob(300, 1002); err != nil {
		t.Fatal(err)
	}
	if _, err := w.SpawnObject(200, 1002); err != nil {
		t.Fatal(err)
	}

	if err := cmdWhere(s, []string{"cityguard"}); err != nil {
		t.Fatal(err)
	}
	out := drainSessionOutput(t, s)
	wantMob := "M  1. a cityguard               - [ 1002]Room Two\r\n"
	if out != wantMob {
		t.Fatalf("where <mob> mismatch:\ngot:  %q\nwant: %q", out, wantMob)
	}

	if err := cmdWhere(s, []string{"bread"}); err != nil {
		t.Fatal(err)
	}
	out = drainSessionOutput(t, s)
	wantObj := "O  1. a loaf of bread           - [ 1002] Room Two\r\n"
	if out != wantObj {
		t.Fatalf("where <obj> mismatch:\ngot:  %q\nwant: %q", out, wantObj)
	}
}

func TestStatCharacterAffectFlagsAndSpellLinesMatchC(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room One"}},
		Mobs:  []parser.Mob{{VNum: 300, ShortDesc: "a guard", Keywords: "guard", AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11, Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	s := makeTestSession(t, manager, "Alice", 1001, true)

	// Apply bless (+2 HITROLL and -1 SAVING_SPELL, AFF_NOTHING) and sanctuary (AFF_SANCTUARY) to Alice.
	s.player.AddAffect(&engine.Affect{
		Origin:    engine.OriginSpell,
		SpellID:   spells.SpellBless,
		Source:    "bless",
		Duration:  5,
		Location:  engine.ApplyHitroll,
		Magnitude: 2,
		Flags:     engine.AFFNothing,
	})
	s.player.AddAffect(&engine.Affect{
		Origin:    engine.OriginSpell,
		SpellID:   spells.SpellBless,
		Source:    "bless",
		Duration:  5,
		Location:  engine.ApplySavingSpell,
		Magnitude: -1,
		Flags:     engine.AFFNothing,
	})
	s.player.AddAffect(&engine.Affect{
		Origin:   engine.OriginSpell,
		SpellID:  spells.SpellSanctuary,
		Source:   "sanctuary",
		Duration: 3,
		Flags:    engine.AFFSanctuary,
	})

	s.sendStatPlayerReport(s.player, 1001, true, 0)
	out := drainSessionOutput(t, s)
	if !strings.Contains(out, "AFF: SANCT NOTHING\r\n") {
		t.Fatalf("player stat AFF: line missing active spell bits SANCT NOTHING:\n%s", out)
	}
	wantSPL := "SPL: (  4hr) sanctuary             sets SANCT\r\n" +
		"SPL: (  6hr) bless                 -1 to SAVING_SPELL, sets NOTHING\r\n" +
		"SPL: (  6hr) bless                 +2 to HITROLL, sets NOTHING\r\n"
	if !strings.Contains(out, wantSPL) {
		t.Fatalf("player stat SPL: lines mismatch:\ngot:\n%s\nwant substring:\n%s", out, wantSPL)
	}

	// Verify MobInstance stat also renders AFF: and SPL: lines in newest-first order,
	// and that smokescreen (AFF_BLIND = C bit 0) sets BLIND in AFF: without emitting a bitvector suffix on SPL:.
	mob, err := w.SpawnMob(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.AddAffect(&engine.Affect{
		Origin:    engine.OriginSpell,
		SpellID:   spells.SpellSmokescreen,
		Source:    "smokescreen",
		Duration:  1,
		Location:  engine.ApplyHitroll,
		Magnitude: -4,
		Flags:     engine.AFFBlind,
	})
	mob.AddAffect(&engine.Affect{
		Origin:    engine.OriginSpell,
		SpellID:   spells.SpellSmokescreen,
		Source:    "smokescreen",
		Duration:  1,
		Location:  engine.ApplyAC,
		Magnitude: 40,
		Flags:     engine.AFFBlind,
	})

	s.sendStatMob(mob)
	mobOut := drainSessionOutput(t, s)
	if !strings.Contains(mobOut, "AFF: BLIND\r\n") {
		t.Fatalf("mob stat AFF: line missing BLIND:\n%s", mobOut)
	}
	wantMobSPL := "SPL: (  2hr) smokescreen           +40 to ARMOR\r\n" +
		"SPL: (  2hr) smokescreen           -4 to HITROLL\r\n"
	if !strings.Contains(mobOut, wantMobSPL) {
		t.Fatalf("mob stat SPL: lines mismatch:\ngot:\n%s\nwant substring:\n%s", mobOut, wantMobSPL)
	}
}
