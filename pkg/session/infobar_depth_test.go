package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func readRawInfobarEvent(t *testing.T, s *Session) string {
	t.Helper()
	sm := unmarshalServerMsg(t, mustDrainSend(t, s))
	if sm.Type != MsgEvent {
		t.Fatalf("infobar message type = %q, want %q", sm.Type, MsgEvent)
	}
	data, err := json.Marshal(sm.Data)
	if err != nil {
		t.Fatalf("marshal infobar event data: %v", err)
	}
	var event EventData
	if err := json.Unmarshal(data, &event); err != nil {
		t.Fatalf("unmarshal infobar event data: %v", err)
	}
	if event.Type != "raw" {
		t.Fatalf("infobar event type = %q, want raw", event.Type)
	}
	return event.Text
}

func TestInfobarRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["infobar"]
	if !ok {
		t.Fatal("infobar command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosDead {
		t.Fatalf("infobar gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("infobar")
	if !ok {
		t.Fatal("infobar command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("infobar registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestInfobarUnknownStateResetsToOff(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Infobarstate", 1, 1001)
	s.infobarMode = 99

	if err := cmdInfoBar(s, nil); err != nil {
		t.Fatalf("cmdInfoBar: %v", err)
	}
	if got := readSessionText(t, s); got != "You had an unknown infobar setting.\r\n" {
		t.Fatalf("unknown-state output = %q", got)
	}
	if got := readSessionText(t, s); got != "It is being set to OFF.\r\n" {
		t.Fatalf("unknown-state reset output = %q", got)
	}
	if s.infobarMode != InfobarOff {
		t.Fatalf("infobar mode = %d, want off", s.infobarMode)
	}
}

func TestInfobarUpdateUsesCLayoutAndBitOrder(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Infobarupdate", game.LVL_IMMORT, 1001)
	s.player.SetPlrFlag(game.PrfColor2, true)
	s.screenSize = 25
	s.player.Health, s.player.MaxHealth = 100, 100
	s.player.Mana, s.player.MaxMana = 100, 100
	s.player.Move, s.player.MaxMove = 100, 100
	s.player.Exp, s.player.Gold = 1000, 10
	s.infobarMode = InfobarOn
	s.rememberInfobarValues()

	s.player.Health = 40
	s.player.Mana = 50
	s.player.Move = 80
	s.player.Exp = 123456
	s.player.Gold = 99
	cmdInfoBarUpdate(s)

	got := readRawInfobarEvent(t, s)
	if strings.Count(got, vtCurSave) != 5 || strings.Count(got, vtCurRest) != 5 {
		t.Fatalf("update save/restore count = %d/%d, want 5/5: %q", strings.Count(got, vtCurSave), strings.Count(got, vtCurRest), got)
	}
	positions := []string{
		fmt.Sprintf(vtCurSp, 22, 36), // INFO_MANA is rendered first.
		fmt.Sprintf(vtCurSp, 22, 63),
		fmt.Sprintf(vtCurSp, 22, 10),
		fmt.Sprintf(vtCurSp, 23, 6),
		fmt.Sprintf(vtCurSp, 24, 7),
	}
	last := -1
	for _, position := range positions {
		idx := strings.Index(got, vtCurSave+position)
		if idx <= last {
			t.Fatalf("update position %q is out of C bit order in %q", position, got)
		}
		last = idx
	}
	for _, want := range []string{
		vtYellow + "40" + vtNorm + "(" + vtGreen + "100",
		vtYellow + "50" + vtNorm + "(" + vtGreen + "100",
		vtYellow + "80" + vtNorm + "(" + vtGreen + "100",
		vtBlue + "123456" + vtNorm,
		vtMagenta + "99" + vtNorm,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("update output missing %q: %q", want, got)
		}
	}

	if s.infobarLastHit != 40 || s.infobarLastMana != 50 || s.infobarLastMove != 80 || s.infobarLastExp != 123456 || s.infobarLastGold != 99 {
		t.Fatalf("infobar last-value state was not refreshed: hit=%d mana=%d move=%d exp=%d gold=%d", s.infobarLastHit, s.infobarLastMana, s.infobarLastMove, s.infobarLastExp, s.infobarLastGold)
	}
	if _, ok := drainSend(s); ok {
		t.Fatal("unchanged values emitted a second infobar update")
	}
}

// Exercise the real ON and repaint paths, not an isolated color formatter.
func TestInfobarColorLevels(t *testing.T) {
	for level := 0; level < 4; level++ {
		t.Run(fmt.Sprint(level), func(t *testing.T) {
			m := makeTestManager(t)
			s := makeCommandTestSession(t, m, "Infobarcolor", game.LVL_IMMORT, 1001)
			s.player.SetPlrFlag(game.PrfColor1, level&1 != 0)
			s.player.SetPlrFlag(game.PrfColor2, level&2 != 0)
			s.screenSize = 25
			s.infobarMode = InfobarOn
			s.player.Health, s.player.MaxHealth = 100, 100
			s.player.Mana, s.player.MaxMana = 50, 100
			s.player.Move, s.player.MaxMove = 10, 100
			s.player.Exp, s.player.Gold = 1000, 10
			color := func(ansi string) string {
				if level >= 2 {
					return ansi
				}
				return ""
			}
			check := func(got string, hp, mana, move, exp, gold int, hpColor, manaColor, moveColor string) {
				t.Helper()
				for _, field := range []struct {
					row, col, count int
					ansi            string
				}{
					{22, 10, hp, hpColor}, {22, 36, mana, manaColor}, {22, 63, move, moveColor},
				} {
					want := fmt.Sprintf("\x1b[%d;%dH%s%d%s(%s100%s)", field.row, field.col, color(field.ansi), field.count, color("\x1b[0m"), color("\x1b[32m"), color("\x1b[0m"))
					if !strings.Contains(got, want) {
						t.Fatalf("resource missing %q in %q", want, got)
					}
				}
				for _, field := range []struct {
					col, count int
					ansi       string
				}{{6, exp, "\x1b[34m"}, {7, gold, "\x1b[35m"}} {
					row := 23
					if field.col == 7 {
						row = 24
					}
					want := fmt.Sprintf("\x1b[%d;%dH%s%d%s", row, field.col, color(field.ansi), field.count, color("\x1b[0m"))
					if !strings.Contains(got, want) {
						t.Fatalf("field missing %q in %q", want, got)
					}
				}
				if level < 2 {
					for _, ansi := range []string{"\x1b[0m", "\x1b[31m", "\x1b[32m", "\x1b[33m", "\x1b[34m", "\x1b[35m"} {
						if strings.Contains(got, ansi) {
							t.Fatalf("unexpected color %q in %q", ansi, got)
						}
					}
				}
			}
			cmdInfoBarOn(s)
			check(readRawInfobarEvent(t, s), 100, 50, 10, 1000, 10, "\x1b[32m", "\x1b[33m", "\x1b[31m")
			s.player.Health, s.player.Mana, s.player.Move = 50, 10, 100
			s.player.Exp, s.player.Gold = 2000, 20
			cmdInfoBarUpdate(s)
			got := readRawInfobarEvent(t, s)
			// Repaint clears fields before writing the same gated value bytes.
			got = strings.ReplaceAll(got, "          ", "")
			got = strings.ReplaceAll(got, "        ", "")
			check(got, 50, 10, 100, 2000, 20, "\x1b[33m", "\x1b[31m", "\x1b[32m")
		})
	}
}
