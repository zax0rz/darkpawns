package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// The selected game-level proof cannot execute the command's RawKill tail.
// Exercise that tail with the production world callbacks, not no-op death hooks.
// C: new_cmds.c:1155-1175; fight.c:534-582; handler.c:1221-1265.
func TestCmdSpike_PlayerRawKillContract(t *testing.T) {
	s := newSkillCommandSession(t)
	t.Cleanup(s.world.StopAITicker)
	s.player.SetLevel(50) // C short-circuits the success roll for a weaker victim.
	s.player.PKs = 7
	victim := game.NewPlayer(2, "Wolf", 1001)
	victim.SetLevel(5)
	victim.Deaths = 3
	victim.SetAffect(game.AffWerewolf, true)
	victim.SetPlrFlag(game.PlrWerewolf, true)
	victim.SetPlrFlag(game.PlrVampire, true)
	victim.SetFightingBody(s.player)
	if err := s.world.AddPlayer(victim); err != nil {
		t.Fatal(err)
	}
	weapon := &game.ObjectInstance{Prototype: &parser.Obj{
		VNum: 1, Keywords: "spike weapon", ShortDesc: "a sharp spike",
		TypeFlag: 5, WearFlags: [4]int{1 << 13}, Values: [4]int{0, 1, 4, 3},
	}}
	s.player.Inventory.Items = append(s.player.Inventory.Items, weapon)
	if err := s.player.Equipment.Equip(weapon, s.player.Inventory); err != nil {
		t.Fatal(err)
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(s.world.WireCombatCallbacks())

	if err := CmdSpike(s, []string{"Wolf"}); err != nil {
		t.Fatal(err)
	}
	if s.player.PKs != 8 || victim.Deaths != 4 {
		t.Errorf("PKs/deaths = %d/%d, want 8/4", s.player.PKs, victim.Deaths)
	}
	if victim.HasPLRFlag(game.PlrWerewolf) || victim.HasPLRFlag(game.PlrVampire) {
		t.Error("spike must clear both player nightbreed flags")
	}
	if victim.GetFighting() != "" || !victim.HasPLRFlag(game.PlrExtract) {
		t.Errorf("RawKill lifecycle: fighting=%q extract=%t, want stopped and queued", victim.GetFighting(), victim.HasPLRFlag(game.PlrExtract))
	}
	// raw_kill clears AFF_WEREWOLF before making a corpse (fight.c:554-555).
	if victim.IsAffected(game.AffWerewolf) {
		t.Error("RawKill left AFF_WEREWOLF set; C clears it before extraction")
	}
	// C raw_kill calls make_corpse before extract_char. It does not set HP or
	// POS_DEAD: the corpse and deferred extraction are this kill's contract.
	if got := len(s.world.GetItemsInRoom(1001)); got != 1 {
		t.Errorf("RawKill room objects = %d, want one corpse before extraction", got)
	}
	extracted := s.world.ExtractPendingPlayers()
	if len(extracted) != 1 || extracted[0] != victim {
		t.Errorf("extraction pass = %v, want exactly Wolf", extracted)
	}
	if _, present := s.world.GetPlayer(victim.Name); present {
		t.Error("RawKill victim remains in the world after extraction")
	}
}
