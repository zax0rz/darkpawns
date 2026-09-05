package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newQuanLoTestWorld(t *testing.T) (*World, *MobInstance, *Player, *Player, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 19424, Name: "Shaolin Hall", Zone: 194},
			{VNum: 10049, Name: "Remote Room", Zone: 100},
			{VNum: 10050, Name: "Writing Room", Zone: 100},
			{VNum: 10051, Name: "Soundproof Room", Zone: 100, Flags: []string{"32", "0", "0", "0"}},
		},
		Mobs: []parser.Mob{{
			VNum:      19405,
			Keywords:  "master shaolin quan lo",
			ShortDesc: "Quan Lo",
			Level:     24,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "QuanActor", 19424)
	actor.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	peer := NewPlayer(2, "QuanPeer", 10049)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	mob, err := w.SpawnMob(19405, 19424)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	clearQuanLoMessages(messages)
	return w, mob, actor, peer, messages
}

func clearQuanLoMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecQuanLo_FleeAliasesUseGlobalMobGossip(t *testing.T) {
	for _, command := range []string{"flee", "retreat", "escape"} {
		t.Run(command, func(t *testing.T) {
			w, mob, actor, peer, messages := newQuanLoTestWorld(t)
			if specQuanLo(w, actor, mob, command, "") {
				t.Fatal("quan_lo should fall through after gossip")
			}
			want := "Quan Lo gossips, 'What was that, QuanActor? This is not a shawade. Try it again. This time with fewing.'\r\n"
			if got := messages[actor.Name]; got != want {
				t.Fatalf("actor gossip = %q, want %q", got, want)
			}
			if got := messages[peer.Name]; got != want {
				t.Fatalf("remote peer gossip = %q, want %q", got, want)
			}
		})
	}
}

func TestSpecQuanLo_LookAndExamineUseExactKeywords(t *testing.T) {
	w, mob, actor, peer, messages := newQuanLoTestWorld(t)
	if err := w.CharTransfer(peer.Name, false, actor.GetRoom()); err != nil {
		t.Fatalf("CharTransfer peer: %v", err)
	}
	clearQuanLoMessages(messages)

	for _, command := range []string{"look", "examine"} {
		t.Run(command, func(t *testing.T) {
			clearQuanLoMessages(messages)
			if specQuanLo(w, actor, mob, command, "master") {
				t.Fatal("quan_lo should fall through after look response")
			}
			want := "Quan Lo says, 'What is it you seek, QuanActor? Tell me and be gone.'\r\n"
			if got := messages[actor.Name]; got != want {
				t.Fatalf("actor response = %q, want %q", got, want)
			}
			if got := messages[peer.Name]; got != want {
				t.Fatalf("peer response = %q, want %q", got, want)
			}
		})
	}

	clearQuanLoMessages(messages)
	if specQuanLo(w, actor, mob, "look", "ma") {
		t.Fatal("non-matching keyword should fall through")
	}
	if got := messages[actor.Name] + messages[peer.Name]; got != "" {
		t.Fatalf("prefix-only keyword response = %q, want empty", got)
	}
}

func TestSpecQuanLo_AwakeCommandGate(t *testing.T) {
	w, mob, actor, peer, messages := newQuanLoTestWorld(t)
	mob.SetPosition(combat.PosSleeping)

	for _, command := range []string{"flee", "look", ""} {
		clearQuanLoMessages(messages)
		if specQuanLo(w, actor, mob, command, "master") {
			t.Fatalf("command %q should fall through", command)
		}
		if got := messages[actor.Name] + messages[peer.Name]; got != "" {
			t.Fatalf("sleeping command %q output = %q, want empty", command, got)
		}
	}
}

func TestMobGlobalGossipHonorsCRecipientGates(t *testing.T) {
	w, mob, actor, peer, messages := newQuanLoTestWorld(t)
	muted := NewPlayer(3, "QuanMuted", 10049)
	muted.SetPosition(combat.PosStanding)
	muted.SetPlrFlag(PrfNoGossip, true)
	if err := w.AddPlayer(muted); err != nil {
		t.Fatalf("AddPlayer muted: %v", err)
	}
	writing := NewPlayer(4, "QuanWriting", 10050)
	writing.SetPosition(combat.PosStanding)
	writing.SetPlrFlag(PlrWriting, true)
	if err := w.AddPlayer(writing); err != nil {
		t.Fatalf("AddPlayer writing: %v", err)
	}
	soundproof := NewPlayer(5, "QuanSoundproof", 10051)
	soundproof.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(soundproof); err != nil {
		t.Fatalf("AddPlayer soundproof: %v", err)
	}
	clearQuanLoMessages(messages)

	w.mobGlobalGossip(mob, "A native test message.")
	want := "Quan Lo gossips, 'A native test message.'\r\n"
	for _, player := range []*Player{actor, peer} {
		if got := messages[player.Name]; got != want {
			t.Errorf("%s gossip = %q, want %q", player.Name, got, want)
		}
	}
	for _, player := range []*Player{muted, writing, soundproof} {
		if got := messages[player.Name]; got != "" {
			t.Errorf("%s gated gossip = %q, want empty", player.Name, got)
		}
	}
}
