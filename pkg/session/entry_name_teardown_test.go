package session

import (
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// blockingSaveStore stops a descriptor's retirement inside cleanupSession:
// after the session has left the manager's map but before the send channel —
// and, without the fix, the entry-name claim — is released.
type blockingSaveStore struct {
	db.GameStore
	entered chan struct{}
	release chan struct{}
}

func (d *blockingSaveStore) SavePlayer(rec *db.PlayerRecord) error {
	select {
	case <-d.entered:
	default:
		close(d.entered)
	}
	<-d.release
	return d.GameStore.SavePlayer(rec)
}

// C frees a descriptor and the entry name it holds together: close_socket
// removes the descriptor from descriptor_list (comm.c:2148) and Valid_Name then
// stops seeing it (ban.c:266-268). A reconnect that no longer finds the session
// must therefore find the name free, even while its retirement is still
// running; otherwise a client that reconnects during teardown is refused a name
// its dropped descriptor no longer holds.
// TestEntryNameFreeOnceDescriptorLeavesManager fails without releasing the
// claim before the session leaves m.sessions (R5h).
func TestEntryNameFreeOnceDescriptorLeavesManager(t *testing.T) {
	store := &blockingSaveStore{
		GameStore: testutil.NewMockDatabase(),
		entered:   make(chan struct{}),
		release:   make(chan struct{}),
	}
	var once sync.Once
	unblock := func() { once.Do(func() { close(store.release) }) }
	t.Cleanup(unblock)

	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, store)

	// A descriptor past accepted stats and still pre-world: exactly the state
	// the new-character disconnect test reconnects into.
	owner := makeTestSession(t, m, "Freshwire", 8004, true)
	owner.creationSaved = true
	if !owner.claimEntryName("Freshwire") {
		t.Fatal("could not claim a free entry name")
	}
	registerTestSession(t, m, owner, "Freshwire")

	retired := make(chan struct{})
	go func() {
		m.UnregisterSession(owner)
		close(retired)
	}()

	// cleanupSession's DB save runs after the session has left m.sessions and
	// before CloseSend; hold retirement here to inspect the interval.
	select {
	case <-store.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("retirement never reached the DB save")
	}
	if _, exists := m.GetSession("Freshwire"); exists {
		t.Fatal("session still registered while retirement is mid-flight")
	}

	probe := makeCharSession(t, m)
	if !probe.claimEntryName("Freshwire") {
		t.Fatal("descriptor left the session map but its entry name is still held")
	}
	probe.CloseSend()

	unblock()
	select {
	case <-retired:
	case <-time.After(5 * time.Second):
		t.Fatal("retirement did not finish")
	}
}
