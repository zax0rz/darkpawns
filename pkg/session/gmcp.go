package session

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/mudletmap"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// GMCP — the Generic MUD Communication Protocol (telnet option 201) — carries
// structured copies of what a player is shown, for Mudlet and other rich
// clients. It is out of band: a plain telnet client never negotiates it and
// receives exactly the bytes it always did, and every message below is sent
// after, never instead of, the C-faithful text for the same game moment.
//
// The rule every message keeps (R4): GMCP may restate what the text just told
// the player; it never tells them anything the text withheld. Rooms are
// reported only when a room is rendered (not in darkness or blindness), exits
// follow C's autoexit visibility, and a channel speaker is named exactly as
// the line named them.
//
// The message set is versioned per module through Core.Supports, the GMCP
// negotiation clients already speak; docs/gmcp.md is the reference.

// GMCPMessageSetVersion names the whole Dark Pawns message set. Bump it, and
// the module versions in gmcpModules, when a payload shape changes; clients
// read it from Client.GUI and from MSSP.
const GMCPMessageSetVersion = 1

// gmcpModules are the modules this server implements, at the version Core.Supports
// negotiation accepts.
var gmcpModules = map[string]int{
	"Char":         1,
	"Room":         1,
	"Comm.Channel": 1,
}

// MsgGMCP carries one encoded GMCP message from the session to a telnet
// transport. Other transports never receive it: only a telnet connection that
// negotiated option 201 enables GMCP on its session.
const MsgGMCP = "gmcp"

// GMCPData is the MsgGMCP payload. JSON is kept pre-encoded so the field order
// a client parses is the struct order below, not a map's.
type GMCPData struct {
	Package string `json:"package"`
	JSON    string `json:"json"`
}

// GMCP payload shapes. Field order and types are part of the protocol:
// clients parse them cold, so a field is never renamed or retyped within a
// module version.
type (
	gmcpCharName struct {
		Name string `json:"name"`
	}
	gmcpCharVitals struct {
		HP    int `json:"hp"`
		MaxHP int `json:"maxhp"`
		MP    int `json:"mp"`
		MaxMP int `json:"maxmp"`
		MV    int `json:"mv"`
		MaxMV int `json:"maxmv"`
	}
	gmcpCharStatus struct {
		Name  string `json:"name"`
		Level int    `json:"level"`
		Race  string `json:"race"`
		Class string `json:"class"`
		Gold  int    `json:"gold"`
	}
	gmcpRoomInfo struct {
		Num         int            `json:"num"`
		Name        string         `json:"name"`
		Area        string         `json:"area"`
		Environment string         `json:"environment"`
		Exits       map[string]int `json:"exits"`
	}
	gmcpChannelText struct {
		Channel string `json:"channel"`
		Talker  string `json:"talker"`
		Text    string `json:"text"`
	}
	gmcpClientGUI struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}
	// gmcpClientMap is Mudlet's MMP map-location message. Mudlet reads
	// "url"; the package also reads "version" to tell a new map from the one
	// it loaded.
	gmcpClientMap struct {
		URL     string `json:"url"`
		Version string `json:"version"`
	}
)

// mudletMapTTL bounds how stale the generated map may be after an OLC edit.
const mudletMapTTL = 5 * time.Minute

// gmcpExitKeys are the direction abbreviations GMCP mappers key exits by, in
// C's dirs[] order.
var gmcpExitKeys = map[string]string{
	"north": "n", "east": "e", "south": "s", "west": "w", "up": "u", "down": "d",
}

// gmcpState is a session's GMCP negotiation and change-tracking state.
type gmcpState struct {
	mu sync.Mutex
	// syncMu serializes gmcpSync from the command and tick goroutines, so a
	// state snapshot is always sent before any snapshot taken after it.
	syncMu  sync.Mutex
	enabled bool
	// negotiated is set by the first Core.Supports message. Until then every
	// module is sent, for clients that never negotiate modules.
	negotiated bool
	supports   map[string]int
	client     string
	version    string
	// last* hold the most recent payload sent, so a prompt only carries a
	// message when its values changed.
	lastName   string
	lastVitals string
	lastStatus string
}

// gmcpGUI is the Mudlet package the server offers through Client.GUI. It is
// empty until the server is configured with a package URL.
var gmcpGUI struct {
	mu      sync.RWMutex
	url     string
	version string
}

// gmcpMapURL is the public URL of the /darkpawns-map.xml endpoint, offered
// through Client.Map. Empty until configured.
var gmcpMapURL struct {
	mu  sync.RWMutex
	url string
}

// SetGMCPClientMap configures the Client.Map offer. An empty url disables it.
func SetGMCPClientMap(url string) {
	gmcpMapURL.mu.Lock()
	defer gmcpMapURL.mu.Unlock()
	gmcpMapURL.url = url
}

// MudletMap serves the generated Mudlet world map.
func (m *Manager) MudletMap() http.Handler { return m.mudletMap }

// SetGMCPClientGUI configures the Client.GUI offer: Mudlet downloads the
// package at url and reinstalls it whenever version changes. An empty url
// disables the offer.
func SetGMCPClientGUI(url, version string) {
	gmcpGUI.mu.Lock()
	defer gmcpGUI.mu.Unlock()
	gmcpGUI.url, gmcpGUI.version = url, version
}

// EnableGMCP turns GMCP on for this session. The telnet transport calls it
// once the client agrees to option 201; it is idempotent.
func (s *Session) EnableGMCP() {
	s.gmcp.mu.Lock()
	already := s.gmcp.enabled
	s.gmcp.enabled = true
	s.gmcp.mu.Unlock()
	if already {
		return
	}
	s.sendGMCPClientGUI()
	s.sendGMCPClientMap()
	if s.player != nil && s.IsAuthenticated() {
		s.gmcpSync()
	}
}

// GMCPEnabled reports whether the transport negotiated GMCP.
func (s *Session) GMCPEnabled() bool {
	s.gmcp.mu.Lock()
	defer s.gmcp.mu.Unlock()
	return s.gmcp.enabled
}

// HandleGMCP processes one client→server GMCP message.
func (s *Session) HandleGMCP(pkg, payload string) {
	switch strings.ToLower(pkg) {
	case "core.hello":
		var hello struct {
			Client  string `json:"client"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal([]byte(payload), &hello); err != nil {
			return
		}
		s.gmcp.mu.Lock()
		s.gmcp.client, s.gmcp.version = hello.Client, hello.Version
		s.gmcp.mu.Unlock()
		slog.Info("GMCP client identified", "client", hello.Client, "version", hello.Version)
	case "core.supports.set", "core.supports.add", "core.supports.remove":
		var entries []string
		if err := json.Unmarshal([]byte(payload), &entries); err != nil {
			return
		}
		s.applyGMCPSupports(strings.ToLower(pkg), entries)
	case "core.ping":
		s.sendGMCPRaw("Core.Ping", "")
	}
}

// applyGMCPSupports records a Core.Supports.Set/Add/Remove list. Entries are
// "Module version"; a module this server implements at a lower version than
// requested is still accepted at the server's version, the GMCP convention.
func (s *Session) applyGMCPSupports(op string, entries []string) {
	s.gmcp.mu.Lock()
	if op == "core.supports.set" || s.gmcp.supports == nil {
		s.gmcp.supports = make(map[string]int)
	}
	s.gmcp.negotiated = true
	added := false
	for _, entry := range entries {
		fields := strings.Fields(entry)
		if len(fields) == 0 {
			continue
		}
		module := strings.ToLower(fields[0])
		if op == "core.supports.remove" {
			delete(s.gmcp.supports, module)
			continue
		}
		version := 1
		if len(fields) > 1 {
			if n, err := parseGMCPVersion(fields[1]); err == nil {
				version = n
			}
		}
		if _, known := lookupGMCPModule(module); known {
			s.gmcp.supports[module] = version
			added = true
		}
	}
	// A client that has just enabled a module expects its current state, not
	// a wait for the next change.
	if added {
		s.gmcp.lastName, s.gmcp.lastVitals, s.gmcp.lastStatus = "", "", ""
	}
	s.gmcp.mu.Unlock()
	if added && s.player != nil && s.IsAuthenticated() {
		s.gmcpSync()
	}
}

func parseGMCPVersion(text string) (int, error) {
	var n int
	err := json.Unmarshal([]byte(text), &n)
	return n, err
}

// lookupGMCPModule reports whether a Core.Supports entry names something
// this server sends: an implemented module, a package inside one
// ("Char.Vitals"), or a parent of one ("Comm" for "Comm.Channel").
func lookupGMCPModule(module string) (int, bool) {
	for name, version := range gmcpModules {
		name = strings.ToLower(name)
		if name == module || strings.HasPrefix(module, name+".") || strings.HasPrefix(name, module+".") {
			return version, true
		}
	}
	return 0, false
}

// gmcpWants reports whether the client should receive package pkg: GMCP is
// on, and either the client never negotiated modules or it enabled pkg's
// module or a parent of it ("Char" covers "Char.Vitals").
func (s *Session) gmcpWants(pkg string) bool {
	s.gmcp.mu.Lock()
	defer s.gmcp.mu.Unlock()
	if !s.gmcp.enabled {
		return false
	}
	if !s.gmcp.negotiated {
		return true
	}
	name := strings.ToLower(pkg)
	for {
		if _, ok := s.gmcp.supports[name]; ok {
			return true
		}
		dot := strings.LastIndexByte(name, '.')
		if dot < 0 {
			return false
		}
		name = name[:dot]
	}
}

// sendGMCP encodes data and queues it for the transport, in order with the
// session's text.
func (s *Session) sendGMCP(pkg string, data interface{}) {
	if !s.gmcpWants(pkg) {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("GMCP marshal failed", "package", pkg, "error", err)
		return
	}
	s.sendGMCPRaw(pkg, string(payload))
}

// sendGMCPChanged is sendGMCP for state messages: it sends only when the
// payload differs from the last one sent through *last.
func (s *Session) sendGMCPChanged(pkg string, data interface{}, last *string) {
	if !s.gmcpWants(pkg) {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("GMCP marshal failed", "package", pkg, "error", err)
		return
	}
	s.gmcp.mu.Lock()
	if *last == string(payload) {
		s.gmcp.mu.Unlock()
		return
	}
	*last = string(payload)
	s.gmcp.mu.Unlock()
	s.sendGMCPRaw(pkg, string(payload))
}

func (s *Session) sendGMCPRaw(pkg, payload string) {
	msg, err := json.Marshal(ServerMessage{Type: MsgGMCP, Data: GMCPData{Package: pkg, JSON: payload}})
	if err != nil {
		slog.Error("GMCP envelope marshal failed", "package", pkg, "error", err)
		return
	}
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()
	if s.sendClosed {
		return
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("session send channel full — dropping GMCP", "player", s.playerName, "package", pkg)
	}
}

// sendGMCPClientGUI offers the configured Mudlet package. Client.GUI is a
// Mudlet core message rather than a negotiated module, so it is sent to every
// GMCP client; others ignore it.
func (s *Session) sendGMCPClientGUI() {
	gmcpGUI.mu.RLock()
	url, version := gmcpGUI.url, gmcpGUI.version
	gmcpGUI.mu.RUnlock()
	if url == "" {
		return
	}
	payload, err := json.Marshal(gmcpClientGUI{Version: version, URL: url})
	if err != nil {
		return
	}
	s.sendGMCPRaw("Client.GUI", string(payload))
}

// reofferGMCPClientGUI offers the Mudlet package again as the character
// enters the game. Mudlet 5.0.1 upgrades a package by uninstalling it and then
// downloading the new one, and when the download lands while the profile is
// still saving after the uninstall, the install can be lost: the player is
// left with no package until the next connection. A second offer finds the
// package missing and installs it fresh, a path without that race. A client
// that already has this version ignores it.
func (s *Session) reofferGMCPClientGUI() {
	if s.GMCPEnabled() {
		s.sendGMCPClientGUI()
	}
}

// sendGMCPClientMap offers the world map. Like Client.GUI it is a Mudlet
// core message, not a negotiated module.
func (s *Session) sendGMCPClientMap() {
	gmcpMapURL.mu.RLock()
	url := gmcpMapURL.url
	gmcpMapURL.mu.RUnlock()
	if url == "" {
		return
	}
	_, version := s.manager.mudletMap.Map()
	if version == "" {
		return
	}
	payload, err := json.Marshal(gmcpClientMap{URL: url, Version: version})
	if err != nil {
		return
	}
	s.sendGMCPRaw("Client.Map", string(payload))
}

// gmcpSync sends the character state messages whose values changed since the
// last send. It runs before every prompt, when a player enters the game, and
// after the silent regeneration tick.
func (s *Session) gmcpSync() {
	if s.player == nil || !s.GMCPEnabled() {
		return
	}
	s.gmcp.syncMu.Lock()
	defer s.gmcp.syncMu.Unlock()
	p := s.player
	s.sendGMCPChanged("Char.Name", gmcpCharName{Name: p.GetName()}, &s.gmcp.lastName)
	s.gmcpVitalsLocked()
	s.sendGMCPChanged("Char.Status", gmcpCharStatus{
		Name:  p.GetName(),
		Level: p.GetLevel(),
		Race:  gmcpTableName(game.PCRaceTypes, p.GetRace()),
		Class: gmcpTableName(game.PCClassTypes, p.GetClass()),
		Gold:  p.GetGold(),
	}, &s.gmcp.lastStatus)
}

// gmcpVitals sends Char.Vitals when hit points, mana, or movement changed.
func (s *Session) gmcpVitals() {
	if s.player == nil || !s.GMCPEnabled() {
		return
	}
	s.gmcp.syncMu.Lock()
	defer s.gmcp.syncMu.Unlock()
	s.gmcpVitalsLocked()
}

func (s *Session) gmcpVitalsLocked() {
	v := s.player.VitalsSnapshot()
	s.sendGMCPChanged("Char.Vitals", gmcpCharVitals{
		HP: v.Health, MaxHP: v.MaxHealth,
		MP: v.Mana, MaxMP: v.MaxMana,
		MV: v.Move, MaxMV: v.MaxMove,
	}, &s.gmcp.lastVitals)
}

func gmcpTableName(table []string, index int) string {
	if index < 0 || index >= len(table) {
		return ""
	}
	return table[index]
}

// gmcpRoomInfo reports a room the player was just shown.
func (s *Session) gmcpRoomInfo(roomVNum int) {
	if s.player == nil || !s.gmcpWants("Room.Info") {
		return
	}
	world := s.manager.world
	room := world.GetRoomInWorld(roomVNum)
	if room == nil {
		return
	}
	// The area and environment names are the downloadable map's, so a room
	// mapped live lands in the same Mudlet area as the downloaded one.
	environment, _ := mudletmap.Environment(room.Sector)
	s.sendGMCP("Room.Info", gmcpRoomInfo{
		Num:         room.VNum,
		Name:        room.Name,
		Area:        s.manager.mudletMap.AreaName(room.Zone),
		Environment: environment,
		Exits:       gmcpVisibleExits(room, s.player.GetLevel() >= game.LVL_IMMORT),
	})
}

// gmcpVisibleExits lists the exits C's do_auto_exits shows: every exit with a
// destination, except closed ones, which only immortals see.
func gmcpVisibleExits(room *parser.Room, immortal bool) map[string]int {
	exits := make(map[string]int, len(room.Exits))
	for direction, key := range gmcpExitKeys {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom <= 0 {
			continue
		}
		if exit.ExitInfo&parser.ExitClosed != 0 && !immortal {
			continue
		}
		exits[key] = exit.ToRoom
	}
	return exits
}

// gmcpChannelText mirrors one delivered channel line. The text is the line as
// sent, without its trailing line ending.
func (s *Session) gmcpChannelText(channel, talker, line string) {
	s.sendGMCP("Comm.Channel.Text", gmcpChannelText{
		Channel: channel,
		Talker:  talker,
		Text:    strings.TrimRight(line, "\r\n"),
	})
}

// gmcpObserver adapts the manager to game.OutOfBandObserver.
type gmcpObserver struct{ m *Manager }

func (o gmcpObserver) RoomShown(p *game.Player, roomVNum int) {
	if s, ok := o.m.GetSession(p.Name); ok && s != nil && s.player == p {
		s.gmcpRoomInfo(roomVNum)
	}
}

func (o gmcpObserver) ChannelLine(p *game.Player, channel, talker, line string) {
	if s, ok := o.m.GetSession(p.Name); ok && s != nil && s.player == p {
		s.gmcpChannelText(channel, talker, line)
	}
}

func (o gmcpObserver) PointUpdated() {
	o.m.mu.RLock()
	sessions := make([]*Session, 0, len(o.m.sessions))
	for _, s := range o.m.sessions {
		sessions = append(sessions, s)
	}
	o.m.mu.RUnlock()
	for _, s := range sessions {
		s.gmcpSync()
	}
}
