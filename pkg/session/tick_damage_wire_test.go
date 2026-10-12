package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// limits.c:503-504 -> fight.c:1064-1080 -> comm.c:2473-2475.
// Assert the terminal bytes after real combat dispatch, rather than requiring
// a line ending at the internal callback boundary.
func TestPointUpdatePoisonWireMatchesC(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	m.world.StopAITicker()
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	victim := makeTestSession(t, m, "Poisoned", 1001, true)
	peer := makeTestSession(t, m, "Peer", 1001, true)
	for _, s := range []*Session{victim, peer} {
		s.player.SetLevel(10)
		s.player.SetHP(100)
		s.player.MaxHealth = 100
		s.player.Conditions = [3]int{-1, -1, -1}
		registerTestSession(t, m, s, s.playerName)
		_ = captureWire(s)
	}
	m.WireCombatCallbacks()
	m.SetCombatMessageFunc()
	victim.player.SetAffect(game.AffPoison, true)
	m.world.PointUpdate()
	if got := captureWire(victim); got != "You feel burning poison in your blood, and suffer.\r\n" {
		t.Fatalf("poison victim wire=%q", got)
	}
	if got := captureWire(peer); got != "Poisoned looks really sick and shivers uncomfortably.\r\n" {
		t.Fatalf("poison peer wire=%q", got)
	}
}

func TestPointUpdatePoisonShopkeeperWire(t *testing.T) {
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	m, wizard, _, _, _, keeper := combatDescriptorFixture(t)
	m.world.StopAITicker()
	// PointUpdate's poison arm dispatches through the combat callbacks. Install
	// them here, as TestPointUpdatePoisonWireMatchesC does, so the claim cannot
	// depend on whatever package-level callback set another test left behind.
	m.WireCombatCallbacks()
	m.SetCombatMessageFunc()
	keeper.VNum = 8003 // C's is_shopkeeper hardcoded protector.
	keeper.SetHealth(keeper.GetMaxHP())
	keeper.SetAffected(game.AffPoison)
	if !keeper.HasAffect(game.AffPoison) {
		t.Fatal("fixture keeper is not poisoned")
	}
	_ = captureWire(wizard)
	before := keeper.GetHP()
	m.world.PointUpdate()
	if keeper.GetHP() != before {
		t.Fatalf("poisoned keeper HP=%d, want %d", keeper.GetHP(), before)
	}
	// The claim is that the switched keeper keeps its HP and sees C's line.
	// PointUpdate walks the whole world, so an unrelated act can precede it when
	// the fidelity-units harness runs this claim in a combined binary (observed:
	// "$n smiles in $s sleep.\r\n"); assert the line is present rather than that
	// it is the only byte on the wire.
	if got := captureWire(wizard); !strings.Contains(got, "Ha ha... Don't think so.\r\n") {
		t.Fatalf("keeper descriptor bytes=%q", got)
	}
}
