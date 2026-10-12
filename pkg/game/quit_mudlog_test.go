package game

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1371 batch 6 — quit (act.other.c:136) and LOSTEQ (act.other.c:173).
//
// C (src/act.other.c, do_quit):
//
//	:135-136  sprintf(buf, "%s has quit the game.", GET_NAME(ch));
//	          mudlog(buf, NRM, MAX(LVL_IMMORT, GET_INVIS_LEV(ch)), TRUE);
//	:171-173  sprintf(buf, "LOSTEQ:%s has quit out of a save room.", GET_NAME(ch));
//	          mudlog(buf, NRM, MAX(LVL_IMMORT, GET_INVIS_LEV(ch)), TRUE);
//
// Go producers: pkg/game/quit.go:77 (quit) and :85 (LOSTEQ), both
// MudlogNormal (NRM = 2) at max(lvlImmort, GetInvisLevel()) with file TRUE.
// The observed payloads are in the reallyquit scenario's own transcript
// (…/dp-1371-p4-entry-name-claims-final/seed1/initial-dump/mudlog-quit-syslog.txt:
// `[ Quitter has quit the game. ]` at :5 and
// `[ LOSTEQ:Quitter has quit out of a save room. ]` at :6); these tests turn
// that observation into assertions.
//
// Discriminators, from the delivery gate (logging.go: "an observer's syslog
// level is PRF_LOG1 = 1 + PRF_LOG2 = 2 and must be at least the type"):
//
//	type  NRM: a LOG2-only observer (log_level 2) receives  => type <= NRM,
//	          a LOG1-only observer (log_level 1) stays silent => type > BRF.
//	level      MAX(LVL_IMMORT, invis): a both-bits observer one level below the
//	          threshold is silent and one exactly at it receives — for both the
//	          LVL_IMMORT floor and an invisibility of 36 above it.

// quitMudlogObservers stages the two producers' contract. All observers sit in
// room 1001; the quitter's room is what selects the producer path.
type quitObservers struct {
	watch      *Player // LVL_IMMORT, LOG2 only  (log_level 2)
	brief      *Player // level 40,   LOG1 only  (log_level 1)
	below      *Player // LVL_IMMORT-1, both bits: under the floor
	atFloor    *Player // LVL_IMMORT,   both bits: exactly at the floor
	atInvis    *Player // 36, both bits: exactly at invisibility 36
	underInvis *Player // 35, both bits: under invisibility 36
}

func quitMudlogObservers(t *testing.T) (*milestoneProvider, *bytes.Buffer, quitObservers) {
	t.Helper()
	newObserver := func(id int, name string, level int, log1, log2 bool) *Player {
		p := NewPlayer(id, name, 1001)
		p.SetLevel(level)
		p.SetPlrFlag(PrfLog1, log1)
		p.SetPlrFlag(PrfLog2, log2)
		return p
	}
	obs := quitObservers{
		watch:      newObserver(201, "Quitwatch", LVL_IMMORT, false, true),
		brief:      newObserver(202, "Quitbrief", 40, true, false),
		below:      newObserver(203, "Quitbelow", LVL_IMMORT-1, true, true),
		atFloor:    newObserver(204, "Quitatfloor", LVL_IMMORT, true, true),
		atInvis:    newObserver(205, "Quitat36", 36, true, true),
		underInvis: newObserver(206, "Quitunder36", 35, true, true),
	}
	provider := &milestoneProvider{
		players: []*Player{obs.watch, obs.brief, obs.below, obs.atFloor, obs.atInvis, obs.underInvis},
		lines:   map[string]string{},
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(provider)
	t.Cleanup(func() { SetImmortalSessionProvider(old) })
	file := &bytes.Buffer{}
	oldWriter := getLogWriter()
	SetLogWriter(file)
	t.Cleanup(func() { SetLogWriter(oldWriter) })
	return provider, file, obs
}

// quitMudlogWorld builds the two rooms the quit decision needs: 8008 is a
// temple (isokquit, act.other.c:87-112) and 1001 is an ordinary unsafe room,
// so only reallyquit from 1001 reaches the LOSTEQ branch.
func quitMudlogWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Town Square", Zone: 1},
		{VNum: 8008, Name: "Temple Annex", Zone: 1},
	}})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func quitLine(name string) string   { return name + " has quit the game." }
func losteqLine(name string) string { return "LOSTEQ:" + name + " has quit out of a save room." }

// bracketed is the exact consumer bytes for one mudlog payload.
func bracketed(payload string) string { return "[ " + payload + " ]\r\n" }

// sawMudlogLine reports whether an observer's stream contains this producer's
// exact bracketed line. The reallyquit path emits the quit line *and* the LOSTEQ
// line, so every assertion names the bytes it means: "the observer received
// something" could pass on the sibling line even with LOSTEQ's type or level
// wrong.
func sawMudlogLine(provider *milestoneProvider, p *Player, payload string) bool {
	return strings.Contains(provider.lines[p.Name], bracketed(payload))
}

// TestQuitMudlogProducers pins payload, file and the two negative observers for
// each producer.
func TestQuitMudlogProducers(t *testing.T) {
	t.Run("quit_line_from_a_temple", func(t *testing.T) {
		provider, file, obs := quitMudlogObservers(t)
		w := quitMudlogWorld(t)
		p := NewPlayer(301, "Quitter", 8008)
		p.SetLevel(1)
		if out := w.DoQuit(p, false); out != QuitLogoutKeepEQ {
			t.Fatalf("DoQuit = %d, want QuitLogoutKeepEQ", out)
		}
		payload := quitLine("Quitter")
		if got := strings.Count(file.String(), payload); got != 1 {
			t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
		}
		if !sawMudlogLine(provider, obs.watch, payload) {
			t.Fatalf("LOG2-only observer did not receive %q", bracketed(payload))
		}
		if sawMudlogLine(provider, obs.brief, payload) {
			t.Fatalf("LOG1-only observer received %q: type would be BRF", bracketed(payload))
		}
		if sawMudlogLine(provider, obs.below, payload) {
			t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracketed(payload))
		}
	})

	t.Run("losteq_line_from_an_unsafe_room", func(t *testing.T) {
		provider, file, obs := quitMudlogObservers(t)
		w := quitMudlogWorld(t)
		p := NewPlayer(302, "Risker", 1001)
		p.SetLevel(1)
		if out := w.DoQuit(p, true); out != QuitLogoutLoseEQ {
			t.Fatalf("DoQuit = %d, want QuitLogoutLoseEQ", out)
		}
		lost := losteqLine("Risker")
		if got := strings.Count(file.String(), lost); got != 1 {
			t.Fatalf("file LOSTEQ count = %d, want 1; log = %q", got, file.String())
		}
		if !sawMudlogLine(provider, obs.watch, lost) {
			t.Fatalf("LOG2-only observer did not receive %q", bracketed(lost))
		}
		// The quit line is unconditional (act.other.c:135-136) and precedes it;
		// assert both by their own bytes, in order.
		q := quitLine("Risker")
		if got := strings.Count(file.String(), q); got != 1 {
			t.Fatalf("file quit count = %d, want 1; log = %q", got, file.String())
		}
		stream := provider.lines[obs.watch.Name]
		qIdx, lIdx := strings.Index(stream, bracketed(q)), strings.Index(stream, bracketed(lost))
		if qIdx < 0 || lIdx < 0 || qIdx > lIdx {
			t.Fatalf("order: quit=%d losteq=%d in %q", qIdx, lIdx, stream)
		}
		if sawMudlogLine(provider, obs.brief, lost) {
			t.Fatalf("LOG1-only observer received %q: type would be BRF", bracketed(lost))
		}
		if sawMudlogLine(provider, obs.below, lost) {
			t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracketed(lost))
		}
	})
}

// TestQuitMudlogTypeIsNRM: a LOG2-only observer receives this producer's exact
// line (type <= NRM) while a LOG1-only observer does not (type > BRF). NRM->BRF
// breaks the second half, NRM->CMP the first, and judging each half on the
// producer's own bytes keeps the sibling quit line from standing in for LOSTEQ.
func TestQuitMudlogTypeIsNRM(t *testing.T) {
	for _, tc := range []struct {
		name    string
		room    int
		really  bool
		payload func(string) string
	}{
		{"quit_line", 8008, false, quitLine},
		{"losteq_line", 1001, true, losteqLine},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider, _, obs := quitMudlogObservers(t)
			w := quitMudlogWorld(t)
			p := NewPlayer(310, "Typer", tc.room)
			p.SetLevel(1)
			w.DoQuit(p, tc.really)
			payload := tc.payload("Typer")
			if !sawMudlogLine(provider, obs.watch, payload) {
				t.Fatalf("LOG2-only observer did not receive %q: type > NRM", bracketed(payload))
			}
			if sawMudlogLine(provider, obs.brief, payload) {
				t.Fatalf("LOG1-only observer received %q: type <= BRF", bracketed(payload))
			}
		})
	}
}

// TestQuitMudlogLevelDiscriminators: the LVL_IMMORT floor and the invisibility
// override, each judged on this producer's own bracketed line.
func TestQuitMudlogLevelDiscriminators(t *testing.T) {
	for _, tc := range []struct {
		name    string
		room    int
		really  bool
		payload func(string) string
	}{
		{"quit_line", 8008, false, quitLine},
		{"losteq_line", 1001, true, losteqLine},
	} {
		t.Run(tc.name+"/lv_immort_floor", func(t *testing.T) {
			provider, _, obs := quitMudlogObservers(t)
			w := quitMudlogWorld(t)
			p := NewPlayer(320, "Floor", tc.room)
			p.SetLevel(1)
			w.DoQuit(p, tc.really)
			payload := tc.payload("Floor")
			if sawMudlogLine(provider, obs.below, payload) {
				t.Fatalf("level-%d observer received %q", LVL_IMMORT-1, bracketed(payload))
			}
			if !sawMudlogLine(provider, obs.atFloor, payload) {
				t.Fatalf("level-%d observer did not receive %q", LVL_IMMORT, bracketed(payload))
			}
		})
		t.Run(tc.name+"/invis_raises_threshold", func(t *testing.T) {
			provider, _, obs := quitMudlogObservers(t)
			w := quitMudlogWorld(t)
			p := NewPlayer(321, "Invis", tc.room)
			p.SetLevel(1)
			p.SetInvisLevel(36)
			w.DoQuit(p, tc.really)
			payload := tc.payload("Invis")
			if sawMudlogLine(provider, obs.underInvis, payload) {
				t.Fatalf("level-35 observer received %q", bracketed(payload))
			}
			if !sawMudlogLine(provider, obs.atInvis, payload) {
				t.Fatalf("level-36 observer did not receive %q at invisibility 36", bracketed(payload))
			}
		})
	}
}
