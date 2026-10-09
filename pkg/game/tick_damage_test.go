package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func tickDamageWorld(t *testing.T) (*World, *Player, *Player, map[string]*strings.Builder) {
	t.Helper()
	w, p := newCombatTestWorld(t)
	w.StopAITicker()
	p.SetHP(100)
	p.MaxHealth = 100
	p.Conditions = [3]int{-1, -1, -1}
	peer := NewPlayer(2, "Peer", 1001)
	peer.Conditions = [3]int{-1, -1, -1}
	if err := w.AddPlayer(peer); err != nil {
		t.Fatal(err)
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	cb := w.WireCombatCallbacks()
	cb.SendToChar = func(body combat.Combatant, message string) { body.SendMessage(message) }
	cb.Broadcast = func(room int, message string, excluded []combat.Combatant) {
		for _, recipient := range w.GetPlayersInRoom(room) {
			skip := false
			for _, body := range excluded {
				if body == recipient {
					skip = true
				}
			}
			if !skip {
				recipient.SendMessage(message)
			}
		}
	}
	combat.SetCallbacks(cb)
	combat.InitFightMessages(cb, loadMessagesFile(t))
	return w, p, peer, captureOutput(w)
}

// limits.c:503-513 calls damage(), including its modifiers and messages.
func TestPointUpdatePoisonDamageTail(t *testing.T) {
	for _, tc := range []struct {
		name   string
		affect int
		level  int
		wantHP int
	}{
		{"ordinary", 0, 10, 90},
		{"spell-affect", 0, 10, 90},
		{"sanctuary", affSanctuary, 10, 95},
		{"immortal", 0, LVL_IMMORT, 100},
		{"hidden", affHide, 10, 90},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, p, peer, out := tickDamageWorld(t)
			p.SetLevel(tc.level)
			if tc.name == "spell-affect" {
				p.AddAffect(engine.NewAffectDirect(33, engine.ApplyStr, 10, -2, engine.AFFPoison, "poison"))
			} else {
				p.SetAffect(AffPoison, true)
			}
			if tc.affect != 0 {
				p.SetAffect(tc.affect, true)
			}
			w.PointUpdate()
			if p.GetHP() != tc.wantHP {
				t.Fatalf("tick HP=%d, want %d", p.GetHP(), tc.wantHP)
			}
			if tc.name == "hidden" {
				if p.IsAffected(affHide) {
					t.Fatal("tick left victim hidden")
				}
				if !strings.Contains(outputOf(out, peer.Name), "TestPlayer slowly fades into existence.\r\n") {
					t.Fatalf("peer bytes=%q", outputOf(out, peer.Name))
				}
			}
			if tc.name == "ordinary" || tc.name == "spell-affect" {
				if got := outputOf(out, p.Name); got != "You feel burning poison in your blood, and suffer." {
					t.Fatalf("poison victim bytes=%q", got)
				}
				if got := outputOf(out, peer.Name); got != "TestPlayer looks really sick and shivers uncomfortably." {
					t.Fatalf("poison room bytes=%q", got)
				}
			}
		})
	}
}

func TestPointUpdateWoundedDamageTail(t *testing.T) {
	for _, tc := range []struct {
		pos, hp, wantHP int
		victim, room    string
	}{
		{combat.PosIncap, -4, -5, "You are incapacitated an will slowly die, if not aided.\r\n", "TestPlayer is incapacitated and will slowly die, if not aided.\r\n"},
		{combat.PosMortally, -7, -9, "You are mortally wounded, and will die soon, if not aided.\r\n", "TestPlayer is mortally wounded, and will die soon, if not aided.\r\n"},
	} {
		w, p, peer, out := tickDamageWorld(t)
		p.SetHP(tc.hp)
		p.SetPosition(tc.pos)
		w.PointUpdate()
		if p.GetHP() != tc.wantHP {
			t.Fatalf("wounded tick HP=%d, want %d", p.GetHP(), tc.wantHP)
		}
		if !strings.HasSuffix(outputOf(out, p.Name), tc.victim) {
			t.Fatalf("wounded victim bytes=%q", outputOf(out, p.Name))
		}
		if !strings.HasSuffix(outputOf(out, peer.Name), tc.room) {
			t.Fatalf("wounded room bytes=%q", outputOf(out, peer.Name))
		}
	}
}

func TestPointUpdateCutthroatDamageTail(t *testing.T) {
	w, p, _, out := tickDamageWorld(t)
	p.SetAffect(AffCutthroat, true)
	w.PointUpdate()
	if p.GetHP() != 87 {
		t.Fatalf("cutthroat HP=%d, want 87", p.GetHP())
	}
	if outputOf(out, p.Name) == "" {
		t.Fatal("cutthroat omitted skill_message")
	}
}

type tickLowRoller struct{}

func (tickLowRoller) IntN(n int) int { return 0 }

func (tickLowRoller) Number(low, high int) int  { return low }
func (tickLowRoller) Dice(number, size int) int { return number }

func TestPointUpdateMountedPoisonDamageTail(t *testing.T) {
	w, p, _, out := tickDamageWorld(t)
	original := combat.GetRoller()
	combat.SetRoller(tickLowRoller{})
	t.Cleanup(func() { combat.SetRoller(original) })
	mount := spawnTargetMob(t, w)
	p.MountName = mount.GetName()
	mount.SetFollowingBody(p)
	mount.SetAffected(affMounted)
	p.SetAffect(affMounted, true)
	mount.SetMountRider(p.Name)
	p.Stats.Dex = 18
	p.CopyBaseAttributes()
	p.SetAffect(AffPoison, true)
	w.PointUpdate()
	if p.IsMounted() || mount.GetMountRider() != "" {
		t.Fatal("tick did not dismount")
	}
	got := outputOf(out, p.Name)
	for _, line := range []string{"The hit knocks you off of your mount!\r\n\r\n", "You hop off your mount.\r\n", "You land on your feet!\r\n"} {
		if !strings.Contains(got, line) {
			t.Fatalf("dismount bytes=%q missing %q", got, line)
		}
	}
}

// C limits.c:108-116,176-184,241-246 reads effective AFF flags,
// including affects installed by mag_affects, rather than innate bits alone.
func TestRegenActiveSpellFlags(t *testing.T) {
	for _, flag := range []struct {
		name  string
		flags uint64
	}{
		{"poison", engine.AFFPoison}, {"flaming", engine.AFFFlaming}, {"cutthroat", engine.AFFCutthroat},
	} {
		t.Run(flag.name, func(t *testing.T) {
			w := &World{}
			p := NewPlayer(1, "Affected", 0)
			p.Class = ClassWarrior
			p.Level = 10
			p.Position = PosStanding
			p.Conditions = [3]int{20, 0, 20}
			for _, gain := range []struct {
				name string
				call func(*Player) int
			}{
				{"hit", w.HitGain}, {"mana", w.ManaGain}, {"move", w.MoveGain},
			} {
				t.Run(gain.name, func(t *testing.T) {
					p.ActiveAffects = nil
					baseline := gain.call(p)
					p.AddAffect(engine.NewAffectDirect(33, engine.ApplyNone, 10, 0, flag.flags, flag.name))
					if got := gain.call(p); got != baseline/4 || got == baseline {
						t.Fatalf("active %s %s gain=%d, want %d (baseline %d)", flag.name, gain.name, got, baseline/4, baseline)
					}
				})
			}
		})
	}
}
