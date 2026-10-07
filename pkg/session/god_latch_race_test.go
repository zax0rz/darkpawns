package session

import (
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// VULN-023 re-verification on current main: N concurrent character creations
// against an empty store must crown exactly one God. The benchmark's exploit
// test passed on the Sep-3 bench tree; creationMu (persistAcceptedCharacter,
// char_creation.go:574) has serialized count-vs-insert since Sep 8 — this
// test decides whether the race is closed by that lock or still live.
func TestConcurrentFirstCreationsCrownExactlyOne(t *testing.T) {
	database := testutil.NewMockDatabase()
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)

	const n = 16
	gods := make(chan int, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := makeCharSession(t, m)
			s.charName = string(rune('A'+i%26)) + "acer" + string(rune('0'+i%10))
			s.charClass = game.ClassWarrior
			s.charRace = game.RaceHuman
			s.charSex = 'M'
			s.charStats = game.CharStats{Str: 12, Int: 12, Wis: 12, Dex: 12, Con: 12, Cha: 12}
			s.charHometown = 'K'
			if err := s.persistAcceptedCharacter(); err != nil {
				t.Errorf("persist %d: %v", i, err)
				return
			}
			if s.player != nil && s.player.GetLevel() >= game.LVL_IMPL {
				gods <- i
			}
		}(i)
	}
	wg.Wait()
	close(gods)
	crowned := 0
	for range gods {
		crowned++
	}
	if crowned != 1 {
		t.Fatalf("concurrent empty-store creations crowned %d Gods, want exactly 1 (VULN-023 race live)", crowned)
	}
}
