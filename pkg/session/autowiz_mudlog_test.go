package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// autowizObservers registers three immortals that differ only in the two
// audience gates a CMP / LVL_IMMORT / file-FALSE producer must pass: watch is
// at the level with complete syslog, below is one level short of the gate
// (still complete syslog), and brief is high enough but has only the brief
// syslog bit.
func autowizObservers(t *testing.T, m *Manager) (watch, below, brief *Session) {
	t.Helper()
	watch = makeTestSession(t, m, "Autowatch", 1001, true)
	watch.player.SetLevel(LVL_IMMORT)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	registerTestSession(t, m, watch, "Autowatch")

	below = makeTestSession(t, m, "Autobelow", 1001, true)
	below.player.SetLevel(LVL_IMMORT - 1)
	below.player.SetPlrFlag(game.PrfLog1, true)
	below.player.SetPlrFlag(game.PrfLog2, true)
	registerTestSession(t, m, below, "Autobelow")

	brief = makeTestSession(t, m, "Autobrief", 1001, true)
	brief.player.SetLevel(LVL_IMPL)
	brief.player.SetPlrFlag(game.PrfLog1, true)
	registerTestSession(t, m, brief, "Autobrief")
	return watch, below, brief
}

// autowizAdvanceFixture is do_advance's promotion path: an implementor actor,
// a mortal victim, and the three observers.
func autowizAdvanceFixture(t *testing.T) (m *Manager, actor, victim, watch, below, brief *Session) {
	t.Helper()
	m = makeTestManager(t)
	actor = makeTestSession(t, m, "Implementor", 1001, true)
	actor.player.SetLevel(LVL_IMPL)
	registerTestSession(t, m, actor, "Implementor")

	victim = makeTestSession(t, m, "Ghost", 1001, true)
	victim.player.SetLevel(1)
	registerTestSession(t, m, victim, "Ghost")

	watch, below, brief = autowizObservers(t, m)
	return m, actor, victim, watch, below, brief
}

// src/limits.c:269-283: check_autowiz logs "Initiating autowiz." at
// CMP / LVL_IMMORT / file FALSE when the promoted character reaches LVL_IMMORT,
// and do_advance reaches it through one gain_exp_regardless per level
// (src/act.wizard.c:1569-1571). Drive the real command and read the real
// consumers.
func TestAutowizMudlogProducer(t *testing.T) {
	t.Run("crossing the gate", func(t *testing.T) {
		_, actor, victim, watch, below, brief := autowizAdvanceFixture(t)
		victim.player.SetLevel(30)
		file := captureMudlogFile(t)

		if err := cmdAdvance(actor, []string{"Ghost", "31"}); err != nil {
			t.Fatal(err)
		}

		// advance_level's own per-level producer runs first (it is file TRUE),
		// then the rise message, then check_autowiz — C's order inside
		// gain_exp_regardless.
		want := "[ Ghost advanced to level 31 ]\r\n[ Initiating autowiz. ]\r\n"
		if got := strings.Join(drainSessionText(t, watch), ""); got != want {
			t.Fatalf("observer = %q, want %q", got, want)
		}
		if got := strings.Join(drainSessionText(t, below), ""); got != "" {
			t.Fatalf("a level-%d observer saw %q", LVL_IMMORT-1, got)
		}
		// The type gate: CMP is 3, the brief bit alone is 1, so only the
		// BRF millstone line reaches this observer.
		if got := strings.Join(drainSessionText(t, brief), ""); got != "[ Ghost advanced to level 31 ]\r\n" {
			t.Fatalf("brief-syslog observer saw %q", got)
		}
		// The file flag: file FALSE leaves no alog line.
		f := file.String()
		if !strings.Contains(f, "Ghost advanced to level 31") {
			t.Fatalf("file lost the file-TRUE producer: %q", f)
		}
		if strings.Contains(f, "Initiating autowiz.") {
			t.Fatalf("autowiz producer reached the file with file FALSE: %q", f)
		}
	})

	t.Run("an immortal victim interleaves the rise and autowiz lines", func(t *testing.T) {
		_, actor, victim, _, _, _ := autowizAdvanceFixture(t)
		// An already-immortal victim whose syslog is complete hears both lines.
		victim.player.SetLevel(LVL_IMMORT)
		victim.player.SetPlrFlag(game.PrfLog1, true)
		victim.player.SetPlrFlag(game.PrfLog2, true)
		drainSessionText(t, victim)

		if err := cmdAdvance(actor, []string{"Ghost", "33"}); err != nil {
			t.Fatal(err)
		}
		// C calls gain_exp_regardless once per level (src/act.wizard.c:1569-1571)
		// and each call ends with the rise message and then check_autowiz
		// (src/limits.c:350-360), with advance_level's own producer inside the
		// call. Batching the announcements in the caller would put both autowiz
		// lines ahead of both rise lines, which is not C's stream.
		want := "[ Ghost advanced to level 32 ]\r\nYou rise a level!\r\n[ Initiating autowiz. ]\r\n" +
			"[ Ghost advanced to level 33 ]\r\nYou rise a level!\r\n[ Initiating autowiz. ]\r\n"
		stream := strings.Join(drainSessionText(t, victim), "")
		if !strings.HasSuffix(stream, want) {
			t.Fatalf("victim stream = %q, want it to end with the per-level interleave %q", stream, want)
		}
		if !strings.Contains(stream, "You feel slightly different.") {
			t.Fatalf("victim stream lost C's promotion text: %q", stream)
		}
	})

	t.Run("below the gate", func(t *testing.T) {
		_, actor, victim, watch, _, brief := autowizAdvanceFixture(t)
		victim.player.SetLevel(29)
		file := captureMudlogFile(t)

		if err := cmdAdvance(actor, []string{"Ghost", "30"}); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(drainSessionText(t, watch), ""); got != "[ Ghost advanced to level 30 ]\r\n" {
			t.Fatalf("observer = %q, want only the level line", got)
		}
		if got := strings.Join(drainSessionText(t, brief), ""); got != "[ Ghost advanced to level 30 ]\r\n" {
			// The BRF level line reaches the brief-syslog observer; the CMP
			// autowiz line would not, and must not exist at all here.
			t.Fatalf("brief-syslog observer saw %q", got)
		}
		if f := file.String(); strings.Contains(f, "Initiating autowiz.") {
			t.Fatalf("the producer fired below LVL_IMMORT: %q", f)
		}
	})
}
