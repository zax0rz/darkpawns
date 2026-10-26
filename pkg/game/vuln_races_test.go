package game

import (
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Regression tests for the concurrency findings triaged on 2026-10-26
// (docs/triage/2026-10-26-openvuln-16-finding-triage.md). The port runs two
// command lanes (the 100ms heartbeat drain and each connection's readPump fast
// path) plus independent combat/zone goroutines, so any check-then-mutate
// sequence C's single-threaded interpreter serialized must be atomic here.

// --- BUG-R2-C2-A1-H4: bank transfers must be one atomic step ---------------

func TestBankTransferAtomicSemantics(t *testing.T) {
	t.Run("deposit moves coins and reports the amount", func(t *testing.T) {
		p := NewPlayer(1, "Banker", 1001)
		p.SetGold(100)
		p.SetBankGold(0)

		if got := p.DepositGold(60); got != 60 {
			t.Fatalf("DepositGold(60) = %d, want 60", got)
		}
		if p.GetGold() != 40 || p.GetBankGold() != 60 {
			t.Fatalf("after deposit 60: gold=%d bank=%d, want 40/60", p.GetGold(), p.GetBankGold())
		}
	})

	t.Run("deposit beyond carried gold is refused untouched", func(t *testing.T) {
		p := NewPlayer(1, "Banker", 1001)
		p.SetGold(40)
		p.SetBankGold(60)

		if got := p.DepositGold(1000); got != 0 {
			t.Fatalf("DepositGold(1000) = %d, want 0", got)
		}
		if p.GetGold() != 40 || p.GetBankGold() != 60 {
			t.Fatalf("refused deposit changed balances: gold=%d bank=%d", p.GetGold(), p.GetBankGold())
		}
	})

	t.Run("withdraw beyond the balance is refused untouched", func(t *testing.T) {
		p := NewPlayer(1, "Banker", 1001)
		p.SetGold(10)
		p.SetBankGold(60)

		if got := p.WithdrawBankGold(1000); got != 0 {
			t.Fatalf("WithdrawBankGold(1000) = %d, want 0", got)
		}
		if p.GetGold() != 10 || p.GetBankGold() != 60 {
			t.Fatalf("refused withdrawal changed balances: gold=%d bank=%d", p.GetGold(), p.GetBankGold())
		}
	})

	t.Run("non-positive amounts are no-ops", func(t *testing.T) {
		p := NewPlayer(1, "Banker", 1001)
		p.SetGold(10)
		p.SetBankGold(10)

		if p.DepositGold(0) != 0 || p.WithdrawBankGold(-5) != 0 {
			t.Fatal("non-positive amount was accepted")
		}
		if p.GetGold() != 10 || p.GetBankGold() != 10 {
			t.Fatalf("non-positive amount changed balances: gold=%d bank=%d", p.GetGold(), p.GetBankGold())
		}
	})
}

// TestBankConcurrentWithdrawPaysOnce drives the real defect site (specBank)
// from concurrent goroutines. Before the fix the balance check and the two
// read-modify-writes were separate p.mu acquisitions, so several withdrawals
// could all pass the check and all pay out — minting carried gold and driving
// the bank negative. Every pass of the check must consume the balance.
func TestBankConcurrentWithdrawPaysOnce(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Bank", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 8034, Keywords: "banker", ShortDesc: "the banker"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer w.StopAITicker()
	// No-op sink: the racers all send output, and a shared builder would race.
	w.MessageSink = func(string, []byte) {}

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	ch.SetGold(0)
	ch.SetBankGold(100)

	proto, ok := w.GetMobPrototype(8034)
	if !ok {
		t.Fatal("banker prototype missing")
	}
	banker := NewMob(proto, 1001)

	const racers = 32
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			specBank(w, ch, banker, "withdraw", "100")
		}()
	}
	close(start)
	wg.Wait()

	// Exactly one withdrawal can have succeeded, so the single 100-coin
	// balance must have moved to carried gold in full and the account must
	// land on zero — never negative, never paid twice.
	if ch.GetGold() != 100 || ch.GetBankGold() != 0 {
		t.Fatalf("after %d concurrent withdrawals of the whole balance: gold=%d bank=%d, want 100/0 "+
			"(bank must never go negative and the balance must pay out once)",
			racers, ch.GetGold(), ch.GetBankGold())
	}
}

// --- BUG-R2-C2-A1-H2: a money pile is credited exactly once ----------------

// TestGetCheckMoneyConsumesOnce: ExtractObject is idempotent, so a second
// call on the same pile used to credit the amount again.
func TestGetCheckMoneyConsumesOnce(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	ch.SetGold(0)

	money := newMoneyItem(6450, 7)
	registerTransferObject(w, money)
	if err := w.MoveObjectToPlayerInventory(money, ch); err != nil {
		t.Fatalf("seat money: %v", err)
	}

	w.getCheckMoney(ch, money)
	w.getCheckMoney(ch, money) // already consumed: must be a no-op

	if ch.GetGold() != 7 {
		t.Fatalf("one 7-coin pile credited %d gold, want 7", ch.GetGold())
	}
}

// TestGetFromRoomStaleSnapshotCreditedOnce: getFromRoom works from a snapshot
// copy, so two getters can both still name the same pile. The second getter
// must not be paid again, and must not resurrect the extracted object.
func TestGetFromRoomStaleSnapshotCreditedOnce(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	ch.SetGold(0)

	money := newMoneyItem(6451, 100)
	registerTransferObject(w, money)
	if err := w.MoveObjectToRoomFront(money, ch.GetRoomVNum()); err != nil {
		t.Fatalf("seat money on the floor: %v", err)
	}

	// Both calls model the same stale room snapshot, as two concurrent
	// getters would.
	w.performGetFromRoom(ch, money)
	w.performGetFromRoom(ch, money)

	if ch.GetGold() != 100 {
		t.Fatalf("one 100-coin pile credited %d gold, want 100", ch.GetGold())
	}
	if _, ok := ch.Inventory.FindItem("coins"); ok {
		t.Fatal("extracted pile was resurrected into the inventory")
	}
	if money.Location.Kind != ObjNowhere {
		t.Fatalf("consumed pile Location = %v, want ObjNowhere", money.Location.Kind)
	}
}

// --- BUG-R2-C2-A1-H7: a carried steal leaves exactly one copy -------------

func TestStealCarriedLeavesOneCopy(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	obj := newFlagItem(w, 7150, 1)
	if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}

	if got := driveCarriedSteal(t, w, thief, victim, "thing"); !got.success {
		t.Fatal("steal did not succeed")
	}
	if !holdsObject(thief, obj) {
		t.Fatal("thief did not receive the stolen item")
	}
	if holdsObject(victim, obj) {
		t.Fatal("victim still holds the stolen item")
	}
	if obj.Location.Kind != ObjInInventory {
		t.Fatalf("stolen item Location = %v, want ObjInInventory", obj.Location.Kind)
	}
}

// holdsObject reports whether p's inventory list names exactly this instance.
// Keyword lookup is not usable here: rounds reuse item names, and a duplicate
// is precisely a second list naming the same pointer.
func holdsObject(p *Player, obj *ObjectInstance) bool {
	for _, it := range p.Inventory.Items {
		if it == obj {
			return true
		}
	}
	return false
}

// TestStealConcurrentWithGiveKeepsOneCopy races the carried steal against a
// competing world-locked transfer of the same instance. The steal used to
// mutate both inventories outside w.mu, so the transfer could detach through
// the by-then-stale Location (a silent no-op — the victim's list no longer
// held it) and attach the same *ObjectInstance to a second inventory, after
// which the steal added it again: one instance reachable from two
// Inventory.Items lists.
//
// Deterministic proof rather than a timing lottery: the test holds w.mu
// itself and asserts the steal cannot complete until it is released. An
// unlocked transfer (the pre-fix code) finishes while the lock is held.
func TestStealCarriedTransferHoldsWorldLock(t *testing.T) {
	w, thief := newFlagWorld(t)
	victim := NewPlayer(2, "Victimus", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	obj := newFlagItem(w, 7151, 1)
	if err := w.MoveObjectToPlayerInventory(obj, victim); err != nil {
		t.Fatalf("carry: %v", err)
	}
	thief.Level = LVL_IMPL // percent stays below the skill roll: always succeeds
	thief.SetSkill(SkillSteal, 100)
	thief.SetPlrFlag(PlrOutlaw, true)
	thief.Stats.Str = 15
	thief.CopyBaseAttributes()

	w.mu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		stealInventoryItem(w, thief, victim, obj, -1,
			GetPronouns(thief.Name, thief.GetSex()), GetPronouns(victim.Name, victim.GetSex()))
	}()

	select {
	case <-done:
		w.mu.Unlock()
		t.Fatal("carried steal completed while w.mu was held: the transfer is not world-locked")
	case <-time.After(150 * time.Millisecond):
	}
	w.mu.Unlock()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("carried steal did not complete after w.mu was released")
	}
	if !holdsObject(thief, obj) {
		t.Fatal("carried steal did not move the item to the thief")
	}
	if holdsObject(victim, obj) {
		t.Fatal("carried steal left the item with the victim")
	}
}

// --- BUG-R2-C2-A1-H3 residual: doPut must not read roomItems unlocked -----

// newRoomContainer seats an ITEM_CONTAINER on room 1001's floor.
func newRoomContainer(t *testing.T, w *World, vnum int) *ObjectInstance {
	t.Helper()
	cont := NewObjectInstance(&parser.Obj{
		VNum: vnum, ShortDesc: "a chest", Keywords: "chest",
		TypeFlag: ITEM_CONTAINER, Values: [4]int{100, 0, -1, 0}, WearFlags: [4]int{1},
	}, -1)
	registerTransferObject(w, cont)
	if err := w.MoveObjectToRoomFront(cont, 1001); err != nil {
		t.Fatalf("seat container: %v", err)
	}
	return cont
}

func TestPutFindsRoomContainer(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	chest := newRoomContainer(t, w, 6500)
	gem := newTransferItem(6501, "a gem", "gem", 1)
	gem.SetWeight(0)
	registerTransferObject(w, gem)
	if err := w.MoveObjectToPlayerInventory(gem, ch); err != nil {
		t.Fatalf("carry gem: %v", err)
	}

	w.doPut(ch, nil, "put", "gem in chest")

	if _, ok := ch.Inventory.FindItem("gem"); ok {
		t.Fatal("gem still in inventory; the room container was not found")
	}
	if gem.Location.Kind != ObjInContainer || gem.Location.ContainerObjID != chest.ID {
		t.Fatalf("gem Location = %+v, want in container %d", gem.Location, chest.ID)
	}
}

// TestPutRoomLookupConcurrentWithRoomWrites hammers doPut's room-container
// lookup while another goroutine writes the same map under w.mu. The lookup
// used to index w.roomItems with no lock, which Go reports as an unrecoverable
// "concurrent map read and map write" throw. Run under -race.
func TestPutRoomLookupConcurrentWithRoomWrites(t *testing.T) {
	w, ch, _ := newDonateTestWorld(t)
	w.MessageSink = func(string, []byte) {}

	newRoomContainer(t, w, 6510)

	stop := make(chan struct{})
	var writer sync.WaitGroup
	writer.Add(1)
	go func() {
		defer writer.Done()
		// Add/remove pairs keep the room list tiny — the point is the map
		// store's hashWriting window, not list depth — so doPut's locked
		// snapshot stays O(1) and the loop stays fast.
		for i := 0; i < 40000; i++ {
			select {
			case <-stop:
				return
			default:
			}
			litter := newTransferItem(6600+i%8, "litter", "litter", 0)
			w.AddItemToRoomFront(litter, ch.GetRoomVNum())
			w.RemoveItemFromRoom(litter, ch.GetRoomVNum())
		}
	}()

	for i := 0; i < 2000; i++ {
		gem := newTransferItem(6700, "a gem", "gem", 1)
		gem.SetWeight(0)
		registerTransferObject(w, gem)
		if err := w.MoveObjectToPlayerInventory(gem, ch); err != nil {
			t.Fatalf("iteration %d: carry gem: %v", i, err)
		}
		w.doPut(ch, nil, "put", "gem in chest")
	}
	close(stop)
	writer.Wait()
}
