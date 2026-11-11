package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// yankTestWorld builds a minimal world with a leader (the yank actor) whose
// output is captured via a MessageSink, plus helpers to add follower victims.
func yankTestWorld(t *testing.T) (*World, *Player, *strings.Builder) {
	t.Helper()
	w := &World{}
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	w.players = make(map[string]*Player)

	leader := NewPlayer(1, "Leader", 1001)
	leader.worldRef = w
	w.AddPlayer(leader)
	// Return the leader and a closure-backed builder. Tests read out.String().
	return w, leader, &out
}

// addYankFollower adds a player follower in the leader's room at a given position.
func addYankFollower(t *testing.T, w *World, leader *Player, name string, pos int) *Player {
	t.Helper()
	f := NewPlayer(2, name, 1001)
	f.worldRef = w
	f.SetFollowing(leader.Name)
	f.SetPosition(pos)
	w.AddPlayer(f)
	return f
}

// --- Branch: no argument ---
func TestDoYank_NoArg(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	w.ExecYank(leader, nil, "")
	if got, want := out.String(), "Who do you wish to yank?\r\n"; got != want {
		t.Errorf("no-arg = %q, want %q", got, want)
	}
}

// --- Branch: target not found (NOPERSON) ---
func TestDoYank_NotFound(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	w.ExecYank(leader, nil, "nobody")
	if got, want := out.String(), "No-one by that name here.\r\n"; got != want {
		t.Errorf("not-found = %q, want exact C NOPERSON %q", got, want)
	}
}

// --- Branch: self-target (the "wierd" typo) ---
func TestDoYank_SelfTarget(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	// Leader follows themselves so they pass findCharInRoom (name match) and
	// reach the self-check before the follower check.
	w.ExecYank(leader, nil, "Leader")
	if got, want := out.String(), "That's wierd.\r\n"; got != want {
		t.Errorf("self = %q, want %q (sic: wierd)", got, want)
	}
}

// --- Branch: not your follower ---
func TestDoYank_NotFollower(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	// A non-follower in the room.
	stranger := NewPlayer(2, "Stranger", 1001)
	stranger.worldRef = w
	stranger.SetFollowing("") // follows no one
	w.AddPlayer(stranger)

	w.ExecYank(leader, nil, "Stranger")
	if got, want := out.String(), "That probably wouldn't be appreciated.\r\n"; got != want {
		t.Errorf("not-follower = %q, want %q", got, want)
	}
}

// --- Branch: already up (not mounted) ---
func TestDoYank_AlreadyUp_NotMounted(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	addYankFollower(t, w, leader, "Follower", combat.PosStanding)

	w.ExecYank(leader, nil, "Follower")
	// C: "$N is already on $S feet." — $N = name, $S = possessive pronoun.
	// Male follower (default sex 0) → "his".
	got := out.String()
	want := "Follower is already on his feet.\r\n"
	if got != want {
		t.Errorf("already-up not-mounted = %q, want %q", got, want)
	}
}

// --- Branch: already up (mounted) ---
func TestDoYank_AlreadyUp_Mounted(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	f := addYankFollower(t, w, leader, "Follower", combat.PosStanding)
	f.SetAffect(affMounted, true)

	w.ExecYank(leader, nil, "Follower")
	// C: "You can't yank $M off $S mount!" — $M = objective pronoun (him),
	// $S = possessive (his). Male follower → "him"/"his".
	got := out.String()
	want := "You can't yank him off his mount!\r\n"
	if got != want {
		t.Errorf("already-up mounted = %q, want %q", got, want)
	}
}

// --- Branch: sleeping/below (the "is is" typo) ---
func TestDoYank_Sleeping(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	addYankFollower(t, w, leader, "Follower", combat.PosSleeping)

	w.ExecYank(leader, nil, "Follower")
	// C: "$N is is no position to be yanked around!" — sic: "is is".
	got := out.String()
	want := "Follower is is no position to be yanked around!\r\n"
	if got != want {
		t.Errorf("sleeping = %q, want %q (sic: is is)", got, want)
	}
}

// --- Branch: success with a female follower (pronoun expansion she/her) ---
func TestDoYank_Success_FemalePronouns(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	f := addYankFollower(t, w, leader, "Girl", combat.PosSitting)
	f.Sex = 1 // female

	w.ExecYank(leader, nil, "Girl")
	// $M = "her", $S = "her" for female.
	got := out.String()
	if want := "You yank her to her feet.\r\n"; !strings.Contains(got, want) {
		t.Errorf("female success actor msg missing %q in %q", want, got)
	}
}

// --- Threshold: a RESTING follower is yankable (above SLEEPING, ≤ SITTING) ---
func TestDoYank_RestingFollowerIsYankable(t *testing.T) {
	w, leader, _ := yankTestWorld(t)
	f := addYankFollower(t, w, leader, "Follower", combat.PosResting)

	w.ExecYank(leader, nil, "Follower")
	if got := f.GetPosition(); got != combat.PosStanding {
		t.Errorf("resting follower should be yankable → POS_STANDING, got pos %d", got)
	}
}

// C's success room line (src/act.other.c:1658) is TO_NOTVICT, which excludes the
// actor AND the victim. The victim must see only their TO_VICT line; a third
// occupant sees the room line. Revert proof: the old actToRoom(..., ch.Name)
// arm delivered the room line to the victim too.
func TestDoYank_SuccessRoomLineSkipsVictim(t *testing.T) {
	w := &World{}
	msgs := map[string]*strings.Builder{}
	w.MessageSink = func(name string, msg []byte) {
		if msgs[name] == nil {
			msgs[name] = &strings.Builder{}
		}
		msgs[name].Write(msg)
	}
	w.players = make(map[string]*Player)

	leader := NewPlayer(1, "Leader", 1001)
	leader.worldRef = w
	w.AddPlayer(leader)
	victim := NewPlayer(2, "Follower", 1001)
	victim.worldRef = w
	victim.SetFollowing(leader.Name)
	victim.SetPosition(combat.PosSitting)
	w.AddPlayer(victim)
	observer := NewPlayer(3, "Watcher", 1001)
	observer.worldRef = w
	w.AddPlayer(observer)

	w.ExecYank(leader, nil, "Follower")

	victimOut := msgs["Follower"].String()
	if !strings.Contains(victimOut, "Leader yanks you to your feet.\r\n") {
		t.Fatalf("victim missing TO_VICT line: %q", victimOut)
	}
	if strings.Contains(victimOut, "Leader yanks Follower to his feet.") {
		t.Fatalf("victim received the TO_NOTVICT room line: %q", victimOut)
	}
	if obsOut := msgs["Watcher"].String(); !strings.Contains(obsOut, "Leader yanks Follower to his feet.\r\n") {
		t.Fatalf("observer missing TO_NOTVICT room line: %q", obsOut)
	}
	// The success arm still stands the victim (act.other.c:1659).
	if got := victim.GetPosition(); got != combat.PosStanding {
		t.Fatalf("victim position = %d, want POS_STANDING (%d)", got, combat.PosStanding)
	}
}
