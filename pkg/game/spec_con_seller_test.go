package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newConSellerTestWorld(t *testing.T, dark bool) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()
	flags := []string{"0", "0", "0", "0"}
	if dark {
		flags[0] = "1" // ROOM_DARK
	}
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 21234, Name: "Con Seller Shop", Zone: 212, Flags: flags}},
		Mobs: []parser.Mob{{
			VNum:      21246,
			Keywords:  "shadow mage shadowmage",
			ShortDesc: "the shadowmage",
			Level:     32,
			Con:       11,
			AffectFlags: []string{
				"INFRAVISION",
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "ConSellerActor", 21234)
	actor.SetPosition(combat.PosStanding)
	actor.Stats.Con = 10
	actor.SetOrigCon(14)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	peer := NewPlayer(2, "ConSellerPeer", 21234)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	mob, err := w.SpawnMob(21246, 21234)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, peer, mob, messages
}

func clearConSellerMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecConSeller_EntryGatesAndCommandSurface(t *testing.T) {
	w, actor, _, mob, messages := newConSellerTestWorld(t, false)

	if specConSeller(w, nil, mob, "", "") {
		t.Fatal("commandless call should fall through")
	}
	if specConSeller(w, actor, mob, "look", "") {
		t.Fatal("unrelated command should fall through")
	}
	mob.SetAffected(affBlind)
	if specConSeller(w, actor, mob, "look", "") {
		t.Fatal("blind seller should not intercept unrelated command")
	}
	if got := messages[actor.Name] + messages["ConSellerPeer"]; got != "" {
		t.Fatalf("entry-gate output = %q, want empty", got)
	}

	mob.RemoveAffected(affBlind)
	actor.SetPosition(combat.PosSleeping)
	if specConSeller(w, actor, mob, "list", "") {
		t.Fatal("sleeping actor should not reach seller")
	}
	actor.SetPosition(combat.PosStanding)
	actor.SetFighting("another character")
	if specConSeller(w, actor, mob, "list", "") {
		t.Fatal("fighting actor should not reach seller")
	}
}

func TestSpecConSeller_DirectTellVisibilityAndOriginalCon(t *testing.T) {
	w, actor, peer, mob, messages := newConSellerTestWorld(t, true)

	if !specConSeller(w, actor, mob, "list", "") {
		t.Fatal("list should be handled")
	}
	if got, want := messages[actor.Name], "Someone tells you, 'You can buy up to 4 points, at 400 per point.'\r\n"; got != want {
		t.Fatalf("dark-room list output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("direct list tell leaked to peer: %q", got)
	}

	clearConSellerMessages(messages)
	if !specConSeller(w, actor, mob, "buy", "con ") {
		t.Fatal("trailing-space buy should still be handled")
	}
	if got, want := messages[actor.Name], "Someone tells you, 'BUY CON, if you really want to do it.'\r\n"; got != want {
		t.Fatalf("trailing-space argument output = %q, want %q", got, want)
	}
	if actor.GetGold() != 0 {
		t.Fatalf("argument gate changed gold to %d", actor.GetGold())
	}

	if got := actor.GetOrigCon(); got != 14 {
		t.Fatalf("original constitution = %d, want 14", got)
	}
	actor.SetOrigCon(0)
	if got := actor.GetOrigCon(); got != actor.Stats.Con {
		t.Fatalf("zero original constitution fallback = %d, want current %d", got, actor.Stats.Con)
	}
	actor.SetOrigCon(14)
}

func TestSpecConSeller_BranchOrderAndSuccessAudience(t *testing.T) {
	w, actor, peer, mob, messages := newConSellerTestWorld(t, false)

	actor.SetGold(0)
	if !specConSeller(w, actor, mob, "buy", "con") {
		t.Fatal("unaffordable buy should be handled")
	}
	if got, want := messages[actor.Name], "The shadowmage tells you, 'You can't afford it!'\r\n"; got != want {
		t.Fatalf("unaffordable output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("unaffordable buy leaked to peer: %q", got)
	}

	clearConSellerMessages(messages)
	actor.SetOrigCon(actor.Stats.Con)
	actor.SetGold(400)
	if !specConSeller(w, actor, mob, "list", "") {
		t.Fatal("healthy list should be handled")
	}
	if got, want := messages[actor.Name], "The shadowmage tells you, 'You seem perfectly healthy!'\r\n"; got != want {
		t.Fatalf("healthy list output = %q, want %q", got, want)
	}
	clearConSellerMessages(messages)
	if !specConSeller(w, actor, mob, "buy", "con") {
		t.Fatal("healthy buy should be handled")
	}
	if got, want := messages[actor.Name], "The shadowmage tells you, 'You seem perfectly healthy!'\r\n"; got != want {
		t.Fatalf("healthy buy output = %q, want %q", got, want)
	}
	if got := actor.GetGold(); got != 400 {
		t.Fatalf("healthy buy gold = %d, want 400", got)
	}

	clearConSellerMessages(messages)
	actor.SetOrigCon(14)
	actor.SetGold(400)
	if !specConSeller(w, actor, mob, "buy", "con") {
		t.Fatal("successful buy should be handled")
	}
	if got, want := messages[actor.Name], "The shadowmage tells you, 'That'll be 400 coins, you should feel much better.. if you wake up.'\r\n"; got != want {
		t.Fatalf("success direct tell = %q, want %q", got, want)
	}
	if got, want := messages[peer.Name], "The shadowmage stares at ConSellerActor and mutters some arcane words.\r\nConSellerActor falls, stunned.\r\n"; got != want {
		t.Fatalf("success room audience = %q, want %q", got, want)
	}
	if got := actor.Stats.Con; got != 11 {
		t.Fatalf("success constitution = %d, want 11", got)
	}
	if got := actor.GetGold(); got != 0 {
		t.Fatalf("success gold = %d, want 0", got)
	}
	if got := actor.GetPosition(); got != combat.PosStunned {
		t.Fatalf("success position = %d, want %d", got, combat.PosStunned)
	}
	if specConSeller(w, actor, mob, "list", "") {
		t.Fatal("stunned seller invocation should fall through")
	}
}
