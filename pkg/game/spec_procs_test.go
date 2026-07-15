package game

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newSpecProcTestWorld creates a minimal world for spec proc tests.
func newSpecProcTestWorld(t *testing.T) (*World, *Player, func() string) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Dump Room", Zone: 1},
		},
		Objs: []parser.Obj{
			{
				VNum:      3001,
				Keywords:  "sword steel",
				ShortDesc: "a steel sword",
				LongDesc:  "A steel sword lies here.",
				Cost:      100,
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	lastMsg := func() string { s := out.String(); out.Reset(); return s }

	player := NewPlayer(1, "Tester", 1001)
	player.SetLevel(5)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	return w, player, lastMsg
}

var specProcTestMobVNum atomic.Int64

// newSpecProcTestMob creates a minimal, non-spec-bearing mob instance for use
// in spec proc tests. The mob is positioned in roomVNum with the given level,
// standing, and positive HP.
func newSpecProcTestMob(t *testing.T, w *World, roomVNum, level int) *MobInstance {
	t.Helper()

	vnum := int(specProcTestMobVNum.Add(1)) + 99000
	proto := &parser.Mob{
		VNum:      vnum,
		Keywords:  "testmob",
		ShortDesc: "a test mob",
		LongDesc:  "A test mob is here.",
		Level:     level,
		HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		Alignment: 0,
		Race:      1,
	}
	w.mu.Lock()
	w.mobs[vnum] = proto
	w.mu.Unlock()

	mob, err := w.SpawnMob(vnum, roomVNum)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	mob.SetLevel(level)
	mob.SetPosition(combat.PosStanding)
	mob.CurrentHP = 50
	return mob
}

// assertNotPanic wraps fn in a deferred recover and fails the test if it panics.
func assertNotPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v\n%s", r, debug.Stack())
		}
	}()
	fn()
}

// isMobSpecName returns true if name appears in MobSpecAssign.
func isMobSpecName(name string) bool {
	for _, n := range MobSpecAssign {
		if n == name {
			return true
		}
	}
	return false
}

// isObjSpecName returns true if name appears in ObjSpecAssign.
func isObjSpecName(name string) bool {
	for _, n := range ObjSpecAssign {
		if n == name {
			return true
		}
	}
	return false
}

// isRoomSpecName returns true if name appears in RoomSpecAssign.
func isRoomSpecName(name string) bool {
	for _, n := range RoomSpecAssign {
		if n == name {
			return true
		}
	}
	return false
}

// TestSpecProc_SmokeAll invokes every registered spec proc with benign input
// and asserts no panic. This catches nil-pointer crashes across the entire
// spec proc registry (DP-965).
func TestSpecProc_SmokeAll(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	names := AllSpecNames()
	for name := range names {
		t.Run(name, func(t *testing.T) {
			fn := SpecRegistry[name]
			if fn == nil {
				t.Skipf("spec %q registered but function is nil", name)
			}

			if isMobSpecName(name) {
				// Mob specs are dispatched with a non-nil me.
				assertNotPanic(t, func() {
					fn(w, player, mob, "look", "")
				})
				// Autonomous AI tick: ch is nil, cmd is empty.
				assertNotPanic(t, func() {
					fn(w, nil, mob, "", "")
				})
			}

			if isObjSpecName(name) || isRoomSpecName(name) {
				// Object and room specs are dispatched with me == nil.
				assertNotPanic(t, func() {
					fn(w, player, nil, "look", "")
				})
			}
		})
	}
}

// TestSpecGuild_Golden verifies guildmaster practice behavior.
func TestSpecGuild_Golden(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)

	// A thief who can learn backstab (skill 131, thief min level 1).
	player.Class = ClassThief
	player.SetLevel(5)
	player.SetPractices(5)
	player.Stats.Int = 13 // int_app[13].learn = 25 → full gain (thief MAXGAIN = 25)

	mob := newSpecProcTestMob(t, w, 1001, 10)

	// Non-practice commands pass through (spec_procs.c:208).
	if specGuild(w, player, mob, "look", "") {
		t.Error("specGuild should return false for non-practice commands")
	}

	// No-arg → list_skills catalog.
	specGuild(w, player, mob, "practice", "")
	if msg := lastMsg(); !strings.Contains(msg, "You know of the following skills:") {
		t.Errorf("expected catalog listing, got %q", msg)
	}

	// A skill the class cannot learn → "You do not know of that <splskl>."
	specGuild(w, player, mob, "practice", "fireball") // a mage spell
	if msg := lastMsg(); !strings.Contains(msg, "You do not know of that skill.") {
		t.Errorf("expected unknown-skill message, got %q", msg)
	}

	// Practicing a learnable skill: message, one practice consumed, skill gains
	// MIN(MAXGAIN, MAX(MINGAIN, int_app[INT].learn)) = MIN(25, 25) from 0 → 25.
	specGuild(w, player, mob, "practice", "backstab")
	if msg := lastMsg(); !strings.Contains(msg, "You practice for a while...") {
		t.Errorf("expected practice message, got %q", msg)
	}
	if player.GetPractices() != 4 {
		t.Errorf("practices = %d, want 4 (one consumed)", player.GetPractices())
	}
	if got := player.GetSkill("backstab"); got != 25 {
		t.Errorf("backstab %% = %d, want 25", got)
	}
}

// TestSpecDump_Golden verifies C-fidelity dump value awards.
func TestSpecDump_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)

	sword := NewObjectInstance(w.objs[3001], -1)
	if err := w.MoveObjectToPlayerInventory(sword, player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory failed: %v", err)
	}

	if got := specDump(w, player, nil, "look", ""); got {
		t.Error("specDump should return false for non-drop commands")
	}

	startGold := player.GetGold()
	if got := specDump(w, player, nil, "drop", "sword"); !got {
		t.Fatal("specDump should handle drop command")
	}
	if player.GetGold() != startGold+10 {
		t.Errorf("level 5 dump of 100-cost sword awarded %d gold, want %d", player.GetGold()-startGold, 10)
	}

	// Non-matching drop consumes the command but awards nothing.
	startGold = player.GetGold()
	if got := specDump(w, player, nil, "drop", "nonexistent"); !got {
		t.Error("specDump should consume drop command even with no match")
	}
	if player.GetGold() != startGold {
		t.Errorf("non-matching drop awarded %d gold, want 0", player.GetGold()-startGold)
	}

	// Low-level player receives EXP instead of gold.
	lowLevel := NewPlayer(2, "Newbie", 1001)
	lowLevel.SetLevel(2)
	lowLevel.SkillManager = engine.NewSkillManager()
	lowLevel.Inventory = NewInventory()
	lowLevel.Inventory.SetCapacity(10, 0, 10, 1)
	if err := w.AddPlayer(lowLevel); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	cheap := NewObjectInstance(w.objs[3001], -1)
	if err := w.MoveObjectToPlayerInventory(cheap, lowLevel); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory failed: %v", err)
	}
	startExp := lowLevel.GetExp()
	if got := specDump(w, lowLevel, nil, "drop", "sword"); !got {
		t.Error("specDump should handle drop for low-level player")
	}
	if lowLevel.GetExp() != startExp+10 {
		t.Errorf("level 2 dump awarded %d exp, want %d", lowLevel.GetExp()-startExp, 10)
	}
}

// TestSpecSnake_Golden verifies guard conditions for the snake poison bite.
func TestSpecSnake_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	victim := newSpecProcTestMob(t, w, 1001, 10)

	if got := specSnake(w, player, mob, "look", ""); got {
		t.Error("specSnake should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	if got := specSnake(w, nil, mob, "", ""); got {
		t.Error("specSnake should return false when not fighting")
	}

	mob.SetPosition(combat.PosFighting)
	mob.SetTarget(victim)
	assertNotPanic(t, func() {
		_ = specSnake(w, nil, mob, "", "")
	})
}

// TestSpecThief_Golden verifies thief guard conditions and steal path.
func TestSpecThief_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	if got := specThief(w, player, mob, "look", ""); got {
		t.Error("specThief should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	player.SetLevel(60)
	if got := specThief(w, nil, mob, "", ""); got {
		t.Error("specThief should return false when all players are level 50+")
	}

	player.SetLevel(10)
	player.SetPosition(combat.PosSleeping)
	player.Gold = 1000
	startGold := player.GetGold()
	// Sleeping victims are stealable; specThief returns true if it attempts a steal.
	if got := specThief(w, nil, mob, "", ""); got {
		if player.GetGold() >= startGold {
			t.Error("specThief should steal gold from a sleeping victim")
		}
	}

	player.SetPosition(combat.PosStanding)
	player.Gold = 1000
	assertNotPanic(t, func() {
		_ = specThief(w, nil, mob, "", "")
	})
}

// TestSpecMagicUser_Golden verifies magic user guard conditions and cast path.
func TestSpecMagicUser_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	if got := specMagicUser(w, player, mob, "look", ""); got {
		t.Error("specMagicUser should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	if got := specMagicUser(w, nil, mob, "", ""); got {
		t.Error("specMagicUser should return false when not fighting")
	}

	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.Name)
	player.SetFighting(mob.GetName())
	if got := specMagicUser(w, nil, mob, "", ""); !got {
		t.Error("specMagicUser should return true when fighting with a target")
	}
}

// TestSpecFighter_Golden verifies fighter guard conditions.
func TestSpecFighter_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	victim := newSpecProcTestMob(t, w, 1001, 10)

	if got := specFighter(w, player, mob, "look", ""); got {
		t.Error("specFighter should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	if got := specFighter(w, nil, mob, "", ""); got {
		t.Error("specFighter should return false when not fighting")
	}

	mob.SetPosition(combat.PosFighting)
	mob.SetTarget(victim)
	assertNotPanic(t, func() {
		_ = specFighter(w, nil, mob, "", "")
	})
}

// TestSpecCleric_Golden verifies cleric guard conditions and cast path.
func TestSpecCleric_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	if got := specCleric(w, player, mob, "look", ""); got {
		t.Error("specCleric should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	if got := specCleric(w, nil, mob, "", ""); got {
		t.Error("specCleric should return false when not fighting")
	}

	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.Name)
	player.SetFighting(mob.GetName())
	assertNotPanic(t, func() {
		_ = specCleric(w, nil, mob, "", "")
	})
}

// TestSpecCityguard_Golden verifies cityguard patrol and intervention logic.
func TestSpecCityguard_Golden(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	guard := newSpecProcTestMob(t, w, 1001, 10)

	if got := specCityguard(w, player, guard, "look", ""); got {
		t.Error("specCityguard should return false for non-empty cmd")
	}

	guard.SetPosition(combat.PosStanding)
	if got := specCityguard(w, nil, guard, "", ""); got {
		t.Error("specCityguard should return false with no outlaws or crime")
	}

	player.SetPlrFlag(PlrOutlaw, true)
	beforeHP := player.GetHP()
	_ = lastMsg()
	specCityguard(w, nil, guard, "", "")
	if player.GetHP() >= beforeHP {
		t.Error("specCityguard should damage an outlaw")
	}
	if msg := lastMsg(); !strings.Contains(msg, "OUTLAWS") {
		t.Errorf("expected outlaw warning message, got %q", msg)
	}
	player.SetPlrFlag(PlrOutlaw, false)

	evildoer := NewPlayer(3, "Evildoer", 1001)
	evildoer.SetLevel(5)
	evildoer.SetAlignment(-500)
	if err := w.AddPlayer(evildoer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	victim := NewPlayer(4, "Victim", 1001)
	victim.SetLevel(5)
	victim.SetAlignment(100)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	evildoer.SetFighting(victim.Name)
	victim.SetFighting(evildoer.Name)
	beforeHP = evildoer.GetHP()
	_ = lastMsg()
	specCityguard(w, nil, guard, "", "")
	if evildoer.GetHP() >= beforeHP {
		t.Error("specCityguard should damage an evil attacker")
	}
	if msg := lastMsg(); !strings.Contains(msg, "PROTECT THE INNOCENT") {
		t.Errorf("expected protect message, got %q", msg)
	}
}

// TestSpecDragonBreath_Golden verifies dragon breath guard conditions.
func TestSpecDragonBreath_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	victim := newSpecProcTestMob(t, w, 1001, 10)

	if got := specDragonBreath(w, player, mob, "look", ""); got {
		t.Error("specDragonBreath should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	if got := specDragonBreath(w, nil, mob, "", ""); got {
		t.Error("specDragonBreath should return false when not fighting")
	}

	mob.SetPosition(combat.PosFighting)
	mob.SetTarget(victim)
	assertNotPanic(t, func() {
		_ = specDragonBreath(w, nil, mob, "", "")
	})
}

// TestSpecJanitor_Golden verifies janitor cleanup behavior.
func TestSpecJanitor_Golden(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)

	if got := specJanitor(w, player, mob, "look", ""); got {
		t.Error("specJanitor should return false for non-empty cmd")
	}

	mob.SetPosition(combat.PosStanding)
	if got := specJanitor(w, nil, mob, "", ""); got {
		t.Error("specJanitor should return false in an empty room")
	}

	trash := NewObjectInstance(w.objs[3001], 1001)
	w.AddItemToRoom(trash, 1001)
	assertNotPanic(t, func() {
		_ = specJanitor(w, nil, mob, "", "")
	})
}

// TestSpecDumpPerformDrop verifies that specDump actually drops the item,
// removes it from inventory, and awards gold for the dumped value (DP-944).
func TestSpecDumpPerformDrop(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)

	sword := NewObjectInstance(w.objs[3001], -1)
	if err := w.MoveObjectToPlayerInventory(sword, player); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory failed: %v", err)
	}

	startGold := player.GetGold()
	if got := specDump(w, player, nil, "drop", "sword"); !got {
		t.Fatal("specDump should handle drop command")
	}

	if _, ok := player.Inventory.FindItem("sword"); ok {
		t.Error("dropped sword should no longer be in inventory")
	}

	// Cost 100 → value clamp(100/10, 1, 10) = 10 gold for a level 5 player.
	if player.GetGold() <= startGold {
		t.Errorf("player should receive gold award, got %d want > %d", player.GetGold(), startGold)
	}

	// Room should be cleaned after awarding.
	items := w.GetItemsInRoom(1001)
	for _, item := range items {
		if item == sword {
			t.Error("dropped sword should have been cleaned from room")
		}
	}
}

// TestSpecDumpNonDropPassThrough verifies that specDump returns false for
// commands other than drop, allowing normal command processing.
func TestSpecDumpNonDropPassThrough(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)

	if got := specDump(w, player, nil, "look", ""); got {
		t.Error("specDump should return false for non-drop commands")
	}
}

// TestSpecHorn_UseHornDoesNotPanic verifies that specHorn handles object-spec
// dispatch where me is nil (DP-QA#3). It must not panic and must emit correct
// actor/room messages using the player's name instead of un-interpolated $n/$P.
func TestSpecHorn_UseHornDoesNotPanic(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)

	// Object specs are dispatched with me = nil; arg carries the use target.
	if got := specHorn(w, player, nil, "use", "horn"); !got {
		t.Fatal("specHorn should handle 'use horn'")
	}

	output := lastMsg()
	if !strings.Contains(output, "You inhale deeply then blow hard!") {
		t.Errorf("output missing blow text: %q", output)
	}
	if !strings.Contains(output, "A blaring note resounds through the air.") {
		t.Errorf("output missing blare text: %q", output)
	}
	if strings.Contains(output, "$n") || strings.Contains(output, "$P") {
		t.Errorf("output contains un-interpolated tokens: %q", output)
	}
	if !strings.Contains(output, "Tester blows into a horn.") {
		t.Errorf("output missing horn use text: %q", output)
	}
	if !strings.Contains(output, "A horn lets out a blaring note...") {
		t.Errorf("output missing second blare text: %q", output)
	}
}

// TestSpecDump_ExpGoldLocked exercises the specDump award path from multiple
// goroutines. Each goroutine uses its own player in a distinct room so that
// only the ch.Exp/ch.Gold writes are exercised concurrently. Under the race
// detector this verifies those writes are protected by ch.mu (DP-QA#2).
func TestSpecDump_ExpGoldLocked(t *testing.T) {
	parsed := &parser.World{
		Objs: []parser.Obj{
			{
				VNum:      3001,
				Keywords:  "sword steel",
				ShortDesc: "a steel sword",
				LongDesc:  "A steel sword lies here.",
				Cost:      100,
			},
		},
	}
	for i := 0; i < 10; i++ {
		vnum := 1000 + i
		parsed.Rooms = append(parsed.Rooms, parser.Room{VNum: vnum, Name: "Dump Room", Zone: 1})
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			roomVNum := 1000 + idx
			player := NewPlayer(idx, fmt.Sprintf("Tester%d", idx), roomVNum)
			player.SetLevel(5)
			if err := w.AddPlayer(player); err != nil {
				t.Errorf("AddPlayer failed: %v", err)
				return
			}
			sword := NewObjectInstance(w.objs[3001], -1)
			if err := w.MoveObjectToPlayerInventory(sword, player); err != nil {
				t.Errorf("MoveObjectToPlayerInventory failed: %v", err)
				return
			}
			if !specDump(w, player, nil, "drop", "sword") {
				t.Errorf("specDump should handle drop command")
			}
		}(i)
	}
	wg.Wait()
}

// TestActMessageAudienceRouting verifies actMessage sends the correct message
// to actor, victim, and bystanders (DP-945 helper regression).
func TestActMessageAudienceRouting(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Theater", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	actor := NewPlayer(1, "Actor", 1001)
	victim := NewPlayer(2, "Victim", 1001)
	bystander := NewPlayer(3, "Bystander", 1001)
	for _, p := range []*Player{actor, victim, bystander} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer failed: %v", err)
		}
	}

	var msgs syncMap
	w.MessageSink = func(name string, msg []byte) { msgs.Store(name, string(msg)) }

	w.actMessage(
		1001, actor, victim,
		"You poke yourself.",
		"Actor pokes you!",
		"Actor pokes Victim.",
	)

	if got := msgs.Load("Actor"); !strings.Contains(got, "You poke yourself.") {
		t.Errorf("actor message = %q, want containing 'You poke yourself.'", got)
	}
	if got := msgs.Load("Victim"); !strings.Contains(got, "Actor pokes you!") {
		t.Errorf("victim message = %q, want containing 'Actor pokes you!'", got)
	}
	if got := msgs.Load("Bystander"); !strings.Contains(got, "Actor pokes Victim.") {
		t.Errorf("bystander message = %q, want containing 'Actor pokes Victim.'", got)
	}
}

// syncMap is a tiny test helper for collecting per-player messages.
type syncMap struct {
	m map[string]string
}

func (s *syncMap) Store(key, value string) {
	if s.m == nil {
		s.m = make(map[string]string)
	}
	s.m[key] += value
}

func (s *syncMap) Load(key string) string {
	if s.m == nil {
		return ""
	}
	return s.m[key]
}

// TestSpecTakeToJail_NilChNoPanic reproduces a production crash: specTakeToJail
// runs only autonomously (cmd == ""), where ch is always nil, but it sent the
// jail taunt to ch — panicking in sendToChar the moment it actually jailed a
// player. Loop enough that the 1-in-7 jail roll fires with a player present.
func TestSpecTakeToJail_NilChNoPanic(t *testing.T) {
	w, player, _ := newSpecProcTestWorld(t)
	player.SetLevel(5) // < 50, so eligible to be jailed
	mob := newSpecProcTestMob(t, w, 1001, 10)

	assertNotPanic(t, func() {
		for i := 0; i < 300; i++ {
			player.SetRoom(1001) // reset in case a prior iteration jailed them
			specTakeToJail(w, nil, mob, "", "")
		}
	})
}
