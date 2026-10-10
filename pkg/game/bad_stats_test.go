package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func badStatsWorld(t *testing.T, location int) (*World, *Player, *ObjectInstance, map[string]*strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 7907, ShortDesc: "Pestilence"}}, Objs: []parser.Obj{{VNum: 201, Keywords: "helm", ShortDesc: "a cursed helm", TypeFlag: ITEM_ARMOR, WearFlags: [4]int{1 << 4}, Affects: []parser.ObjAffect{{Location: location, Modifier: -10}}, LoadPercent: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(11, "Wearer", 1001)
	p.Stats = CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10}
	p.CopyBaseAttributes()
	p.SetLevel(15)
	p.SetMaxHP(200)
	p.SetHP(200)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(w.WireCombatCallbacks())
	return w, p, obj, captureOutput(w)
}

func TestPlayerEquipmentBadStats(t *testing.T) {
	for _, tc := range []struct {
		name     string
		location int
		text     string
		damage   int
	}{
		{"strength", ApplyStr, "You are too weak to fight!\n\r", 0},
		{"intelligence", ApplyInt, "You are too dumb to do much of anything!\n\r", 0},
		{"wisdom", ApplyWis, "You are too dumb to do much of anything!\n\r", 0},
		{"charisma", ApplyCha, "The world hates you!\n\r", 0},
		{"dexterity", ApplyDex, "You trip over your own feet and hit your head!\n\r", 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, p, obj, out := badStatsWorld(t, tc.location)
			if err := w.EquipItem(p, obj, 6); err != nil {
				t.Fatal(err)
			}
			if got := outputOf(out, p.Name); got != tc.text {
				t.Fatalf("bad-stat bytes=%q, want %q", got, tc.text)
			}
			if p.GetHP() != 200-tc.damage {
				t.Fatalf("bad-stat HP=%d, want %d", p.GetHP(), 200-tc.damage)
			}
		})
	}
}

func TestPlayerEquipmentBadStatPriority(t *testing.T) {
	w, p, obj, out := badStatsWorld(t, ApplyStr)
	p.Stats = CharStats{}
	p.CopyBaseAttributes()
	if err := w.EquipItem(p, obj, 6); err != nil {
		t.Fatal(err)
	}
	if got := outputOf(out, p.Name); got != "You trip over your own feet and hit your head!\n\r" {
		t.Fatalf("last-zero bytes=%q", got)
	}
	if p.GetHP() != 160 {
		t.Fatalf("last-zero HP=%d", p.GetHP())
	}
}

func TestPlayerEquipmentDexUsesDamageTail(t *testing.T) {
	w, p, obj, out := badStatsWorld(t, ApplyDex)
	p.SetAffect(affSanctuary, true)
	p.SetAffect(affHide, true)
	if err := w.EquipItem(p, obj, 6); err != nil {
		t.Fatal(err)
	}
	if p.GetHP() != 180 || p.IsAffected(affHide) {
		t.Fatalf("DEX bypassed damage(): hp=%d hidden=%v", p.GetHP(), p.IsAffected(affHide))
	}
	if !strings.HasPrefix(outputOf(out, p.Name), "You trip over your own feet and hit your head!\n\r") {
		t.Fatalf("DEX bytes=%q", outputOf(out, p.Name))
	}
}

func TestPlayerEquipmentBadStatEntryPoints(t *testing.T) {
	for _, path := range []string{"lua", "move", "restore"} {
		t.Run(path, func(t *testing.T) {
			w, p, obj, out := badStatsWorld(t, ApplyDex)
			switch path {
			case "lua":
				if err := w.MoveObject(obj, LocInventoryPlayer(p.Name)); err != nil {
					t.Fatal(err)
				}
				if !w.EquipChar(p.Name, false, obj.VNum) {
					t.Fatal("Lua equip failed")
				}
			case "move":
				if err := w.MoveObject(obj, LocEquippedPlayer(p.Name, SlotHead)); err != nil {
					t.Fatal(err)
				}
			case "restore":
				if !restoreEquippedItem(p, obj, int(SlotHead)) {
					t.Fatal("equip restore failed")
				}
			}
			if p.GetHP() != 160 || outputOf(out, p.Name) != "You trip over your own feet and hit your head!\n\r" {
				t.Fatalf("%s HP=%d bytes=%q", path, p.GetHP(), outputOf(out, p.Name))
			}
		})
	}
}
