package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func combatDescriptorFixture(t *testing.T) (*Manager, *Session, *Session, *Session, *game.MobInstance, *game.MobInstance) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 300, Keywords: "guard", ShortDesc: "Guard", Level: 20, Sex: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	first, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	first.SetPosition(combat.PosStanding)
	second.SetPosition(combat.PosStanding)
	m := newTestManager(t, w, nil)
	wizard := makeTestSession(t, m, "Wizard", 1001, true)
	wizard.transportDone = make(chan struct{})
	registerTestSession(t, m, wizard, "Wizard")
	peer := makeTestSession(t, m, "Guard", 1001, true)
	peer.transportDone = make(chan struct{})
	registerTestSession(t, m, peer, "Guard")
	sleeper := makeTestSession(t, m, "Sleeper", 1001, true)
	sleeper.transportDone = make(chan struct{})
	registerTestSession(t, m, sleeper, "Sleeper")
	sleeper.player.SetPosition(combat.PosSleeping)
	// Same fields populated by the already playtested NPC switch path.
	m.mu.Lock()
	wizard.isSwitched = true
	wizard.switchedMob = second
	m.mu.Unlock()
	m.WireCombatCallbacks()
	m.SetCombatMessageFunc()
	return m, wizard, peer, sleeper, first, second
}

func TestCombatBodyMessageRouting(t *testing.T) {
	m, wizard, peer, sleeper, first, second := combatDescriptorFixture(t)
	cb := m.combatEngine.Callbacks
	cb.SendToChar(first, "wrong duplicate")
	if got := renderedOutput(wizard); got != "" {
		t.Fatalf("unattached duplicate reached descriptor: %q", got)
	}
	cb.SendText(second, "selected mobile\r\n")
	if got := renderedOutput(wizard); got != "selected mobile\r\n" {
		t.Fatalf("NPC output=%q", got)
	}
	if got := renderedOutput(peer); got != "" {
		t.Fatalf("NPC output reached same-name PC: %q", got)
	}
	cb.SendRaw(second, "\x1b[31m")
	if got := renderedOutput(wizard); got != "\x1b[31m" {
		t.Fatalf("raw NPC bytes=%q", got)
	}
	cb.Broadcast(1001, "audience", []combat.Combatant{second})
	if got := renderedOutput(wizard); got != "" {
		t.Fatalf("excluded body saw audience: %q", got)
	}
	if got := renderedOutput(peer); got != "audience\r\n" {
		t.Fatalf("same-name PC incorrectly excluded: %q", got)
	}
	if got := renderedOutput(sleeper); got != "" {
		t.Fatalf("sleeping audience=%q", got)
	}
	cb.SendToChar(sleeper.player, "direct victim")
	if got := renderedOutput(sleeper); got != "direct victim\r\n" {
		t.Fatalf("direct sleeping victim lost: %q", got)
	}
}

func TestCombatBodyMessageColorsAndDraw(t *testing.T) {
	m, wizard, peer, _, _, second := combatDescriptorFixture(t)
	cb := m.combatEngine.Callbacks
	old := combat.GetRoller()
	t.Cleanup(func() { combat.SetRoller(old) })
	combat.SetRoller(combat.NewSeededRoller(123))
	combat.InitFightMessages(cb, combat.FightMessages{900: {{Hit: combat.FightMessageAction{Attacker: "You hit $N ($S).", Victim: "$n hits you.", Room: "$n hits $N."}}}})
	cb.GetColorLevel = func(body combat.Combatant) int {
		if body == peer.player {
			return 3
		}
		return 0
	}
	if !cb.SkillMessage(1, peer.player, second, 900, 1001) {
		t.Fatal("message not handled")
	}
	if got := renderedOutput(peer); got != "\x1b[33mYou hit Guard (her).\r\n\x1b[0m" {
		t.Fatalf("actor/color bytes=%q", got)
	}
	if got := renderedOutput(wizard); got != "Guard hits you.\r\n" {
		t.Fatalf("selected victim bytes=%q", got)
	}
	expected := combat.NewSeededRoller(123)
	expected.Dice(1, 1)
	if got, want := combat.GetRoller().Number(0, 100000), expected.Number(0, 100000); got != want {
		t.Fatalf("downstream draw=%d want %d", got, want)
	}
}

func TestCombatBodyPCSwitchRouting(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	m.WireCombatCallbacks()
	m.SetCombatMessageFunc()
	cb := m.combatEngine.Callbacks
	cb.SendToChar(original, "original has no descriptor")
	if got := renderedOutput(s); got != "" {
		t.Fatalf("switched original got output: %q", got)
	}
	cb.SendToChar(h.player, "borrowed body")
	if got := renderedOutput(s); !strings.Contains(got, "borrowed body") {
		t.Fatalf("borrowed output=%q", got)
	}
	if got := renderedOutput(h); got != "" {
		t.Fatalf("linkdead holder got output: %q", got)
	}
}

func TestCombatBodyDamageNotification(t *testing.T) {
	m, wizard, peer, _, first, second := combatDescriptorFixture(t)
	peer.player.SetFightingBody(first)
	peer.wantsStructuredData = true
	peer.subscribedVars[VarFighting] = true
	m.SetDamageFunc()
	m.combatEngine.DamageFunc(second)
	if len(peer.send) != 0 {
		t.Fatal("different duplicate emitted a structured target notification")
	}
	if got := renderedOutput(peer); got != "" {
		t.Fatalf("different duplicate updated target bytes: %q", got)
	}
	peer.player.SetFightingBody(second)
	m.combatEngine.DamageFunc(second)
	if len(peer.send) == 0 {
		t.Fatal("actual target lost dirty notification")
	}
	_ = wizard
}

func TestCombatBodyFollowerDetachMessages(t *testing.T) {
	m, _, peer, _, _, second := combatDescriptorFixture(t)
	follower := makeTestSession(t, m, "Follower", 1001, true)
	follower.transportDone = make(chan struct{})
	registerTestSession(t, m, follower, "Follower")
	follower.player.SetFollowingBody(second)
	m.combatEngine.Callbacks.StopFollowerOfMaster(follower.player, second)
	if got := renderedOutput(peer); !strings.Contains(got, "Follower stops following Guard.") || strings.Contains(got, "stops following you") {
		t.Fatalf("same-name PC received master message: %q", got)
	}
	if follower.player.GetFollowing() != "" || m.world.FollowingBody(follower.player) != nil {
		t.Fatal("relation retained after detach")
	}
}
