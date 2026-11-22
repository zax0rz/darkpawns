package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type castleGuardEastCombatEngine struct {
	starts         [][2]string
	initialAttacks [][2]string
}

func (e *castleGuardEastCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	e.starts = append(e.starts, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *castleGuardEastCombatEngine) PerformInitialAttack(attacker, defender combat.Combatant) error {
	e.initialAttacks = append(e.initialAttacks, [2]string{attacker.GetName(), defender.GetName()})
	return nil
}

func (e *castleGuardEastCombatEngine) IsFighting(combat.Combatant) bool { return false }

func (e *castleGuardEastCombatEngine) GetCombatTarget(combat.Combatant) (combat.Combatant, bool) {
	return nil, false
}

// TestSpecCastleGuardEast_BlockAndHit pins SPECIAL(castle_guard_east)
// (src/spec_procs2.c:1934-1970): a non-owner moving east hears the yell via
// act TO_ROOM and is hit through the combat engine, returning TRUE. The
// allow branch snaps via act TO_ROOM and falls through FALSE.
func TestSpecCastleGuardEast_BlockAndHit(t *testing.T) {
	t.Run("block yells and hits through engine", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		lastMsg()
		peer := NewPlayer(2, "Peer", player.GetRoomVNum())
		if err := w.AddPlayer(peer); err != nil {
			t.Fatalf("AddPlayer peer: %v", err)
		}
		messages := make(map[string]string)
		w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
		engine := &castleGuardEastCombatEngine{}
		w.SetCombatEngine(engine)
		if got := specCastleGuardEast(w, player, guard, "east", ""); !got {
			t.Fatal("blocked east should be handled")
		}
		if got := messages[peer.GetName()]; !strings.Contains(got, "yells, 'Stay outta there!'") {
			t.Fatalf("peer = %q, want yell", got)
		}
		if len(engine.starts) != 1 {
			t.Fatalf("starts = %#v, want one hit at %q", engine.starts, player.GetName())
		}
		if got, want := engine.starts[0], [2]string{guard.GetName(), player.GetName()}; got != want {
			t.Fatalf("combat start = %#v, want guard attacking eastbound player %#v", got, want)
		}
	})
	t.Run("grouped owner snaps without hit", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		lastMsg()
		owner := NewPlayer(2, "Owner", player.GetRoomVNum())
		if err := w.AddPlayer(owner); err != nil {
			t.Fatalf("AddPlayer owner: %v", err)
		}
		player.SetFollowing(owner.GetName())
		w.HouseControl = []HouseControl{{VNum: player.GetRoomVNum() + 2, Owner: int64(owner.GetID())}}
		messages := make(map[string]string)
		w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
		engine := &castleGuardEastCombatEngine{}
		w.SetCombatEngine(engine)
		if got := specCastleGuardEast(w, player, guard, "east", ""); got {
			t.Fatal("grouped owner should pass through")
		}
		if got := messages[owner.GetName()]; !strings.Contains(got, "snaps to attention") {
			t.Fatalf("owner = %q, want snaps", got)
		}
		if len(engine.starts) != 0 {
			t.Fatalf("grouped owner must not be hit: %#v", engine.starts)
		}
	})
}

// TestSpecCastleGuardEast_AssistsOnlyEast pins the pulse scan
// (src/spec_procs2.c:1959-1968): only a fighting castle_guard_east peer
// is assisted, via the mob-hit seam.
func TestSpecCastleGuardEast_AssistsOnlyEast(t *testing.T) {
	cleanup := func(a, b int) {
		delete(MobSpecAssign, a)
		delete(MobSpecAssign, b)
	}
	t.Cleanup(func() { cleanup(99041, 99042) })
	MobSpecAssign[99041] = "castle_guard_east"
	MobSpecAssign[99042] = "castle_guard_north"

	t.Run("assists east peer", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		other := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		guard.VNum = 99041
		other.VNum = 99041
		other.SetFightingBody(player)
		lastMsg()
		engine := &castleGuardEastCombatEngine{}
		w.SetCombatEngine(engine)
		if got := specCastleGuardEast(w, nil, guard, "", ""); !got {
			t.Fatal("east peer should be assisted")
		}
		if len(engine.starts) != 1 {
			t.Fatalf("starts = %#v, want one", engine.starts)
		}
		if got, want := engine.starts[0], [2]string{guard.GetName(), player.GetName()}; got != want {
			t.Fatalf("combat start = %#v, want east guard assisting against its peer's target %#v", got, want)
		}
	})
	t.Run("ignores north peer", func(t *testing.T) {
		w, player, lastMsg := newSpecProcTestWorld(t)
		guard := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		other := newSpecProcTestMob(t, w, player.GetRoomVNum(), 10)
		guard.VNum = 99041
		other.VNum = 99042
		other.SetFightingBody(player)
		lastMsg()
		engine := &castleGuardEastCombatEngine{}
		w.SetCombatEngine(engine)
		if got := specCastleGuardEast(w, nil, guard, "", ""); got {
			t.Fatal("north peer must not be assisted")
		}
		if len(engine.starts) != 0 {
			t.Fatalf("starts = %#v, want none", engine.starts)
		}
	})
}
