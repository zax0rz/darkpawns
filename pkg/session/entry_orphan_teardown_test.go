package session

import (
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
)

type entryTeardownSaveStore struct {
	db.GameStore
	first   sync.Once
	started chan struct{}
	release chan struct{}
}

func (d *entryTeardownSaveStore) SavePlayer(p *db.PlayerRecord) error {
	d.first.Do(func() { close(d.started) })
	<-d.release
	return d.GameStore.SavePlayer(p)
}

// src/comm.c:2086-2156 and interpreter.c:1528-1659 execute serially in
// C's game loop. A real Go teardown must not admit a dupe lookup between
// descriptor deletion and completion of the saved body's removal.
func TestEntryDuplicateTeardownSerialization(t *testing.T) {
	for _, mode := range []string{"unregister", "transport", "close", "admission", "shutdown"} {
		t.Run(mode, func(t *testing.T) {
			database := entryDatabase(t)
			entrySeed(t, database, "Returner")
			m := entryWorldManager(t, database)
			old := makeCharSession(t, m)
			if err := old.handleLogin(loginMsg("Returner", "oraclepass")); err != nil {
				t.Fatal(err)
			}
			renderedOutput(old)
			sendMenuInput(t, old, "")
			renderedOutput(old)
			sendMenuInput(t, old, "1")
			renderedOutput(old)
			rec, err := database.GetPlayer("Returner")
			if err != nil {
				t.Fatal(err)
			}
			fresh := makeCharSession(t, m)
			fresh.player, err = db.RecordToPlayer(rec, m.world)
			if err != nil {
				t.Fatal(err)
			}
			blocker := &entryTeardownSaveStore{GameStore: database, started: make(chan struct{}), release: make(chan struct{})}
			m.db = blocker
			cleanupDone := make(chan struct{})
			go func() {
				defer close(cleanupDone)
				switch mode {
				case "unregister", "admission":
					m.Unregister("Returner")
				case "transport":
					m.UnregisterSession(old)
				case "close":
					m.UnregisterAndClose("Returner")
				case "shutdown":
					m.ShutdownGracefullyWithoutNotice(time.Second)
				}
			}()
			select {
			case <-blocker.started:
			case <-time.After(5 * time.Second):
				t.Fatal("teardown did not reach save")
			}
			// This is the real production window, not a manually constructed orphan.
			if _, ok := m.GetSession("Returner"); ok {
				close(blocker.release)
				<-cleanupDone
				t.Fatal("expected descriptor removal before save")
			}
			if m.world.GetPlayerCount() != 1 {
				close(blocker.release)
				<-cleanupDone
				t.Fatal("body not retained during save")
			}
			checked := make(chan bool, 1)
			go func() {
				if mode == "admission" {
					if err := m.enterWorld("Returner", fresh); err != nil {
						checked <- true
						return
					}
					checked <- false
					return
				}
				checked <- fresh.performDupeCheck()
			}()
			premature := false
			select {
			case <-checked:
				premature = true
			case <-time.After(100 * time.Millisecond):
			}
			close(blocker.release)
			select {
			case <-cleanupDone:
			case <-time.After(5 * time.Second):
				t.Fatal("teardown deadlocked")
			}
			if !premature {
				select {
				case adopted := <-checked:
					if adopted {
						t.Error("lifecycle operation unexpectedly adopted or failed")
					}
				case <-time.After(5 * time.Second):
					t.Fatal("lookup did not resume")
				}
			}
			if premature {
				t.Error("duplicate lookup ran in descriptorless teardown window")
			}
			want := 0
			if mode == "admission" {
				want = 1
				if current, ok := m.GetSession("Returner"); !ok || current != fresh {
					t.Error("admitted body lacks its retained session")
				}
			}
			if m.world.GetPlayerCount() != want {
				t.Error("cleanup retained orphan body or removed replacement")
			}
		})
	}
}
