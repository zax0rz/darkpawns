package command

// DP-1212 pipeline-level draw-order tests (R3b/R5a).
//
// These tests drive the REAL sendSkillResult path (Do* → sendSkillResult) —
// not Do* and SkillMessage separately, which is what hid the DP-1212 ordering
// bug — and assert the shared-stream draws happen in C order:
//
//	skill roll(s) → skill_message dice(1,N) → improve_skill number(1,200) [+ number(1,3)]
//
// C runs skill_message inside damage()/hit() and improve_skill after it
// returns (act.offensive.c do_kick/do_backstab, new_cmds.c do_trip/do_headbutt).
//
// Order is asserted through three observables, not just a stream-position
// check (the CMWC stream advances per draw regardless of range, so a "next
// value" check alone only proves draw COUNT):
//  1. the emitted attacker message — which variant the dice(1,N) selected;
//  2. the skill gain — which number(1,3) the improve rolled (stat gate forced
//     to pass with WIS+INT=200);
//  3. the next shared-stream value after the full operation (draw count).
//
// Each seed is accepted only when the C-order reference differs from the old
// (improve-first) reference, so every test is GUARANTEED sharp: it would fail
// on the pre-fix order (GLM.md "test the test"; R5a).

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// drawOrderRig wires a real CombatEngine with the production skill_message
// path (the embedded lib/misc/messages corpus) so sendSkillResult's
// SkillMessage call draws Dice(1,N) from the shared dprng stream exactly as in
// production, and captures the attacker message for variant identification.
type drawOrderRig struct {
	engine   *combat.CombatEngine
	messages combat.FightMessages
	attMsg   *string
	teardown func()
}

func newDrawOrderRig(t *testing.T, chName string) *drawOrderRig {
	t.Helper()
	orig := combat.GetCallbacks()
	messages, err := combat.LoadEmbeddedFightMessages()
	if err != nil {
		t.Fatalf("LoadEmbeddedFightMessages: %v", err)
	}
	var attMsg string
	cb := &combat.GameCallbacks{
		Broadcast: func(_ int, _, _ string) {},
		SendToChar: func(name, msg string) {
			if name == chName {
				attMsg = msg
			}
		},
		GetSex:   func(string) int { return 0 },
		GetHP:    func(string) int { return 100 }, // > -11 → never the Die action
		GetLevel: func(string) int { return 1 },   // < LVL_IMMORT → never the God action
		IsNPC:    func(string) bool { return false },
	}
	combat.InitFightMessages(cb, messages)
	engine := combat.NewCombatEngine()
	engine.SetCallbacks(cb)
	return &drawOrderRig{
		engine:   engine,
		messages: messages,
		attMsg:   &attMsg,
		teardown: func() { combat.SetCallbacks(orig) },
	}
}

// assertPipelineDrawOrder runs the seed search, the C-order and old-order
// references, and the actual sendSkillResult run, asserting C order.
//
//   - trySeed resets the stream to the seed and runs the Do* call, returning
//     the result and whether it was a success. It must reset any per-attempt
//     state (e.g. headbutt recoil HP).
//   - rollReplay consumes, on a reset stream, exactly the draws the Do* success
//     path makes BEFORE sendSkillResult (skill roll, to-hit, weapon dice).
//   - numImproves is the number of improve_skill calls C makes on success.
//   - assertEnrolled verifies combat enrollment after the actual run.
func assertPipelineDrawOrder(t *testing.T, rig *drawOrderRig, sess *killPayoutSession,
	p *game.Player, mob *game.MobInstance, msgType int, skillName string, numImproves int,
	trySeed func(s uint32) (game.SkillResult, bool),
	rollReplay func(),
	assertEnrolled func(t *testing.T),
) {
	t.Helper()
	if variants, ok := rig.messages.Variants(msgType); !ok || len(variants) == 0 {
		t.Fatalf("messages set %d (%s) not in embedded corpus", msgType, skillName)
	}

	// Reference players: skill 50 (in (0,97) → the number(1,3) increment draw
	// fires) and WIS+INT=200 (the number(1,200) gate always passes), so each
	// improve consumes exactly number(1,200) then number(1,3) and the gain is
	// directly observable.
	mkRef := func(name string, id int) *game.Player {
		r := game.NewPlayer(id, name, testRoomVNum)
		r.SetSkill(skillName, 50)
		r.Stats.Int = 100
		r.Stats.Wis = 100
		return r
	}

	var (
		seed     uint32
		result   game.SkillResult
		refMsg   string
		refGain  int
		wantNext int
	)
	found := false
	for s := uint32(1); s < 500 && !found; s++ {
		res, ok := trySeed(s)
		if !ok || !res.Success {
			continue
		}
		if len(res.DeferredImprove) != numImproves {
			t.Fatalf("%s success DeferredImprove = %v, want %d entries (C improve_skill count)",
				skillName, res.DeferredImprove, numImproves)
		}
		for _, sk := range res.DeferredImprove {
			if sk != skillName {
				t.Fatalf("%s success DeferredImprove = %v, want all %q", skillName, res.DeferredImprove, skillName)
			}
		}

		// C-order reference: rolls → skill_message dice → improves.
		refCh := mkRef("RefC", 900)
		dprng.ResetStream(s)
		rollReplay()
		*rig.attMsg = ""
		rig.engine.SkillMessage(res.Damage, p.Name, mob.GetName(), msgType, p.GetRoom())
		cMsg := *rig.attMsg
		for i := 0; i < numImproves; i++ {
			game.ImproveSkill(refCh, skillName)
		}
		cGain := refCh.GetSkill(skillName) - 50
		cNext := dprng.Number(0, 999)

		// Old (pre-DP-1212) reference: improves BEFORE the skill_message dice.
		refChB := mkRef("RefB", 901)
		dprng.ResetStream(s)
		rollReplay()
		for i := 0; i < numImproves; i++ {
			game.ImproveSkill(refChB, skillName)
		}
		bGain := refChB.GetSkill(skillName) - 50
		*rig.attMsg = ""
		rig.engine.SkillMessage(res.Damage, p.Name, mob.GetName(), msgType, p.GetRoom())
		bMsg := *rig.attMsg

		if cMsg == bMsg && cGain == bGain {
			continue // this seed cannot distinguish the two orders — keep looking
		}
		seed, result, refMsg, refGain, wantNext = s, res, cMsg, cGain, cNext
		found = true
	}
	if !found {
		t.Skipf("no seed in 1..499 both succeeds %s and distinguishes message-before-improve order", skillName)
	}

	// Actual: drive the REAL sendSkillResult path on the chosen seed.
	dprng.ResetStream(seed)
	res, ok := trySeed(seed)
	if !ok || !res.Success {
		t.Fatalf("seed %d: %s did not succeed on re-run (non-deterministic?)", seed, skillName)
	}
	*rig.attMsg = ""
	if err := sendSkillResult(sess, p, mob, res); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}
	gotNext := dprng.Number(0, 999)

	if *rig.attMsg != refMsg {
		t.Errorf("%s: attacker message = %q, want %q — the skill_message dice drew from the wrong "+
			"stream position (C order: dice BEFORE improve)", skillName, *rig.attMsg, refMsg)
	}
	if got := p.GetSkill(skillName) - 50; got != refGain {
		t.Errorf("%s: skill gain = %d, want %d — the improve draws consumed the wrong stream values "+
			"(C order: improve AFTER the skill_message dice)", skillName, got, refGain)
	}
	if gotNext != wantNext {
		t.Errorf("%s: next stream value = %d, want %d — draw count mismatch through the full operation",
			skillName, gotNext, wantNext)
	}
	if !result.StartCombat {
		t.Errorf("%s: success should set StartCombat (C's damage() enrolls both combatants)", skillName)
	}
	if assertEnrolled != nil {
		assertEnrolled(t)
	}
}

// TestSendSkillResult_KickSuccess_DrawOrderMatchesC — kick success (level-1
// caster → GET_LEVEL>>1 == 0 damage, the DP-1212 enrollment gap case):
// number(1,101) → dice(1,N) → number(1,200) → number(1,3), and combat IS
// enrolled via StartCombat even with zero damage (C: damage(…,0,SKILL_KICK)
// calls set_fighting).
func TestSendSkillResult_KickSuccess_DrawOrderMatchesC(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker() // quiesce the shared stream during draw assertions
	p := ktw.addPlayer(t, 1, "Kicker", 1, game.ClassWarrior, false)
	p.SetSkill(game.SkillKick, 50)
	p.Stats.Int = 100
	p.Stats.Wis = 100

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	assertPipelineDrawOrder(
		t, rig, sess, p, ktw.mob, game.SkillKickNum, game.SkillKick, 1,
		func(s uint32) (game.SkillResult, bool) {
			dprng.ResetStream(s)
			res := game.DoKick(p, ktw.mob)
			return res, res.Success
		},
		func() {
			dprng.Number(1, 101) // skill roll (act.offensive.c:622)
		},
		func(t *testing.T) {
			t.Helper()
			// Zero-damage success: enrollment comes from the StartCombat fix.
			if p.GetFighting() != ktw.mob.GetName() {
				t.Errorf("kick zero-damage success should enroll the kicker in combat: fighting=%q want %q",
					p.GetFighting(), ktw.mob.GetName())
			}
		},
	)
}

// TestSendSkillResult_TripSuccess_DrawOrderMatchesC — trip success:
// number(1,121) → dice(1,N) → number(1,200) → number(1,3); positive damage
// enrolls via DoSpellDamage.
func TestSendSkillResult_TripSuccess_DrawOrderMatchesC(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker()
	p := ktw.addPlayer(t, 1, "Tripper", 10, game.ClassThief, false)
	p.SetSkill(game.SkillTrip, 50)
	p.Stats.Int = 100
	p.Stats.Wis = 100

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	assertPipelineDrawOrder(
		t, rig, sess, p, ktw.mob, game.SkillTripNum, game.SkillTrip, 1,
		func(s uint32) (game.SkillResult, bool) {
			ktw.mob.SetPosition(combat.PosStanding) // undo TargetFalls between probes
			dprng.ResetStream(s)
			res := game.DoTrip(p, ktw.mob, ktw.world)
			return res, res.Success
		},
		func() {
			dprng.Number(1, 121) // skill roll (new_cmds.c:788)
		},
		func(t *testing.T) {
			t.Helper()
			if ktw.mob.GetFighting() != p.Name {
				t.Errorf("trip success should enroll the victim on the attacker: fighting=%q want %q",
					ktw.mob.GetFighting(), p.Name)
			}
		},
	)
}

// TestSendSkillResult_HeadbuttSuccess_DrawOrderMatchesC — headbutt success:
// number(1,121) → dice(1,N) → TWO improve_skill calls (new_cmds.c:450,457),
// each number(1,200) → number(1,3).
func TestSendSkillResult_HeadbuttSuccess_DrawOrderMatchesC(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker()
	p := ktw.addPlayer(t, 1, "Headbutter", 10, game.ClassWarrior, false)
	p.SetSkill(game.SkillHeadbutt, 50)
	p.Stats.Int = 100
	p.Stats.Wis = 100

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	assertPipelineDrawOrder(
		t, rig, sess, p, ktw.mob, game.SkillHeadbuttNum, game.SkillHeadbutt, 2,
		func(s uint32) (game.SkillResult, bool) {
			p.SetHP(200) // reset recoil between probes
			ktw.mob.SetPosition(combat.PosStanding)
			dprng.ResetStream(s)
			res := game.DoHeadbutt(p, ktw.mob, ktw.world)
			return res, res.Success
		},
		func() {
			dprng.Number(1, 121) // skill roll (new_cmds.c:422); recoil draws nothing
		},
		func(t *testing.T) {
			t.Helper()
			if ktw.mob.GetFighting() != p.Name {
				t.Errorf("headbutt success should enroll the victim on the attacker: fighting=%q want %q",
					ktw.mob.GetFighting(), p.Name)
			}
		},
	)
}

// TestSendSkillResult_BackstabSuccess_DrawOrderMatchesC — backstab success on
// a sleeping victim (guaranteed to-hit): number(1,101) [skill roll] →
// number(1,20) [THAC0 to-hit inside C's hit()] → weapon dice → dice(1,N)
// [skill_message] → number(1,200) → number(1,3) [improve].
func TestSendSkillResult_BackstabSuccess_DrawOrderMatchesC(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker()
	p := ktw.addPlayer(t, 1, "Rogue", 20, game.ClassThief, false)
	p.SetSkill(game.SkillBackstab, 50)
	p.Stats.Int = 100
	p.Stats.Wis = 100
	equipPiercingWeapon(t, p)

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	assertPipelineDrawOrder(
		t, rig, sess, p, ktw.mob, game.SkillBackstabNum, game.SkillBackstab, 1,
		func(s uint32) (game.SkillResult, bool) {
			ktw.mob.SetPosition(combat.PosSleeping) // sleeping → skill-roll pass + to-hit hit
			dprng.ResetStream(s)
			res := game.DoBackstab(p, ktw.mob, ktw.world)
			return res, res.Success && res.Damage > 0
		},
		func() {
			dprng.Number(1, 101) // skill roll (act.offensive.c:221)
			dprng.Number(1, 20)  // THAC0 to-hit inside hit() (fight.c:1825)
			weaponNum, weaponSides := p.Equipment.GetWeaponDamage()
			dprng.Dice(weaponNum, weaponSides) // weapon damage dice
		},
		func(t *testing.T) {
			t.Helper()
			if ktw.mob.GetFighting() != p.Name {
				t.Errorf("backstab success should enroll the victim on the attacker: fighting=%q want %q",
					ktw.mob.GetFighting(), p.Name)
			}
		},
	)
}

// TestSendSkillResult_BashSuccess_DrawOrderMatchesC — bash success: number(1,101)
// [percent roll] → dice(1,N) [skill_message] → number(1,200) → number(1,3)
// [deferred improve]. Positive damage ((level/2)+1) enrolls via DoSpellDamage.
// Bash improves only on hit, once (act.offensive.c:492).
func TestSendSkillResult_BashSuccess_DrawOrderMatchesC(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker()
	p := ktw.addPlayer(t, 1, "Basher", 10, game.ClassWarrior, false)
	p.SetSkill(game.SkillBash, 50)
	p.Stats.Int = 100
	p.Stats.Wis = 100

	rig := newDrawOrderRig(t, p.Name)
	defer rig.teardown()
	sess := &killPayoutSession{player: p, world: ktw.world, combatEngine: rig.engine}

	assertPipelineDrawOrder(
		t, rig, sess, p, ktw.mob, game.SkillBashNum, game.SkillBash, 1,
		func(s uint32) (game.SkillResult, bool) {
			p.Move = 100 // reset move (SpendMove(10) drains it)
			ktw.mob.SetPosition(combat.PosFighting)
			dprng.ResetStream(s)
			res := game.DoBash(p, ktw.mob, ktw.world)
			return res, res.Success
		},
		func() {
			dprng.Number(1, 101) // percent roll (act.offensive.c:475)
		},
		func(t *testing.T) {
			t.Helper()
			if ktw.mob.GetFighting() != p.Name {
				t.Errorf("bash success should enroll the victim on the attacker: fighting=%q want %q",
					ktw.mob.GetFighting(), p.Name)
			}
		},
	)
}
