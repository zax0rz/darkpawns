package session

// DP-1622: C doubles every $ in typed input (process_input, src/comm.c:1968-1977)
// before any command sees it, so text that reaches act() renders typed $-codes
// literally. These tests pin that behavior for the command families whose Go
// implementations route the typed text through game.Act: say, tell, the
// gen-comm channels, whisper/ask, and group say. Each test fails if Go
// substitutes the typed code instead of printing it.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// dollarPair registers an actor and a peer in room 1001 with live sessions.
func dollarPair(t *testing.T, actorName, peerName string) (*Session, *Session) {
	t.Helper()
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, actorName, 1, 1001)
	peer := makeCommandTestSession(t, m, peerName, 1, 1001)
	// say/gen-comm refuse INT-0 speakers (C's checkStupid, act.comm.c); some
	// channels (gossip) also carry C's minimum-level gate.
	actor.player.SetLevel(5)
	actor.player.Stats.Int = 10
	actor.player.Stats.Wis = 10
	actor.player.CopyBaseAttributes()
	peer.player.SetLevel(5)
	peer.player.Stats.Int = 10
	peer.player.CopyBaseAttributes()
	registerInWorld(t, actor)
	registerInWorld(t, peer)
	return actor, peer
}

func assertLiteralDollarN(t *testing.T, got, context string) {
	t.Helper()
	if !strings.Contains(got, "'$n'") {
		t.Errorf("%s = %q, want the typed $n printed literally", context, got)
	}
	if strings.Contains(got, "'"+strings.Split(context, "'")[0]) && strings.Contains(got, "'Actor'") {
		t.Errorf("%s = %q, typed $n was substituted", context, got)
	}
}

func TestCmdSayTypedDollarPrintsLiterally(t *testing.T) {
	actor, peer := dollarPair(t, "Actor", "Peer")
	if err := cmdSay(actor, []string{"$n"}); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, peer), "peer say block")
}

func TestCmdTellTypedDollarPrintsLiterally(t *testing.T) {
	actor, peer := dollarPair(t, "Actor", "Target")
	if err := cmdTell(actor, []string{"Target", "$n"}); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, peer), "victim tell block")
}

func TestCmdShoutTypedDollarPrintsLiterally(t *testing.T) {
	actor, peer := dollarPair(t, "Actor", "Peer")
	if err := cmdShout(actor, []string{"$n"}); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, peer), "peer shout block")
}

func TestCmdGossipTypedDollarPrintsLiterally(t *testing.T) {
	actor, peer := dollarPair(t, "Actor", "Peer")
	if err := cmdGossip(actor, []string{"$n"}); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, peer), "peer gossip block")
}

func TestCmdWhisperTypedDollarPrintsLiterally(t *testing.T) {
	actor, peer := dollarPair(t, "Actor", "Target")
	if err := cmdWhisper(actor, []string{"Target", "$n"}); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, peer), "victim whisper block")
}

func TestCmdAskTypedDollarPrintsLiterally(t *testing.T) {
	actor, peer := dollarPair(t, "Actor", "Target")
	if err := cmdAsk(actor, []string{"Target", "$n"}); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, peer), "victim ask block")
}

func TestCmdGtellTypedDollarPrintsLiterally(t *testing.T) {
	m := makeTestManager(t)
	leader := makeCommandTestSession(t, m, "Leader", 1, 1001)
	member := makeCommandTestSession(t, m, "Member", 1, 1001)
	leader.player.SetInGroup(true)
	leader.player.SetAffect(game.AffGroup, true)
	member.player.SetInGroup(true)
	member.player.SetAffect(game.AffGroup, true)
	member.player.SetFollowing(leader.player.Name)
	registerInWorld(t, leader)
	registerInWorld(t, member)

	if err := cmdGtellText(leader, "$n"); err != nil {
		t.Fatal(err)
	}
	assertLiteralDollarN(t, readMsgText(t, member), "member gtell block")
}
