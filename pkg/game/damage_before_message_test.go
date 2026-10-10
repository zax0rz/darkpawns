package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func damageRescueWorld(t *testing.T) (*World, *Player, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Neutral arena", Flags: []string{"neutral"}}, {VNum: 8004, Name: "Rescue destination"}},
		Mobs:  []parser.Mob{{VNum: 2001, ShortDesc: "a training dummy", Level: 30}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	p := NewPlayer(1, "Victim", 1001)
	p.SetLevel(15)
	p.SetHP(1)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old); w.StopAITicker() })
	combat.SetCallbacks(w.WireCombatCallbacks())
	return w, p, mob
}

// The real MagDamage boundary must rescue before skill_message and death.

// The weapon damage tail must use the same complete rescue as spell damage.
// The NPC arm proves the reciprocal opponent, memory and hunting teardown;
// the PC arm proves C's separate TO_CHAR act (an NPC has no descriptor).

func TestDamageNewbieExperienceBeforeMessage(t *testing.T) {
	w, p, mob := damageRescueWorld(t)
	p.SetLevel(1)
	p.SetExp(0)
	mob.SetHealth(100)
	mob.SetLevel(5)
	before := p.GetExp()
	seen := false
	w.ApplySkillDamageWithMessage(p, mob, 3, SkillKick, func(int) {
		seen = true
		if got := p.GetExp() - before; got != 15 {
			t.Fatalf("newbie XP before message=%d want 15", got)
		}
	})
	if !seen {
		t.Fatal("no message callback")
	}
	// The source condition excludes level 2 and self-damage.
	p.SetLevel(2)
	before = p.GetExp()
	w.ApplySkillDamage(p, mob, 3, SkillKick)
	if p.GetExp() != before {
		t.Fatal("level 2 got newbie XP")
	}
	p.SetLevel(1)
	p.SetHP(100)
	w.ApplySkillDamage(p, p, 3, SkillKick)
	if p.GetExp() != before {
		t.Fatal("self-damage got newbie XP")
	}
}

func TestSpellLowLevelStopBeforeMessage(t *testing.T) {
	for _, level := range []int{5, 6} {
		t.Run(string(rune('0'+level)), func(t *testing.T) {
			w, p, mob := damageRescueWorld(t)
			w.GetRoomInWorld(1001).Flags = nil
			p.SetLevel(level)
			stopped := false
			combat.GetCallbacks().SkillMessage = func(int, combat.Combatant, combat.Combatant, int, int) bool {
				stopped = mob.GetFightingBody() == nil
				return true
			}
			spells.MagDamage(30, mob, p, 32, 0, w)
			if stopped != (level <= 5) {
				t.Fatalf("level %d stopped before message=%v", level, stopped)
			}
		})
	}
}

func TestNeutralRescueDamageTailClass(t *testing.T) {
	for _, path := range []string{"melee", "mob skill", "room", "affect", "breath", "numbered skill", "engine"} {
		t.Run(path, func(t *testing.T) {
			w, p, mob := damageRescueWorld(t)
			messages := 0
			combat.GetCallbacks().SkillMessage = func(int, combat.Combatant, combat.Combatant, int, int) bool { messages++; return true }
			switch path {
			case "engine":
				ce := combat.NewCombatEngine()
				w.SetCombatEngine(ce)
				ce.SetCallbacks(w.WireCombatCallbacks())
				combat.GetCallbacks().SkillMessage = func(int, combat.Combatant, combat.Combatant, int, int) bool { messages++; return true }
				p.SetPosition(combat.PosSleeping)
				mob.SetDamroll(100)
				ce.DamageFunc = func(victim combat.Combatant) {
					if victim.GetHP() != 1 || victim.GetRoom() != 8004 {
						t.Fatal("structured damage observer saw pre-rescue state")
					}
				}
				if err := ce.StartCombat(mob, p); err != nil {
					t.Fatal(err)
				}
				if err := ce.PerformInitialAttack(mob, p); err != nil {
					t.Fatal(err)
				}
				if _, ok := ce.GetCombatTarget(mob); ok {
					t.Fatal("engine retained attacker pair")
				}
			case "melee":
				if combat.TakeDamage(mob, p, 40, combat.TYPE_HIT) {
					t.Fatal("rescue returned TRUE")
				}
			case "mob skill":
				if w.mobSkillDamage(mob, p, 40, SkillKickNum) {
					t.Fatal("rescue returned TRUE")
				}
			case "room":
				if w.selfDamage(p, 40, combat.TYPE_UNDEFINED) {
					t.Fatal("rescue returned TRUE")
				}
			case "affect":
				p.SetAffect(AffPoison, true)
				w.PointUpdate()
			case "breath":
				p.SetHP(0)
				p.SetPosition(combat.PosStunned)
				spells.MagDamage(30, mob, p, spells.SpellFrostBreath, 0, w)
			case "numbered skill":
				if w.ApplySkillDamageWithMessage(mob, p, 40, SkillKick, func(int) { messages++ }) {
					t.Fatal("rescue returned TRUE")
				}
			}
			if p.GetRoom() != 8004 || p.GetHP() != 1 || messages != 0 {
				t.Fatalf("tail=%s room=%d HP=%d messages=%d", path, p.GetRoom(), p.GetHP(), messages)
			}
		})
	}
}

func TestNeutralRescueActorAndMount(t *testing.T) {
	w, p, mount := damageRescueWorld(t)
	caster := NewPlayer(3, "Caster", 1001)
	caster.SetLevel(30)
	if err := w.AddPlayer(caster); err != nil {
		t.Fatal(err)
	}
	p.MountName = mount.GetName()
	p.SetAffect(affMounted, true)
	mount.SetMountRider(p.Name)
	mount.SetFollowing(p.Name)
	mount.SetAffected(affMounted)
	out := captureOutput(w)
	if w.ApplySkillDamageWithMessage(caster, p, 40, SkillKick, func(int) { t.Fatal("skill message after rescue") }) {
		t.Fatal("rescue returned TRUE")
	}
	if got := outputOf(out, caster.Name); got != "Victim is saved by the powers of the gods!\r\n" {
		t.Fatalf("actor bytes=%q", got)
	}
	if p.IsMounted() || mount.IsAffected(affMounted) || mount.GetMountRider() != "" {
		t.Fatal("mount links or affects retained")
	}
	if mount.GetFollowing() != p.Name {
		t.Fatal("unmount broke C's retained follower edge")
	}
}

func TestNeutralRescueRoomTransfer(t *testing.T) {
	w, p, mob := damageRescueWorld(t)
	resident := NewPlayer(2, "Resident", 8004)
	watcher := NewPlayer(3, "Watcher", 8004)
	for _, body := range []*Player{resident, watcher} {
		if err := w.AddPlayer(body); err != nil {
			t.Fatal(err)
		}
	}
	torch := w.NewObjectFromProto(&parser.Obj{VNum: 3001, Keywords: "torch", TypeFlag: 1, WearFlags: [4]int{16385}, Values: [4]int{0, 10, 10}}, 1001)
	if err := p.Inventory.AddItem(torch); err != nil {
		t.Fatal(err)
	}
	if err := w.EquipItem(p, torch, eqWearLight); err != nil {
		t.Fatal(err)
	}
	if !p.HasLight() {
		t.Fatal("fixture torch is not lit")
	}
	w.mu.Lock()
	w.adjustRoomLight(1001, 1)
	w.mu.Unlock()
	out := captureOutput(w)
	spells.MagDamage(30, mob, p, 32, 0, w)
	if w.GetRoomInWorld(1001).Light != 0 || w.GetRoomInWorld(8004).Light != 1 {
		t.Fatalf("rescue light counts: old=%d new=%d", w.GetRoomInWorld(1001).Light, w.GetRoomInWorld(8004).Light)
	}
	if p.GetRoomEntrySequence() <= watcher.GetRoomEntrySequence() {
		t.Fatal("rescue did not prepend room arrival")
	}
	w.lookAtRoom(watcher, false)
	got := outputOf(out, watcher.Name)
	victimIndex, residentIndex := strings.Index(got, "Victim"), strings.Index(got, "Resident")
	if victimIndex < 0 || residentIndex < 0 || victimIndex >= residentIndex {
		t.Fatalf("destination people order = %q", got)
	}
}

func TestSkillDamageNilAttacker(t *testing.T) {
	w, p, _ := damageRescueWorld(t)
	p.SetRoom(8004)
	p.SetHP(100)
	if !w.ApplySkillDamage(nil, p, 3, SkillKick) || p.GetHP() != 97 {
		t.Fatalf("source-less skill HP = %d, want 97", p.GetHP())
	}
}
