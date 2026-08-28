package game

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestSpecGuildGuard_GuildInfoMatchesC(t *testing.T) {
	want := []GuildInfoEntry{
		{ClassMageUser, 8014, 0},
		{ClassThief, 8028, 0},
		{ClassCleric, 8027, 1},
		{ClassWarrior, 8015, 0},
		{ClassPsionic, 8518, 3},
		{ClassNinja, 8525, 1},
		{-1, -1, -1},
	}
	if !reflect.DeepEqual(GuildInfo, want) {
		t.Fatalf("GuildInfo = %#v, want C table %#v", GuildInfo, want)
	}
}

func TestSpecGuildGuard_EntryGatesAndAudiences(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	player.SetRoom(8015)
	player.SetLevel(1)
	player.Class = ClassThief
	observer := NewPlayer(2, "Observer", 8015)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 1001, 10)
	mob.SetRoom(8015)

	var messages map[string]string
	resetMessages := func() {
		messages = make(map[string]string)
		w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	}
	resetMessages()

	if !specGuildGuard(w, player, mob, "north", "") {
		t.Fatal("unauthorized thief north should be blocked")
	}
	if got := messages[player.Name]; !strings.Contains(got, "The guard humiliates you, and blocks your way.\r\n") {
		t.Errorf("actor block = %q, want C victim line", got)
	}
	if got := messages[observer.Name]; !strings.Contains(got, "The guard humiliates Tester, and blocks his way.\r\n") {
		t.Errorf("observer block = %q, want C room line", got)
	}
	if strings.Contains(messages[player.Name], "The guard humiliates Tester") {
		t.Errorf("actor received TO_ROOM line: %q", messages[player.Name])
	}

	resetMessages()
	if !specGuildGuard(w, player, mob, "n", "") {
		t.Fatal("the north alias should resolve to C's SCMD_NORTH")
	}

	resetMessages()
	player.Class = ClassWarrior
	if specGuildGuard(w, player, mob, "north", "") {
		t.Fatal("authorized warrior north should fall through")
	}
	if len(messages) != 0 {
		t.Errorf("authorized movement emitted bytes: %#v", messages)
	}

	resetMessages()
	player.Class = ClassThief
	player.SetLevel(LVL_IMMORT)
	if specGuildGuard(w, player, mob, "north", "") {
		t.Fatal("immortal north should fall through")
	}
	if len(messages) != 0 {
		t.Errorf("immortal movement emitted bytes: %#v", messages)
	}

	resetMessages()
	player.SetLevel(1)
	player.Class = ClassPaladin
	if specGuildGuard(w, player, mob, "north", "") {
		t.Fatal("remort-only class north should fall through")
	}
	if len(messages) != 0 {
		t.Errorf("remort-only movement emitted bytes: %#v", messages)
	}
}

func TestSpecGuildGuard_FleeAliasesAndExactRoomText(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	player.SetRoom(8015)
	observer := NewPlayer(2, "Observer", 8015)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 1001, 10)
	mob.SetRoom(8015)

	for _, cmd := range []string{"flee", "escape", "retreat"} {
		var messages map[string]string
		w.MessageSink = func(name string, msg []byte) {
			if messages == nil {
				messages = make(map[string]string)
			}
			messages[name] += string(msg)
		}
		if !specGuildGuard(w, player, mob, cmd, "") {
			t.Fatalf("%s should be intercepted", cmd)
		}
		if got := messages[player.Name]; !strings.Contains(got, "You try to flee inside the guild but the guard stops you!\r\n") {
			t.Errorf("%s actor output = %q, want C flee line", cmd, got)
		}
		if got := messages[observer.Name]; !strings.Contains(got, "Tester tries to flee inside the guild but the guard block his way!\r\n") {
			t.Errorf("%s room output = %q, want C room line", cmd, got)
		}
	}
}

func TestSpecGuildGuard_BlindAndNonMoveDelegateToFighter(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	player.SetPosition(combat.PosFighting)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.Name)
	player.SetFighting(mob.GetName())
	mob.SetAffected(affBlind)

	var seed uint32
	for seed = 1; seed < 10000; seed++ {
		dprng.ResetStream(seed)
		if dprng.Number(0, 10) == 4 {
			break
		}
	}
	if seed == 10000 {
		t.Fatal("failed to find deterministic fighter delegation seed")
	}

	dprng.ResetStream(seed)
	if !specGuildGuard(w, player, mob, "north", "") {
		t.Fatal("blind guard should delegate movement to fighter")
	}

	dprng.ResetStream(seed)
	if !specGuildGuard(w, player, mob, "look", "") {
		t.Fatal("fighting guard should delegate non-movement commands to fighter")
	}
}
