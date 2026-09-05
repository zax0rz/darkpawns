package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newClerkTestWorld(t *testing.T) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()

	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 18228, Name: "Kir-Oshi Clerk", Zone: 182},
			{VNum: 8162, Name: "Kir Drax'in Clerk", Zone: 80},
			{VNum: 21202, Name: "Alaozar Clerk", Zone: 212},
			{VNum: 2701, Name: "Arden Clerk Test", Zone: 27},
		},
		Zones: []parser.Zone{
			{Number: 27, Name: "Arden"},
			{Number: 80, Name: "Kir Drax'in"},
			{Number: 182, Name: "Kir-Oshi"},
			{Number: 212, Name: "Alaozar"},
		},
		Mobs: []parser.Mob{{
			VNum:      18228,
			Keywords:  "clerk",
			ShortDesc: "a clerk",
			Level:     10,
			HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "ClerkActor", 18228)
	actor.SetPosition(combat.PosStanding)
	actor.Hometown = 1
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	peer := NewPlayer(2, "ClerkPeer", 18228)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	mob, err := w.SpawnMob(18228, 18228)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosStanding)
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, peer, mob, messages
}

func TestSpecClerk_EntryGates(t *testing.T) {
	w, actor, _, mob, messages := newClerkTestWorld(t)

	if specClerk(w, actor, mob, "", "") {
		t.Fatal("commandless clerk call should fall through")
	}
	actor.SetPosition(combat.PosSleeping)
	if specClerk(w, actor, mob, "list", "") {
		t.Fatal("sleeping actor should not reach clerk")
	}
	actor.SetPosition(combat.PosStanding)
	actor.SetFighting("another character")
	if specClerk(w, actor, mob, "list", "") {
		t.Fatal("fighting actor should not reach clerk")
	}
	if got := messages[actor.Name]; got != "" {
		t.Fatalf("entry-gate output = %q, want empty", got)
	}
}

func TestSpecClerk_DirectTellAndCitizenshipState(t *testing.T) {
	w, actor, peer, mob, messages := newClerkTestWorld(t)

	if !specClerk(w, actor, mob, "list", "") {
		t.Fatal("list should be consumed")
	}
	if got, want := messages[actor.Name], "A clerk tells you, 'Citizenship costs 2,000 coins.'\r\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("peer received direct tell %q", got)
	}
	messages[actor.Name] = ""

	if !specClerk(w, actor, mob, "buy", "junk") {
		t.Fatal("wrong buy argument should be consumed")
	}
	if got, want := messages[actor.Name], "A clerk tells you, 'BUY CITIZENSHIP, if you're interested.'\r\n"; got != want {
		t.Fatalf("wrong-argument output = %q, want %q", got, want)
	}
	messages[actor.Name] = ""

	actor.SetGold(1000)
	if !specClerk(w, actor, mob, "buy", "citizenship") {
		t.Fatal("insufficient-gold buy should be consumed")
	}
	if got, want := messages[actor.Name], "A clerk tells you, 'You cannot afford it!'\r\n"; got != want {
		t.Fatalf("insufficient-gold output = %q, want %q", got, want)
	}
	messages[actor.Name] = ""

	actor.SetGold(3000)
	if !specClerk(w, actor, mob, "buy", "citizenship") {
		t.Fatal("citizenship purchase should be consumed")
	}
	if got, want := messages[actor.Name], "A clerk tells you, 'You are now a citizen of Kir-Oshi.'\r\n"; got != want {
		t.Fatalf("success output = %q, want %q", got, want)
	}
	if got := actor.GetGold(); got != 1000 {
		t.Fatalf("gold after citizenship = %d, want 1000", got)
	}
	if got := actor.GetHometown(); got != 2 {
		t.Fatalf("hometown after citizenship = %d, want 2", got)
	}
	messages[actor.Name] = ""

	actor.SetGold(3000)
	if !specClerk(w, actor, mob, "buy", "citizenship") {
		t.Fatal("already-citizen buy should be consumed")
	}
	if got, want := messages[actor.Name], "A clerk tells you, 'You are already a citizen here!'\r\n"; got != want {
		t.Fatalf("already-citizen output = %q, want %q", got, want)
	}
}

func TestSpecClerk_VisibilityAndUnknownZone(t *testing.T) {
	w, actor, peer, mob, messages := newClerkTestWorld(t)

	mob.SetAffected(affBlind)
	if !specClerk(w, actor, mob, "list", "") {
		t.Fatal("blind clerk list should be consumed")
	}
	want := "A clerk exclaims, 'Who's there? I can't see you!'\r\n"
	if got := messages[actor.Name]; got != want {
		t.Fatalf("blind-clerk actor output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != want {
		t.Fatalf("blind-clerk peer output = %q, want %q", got, want)
	}

	mob.RemoveAffected(affBlind)
	actor.SetAffect(affInvisible, true)
	messages[actor.Name] = ""
	messages[peer.Name] = ""
	if !specClerk(w, actor, mob, "list", "") {
		t.Fatal("invisible actor list should be consumed")
	}
	if got := messages[actor.Name]; got != want {
		t.Fatalf("invisible-actor output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != want {
		t.Fatalf("invisible-actor peer output = %q, want %q", got, want)
	}
	actor.SetAffect(affInvisible, false)

	messages[actor.Name] = ""
	messages[peer.Name] = ""
	actor.SetRoom(2701)
	if err := w.MobTransfer(mob, 2701); err != nil {
		t.Fatalf("MobTransfer: %v", err)
	}
	if got := specClerk(w, actor, mob, "look", ""); got {
		t.Fatal("unknown command should fall through after the zone warning")
	}
	if got, want := messages[actor.Name], "default case reached in clerk special - tell a god\r\n"; got != want {
		t.Fatalf("unknown-zone output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("unknown-zone warning leaked to peer: %q", got)
	}
}

func TestSpecClerk_HometownZoneNames(t *testing.T) {
	w, actor, peer, mob, messages := newClerkTestWorld(t)

	for _, test := range []struct {
		name      string
		room      int
		oldHome   int
		wantHome  int
		wantReply string
	}{
		{name: "Kir Drax'in", room: 8162, oldHome: 2, wantHome: 1, wantReply: "Kir Drax'in"},
		{name: "Alaozar", room: 21202, oldHome: 2, wantHome: 3, wantReply: "Alaozar"},
	} {
		t.Run(test.name, func(t *testing.T) {
			actor.SetRoom(test.room)
			peer.SetRoom(test.room)
			if err := w.MobTransfer(mob, test.room); err != nil {
				t.Fatalf("MobTransfer: %v", err)
			}
			actor.Hometown = test.oldHome
			actor.SetGold(3000)
			messages[actor.Name] = ""
			messages[peer.Name] = ""

			if !specClerk(w, actor, mob, "buy", "citizenship") {
				t.Fatal("citizenship purchase should be consumed")
			}
			want := "A clerk tells you, 'You are now a citizen of " + test.wantReply + ".'\r\n"
			if got := messages[actor.Name]; got != want {
				t.Fatalf("purchase output = %q, want %q", got, want)
			}
			if got := actor.GetHometown(); got != test.wantHome {
				t.Fatalf("hometown = %d, want %d", got, test.wantHome)
			}
			if got := actor.GetGold(); got != 1000 {
				t.Fatalf("gold = %d, want 1000", got)
			}
			if got := messages[peer.Name]; got != "" {
				t.Fatalf("peer received direct tell %q", got)
			}
		})
	}
}
