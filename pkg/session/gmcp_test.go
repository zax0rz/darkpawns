package session

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type queued struct {
	kind, pkg, payload string
}

// drainQueued returns everything waiting on the session's send channel.
func drainQueued(t *testing.T, s *Session) []queued {
	t.Helper()
	var out []queued
	for {
		select {
		case raw := <-s.send:
			var msg struct {
				Type string          `json:"type"`
				Data json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				t.Fatalf("queued message: %v", err)
			}
			entry := queued{kind: msg.Type}
			if msg.Type == MsgGMCP {
				var data GMCPData
				if err := json.Unmarshal(msg.Data, &data); err != nil {
					t.Fatalf("GMCP envelope: %v", err)
				}
				entry.pkg, entry.payload = data.Package, data.JSON
			}
			out = append(out, entry)
		default:
			return out
		}
	}
}

func gmcpPackages(entries []queued) []string {
	var pkgs []string
	for _, e := range entries {
		if e.kind == MsgGMCP {
			pkgs = append(pkgs, e.pkg)
		}
	}
	return pkgs
}

func newGMCPSession(t *testing.T) *Session {
	t.Helper()
	lit := []string{"0", "0", "0", "0"}
	w, err := game.NewWorld(&parser.World{
		Zones: []parser.Zone{{Number: 1, Name: "Proving Grounds", TopRoom: 1099}},
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room A", Zone: 1, Sector: 3, Flags: lit, Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 1002},
				"east":  {Direction: "east", ToRoom: 1002, ExitInfo: parser.ExitIsDoor | parser.ExitClosed},
			}},
			{VNum: 1002, Name: "Room B", Zone: 1, Sector: 99, Flags: lit},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Walker", 1001, true)
	m.mu.Lock()
	m.sessions["walker"] = s
	m.mu.Unlock()
	return s
}

func TestGMCPOffUntilNegotiated(t *testing.T) {
	s := newGMCPSession(t)
	s.gmcpSync()
	s.gmcpRoomInfo(1001)
	s.gmcpChannelText("say", "Walker", "You say 'hi'\r\n")
	if got := drainQueued(t, s); len(got) != 0 {
		t.Fatalf("session without GMCP queued %+v", got)
	}
}

// TestGMCPSupportsGating follows Core.Supports: every module until the client
// negotiates, then only the modules (or parents/children of them) it enabled.
func TestGMCPSupportsGating(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	drainQueued(t, s)

	if !s.gmcpWants("Comm.Channel.Text") || !s.gmcpWants("Room.Info") {
		t.Fatal("a client that has not negotiated modules should receive every module")
	}

	s.HandleGMCP("Core.Supports.Set", `["Char 1","Char.Skills 1","Room 1"]`)
	for pkg, want := range map[string]bool{
		"Char.Vitals": true, "Char.Status": true, "Room.Info": true, "Comm.Channel.Text": false,
	} {
		if got := s.gmcpWants(pkg); got != want {
			t.Fatalf("after Set, gmcpWants(%s) = %v", pkg, got)
		}
	}

	s.HandleGMCP("Core.Supports.Add", `["Comm 1"]`)
	if !s.gmcpWants("Comm.Channel.Text") {
		t.Fatal(`"Comm 1" should enable Comm.Channel`)
	}
	s.HandleGMCP("Core.Supports.Remove", `["Room"]`)
	if s.gmcpWants("Room.Info") {
		t.Fatal("Remove did not disable Room")
	}
	s.HandleGMCP("Core.Supports.Set", `["Char.Vitals 1"]`)
	if !s.gmcpWants("Char.Vitals") || s.gmcpWants("Char.Status") {
		t.Fatal("a package-level Supports entry should enable only that package")
	}
}

// TestGMCPSupportsAddSendsCurrentState: a module switched on mid-session gets
// its current state immediately rather than at the next change.
func TestGMCPSupportsAddSendsCurrentState(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	s.HandleGMCP("Core.Supports.Set", `["Room 1"]`)
	drainQueued(t, s)
	s.HandleGMCP("Core.Supports.Add", `["Char 1"]`)
	if got := gmcpPackages(drainQueued(t, s)); !reflect.DeepEqual(got, []string{"Char.Name", "Char.Vitals", "Char.Status"}) {
		t.Fatalf("Add Char sent %v", got)
	}
}

func TestGMCPCharStateShapesAndDedupe(t *testing.T) {
	s := newGMCPSession(t)
	s.player.SetClass(game.ClassMageUser)
	s.player.Race = game.RaceElf
	s.player.SetLevel(7)
	s.EnableGMCP()

	got := drainQueued(t, s)
	want := []queued{
		{kind: MsgGMCP, pkg: "Char.Name", payload: `{"name":"Walker"}`},
		{kind: MsgGMCP, pkg: "Char.Vitals", payload: gmcpVitalsJSON(s)},
		{kind: MsgGMCP, pkg: "Char.Status", payload: `{"name":"Walker","level":7,"race":"Elven","class":"Magic User","gold":0}`},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("initial state:\n got %+v\nwant %+v", got, want)
	}

	s.gmcpSync()
	if got := drainQueued(t, s); len(got) != 0 {
		t.Fatalf("unchanged state was resent: %+v", got)
	}

	s.player.SetMove(s.player.GetMove() - 3)
	s.gmcpSync()
	if got := drainQueued(t, s); len(got) != 1 || got[0].pkg != "Char.Vitals" || got[0].payload != gmcpVitalsJSON(s) {
		t.Fatalf("movement change sent %+v", got)
	}
}

func gmcpVitalsJSON(s *Session) string {
	v := s.player.VitalsSnapshot()
	raw, _ := json.Marshal(gmcpCharVitals{HP: v.Health, MaxHP: v.MaxHealth, MP: v.Mana, MaxMP: v.MaxMana, MV: v.Move, MaxMV: v.MaxMove})
	return string(raw)
}

// TestGMCPStateRidesAheadOfPrompt: the prompt that shows new vitals is
// preceded by the Char.Vitals that reports them.
func TestGMCPStateRidesAheadOfPrompt(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	drainQueued(t, s)
	s.player.SetHP(s.player.GetHP() - 1)
	s.SendPrompt()
	got := drainQueued(t, s)
	if len(got) != 2 || got[0].pkg != "Char.Vitals" || got[1].kind != MsgPrompt {
		t.Fatalf("prompt sequence = %+v", got)
	}
}

func TestGMCPRoomInfoExitsFollowAutoexit(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	drainQueued(t, s)

	s.gmcpRoomInfo(1001)
	s.gmcpRoomInfo(1002)
	got := drainQueued(t, s)
	want := []queued{
		{kind: MsgGMCP, pkg: "Room.Info", payload: `{"num":1001,"name":"Room A","area":"Proving Grounds","environment":"Forest","exits":{"n":1002}}`},
		{kind: MsgGMCP, pkg: "Room.Info", payload: `{"num":1002,"name":"Room B","area":"Proving Grounds","environment":"Unknown","exits":{}}`},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mortal Room.Info:\n got %+v\nwant %+v", got, want)
	}

	// do_auto_exits shows immortals closed exits too.
	s.player.SetLevel(game.LVL_IMMORT)
	s.gmcpRoomInfo(1001)
	if got := drainQueued(t, s); len(got) != 1 || got[0].payload != `{"num":1001,"name":"Room A","area":"Proving Grounds","environment":"Forest","exits":{"e":1002,"n":1002}}` {
		t.Fatalf("immortal Room.Info = %+v", got)
	}
}

func TestGMCPObserverRoutesToThePlayersSession(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	drainQueued(t, s)
	observer := gmcpObserver{m: s.manager}

	observer.RoomShown(s.player, 1001)
	observer.ChannelLine(s.player, "tell", "Someone", "Someone tells you, 'boo'\r\n")
	other := game.NewPlayer(9, "Walker", 1001) // same name, different character
	observer.RoomShown(other, 1002)

	got := drainQueued(t, s)
	if len(got) != 2 || got[0].pkg != "Room.Info" || got[1].pkg != "Comm.Channel.Text" {
		t.Fatalf("observer delivered %+v", got)
	}
	if got[1].payload != `{"channel":"tell","talker":"Someone","text":"Someone tells you, 'boo'"}` {
		t.Fatalf("Comm.Channel.Text = %s", got[1].payload)
	}
}

func TestGMCPClientGUIOffer(t *testing.T) {
	t.Cleanup(func() { SetGMCPClientGUI("", "") })

	s := newGMCPSession(t)
	s.EnableGMCP()
	if pkgs := gmcpPackages(drainQueued(t, s)); len(pkgs) > 0 && pkgs[0] == "Client.GUI" {
		t.Fatal("Client.GUI offered with no package configured")
	}

	SetGMCPClientGUI("https://example.invalid/darkpawns.mpackage", "3")
	s = newGMCPSession(t)
	s.EnableGMCP()
	got := drainQueued(t, s)
	if len(got) == 0 || got[0].pkg != "Client.GUI" || got[0].payload != `{"version":"3","url":"https://example.invalid/darkpawns.mpackage"}` {
		t.Fatalf("Client.GUI offer = %+v", got)
	}

	s.EnableGMCP() // a repeated DO must not re-offer
	if got := drainQueued(t, s); len(got) != 0 {
		t.Fatalf("second EnableGMCP queued %+v", got)
	}
}

func TestGMCPCorePing(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	drainQueued(t, s)
	s.HandleGMCP("Core.Ping", "")
	if got := drainQueued(t, s); len(got) != 1 || got[0].pkg != "Core.Ping" || got[0].payload != "" {
		t.Fatalf("Core.Ping reply = %+v", got)
	}
}

// TestGMCPDoesNotMakeASessionStructured guards the pager regression: GMCP
// must never switch on the agent/structured-client mode, which bypasses
// page_string and so changes the text a player reads.
func TestGMCPDoesNotMakeASessionStructured(t *testing.T) {
	s := newGMCPSession(t)
	s.EnableGMCP()
	s.HandleGMCP("Core.Hello", `{"client":"Mudlet","version":"4.19.1"}`)
	if s.WantsStructuredData() {
		t.Fatal("GMCP negotiation made the session a structured client")
	}
}

// TestGMCPClientMapOffer: with a map URL configured, a GMCP client is told
// where the world map is and which version it is, the version the map
// endpoint serves.
func TestGMCPClientMapOffer(t *testing.T) {
	t.Cleanup(func() { SetGMCPClientMap("") })

	s := newGMCPSession(t)
	s.EnableGMCP()
	for _, pkg := range gmcpPackages(drainQueued(t, s)) {
		if pkg == "Client.Map" {
			t.Fatal("Client.Map offered with no map URL configured")
		}
	}

	SetGMCPClientMap("https://example.invalid/darkpawns-map.xml")
	s = newGMCPSession(t)
	s.EnableGMCP()
	var offer string
	for _, entry := range drainQueued(t, s) {
		if entry.pkg == "Client.Map" {
			offer = entry.payload
		}
	}
	_, version := s.manager.mudletMap.Map()
	if want := `{"url":"https://example.invalid/darkpawns-map.xml","version":"` + version + `"}`; offer != want || version == "" {
		t.Fatalf("Client.Map = %s, want %s", offer, want)
	}
}
