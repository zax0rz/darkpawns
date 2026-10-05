package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDeathOrderWorld(t *testing.T, hp, exp int) (*World, *Player, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Death proof", Zone: 1}}, Mobs: []parser.Mob{{VNum: 100, Keywords: "victim", ShortDesc: "a victim", Level: 10, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: hp - 1}, Exp: exp, Gold: 7}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Killer", 1001)
	p.SetLevel(10)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(100, 1001)
	if err != nil {
		t.Fatal(err)
	}
	mob.SetPosition(combat.PosStanding)
	mob.SetGold(7)
	return w, p, mob
}

// src/fight.c:1644-1667, 573-578: XP, autogold, death cry, silent corpse.
// All command-owned damage boundaries must delegate the complete sequence.
func TestSkillDeathOrderExactBytes(t *testing.T) {
	for _, seam := range []struct {
		name   string
		damage func(*World, combat.Combatant, combat.Combatant, int) bool
	}{
		{"dragon", (*World).DoDragonKickDamage},
		{"smackheads", (*World).DoSmackheadsDamage},
		{"disembowel", (*World).DoDisembowelDamage},
		{"groinrip", (*World).DoGroinripDamage},
		{"neckbreak", (*World).DoNeckbreakDamage},
		{"tiger", (*World).DoTigerPunchDamage},
		{"strike", (*World).DoStrikeDamage},
		{"cutthroat", (*World).DoCutthroatDamage},
		{"plain", func(w *World, ch, v combat.Combatant, dam int) bool { return w.ApplySkillDamage(ch, v, dam, SkillKick) }},
	} {
		for _, exp := range []int{0, 100} {
			variant := "one_lousy"
			if exp > 1 {
				variant = "plural"
			}
			t.Run(seam.name+"/"+variant, func(t *testing.T) {
				w, killer, mob := newDeathOrderWorld(t, 10, exp)
				killer.AutoGold = true
				killer.AutoSplit = true // solo damage() ignores autosplit
				var got strings.Builder
				w.MessageSink = func(name string, msg []byte) {
					if name == killer.Name {
						got.Write(msg)
					}
				}
				old := combat.GetCallbacks()
				t.Cleanup(func() { combat.SetCallbacks(old) })
				cryCount := 0
				combat.SetCallbacks(&combat.GameCallbacks{GetAdjacentRoom: func(int, int) int { return -1 }, Broadcast: func(room int, msg string, excludedBodies []combat.Combatant) {
					exclude := testBodyNames(excludedBodies)

					if strings.Contains(msg, "death cry") {
						cryCount++
						if len(w.GetItemsInRoom(room)) != 0 {
							t.Error("corpse exists before death cry")
						}
						if killer.GetGold() != 7 || killer.GetExp() != max(1, exp) {
							t.Error("death cry preceded XP/autogold")
						}
					}
					if exclude != killer.Name {
						killer.SendMessage(msg + "\r\n")
					}
				}})
				if !seam.damage(w, killer, mob, 30) {
					t.Fatal("lethal damage did not execute")
				}
				xp := "You receive one lousy experience point.\r\n"
				if exp > 1 {
					xp = "You receive 100 experience points.\r\n"
				}
				want := "A victim is dead!  R.I.P.\r\n" + xp + "You loot 7 gold from the corpse.\r\nYour blood freezes as you hear a victim's death cry.\r\n"
				if got.String() != want {
					t.Fatalf("death bytes = %q; want %q", got.String(), want)
				}
				if cryCount != 1 {
					t.Fatalf("death cries = %d", cryCount)
				}
				items := w.GetItemsInRoom(1001)
				if len(items) != 1 || !items[0].IsCorpse {
					t.Fatalf("corpse not built after cry: %v", items)
				}
			})
		}
	}
}

func TestAmbushTailPreservesWoundedVictimStop(t *testing.T) {
	for _, wounded := range []bool{false, true} {
		name := "fresh"
		hp := 500
		if wounded {
			name = "wounded"
			hp = 10
		}
		t.Run(name, func(t *testing.T) {
			w, killer, mob := newDeathOrderWorld(t, hp, 0)
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			combat.SetCallbacks(&combat.GameCallbacks{})
			ce := combat.NewCombatEngine()
			w.SetCombatEngine(ce)
			w.applyAmbushDamage(killer, mob, 16)
			if killer.GetFighting() != mob.GetName() {
				t.Fatal("attacker not enrolled")
			}
			if wounded {
				if mob.GetFighting() != "" {
					t.Fatal("ambush re-enrolled wounded victim")
				}
				killer.SetWaitState(10)
				before := killer.GetHP()
				ce.PerformRound()
				if mob.GetPosition() != combat.PosMortally || killer.GetHP() != before {
					t.Fatal("ambush victim got an extra round")
				}
			} else {
				if mob.GetFighting() != killer.Name {
					t.Fatal("fresh ambush victim not enrolled")
				}
				mob.SetPosition(combat.PosSitting)
				ce.PerformRound()
				if mob.GetPosition() != combat.PosFighting {
					t.Fatal("fresh ambush victim never got its engine round")
				}
			}
		})
	}
}
