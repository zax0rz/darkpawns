package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newLookNotifyWorld builds one room with a looker, a peer, and a single NPC so
// the description rendering and the look_at_target observer echo can both be
// asserted. peerMsgs captures everything the peer's client receives.
func newLookNotifyWorld(t *testing.T) (*World, *Player, *Player, *MobInstance, *[]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Look Room"}},
		Mobs: []parser.Mob{{
			VNum:         2001,
			Keywords:     "harbor guard",
			ShortDesc:    "A watchful guard",
			LongDesc:     "A watchful guard stands here.\n",
			DetailedDesc: "A scrawny figure in a harbor cloak.",
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	looker := NewPlayer(1, "Looker", 1001)
	peer := NewPlayer(2, "Peer", 1001)
	for _, p := range []*Player{looker, peer} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s): %v", p.Name, err)
		}
	}
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	// The sink is installed after the spawn so the room's "appears" act is not
	// part of the captured peer transcript.
	var peerMsgs []string
	w.MessageSink = func(name string, msg []byte) {
		if name == peer.Name {
			peerMsgs = append(peerMsgs, string(msg))
		}
	}
	return w, looker, peer, mob, &peerMsgs
}

// TestLookAtMobUnterminatedDescriptionRunsIntoTheNextLine pins C's
// send_to_char(i->player.description, ch) semantics (act.informative.c:394-395):
// no terminator and no capitalization, so a description without a line ending —
// exactly what do_string's inline write stores, since modify.c:762 keeps the
// trailing-CRLF append commented out — runs straight into diag_char_to_char's
// condition sentence.
func TestLookAtMobUnterminatedDescriptionRunsIntoTheNextLine(t *testing.T) {
	w, looker, _, mob, _ := newLookNotifyWorld(t)

	result := w.DoLookTarget(looker, "guard")
	if len(result.Messages) == 0 {
		t.Fatal("no messages produced")
	}
	first := result.Messages[0]
	if !first.Raw {
		t.Error("the description must bypass act() so it is not capitalized or re-terminated")
	}
	want := "A scrawny figure in a harbor cloak.A watchful guard " + diagCondition(mob.GetHP(), mob.GetMaxHP()) + "\r\n"
	if first.Format != want {
		t.Errorf("first line = %q, want %q", first.Format, want)
	}
}

// TestLookAtMobTerminatedDescriptionKeepsItsOwnLine covers the world-file shape:
// a description that already ends in a newline leaves the condition sentence on
// its own line, exactly as before this fix.
func TestLookAtMobTerminatedDescriptionKeepsItsOwnLine(t *testing.T) {
	w, looker, _, mob, _ := newLookNotifyWorld(t)
	mob.SetLiveDetailedDesc("A scrawny figure in a harbor cloak.\r\n")

	result := w.DoLookTarget(looker, "guard")
	if len(result.Messages) < 2 {
		t.Fatalf("messages = %#v, want the description and the condition line", result.Messages)
	}
	if got := result.Messages[0].Format; got != "A scrawny figure in a harbor cloak.\r\n" {
		t.Errorf("description = %q, want it verbatim", got)
	}
	if got := result.Messages[1].Format; !strings.HasPrefix(got, "$N ") {
		t.Errorf("condition line = %q, want the act-rendered form", got)
	}
}

// TestLookAtMobDescriptionIsNotCapitalized: only diag_char_to_char CAPs its
// buffer (act.informative.c:363-365); the description itself is sent as stored.
func TestLookAtMobDescriptionIsNotCapitalized(t *testing.T) {
	w, looker, _, mob, _ := newLookNotifyWorld(t)
	mob.SetLiveDetailedDesc("the guard is unremarkable")

	result := w.DoLookTarget(looker, "guard")
	got := result.Messages[0].Format
	if !strings.HasPrefix(got, "the guard is unremarkable") {
		t.Errorf("description = %q, want the stored lower-case text", got)
	}
}

// TestLookAtCharacterNotifiesTheRoom pins look_at_target's observer echo
// (act.informative.c:1022-1029), which fires for mob targets as well as players.
func TestLookAtCharacterNotifiesTheRoom(t *testing.T) {
	t.Run("mob target", func(t *testing.T) {
		w, looker, _, _, peerMsgs := newLookNotifyWorld(t)
		w.doLook(looker, nil, "look", "guard")

		want := "Looker looks at A watchful guard.\r\n"
		if got := strings.Join(*peerMsgs, ""); got != want {
			t.Errorf("peer output = %q, want %q", got, want)
		}
	})

	t.Run("player target", func(t *testing.T) {
		w, looker, _, _, peerMsgs := newLookNotifyWorld(t)
		w.doLook(looker, nil, "look", "peer")

		// C's act() sends TO_NOTVICT to the room minus the actor and minus the
		// victim (comm.c:2533-2534), so the victim only hears the TO_VICT line.
		want := "Looker looks at you.\r\n"
		if got := strings.Join(*peerMsgs, ""); got != want {
			t.Errorf("peer output = %q, want %q", got, want)
		}
	})

	t.Run("looking at self is silent", func(t *testing.T) {
		w, looker, _, _, peerMsgs := newLookNotifyWorld(t)
		w.doLook(looker, nil, "look", "self")

		if got := strings.Join(*peerMsgs, ""); got != "" {
			t.Errorf("peer output = %q, want nothing when the looker is the target", got)
		}
	})
}
