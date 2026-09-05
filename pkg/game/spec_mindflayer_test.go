package game

import (
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

type mindflayerCombatEngine struct {
	target combat.Combatant
}

func (e *mindflayerCombatEngine) StartCombat(combat.Combatant, combat.Combatant) error {
	return nil
}

func (e *mindflayerCombatEngine) IsFighting(string) bool { return false }

func (e *mindflayerCombatEngine) GetCombatTarget(string) (combat.Combatant, bool) {
	if e.target == nil {
		return nil, false
	}
	return e.target, true
}

func mindflayerSeed(t *testing.T, want int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		if dprng.New(seed).Number(0, 15) == want {
			return seed
		}
	}
	t.Fatalf("could not find a seed for mindflayer roll %d", want)
	return 0
}

func TestSpecMindflayer_RngArms(t *testing.T) {
	for _, want := range []int{0, 5, 15} {
		seed := mindflayerSeed(t, want)
		if got := dprng.New(seed).Number(0, 15); got != want {
			t.Fatalf("mindflayer seed %d produced roll %d, want %d", seed, got, want)
		}
	}
}

func prepareMindflayerCombat(t *testing.T) (*World, *Player, *MobInstance, *mindflayerCombatEngine, *string, func() string) {
	t.Helper()
	w, player, lastMessage := newSpecProcTestWorld(t)
	player.SetMaxHP(1000)
	player.SetHP(1000)
	observer := NewPlayer(2, "Witness", player.GetRoomVNum())
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, player.GetRoomVNum(), 32)
	mob.SetPosition(combat.PosFighting)
	mob.SetFighting(player.GetName())
	player.SetPosition(combat.PosFighting)
	player.SetFighting(mob.GetName())
	engine := &mindflayerCombatEngine{target: player}
	w.SetCombatEngine(engine)

	var skillMessage string
	originalCallbacks := combat.GetCallbacks()
	combat.SetCallbacks(&combat.GameCallbacks{
		SkillMessage: func(dam int, ch, vict string, attackType, roomVNum int) bool {
			skillMessage = ch + "|" + vict + "|" + strconv.Itoa(dam) + "|" + strconv.Itoa(attackType) + "|" + strconv.Itoa(roomVNum)
			return true
		},
	})
	t.Cleanup(func() { combat.SetCallbacks(originalCallbacks) })
	_ = lastMessage() // discard the spawn's room-arrival bytes
	return w, player, mob, engine, &skillMessage, lastMessage
}

func TestSpecMindflayer_EntryGatesAndFallthrough(t *testing.T) {
	w, player, mob, _, _, _ := prepareMindflayerCombat(t)

	if specMindflayer(w, nil, mob, "look", "") {
		t.Fatal("mindflayer should reject non-empty commands")
	}
	mob.SetFighting("")
	if specMindflayer(w, nil, mob, "", "") {
		t.Fatal("mindflayer should reject a mob without a fighting target")
	}
	mob.SetFighting(player.GetName())
	mob.SetPosition(combat.PosSleeping)
	if specMindflayer(w, nil, mob, "", "") {
		t.Fatal("mindflayer should reject a sleeping mob")
	}
	mob.SetPosition(combat.PosFighting)
	dprng.ResetStream(mindflayerSeed(t, 1))
	if specMindflayer(w, nil, mob, "", "") {
		t.Fatal("mindflayer's default roll should fall through")
	}
}

func TestSpecMindflayer_SoulLeechUsesDirectDamageAndHeals(t *testing.T) {
	w, player, mob, _, skillMessage, lastMessage := prepareMindflayerCombat(t)
	mob.SetHealth(40)
	player.SetHP(1000)
	dprng.ResetStream(mindflayerSeed(t, 0))

	if !specMindflayer(w, nil, mob, "", "") {
		t.Fatal("mindflayer soul-leech roll should be handled")
	}
	if got, want := player.GetHP(), 995; got != want {
		t.Errorf("soul leech victim HP = %d, want %d", got, want)
	}
	if got, want := mob.GetHP(), 45; got != want {
		t.Errorf("soul leech mob HP = %d, want uncapped %d", got, want)
	}
	if got, want := *skillMessage, "a test mob|Tester|5|83|1001"; got != want {
		t.Errorf("soul leech skill message = %q, want %q", got, want)
	}
	if got, want := lastMessage(), "The tentacles on a test mob's face surge forward, wrapping around Tester's head!\r\nThe tentacles on a test mob's face surge forward, wrapping around your head!\r\n"; got != want {
		t.Errorf("soul leech audience Acts = %q, want %q", got, want)
	}
}

func TestSpecMindflayer_PsiblastUsesDirectDamage(t *testing.T) {
	w, player, mob, _, skillMessage, lastMessage := prepareMindflayerCombat(t)
	mob.SetHealth(40)
	player.SetHP(1000)
	dprng.ResetStream(mindflayerSeed(t, 15))

	if !specMindflayer(w, nil, mob, "", "") {
		t.Fatal("mindflayer psiblast roll should be handled")
	}
	if got, want := player.GetHP(), 968; got != want {
		t.Errorf("psiblast victim HP = %d, want %d", got, want)
	}
	if got, want := mob.GetHP(), 40; got != want {
		t.Errorf("psiblast mob HP = %d, want unchanged %d", got, want)
	}
	if got, want := *skillMessage, "a test mob|Tester|32|100|1001"; got != want {
		t.Errorf("psiblast skill message = %q, want %q", got, want)
	}
	if got, want := lastMessage(), "Blood runs from Tester's nose and ears as a test mob stares intently at him.\r\nA test mob stares intently at you.. you feel it battering your mind!\r\n"; got != want {
		t.Errorf("psiblast audience Acts = %q, want %q", got, want)
	}
}
