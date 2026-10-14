package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMobileActivityRunsMobGreetAfterWander(t *testing.T) {
	w, wanderer := newMobactWanderTestWorld(t, SECT_FIELD)
	dog := NewMob(&parser.Mob{
		VNum: 8063, Keywords: "dog", ShortDesc: "a dog",
		Position: int(combat.PosSleeping), DefaultPos: int(combat.PosSleeping),
	}, 1002)
	dog.ID = 1
	w.mu.Lock()
	w.activeMobs[dog.ID] = dog
	w.nextMobID = 2
	w.mu.Unlock()
	observer := NewPlayer(1, "Observer", 1002)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	messages := make(map[string]string)
	w.MessageSink = func(name string, message []byte) { messages[name] += string(message) }

	var draws []mobactDrawRange
	oldMobactNumber, oldMobProgNumber := mobactNumber, mobProgNumber
	mobactNumber = func(from, to int) int {
		draws = append(draws, mobactDrawRange{from: from, to: to})
		if from == 0 && to == 18 {
			return 0 // wander north into the sleeping dog
		}
		return 1 // skip the sound branch
	}
	mobProgNumber = func(from, to int) int {
		draws = append(draws, mobactDrawRange{from: from, to: to})
		return 0 // the dog reacts to the arriving mobile
	}
	t.Cleanup(func() {
		mobactNumber, mobProgNumber = oldMobactNumber, oldMobProgNumber
	})

	w.MobileActivity()

	if got, want := wanderer.GetRoom(), 1002; got != want {
		t.Fatalf("wanderer room = %d, want %d", got, want)
	}
	assertMobactDraws(t, &draws, []mobactDrawRange{{0, 18}, {0, 5}, {0, 15}})
	if got := messages[observer.Name]; !strings.Contains(got, "dog") {
		t.Fatalf("observer greet output = %q, want the dog's greet action", got)
	}
}
