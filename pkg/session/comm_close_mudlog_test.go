package session

import (
	"bytes"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1371 batch 6 — comm.c:2133, close_socket's CON_PLAYING arm.
//
// C (src/comm.c, close_socket):
//
//	:2131-2133  act("$n has lost $s link.", TRUE, d->character, 0, 0, TO_ROOM);
//	            sprintf(buf, "Closing link to: %s.", GET_NAME(d->character));
//	            mudlog(buf, NRM, MAX(LVL_IMMORT, GET_INVIS_LEV(d->character)), TRUE);
//
// Go producer: pkg/session/manager.go:1532 in handleTransportDisconnect — the
// transport-disconnect caller of close_socket (DP-1323). Two sibling emitters
// mirror *other* callers of the same C line and belong to other rows:
// idle_close.go:36 (C limits.c:442-444, the idle row) and wiz_object.go:239
// (the purge path). The observed payload is in the census dump
// (…/census-dump/mudlog-wizard-producers.txt:66, `Closing link to:`).
//
// The row's two unproven cells are the same discriminators as the quit/LOSTEQ
// pair: type NRM (a LOG2-only observer receives, a LOG1-only observer stays
// silent) and level MAX(LVL_IMMORT, invis).

type closeObservers struct {
	watch      *Session // game.LVL_IMMORT, LOG2 only
	brief      *Session // level 40, LOG1 only
	below      *Session // game.LVL_IMMORT-1, both bits
	atFloor    *Session // game.LVL_IMMORT, both bits
	atInvis    *Session // 36, both bits
	underInvis *Session // 35, both bits
}

func closeMudlogFixture(t *testing.T) (*Manager, *bytes.Buffer, closeObservers) {
	t.Helper()
	file := captureMudlogFile(t)
	m := makeTestManager(t)
	obs := closeObservers{
		watch:      addObserver(t, m, "Closewatch", game.LVL_IMMORT, false, true),
		brief:      addObserver(t, m, "Closebrief", 40, true, false),
		below:      addObserver(t, m, "Closebelow", game.LVL_IMMORT-1, true, true),
		atFloor:    addObserver(t, m, "Closeatfloor", game.LVL_IMMORT, true, true),
		atInvis:    addObserver(t, m, "Closeat36", 36, true, true),
		underInvis: addObserver(t, m, "Closeunder36", 35, true, true),
	}
	return m, file, obs
}

// closeLinkdeadSession stages an authenticated playing character with a live
// transport and takes it through the transport-disconnect close. The channel is
// the one reaper_test.go:426 sets: DetachTransport closes transportDone, so a nil
// channel panics (session_player.go:163-167).
func closeLinkdeadSession(t *testing.T, m *Manager, name string) *Session {
	t.Helper()
	s := makeTestSession(t, m, name, 1001, true)
	s.transportDone = make(chan struct{})
	registerTestSession(t, m, s, name)
	return s
}

// closeBracketed is the exact consumer bytes for the close payload.
func closeBracketed(payload string) string { return "[ " + payload + " ]\r\n" }

// closeLineSeen reports whether a session's stream contains the exact bracketed
// mudlog line. The close path also broadcasts the room act
// ("$n has lost $s link."), which every observer in the room receives; that is a
// different message and must never satisfy — or trip — a syslog assertion.
func closeLineSeen(t *testing.T, s *Session, payload string) bool {
	t.Helper()
	return strings.Contains(strings.Join(drainSessionText(t, s), ""), closeBracketed(payload))
}

// TestCloseSocketMudlogProducer pins payload, file and the two negative
// observers for comm.c:2133, each against this producer's own bytes.
func TestCloseSocketMudlogProducer(t *testing.T) {
	m, file, obs := closeMudlogFixture(t)
	s := closeLinkdeadSession(t, m, "Linkless")
	if kept := m.HandleTransportDisconnect(s); !kept {
		t.Fatal("HandleTransportDisconnect = false, want the linkdead retention")
	}
	payload := "Closing link to: Linkless."
	if got := strings.Count(file.String(), payload); got != 1 {
		t.Fatalf("file payload count = %d, want 1; log = %q", got, file.String())
	}
	if !closeLineSeen(t, obs.watch, payload) {
		t.Fatalf("LOG2-only observer did not receive %q", closeBracketed(payload))
	}
	if closeLineSeen(t, obs.brief, payload) {
		t.Fatalf("LOG1-only observer received %q: type would be BRF", closeBracketed(payload))
	}
	if closeLineSeen(t, obs.below, payload) {
		t.Fatalf("level-%d observer received %q", game.LVL_IMMORT-1, closeBracketed(payload))
	}
}

// TestCloseSocketMudlogTypeIsNRM: a LOG2-only observer receives (type <= NRM)
// while a LOG1-only observer stays silent (type > BRF), both judged on this
// producer's exact line. The retention check keeps the negative half from
// passing vacuously if the close path is not taken.
func TestCloseSocketMudlogTypeIsNRM(t *testing.T) {
	m, _, obs := closeMudlogFixture(t)
	s := closeLinkdeadSession(t, m, "Typeless")
	if kept := m.HandleTransportDisconnect(s); !kept {
		t.Fatal("HandleTransportDisconnect = false, want the linkdead retention")
	}
	payload := "Closing link to: Typeless."
	if !closeLineSeen(t, obs.watch, payload) {
		t.Fatalf("LOG2-only observer did not receive %q: type > NRM", closeBracketed(payload))
	}
	if closeLineSeen(t, obs.brief, payload) {
		t.Fatalf("LOG1-only observer received %q: type <= BRF", closeBracketed(payload))
	}
}

// TestCloseSocketMudlogLevelDiscriminators: the LVL_IMMORT floor and the
// invisibility override, per producer bytes.
func TestCloseSocketMudlogLevelDiscriminators(t *testing.T) {
	t.Run("lv_immort_floor", func(t *testing.T) {
		m, _, obs := closeMudlogFixture(t)
		s := closeLinkdeadSession(t, m, "Floorlink")
		if kept := m.HandleTransportDisconnect(s); !kept {
			t.Fatal("HandleTransportDisconnect = false, want the linkdead retention")
		}
		payload := "Closing link to: Floorlink."
		if closeLineSeen(t, obs.below, payload) {
			t.Fatalf("level-%d observer received %q", game.LVL_IMMORT-1, closeBracketed(payload))
		}
		if !closeLineSeen(t, obs.atFloor, payload) {
			t.Fatalf("level-%d observer did not receive %q", game.LVL_IMMORT, closeBracketed(payload))
		}
	})
	t.Run("invis_raises_threshold", func(t *testing.T) {
		m, _, obs := closeMudlogFixture(t)
		s := closeLinkdeadSession(t, m, "Invislink")
		s.player.SetInvisLevel(36)
		if kept := m.HandleTransportDisconnect(s); !kept {
			t.Fatal("HandleTransportDisconnect = false, want the linkdead retention")
		}
		payload := "Closing link to: Invislink."
		if closeLineSeen(t, obs.underInvis, payload) {
			t.Fatalf("level-35 observer received %q", closeBracketed(payload))
		}
		if !closeLineSeen(t, obs.atInvis, payload) {
			t.Fatalf("level-36 observer did not receive %q at invisibility 36", closeBracketed(payload))
		}
	})
}
