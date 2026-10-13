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
