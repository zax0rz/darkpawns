package session

// autosave_test.go — R5h unit proofs for the autosave slot (DP-1365): the
// ten-minute counter, the eligibility filters (C walks descriptor_list for
// CON_PLAYING non-NPCs with PLR_CRASH, objsave.c:1211-1223), the
// clear-after-success policy, and retry after a failed write. The no-flag
// case (case 3 of the proof plan) lives here too: an unflagged character is
// never rewritten.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// autosaveWorld builds a manager whose world's save seam counts writes and
// returns a scripted result, plus a registered playing session.
type autosaveHarness struct {
	m      *Manager
	w      *game.World
	calls  *[]string
	result *game.SaveResult
}

func newAutosaveHarness(t *testing.T, result game.SaveResult) *autosaveHarness {
	t.Helper()
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Vault"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	m := newTestManager(t, w, nil)
	calls := &[]string{}
	res := result
	h := &autosaveHarness{m: m, w: w, calls: calls, result: &res}
	w.PlayerSaver = func(p *game.Player, why string, _ int) game.SaveResult {
		*calls = append(*calls, p.GetName()+":"+why)
		return res
	}
	return h
}

func (h *autosaveHarness) addSession(t *testing.T, name string) *Session {
	t.Helper()
	s := makeTestSession(t, h.m, name, 1001, true)
	h.m.mu.Lock()
	h.m.sessions[name] = s
	h.m.mu.Unlock()
	return s
}

// TestAutosaveFiresAtMinuteTenNotNine: C's counter fires Crash_save_all on
// the tenth minute slot and resets (comm.c:832-837).
func TestAutosaveFiresAtMinuteTenNotNine(t *testing.T) {
	h := newAutosaveHarness(t, game.SaveSucceeded)
	s := h.addSession(t, "Minutecount")
	s.player.MarkCrashNeeded()

	for i := 1; i <= 9; i++ {
		h.m.AutosaveTick()
	}
	if len(*h.calls) != 0 {
		t.Fatalf("autosave fired before minute 10: %v", *h.calls)
	}
	h.m.AutosaveTick()
	if len(*h.calls) != 1 || (*h.calls)[0] != "Minutecount:autosave" {
		t.Fatalf("minute-10 autosave calls = %v", *h.calls)
	}
	// The counter reset: ten more ticks fire exactly once more, and the
	// flag cleared by the first pass keeps the second pass silent unless
	// new inventory changes re-flag the character.
	for i := 1; i <= 10; i++ {
		h.m.AutosaveTick()
	}
	if len(*h.calls) != 1 {
		t.Fatalf("autosave repeated without a re-flag: %v", *h.calls)
	}
}

// TestAutosaveEligibility: only connected, playing, PLR_CRASH-flagged,
// non-guest characters are saved. Menu, creation-pending, link-dead and
// unflagged sessions are not (C's descriptor_list walk skips them all).
func TestAutosaveEligibility(t *testing.T) {
	h := newAutosaveHarness(t, game.SaveSucceeded)

	flagged := h.addSession(t, "Flagme")
	flagged.player.MarkCrashNeeded()

	unflagged := h.addSession(t, "Cleanhands") // case 3: no flag, no write

	guest := h.addSession(t, "Guestling")
	guest.isGuest = true
	guest.player.MarkCrashNeeded()

	menu := h.addSession(t, "Menulurker")
	menu.menuActive = true
	menu.player.MarkCrashNeeded()

	creating := h.addSession(t, "Statroller")
	creating.creationSaved = true
	creating.player.MarkCrashNeeded()

	linkdead := h.addSession(t, "Linkdeadish")
	linkdead.player.SetLinkless(true)
	linkdead.player.MarkCrashNeeded()

	for i := 0; i < 10; i++ {
		h.m.AutosaveTick()
	}

	if len(*h.calls) != 1 || (*h.calls)[0] != "Flagme:autosave" {
		t.Fatalf("autosave calls = %v, want only the flagged playing character", *h.calls)
	}
	if flagged.player.NeedsCrashSave() {
		t.Fatal("successful autosave did not clear the saved character's flag")
	}
	// Skipped-but-flagged characters keep PLR_CRASH (the retry policy):
	// the calls list above proves none of them was written.
	for name, s := range map[string]*Session{"guest": guest, "menu": menu, "creating": creating, "linkdead": linkdead} {
		if !s.player.NeedsCrashSave() {
			t.Fatalf("%s character lost its flag without a save", name)
		}
	}
	if unflagged.player.NeedsCrashSave() {
		t.Fatal("unflagged character gained a flag")
	}
}

// TestAutosaveRetriesAfterFailure: a failed write keeps PLR_CRASH, so the
// next eligible pass saves again (the documented durability exception to
// C's drop-on-failure behavior).
func TestAutosaveRetriesAfterFailure(t *testing.T) {
	h := newAutosaveHarness(t, game.SaveFailed)
	s := h.addSession(t, "Retryme")
	s.player.MarkCrashNeeded()

	for i := 0; i < 10; i++ {
		h.m.AutosaveTick()
	}
	if len(*h.calls) != 1 {
		t.Fatalf("first pass calls = %v", *h.calls)
	}
	if !s.player.NeedsCrashSave() {
		t.Fatal("a failed autosave cleared PLR_CRASH; the retry would be lost")
	}
	*h.result = game.SaveSucceeded
	for i := 0; i < 10; i++ {
		h.m.AutosaveTick()
	}
	if len(*h.calls) != 2 {
		t.Fatalf("no retry after failure: calls = %v", *h.calls)
	}
	if s.player.NeedsCrashSave() {
		t.Fatal("successful retry did not clear the flag")
	}
}
