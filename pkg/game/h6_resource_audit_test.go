package game

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// H6 handler audit: each test drives one resource a concurrent command could
// double and asserts it is consumed exactly once. They are revert-failing —
// putting the pre-fix check-then-act back makes every racer succeed.

// racers runs fn n times concurrently from one starting gun and returns the
// number of calls that reported success.
func racers(n int, fn func() bool) int64 {
	var granted int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			if fn() {
				atomic.AddInt64(&granted, 1)
			}
		}()
	}
	close(start)
	wg.Wait()
	return granted
}

// assertSerialized asserts fn does not complete while the caller holds the lock
// that guards the resource, proving the operation is atomic on it. An unlocked
// check-then-act finishes immediately, so this fails on a reverted fix.
func assertSerialized(t *testing.T, name string, lock, unlock func(), fn func()) {
	t.Helper()
	lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
		unlock()
		t.Fatalf("%s completed while its lock was held: the check-then-act is not serialized", name)
	case <-time.After(150 * time.Millisecond):
	}
	unlock()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s did not complete after the lock was released", name)
	}
}

// TestSpendManaConsumedOnce: cast/turn reserved the cost, so one pool pays once.
func TestSpendManaConsumedOnce(t *testing.T) {
	p := NewPlayer(1, "Caster", 1001)
	p.SetMana(50)
	assertSerialized(t, "TrySpendMana", func() { p.mu.Lock() }, func() { p.mu.Unlock() },
		func() { p.TrySpendMana(50) })
	p.SetMana(50)

	if got := racers(16, func() bool { return p.TrySpendMana(50) }); got != 1 {
		t.Fatalf("one 50-point pool paid out %d times, want 1", got)
	}
	if p.GetMana() != 0 {
		t.Fatalf("mana = %d, want 0", p.GetMana())
	}
}

// TestSpendManaUpToNeverNegative: C deducts unconditionally after its check, so
// concurrent casts must clamp rather than drive mana below zero.
func TestSpendManaUpToNeverNegative(t *testing.T) {
	p := NewPlayer(1, "Caster", 1001)
	p.SetMana(50)
	assertSerialized(t, "SpendManaUpTo", func() { p.mu.Lock() }, func() { p.mu.Unlock() },
		func() { p.SpendManaUpTo(50) })
	p.SetMana(50)

	var taken int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			atomic.AddInt64(&taken, int64(p.SpendManaUpTo(50)))
		}()
	}
	close(start)
	wg.Wait()

	if taken != 50 {
		t.Fatalf("mana deducted %d times over, want exactly the 50 available", taken)
	}
	if p.GetMana() != 0 {
		t.Fatalf("mana = %d, want 0", p.GetMana())
	}
}

// TestMagicItemChargeConsumedOnce: a one-charge wand must fire once.
func TestMagicItemChargeConsumedOnce(t *testing.T) {
	w, _ := newFlagWorld(t)
	wand := NewObjectInstance(&parser.Obj{
		VNum: 7800, ShortDesc: "a wand", Keywords: "wand",
		Values: [4]int{0, 0, 1, 0},
	}, -1)
	registerTransferObject(w, wand)
	assertSerialized(t, "takeMagicItemCharge", func() { w.mu.Lock() }, func() { w.mu.Unlock() },
		func() { w.takeMagicItemCharge(wand) })
	wand.SetValue(2, 1)

	if got := racers(16, func() bool { return w.takeMagicItemCharge(wand) }); got != 1 {
		t.Fatalf("one-charge wand fired %d times, want 1", got)
	}
	if wand.GetValue(2) != 0 {
		t.Fatalf("charges = %d, want 0", wand.GetValue(2))
	}
}

// TestDrinkDoseConsumedOnce: a one-dose container must pour once.
func TestDrinkDoseConsumedOnce(t *testing.T) {
	w, _ := newFlagWorld(t)
	flask := NewObjectInstance(&parser.Obj{
		VNum: 7801, ShortDesc: "a flask", Keywords: "flask",
		Values: [4]int{0, 1, 0, 0},
	}, -1)
	registerTransferObject(w, flask)
	assertSerialized(t, "claimDrinkDose", func() { w.mu.Lock() }, func() { w.mu.Unlock() },
		func() { w.claimDrinkDose(flask) })
	flask.SetValue(1, 1)

	if got := racers(16, func() bool { _, ok := w.claimDrinkDose(flask); return ok }); got != 1 {
		t.Fatalf("one-dose flask poured %d times, want 1", got)
	}
	if flask.GetValue(1) != 0 {
		t.Fatalf("liquid left = %d, want 0", flask.GetValue(1))
	}
}

// TestSpendPracticesConsumedOnce: one session must buy one practice.
func TestSpendPracticesConsumedOnce(t *testing.T) {
	p := NewPlayer(1, "Student", 1001)
	p.SetPractices(1)
	assertSerialized(t, "TrySpendPractices", func() { p.mu.Lock() }, func() { p.mu.Unlock() },
		func() { p.TrySpendPractices(1) })
	p.SetPractices(1)

	if got := racers(16, func() bool { return p.TrySpendPractices(1) }); got != 1 {
		t.Fatalf("one practice session bought %d practices, want 1", got)
	}
	if p.GetPractices() != 0 {
		t.Fatalf("practices = %d, want 0", p.GetPractices())
	}
}

// TestClanTreasureWithdrawPaysOnce: one treasury balance must pay out once.
func TestClanTreasureWithdrawPaysOnce(t *testing.T) {
	c := &Clan{Treasure: 100}
	assertSerialized(t, "SpendTreasure", func() { c.mu.Lock() }, func() { c.mu.Unlock() },
		func() { c.SpendTreasure(100) })
	c.Treasure = 100

	if got := racers(16, func() bool { return c.SpendTreasure(100) }); got != 1 {
		t.Fatalf("one 100-coin treasury paid out %d times, want 1", got)
	}
	if c.TreasureBalance() != 0 {
		t.Fatalf("treasury = %d, want 0 (never negative)", c.TreasureBalance())
	}
}

// TestConsumeSpellReagentOnce drives the real reagent path: one carried
// component must buy one spell, not two.
func TestConsumeSpellReagentOnce(t *testing.T) {
	w, p := newFlagWorld(t)
	// The check-and-remove window is narrow, so race it enough times that a
	// reverted guard is caught: the pre-fix shape double-consumes early.
	const rounds = 200
	for r := 0; r < rounds; r++ {
		reagent := newFlagItem(w, 7802+r, 1) // keywords "thing"
		if err := w.MoveObjectToPlayerInventory(reagent, p); err != nil {
			t.Fatalf("round %d: carry reagent: %v", r, err)
		}
		if got := racers(32, func() bool { return p.ConsumeSpellReagent("thing") }); got != 1 {
			t.Fatalf("round %d: one reagent bought %d spells, want 1", r, got)
		}
		if n := p.Inventory.GetItemCount(); n != 0 {
			t.Fatalf("round %d: inventory holds %d items, want the reagent consumed", r, n)
		}
	}
}

// TestStealCoinsConcurrentConservesGold drives the real coin-steal path: the
// thief gains exactly what the victim loses.
func TestStealCoinsConcurrentConservesGold(t *testing.T) {
	thief := NewPlayer(1, "Thief", 1001)
	victim := NewPlayer(2, "Mark", 1001)
	victim.SetGold(1000)
	thief.SetGold(0)
	chP := GetPronouns(thief.Name, thief.GetSex())
	vP := GetPronouns(victim.Name, victim.GetSex())

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			// percent = -1: the steal always succeeds.
			stealCoins(thief, victim, -1, chP, vP)
		}()
	}
	close(start)
	wg.Wait()

	if got := thief.GetGold() + victim.GetGold(); got != 1000 {
		t.Fatalf("gold after concurrent coin steals = %d, want 1000 (minting)", got)
	}
	if victim.GetGold() < 0 || thief.GetGold() < 0 {
		t.Fatalf("negative balance: thief %d, victim %d", thief.GetGold(), victim.GetGold())
	}
}

// TestDonateGoldConcurrentJunksOnce drives performDisposeGold: one balance must
// fund one junk, not two. The lost-update shape the fix removes is invisible in
// the final balance (both writers land on the same value), so the assertion
// counts the acknowledgements instead.
func TestDonateGoldConcurrentJunksOnce(t *testing.T) {
	w, ch := newFlagWorld(t)
	var mu sync.Mutex
	junks := 0
	w.MessageSink = func(_ string, msg []byte) {
		if strings.Contains(string(msg), "You drop some gold which disappears in a puff of smoke!") {
			mu.Lock()
			junks++
			mu.Unlock()
		}
	}
	ch.SetGold(100)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			w.performDisposeGold(ch, 60, scmdJunk, 0)
		}()
	}
	close(start)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if junks != 1 {
		t.Fatalf("%d junks spent a 100-coin balance twice over, want 1", junks)
	}
	if ch.GetGold() != 40 {
		t.Fatalf("gold = %d, want 40 (one 60-coin junk)", ch.GetGold())
	}
}
