package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

// TestSpecGuildGuard_GuildInfoMatchesC pins every GuildInfo row to C's
// guild_info (src/class.c:277-290) and proves each one through the guard.
// C writes an SCMD_* constant in the third column, the one-based cmd_info
// index guild_guard compares at src/spec_procs.c:599 (src/interpreter.h:93-98,
// src/interpreter.c:311-319); GuildInfo holds that constant minus 1. The
// rows are spelled as C spells them, so a port that stores the wrong number
// fails both halves.
func TestSpecGuildGuard_GuildInfoMatchesC(t *testing.T) {
	cRows := []struct {
		class int
		room  int
		scmd  int    // third column as written in src/class.c
		cmd   string // the direction that SCMD_* names
	}{
		{ClassMageUser, 8014, 1, "north"}, // SCMD_NORTH
		{ClassThief, 8028, 1, "north"},    // SCMD_NORTH
		{ClassCleric, 8027, 3, "south"},   // SCMD_SOUTH
		{ClassWarrior, 8015, 1, "north"},  // SCMD_NORTH
		{ClassPsionic, 8518, 4, "west"},   // SCMD_WEST
		{ClassNinja, 8525, 3, "south"},    // SCMD_SOUTH
	}
	if len(GuildInfo) != len(cRows)+1 {
		t.Fatalf("GuildInfo has %d rows, want %d plus the sentinel", len(GuildInfo), len(cRows)+1)
	}
	if got := GuildInfo[len(cRows)]; got != (GuildInfoEntry{-1, -1, -1}) {
		t.Errorf("GuildInfo sentinel = %+v, want {-1 -1 -1}", got)
	}

	w, player, lastMsg := newSpecProcTestWorld(t)
	player.SetLevel(1)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	for i, c := range cRows {
		row := GuildInfo[i]
		if row.Class != c.class || row.Room != c.room || row.Direction != c.scmd-1 {
			t.Errorf("GuildInfo[%d] = {%d, %d, %d}, want C's {%d, %d, SCMD %d} stored as direction %d",
				i, row.Class, row.Room, row.Direction, c.class, c.room, c.scmd, c.scmd-1)
		}

		// A base class the row does not authorize, in the row's own room.
		wrongClass := ClassWarrior
		if row.Class == ClassWarrior {
			wrongClass = ClassThief
		}
		player.Class = wrongClass
		player.SetRoom(c.room)
		mob.SetRoom(c.room)

		if !specGuildGuard(w, player, mob, c.cmd, "") {
			t.Errorf("row %d: wrong-class %s in %d should be blocked", i, c.cmd, c.room)
		}
		if got := lastMsg(); !strings.Contains(got, "The guard humiliates you, and blocks your way.\r\n") {
			t.Errorf("row %d: %s block = %q, want C victim line", i, c.cmd, got)
		}
		for _, other := range dirs {
			if other == c.cmd {
				continue
			}
			if specGuildGuard(w, player, mob, other, "") {
				t.Errorf("row %d: wrong-class %s in %d should not be blocked by this row", i, other, c.room)
			}
			if got := lastMsg(); got != "" {
				t.Errorf("row %d: %s emitted bytes: %q", i, other, got)
			}
		}
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
	// The dispatcher resolves the typed word before it calls a special and hands
	// it CMD_NAME (src/interpreter.c:905-949), so an unresolved abbreviation
	// never reaches this helper; only the full direction name does.
	if specGuildGuard(w, player, mob, "n", "") {
		t.Fatal("an unresolved abbreviation must not reach the guard")
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
	mob.SetFightingBody(player)
	player.SetFightingBody(mob)
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
