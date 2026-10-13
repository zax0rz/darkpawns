package session

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWiznetRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("wiznet")
	if !ok {
		t.Fatal("wiznet command is not registered")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("wiznet gate = level %d position %d, want level 0 position dead", entry.MinLevel, entry.MinPosition)
	}
}

func TestCmdWiznetEntryAllowsChosenMortal(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Chosenwiz", 1, 1001)
	m.sessions[actor.player.Name] = actor

	if err := cmdWiznetText(actor, "ordinary"); err != nil {
		t.Fatalf("ordinary wiznet: %v", err)
	}
	if got, want := readSessionText(t, actor), "Huh?!?"; got != want {
		t.Fatalf("unchosen mortal output = %q, want %q", got, want)
	}

	actor.player.SetPLRFlag(game.PlrChosen)
	if err := cmdWiznetText(actor, "chosen message"); err != nil {
		t.Fatalf("chosen wiznet: %v", err)
	}
	if got, want := readSessionText(t, actor), "Chosenwiz: chosen message\r\n"; got != want {
		t.Fatalf("chosen mortal output = %q, want %q", got, want)
	}
}

func TestWiznetHalfChopMatchesCWordBoundary(t *testing.T) {
	cases := []struct {
		input, first, remainder string
	}{
		{"  32  message  ", "32", "message  "},
		{"message", "message", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		first, remainder := wiznetHalfChop(tc.input)
		if first != tc.first || remainder != tc.remainder {
			t.Errorf("wiznetHalfChop(%q) = (%q, %q), want (%q, %q)", tc.input, first, remainder, tc.first, tc.remainder)
		}
	}
}

// wiznetListFixture registers an Implementor and one god in each state
// do_wiznet's '@' loops distinguish (act.wizard.c:1948-1980). Sessions are
// ordered newest first, as descriptor_list prepends each new descriptor.
func wiznetListFixture(t *testing.T) (*Manager, *Session, map[string]*Session) {
	t.Helper()
	m := makeTestManager(t)
	base := time.Now()
	add := func(name string, level int, age int) *Session {
		s := makeCommandTestSession(t, m, name, level, 1001)
		s.connectedAt = base.Add(-time.Duration(age) * time.Minute)
		m.sessions[name] = s
		return s
	}
	actor := add("Wizlister", LVL_IMPL, 10)
	gods := map[string]*Session{
		"online":   add("Ongod", 34, 5),
		"nowiz":    add("Offgod", 34, 4),
		"linkless": add("Linkgod", 34, 3),
		"menu":     add("Menugod", 34, 2),
		"mortal":   add("Mortal", 10, 1),
	}
	gods["nowiz"].player.Flags |= 1 << uint(game.PrfNowiz)
	done := make(chan struct{})
	close(done)
	gods["linkless"].transportDone = done
	gods["menu"].menuActive = true
	return m, actor, gods
}

// TestWiznetListSplitsNowizGodsOffline proves '@': the first loop lists
// playing gods without NOWIZ, the second lists playing gods with NOWIZ, and
// a linkless god or one at the menu (d->connected != 0) is in neither.
func TestWiznetListSplitsNowizGodsOffline(t *testing.T) {
	_, actor, _ := wiznetListFixture(t)
	if err := cmdWiznetText(actor, "@"); err != nil {
		t.Fatal(err)
	}
	want := "Gods online:\r\n  Ongod\r\n  Wizlister\r\nGods offline:\r\n  Offgod\r\n"
	if got := readSessionText(t, actor); got != want {
		t.Fatalf("wiznet @ = %q, want %q", got, want)
	}
}

// TestWiznetListImplementorBypassIsOnlineOnly proves the LVL_IMPL clause
// guards only the first loop: a blind Implementor still lists online gods,
// but CAN_SEE alone filters the NOWIZ list.
func TestWiznetListImplementorBypassIsOnlineOnly(t *testing.T) {
	_, actor, _ := wiznetListFixture(t)
	actor.player.SetAffect(game.AffBlind, true)
	if err := cmdWiznetText(actor, "@"); err != nil {
		t.Fatal(err)
	}
	want := "Gods online:\r\n  Ongod\r\n  Wizlister\r\n"
	if got := readSessionText(t, actor); got != want {
		t.Fatalf("blind implementor wiznet @ = %q, want %q", got, want)
	}
}

// TestWiznetBroadcastSkipsGodAtMenu proves the broadcast loop's !d->connected
// test (act.wizard.c:2010): a god at the menu hears nothing.
func TestWiznetBroadcastSkipsGodAtMenu(t *testing.T) {
	_, actor, gods := wiznetListFixture(t)
	if err := cmdWiznetText(actor, "hello gods"); err != nil {
		t.Fatal(err)
	}
	if got := readSessionText(t, gods["online"]); !strings.Contains(got, "Wizlister: hello gods") {
		t.Fatalf("online god heard %q", got)
	}
	if got := renderedOutput(gods["menu"]); got != "" {
		t.Fatalf("god at the menu heard %q", got)
	}
}

// TestWiznetListApprovedFreshBuffer pins the approved divergence
// (wiznet.gods-list-stale-buffer): when no god qualifies for the online loop,
// C appends to or resends the stale global buf1 (act.wizard.c:1948-1980;
// db.h:223). The port sends only what this listing built.
func TestWiznetListApprovedFreshBuffer(t *testing.T) {
	m := makeTestManager(t)
	alone := makeCommandTestSession(t, m, "Lonegod", 34, 1001)
	alone.player.Flags |= 1 << uint(game.PrfNowiz)
	m.sessions[alone.player.Name] = alone
	if err := cmdWiznetText(alone, "@"); err != nil {
		t.Fatal(err)
	}
	if got, want := readSessionText(t, alone), "Gods offline:\r\n  Lonegod\r\n"; got != want {
		t.Fatalf("NOWIZ god alone = %q, want %q", got, want)
	}

	m2 := makeTestManager(t)
	chosen := makeCommandTestSession(t, m2, "Chosenone", 10, 1001)
	chosen.player.SetPLRFlag(game.PlrChosen)
	m2.sessions[chosen.player.Name] = chosen
	if err := cmdWiznetText(chosen, "@"); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(chosen); got != "" {
		t.Fatalf("chosen mortal with no gods = %q, want nothing", got)
	}
}
