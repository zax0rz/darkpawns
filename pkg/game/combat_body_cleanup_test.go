package game

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/spells"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func duplicateCleanupWorld(t *testing.T) (*World, *combat.CombatEngine, *MobInstance, *MobInstance, *Player, *Player) {
	t.Helper()
	w, one, p := zoneArmedMob(t)
	two, err := w.SpawnMobQuiet(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	two.SetHealth(1000)
	two.MaxHP = 1000
	q := NewPlayer(2, "Other", 100)
	q.SetHP(1000)
	q.SetMaxHP(1000)
	if err := w.AddPlayer(q); err != nil {
		t.Fatal(err)
	}
	w.rooms[101] = &parser.Room{VNum: 101}
	old := combat.GetCallbacks()
	ce := combat.NewCombatEngine()
	ce.SetCallbacks(w.WireCombatCallbacks())
	w.SetCombatEngine(ce)
	t.Cleanup(func() { ce.Stop(); combat.SetCallbacks(old) })
	for _, pair := range [][2]combat.Combatant{{one, p}, {two, q}} {
		if err := ce.StartCombat(pair[0], pair[1]); err != nil {
			t.Fatal(err)
		}
	}
	return w, ce, one, two, p, q
}

func TestCombatBodyWorldCleanup(t *testing.T) {
	for _, kind := range []string{"death", "raw-kill", "extract-mob", "pending-mob", "pending-player", "remove-player", "transfer-mob", "transfer-player", "same-room-transfer", "set-room", "spell-transfer"} {
		t.Run(kind, func(t *testing.T) {
			w, ce, one, two, p, q := duplicateCleanupWorld(t)
			var dead combat.Combatant = one
			switch kind {
			case "death":
				one.SetHealth(-11)
				one.SetPosition(combat.PosDead)
				w.HandleDeath(one, p, combat.TYPE_UNDEFINED)
			case "raw-kill":
				w.RawKillCombatant(one, combat.TYPE_UNDEFINED)
			case "extract-mob":
				w.ExtractMob(one)
			case "pending-mob":
				one.SetMobFlag(MobFlagExtract)
				w.ExtractPendingChars()
			case "pending-player":
				dead = p
				w.QueuePlayerExtraction(p)
				w.ExtractPendingChars()
			case "remove-player":
				dead = p
				w.RemovePlayerBody(p)
			case "transfer-mob":
				if err := w.MobTransfer(one, 101); err != nil {
					t.Fatal(err)
				}
			case "transfer-player":
				dead = p
				if err := w.PlayerTransfer(p, 101); err != nil {
					t.Fatal(err)
				}
			case "same-room-transfer":
				if err := w.MobTransfer(one, 100); err != nil {
					t.Fatal(err)
				}
			case "set-room":
				one.SetRoom(101)
			case "spell-transfer":
				if err := w.TransferCombatant(one, 101); err != nil {
					t.Fatal(err)
				}
			}

			switch kind {
			case "death", "raw-kill", "extract-mob", "pending-mob", "pending-player", "remove-player":
				if err := ce.PerformUnenrolledInitialAttack(dead, two); err == nil {
					t.Fatal("removed body admitted a stale direct hit")
				}
			}
			if one.GetFightingBody() != nil || p.GetFightingBody() != nil {
				t.Fatalf("stale retired/departed references %p/%p", one.GetFightingBody(), p.GetFightingBody())
			}
			if two.GetFightingBody() != q || q.GetFightingBody() != two || !ce.IsFighting(two) {
				t.Fatal("duplicate survivor lost fight")
			}
			if combat.BodyRetired(two) || combat.BodyRetired(q) {
				t.Fatal("retired wrong duplicate")
			}
			if kind == "death" || kind == "raw-kill" {
				found := false
				for _, obj := range w.GetItemsInRoom(100) {
					if obj.IsCorpse {
						for _, item := range obj.Contains {
							if item.GetVNum() == 200 {
								found = true
							}
						}
					}
				}
				if !found {
					t.Fatal("corpse lost actual armed body's weapon")
				}
			}
			oldRoller := combat.GetRoller()
			combat.SetRoller(combat.NewSeededRoller(123))
			defer combat.SetRoller(oldRoller)
			var swings []combat.Combatant
			ce.MessageFunc = func(a, b combat.Combatant, _, _ int) bool {
				if a == dead || b == dead {
					t.Fatal("retired/departed body got another swing")
				}
				swings = append(swings, a)
				return true
			}
			ce.PerformRound()
			if len(swings) < 2 || swings[0] != q {
				t.Fatalf("survivor ordered swings=%v", swings)
			}
			found := false
			for _, body := range swings {
				if body == two {
					found = true
				}
			}
			if !found {
				t.Fatal("survivor NPC got no turn")
			}
			if weapon, _, _, _ := w.WireCombatCallbacks().GetWeaponInfo(two); weapon != 0 {
				t.Fatal("survivor inherited retired weapon")
			}
		})
	}
}

func TestCombatBodyRoundRetirement(t *testing.T) {
	w, ce, one, two, _, _ := duplicateCleanupWorld(t)
	old := combat.GetRoller()
	combat.SetRoller(combat.NewSeededRoller(123))
	defer combat.SetRoller(old)
	var retired bool
	ce.MessageFunc = func(a, b combat.Combatant, _, _ int) bool {
		if retired && (a == one || b == one) {
			t.Fatal("stale round snapshot revisited retired body")
		}
		if !retired {
			retired = true
			w.ExtractMob(one)
		}
		return true
	}
	ce.PerformRound()
	if !retired || !ce.IsFighting(two) {
		t.Fatal("round retirement disturbed survivor")
	}
	if err := ce.StartCombat(one, two); err == nil {
		t.Fatal("retired body re-enrolled")
	}
}

func TestCombatBodyConcurrentRetirement(t *testing.T) {
	w, ce, one, two, p, _ := duplicateCleanupWorld(t)
	ce.MessageFunc = func(_, _ combat.Combatant, _, _ int) bool { return true }
	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)
	go func() { defer workers.Done(); <-start; ce.PerformRound() }()
	go func() { defer workers.Done(); <-start; w.ExtractMob(one) }()
	close(start)
	workers.Wait()
	if one.GetFightingBody() != nil || p.GetFightingBody() != nil || !ce.IsFighting(two) {
		t.Fatal("concurrent cleanup lost exact ownership")
	}
	if err := ce.StartCombat(one, two); err == nil {
		t.Fatal("retired snapshot re-enrolled")
	}
}

func TestCombatBodyTeleportDuplicate(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}, Mobs: []parser.Mob{{VNum: 300, Keywords: "guard", ShortDesc: "Guard", Level: 20}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	first, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	first.SetPosition(combat.PosStanding)
	second.SetPosition(combat.PosStanding)
	// Find a deterministic room draw choosing the second runtime room.
	var seed uint32
	for seed = 1; seed < 100; seed++ {
		dprng.ResetStream(seed)
		index := dprng.Number(0, 1)
		room, _ := w.GetRoomVNumAtIndex(index)
		if room == 1002 {
			break
		}
	}
	dprng.ResetStream(seed)
	if !spells.CastFromSpecial(second, second, spells.SpellTeleport, 20, w) {
		t.Fatal("teleport not dispatched")
	}
	if second.GetRoom() != 1002 || first.GetRoom() != 1001 {
		t.Fatalf("teleported wrong duplicate: first=%d second=%d", first.GetRoom(), second.GetRoom())
	}
}

func TestCombatBodyRetiredLeader(t *testing.T) {
	w, ce, first, second, _, _ := duplicateCleanupWorld(t)
	follower, err := w.SpawnMobQuiet(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	other, err := w.SpawnMobQuiet(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	follower.SetFollowingBody(first)
	other.SetFollowingBody(second)
	w.ExtractMob(first)
	if w.combatFollowingBody(follower) != nil || w.combatFollowingBody(other) != second || !ce.IsFighting(second) {
		t.Fatal("leader retirement conflated duplicate relationships")
	}
}
