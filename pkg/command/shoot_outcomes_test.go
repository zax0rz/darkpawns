package command

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// These fixtures call the registered command with real world-owned objects
// and bodies. The fake engine records only the direct retaliation boundary.
type shootRetaliation struct {
	calls              int
	attacker, defender combat.Combatant
}

func (e *shootRetaliation) PerformRangedRetaliation(a, b combat.Combatant) error {
	e.calls++
	e.attacker = a
	e.defender = b
	return nil
}

func shootOutcomeFixture(t *testing.T, npc bool, skill int) (*killPayoutSession, *game.ObjectInstance, combat.Combatant, *shootRetaliation, map[string]string) {
	t.Helper()
	s, arrow := newShootGateSession(t)
	s.player.SetLevel(20)
	s.player.SetSkill(game.SkillShoot, skill)
	s.player.Stats.Dex = 14
	s.player.CopyBaseAttributes()
	s.player.Damroll = 7
	arrow.SetValue(1, 2)
	arrow.SetValue(2, 7)
	bow, _ := s.player.Equipment.GetItemInSlot(game.SlotWield)
	bow.SetValue(1, 3)
	bow.SetValue(2, 4)
	var victim combat.Combatant
	if npc {
		m, err := s.world.SpawnMobQuiet(3001, 1002)
		if err != nil {
			t.Fatal(err)
		}
		m.ClearMobFlag(1)
		m.Dex = 14
		m.CopyBaseAttributes()
		m.SetHealth(1000)
		victim = m
	} else {
		p := game.NewPlayer(2, "Victim", 1002)
		p.SetLevel(20)
		p.Stats.Dex = 14
		p.CopyBaseAttributes()
		p.SetHP(1000)
		if err := s.world.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
		victim = p
	}
	for i, room := range []int{1001, 1002} {
		p := game.NewPlayer(i+3, fmt.Sprintf("Observer%d", i), room)
		if err := s.world.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	wire := map[string]string{}
	s.world.MessageSink = func(name string, msg []byte) { wire[name] += string(msg) }
	e := &shootRetaliation{}
	s.combatEngine = e
	return s, arrow, victim, e, wire
}

func fireOutcome(t *testing.T, s *killPayoutSession, v combat.Combatant) {
	t.Helper()
	name := v.GetName()
	if v.IsNPC() {
		name = "guard"
	}
	if err := CmdShoot(s, []string{"arrow", "north", name}); err != nil {
		t.Fatal(err)
	}
}

func TestShootMobOutcomes(t *testing.T) {
	for _, hit := range []bool{false, true} {
		t.Run(fmt.Sprint(hit), func(t *testing.T) {
			skill := 1
			if hit {
				skill = 200
			}
			s, arrow, v, e, _ := shootOutcomeFixture(t, true, skill)
			// NPC DEX defaults to 11: zero reaction, so these force both outcomes.
			dprng.ResetStream(71)
			_ = dprng.Number(1, 101)
			damage := 7
			if hit {
				damage += dprng.Dice(2, 7) + dprng.Dice(3, 4)
				_ = dprng.Number(1, 200)
			}
			next := dprng.Next()
			dprng.ResetStream(71)
			fireOutcome(t, s, v)
			if hit {
				if v.GetHP() != 1000-damage || v.GetRoom() != 1001 || e.calls != 1 || e.attacker != v || e.defender != s.player {
					t.Fatalf("mob hit hp/room/retaliation=%d/%d/%d want %d/1001/1", v.GetHP(), v.GetRoom(), e.calls, 1000-damage)
				}
				if arrow.Location.Kind != game.ObjNowhere {
					t.Fatal("hit did not extract arrow")
				}
			} else {
				if v.GetHP() != 1000 || v.GetRoom() != 1002 || e.calls != 0 {
					t.Fatal("miss damaged, moved or retaliated")
				}
				if arrow.Location != game.LocRoom(1002) {
					t.Fatalf("miss location=%+v", arrow.Location)
				}
			}
			if got := dprng.Next(); got != next {
				t.Fatalf("next draw=%d want %d", got, next)
			}
		})
	}
}

func TestShootDexProbabilityAndStrictComparison(t *testing.T) {
	for _, dex := range []struct{ shooter, victim, adjust int }{{0, 14, -70}, {18, 14, 20}, {14, 0, 70}, {14, 18, -20}, {18, 18, 0}} {
		for _, delta := range []int{-1, 0, 1} {
			t.Run(fmt.Sprintf("%d/%d/%d", dex.shooter, dex.victim, delta), func(t *testing.T) {
				s, _, v, _, _ := shootOutcomeFixture(t, false, 100)
				s.player.Stats.Dex = dex.shooter
				s.player.CopyBaseAttributes()
				p := v.(*game.Player)
				p.Stats.Dex = dex.victim
				p.CopyBaseAttributes()
				dprng.ResetStream(71)
				roll := dprng.Number(1, 101)
				s.player.SetSkill(game.SkillShoot, roll-dex.adjust+delta)
				dprng.ResetStream(71)
				fireOutcome(t, s, v)
				wantHit := delta > 0
				gotHit := strings.Contains(strings.Join(s.getMessages(), ""), "roar of pain")
				if gotHit != wantHit {
					t.Fatalf("roll=%d skill=%d dex adjustment=%d hit=%t want %t", roll, s.player.GetSkill(game.SkillShoot), dex.adjust, gotHit, wantHit)
				}
			})
		}
	}
}

func TestShootImprovementFollowsExtractionBeforeHP(t *testing.T) {
	s, arrow, v, _, wire := shootOutcomeFixture(t, false, 60)
	s.player.Stats.Dex = 18
	s.player.Stats.Wis = 18
	s.player.Stats.Int = 18
	s.player.CopyBaseAttributes()
	p := v.(*game.Player)
	p.Stats.Dex = 0
	p.CopyBaseAttributes()
	var seed uint32
	var damage int
	var next uint32
	for candidate := uint32(1); candidate < 10000; candidate++ {
		dprng.ResetStream(candidate)
		_ = dprng.Number(1, 101)
		dam := 7 + dprng.Dice(2, 7) + dprng.Dice(3, 4)
		if dprng.Number(1, 200) <= 36 && dprng.Number(1, 3) == 3 {
			seed = candidate
			damage = dam
			next = dprng.Next()
			break
		}
	}
	if seed == 0 {
		t.Fatal("no improvement fixture seed")
	}
	improves := 0
	s.world.MessageSink = func(name string, msg []byte) {
		wire[name] += string(msg)
		if name == s.player.Name && strings.Contains(string(msg), "Your skill in shoot improves.") {
			improves++
			if len(s.world.GetAllObjects()) != 1 || arrow.Location.Kind != game.ObjNowhere {
				t.Fatal("improvement preceded projectile extraction")
			}
			if !strings.Contains(strings.Join(s.getMessages(), ""), "You hear a roar of pain!") || v.GetHP() != 1000 {
				t.Fatal("improvement not between roar and direct victim HP loss")
			}
		}
	}
	dprng.ResetStream(seed)
	fireOutcome(t, s, v)
	if improves != 1 || s.player.GetSkill(game.SkillShoot) != 63 || v.GetHP() != 1000-damage {
		t.Fatalf("improves/skill/HP=%d/%d/%d", improves, s.player.GetSkill(game.SkillShoot), v.GetHP())
	}
	if got := dprng.Next(); got != next {
		t.Fatalf("improvement draw order=%d want %d", got, next)
	}
}

func TestShootDirectFightingGuardPrecedesArguments(t *testing.T) {
	s, arrow := newShootGateSession(t)
	s.player.SetFightingBody(game.NewPlayer(99, "Other", 1001))
	location := arrow.Location
	dprng.ResetStream(71)
	next := dprng.Next()
	dprng.ResetStream(71)
	if err := CmdShoot(s, nil); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(s.getMessages(), ""); got != "But you are already engaged in close-range combat!\r\n" {
		t.Fatalf("direct fighting guard=%q", got)
	}
	if arrow.Location != location || dprng.Next() != next {
		t.Fatal("fighting guard mutated projectile or RNG")
	}
}
