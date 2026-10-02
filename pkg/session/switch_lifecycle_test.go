package session

import (
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSwitchHandleTransportDisconnect(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	body := h.player
	if !m.HandleTransportDisconnect(s) {
		t.Fatal("switched transport not retained")
	}
	if s.player != original || s.isSwitched || s.hasTransport() {
		t.Fatal("closed descriptor left active attachment")
	}
	if !body.IsLinkless() || !original.IsLinkless() {
		t.Fatal("close_socket must leave both bodies descriptorless")
	}
	if m.world.GetPlayerCount() != 2 {
		t.Fatal("disconnect extracted a body")
	}
	if got := renderedOutput(s); strings.Contains(got, "return to your original") {
		t.Fatal("disconnect invented an explicit return")
	}
}

func TestSwitchPerformDupeCheckOriginal(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	fresh := makeTestSession(t, m, "Wizard", 1001, true)
	fresh.player = game.NewPlayer(original.GetID(), "Wizard", 1001)
	if !fresh.performDupeCheck() {
		t.Fatal("original did not reconnect")
	}
	if fresh.player != original || fresh.player.IsLinkless() {
		t.Fatal("UNSWITCH adopted the borrowed body")
	}
	if !strings.Contains(renderedOutput(fresh), "Reconnecting to unswitched char.") {
		t.Fatal("missing UNSWITCH mode")
	}
	if !h.player.IsLinkless() || m.world.GetPlayerCount() != 2 {
		t.Fatal("UNSWITCH lost borrowed body")
	}
	m.UnregisterSession(s)
	if m.world.GetPlayerCount() != 2 {
		t.Fatal("stale descriptor removed reconnected original")
	}
}

func TestSwitchPerformDupeCheckBorrowed(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	body := h.player
	fresh := makeTestSession(t, m, "Borrowed", 1002, true)
	fresh.player = game.NewPlayer(body.GetID(), "Borrowed", 1002)
	if !fresh.performDupeCheck() {
		t.Fatal("borrowed identity did not reconnect")
	}
	if fresh.player != body || body.IsLinkless() {
		t.Fatal("USURP adopted a candidate")
	}
	if !strings.Contains(renderedOutput(fresh), "You take over your own body, already in use!") {
		t.Fatal("borrowed descriptor was treated as linkdead")
	}
	if !original.IsLinkless() {
		t.Fatal("USURP failed to leave original descriptorless")
	}
	if got, _ := m.GetSession("Wizard"); got == nil || got.player != original || got.hasTransport() {
		t.Fatal("USURP lost original lifecycle holder")
	}
	m.UnregisterSession(s)
	m.UnregisterSession(h)
	if m.world.GetPlayerCount() != 2 {
		t.Fatal("stale teardown removed a live body")
	}
}

func TestSwitchExtractPendingChars(t *testing.T) {
	for _, which := range []string{"borrowed", "original", "borrowed-idle", "original-idle"} {
		t.Run(which, func(t *testing.T) {
			m, s, h, original := switchedPCFixture(t)
			victim := h.player
			if which == "original" || which == "original-idle" {
				victim = original
			}
			if which == "borrowed-idle" || which == "original-idle" {
				victim.IdleDisconnect = true
				victim.RentedOut = true
			}
			m.world.QueuePlayerExtraction(victim)
			m.ExtractPendingChars()
			if s.player != original || s.isSwitched {
				t.Fatal("extraction retained switched attachment")
			}
			if which == "original" || which == "original-idle" {
				if !s.menuActive || original.IsLinkless() || !h.player.IsLinkless() {
					t.Fatal("original extraction did not return descriptor to menu")
				}
			} else {
				if s.menuActive {
					t.Fatal("borrowed extraction put original at the menu")
				}
				if _, ok := m.GetSession("Borrowed"); ok {
					t.Fatal("extracted borrowed lifecycle holder survived")
				}
				if which == "borrowed-idle" && (s.hasTransport() || !original.IsLinkless()) {
					t.Fatal("idle close failed to detach original")
				}
			}
			if m.world.GetPlayerCount() != 1 {
				t.Fatal("extraction removed the other body")
			}
		})
	}
}

func TestSwitchReturnDisconnectsOccupiedOriginal(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	other := makeTestSession(t, m, "Otherwizard", 1001, true)
	other.player = game.NewPlayer(3, "Otherwizard", 1001)
	other.player.SetLevel(LVL_IMPL)
	other.transportDone = make(chan struct{})
	registerTestSession(t, m, other, "Otherwizard")
	if err := cmdSwitch(other, []string{"Wizard"}); err != nil {
		t.Fatal(err)
	}
	if other.player != original {
		t.Fatal("second switch did not attach to original")
	}
	renderedOutput(other)
	if err := cmdReturn(s, nil); err != nil {
		t.Fatal(err)
	}
	if other.hasTransport() || other.isSwitched || !other.player.IsLinkless() {
		t.Fatal("return did not close original occupant")
	}
	if s.player != original || original.IsLinkless() || !h.player.IsLinkless() {
		t.Fatal("return failed to reclaim original")
	}
}

func TestSwitchReadersConcurrentLifecycle(t *testing.T) {
	m, s, h := switchGateFixture(t)
	s.send = make(chan []byte, 10000)
	h.send = make(chan []byte, 10000)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			m.world.MessageSink("Borrowed", []byte("test\r\n"))
			m.attachedBody(h.player)
			getEffectiveLevel(s)
			m.reserveAndReleaseSwitchTestName()
			m.world.MovementLook(h.player)
		}
	}()
	for i := 0; i < 100; i++ {
		if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
			t.Fatal(err)
		}
		if err := cmdReturn(s, nil); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
}

func (m *Manager) reserveAndReleaseSwitchTestName() {
	if release, ok := m.reserveOfflineRename("Borrowed", "Unused"); ok {
		release()
	}
}

func TestSwitchReconnectCandidateObjects(t *testing.T) {
	for _, name := range []string{"Wizard", "Borrowed"} {
		t.Run(name, func(t *testing.T) {
			m, s, h, original := switchedPCFixture(t)
			for _, p := range []*game.Player{original, h.player} {
				obj, err := m.world.SpawnObject(8023, p.GetRoom())
				if err != nil {
					t.Fatal(err)
				}
				if err := m.world.MoveObject(obj, game.LocInventoryPlayer(p.GetName())); err != nil {
					t.Fatal(err)
				}
				p.SetPlrFlag(game.PlrCrash, false)
			}
			target := original
			if name == "Borrowed" {
				target = h.player
			}
			record, err := db.PlayerToRecord(target, nil)
			if err != nil {
				t.Fatal(err)
			}
			candidate, err := db.RecordToPlayer(record, m.world)
			if err != nil {
				t.Fatal(err)
			}
			if len(m.world.GetAllObjects()) != 3 {
				t.Fatal("separate candidate tree not loaded")
			}
			fresh := makeTestSession(t, m, name, target.GetRoom(), true)
			fresh.player = candidate
			if !fresh.performDupeCheck() || fresh.player != target {
				t.Fatal("live body not adopted")
			}
			if len(m.world.GetAllObjects()) != 2 || candidate.Inventory.GetItemCount() != 0 {
				t.Fatal("candidate objects leaked")
			}
			if original.Inventory.GetItemCount() != 1 || h.player.Inventory.GetItemCount() != 1 || target.NeedsCrashSave() {
				t.Fatal("cleanup touched retained same-name owner")
			}
			m.UnregisterSession(s)
			if m.world.GetPlayerCount() != 2 {
				t.Fatal("stale cleanup lost a body")
			}
		})
	}
}

func TestSwitchDisconnectConcurrentReconnect(t *testing.T) {
	for _, name := range []string{"Wizard", "Borrowed"} {
		t.Run(name, func(t *testing.T) {
			m, s, h, original := switchedPCFixture(t)
			target := original
			if name == "Borrowed" {
				target = h.player
			}
			fresh := makeTestSession(t, m, name, target.GetRoom(), true)
			fresh.player = game.NewPlayer(target.GetID(), name, target.GetRoom())
			start := make(chan struct{})
			done := make(chan bool, 1)
			go func() { <-start; done <- m.HandleTransportDisconnect(s) }()
			close(start)
			if !fresh.performDupeCheck() {
				t.Fatal("concurrent reconnect failed")
			}
			<-done
			if fresh.player != target || target.IsLinkless() || m.world.GetPlayerCount() != 2 {
				t.Fatal("concurrent teardown lost reconnect body")
			}
		})
	}
}
