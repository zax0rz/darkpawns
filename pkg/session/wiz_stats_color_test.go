package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// statColorOutput drains a session's queued frames into raw player bytes,
// skipping the input-mark frame so escape-code assertions see MUD text only.
func statColorOutput(t *testing.T, s *Session) string {
	t.Helper()
	var output strings.Builder
	for len(s.send) > 0 {
		msg := <-s.send
		if IsInputMarkFrame(msg) {
			continue
		}
		frame, ok := RenderTerminalFrame(msg)
		if !ok || frame.Kind != FrameText {
			t.Fatalf("stat emitted a non-text terminal frame: %s", msg)
		}
		output.WriteString(frame.Text)
	}
	return output.String()
}

// statColorLevel sets the two preference bits C's _clrlevel() reads
// (src/screen.h:45-49): COLOR_1 counts 1, COLOR_2 counts 2.
func statColorLevel(t *testing.T, s *Session, color1, color2 bool) {
	t.Helper()
	s.player.SetPlrFlag(game.PrfColor1, color1)
	s.player.SetPlrFlag(game.PrfColor2, color2)
}

// statColorLevels is every state _clrlevel() can reach, with the two gates the
// stat surface uses: C_NRM is _clrlevel >= 2 and C_CMP is _clrlevel >= 3.
var statColorLevels = []struct {
	name     string
	color1   bool
	color2   bool
	normal   bool // clr(ch, C_NRM)
	complete bool // clr(ch, C_CMP)
}{
	{name: "color off"},
	{name: "color 1 only (C_SPR)", color1: true},
	{name: "normal (color 2 only)", color2: true, normal: true},
	{name: "complete (both bits)", color1: true, color2: true, normal: true, complete: true},
}

// do_stat_character colours almost every row at C_NRM but the kills row at
// C_CMP (src/act.wizard.c:745-976), so PRF_COLOR_2 alone must colour the report
// except for that one row, which needs both bits. The expected bytes are built
// from the literal codes in src/screen.h:34-41, not from the production
// constants, so the test fails if the constants or the gates drift.
func TestStatPlayerColorLevelsMatchC(t *testing.T) {
	for _, level := range statColorLevels {
		t.Run(level.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeTestSession(t, m, "Statgod", 1001, true)
			statColorLevel(t, s, level.color1, level.color2)

			s.sendStatPlayerReport(s.player, 1001, true, 0)
			got := statColorOutput(t, s)

			cyan, green, yellow, reset := "", "", "", ""
			if level.normal {
				cyan, green, yellow, reset = "\x1b[36m", "\x1b[32m", "\x1b[33m", "\x1b[0m"
			}
			red := ""
			if level.complete {
				red = "\x1b[31m"
			}

			wants := []string{
				// act.wizard.c:745-750
				fmt.Sprintf("Lev: [%s%2d%s], XP: [%s%7d%s], Align:",
					yellow, s.player.GetLevel(), reset, yellow, s.player.GetExp(), reset),
				// act.wizard.c:797-823
				"Str: [" + cyan, "]\tInt: [" + cyan, "]\t*\tHit p.:  [" + green,
				"Dex: [" + cyan, "]\tWis: [" + cyan, "]\t*\tMana p.: [" + green,
				"Con: [" + cyan, "]\tCha: [" + cyan, "]\t*\tMove p.: [" + green,
				// act.wizard.c:884-898
				"PLR: " + cyan,
				"PRF: " + green,
				"Racial Hatreds: " + green + "None None None None None" + reset,
				// act.wizard.c:968
				"AFF: " + yellow,
			}
			// act.wizard.c:831-835 is the only C_CMP site, so it stays plain at
			// C_NRM even though the rest of the report is coloured.
			if level.complete {
				wants = append(wants, fmt.Sprintf(
					"Kills: [%s%9d%s], PKills: [%s%9d%s], Deaths: [%s%9d%s]",
					red, s.player.Kills, reset, red, s.player.PKs, reset, red, s.player.Deaths, reset))
			} else {
				wants = append(wants, fmt.Sprintf("Kills: [%9d], PKills: [%9d], Deaths: [%9d]",
					s.player.Kills, s.player.PKs, s.player.Deaths))
			}

			for _, want := range wants {
				if !strings.Contains(got, want) {
					t.Errorf("stat report lacks %q\n  got=%q", want, got)
				}
			}
			if !level.normal && strings.Contains(got, "\x1b") {
				t.Errorf("stat report emitted colour below C_NRM: %q", got)
			}
		})
	}
}

// do_stat_character's SPL: row wraps the %-21s spell name in CCCYN
// (src/act.wizard.c:975-976), so the padding sits inside the colour.
func TestStatPlayerSpellLineColorMatchesC(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Statgod", 1001, true)
	s.player.AddAffect(&engine.Affect{
		Origin:   engine.OriginSpell,
		SpellID:  spells.SpellSanctuary,
		Source:   "sanctuary",
		Duration: 3,
		Flags:    engine.AFFSanctuary,
	})

	statColorLevel(t, s, false, true)
	s.sendStatPlayerReport(s.player, 1001, true, 0)
	got := statColorOutput(t, s)
	if want := "SPL: (  4hr) \x1b[36msanctuary            \x1b[0m "; !strings.Contains(got, want) {
		t.Fatalf("colored SPL row = %q, want substring %q", got, want)
	}

	statColorLevel(t, s, false, false)
	s.sendStatPlayerReport(s.player, 1001, true, 0)
	got = statColorOutput(t, s)
	if want := "SPL: (  4hr) sanctuary             "; !strings.Contains(got, want) {
		t.Fatalf("uncolored SPL row = %q, want substring %q", got, want)
	}
}

// do_stat_room colours the room name cyan, only the VNum green, and both the
// direction and the target room cyan on exit rows (src/act.wizard.c:422-500).
// A NOWHERE exit prints " NONE" with the leading space outside the colour.
func TestStatRoomColorLevelsMatchC(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Map Room", Zone: 1, Exits: map[string]parser.Exit{
			"up":    {Direction: "up", ToRoom: 1002},
			"south": {Direction: "south", ToRoom: -1},
		}},
		{VNum: 1002, Name: "Destination", Zone: 1},
	}}
	for _, level := range statColorLevels {
		t.Run(level.name, func(t *testing.T) {
			w, err := game.NewWorld(parsed)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			s := makeTestSession(t, newTestManager(t, w, nil), "Statgod", 1001, true)
			if err := w.AddPlayer(s.player); err != nil {
				t.Fatal(err)
			}
			statColorLevel(t, s, level.color1, level.color2)

			s.sendStatRoom()
			got := statColorOutput(t, s)

			cyan, green, reset := "", "", ""
			if level.normal {
				cyan, green, reset = "\x1b[36m", "\x1b[32m", "\x1b[0m"
			}
			wants := []string{
				"Room name: " + cyan + "Map Room" + reset + "\r\n",
				// Only the VNum is green; the zone number and RNum stay plain.
				"], VNum: [" + green + " 1001" + reset + "], RNum: [",
				// %-5s padding stays inside the colour.
				"Exit " + cyan + "up   " + reset + ":  To: [" + cyan + " 1002" + reset + "], Key: [",
				// NOWHERE keeps its leading space outside the colour.
				"Exit " + cyan + "south" + reset + ":  To: [ " + cyan + "NONE" + reset + "], Key: [",
				// The exit Type: field is sprintbit's output, which also ends in
				// a space, and C does not colour it (src/act.wizard.c:498-502).
				"Type: NOBITS \r\n ",
			}
			for _, want := range wants {
				if !strings.Contains(got, want) {
					t.Errorf("stat room lacks %q\n  got=%q", want, got)
				}
			}
			if !level.normal && strings.Contains(got, "\x1b") {
				t.Errorf("stat room emitted colour below C_NRM: %q", got)
			}
		})
	}
}

// do_stat_object colours the short description yellow and the VNum green
// (src/act.wizard.c:522-533).
func TestStatObjectColorLevelsMatchC(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room One"}},
		Objs:  []parser.Obj{{VNum: 200, ShortDesc: "a worn ring", Keywords: "ring", TypeFlag: game.ITEM_WORN}},
	}
	for _, level := range statColorLevels {
		t.Run(level.name, func(t *testing.T) {
			w, err := game.NewWorld(parsed)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			s := makeTestSession(t, newTestManager(t, w, nil), "Statgod", 1001, true)
			statColorLevel(t, s, level.color1, level.color2)

			obj := game.NewObjectInstance(&parser.Obj{
				VNum: 200, ShortDesc: "a worn ring", Keywords: "ring", TypeFlag: game.ITEM_WORN,
			}, 0)
			s.sendStatObjectInstance(obj)
			got := statColorOutput(t, s)

			yellow, green, reset := "", "", ""
			if level.normal {
				yellow, green, reset = "\x1b[33m", "\x1b[32m", "\x1b[0m"
			}
			wants := []string{
				"Name: '" + yellow + "a worn ring" + reset + "', Aliases: ring\r\n",
				"VNum: [" + green + "  200" + reset + "], RNum: [",
			}
			for _, want := range wants {
				if !strings.Contains(got, want) {
					t.Errorf("stat object lacks %q\n  got=%q", want, got)
				}
			}
			if !level.normal && strings.Contains(got, "\x1b") {
				t.Errorf("stat object emitted colour below C_NRM: %q", got)
			}
		})
	}
}

// do_stat_character's NPC rows — the level/experience pair, NPC flags and AFF —
// are all C_NRM, so PRF_COLOR_2 alone colours them (src/act.wizard.c:745-976).
func TestStatMobColorLevelsMatchC(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room One"}},
		Mobs: []parser.Mob{{
			VNum: 300, ShortDesc: "a guard", Keywords: "guard",
			AC: 100, THAC0: 20, Str: 11, Int: 11, Wis: 11, Dex: 11, Con: 11, Cha: 11,
			Damage: parser.DiceRoll{Num: 1, Sides: 4, Plus: 1},
		}},
	}
	for _, level := range statColorLevels {
		t.Run(level.name, func(t *testing.T) {
			w, err := game.NewWorld(parsed)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			s := makeTestSession(t, newTestManager(t, w, nil), "Statgod", 1001, true)
			mob, err := w.SpawnMob(300, 1001)
			if err != nil {
				t.Fatal(err)
			}
			statColorLevel(t, s, level.color1, level.color2)

			s.sendStatMob(mob)
			got := statColorOutput(t, s)

			yellow, cyan := "", ""
			if level.normal {
				yellow, cyan = "\x1b[33m", "\x1b[36m"
			}
			wants := []string{
				"Monster Class: Normal, Lev: [" + yellow,
				"NPC flags: " + cyan,
				"AFF: " + yellow,
			}
			for _, want := range wants {
				if !strings.Contains(got, want) {
					t.Errorf("stat mob lacks %q\n  got=%q", want, got)
				}
			}
			if !level.normal && strings.Contains(got, "\x1b") {
				t.Errorf("stat mob emitted colour below C_NRM: %q", got)
			}
		})
	}
}

// C's sprintbitarray never trims: it appends "NAME " per set bit and "NOBITS "
// when none are set (src/utils.c:825-843), and every stat site prints the result
// straight into the report. The trailing space is therefore part of the line,
// and in the colored form it sits before CCNRM — mid-line, where the oracle
// normalizer's trailing-whitespace rule cannot hide it. wizard-stat-color-depth
// is what exposed the port trimming it (R1/R5h).
func TestStatFlagListsKeepSprintbitarrayTrailingSpace(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Statgod", 1001, true)
	statColorLevel(t, s, false, true)

	s.sendStatPlayerReport(s.player, 1001, true, 0)
	got := statColorOutput(t, s)
	for _, want := range []string{
		"AFF: \x1b[33mNOBITS \x1b[0m\r\n",
		// The PLR word is the player's real PLR half, not a fabricated crash-save
		// marker: this viewer carries only the two colour preference bits.
		"PLR: \x1b[36mNOBITS \x1b[0m\r\n",
		"PRF: \x1b[32mAUTOEX C2 \x1b[0m\r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("player stat flag line lacks %q\n  got=%q", want, got)
		}
	}

	// The same bytes with colour off, where the space now abuts the CRLF. With
	// no colour bits set the whole flag word is empty, so C's PLR list is the
	// NOBITS fallback rather than the crash-save marker.
	statColorLevel(t, s, false, false)
	s.sendStatPlayerReport(s.player, 1001, true, 0)
	got = statColorOutput(t, s)
	for _, want := range []string{
		"AFF: NOBITS \r\n",
		"PLR: NOBITS \r\n",
		"PRF: AUTOEX \r\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("uncolored player stat flag line lacks %q\n  got=%q", want, got)
		}
	}
}
