package session

// DP-1621: every room line whose C counterpart is act() must route through
// game.Act, so sleepers get nothing, and viewers who cannot see the actor get
// "someone" (hide_invisible FALSE, C LIGHT_OK is false for a blind viewer,
// src/utils.h:515) or nothing at all (hide_invisible TRUE, src/comm.c:2543).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func makeRoomActTestManager(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			// Every direction leaves 1001, so a flee's first draw is always a
			// viable direction and the panics act fires exactly once.
			{VNum: 1001, Name: "Act Room", Zone: 1, Exits: map[string]parser.Exit{
				"north": {ToRoom: 1002}, "east": {ToRoom: 1002}, "south": {ToRoom: 1002},
				"west": {ToRoom: 1002}, "up": {ToRoom: 1002}, "down": {ToRoom: 1002},
			}},
			{VNum: 1002, Name: "Next Room", Zone: 1},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// C do_echo's room copy is act(buf, FALSE, ch, 0, 0, TO_ROOM)
// (src/act.wizard.c:147): a sleeper gets nothing.
func TestEchoRoomActSkipsSleepingViewer(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	god := makeCommandTestSession(t, m, "Echoer", 40, 1001)
	sleeper := makeCommandTestSession(t, m, "Sleeper", 1, 1001)
	awake := makeCommandTestSession(t, m, "Awake", 1, 1001)
	registerInWorld(t, god)
	registerInWorld(t, sleeper)
	registerInWorld(t, awake)
	sleeper.player.SetPosition(combat.PosSleeping)

	if err := ExecuteCommand(god, "echo", []string{"plain", "words"}); err != nil {
		t.Fatalf("ExecuteCommand echo: %v", err)
	}
	if got := drainSendChannel(t, sleeper); got != "" {
		t.Fatalf("sleeping viewer got %q, want nothing (src/act.wizard.c:147)", got)
	}
	if got := readMsgText(t, awake); got != "Plain words\r\n" {
		t.Fatalf("awake viewer got %q, want the echoed words (act capitalizes, C perform_act)", got)
	}
}

// C doubles typed $ signs at input (src/comm.c:1968-1977), so an echo's
// typed "$n" reaches act as "$$n" and prints literally for everyone. The
// empirically retained oracle bytes for "echo $n lit test." are the literal
// argument, capitalized by act's CAP.
func TestEchoActPassesTypedDollarsThrough(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	god := makeCommandTestSession(t, m, "Echoer", 40, 1001)
	awake := makeCommandTestSession(t, m, "Awake", 1, 1001)
	registerInWorld(t, god)
	registerInWorld(t, awake)

	if err := ExecuteCommand(god, "echo", []string{"$n", "tests", "$s", "codes."}); err != nil {
		t.Fatalf("ExecuteCommand echo: %v", err)
	}
	if got := readMsgText(t, awake); got != "$n tests $s codes.\r\n" {
		t.Fatalf("awake viewer got %q, want the literal argument (src/comm.c:1975-1976 doubling)", got)
	}
	if got := readMsgText(t, god); got != "$n tests $s codes.\r\n" {
		t.Fatalf("echoer got %q, want the literal TO_CHAR copy", got)
	}
}

// do_think's sprintf'd "$n thinks..." prefix substitutes with PERS, so a
// viewer who cannot see the thinker reads "Someone"; the thought text itself
// passes through literally for the same input-doubling reason as echo.
func TestThinkTypedDollarsStayLiteral(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	thinker := makeCommandTestSession(t, m, "Thinker", 40, 1001)
	blind := makeCommandTestSession(t, m, "Blindman", 1, 1001)
	awake := makeCommandTestSession(t, m, "Awake", 1, 1001)
	registerInWorld(t, thinker)
	registerInWorld(t, blind)
	registerInWorld(t, awake)
	thinker.player.Stats.Int = 10 // do_think refuses INT-0 actors (src/act.comm.c:1370)
	thinker.player.CopyBaseAttributes()
	blind.player.SetAffect(game.AffBlind, true)

	if err := ExecuteCommand(thinker, "think", []string{"$n", "quietly"}); err != nil {
		t.Fatalf("ExecuteCommand think: %v", err)
	}
	if got := readMsgText(t, awake); got != "Thinker thinks . o O ( $n quietly )\r\n" {
		t.Fatalf("awake viewer got %q, want the literal thought text", got)
	}
	if got := readMsgText(t, blind); got != "Someone thinks . o O ( $n quietly )\r\n" {
		t.Fatalf("blind viewer got %q, want Someone with the literal thought text", got)
	}
}

// do_echo's SCMD_EMOTE branch sprintf's "$n %s" over the doubled argument
// (src/act.wizard.c:149): the prefix substitutes, the body stays literal.
func TestEmotePrefixSubstitutesBodyLiteral(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	actor := makeCommandTestSession(t, m, "Emoter", 10, 1001)
	watcher := makeCommandTestSession(t, m, "Awake", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, watcher)
	actor.player.Stats.Int = 10 // do_echo's emote gate refuses INT-0 (src/act.wizard.c:139)
	actor.player.CopyBaseAttributes()

	if err := ExecuteCommand(actor, "emote", []string{"$n", "$s", "probe."}); err != nil {
		t.Fatalf("ExecuteCommand emote: %v", err)
	}
	if got := readMsgText(t, watcher); got != "Emoter $n $s probe.\r\n" {
		t.Fatalf("watcher got %q, want prefix substituted and body literal", got)
	}
}

// C do_think sends act(buf, FALSE, ch, 0, 0, TO_ROOM)
// (src/act.comm.c:1363, :1376): a sleeper gets nothing; a blind viewer gets
// "someone" in place of the thinker.
func TestThinkRoomActRespectsViewerSight(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	thinker := makeCommandTestSession(t, m, "Thinker", 10, 1001)
	sleeper := makeCommandTestSession(t, m, "Sleeper", 1, 1001)
	blind := makeCommandTestSession(t, m, "Blindman", 1, 1001)
	registerInWorld(t, thinker)
	registerInWorld(t, sleeper)
	registerInWorld(t, blind)
	sleeper.player.SetPosition(combat.PosSleeping)
	blind.player.SetAffect(game.AffBlind, true)

	if err := ExecuteCommand(thinker, "think", nil); err != nil {
		t.Fatalf("ExecuteCommand think: %v", err)
	}
	if got := drainSendChannel(t, sleeper); got != "" {
		t.Fatalf("sleeping viewer got %q, want nothing (src/act.comm.c:1363)", got)
	}
	if got := drainSendChannel(t, blind); !strings.Contains(got, "Someone thinks about life, the universe, and everything.") {
		t.Fatalf("blind viewer got %q, want Someone thinks ...", got)
	}
}

// C do_flee sends act("$n panics, and attempts to flee!", TRUE, ch, 0, 0,
// TO_ROOM) (src/act.offensive.c:397): hide is TRUE, so a blind viewer gets
// nothing at all and a sleeper is skipped.
func TestFleeRoomActRespectsViewerSight(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	fleer := makeCommandTestSession(t, m, "Fleer", 10, 1001)
	sleeper := makeCommandTestSession(t, m, "Sleeper", 1, 1001)
	blind := makeCommandTestSession(t, m, "Blindman", 1, 1001)
	awake := makeCommandTestSession(t, m, "Awake", 1, 1001)
	registerInWorld(t, fleer)
	registerInWorld(t, sleeper)
	registerInWorld(t, blind)
	registerInWorld(t, awake)
	sleeper.player.SetPosition(combat.PosSleeping)
	blind.player.SetAffect(game.AffBlind, true)

	dprng.ResetStream(1)
	if err := ExecuteCommand(fleer, "flee", nil); err != nil {
		t.Fatalf("ExecuteCommand flee: %v", err)
	}
	if got := drainSendChannel(t, blind); got != "" {
		t.Fatalf("blind viewer got %q, want nothing (src/act.offensive.c:397 hides it)", got)
	}
	if got := drainSendChannel(t, sleeper); got != "" {
		t.Fatalf("sleeping viewer got %q, want nothing", got)
	}
	if got := drainSendChannel(t, awake); !strings.Contains(got, "Fleer panics, and attempts to flee!") {
		t.Fatalf("awake viewer got %q, want the panics act", got)
	}
}

// C do_wizutil freeze sends act("A sudden cold wind conjured from nowhere
// freezes $n!", FALSE, vict, 0, 0, TO_ROOM) (src/act.wizard.c:2152): the
// victim is act's ch, so sleepers get nothing and a blind viewer reads
// "someone" in place of the victim.
func TestFreezeRoomActRespectsViewerSight(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	god := makeCommandTestSession(t, m, "Freezer", 40, 1001)
	victim := makeCommandTestSession(t, m, "Victim", 10, 1001)
	sleeper := makeCommandTestSession(t, m, "Sleeper", 1, 1001)
	blind := makeCommandTestSession(t, m, "Blindman", 1, 1001)
	awake := makeCommandTestSession(t, m, "Awake", 1, 1001)
	registerInWorld(t, god)
	registerInWorld(t, victim)
	registerInWorld(t, sleeper)
	registerInWorld(t, blind)
	registerInWorld(t, awake)
	sleeper.player.SetPosition(combat.PosSleeping)
	blind.player.SetAffect(game.AffBlind, true)

	if err := ExecuteCommand(god, "freeze", []string{"Victim"}); err != nil {
		t.Fatalf("ExecuteCommand freeze: %v", err)
	}
	if got := drainSendChannel(t, sleeper); got != "" {
		t.Fatalf("sleeping viewer got %q, want nothing (src/act.wizard.c:2152)", got)
	}
	if got := drainSendChannel(t, awake); !strings.Contains(got, "freezes Victim!") {
		t.Fatalf("sighted viewer got %q, want the victim's name", got)
	}
	if got := drainSendChannel(t, blind); !strings.Contains(got, "freezes someone!") {
		t.Fatalf("blind viewer got %q, want freezes someone!", got)
	}
	if victim.player.GetFlags()&(1<<game.PlrFrozen) == 0 {
		t.Fatal("victim was not frozen")
	}
}

// C do_wizutil thaw sends act("A sudden fireball conjured from nowhere thaws
// $n!", FALSE, vict, 0, 0, TO_ROOM) (src/act.wizard.c:2172), same audience
// rules as freeze.
func TestThawRoomActRespectsViewerSight(t *testing.T) {
	w := makeRoomActTestManager(t)
	m := newTestManager(t, w, nil)
	god := makeCommandTestSession(t, m, "Thawer", 40, 1001)
	victim := makeCommandTestSession(t, m, "Victim", 10, 1001)
	sleeper := makeCommandTestSession(t, m, "Sleeper", 1, 1001)
	blind := makeCommandTestSession(t, m, "Blindman", 1, 1001)
	registerInWorld(t, god)
	registerInWorld(t, victim)
	registerInWorld(t, sleeper)
	registerInWorld(t, blind)
	sleeper.player.SetPosition(combat.PosSleeping)
	blind.player.SetAffect(game.AffBlind, true)

	if err := ExecuteCommand(god, "freeze", []string{"Victim"}); err != nil {
		t.Fatalf("ExecuteCommand freeze: %v", err)
	}
	drainSendChannel(t, sleeper)
	drainSendChannel(t, blind)
	drainSendChannel(t, victim)

	if err := ExecuteCommand(god, "thaw", []string{"Victim"}); err != nil {
		t.Fatalf("ExecuteCommand thaw: %v", err)
	}
	if got := drainSendChannel(t, sleeper); got != "" {
		t.Fatalf("sleeping viewer got %q, want nothing (src/act.wizard.c:2172)", got)
	}
	if got := drainSendChannel(t, blind); !strings.Contains(got, "thaws someone!") {
		t.Fatalf("blind viewer got %q, want thaws someone!", got)
	}
}
