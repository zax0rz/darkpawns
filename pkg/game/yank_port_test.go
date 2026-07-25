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
	w.ExecYank(leader, "")
	if got, want := out.String(), "Who do you wish to yank?\r\n"; got != want {
		t.Errorf("no-arg = %q, want %q", got, want)
	}
}

// --- Branch: target not found (NOPERSON) ---
func TestDoYank_NotFound(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	w.ExecYank(leader, "nobody")
	if got, want := out.String(), "No-one by that name here.\r\n"; got != want {
		t.Errorf("not-found = %q, want exact C NOPERSON %q", got, want)
	}
}

// --- Branch: self-target (the "wierd" typo) ---
func TestDoYank_SelfTarget(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	// Leader follows themselves so they pass findCharInRoom (name match) and
	// reach the self-check before the follower check.
	w.ExecYank(leader, "Leader")
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

	w.ExecYank(leader, "Stranger")
	if got, want := out.String(), "That probably wouldn't be appreciated.\r\n"; got != want {
		t.Errorf("not-follower = %q, want %q", got, want)
	}
}

// --- Branch: already up (not mounted) ---
func TestDoYank_AlreadyUp_NotMounted(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	addYankFollower(t, w, leader, "Follower", combat.PosStanding)

	w.ExecYank(leader, "Follower")
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

	w.ExecYank(leader, "Follower")
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

	w.ExecYank(leader, "Follower")
	// C: "$N is is no position to be yanked around!" — sic: "is is".
	got := out.String()
	want := "Follower is is no position to be yanked around!\r\n"
	if got != want {
		t.Errorf("sleeping = %q, want %q (sic: is is)", got, want)
	}
}

// --- Branch: success (the 3 act() lines + POS_STANDING, pronoun expansion) ---
func TestDoYank_Success(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	f := addYankFollower(t, w, leader, "Follower", combat.PosSitting)

	w.ExecYank(leader, "Follower")

	// Actor message: "You yank $M to $S feet." → male: "You yank him to his feet."
	// (The room broadcast also reaches the leader via the sink; assert the
	// actor's TO_CHAR line is present with correct pronoun expansion.)
	got := out.String()
	if want := "You yank him to his feet.\r\n"; !strings.Contains(got, want) {
		t.Errorf("success actor msg missing %q in %q ($M/$S pronoun expansion)", want, got)
	}
	// Victim message: "$n yanks you to your feet." → "$n" = actor name.
	if want := "Leader yanks you to your feet.\r\n"; !strings.Contains(got, want) {
		t.Errorf("success victim msg missing %q in %q", want, got)
	}
	// Room message: "$n yanks $N to $S feet." → "$n" = Leader, "$N" = Follower,
	// "$S" = his.
	if want := "Leader yanks Follower to his feet.\r\n"; !strings.Contains(got, want) {
		t.Errorf("success room msg missing %q in %q", want, got)
	}
	// Victim is now standing.
	if got := f.GetPosition(); got != combat.PosStanding {
		t.Errorf("victim position = %d, want POS_STANDING (%d)", got, combat.PosStanding)
	}
}

// --- Branch: success with a female follower (pronoun expansion she/her) ---
func TestDoYank_Success_FemalePronouns(t *testing.T) {
	w, leader, out := yankTestWorld(t)
	f := addYankFollower(t, w, leader, "Girl", combat.PosSitting)
	f.Sex = 1 // female

	w.ExecYank(leader, "Girl")
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

	w.ExecYank(leader, "Follower")
	if got := f.GetPosition(); got != combat.PosStanding {
		t.Errorf("resting follower should be yankable → POS_STANDING, got pos %d", got)
	}
}
