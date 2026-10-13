package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func deferredMobFixture(t *testing.T, dark bool) (*Manager, *Session, *game.MobInstance) {
	t.Helper()
	flags := []string{"0", "0", "0", "0"}
	if dark {
		flags[0] = "1"
	}
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Pending room", Description: "A room for extraction proofs.", Flags: flags}},
		Mobs: []parser.Mob{{
			VNum: 3001, Keywords: "guard pending", ShortDesc: "a pending test guard",
			LongDesc: "A PENDING-EXTCHAR-GUARD stands here.\n", Level: 1,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	actor := makeTestSession(t, m, "Extractor", 1001, true)
	actor.player.SetLevel(40)
	registerTestSession(t, m, actor, actor.player.Name)
	mob, err := w.SpawnMobQuiet(3001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetGold(17)
	return m, actor, mob
}

func assertDeferredMobVisibleToTargetingButHiddenFromLook(t *testing.T, m *Manager, actor *Session, mob *game.MobInstance, dark bool) {
	t.Helper()
	w := m.world
	if !mob.HasMobFlag(game.MobFlagExtract) {
		t.Fatal("mob is not marked for extraction")
	}
	if got, ok := w.GetMobByID(mob.GetID()); !ok || got != mob {
		t.Fatal("pending mob disappeared from active lookup before heartbeat")
	}
	if target, ok := w.ResolveCharInRoom(actor.player, "guard"); !ok || target.Mob != mob {
		t.Fatal("pending mob disappeared from room targeting before heartbeat")
	}
	if ref, ok := game.NewWorldScriptableAdapter(w).InWorldMob(3001); !ok || !ref.NPC || ref.ID != mob.GetID() {
		t.Fatal("inworld did not find pending mob before heartbeat")
	}
	if err := cmdLook(actor, nil); err != nil {
		t.Fatal(err)
	}
	look := strings.Join(drainSessionText(t, actor), "")
	if strings.Contains(look, "PENDING-EXTCHAR-GUARD") || strings.Contains(look, "pending test guard") {
		t.Fatalf("look showed pending mob: %q", look)
	}
	if dark && strings.Contains(look, "moving around nearby") {
		t.Fatalf("dark look reported pending mob movement: %q", look)
	}
}

func assertDeferredMobRemovedAtHeartbeat(t *testing.T, m *Manager, actor *Session, mob *game.MobInstance) {
	t.Helper()
	m.world.ExtractPendingChars()
	if _, ok := m.world.GetMobByID(mob.GetID()); ok {
		t.Fatal("heartbeat extraction left mob active")
	}
	if _, ok := m.world.ResolveCharInRoom(actor.player, "guard"); ok {
		t.Fatal("heartbeat extraction left mob targetable")
	}
	if _, ok := game.NewWorldScriptableAdapter(m.world).InWorldMob(3001); ok {
		t.Fatal("inworld found mob after heartbeat extraction")
	}
}

func TestDeferredMobExtcharRetainsListsUntilHeartbeat(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "lit look"
		if dark {
			name = "dark look"
		}
		t.Run(name, func(t *testing.T) {
			m, actor, mob := deferredMobFixture(t, dark)
			dir := t.TempDir()
			script := `function pulse()
  local target = inworld("mob", 3001)
  assert(target ~= nil, "mob missing before extchar")
  target.gold = 4242
  extchar(target)
  assert(inworld("mob", 3001) ~= nil, "extchar removed mob before heartbeat")
  return true
end
`
			if err := os.WriteFile(filepath.Join(dir, "extchar.lua"), []byte(script), 0o600); err != nil {
				t.Fatal(err)
			}
			adapter := game.NewWorldScriptableAdapter(m.world)
			engine := scripting.NewEngine(dir, adapter)
			defer engine.Close()
			ref := scripting.CharRef{NPC: true, ID: mob.GetID()}
			if _, err := engine.RunScript(&scripting.ScriptContext{MeRef: &ref, RoomVNum: 1001}, "extchar.lua", "pulse"); err != nil {
				t.Fatal(err)
			}
			if mob.GetGold() != 17 {
				t.Fatalf("Lua write-back changed extracted mob gold to %d", mob.GetGold())
			}
			assertDeferredMobVisibleToTargetingButHiddenFromLook(t, m, actor, mob, dark)
			assertDeferredMobRemovedAtHeartbeat(t, m, actor, mob)
		})
	}
}

func TestPurgeMobRetainsListsUntilHeartbeat(t *testing.T) {
	m, actor, mob := deferredMobFixture(t, false)
	if err := executeCommand(actor, "purge", []string{"guard"}, false); err != nil {
		t.Fatal(err)
	}
	assertDeferredMobVisibleToTargetingButHiddenFromLook(t, m, actor, mob, false)
	assertDeferredMobRemovedAtHeartbeat(t, m, actor, mob)
}

func TestPurgeRoomDrainsPendingMobsImmediately(t *testing.T) {
	m, actor, mob := deferredMobFixture(t, false)
	if err := executeCommand(actor, "purge", nil, false); err != nil {
		t.Fatal(err)
	}
	// C's no-argument do_purge calls extract_pending_chars after marking each
	// room NPC and before deleting room objects (act.wizard.c:1463-1473).
	if _, ok := m.world.GetMobByID(mob.GetID()); ok {
		t.Fatal("room purge left NPC registered after its explicit pending drain")
	}
	if mob.GetRoom() != game.LoadRoomNowhere {
		t.Fatalf("room-purged mob room = %d, want NOWHERE", mob.GetRoom())
	}
}

func TestPurgeRoomReturnsPendingPlayerToHeartbeatMenu(t *testing.T) {
	queuePlayer := func(t *testing.T, m *Manager) *Session {
		t.Helper()
		victim := makeTestSession(t, m, "Queuedplayer", 1001, true)
		victim.player = game.NewPlayer(2, "Queuedplayer", 1001)
		registerTestSession(t, m, victim, "Queuedplayer")
		m.world.QueuePlayerExtraction(victim.player)
		return victim
	}

	// Capture the player-facing transition from the heartbeat path, which
	// routes the world's extracted-player result back to the owning session.
	heartbeatManager, _, _ := deferredMobFixture(t, false)
	heartbeatVictim := queuePlayer(t, heartbeatManager)
	heartbeatManager.ExtractPendingChars()
	if !heartbeatVictim.menuActive || heartbeatVictim.menuStage != "menu" {
		t.Fatalf("heartbeat transition = (%v, %q), want active main menu", heartbeatVictim.menuActive, heartbeatVictim.menuStage)
	}
	heartbeatMenu := renderedOutput(heartbeatVictim)
	if heartbeatMenu == "" {
		t.Fatal("heartbeat extraction did not send the main menu")
	}

	// The room-wide purge must produce that same menu transition for a player
	// already queued when it calls extract_pending_chars.
	purgeManager, actor, _ := deferredMobFixture(t, false)
	purgeVictim := queuePlayer(t, purgeManager)
	if err := executeCommand(actor, "purge", nil, false); err != nil {
		t.Fatal(err)
	}
	if !purgeVictim.menuActive || purgeVictim.menuStage != "menu" {
		t.Fatalf("purge transition = (%v, %q), want active main menu", purgeVictim.menuActive, purgeVictim.menuStage)
	}
	if got := renderedOutput(purgeVictim); !strings.HasSuffix(got, heartbeatMenu) {
		t.Fatalf("purge output did not end with heartbeat menu; got %q, want suffix %q", got, heartbeatMenu)
	}
}

func TestCombatMobSpecialSkipsPendingExtraction(t *testing.T) {
	m, _, mob := deferredMobFixture(t, false)
	m.world.StopAITicker()
	const specialName = "test_deferred_extract_special"
	oldName, hadName := game.MobSpecAssign[3001]
	oldEntry, hadEntry := game.SpecRegistry[specialName]
	game.MobSpecAssign[3001] = specialName
	calls := 0
	game.RegisterSpec(specialName, func(*game.World, *game.Player, *game.MobInstance, string, string) bool {
		calls++
		return false
	})
	t.Cleanup(func() {
		if hadName {
			game.MobSpecAssign[3001] = oldName
		} else {
			delete(game.MobSpecAssign, 3001)
		}
		if hadEntry {
			game.SpecRegistry[specialName] = oldEntry
		} else {
			delete(game.SpecRegistry, specialName)
		}
	})
	mob.SetMobFlag(game.MobFlagSpec)
	m.SetMobSpecialFunc()
	if m.combatEngine.MobSpecialFunc == nil || m.combatEngine.MobSpecialFunc(mob) || calls != 1 {
		t.Fatalf("unflagged special call count=%d", calls)
	}
	m.world.ExtractMob(mob)
	if m.combatEngine.MobSpecialFunc(mob) || calls != 1 {
		t.Fatalf("pending mob ran MOB_SPEC: calls=%d", calls)
	}
	m.world.ExtractPendingChars()
}

// TestPlayerExtcharDropsPossessionsAndReturnsToMenuAtHeartbeat proves
// lua_extchar on a player (scripts.c:480-494): extract_char only flags them
// (handler.c:1194-1209), and the next extract_pending_chars drops what they
// carry then wear onto the floor, saves them and sends the menu
// (handler.c:1131-1175).
func TestPlayerExtcharDropsPossessionsAndReturnsToMenuAtHeartbeat(t *testing.T) {
	obj := func(vnum int, short string) parser.Obj {
		return parser.Obj{
			VNum: vnum, Keywords: "token", ShortDesc: short,
			WearFlags: [4]int{(1 << 0) | (1 << 13)}, // TAKE + WEAR_HOLD
		}
	}
	m := newTestManager(t, mustWorld(t, &parser.World{
		// A preceding room gives 1001 rnum 1: run_script writes ch back only
		// when ch->in_room > 0 (scripts.c:1805), so the flag gate is what
		// keeps the script's gold change from landing.
		Rooms: []parser.Room{{VNum: 1000, Name: "Rnum zero", Zone: 1}, {VNum: 1001, Name: "Extchar room", Zone: 1}},
		Objs:  []parser.Obj{obj(7001, "a worn token"), obj(7002, "a carried token")},
	}), nil)
	w := m.world
	victim := makeTestSession(t, m, "Scriptvictim", 1001, true)
	victim.player = game.NewPlayer(2, "Scriptvictim", 1001)
	registerTestSession(t, m, victim, "Scriptvictim")
	victim.player.SetGold(17)

	worn, err := w.SpawnObject(7001, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := victim.player.Equipment.Equip(worn, victim.player.Inventory); err != nil {
		t.Fatal(err)
	}
	carried, err := w.SpawnObject(7002, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(carried, victim.player); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	script := `function oncmd()
  ch.gold = 4242
  extchar(ch)
  return true
end
`
	if err := os.MkdirAll(filepath.Join(dir, "room"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "room", "extchar_pc.lua"), []byte(script), 0o600); err != nil {
		t.Fatal(err)
	}
	engine := scripting.NewEngine(dir, game.NewWorldScriptableAdapter(w))
	defer engine.Close()
	ref := scripting.CharRef{ID: victim.player.ID}
	if _, err := engine.RunScript(&scripting.ScriptContext{ChRef: &ref, OwnerType: "room", RoomVNum: 1001}, "extchar_pc.lua", "oncmd"); err != nil {
		t.Fatal(err)
	}

	// Pending window: flagged, still in the world, still holding everything,
	// and run_script's write-back skips the flagged ch (scripts.c:1985).
	if got := w.GetPlayerByID(victim.player.ID); got != victim.player {
		t.Fatal("extchar removed the player before the heartbeat")
	}
	if victim.player.GetGold() != 17 {
		t.Fatalf("Lua write-back changed extracted player gold to %d", victim.player.GetGold())
	}
	if len(w.GetItemsInRoom(1001)) != 0 {
		t.Fatal("extchar dropped possessions before the heartbeat")
	}
	if victim.menuActive {
		t.Fatal("extchar sent the menu before the heartbeat")
	}

	m.ExtractPendingChars()
	if !victim.menuActive || victim.menuStage != "menu" {
		t.Fatalf("heartbeat transition = (%v, %q), want active main menu", victim.menuActive, victim.menuStage)
	}
	// Carried list first, then worn slots; obj_to_room prepends each.
	floor := w.GetItemsInRoom(1001)
	if len(floor) != 2 || floor[0] != worn || floor[1] != carried {
		t.Fatalf("floor after extraction = %v, want [worn carried]", floor)
	}
}
