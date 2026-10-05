package game

import (
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

func TestCombatBodyMovementWindow(t *testing.T) {
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
			go func() { _, err := w.MovePlayer(p, "north"); done <- err }()
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
			if kind == "cost" {
				if err != nil || p.GetRoom() != 1002 || p.GetMove() != 98 {
					t.Fatalf("fresh cost/state: err=%v room=%d move=%d", err, p.GetRoom(), p.GetMove())
				}
			} else if err == nil || p.GetMove() != 100 {
				t.Fatalf("stale movement committed: err=%v move=%d", err, p.GetMove())
			}
		})
	}
}
