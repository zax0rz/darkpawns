package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newHuntItemsTestWorld(t *testing.T) (*World, *Player, map[string]string) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hunt items", Zone: 1}},
		Objs: []parser.Obj{
			{VNum: 4209, Keywords: "opal ring", ShortDesc: "an opal ring", TypeFlag: ITEM_ARMOR},
			{VNum: 4210, Keywords: "jade ring", ShortDesc: "a jade ring", TypeFlag: ITEM_ARMOR},
			{VNum: 14223, Keywords: "spiked armor", ShortDesc: "a suit of spiked, black armor", TypeFlag: ITEM_ARMOR},
		},
		Mobs: []parser.Mob{
			{VNum: 4207, Keywords: "bhyroga warrior", ShortDesc: "the Bhyroga warrior", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
			{VNum: 14203, Keywords: "skarash", ShortDesc: "Skarash", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
			{VNum: 14205, Keywords: "infidel", ShortDesc: "an Infidel", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
			{VNum: 14207, Keywords: "nonesuch", ShortDesc: "a Nonesuch", Position: combat.PosStanding, DefaultPos: combat.PosStanding},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	player := NewPlayer(11, "Wearer", 1001)
	player.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	w.PlayingDescriptorBodies = func() []Actor { return []Actor{player} }
	return w, player, out
}

func equipHuntItem(t *testing.T, player *Player, vnum int, slot EquipmentSlot, shortDesc string) {
	t.Helper()
	proto := &parser.Obj{VNum: vnum, ShortDesc: shortDesc}
	item := NewObjectInstance(proto, -1)
	if player.Equipment == nil {
		player.Equipment = NewEquipment()
	}
	player.Equipment.mu.Lock()
	player.Equipment.slots[slot] = item
	player.Equipment.mu.Unlock()
}

func setHuntItemsDraws(t *testing.T, values ...int) *[]int {
	t.Helper()
	previous := huntItemsNumber
	var calls []int
	idx := 0
	huntItemsNumber = func(from, to int) int {
		if from != 0 || to != 100 {
			t.Fatalf("hunt_items draw = number(%d,%d), want number(0,100)", from, to)
		}
		if idx >= len(values) {
			t.Fatalf("unexpected hunt_items draw #%d", idx+1)
		}
		value := values[idx]
		idx++
		calls = append(calls, value)
		return value
	}
	t.Cleanup(func() { huntItemsNumber = previous })
	return &calls
}

func TestHuntItemsUsesWearOrderAndFirstAvailableCharacter(t *testing.T) {
	w, player, out := newHuntItemsTestWorld(t)
	// Spawn order is the reverse of C character_list order: latest body first.
	opalOldest, err := w.SpawnMob(4207, 1001)
	if err != nil {
		t.Fatal(err)
	}
	opalNewest, err := w.SpawnMob(4207, 1001)
	if err != nil {
		t.Fatal(err)
	}
	armorHunter, err := w.SpawnMob(14203, 1001)
	if err != nil {
		t.Fatal(err)
	}
	equipHuntItem(t, player, 14223, SlotBody, "a suit of spiked, black armor")
	equipHuntItem(t, player, 4210, SlotFingerL, "a jade ring")
	equipHuntItem(t, player, 4209, SlotFingerR, "an opal ring")
	calls := setHuntItemsDraws(t, 0, 0, 0)

	w.HuntItems()

	if len(*calls) != 3 {
		t.Fatalf("matching worn items consumed %d RNG draws, want 3", len(*calls))
	}
	if got := opalNewest.GetHuntingID(); got != "11" {
		t.Fatalf("newest opal hunter player ID=%q, want 11", got)
	}
	if got := opalOldest.GetHuntingID(); got != "11" {
		t.Fatalf("remaining opal hunter jade target ID=%q, want 11", got)
	}
	if got := armorHunter.GetHuntingID(); got != "11" {
		t.Fatalf("armor hunter player ID=%q, want 11", got)
	}
	if got := out[player.Name]; got != "" {
		t.Fatalf("wearer got invented hunt-items output %q; C's self-addressed NPC tells have no recipient", got)
	}
}

func TestHuntItemsConsumesDrawBeforeHunterEligibility(t *testing.T) {
	for _, tc := range []struct {
		name string
		gate func(*MobInstance, *Player)
	}{
		{name: "fighting", gate: func(m *MobInstance, p *Player) { m.SetFightingBody(p) }},
		{name: "asleep", gate: func(m *MobInstance, _ *Player) { m.SetPosition(combat.PosSleeping) }},
		{name: "charmed", gate: func(m *MobInstance, _ *Player) { m.SetAffected(affCharm) }},
		{name: "already hunting", gate: func(m *MobInstance, p *Player) {
			m.mu.Lock()
			m.Hunting, m.HuntingMobID = "busy", 999
			m.mu.Unlock()
			_ = p
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, player, _ := newHuntItemsTestWorld(t)
			hunter, err := w.SpawnMob(14203, 1001)
			if err != nil {
				t.Fatal(err)
			}
			item := NewObjectInstance(&parser.Obj{VNum: 14223, ShortDesc: "a suit of spiked, black armor"}, -1)
			player.Equipment.mu.Lock()
			player.Equipment.slots[SlotBody] = item
			player.Equipment.mu.Unlock()
			tc.gate(hunter, player)
			calls := setHuntItemsDraws(t, 0)

			w.HuntItems()

			if len(*calls) != 1 {
				t.Fatalf("draw count=%d, want one draw before candidate gates", len(*calls))
			}
			hunter.mu.RLock()
			gotName, gotPlayerID, gotMobID := hunter.Hunting, hunter.HuntingID, hunter.HuntingMobID
			hunter.mu.RUnlock()
			if tc.name == "already hunting" {
				if gotName != "busy" || gotMobID != 999 || gotPlayerID != "" {
					t.Fatalf("existing hunt changed to name=%q player=%q mob=%d", gotName, gotPlayerID, gotMobID)
				}
			} else if gotName != "" || gotPlayerID != "" || gotMobID != 0 {
				t.Fatalf("ineligible hunter acquired name=%q player=%q mob=%d", gotName, gotPlayerID, gotMobID)
			}
		})
	}
}

func TestHuntVictimKeepsPlayerIDWhenDescriptorMissing(t *testing.T) {
	w, player, _ := newHuntItemsTestWorld(t)
	mob, err := w.SpawnMob(14203, 1001)
	if err != nil {
		t.Fatal(err)
	}
	w.PlayerHasPlayingDescriptor = func(*Player) bool { return false }
	w.SetHuntingTarget(mob, player)
	if got := mob.GetHuntingID(); got != "11" {
		t.Fatalf("stored hunting player ID=%q, want 11", got)
	}
	w.huntVictim(mob)
	if mob.GetHuntingID() != "11" {
		t.Fatalf("same-room hunt cleared player ID %q", mob.GetHuntingID())
	}
}
