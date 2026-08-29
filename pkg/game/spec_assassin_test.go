package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newAssassinTestWorld(t *testing.T) (*World, *Player, *MobInstance, func() string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Assassin Room", Zone: 1},
			{VNum: 1002, Name: "Assassin Roster", Zone: 1},
		},
		Mobs: []parser.Mob{{
			VNum:      8070,
			Keywords:  "street urchin",
			ShortDesc: "a street urchin",
			Level:     1,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	var output strings.Builder
	w.MessageSink = func(_ string, msg []byte) { output.Write(msg) }
	actor := NewPlayer(1, "AssassinActor", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	roster, err := w.SpawnMob(8070, 1002)
	if err != nil {
		t.Fatalf("SpawnMob roster: %v", err)
	}
	output.Reset()
	return w, actor, roster, func() string {
		value := output.String()
		output.Reset()
		return value
	}
}

func TestSpecAssassinRoomListAndEntryGates(t *testing.T) {
	w, actor, _, lastMessage := newAssassinTestWorld(t)

	if !specAssassin(w, actor, nil, "list", "") {
		t.Fatal("list should be consumed by the room special")
	}
	if got := lastMessage(); got != "To hire an assassin: hire <assassin> <victim>\r\nAvailable assassins are:\r\n    1000 - a street urchin\r\n" {
		t.Fatalf("list output = %q", got)
	}

	for _, test := range []struct {
		name string
		arg  string
		want string
	}{
		{name: "missing assassin", arg: "nobody", want: "There is nobody called that!"},
		{name: "missing victim", arg: "street", want: "Whom do you want to assassinate?"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !specAssassin(w, actor, nil, "hire", test.arg) {
				t.Fatal("hire gate should be consumed")
			}
			if got := lastMessage(); got != test.want+"\r\n" {
				t.Fatalf("output = %q, want %q", got, test.want+"\r\n")
			}
		})
	}
	if !specAssassin(w, actor, nil, "hire", "") {
		t.Fatal("empty hire should be consumed")
	}
	if got := lastMessage(); got != "Hire who?\r\n" {
		t.Fatalf("empty hire output = %q", got)
	}
	if specAssassin(w, actor, nil, "look", "") {
		t.Fatal("unrelated command should fall through")
	}
	if specAssassin(w, nil, nil, "list", "") {
		t.Fatal("nil actor should fall through")
	}
}

func TestSpecAssassinRoomHireGatesAndSuccess(t *testing.T) {
	w, actor, roster, lastMessage := newAssassinTestWorld(t)
	victim := NewPlayer(2, "VictimTarget", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}

	actor.SetGold(0)
	if !specAssassin(w, actor, nil, "hire", "street VictimTarget") {
		t.Fatal("gold gate should be consumed")
	}
	if got := lastMessage(); got != "You don't have enough gold!\r\n" {
		t.Fatalf("gold gate output = %q", got)
	}

	actor.SetGold(1000)
	victim.SetLevel(4)
	if !specAssassin(w, actor, nil, "hire", "street VictimTarget") {
		t.Fatal("low-level victim gate should be consumed")
	}
	if got := lastMessage(); got != "We cannot lower ourselves to such easy prey.\r\n" {
		t.Fatalf("low-level gate output = %q", got)
	}

	victim.SetLevel(5)
	if !specAssassin(w, actor, nil, "hire", "street VictimTarget") {
		t.Fatal("successful hire should be consumed")
	}
	if got := lastMessage(); !strings.Contains(got, "We cannot contact you if the job succeeds or not...security, you know.\r\n") || !strings.Contains(got, "AssassinActor hires a street urchin for a job.\r\n") {
		t.Fatalf("success output = %q", got)
	}
	if actor.GetGold() != 0 {
		t.Errorf("gold after hire = %d, want 0", actor.GetGold())
	}
	hired := false
	for _, mob := range w.GetMobsInRoom(actor.GetRoom()) {
		if mob != roster && mob.GetVNum() == 8070 {
			hired = true
			if !mob.HasMobFlag(MobFlagHunter) {
				t.Error("hired assassin is missing MOB_HUNTER")
			}
			if mob.GetHunting() != victim.GetName() {
				t.Errorf("hired assassin hunting = %q, want %q", mob.GetHunting(), victim.GetName())
			}
		}
	}
	if !hired {
		t.Fatal("successful hire did not spawn an assassin in the actor's room")
	}
}

func TestSpecAssassinRejectsInvisibleVictim(t *testing.T) {
	w, actor, _, lastMessage := newAssassinTestWorld(t)
	victim := NewPlayer(2, "InvisibleTarget", 1001)
	victim.SetLevel(5)
	victim.SetAffect(affInvisible, true)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	actor.SetGold(1000)

	if !specAssassin(w, actor, nil, "hire", "street InvisibleTarget") {
		t.Fatal("invisible-victim branch should be consumed")
	}
	if got := lastMessage(); got != "Our underground doesn't know the whereabouts of the victim!\r\n" {
		t.Fatalf("invisible victim output = %q", got)
	}
	if actor.GetGold() != 1000 {
		t.Errorf("gold after invisible-victim rejection = %d, want 1000", actor.GetGold())
	}
}

func TestSpecAssassinRejectsPlayerRosterMember(t *testing.T) {
	w, actor, _, lastMessage := newAssassinTestWorld(t)
	rosterPlayer := NewPlayer(2, "RosterTarget", 1002)
	if err := w.AddPlayer(rosterPlayer); err != nil {
		t.Fatalf("AddPlayer roster player: %v", err)
	}

	if !specAssassin(w, actor, nil, "hire", "RosterTarget") {
		t.Fatal("player roster rejection should be consumed")
	}
	got := lastMessage()
	if !strings.Contains(got, "GET THE HELL OUT OF THAT ROOM, NOW !!!\r\n") || !strings.Contains(got, "You can't hire players.\r\n") {
		t.Fatalf("player roster output = %q", got)
	}
}
