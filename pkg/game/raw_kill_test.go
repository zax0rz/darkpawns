package game

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func rawKillWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "RawKill proof", Zone: 1}},
		Objs: []parser.Obj{
			{VNum: 18, Keywords: "dust pile", ShortDesc: "a pile of dust", LongDesc: "A pile of dust lies here.", TypeFlag: 13},
			{VNum: 1230, Keywords: "dust vampire", ShortDesc: "some vampire dust", LongDesc: "A small pile of dust lays here.", TypeFlag: 12},
			{VNum: 3001, Keywords: "sword", ShortDesc: "a sword", TypeFlag: 5, WearFlags: [4]int{1 << 13}, Values: [4]int{0, 1, 4, 3}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Victim", 1001)
	p.SetLevel(10)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	cb := w.WireCombatCallbacks()
	cb.Broadcast = w.WoundBroadcast
	combat.SetCallbacks(cb)
	return w, p
}

func TestRawKillClearsEverySpellAffect(t *testing.T) {
	w, p := rawKillWorld(t)
	p.Stats.Str = 12
	p.CopyBaseAttributes()
	p.AddAffect(engine.NewAffectDirect(spells.SpellStrength, engine.ApplyStr, -1, 3, 0, "strength"))
	p.AddAffect(engine.NewAffectDirect(spells.SpellBlindness, engine.ApplyNone, 20, 0, engine.AFFBlind, "blindness"))
	p.SetAffect(affBlind, true)
	p.SetAffect(affWerewolf, true)
	p.AddMasterAffect(&engine.MasterAffect{Type: 900, Location: engine.ApplyNone})
	w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
	if len(p.ActiveAffects) != 0 || len(p.GetMasterAffects()) != 0 || p.IsAffected(affBlind) || p.IsAffected(affWerewolf) || p.GetStr() != 12 {
		t.Fatalf("RawKill affects=%v blind=%t werewolf=%t strength=%d; want none/false/false/12", p.ActiveAffects, p.IsAffected(affBlind), p.IsAffected(affWerewolf), p.GetStr())
	}
}

func TestRawKillVampireManaClamp(t *testing.T) {
	for _, vampire := range []bool{true, false} {
		t.Run(map[bool]string{true: "vampire", false: "ordinary"}[vampire], func(t *testing.T) {
			w, p := rawKillWorld(t)
			p.SetMaxMana(30)
			p.SetMana(90)
			p.SetAffect(affVampire, vampire)
			w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
			want := 90
			if vampire {
				want = 30
			}
			if p.GetMana() != want || p.IsAffected(affVampire) {
				t.Fatalf("mana/vampire=%d/%t want %d/false", p.GetMana(), p.IsAffected(affVampire), want)
			}
		})
	}
}

func TestRawKillRemovesTattoo(t *testing.T) {
	w, p := rawKillWorld(t)
	p.Stats.Str = 12
	p.CopyBaseAttributes()
	p.Tattoo = TattooDragon
	p.TatTimer = 8
	TattooAf(p, true)
	w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
	if p.Tattoo != TattooNone || p.TatTimer != 0 || p.GetStr() != 12 || p.Damroll != 0 {
		t.Fatalf("tattoo teardown: tattoo=%d timer=%d str=%d damroll=%d", p.Tattoo, p.TatTimer, p.GetStr(), p.Damroll)
	}
}

func rawKillHeldItems(t *testing.T, w *World, p *Player) (*ObjectInstance, *ObjectInstance) {
	t.Helper()
	inv, err := w.SpawnObject(3001, -1)
	if err != nil {
		t.Fatal(err)
	}
	worn, err := w.SpawnObject(3001, -1)
	if err != nil {
		t.Fatal(err)
	}
	for _, obj := range []*ObjectInstance{inv, worn} {
		if err := w.MoveObjectToPlayerInventory(obj, p); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.MoveObject(worn, LocEquippedPlayer(p.Name, SlotWield)); err != nil {
		t.Fatal(err)
	}
	return inv, worn
}

func TestRawKillCorpseHoldsInventoryEquipmentAndGold(t *testing.T) {
	w, p := rawKillWorld(t)
	inv, worn := rawKillHeldItems(t, w, p)
	p.SetGold(17)
	newer, err := w.SpawnObject(3001, -1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToPlayerInventory(newer, p); err != nil {
		t.Fatal(err)
	}
	w.RawKillCombatant(p, combat.TYPE_BLAST)
	objects := w.GetItemsInRoom(1001)
	if len(objects) != 1 || !objects[0].IsCorpse {
		t.Fatalf("want one corpse, got %v", objects)
	}
	corpse := objects[0]
	if corpse.GetLongDesc() != "A blasted corpse lies here in pieces." {
		t.Errorf("TYPE_BLAST corpse=%q", corpse.GetLongDesc())
	}
	if len(corpse.Contains) != 4 {
		t.Fatalf("corpse contains %d, want two carried, worn, money", len(corpse.Contains))
	}
	if corpse.GetKeywords() != "Victim corpse" || corpse.Contains[1] != worn || corpse.Contains[2] != newer || corpse.Contains[3] != inv {
		t.Errorf("corpse name/list order: keywords=%q contents=%v", corpse.GetKeywords(), corpse.Contains)
	}
	found := map[*ObjectInstance]bool{}
	for _, obj := range corpse.Contains {
		found[obj] = true
	}
	if !found[inv] || !found[worn] || len(p.Inventory.Items) != 0 || len(p.Equipment.GetEquippedItems()) != 0 || p.GetGold() != 0 {
		t.Error("RawKill did not transfer all possessions into corpse")
	}
	w.ExtractPendingPlayers()
	if _, ok := w.GetPlayer(p.Name); ok {
		t.Error("victim remains after extraction")
	}
	if len(w.GetItemsInRoom(1001)) != 1 {
		t.Error("extraction scattered corpse contents")
	}
}

func TestRawKillRaceDust(t *testing.T) {
	for _, race := range []int{combat.RACE_UNDEAD, combat.RACE_VAMPIRE} {
		t.Run(map[int]string{combat.RACE_UNDEAD: "undead", combat.RACE_VAMPIRE: "vampire"}[race], func(t *testing.T) {
			w, p := rawKillWorld(t)
			p.Race = race
			inv, worn := rawKillHeldItems(t, w, p)
			var output strings.Builder
			w.MessageSink = func(_ string, msg []byte) { output.Write(msg) }
			w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
			wantVNum := 18
			if race == combat.RACE_VAMPIRE {
				wantVNum = 1230
			}
			objects := w.GetItemsInRoom(1001)
			found := map[*ObjectInstance]bool{}
			dust := false
			for _, obj := range objects {
				found[obj] = true
				if obj.IsCorpse {
					t.Error("race must produce dust, not corpse")
				}
				if obj.GetVNum() == wantVNum {
					dust = true
				}
			}
			if len(objects) != 3 || !found[inv] || !found[worn] || !dust {
				t.Fatalf("race dust/scattered items missing: %v", objects)
			}
			if output.String() != "" {
				t.Fatalf("dust invented output=%q", output.String())
			}
		})
	}
}

func TestRawKillDeathCryPrecedesCorpse(t *testing.T) {
	w, p := rawKillWorld(t)
	cries := 0
	combat.GetCallbacks().Broadcast = func(room int, msg, exclude string) {
		if strings.Contains(msg, "death cry") {
			cries++
			if len(w.GetItemsInRoom(room)) != 0 || p.HasPLRFlag(PlrExtract) {
				t.Error("corpse or extraction preceded cry")
			}
		}
	}
	w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
	if cries != 1 || len(w.GetItemsInRoom(1001)) != 1 || !p.HasPLRFlag(PlrExtract) {
		t.Fatalf("cry/body/extract=%d/%d/%t", cries, len(w.GetItemsInRoom(1001)), p.HasPLRFlag(PlrExtract))
	}
}

func TestRawKillUnmountAndForget(t *testing.T) {
	w, p := rawKillWorld(t)
	mount := newSpecProcTestMob(t, w, 1001, 10)
	mount.SetMountRider(p.Name)
	mount.SetAffected(affMounted)
	mount.SetFollowing(p.Name)
	p.MountName = mount.GetName()
	p.SetAffect(affMounted, true)
	guard := newSpecProcTestMob(t, w, 1001, 10)
	guard.SetMobFlag(MobFlagMemory)
	guard.Remember(p.Name)
	guard.SetHunting(p.Name)
	unflagged := newSpecProcTestMob(t, w, 1001, 10)
	unflagged.Remember(p.Name)
	unflagged.SetHunting(p.Name)
	w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
	if p.IsMounted() || p.IsAffected(affMounted) || mount.IsMountedMob() || mount.IsAffected(affMounted) {
		t.Error("RawKill did not unmount both sides")
	}
	if len(guard.GetMemory()) != 0 || guard.IsHunting() || unflagged.IsHunting() || len(unflagged.GetMemory()) != 1 {
		t.Error("RawKill memory/hunting gates wrong")
	}
}

func TestRawKillProtectionBackfire(t *testing.T) {
	for _, spell := range []int{spells.SpellProtFromEvil, spells.SpellProtFromGood} {
		t.Run(map[int]string{spells.SpellProtFromEvil: "evil", spells.SpellProtFromGood: "good"}[spell], func(t *testing.T) {
			w, p := rawKillWorld(t)
			p.SetExp(900)
			p.Deaths = 4
			want := "You cannot protect yourself from the Evil inside you!\r\n"
			p.SetAlignment(-1000)
			if spell == spells.SpellProtFromGood {
				p.SetAlignment(1000)
				want = "The forces of Light destroy you for your betrayal!\r\n"
			}
			var output strings.Builder
			w.MessageSink = func(name string, msg []byte) {
				if name == p.Name {
					output.Write(msg)
				}
			}
			spells.MagAffects(10, p, p, spell, 0, w)
			objects := w.GetItemsInRoom(1001)
			if output.String() != want || len(objects) != 1 || objects[0].GetLongDesc() != "A blasted corpse lies here in pieces." || !p.HasPLRFlag(PlrExtract) {
				t.Fatalf("backfire output=%q objects=%v extracted=%t", output.String(), objects, p.HasPLRFlag(PlrExtract))
			}
			if p.GetExp() != 900 || p.Deaths != 4 {
				t.Error("raw_kill applied die penalties/counters")
			}
		})
	}
}

func TestRawKillLuaBinding(t *testing.T) {
	w, p := rawKillWorld(t)
	p.SetAffect(affVampire, true)
	p.SetMaxMana(30)
	p.SetMana(90)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kill.lua"), []byte(fmt.Sprintf(`function oncmd() raw_kill(ch, me, %d) return TRUE end`, combat.TYPE_BLAST)), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewWorldScriptableAdapter(w)
	e := scripting.NewEngine(dir, adapter)
	t.Cleanup(e.Close)
	chRef := scripting.CharRef{ID: p.ID}
	meRef := scripting.CharRef{NPC: true, ID: mob.GetID()}
	ok, err := e.RunScript(&scripting.ScriptContext{World: adapter, ChRef: &chRef, MeRef: &meRef, RoomVNum: 1001}, "kill.lua", "oncmd")
	if err != nil || !ok {
		t.Fatalf("Lua kill: %t %v", ok, err)
	}
	objects := w.GetItemsInRoom(1001)
	if len(objects) != 1 || objects[0].GetLongDesc() != "A blasted corpse lies here in pieces." || !p.HasPLRFlag(PlrExtract) || p.GetMana() != 30 || p.IsAffected(affVampire) {
		t.Fatalf("Lua RawKill/attack type/writeback: objects=%v extract=%t mana=%d", objects, p.HasPLRFlag(PlrExtract), p.GetMana())
	}
}

// The direct spell caller uses combat.RawKill for NPC casters too. The world
// arm must tear down the same state before handleMobDeath emits the cry.
func TestRawKillNPCUsesFullTeardown(t *testing.T) {
	w, observer := rawKillWorld(t)
	mob := newSpecProcTestMob(t, w, 1001, 10)
	proto := *mob.Proto()
	proto.Race = combat.RACE_VAMPIRE
	mob.SetProto(&proto)
	mob.SetAffected(affVampire)
	mob.SetMana(mob.GetMaxMana() + 20)
	mob.SetAffected(affBlind)
	mob.CustomData["affect_4"] = engine.NewAffectDirect(4, engine.ApplyNone, 10, 0, engine.AFFBlind, "blind")
	observer.MountName = mob.GetName()
	observer.SetAffect(affMounted, true)
	mob.SetMountRider(observer.Name)
	mob.SetAffected(affMounted)
	mob.SetFollowing(observer.Name)
	guard := newSpecProcTestMob(t, w, 1001, 10)
	guard.SetMobFlag(MobFlagMemory)
	guard.Remember(mob.GetName())
	guard.SetHunting(mob.GetName())
	combat.RawKill(mob, combat.TYPE_BLAST)
	if mob.IsAlive() || mob.IsAffected(affBlind) || mob.IsAffected(affVampire) || mob.GetMana() != mob.GetMaxMana() || mob.IsAffected(affMounted) || observer.IsMounted() || guard.IsHunting() || len(guard.GetMemory()) != 0 {
		t.Error("NPC RawKill did not complete teardown")
	}
	objects := w.GetItemsInRoom(1001)
	if len(objects) != 1 || objects[0].GetVNum() != 1230 {
		t.Fatalf("NPC vampire dust=%v", objects)
	}
}

// Verify the ordering at the real callback boundary, including effects before
// the cry/body/extraction. C: src/fight.c:541-580.
func TestRawKillTeardownOrder(t *testing.T) {
	w, p := rawKillWorld(t)
	p.SetFightingBody(NewPlayer(99999, "opponent", 1001))
	p.AddAffect(engine.NewAffectDirect(10, engine.ApplyStr, 5, 2, 0, "buff"))
	p.Tattoo = TattooDragon
	TattooAf(p, true)
	p.SetAffect(affVampire, true)
	p.SetMaxMana(30)
	p.SetMana(90)
	var steps []string
	cb := combat.GetCallbacks()
	wrap := func(name string, original func(string)) func(string) {
		return func(victim string) { steps = append(steps, name); original(victim) }
	}
	remove := cb.RemoveAllAffects
	cb.RemoveAllAffects = func(name string) {
		if p.GetFighting() != "" {
			t.Error("affect removal preceded stop_fighting")
		}
		steps = append(steps, "affects")
		remove(name)
	}
	cb.RemoveTattoo = wrap("tattoo", cb.RemoveTattoo)
	cb.ClearNightbreed = wrap("nightbreed", cb.ClearNightbreed)
	cb.Unmount = wrap("unmount", cb.Unmount)
	cb.ForgetVictim = wrap("forget", cb.ForgetVictim)
	cb.Broadcast = func(_ int, msg, _ string) {
		if strings.Contains(msg, "death cry") {
			steps = append(steps, "cry")
			if len(p.ActiveAffects) != 0 || p.Tattoo != TattooNone || p.IsAffected(affVampire) || p.GetMana() != 30 {
				t.Error("cry preceded state teardown")
			}
		}
	}
	body := cb.MakeCorpse
	cb.MakeCorpse = func(name string, attack int) { steps = append(steps, "corpse"); body(name, attack) }
	extract := cb.ExtractChar
	cb.ExtractChar = func(name string) { steps = append(steps, "extract"); extract(name) }
	w.RawKillCombatant(p, combat.TYPE_UNDEFINED)
	if got, want := strings.Join(steps, ","), "affects,tattoo,nightbreed,unmount,forget,cry,corpse,extract"; got != want {
		t.Fatalf("teardown order=%q want %q", got, want)
	}
}
