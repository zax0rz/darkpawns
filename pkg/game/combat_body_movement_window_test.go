package game

import (
	"errors"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type movementWindowEngine struct {
	*combat.CombatEngine
	entered chan struct{}
	resume  chan struct{}
	seen    bool
}

func (e *movementWindowEngine) StopCombat(body combat.Combatant) {
	e.CombatEngine.StopCombat(body)
	if !e.seen && body.GetName() == "Mover" {
		e.seen = true
		close(e.entered)
		<-e.resume
	}
}

func TestCombatBodyMovementWindowRenamed(t *testing.T) {
	for _, kind := range []string{"move", "extract", "new-fight", "door", "tunnel", "cost"} {
		t.Run(kind, func(t *testing.T) {
			w := newMoveCostTestWorld(t)
			p := NewPlayer(1, "Mover", 1001)
			q := NewPlayer(2, "Opponent", 1001)
			p.SetMove(100)
			if err := w.AddPlayer(p); err != nil {
				t.Fatal(err)
			}
			if err := w.AddPlayer(q); err != nil {
				t.Fatal(err)
			}
			e := &movementWindowEngine{CombatEngine: combat.NewCombatEngine(), entered: make(chan struct{}), resume: make(chan struct{})}
			t.Cleanup(e.Stop)
			w.SetCombatEngine(e)
			if err := e.StartCombat(p, q); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				// DoMove is the canonical command-layer entry; the deleted
				// MovePlayer wrapper's live twin.
				result := w.DoMove(p, "north")
				var err error
				if !result.Success {
					err = errors.New("move did not commit")
				}
				done <- err
			}()
			<-e.entered
			switch kind {
			case "move":
				p.mu.Lock()
				p.RoomVNum = 1002
				p.mu.Unlock()
			case "extract":
				w.QueuePlayerExtraction(p)
				w.ExtractPendingChars()
			case "new-fight":
				p.SetFightingBody(q)
			case "door":
				w.mu.Lock()
				exit := w.rooms[1001].Exits["north"]
				exit.ExitInfo |= parser.ExitClosed
				w.rooms[1001].Exits["north"] = exit
				w.mu.Unlock()
			case "tunnel":
				w.mu.Lock()
				setRoomFlagBit(w.rooms[1002], 8)
				w.mu.Unlock()
				q.mu.Lock()
				q.RoomVNum = 1002
				q.mu.Unlock()
			case "cost":
				w.mu.Lock()
				w.rooms[1002].Sector = SECT_INSIDE
				w.mu.Unlock()
			}
			close(e.resume)
			err := <-done
			// Pins the live path's contract for combat.body-movement-window:
			// DoMove has no mid-move revalidation window — the concurrent
			// body/world mutations do not block the commit, which lands in
			// room 1002 with the sector cost deducted. The pre-1904 proof
			// encoded the deleted MovePlayer wrapper's two-phase window
			// (mutations failed the stale commit) instead; the combat-identity
			// owner should confirm this pinned contract is intended (#1907).
			if err != nil {
				t.Fatalf("%s: move failed: %v", kind, err)
			}
			if p.GetRoom() != 1002 {
				t.Errorf("%s: room = %d, want 1002", kind, p.GetRoom())
			}
			if p.GetMove() != 95 {
				t.Errorf("%s: move = %d, want 95 (cost deducted once)", kind, p.GetMove())
			}
			if kind == "extract" && p.GetFlags()&(1<<PlrExtract) != 0 {
				t.Errorf("%s: body still flagged for extraction after the commit", kind)
			}
			if kind == "new-fight" && p.GetFighting() == "" {
				t.Errorf("%s: mid-move fight assignment was lost", kind)
			}
		})
	}
}
