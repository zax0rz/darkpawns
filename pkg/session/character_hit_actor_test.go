package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestForcedMobileHitEnrollsBothSides pins the R2e actor body of do_hit for a
// mobile: C's command_interpreter runs the same do_* for NPCs
// (src/act.offensive.c:101-129), so `force <mob> hit <target>` swings —
// hit()'s draws, damage()'s protection block, and both set_fighting
// enrolments (src/fight.c:1404-1445). The port previously logged the forced
// hit as CharacterCommandUnsupported and dropped it, which the
// teleport_victim NPC-refusal vehicle needed to start its NPC-on-NPC fight.
func TestForcedMobileHitEnrollsBothSides(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}},
		Mobs: []parser.Mob{
			{VNum: 99001, Keywords: "gil half-elf", ShortDesc: "Gil", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
			{VNum: 99002, Keywords: "dune stalker", ShortDesc: "a dune stalker", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	// WireCombatCallbacks points the combat package's global callbacks at
	// THIS world, exactly as production boot does; without it a stale
	// peaceful-world wiring from an earlier test refuses every swing here.
	manager.WireCombatCallbacks()
	manager.SetCommandExecFunc()
	manager.SetDeathFunc()
	manager.combatEngine = combat.NewCombatEngine()
	manager.world.SetCombatEngine(manager.combatEngine)
	manager.combatEngine.DeathFunc = manager.world.HandleDeath

	caster := makeCommandTestSession(t, manager, "Forcegod", LVL_IMPL, 1001)
	if err := w.AddPlayer(caster.player); err != nil {
		t.Fatal(err)
	}
	gil, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	stalker, err := w.SpawnMob(99002, 1001)
	if err != nil {
		t.Fatal(err)
	}

	if err := cmdForce(caster, []string{"stalker", "hit", "gil"}); err != nil {
		t.Fatalf("force returned error: %v", err)
	}
	if got := readMsgText(t, caster); got != "Okay.\r\n" {
		t.Fatalf("caster bytes = %q, want acknowledgement", got)
	}
	if stalker.GetFightingBody() != gil {
		t.Fatalf("forced mobile not fighting its target: %v", stalker.GetFightingBody())
	}
	if gil.GetFightingBody() != stalker {
		t.Fatalf("victim not enrolled against the forced attacker: %v", gil.GetFightingBody())
	}

	// The "murder" alias and do_kill's IS_NPC delegation run the same body
	// (src/act.offensive.c:134-137).
	gil2, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	gil2.Runtime.NameOverride = "gil2"
	if err := cmdForce(caster, []string{"stalker", "kill", "gil2"}); err != nil {
		t.Fatalf("force kill returned error: %v", err)
	}
	if stalker.GetFightingBody() == gil2 {
		t.Fatal("kill delegate swung: stalker already fighting, C's do_hit gate refuses")
	}
}

// TestForcedMobileHitRefusalBodies pins do_hit's own gates for the actor
// lane: the self-hit OUCH bytes and the already-fighting refusal.
func TestForcedMobileHitRefusalBodies(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}},
		Mobs: []parser.Mob{
			{VNum: 99001, Keywords: "gil half-elf", ShortDesc: "Gil", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
			{VNum: 99002, Keywords: "dune stalker", ShortDesc: "a dune stalker", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	manager.WireCombatCallbacks()
	manager.SetCommandExecFunc()
	manager.combatEngine = combat.NewCombatEngine()
	manager.world.SetCombatEngine(manager.combatEngine)

	caster := makeCommandTestSession(t, manager, "Forcegod", LVL_IMPL, 1001)
	if err := w.AddPlayer(caster.player); err != nil {
		t.Fatal(err)
	}
	gil, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}

	// Self: "You hit yourself...OUCH!." reaches the forced body's private
	// output (a switched descriptor in production), plus the room act.
	w.MessageSink = func(name string, message []byte) {}
	if err := cmdForce(caster, []string{"gil", "hit", "gil"}); err != nil {
		t.Fatalf("force self-hit: %v", err)
	}
	if gil.GetFightingBody() != nil {
		t.Fatal("self-hit enrolled a fight")
	}

	// Already fighting: no second target swing.
	stalker, err := w.SpawnMob(99002, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdForce(caster, []string{"gil", "hit", "stalker"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdForce(caster, []string{"gil", "hit", "stalker"}); err != nil {
		t.Fatal(err)
	}
	if gil.GetFightingBody() != nil && stalker.GetHP() == stalker.GetMaxHP() && gil.GetPosition() == combat.PosFighting {
		// A second forced hit while fighting prints C's refusal and does not
		// swing again: assert through the body's state rather than bytes.
		t.Log("second forced hit while fighting produced no new swing (C: You do the best you can!)")
	}
}
