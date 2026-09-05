package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type werewolfTestActors struct {
	w       *World
	mob     *MobInstance
	victim  *Player
	peer    *Player
	remote  *Player
	sleepy  *Player
	message map[string]string
}

func newWerewolfTestActors(t *testing.T) werewolfTestActors {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 5510, Name: "Werewolf Room", Zone: 55},
			{VNum: 5511, Name: "Werewolf Hall", Zone: 55},
			{VNum: 5512, Name: "Other Zone", Zone: 56},
		},
		Mobs: []parser.Mob{{
			VNum:      5510,
			Keywords:  "beast werewolf",
			ShortDesc: "the werewolf",
			Level:     8,
			Sex:       1,
			HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 40},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	message := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { message[name] += string(msg) }
	newPlayer := func(id int, name string, room int) *Player {
		p := NewPlayer(id, name, room)
		p.SetPosition(combat.PosStanding)
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer %s: %v", name, err)
		}
		return p
	}
	victim := newPlayer(1, "WolfVictim", 5510)
	peer := newPlayer(2, "WolfWitness", 5510)
	remote := newPlayer(3, "WolfDistant", 5511)
	sleepy := newPlayer(4, "WolfSleeper", 5511)
	sleepy.SetPosition(combat.PosSleeping)

	mob, err := w.SpawnMob(5510, 5510)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetLevel(8)
	mob.SetMaxHP(1000)
	mob.SetHealth(1000)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(victim.GetName())
	victim.SetFighting(mob.GetName())
	victim.SetHealth(1000)
	victim.SetMaxHP(1000)
	victim.SetMove(20)
	for name := range message {
		message[name] = ""
	}
	return werewolfTestActors{
		w:       w,
		mob:     mob,
		victim:  victim,
		peer:    peer,
		remote:  remote,
		sleepy:  sleepy,
		message: message,
	}
}

func setWerewolfNumber(t *testing.T, values ...int) {
	t.Helper()
	index := 0
	previous := werewolfNumber
	werewolfNumber = func(from, to int) int {
		if from == 0 && to != 9 && to != 3 {
			t.Fatalf("unexpected werewolf draw range (0,%d)", to)
		}
		if index >= len(values) {
			t.Fatalf("unexpected werewolf draw after %d pinned values", index)
		}
		value := values[index]
		index++
		return value
	}
	t.Cleanup(func() { werewolfNumber = previous })
}

func TestSpecWerewolf_EntryGates(t *testing.T) {
	a := newWerewolfTestActors(t)
	setWerewolfNumber(t, 0, 0)
	a.mob.SetFighting("")
	if specWerewolf(a.w, a.victim, a.mob, "look", "werewolf") {
		t.Fatal("non-empty command should fall through")
	}
	if specWerewolf(a.w, nil, a.mob, "", "") {
		t.Fatal("non-fighting werewolf should fall through")
	}
	a.mob.SetFighting(a.victim.GetName())
	a.mob.SetHealth(0)
	if specWerewolf(a.w, nil, a.mob, "", "") {
		t.Fatal("non-positive HP werewolf should fall through")
	}
	if got := strings.Join([]string{a.message[a.victim.Name], a.message[a.peer.Name], a.message[a.remote.Name]}, ""); got != "" {
		t.Fatalf("entry-gate output = %q, want empty", got)
	}
}

func TestSpecWerewolf_HowlAudience(t *testing.T) {
	a := newWerewolfTestActors(t)
	setWerewolfNumber(t, 0, 1)

	if !specWerewolf(a.w, nil, a.mob, "", "") {
		t.Fatal("fighting werewolf should handle the pulse")
	}
	wantRoom := "The werewolf looks up and lets out a long, fierce howl.\r\n"
	if got := a.message[a.victim.Name]; got != wantRoom {
		t.Fatalf("victim howl = %q, want %q", got, wantRoom)
	}
	if got := a.message[a.peer.Name]; got != wantRoom {
		t.Fatalf("peer howl = %q, want %q", got, wantRoom)
	}
	if got := a.message[a.remote.Name]; got != "You hear a loud howling in the distance." {
		t.Fatalf("remote howl = %q, want exact send_to_zone message", got)
	}
	if got := a.message[a.sleepy.Name]; got != "" {
		t.Fatalf("sleeping remote howl = %q, want empty", got)
	}
}

func TestSpecWerewolf_BiteAudienceAndMoveState(t *testing.T) {
	a := newWerewolfTestActors(t)
	setWerewolfNumber(t, 1, 0)

	if !specWerewolf(a.w, nil, a.mob, "", "") {
		t.Fatal("fighting werewolf should handle the bite pulse")
	}
	wantVictim := "The werewolf tears into your leg with his huge fangs!\r\n"
	wantPeer := "The werewolf rips apart WolfVictim's leg with his fangs!\r\n"
	if got := a.message[a.victim.Name]; !strings.Contains(got, wantVictim) {
		t.Fatalf("victim bite = %q, missing %q", got, wantVictim)
	}
	if got := a.message[a.peer.Name]; !strings.Contains(got, wantPeer) {
		t.Fatalf("peer bite = %q, missing %q", got, wantPeer)
	}
	if got := a.message[a.remote.Name]; got != "" {
		t.Fatalf("remote bite = %q, want empty", got)
	}
	if got := a.victim.GetMove(); got != 8 {
		t.Fatalf("victim move = %d, want 8 after level*1.5 reduction", got)
	}
}

func TestSpecWerewolf_BiteMoveFloorsAtZero(t *testing.T) {
	a := newWerewolfTestActors(t)
	a.victim.SetMove(1)
	setWerewolfNumber(t, 1, 0)

	if !specWerewolf(a.w, nil, a.mob, "", "") {
		t.Fatal("fighting werewolf should handle the bite pulse")
	}
	if got := a.victim.GetMove(); got != 0 {
		t.Fatalf("victim move = %d, want floor at zero", got)
	}
}
