package session

import (
	"testing"
	"time"
)

func TestAdminOfflineReservationSerializesEntry(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Returner")
	manager := entryWorldManager(t, database)
	session := makeCharSession(t, manager)
	release, allowed := manager.ReserveOfflinePlayer("returner")
	if !allowed {
		t.Fatal("offline character blocked")
	}
	done := make(chan bool, 1)
	started := make(chan struct{})
	go func() { close(started); done <- session.claimEntryName("Returner") }()
	<-started
	select {
	case <-done:
		t.Fatal("entry bypassed offline reservation")
	case <-time.After(20 * time.Millisecond):
	}
	release()
	select {
	case ok := <-done:
		if !ok {
			t.Fatal("entry did not resume")
		}
	case <-time.After(time.Second):
		t.Fatal("entry stuck")
	}
	if unlock, ok := manager.ReserveOfflinePlayer("RETURNER"); ok {
		unlock()
		t.Fatal("menu/entry name was editable")
	}
	session.releaseEntryName()
}
