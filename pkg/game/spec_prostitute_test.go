package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newProstituteTestWorld(t *testing.T, dark bool) (*World, *Player, *Player, *MobInstance, map[string]string) {
	t.Helper()
	flags := []string{"0", "0", "0", "0"}
	if dark {
		flags[0] = "1" // ROOM_DARK
	}
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8021, Name: "Temple Square", Zone: 80, Flags: flags}},
		Mobs: []parser.Mob{{
			VNum:      8023,
			Keywords:  "hooker whore prostitute elf",
			ShortDesc: "a half-elven prostitute",
			Level:     1,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "Velvet", 8021)
	actor.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	peer := NewPlayer(2, "Witness", 8021)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	mob, err := w.SpawnMob(8023, 8021)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	for name := range messages {
		messages[name] = ""
	}
	return w, actor, peer, mob, messages
}

func clearProstituteMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecProstitute_EntryGatesAndFallthrough(t *testing.T) {
	w, actor, _, mob, messages := newProstituteTestWorld(t, false)

	if specProstitute(w, nil, mob, "", "") {
		t.Fatal("commandless call should fall through")
	}
	if specProstitute(w, actor, mob, "listing", "") {
		t.Fatal("non-exact command should fall through")
	}
	actor.SetPosition(combat.PosSleeping)
	if specProstitute(w, actor, mob, "list", "") {
		t.Fatal("sleeping actor should not reach prostitute")
	}
	actor.SetPosition(combat.PosStanding)
	actor.SetFighting("another character")
	if specProstitute(w, actor, mob, "buy", "") {
		t.Fatal("fighting actor should not reach prostitute")
	}
	if got := messages[actor.Name]; got != "" {
		t.Fatalf("entry-gate output = %q, want empty", got)
	}
}

func TestSpecProstitute_DirectTellsOnlyReachActor(t *testing.T) {
	w, actor, peer, mob, messages := newProstituteTestWorld(t, false)

	if !specProstitute(w, actor, mob, "list", "ignored argument") {
		t.Fatal("list should be handled")
	}
	if got, want := messages[actor.Name], "A half-elven prostitute tells you, 'For five coins, I'll show you a good time.'\r\n"; got != want {
		t.Fatalf("list output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("list tell leaked to peer: %q", got)
	}

	clearProstituteMessages(messages)
	if !specProstitute(w, actor, mob, "buy", "") {
		t.Fatal("buy should be handled")
	}
	if got, want := messages[actor.Name], "A half-elven prostitute tells you, 'I ain't for sale, just rent. Give me 5 gold for a good time.'\r\n"; got != want {
		t.Fatalf("buy output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("buy tell leaked to peer: %q", got)
	}
}

func TestSpecProstitute_InvisibleTargetUsesRoomAudience(t *testing.T) {
	w, actor, peer, mob, messages := newProstituteTestWorld(t, false)
	actor.SetAffect(affInvisible, true)

	if !specProstitute(w, actor, mob, "list", "") {
		t.Fatal("invisible list should be handled")
	}
	want := "A half-elven prostitute says, 'If I could see you, we could do business..'\r\nA half-elven prostitute winks coyly.\r\n"
	if got := messages[actor.Name]; got != want {
		t.Fatalf("actor hidden output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != want {
		t.Fatalf("peer hidden output = %q, want %q", got, want)
	}

	clearProstituteMessages(messages)
	if !specProstitute(w, actor, mob, "buy", "") {
		t.Fatal("invisible buy should be handled")
	}
	if got := messages[actor.Name]; got != want {
		t.Fatalf("actor hidden buy output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != want {
		t.Fatalf("peer hidden buy output = %q, want %q", got, want)
	}
}

func TestSpecProstitute_CanSeeUsesC_LightGate(t *testing.T) {
	w, actor, peer, mob, messages := newProstituteTestWorld(t, true)

	if !specProstitute(w, actor, mob, "list", "") {
		t.Fatal("dark-room list should be handled")
	}
	wantHidden := "A half-elven prostitute says, 'If I could see you, we could do business..'\r\nA half-elven prostitute winks coyly.\r\n"
	if got := messages[actor.Name]; got != wantHidden {
		t.Fatalf("dark-room output = %q, want %q", got, wantHidden)
	}
	if got := messages[peer.Name]; got != wantHidden {
		t.Fatalf("dark-room peer output = %q, want %q", got, wantHidden)
	}

	clearProstituteMessages(messages)
	mob.SetAffected(affInfravision)
	if !specProstitute(w, actor, mob, "list", "") {
		t.Fatal("infravision list should be handled")
	}
	if got, want := messages[actor.Name], "A half-elven prostitute tells you, 'For five coins, I'll show you a good time.'\r\n"; got != want {
		t.Fatalf("infravision output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != "" {
		t.Fatalf("infravision direct tell leaked to peer: %q", got)
	}
}

func TestSpecProstitute_BlindMobUsesHiddenAudience(t *testing.T) {
	w, actor, peer, mob, messages := newProstituteTestWorld(t, false)
	mob.SetAffected(affBlind)

	if !specProstitute(w, actor, mob, "buy", "") {
		t.Fatal("blind prostitute buy should be handled")
	}
	want := "A half-elven prostitute says, 'If I could see you, we could do business..'\r\nA half-elven prostitute winks coyly.\r\n"
	if got := messages[actor.Name]; got != want {
		t.Fatalf("blind-mob actor output = %q, want %q", got, want)
	}
	if got := messages[peer.Name]; got != want {
		t.Fatalf("blind-mob peer output = %q, want %q", got, want)
	}
}
