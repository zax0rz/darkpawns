package game

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRangedDeathHasNoKillerBookkeeping(t *testing.T) {
	for _, npc := range []bool{false, true} {
		t.Run(map[bool]string{false: "player", true: "mob"}[npc], func(t *testing.T) {
			w, p := rawKillWorld(t)
			shooter := NewPlayer(2, "Shooter", 1001)
			shooter.SetLevel(20)
			shooter.SetExp(5000)
			shooter.Kills = 7
			shooter.PKs = 4
			if err := w.AddPlayer(shooter); err != nil {
				t.Fatal(err)
			}
			p.SetExp(903)
			p.SetHP(5)
			p.Deaths = 6
			p.SetLastDeath(12345)
			p.Stats.Con = 12
			p.CopyBaseAttributes()
			con := p.GetCon()
			var victim combat.Combatant = p
			if npc { // Real runtime mobile in the canonical world registry.
				m := NewMob(&parser.Mob{VNum: 3001, Keywords: "guard", ShortDesc: "a guard", Exp: 903}, 1001)
				m.SetHealth(5)
				exp := 903
				m.Runtime.ExpOverride = &exp
				w.mu.Lock()
				w.activeMobs[m.ID] = m
				w.mu.Unlock()
				victim = m
			}
			if !w.ApplyRangedProjectileDamage(victim, 16) {
				t.Fatal("lethal shot did not enter die")
			}
			if shooter.GetExp() != 5000 || shooter.Kills != 7 || shooter.PKs != 4 || shooter.HasPLRFlag(PlrOutlaw) {
				t.Fatal("ranged death awarded kill credit or PK state")
			}
			if npc {
				m := victim.(*MobInstance)
				if m.GetExp() != 602 || m.IsAlive() || len(w.GetAllMobs()) != 0 {
					t.Fatal("mob die EXP/extraction boundary wrong")
				}
			}
			if !npc {
				if p.GetLastDeath() != 12345 {
					t.Fatal("ranged death updated death timestamp")
				}
				if p.GetExp() != 602 || p.Deaths != 6 || p.GetCon() != con || !p.HasPLRFlag(PlrExtract) {
					t.Fatalf("die exp/deaths/con/extract=%d/%d/%d/%t", p.GetExp(), p.Deaths, p.GetCon(), p.HasPLRFlag(PlrExtract))
				}
			}
			if len(w.GetItemsInRoom(1001)) != 1 {
				t.Fatal("ranged death did not create exactly one corpse")
			}
		})
	}
}

func TestRangedDirectHPThresholdsWithoutWoundMessages(t *testing.T) {
	for _, hp := range []int{5, 0, -2, -3, -5, -6, -10, -11} {
		w, p := rawKillWorld(t)
		p.SetHP(10)
		wire := ""
		w.MessageSink = func(_ string, msg []byte) { wire += string(msg) }
		dead := w.ApplyRangedProjectileDamage(p, 10-hp)
		if dead != (hp <= -11) {
			t.Fatalf("HP %d dead=%t", hp, dead)
		}
		if !dead && p.GetPosition() != combat.GetPositionFromHP(hp, combat.PosStanding) {
			t.Fatalf("HP %d position=%d", hp, p.GetPosition())
		}
		if !dead && wire != "" {
			t.Fatalf("update_pos invented wound bytes %q", wire)
		}
	}
}

func TestRangedTransferPreservesPostureAndCircle(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}}, Mobs: []parser.Mob{{VNum: 3001, Keywords: "guard", ShortDesc: "a guard"}}, Objs: []parser.Obj{{VNum: 64, Keywords: "circle", ShortDesc: "a circle"}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	m, err := w.SpawnMobQuiet(3001, 1002)
	if err != nil {
		t.Fatal(err)
	}
	m.SetPosition(combat.PosSleeping)
	observer := NewPlayer(1, "Observer", 1002)
	observer.SetFightingBody(m)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	first, err := w.SpawnObject(64, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoomFront(first, 1001); err != nil {
		t.Fatal(err)
	}
	first.SetTimer(8)
	last, err := w.SpawnObject(64, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveObjectToRoomFront(last, 1001); err != nil {
		t.Fatal(err)
	}
	last.SetTimer(9)
	head := w.GetItemsInRoom(1001)[0]
	if err := w.TransferRangedVictim(m, 1001); err != nil {
		t.Fatal(err)
	}
	if m.GetRoom() != 1001 || m.GetPosition() != combat.PosSleeping || observer.GetFighting() != m.GetName() {
		t.Fatal("bare ranged transfer stopped combat or changed posture")
	}
	if head.GetTimer() != map[bool]int{true: 7, false: 8}[head == first] {
		t.Fatal("bare char_to_room did not decrement first circle")
	}
	if first != head && first.GetTimer() != 8 || last != head && last.GetTimer() != 9 {
		t.Fatal("transfer decremented more than first circle")
	}
}

func TestRangedDeathExpLimits(t *testing.T) {
	for _, c := range []struct{ level, exp, want int }{{20, 0, 0}, {20, 1, 1}, {20, 903, 602}, {20, 900000, 600000}, {20, 3000000, 2500000}, {31, 903, 903}} {
		w, p := rawKillWorld(t)
		p.SetLevel(c.level)
		p.SetExp(c.exp)
		p.SetHP(1)
		w.ApplyRangedProjectileDamage(p, 12)
		if p.GetExp() != c.want {
			t.Fatalf("level %d EXP %d -> %d want %d", c.level, c.exp, p.GetExp(), c.want)
		}
	}
}

func TestRangedRetaliationHunterCallback(t *testing.T) {
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	w, p := rawKillWorld(t)
	p.SetHP(1000)
	p.SetMaxHP(1000)
	m := NewMob(&parser.Mob{VNum: 3001, Keywords: "guard", ShortDesc: "a guard", Level: 10}, 1001)
	m.SetHealth(1000)
	m.SetMobFlag(MobFlagHunter)
	w.mu.Lock()
	w.activeMobs[m.ID] = m
	w.mu.Unlock()
	cb := w.WireCombatCallbacks()
	cb.DamageRefused = func(combat.Combatant, combat.Combatant) bool { return false }
	combat.SetCallbacks(cb)
	ce := combat.NewCombatEngine()
	ce.MessageFunc = func(combat.Combatant, combat.Combatant, int, int) bool { return true }
	if err := ce.PerformRangedRetaliation(m, p); err != nil {
		t.Fatal(err)
	}
	if m.GetHunting() != p.GetName() {
		t.Fatal("ranged hunter did not retain prey after damage enrollment")
	}
}

func TestRangedDamageConcurrentHPUpdates(t *testing.T) {
	w, p := rawKillWorld(t)
	p.SetHP(100000)
	var group sync.WaitGroup
	for i := 0; i < 32; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for j := 0; j < 100; j++ {
				w.ApplyRangedProjectileDamage(p, 1)
			}
		}()
	}
	group.Wait()
	if p.GetHP() != 96800 {
		t.Fatalf("concurrent direct HP updates lost damage: %d", p.GetHP())
	}
}
