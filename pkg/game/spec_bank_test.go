package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSpecBank verifies the banker spec proc moves coins between carried gold
// and the bank account, matching src/spec_procs.c SPECIAL(bank). Regression
// guard for the broken port where deposit destroyed coins and withdraw minted
// coins from nothing with no balance check (an economy exploit).
func TestSpecBank(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Bank", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 8034, Keywords: "banker", ShortDesc: "the banker"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	ch.SetGold(100)
	ch.SetBankGold(0)

	proto, _ := w.GetMobPrototype(8034)
	banker := NewMobInstance(proto, 1001)

	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	// balance with empty account
	specBank(w, ch, banker, "balance", "")
	if msg := lastMsg(); !strings.Contains(msg, "no money deposited") {
		t.Errorf("empty balance: got %q", msg)
	}

	// deposit 60: gold 100->40, bank 0->60
	specBank(w, ch, banker, "deposit", "60")
	if ch.GetGold() != 40 || ch.GetBankGold() != 60 {
		t.Fatalf("after deposit 60: gold=%d bank=%d, want 40/60", ch.GetGold(), ch.GetBankGold())
	}

	// deposit more than carried: rejected, no change
	specBank(w, ch, banker, "deposit", "1000")
	if ch.GetGold() != 40 || ch.GetBankGold() != 60 {
		t.Errorf("over-deposit should be rejected: gold=%d bank=%d", ch.GetGold(), ch.GetBankGold())
	}
	if msg := lastMsg(); !strings.Contains(msg, "don't have that many coins") {
		t.Errorf("over-deposit message: got %q", msg)
	}

	// balance now reports the bank account, not carried gold
	specBank(w, ch, banker, "balance", "")
	if msg := lastMsg(); !strings.Contains(msg, "60 coins") {
		t.Errorf("balance after deposit: got %q (want 60)", msg)
	}

	// withdraw more than banked: rejected (the exploit guard), no change
	specBank(w, ch, banker, "withdraw", "1000")
	if ch.GetGold() != 40 || ch.GetBankGold() != 60 {
		t.Fatalf("over-withdraw must be rejected (exploit): gold=%d bank=%d", ch.GetGold(), ch.GetBankGold())
	}
	if msg := lastMsg(); !strings.Contains(msg, "don't have that many coins deposited") {
		t.Errorf("over-withdraw message: got %q", msg)
	}

	// withdraw 50: gold 40->90, bank 60->10
	specBank(w, ch, banker, "withdraw", "50")
	if ch.GetGold() != 90 || ch.GetBankGold() != 10 {
		t.Fatalf("after withdraw 50: gold=%d bank=%d, want 90/10", ch.GetGold(), ch.GetBankGold())
	}

	// conservation: carried + banked unchanged by the round-trip (100 total)
	if ch.GetGold()+ch.GetBankGold() != 100 {
		t.Errorf("money not conserved: gold+bank=%d, want 100", ch.GetGold()+ch.GetBankGold())
	}
}

func TestSpecBankObjectCallContract(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Bank", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }

	actor := NewPlayer(1, "Actor", 1001)
	peer := NewPlayer(2, "Peer", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	actor.SetGold(100)

	if !specBank(w, actor, nil, "deposit", "  +10coins") {
		t.Fatal("bank object special did not handle deposit")
	}
	if got, want := messages[actor.Name], "You deposit 10 coins.\r\n"; got != want {
		t.Errorf("actor message = %q, want %q", got, want)
	}
	if got, want := messages[peer.Name], "Actor makes a bank transaction.\r\n"; got != want {
		t.Errorf("peer message = %q, want %q", got, want)
	}
	if actor.GetGold() != 90 || actor.GetBankGold() != 10 {
		t.Errorf("balances = %d/%d, want 90/10", actor.GetGold(), actor.GetBankGold())
	}
}

func TestAtoiC(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{input: "", want: 0},
		{input: "garbage", want: 0},
		{input: "  +50coins", want: 50},
		{input: "-5", want: -5},
		{input: "\t42", want: 42},
	}
	for _, tt := range tests {
		if got := atoiC(tt.input); got != tt.want {
			t.Errorf("atoiC(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
