package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newTrollTestWorld(t *testing.T) (*World, *MobInstance, *Player, *Player, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 10049, Name: "Troll Arena", Zone: 100}},
		Mobs: []parser.Mob{{
			VNum:      10029,
			Keywords:  "targ troll",
			ShortDesc: "Targ",
			Level:     9,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "TrollActor", 10049)
	actor.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	peer := NewPlayer(2, "TrollPeer", 10049)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	mob, err := w.SpawnMob(10029, 10049)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	return w, mob, actor, peer, messages
}

func clearTrollMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecTroll_EntryGates(t *testing.T) {
	w, mob, actor, _, messages := newTrollTestWorld(t)
	mob.SetHealth(1)
	clearTrollMessages(messages)

	if specTroll(w, actor, mob, "look", "") {
		t.Fatal("non-empty command should fall through")
	}
	mob.SetPosition(combat.PosSleeping)
	if specTroll(w, nil, mob, "", "") {
		t.Fatal("sleeping mob should fall through")
	}
	mob.SetPosition(combat.PosStanding)
	mob.SetHealth(0)
	if specTroll(w, nil, mob, "", "") {
		t.Fatal("non-positive HP mob should fall through")
	}
	if got := messages[actor.Name] + messages["TrollPeer"]; got != "" {
		t.Fatalf("entry-gate output = %q, want empty", got)
	}
}

func TestSpecTroll_InjuredIdleGlowUsesCNumberArm(t *testing.T) {
	w, mob, actor, peer, messages := newTrollTestWorld(t)
	clearTrollMessages(messages)
	mob.SetHealth(10)
	mob.SetMaxHP(100)
	mob.SetFighting("")

	previous := trollNumber
	trollNumber = func(from, to int) int {
		if from != 0 || to != 20 {
			t.Fatalf("troll draw = (%d,%d), want (0,20)", from, to)
		}
		return 0
	}
	t.Cleanup(func() { trollNumber = previous })

	if !specTroll(w, nil, mob, "", "") {
		t.Fatal("injured idle troll should handle the successful regen arm")
	}
	if got, want := mob.GetHP(), 28; got != want {
		t.Fatalf("regen HP = %d, want %d", got, want)
	}
	if got, want := messages[actor.Name], "Targ's wounds glow brightly for a moment, then disappear!\r\n"; got != want {
		t.Fatalf("actor room Act = %q, want %q", got, want)
	}
	if got, want := messages[peer.Name], messages[actor.Name]; got != want {
		t.Fatalf("peer room Act = %q, want %q", got, want)
	}
}

func TestSpecTroll_FightingUsesCombatNumberArm(t *testing.T) {
	w, mob, actor, peer, messages := newTrollTestWorld(t)
	clearTrollMessages(messages)
	mob.SetHealth(50)
	mob.SetMaxHP(100)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(actor.Name)
	actor.SetPosition(combat.PosFighting)
	actor.SetFighting(mob.GetName())

	previous := trollNumber
	trollNumber = func(from, to int) int {
		if from != 0 || to != 10 {
			t.Fatalf("troll draw = (%d,%d), want (0,10)", from, to)
		}
		return 0
	}
	t.Cleanup(func() { trollNumber = previous })

	if !specTroll(w, nil, mob, "", "") {
		t.Fatal("fighting troll should handle the successful regen arm")
	}
	if got, want := mob.GetHP(), 68; got != want {
		t.Fatalf("fighting regen HP = %d, want %d", got, want)
	}
	if got, want := messages[actor.Name], "Targ's wounds glow brightly for a moment, then disappear!\r\n"; got != want {
		t.Fatalf("actor fighting Act = %q, want %q", got, want)
	}
	if got, want := messages[peer.Name], messages[actor.Name]; got != want {
		t.Fatalf("peer fighting Act = %q, want %q", got, want)
	}
}
