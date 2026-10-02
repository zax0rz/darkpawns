package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func switchedPCFixture(t *testing.T) (*Manager, *Session, *Session, *game.Player) {
	t.Helper()
	m, s, h := switchGateFixture(t)
	original := s.player
	if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
		t.Fatal(err)
	}
	renderedOutput(s)
	return m, s, h, original
}

func TestSwitchAttachAndReturn(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	if s.player != h.player || s.switchedOriginal != original {
		t.Fatal("descriptor not attached to acting PC")
	}
	if original.IsLinkless() == false || h.player.IsLinkless() {
		t.Fatal("character descriptor flags not transferred")
	}
	if m.attachedBody(h.player) != s || m.attachedBody(original) != nil {
		t.Fatal("attached lookup disagrees")
	}
	if getEffectiveLevel(s) != h.player.GetLevel() {
		t.Fatal("PC command gates use original level")
	}
	if err := cmdSwitch(s, nil); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); got != "You're already switched.\r\n" {
		t.Fatalf("already-switched priority = %q", got)
	}
	if err := cmdReturn(s, nil); err != nil {
		t.Fatal(err)
	}
	if s.player != original || s.isSwitched || original.IsLinkless() || !h.player.IsLinkless() {
		t.Fatal("return did not restore both bodies")
	}
	if got := renderedOutput(s); got != "You return to your original body.\r\n" {
		t.Fatalf("return = %q", got)
	}
	if m.attachedBody(original) != s || m.attachedBody(h.player) != nil {
		t.Fatal("return descriptors wrong")
	}
}

func TestSwitchOutputRouting(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	m.world.MessageSink(h.player.GetName(), []byte("borrowed output\r\n"))
	m.world.MessageSink(original.GetName(), []byte("original output\r\n"))
	if got := renderedOutput(s); got != "borrowed output\r\n" {
		t.Fatalf("actor output = %q", got)
	}
	if got := renderedOutput(h); got != "" {
		t.Fatalf("dormant holder received %q", got)
	}
	if got, _ := m.GetSession("Wizard"); got != s {
		t.Fatal("identity registry no longer holds wizard")
	}
}

func TestSwitchMovementLook(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	m.world.MovementLook(h.player)
	if got := renderedOutput(s); !strings.Contains(got, "Room B") || strings.Contains(got, "Room A") {
		t.Fatalf("acting room look = %q", got)
	}
	if got := renderedOutput(h); got != "" {
		t.Fatalf("holder room look = %q", got)
	}
	m.world.MovementLook(original)
	if got := renderedOutput(s); got != "" {
		t.Fatalf("descriptorless original look = %q", got)
	}
}

type switchSaveStore struct {
	db.GameStore
	records []*db.PlayerRecord
}

func (d *switchSaveStore) SavePlayer(p *db.PlayerRecord) error {
	d.records = append(d.records, p)
	return nil
}

func TestSwitchPlayerSaver(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	store := &switchSaveStore{}
	m.db = store
	m.hasDB = true
	s.olcZone = 11
	h.olcZone = 22
	m.WirePlayerSaver(m.world)
	for _, p := range []*game.Player{original, h.player} {
		if got := m.world.PlayerSaver(p, "switch test", game.LoadRoomNowhere); got != game.SaveSucceeded {
			t.Fatalf("save %s = %v", p.GetName(), got)
		}
	}
	if len(store.records) != 2 || store.records[0].Name != "Wizard" || store.records[0].OlcZone != 11 || store.records[1].Name != "Borrowed" || store.records[1].OlcZone != 22 {
		t.Fatalf("saved wrong body or descriptor metadata: %+v", store.records)
	}
	foreign := game.NewPlayer(99, "Borrowed", 1002)
	if got := m.world.PlayerSaver(foreign, "foreign", game.LoadRoomNowhere); got != game.SaveSkipped {
		t.Fatal("same-name candidate saved live body")
	}
}

func TestSwitchDP1381Holds(t *testing.T) {
	m, _, h, original := switchedPCFixture(t)
	// Isolate the descriptor reader from the redundant world-body fallback.
	// A live name edit leaves its old registry key, as in main's existing proof.
	original.Name = "Originalalias"
	h.player.Name = "Bodyalias"
	world := m.world
	m.world = nil
	defer func() { m.world = world }()
	for _, name := range []string{"Wizard", "Borrowed", "Originalalias", "Bodyalias"} {
		if release, allowed := m.reserveOfflineRename(name, "Unused"); allowed {
			release()
			t.Fatalf("DP-1381 failed to hold %s", name)
		}
		if release, allowed := m.reserveOfflineRename("Offline", strings.ToUpper(name)); allowed {
			release()
			t.Fatalf("destination case fold failed for %s", name)
		}
	}
}

func TestSwitchDescriptorReaders(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	if findSessionForPlayer(m, h.player) != s || findSessionForPlayer(m, original) != nil {
		t.Fatal("wizard concrete reader uses identity holder")
	}
	if findSessionByName(m, "Borrowed") != s || findSessionByName(m, "Wizard") != nil {
		t.Fatal("wizard named reader uses original")
	}
	var bodies []*game.Player
	m.EachSession(func(p interface{}, _ func(string)) { bodies = append(bodies, p.(*game.Player)) })
	if len(bodies) != 1 || bodies[0] != h.player {
		t.Fatalf("descriptor list duplicate/dormant body: %v", bodies)
	}
}

func TestSwitchSnoopProtectsOriginalLevel(t *testing.T) {
	m, _, h, _ := switchedPCFixture(t)
	spy := makeTestSession(t, m, "Spy", 1002, true)
	spy.player.SetLevel(39)
	registerTestSession(t, m, spy, "Spy")
	if err := cmdSnoop(spy, []string{h.player.GetName()}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(spy); got != "You can't.\r\n" {
		t.Fatalf("snoop forgot switched original protection: %q", got)
	}
}

func TestSwitchWaitDrainsOnce(t *testing.T) {
	m, _, h, _ := switchedPCFixture(t)
	h.player.SetWaitStatePulses(3)
	m.DrainInputQueues()
	if got := h.player.GetWaitState(); got != 2 {
		t.Fatalf("wait decremented through both holder and descriptor: %d", got)
	}
}

func TestSwitchOrdinaryReadersMatchMain(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Ordinary", 1001, true)
	registerTestSession(t, m, s, "Ordinary")
	// Preserve even main's linkdead queue behavior outside active switching.
	s.player.SetLinkless(true)
	m.world.MessageSink("ordinary", []byte("ordinary output\r\n"))
	if got := renderedOutput(s); got != "ordinary output\r\n" {
		t.Fatalf("no-switch output changed: %q", got)
	}
	m.world.MovementLook(s.player)
	if got := renderedOutput(s); !strings.Contains(got, "Room A") {
		t.Fatalf("no-switch look changed: %q", got)
	}
	if findSessionForPlayer(m, s.player) != s || findSessionByName(m, "ordinary") != s {
		t.Fatal("no-switch lookup changed")
	}
}
