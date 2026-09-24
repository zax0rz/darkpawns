package session

import (
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newOlcZonePersistenceWorld(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 8004, Name: "A Burning Hut", Zone: 1},
			{VNum: 18201, Name: "Kir-Oshi Docks", Zone: 2},
		},
		Zones: []parser.Zone{
			{Number: 1, TopRoom: 8999},
			{Number: 2, TopRoom: 18999},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func drainOlcZoneMessages(s *Session) {
	for {
		select {
		case <-s.send:
		default:
			return
		}
	}
}

func TestOlcZoneSurvivesDisconnectAndReconnect(t *testing.T) {
	database, err := db.New("sqlite://" + filepath.Join(t.TempDir(), "players.db"))
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	record := &db.PlayerRecord{
		Name:      "Builder",
		RoomVNum:  8004,
		Level:     31,
		Health:    20,
		MaxHealth: 20,
		Mana:      20,
		MaxMana:   20,
		Move:      100,
		MaxMove:   100,
		Class:     game.ClassWarrior,
		Race:      game.RaceHuman,
		StatStr:   10,
		StatInt:   10,
		StatWis:   10,
		StatDex:   10,
		StatCon:   10,
		StatCha:   10,
		Inventory: []byte("[]"),
		Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}

	world := newOlcZonePersistenceWorld(t)
	m := newTestManager(t, world, database)
	builder := makeCharSession(t, m)
	if err := builder.handleLogin(loginMsg("Builder", "unused")); err != nil {
		t.Fatalf("login: %v", err)
	}
	_ = drainMsg(t, builder) // MOTD prompt.
	sendMenuInput(t, builder, "")
	_ = drainMsg(t, builder) // main menu prompt.
	sendMenuInput(t, builder, "1")
	if builder.menuActive {
		t.Fatal("builder remained in the login menu")
	}

	wizard := makeCommandTestSession(t, m, "Wizard", game.LVL_IMPL, 8004)
	if err := cmdSet(wizard, []string{"Builder", "olc", "2"}); err != nil {
		t.Fatalf("set Builder olc 2: %v", err)
	}
	if got := readSessionText(t, wizard); got != "Builder's olc set to 2.\r\n" {
		t.Fatalf("set acknowledgement = %q", got)
	}
	if builder.olcZone != 2 {
		t.Fatalf("live olc zone = %d, want 2", builder.olcZone)
	}

	// makeCharSession is intentionally minimal and does not create a transport
	// lifecycle channel; the real telnet constructor does. Supply that one
	// transport-owned channel so this test can exercise the disconnect path.
	builder.transportDone = make(chan struct{})
	if !m.HandleTransportDisconnect(builder) {
		t.Fatal("HandleTransportDisconnect did not retain the playing builder")
	}
	m.Unregister("Builder")
	stored, err := database.GetPlayer("Builder")
	if err != nil {
		t.Fatalf("GetPlayer after disconnect: %v", err)
	}
	if stored == nil || stored.OlcZone != 2 {
		t.Fatalf("stored olc zone = %+v, want 2", stored)
	}

	reconnected := makeCharSession(t, m)
	if err := reconnected.handleLogin(loginMsg("Builder", "unused")); err != nil {
		t.Fatalf("reconnect: %v", err)
	}
	_ = drainMsg(t, reconnected)
	sendMenuInput(t, reconnected, "")
	_ = drainMsg(t, reconnected)
	sendMenuInput(t, reconnected, "1")
	drainOlcZoneMessages(reconnected)
	if reconnected.olcZone != 2 {
		t.Fatalf("reconnected olc zone = %d, want 2", reconnected.olcZone)
	}
	if err := cmdRedit(reconnected, []string{"8004"}); err != nil {
		t.Fatalf("redit outside assigned zone: %v", err)
	}
	if got := readMsgText(t, reconnected); got != "You do not have permission to edit this zone.\r\n" {
		t.Fatalf("outside-zone refusal = %q", got)
	}
}
