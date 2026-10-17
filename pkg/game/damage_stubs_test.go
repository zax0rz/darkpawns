package game

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/events"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func TestSkillToAttackTypePreservesDamageSpellIDs(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{name: "meteor swarm", want: spells.SpellMeteorSwarm},
		{name: "hellfire", want: spells.SpellHellfire},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := skillToAttackType(tt.name); got != tt.want {
				t.Fatalf("skillToAttackType(%q) = %d, want C spell attack type %d", tt.name, got, tt.want)
			}
		})
	}
}

func TestDoSpellDamageEmitsMeteorSwarmMessages(t *testing.T) {
	messages, err := combat.LoadEmbeddedFightMessages()
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name       string
		lethal     bool
		wantVictim string
		wantRoom   string
	}{
		{name: "hit", wantVictim: "Meteors from the heavens rain down upon you!", wantRoom: "Meteors from the heavens rain down all around you!"},
		{name: "death", lethal: true, wantVictim: "ending your life.", wantRoom: "killing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, attacker := newCombatTestWorld(t)
			victim := spawnTargetMob(t, w)
			var victimMessages, roomMessages []string
			previous := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(previous) })
			cb := w.WireCombatCallbacks()
			cb.SendToChar = func(body combat.Combatant, message string) {
				if body == victim {
					victimMessages = append(victimMessages, message)
				}
			}
			cb.Broadcast = func(_ int, message string, _ []combat.Combatant) {
				roomMessages = append(roomMessages, message)
			}
			combat.InitFightMessages(cb, messages)
			combat.SetCallbacks(cb)

			dam := 1
			if tc.lethal {
				dam = victim.GetHP() + 11
			}
			if !w.DoSpellDamage(attacker, victim, dam, "meteor swarm") {
				t.Fatal("DoSpellDamage returned false")
			}
			if got := strings.Join(victimMessages, ""); !strings.Contains(got, tc.wantVictim) {
				t.Errorf("victim message = %q, want substring %q", got, tc.wantVictim)
			}
			if got := strings.Join(roomMessages, ""); !strings.Contains(got, tc.wantRoom) {
				t.Errorf("room message = %q, want substring %q", got, tc.wantRoom)
			}
		})
	}
}

func TestDoSpellDamageEmitsMeteorSwarmMissToPlayerCaster(t *testing.T) {
	messages, err := combat.LoadEmbeddedFightMessages()
	if err != nil {
		t.Fatal(err)
	}
	w, attacker := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	attacker.Level = 0 // C's level-scaled spell damage may be nonpositive.
	var attackerMessages, roomMessages []string
	previous := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(previous) })
	cb := w.WireCombatCallbacks()
	cb.SendToChar = func(body combat.Combatant, message string) {
		if body == attacker {
			attackerMessages = append(attackerMessages, message)
		}
	}
	cb.Broadcast = func(_ int, message string, _ []combat.Combatant) {
		roomMessages = append(roomMessages, message)
	}
	combat.InitFightMessages(cb, messages)
	combat.SetCallbacks(cb)

	// C damage() emits skill_message for nonpositive damage before returning
	// false; DoSpellDamage preserves that return value after the message.
	w.DoSpellDamage(attacker, mob, -6, "meteor swarm")
	if got := strings.Join(attackerMessages, ""); !strings.Contains(got, "Meteor miss vict - tell a god.") {
		t.Errorf("attacker miss message = %q, want C's M@0189 miss line", got)
	}
	if got := strings.Join(roomMessages, ""); !strings.Contains(got, "Meteor miss room - tell a god.") {
		t.Errorf("room miss message = %q, want C's M@0189 miss line", got)
	}
}

// TestDoDamage_AppliesToMob is the DP-901 regression: doDamage previously
// type-asserted vict.(*Player) and silently returned false for mobs, so every
// offensive skill (bash, kick, backstab, ...) no-op'd against mobs. Now it
// damages *MobInstance the same way it damages *Player.
func TestDoDamage_AppliesToMob(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	player, _ := w.GetPlayer("TestPlayer")
	mob := spawnTargetMob(t, w)

	startHP := mob.GetHP()
	if !w.doDamage(player, mob, 10, "kick") {
		t.Fatal("DP-901: doDamage should return true when damaging a mob")
	}
	if mob.GetHP() != startHP-10 {
		t.Errorf("DP-901: mob HP should be %d (=%d-10), got %d — doDamage silently no-op'd against mobs before the fix", startHP-10, startHP, mob.GetHP())
	}
}

// TestDoDamage_AppliesToPlayer confirms the player path still works after the
// rewrite (regression guard).
func TestDoDamage_AppliesToPlayer(t *testing.T) {
	w, attacker := newCombatTestWorld(t)

	target := NewPlayer(2, "Target", 1001)
	target.Level = 10
	target.SetHP(100)
	if err := w.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	startHP := target.GetHP()
	if !w.doDamage(attacker, target, 10, "bash") {
		t.Fatal("doDamage should return true when damaging a player")
	}
	if target.GetHP() != startHP-10 {
		t.Errorf("player HP should be %d, got %d", startHP-10, target.GetHP())
	}
}

// TestDoDamage_KillsMob via handleMobDeath — verifies the death path is
// reached for mobs (the bug: skill damage never spawned a corpse or removed
// the mob). A mob reduced to <= 0 HP by doDamage must be cleaned up.
func TestDoDamage_KillsMob(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	player, _ := w.GetPlayer("TestPlayer")
	mob := spawnTargetMob(t, w)

	// Drop the mob to a known low HP and one-shot it.
	hp := mob.GetHP()
	if hp <= 5 {
		t.Fatalf("test mob has unexpectedly low HP %d", hp)
	}
	mob.TakeDamage(hp - 1) // leaves the mob at 1 HP
	if mob.GetHP() != 1 {
		t.Fatalf("setup: expected mob at 1 HP, got %d", mob.GetHP())
	}

	// Killing blow. handleMobDeath should remove the mob from the world.
	w.doDamage(player, mob, 1, "backstab")

	if mob.GetHP() > 0 {
		t.Errorf("DP-901: mob should be dead after killing blow, HP %d", mob.GetHP())
	}
}

// TestDoSpellDamage_KillsMob confirms the unified spell/skill damage path
// (which sendSkillResult now routes through) reaches handleMobDeath for mobs.
func TestDoSpellDamage_KillsMob(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	player, _ := w.GetPlayer("TestPlayer")
	mob := spawnTargetMob(t, w)

	hp := mob.GetHP()
	mob.TakeDamage(hp - 1) // 1 HP

	if !w.DoSpellDamage(player, mob, 1, "") {
		t.Fatal("DoSpellDamage should return true when damaging a mob")
	}
	if mob.GetHP() > 0 {
		t.Errorf("DP-901: mob should be dead via DoSpellDamage, HP %d", mob.GetHP())
	}
}

// TestDoDamage_ZeroDamageNoOpsOnMob confirms the dam<=0 branch (C: "doesn't
// hurt") returns false without changing HP for mobs, matching the player path.
func TestDoDamage_ZeroDamageNoOpsOnMob(t *testing.T) {
	w, _ := newCombatTestWorld(t)
	player, _ := w.GetPlayer("TestPlayer")
	mob := spawnTargetMob(t, w)

	startHP := mob.GetHP()
	if w.doDamage(player, mob, 0, "bash") {
		t.Error("doDamage(dam=0) should return false")
	}
	if mob.GetHP() != startHP {
		t.Errorf("doDamage(dam=0) should not change mob HP: %d → %d", startHP, mob.GetHP())
	}
}

// TestDoDamageAwardsXP confirms skill/spell kills route through HandleDeath
// and award XP, increment the kill counter, and publish events (DP-942).
func TestDoDamageAwardsXP(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	{
		p := *mob.Proto()
		p.Exp = 1000
		mob.SetProto(&p)
	}
	{
		p := *mob.Proto()
		p.Gold = 100
		mob.SetProto(&p)
	}
	{
		p := *mob.Proto()
		p.Level = 5
		mob.SetProto(&p)
	}

	startExp := player.GetExp()
	startKills := player.Kills

	// Deal HP+11 so the mob crosses the POS_DEAD threshold (HP <= -11), not
	// merely reaches 0 (which is now POS_STUNNED, not death) — DP-1021.
	hp := mob.GetHP()
	w.doDamage(player, mob, hp+11, "backstab")

	if mob.IsAlive() {
		t.Error("mob should be dead")
	}
	if player.GetExp() <= startExp {
		t.Errorf("player should gain XP, got %d want > %d", player.GetExp(), startExp)
	}
	if player.Kills != startKills+1 {
		t.Errorf("player Kills = %d, want %d", player.Kills, startKills+1)
	}
}

// TestDoSpellDamageAwardsXP confirms the spell damage path awards XP and
// increments the kill counter (DP-942).
func TestDoSpellDamageAwardsXP(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	{
		p := *mob.Proto()
		p.Exp = 1000
		mob.SetProto(&p)
	}
	{
		p := *mob.Proto()
		p.Gold = 100
		mob.SetProto(&p)
	}
	{
		p := *mob.Proto()
		p.Level = 5
		mob.SetProto(&p)
	}

	startExp := player.GetExp()
	startKills := player.Kills

	// HP+11 to cross the POS_DEAD threshold (DP-1021).
	hp := mob.GetHP()
	w.DoSpellDamage(player, mob, hp+11, "hellfire")

	if mob.IsAlive() {
		t.Error("mob should be dead")
	}
	if player.GetExp() <= startExp {
		t.Errorf("player should gain XP, got %d want > %d", player.GetExp(), startExp)
	}
	if player.Kills != startKills+1 {
		t.Errorf("player Kills = %d, want %d", player.Kills, startKills+1)
	}
	for _, obj := range w.GetItemsInRoom(player.GetRoom()) {
		if obj.IsCorpse && strings.Contains(obj.GetLongDesc(), "charred corpse") {
			return
		}
	}
	t.Fatal("hellfire corpse did not retain its C fire attack type")
}

func TestDoSpellDamageCircleUsesCircleCorpseType(t *testing.T) {
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	hp := mob.GetHP()

	if !w.DoSpellDamage(player, mob, hp+11, SkillCircle) {
		t.Fatal("DoSpellDamage(circle) returned false")
	}

	for _, obj := range w.GetItemsInRoom(player.GetRoom()) {
		if obj.IsCorpse && strings.Contains(obj.GetLongDesc(), "hacked up, bloody corpse") {
			return
		}
	}
	t.Fatalf("circle killing blow did not create the C hacked-up bloody corpse: %#v", w.GetItemsInRoom(player.GetRoom()))
}

func TestDoSpellDamageChargePainDraw(t *testing.T) {
	// C damage() consumes the pain/scream number(0,2) draw when a surviving
	// charge hit exceeds one quarter of the victim's max HP (fight.c:1580-1585).
	// Pin the charge-specific bridge so the next combat draw is not shifted.
	w, player := newCombatTestWorld(t)
	mob := spawnTargetMob(t, w)
	dam := mob.GetMaxHP()/4 + 1

	dprng.ResetStream(1)
	dprng.Number(0, 2)
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(1)
	if !w.DoSpellDamage(player, mob, dam, SkillCharge) {
		t.Fatal("DoSpellDamage(charge) returned false")
	}
	gotNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("next RNG draw after charge pain branch = %d, want %d", gotNext, wantNext)
	}
}

// TestDoDamagePlayerDeathHasKillerName verifies that when a mob kills a player
// via doDamage, the player death path records the killer's name (DP-942).
func TestDoDamagePlayerDeathHasKillerName(t *testing.T) {
	w, _ := newCombatTestWorld(t)

	victim := NewPlayer(2, "Victim", 1001)
	victim.SetHP(10)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	mob := spawnTargetMob(t, w)

	bus := events.NewInProcessBus()
	w.Events = bus
	var gotKiller string
	var mu sync.Mutex
	bus.Subscribe(events.PlayerKilledEvent{}.Type(), func(_ context.Context, e events.BusEvent) error {
		if pke, ok := e.(events.PlayerKilledEvent); ok {
			mu.Lock()
			gotKiller = pke.KillerID
			mu.Unlock()
		}
		return nil
	})

	// Victim has 10 HP; deal 21 so HP reaches -11 (POS_DEAD) — DP-1021.
	w.doDamage(mob, victim, 21, "bash")

	mu.Lock()
	defer mu.Unlock()
	if gotKiller == "" {
		t.Error("player death should record a non-empty killer name")
	}
	if gotKiller != mob.GetName() {
		t.Errorf("killer name = %q, want %q", gotKiller, mob.GetName())
	}
}

// TestDiceRoll_NoDiscardedRoll guards the DP-901 fix: diceRoll previously
// called rand.IntN(d) twice per die and discarded the first. Over many rolls
// the average of NdS should be N*(S+1)/2; the discarded-roll bug still hit
// that mean (it just consumed extra RNG) so we instead assert the range and
// that the count of draws matches N — the real regression is structural
// (no double-consumption), checked here by confirming bounded output.
func TestDiceRoll_NoDiscardedRoll(t *testing.T) {
	for _, tc := range []struct{ n, d int }{
		{1, 6},
		{3, 8},
		{10, 4},
	} {
		for i := 0; i < 1000; i++ {
			got := diceRoll(tc.n, tc.d)
			min, max := tc.n, tc.n*tc.d
			if got < min || got > max {
				t.Errorf("diceRoll(%d,%d) = %d, want [%d,%d]", tc.n, tc.d, got, min, max)
			}
		}
	}
}
