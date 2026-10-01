package game

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func attrPlayer() *Player {
	p := NewPlayer(1, "Attr", 1001)
	p.Stats = CharStats{Str: 18, StrAdd: 37, Dex: 18, Int: 18, Wis: 18, Con: 18, Cha: 30}
	p.CopyBaseAttributes()
	return p
}

func attrSnapshot(p *Player) CharStats {
	return CharStats{Str: p.GetStr(), StrAdd: p.GetStrAdd(), Dex: p.GetDex(), Int: p.GetInt(), Wis: p.GetWis(), Con: p.GetCon(), Cha: p.GetCha()}
}

// src/handler.c:351-372: limits are PC/NPC, never mortal/immortal.
func TestE2AttrBounds(t *testing.T) {
	for _, level := range []int{1, LVL_IMPL} {
		for _, value := range []int{-1, 0, 18, 19, 25, 26} {
			t.Run(fmt.Sprintf("PC/%d/%d", level, value), func(t *testing.T) {
				p := attrPlayer()
				p.Level = level
				p.Stats = CharStats{Str: value, StrAdd: 37, Dex: value, Int: value, Wis: value, Con: value, Cha: value}
				base := p.Stats
				p.CopyBaseAttributes()
				if got := attrSnapshot(p); got != base {
					t.Fatalf("initial copy=%+v want %+v", got, base)
				}
				p.Alignment = 1001
				p.AffectTotal()
				want := CharStats{Str: max(0, min(value, 18)), StrAdd: 37, Dex: max(0, min(value, 18)), Int: max(0, min(value, 18)), Wis: max(0, min(value, 18)), Con: max(0, min(value, 18)), Cha: value}
				switch value {
				case 19:
					want.StrAdd = 47
				case 25, 26:
					want.StrAdd = 100
				}
				if got := attrSnapshot(p); got != want {
					t.Fatalf("total=%+v want %+v", got, want)
				}
				if p.Stats != base || p.GetAlignment() != 1000 {
					t.Fatal("base drift or missing alignment bound")
				}
				p.Alignment = -1001
				p.AffectTotal()
				if p.GetAlignment() != -1000 || attrSnapshot(p) != want {
					t.Fatal("negative alignment or repeated STR_ADD conversion")
				}
			})
		}
	}
	for _, value := range []int{-1, 0, 18, 19, 25, 26} {
		t.Run(fmt.Sprintf("NPC/%d", value), func(t *testing.T) {
			m := NewMob(&parser.Mob{VNum: 1, Str: 18, Int: 18, Wis: 18, Dex: 18, Con: 18, Cha: 18}, 1001)
			m.Str, m.Intel, m.Wis, m.Dex, m.Con, m.Cha = value, value, value, value, value, value
			m.CopyBaseAttributes()
			if m.GetDex() != value {
				t.Fatal("prototype copy clipped")
			}
			m.SetAlignment(-1001)
			m.AffectTotal()
			want := max(0, min(value, 25))
			if m.GetStr() != want || m.GetDex() != want || m.GetInt() != want || m.GetWis() != want || m.GetCon() != want || m.GetCha() != value || m.GetAlignment() != -1000 {
				t.Fatalf("NPC effective bounds for %d: str=%d dex=%d cha=%d align=%d", value, m.GetStr(), m.GetDex(), m.GetCha(), m.GetAlignment())
			}
			if m.Str != value || m.Dex != value {
				t.Fatal("NPC base rewritten")
			}
		})
	}
}

func attrItem(location, modifier int) *ObjectInstance {
	return NewObjectInstance(&parser.Obj{VNum: 14425, Keywords: "vest", WearFlags: [4]int{9}, Weight: 1, Affects: []parser.ObjAffect{{Location: location, Modifier: modifier}}}, -1)
}

// Each public equipment entry point must notify after releasing eq.mu.
// src/handler.c:748,792. A failed equip does not total restored stats.
func TestE2AttrEquipmentBoundaries(t *testing.T) {
	for _, entry := range []string{"exact", "equip", "player"} {
		for _, removal := range []string{"slot", "item"} {
			t.Run(entry+"/"+removal, func(t *testing.T) {
				p := attrPlayer()
				p.Stats.Dex = 17
				p.CopyBaseAttributes()
				base := p.Stats
				item := attrItem(ApplyDex, 2)
				switch entry {
				case "exact":
					if err := p.Equipment.SetSlot(SlotBody, item); err != nil {
						t.Fatal(err)
					}
				case "equip":
					if err := p.Equipment.Equip(item, p.Inventory); err != nil {
						t.Fatal(err)
					}
				case "player":
					if z, err := p.Equipment.EquipForPlayer(item, p.Inventory, 0, ClassWarrior); z || err != nil {
						t.Fatalf("equip: %v %v", z, err)
					}
				}
				if p.GetDex() != 18 || p.Stats != base {
					t.Fatalf("equip dex=%d base=%+v", p.GetDex(), p.Stats)
				}
				if removal == "slot" {
					if err := p.Equipment.Unequip(SlotBody, p.Inventory); err != nil {
						t.Fatal(err)
					}
				} else if !p.Equipment.UnequipItem(item, p.Inventory) {
					t.Fatal("unequip failed")
				}
				if p.GetDex() != 17 || p.Stats != base {
					t.Fatalf("remove dex=%d base=%+v", p.GetDex(), p.Stats)
				}
			})
		}
	}
	p := attrPlayer()
	p.Stats.Dex = 25
	p.CopyBaseAttributes()
	if err := p.Equipment.Equip(NewObjectInstance(&parser.Obj{}, -1), p.Inventory); err == nil {
		t.Fatal("invalid item equipped")
	}
	if p.GetDex() != 25 {
		t.Fatal("failed equip totaled")
	}
}

// src/handler.c:395,419,437. Removal restores base rather than subtracting
// a modifier from an already clipped result; joins must total as well.
func TestE2AttrAffectBoundaries(t *testing.T) {
	p := attrPlayer()
	base := p.Stats
	p.AddAffect(engine.NewAffectDirect(1, ApplyStr, 2, 1, 0, "test"))
	if p.GetStr() != 18 || p.GetStrAdd() != 47 {
		t.Fatalf("add str=%d/%d", p.GetStr(), p.GetStrAdd())
	}
	p.JoinAffect(engine.NewAffectDirect(1, ApplyStr, 2, 7, 0, "test"), true, true)
	if p.GetStrAdd() != 100 {
		t.Fatal("join didn't saturate prior ADD")
	}
	p.RemoveAffectBySpell(1)
	if attrSnapshot(p) != base || p.Stats != base {
		t.Fatalf("remove=%+v base=%+v", attrSnapshot(p), p.Stats)
	}
	p.JoinAffect(engine.NewAffectDirect(2, ApplyDex, 0, -30, 0, "test"), false, false)
	if p.GetDex() != 0 {
		t.Fatal("new join didn't floor")
	}
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	w.AffectUpdate()
	if attrSnapshot(p) != base {
		t.Fatalf("expiry=%+v", attrSnapshot(p))
	}
	m := NewMob(&parser.Mob{VNum: 1, Str: 25, Dex: 24, Int: 25, Wis: 25, Con: 25, Cha: 30}, 1001)
	m.AddAffect(engine.NewAffectDirect(1, ApplyDex, 1, 2, 0, "test"))
	if m.GetDex() != 25 {
		t.Fatal("NPC add didn't bound")
	}
	m.JoinAffect(engine.NewAffectDirect(1, ApplyDex, 0, -40, 0, "test"), false, false)
	if m.GetDex() != 0 {
		t.Fatal("NPC join didn't floor")
	}
	m.RemoveAffectBySpell(1)
	if m.GetDex() != 24 {
		t.Fatal("NPC remove didn't restore")
	}
	item := attrItem(ApplyDex, 2)
	m.EquipItem(item, int(SlotBody))
	if m.GetDex() != 25 {
		t.Fatal("NPC equip didn't bound")
	}
	if m.UnequipItem(int(SlotBody)) != item || m.GetDex() != 24 {
		t.Fatal("NPC unequip didn't restore")
	}
}

// src/tattoo.c:104-195 and act.wizard.c:3027-3030: remove old before
// assigning new, add new, then total. The direct tattoo pass is unbounded.
func TestE2AttrTattooReplacement(t *testing.T) {
	p := attrPlayer()
	base := p.Stats
	p.Tattoo = TattooSpider
	TattooAf(p, true)
	if p.GetDex() != 21 || p.Stats != base {
		t.Fatal("tattoo must modify effective only, without bounds")
	}
	p.AffectTotal()
	p.AffectTotal()
	if p.GetDex() != 18 || p.Stats != base {
		t.Fatal("tattoo total drift")
	}
	TattooAf(p, false)
	p.Tattoo = TattooDragon
	TattooAf(p, true)
	p.AffectTotal()
	if p.GetDex() != 18 || p.GetStr() != 18 || p.GetStrAdd() != 57 || p.GetDamroll() != 2 || p.Stats != base {
		t.Fatalf("replacement=%+v damroll=%d", attrSnapshot(p), p.GetDamroll())
	}
	TattooAf(p, false)
	p.Tattoo = 0
	TattooAf(p, true)
	p.AffectTotal()
	if attrSnapshot(p) != base || p.GetDamroll() != 0 {
		t.Fatal("tattoo removal failed to restore")
	}
	p.Stats.Dex = 17
	p.CopyBaseAttributes()
	p.Tattoo = TattooTiger
	TattooAf(p, true)
	move := p.MaxMove
	p.AffectTotal()
	p.AffectTotal()
	if p.GetDex() != 18 || p.MaxMove != move {
		t.Fatal("tattoo non-ability pools drifted")
	}
}

// src/act.wizard.c:1612 and scripts.c:1302,1379: naked restored PC 25
// persists until total even when level is LVL_IMPL. Loading naked saved
// stats follows the same unbounded copy (db.c:2441-2481).
func TestE2AttrCopyAndLuaBoundaries(t *testing.T) {
	p := attrPlayer()
	p.Level = LVL_IMPL
	p.Stats = CharStats{Str: 25, StrAdd: 100, Dex: 25, Int: 25, Wis: 25, Con: 25, Cha: 25}
	p.CopyBaseAttributes()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	adapter := NewWorldScriptableAdapter(w)
	ref := scripting.CharRef{ID: p.ID}
	if p.GetDex() != 25 {
		t.Fatal("restore copy clipped")
	}
	adapter.SetSkill(ref, 134, 37)
	if p.GetDex() != 18 || p.Stats.Dex != 25 || p.GetCha() != 25 {
		t.Fatal("Lua set_skill didn't total")
	}
	p.CopyBaseAttributes()
	fields, ok := adapter.CharFields(ref)
	if !ok || fields.Cha != 25 || p.GetDex() != 25 {
		t.Fatal("Lua read clipped restore")
	}
	adapter.ApplyChar(ref, scripting.CharWrite{Level: LVL_IMPL, Align: 1001})
	if p.GetDex() != 18 || p.GetAlignment() != 1000 {
		t.Fatal("Lua save_char didn't total")
	}
	p.RestoreEffectiveAttributes()
	if p.GetDex() != 25 {
		t.Fatal("naked load clipped")
	}
	p.ActiveAffects = []*engine.Affect{engine.NewAffectDirect(1, ApplyDex, 1, 1, 0, "saved")}
	p.RestoreEffectiveAttributes()
	if p.GetDex() != 18 || p.Stats.Dex != 25 {
		t.Fatal("affected load didn't total")
	}
}

// Every affect location participates; removal of the whole spell restores
// both. An absent removal must not clip the restored copy.
func TestE2AttrNPCAffectLocations(t *testing.T) {
	m := NewMob(&parser.Mob{VNum: 1, Str: 25, Dex: 24, Int: 24, Wis: 25, Con: 25, Cha: 30}, 1001)
	m.AddAffect(engine.NewAffectDirect(1, ApplyDex, 0, 2, 0, "test"))
	m.AddAffect(engine.NewAffectDirect(1, ApplyInt, 0, -30, 0, "test"))
	if m.GetDex() != 25 || m.GetInt() != 0 {
		t.Fatal("one spell's second location lost its first modifier")
	}
	m.RemoveAffectBySpell(1)
	if m.GetDex() != 24 || m.GetInt() != 24 {
		t.Fatal("spell removal lost base")
	}
	p := attrPlayer()
	p.Stats.Dex = 25
	p.CopyBaseAttributes()
	p.RemoveAffectBySpell(99)
	if p.GetDex() != 25 {
		t.Fatal("absent affect removal totaled")
	}
}

// Movement/extraction reach the same equip/unequip boundaries as commands.
func TestE2AttrWorldEquipmentBoundaries(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	p := attrPlayer()
	p.Stats.Dex = 17
	p.CopyBaseAttributes()
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	item := attrItem(ApplyDex, 2)
	if err := w.MoveObjectToPlayerInventory(item, p); err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObject(item, LocEquippedPlayer(p.Name, SlotBody)); err != nil {
		t.Fatal(err)
	}
	if p.GetDex() != 18 {
		t.Fatal("world equip didn't total")
	}
	if err := w.MoveObjectToRoom(item, 1001); err != nil {
		t.Fatal(err)
	}
	if p.GetDex() != 17 {
		t.Fatal("world detach didn't total")
	}
}

// Bulk removers have C's affect_remove boundary too (scripts.c lua_unaffect,
// act.wizard.c do_wizutil, act.other.c stop_follower). No read-time summing
// remains to hide a missing notification.
func TestE2AttrBulkRemovalBoundaries(t *testing.T) {
	for _, path := range []string{"lua", "world", "charm"} {
		t.Run(path, func(t *testing.T) {
			w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(w.StopAITicker)
			p := attrPlayer()
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
			p.AddAffect(engine.NewAffectDirect(7, ApplyDex, 2, -30, engine.AFFCharm, "charm"))
			if p.GetDex() != 0 {
				t.Fatal("fixture must floor before removal")
			}
			switch path {
			case "lua":
				NewWorldScriptableAdapter(w).Unaffect(scripting.CharRef{ID: p.ID})
			case "world":
				w.ClearAffects(p.Name, false)
			case "charm":
				removeCharmAffect(p)
			}
			if p.GetDex() != 18 || p.Stats.Dex != 18 {
				t.Fatalf("%s removal retained stale dex=%d", path, p.GetDex())
			}
		})
	}
}

func TestE2AttrNPCExpiry(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	mob.Dex = 24
	mob.CopyBaseAttributes()
	mob.AddAffect(engine.NewAffectDirect(1, ApplyDex, 0, -30, 0, "test"))
	if mob.GetDex() != 0 {
		t.Fatal("NPC expiry fixture didn't floor")
	}
	w.AffectUpdate()
	if mob.GetDex() != 24 || mob.Dex != 24 {
		t.Fatal("NPC expiry didn't restore effective base")
	}
}

func TestE2AttrBasePersistence(t *testing.T) {
	p := attrPlayer()
	base := p.Stats
	p.AddAffect(engine.NewAffectDirect(1, ApplyDex, 2, -30, 0, "test"))
	raw, err := SerializePlayer(p)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := DeserializePlayer(raw)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Stats != base || restored.GetDex() != 0 {
		t.Fatal("save stored effective values or load failed to rebuild")
	}
	restored.RemoveAffectBySpell(1)
	if restored.GetDex() != 18 || restored.Stats != base {
		t.Fatal("saved clipped affect lost permanent base")
	}
}

func TestE2AttrNPCAlignmentConsumers(t *testing.T) {
	m := NewMob(&parser.Mob{VNum: 1, Alignment: -350}, 1001)
	m.SetAlignment(1001)
	m.AffectTotal()
	if !mobIsGood(m) || mobIsEvil(m) || m.GetAlignment() != 1000 || m.Proto().Alignment != -350 {
		t.Fatal("NPC consumers read prototype instead of bounded instance alignment")
	}
}

// Multi-location storage must retain C's deterministic newest-match join.
func TestE2AttrNPCJoinOrder(t *testing.T) {
	for i := 0; i < 20; i++ {
		m := NewMob(&parser.Mob{VNum: 1, Dex: 20}, 1001)
		m.AddAffect(engine.NewAffectDirect(1, ApplyDex, 2, -1, 0, "older"))
		m.AddAffect(engine.NewAffectDirect(1, ApplyDex, 5, -3, 0, "newer"))
		joined := engine.NewAffectDirect(1, ApplyDex, 1, -2, 0, "joined")
		m.JoinAffect(joined, true, false)
		if joined.Duration != 6 || m.GetDex() != 17 {
			t.Fatalf("join chose wrong record: duration=%d dex=%d", joined.Duration, m.GetDex())
		}
	}
}
