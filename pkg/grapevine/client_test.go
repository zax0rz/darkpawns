package grapevine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// fakeRelay is a scriptable Grapevine relay: it records every frame and
// connection, answers authenticate with a top-level status (the real
// protocol shape), and can then send heartbeats and a broadcast.
type fakeRelay struct {
	mu         sync.Mutex
	connTimes  []time.Time
	frames     []relayFrame
	authStatus string // "success", "failure", or "close4000"
	heartbeats int
	broadcast  []byte
	srv        *httptest.Server
}

type relayFrame struct {
	Event   string          `json:"event"`
	Status  string          `json:"status"`
	Payload json.RawMessage `json:"payload"`
}

func newFakeRelay(t *testing.T, authStatus string) *fakeRelay {
	t.Helper()
	r := &fakeRelay{authStatus: authStatus}
	upgrader := websocket.Upgrader{}
	r.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(w, req, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		r.mu.Lock()
		r.connTimes = append(r.connTimes, time.Now())
		heartbeats, broadcast := r.heartbeats, r.broadcast
		r.mu.Unlock()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var f relayFrame
			if err := json.Unmarshal(data, &f); err != nil {
				continue
			}
			r.mu.Lock()
			r.frames = append(r.frames, f)
			r.mu.Unlock()
			if f.Event != "authenticate" {
				continue
			}
			if r.authStatus == "close4000" {
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(4000, "authentication failed"),
					time.Now().Add(time.Second))
				return
			}
			resp := fmt.Sprintf(`{"event":"authenticate","status":%q,"payload":{"unicode":"✔️","version":"2.3.0"}}`, r.authStatus)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(resp)); err != nil {
				return
			}
			if r.authStatus != "success" {
				continue
			}
			if broadcast != nil {
				if err := conn.WriteMessage(websocket.TextMessage, broadcast); err != nil {
					return
				}
			}
			for i := 0; i < heartbeats; i++ {
				if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"heartbeat"}`)); err != nil {
					return
				}
			}
		}
	}))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *fakeRelay) setScript(heartbeats int, broadcast []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.heartbeats, r.broadcast = heartbeats, broadcast
}

func (r *fakeRelay) url() string {
	return strings.Replace(r.srv.URL, "http://", "ws://", 1) + "/socket"
}

func (r *fakeRelay) snapshot() ([]time.Time, []relayFrame) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]time.Time(nil), r.connTimes...), append([]relayFrame(nil), r.frames...)
}

func (r *fakeRelay) framesWithEvent(event string) []relayFrame {
	_, frames := r.snapshot()
	var out []relayFrame
	for _, f := range frames {
		if f.Event == event {
			out = append(out, f)
		}
	}
	return out
}

func waitForCond(t *testing.T, timeout time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

type authPayload struct {
	ClientID string   `json:"client_id"`
	Supports []string `json:"supports"`
	Channels []string `json:"channels"`
}

func firstAuthPayload(t *testing.T, r *fakeRelay) authPayload {
	t.Helper()
	frames := r.framesWithEvent("authenticate")
	if len(frames) == 0 {
		t.Fatal("relay received no authenticate frame")
	}
	var p authPayload
	if err := json.Unmarshal(frames[0].Payload, &p); err != nil {
		t.Fatalf("unmarshal auth payload: %v", err)
	}
	return p
}

func testWorld(t *testing.T) *game.World {
	t.Helper()
	world, err := game.NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}
	return world
}

func testPlayer(name string, level int) *game.Player {
	p := game.NewCharacterWithStats(0, name, game.ClassWarrior, game.RaceHuman, 0,
		game.CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10})
	p.Level = level
	return p
}

func addPlayer(t *testing.T, world *game.World, p *game.Player) {
	t.Helper()
	if err := world.AddPlayer(p); err != nil {
		t.Fatalf("add player %s: %v", p.Name, err)
	}
}

// sinkRecorder captures everything players would be sent via the world's
// MessageSink.
type sinkRecorder struct {
	mu   sync.Mutex
	msgs map[string][]string
}

func newSinkRecorder() *sinkRecorder { return &sinkRecorder{msgs: map[string][]string{}} }

func (s *sinkRecorder) sink(name string, msg []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgs[name] = append(s.msgs[name], string(msg))
}

func (s *sinkRecorder) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, msgs := range s.msgs {
		out = append(out, msgs...)
	}
	return out
}

func startClient(t *testing.T, world *game.World, relay *fakeRelay, mode string, backoff time.Duration) *Client {
	t.Helper()
	t.Setenv("GRAPEVINE_CLIENT_ID", "test_client")
	t.Setenv("GRAPEVINE_CLIENT_SECRET", "test_secret")
	t.Setenv("GRAPEVINE_URL", relay.url())
	if mode != "" {
		t.Setenv("GRAPEVINE_MODE", mode)
	}
	client := NewClient(world)
	if backoff > 0 {
		client.reconnectBackoff = backoff
	}
	client.Start()
	t.Cleanup(client.Stop)
	return client
}

// Check 1 (presence half) + the original handshake test, updated to the
// real protocol: supports is required, channels ride the auth payload,
// and there is no separate subscribe step.
func TestGrapevineClient(t *testing.T) {
	relay := newFakeRelay(t, "success")
	world := testWorld(t)
	startClient(t, world, relay, "", 0)

	waitForCond(t, 2*time.Second, "authenticate frame", func() bool {
		return len(relay.framesWithEvent("authenticate")) >= 1
	})
	auth := firstAuthPayload(t, relay)
	if auth.ClientID != "test_client" {
		t.Errorf("auth client_id = %q, want test_client", auth.ClientID)
	}
	foundChannels := false
	for _, s := range auth.Supports {
		if s == "channels" {
			foundChannels = true
		}
	}
	if !foundChannels {
		t.Errorf("auth supports = %v, want it to contain \"channels\"", auth.Supports)
	}
	if len(auth.Channels) != 0 {
		t.Errorf("presence auth channels = %v, want []", auth.Channels)
	}

	// No channels/subscribe frame may follow in presence mode.
	time.Sleep(250 * time.Millisecond)
	if got := relay.framesWithEvent("channels/subscribe"); len(got) != 0 {
		t.Errorf("received %d channels/subscribe frames, want 0", len(got))
	}
}

// Check 1 (full half) + check 4 (outbound): in full mode the auth
// payload subscribes to gossip, and local gossip produces channels/send
// once the bridge is installed after a successful auth.
func TestGrapevineOutboundGossip(t *testing.T) {
	relay := newFakeRelay(t, "success")
	world := testWorld(t)
	gossiper := testPlayer("tester", 5)
	addPlayer(t, world, gossiper)
	startClient(t, world, relay, "full", 0)

	waitForCond(t, 2*time.Second, "authenticate frame", func() bool {
		return len(relay.framesWithEvent("authenticate")) >= 1
	})
	auth := firstAuthPayload(t, relay)
	if len(auth.Channels) != 1 || auth.Channels[0] != "gossip" {
		t.Errorf("full auth channels = %v, want [gossip]", auth.Channels)
	}

	// The bridge installs asynchronously after auth success; gossip
	// until the relay sees the relayed copy.
	waitForCond(t, 3*time.Second, "channels/send relay of local gossip", func() bool {
		world.DoChannel(gossiper, "hello grapevine", "gossip")
		for _, f := range relay.framesWithEvent("channels/send") {
			var payload struct {
				Channel string `json:"channel"`
				Name    string `json:"name"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(f.Payload, &payload); err != nil {
				continue
			}
			if payload.Channel == "gossip" && payload.Name == "tester" && payload.Message == "hello grapevine" {
				return true
			}
		}
		return false
	})
}

// Check 2: heartbeats are answered with the mortal-visible player list.
// A wizinvis immortal is absent; a visible mortal and a guest are
// present. Names are bare — the relay knows which game this connection
// belongs to; the Name@Game form is only for displaying remote players
// in-game.
func TestGrapevineHeartbeatPresence(t *testing.T) {
	relay := newFakeRelay(t, "success")
	relay.setScript(3, nil)
	world := testWorld(t)
	addPlayer(t, world, testPlayer("Mortal", 5))
	addPlayer(t, world, testPlayer("Guest", 1))
	hidden := testPlayer("Hidden", game.LVL_IMPL)
	hidden.InvisLevel = game.LVL_IMPL
	addPlayer(t, world, hidden)
	startClient(t, world, relay, "", 0)

	waitForCond(t, 3*time.Second, "three heartbeat replies", func() bool {
		return len(relay.framesWithEvent("heartbeat")) >= 3
	})
	for i, f := range relay.framesWithEvent("heartbeat") {
		var payload struct {
			Players []string `json:"players"`
		}
		if err := json.Unmarshal(f.Payload, &payload); err != nil {
			t.Fatalf("heartbeat %d payload: %v", i, err)
		}
		joined := strings.Join(payload.Players, ",")
		if !strings.Contains(joined, "Mortal") {
			t.Errorf("heartbeat %d players = %v, want Mortal present", i, payload.Players)
		}
		if !strings.Contains(joined, "Guest") {
			t.Errorf("heartbeat %d players = %v, want Guest present", i, payload.Players)
		}
		if strings.Contains(joined, "Hidden") {
			t.Errorf("heartbeat %d players = %v, wizinvis immortal leaked", i, payload.Players)
		}
		if strings.Contains(joined, "@") {
			t.Errorf("heartbeat %d players = %v, want bare names without a game suffix", i, payload.Players)
		}
	}
}

// Check 3: presence mode is silent. A relay broadcast reaches no player,
// and local gossip produces no channels/send.
func TestGrapevinePresenceIsSilent(t *testing.T) {
	relay := newFakeRelay(t, "success")
	relay.setScript(0, []byte(`{"event":"channels/broadcast","payload":{"channel":"gossip","game":"OtherMUD","name":"Peer","message":"hello from elsewhere"}}`))
	world := testWorld(t)
	rec := newSinkRecorder()
	world.MessageSink = rec.sink
	local := testPlayer("Local", 5)
	addPlayer(t, world, local)
	startClient(t, world, relay, "", 0)

	waitForCond(t, 2*time.Second, "authenticate frame", func() bool {
		return len(relay.framesWithEvent("authenticate")) >= 1
	})
	// Give the broadcast time to (not) arrive, then gossip locally.
	time.Sleep(300 * time.Millisecond)
	world.DoChannel(local, "local words", "gossip")
	time.Sleep(300 * time.Millisecond)

	for _, msg := range rec.all() {
		if strings.Contains(msg, "hello from elsewhere") || strings.Contains(msg, "[Grapevine]") {
			t.Errorf("presence mode delivered relay broadcast to a player: %q", msg)
		}
	}
	if got := relay.framesWithEvent("channels/send"); len(got) != 0 {
		t.Errorf("presence mode sent %d channels/send frames, want 0", len(got))
	}
}

// Check 4 (inbound): in full mode a relay broadcast reaches players,
// with peer-authored control characters stripped.
func TestGrapevineFullBridgesInbound(t *testing.T) {
	relay := newFakeRelay(t, "success")
	relay.setScript(0, []byte(`{"event":"channels/broadcast","payload":{"channel":"gossip","game":"OtherMUD","name":"Pe\u0007er","message":"hi\u001b[31m there"}}`))
	world := testWorld(t)
	rec := newSinkRecorder()
	world.MessageSink = rec.sink
	addPlayer(t, world, testPlayer("Local", 5))
	startClient(t, world, relay, "full", 0)

	waitForCond(t, 3*time.Second, "sanitized broadcast delivery", func() bool {
		for _, msg := range rec.all() {
			if strings.Contains(msg, "Peer@OtherMUD gossips, 'hi[31m there'") {
				return true
			}
		}
		return false
	})
	for _, msg := range rec.all() {
		if strings.Contains(msg, "\a") {
			t.Errorf("control character survived sanitization: %q", msg)
		}
	}
}

// Check 5: a failed auth installs no gossip bridge and never hot-loops —
// reconnects are separated by at least the backoff. Covers both the
// status:"failure" reply and a 4000 close.
func TestGrapevineAuthFailure(t *testing.T) {
	for _, authStatus := range []string{"failure", "close4000"} {
		t.Run(authStatus, func(t *testing.T) {
			relay := newFakeRelay(t, authStatus)
			world := testWorld(t)
			gossiper := testPlayer("tester", 5)
			addPlayer(t, world, gossiper)
			client := startClient(t, world, relay, "full", 300*time.Millisecond)

			time.Sleep(1100 * time.Millisecond)
			connTimes, _ := relay.snapshot()
			if len(connTimes) < 2 {
				t.Fatalf("expected at least 2 connection attempts, got %d", len(connTimes))
			}
			if gap := connTimes[1].Sub(connTimes[0]); gap < client.reconnectBackoff {
				t.Errorf("reconnect gap = %v, want at least the %v backoff (hot loop)", gap, client.reconnectBackoff)
			}

			world.DoChannel(gossiper, "should not relay", "gossip")
			time.Sleep(200 * time.Millisecond)
			if got := relay.framesWithEvent("channels/send"); len(got) != 0 {
				t.Errorf("auth failure left a gossip bridge installed: %d channels/send frames", len(got))
			}
		})
	}
}

// Check 6: the mode decision table.
func TestGrapevineModeSelection(t *testing.T) {
	cases := []struct {
		name        string
		id, secret  string
		modeEnv     string
		want        Mode
		wantInvalid string
	}{
		{"no credentials, mode unset", "", "", "", ModeOff, ""},
		{"no credentials, mode full", "", "", "full", ModeOff, ""},
		{"credentials, mode unset", "id", "secret", "", ModePresence, ""},
		{"credentials, mode off", "id", "secret", "off", ModeOff, ""},
		{"credentials, mode presence", "id", "secret", "presence", ModePresence, ""},
		{"credentials, mode full", "id", "secret", "full", ModeFull, ""},
		{"credentials, mode garbage", "id", "secret", "banana", ModePresence, "banana"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GRAPEVINE_CLIENT_ID", tc.id)
			t.Setenv("GRAPEVINE_CLIENT_SECRET", tc.secret)
			t.Setenv("GRAPEVINE_MODE", tc.modeEnv)
			got, invalid := resolveMode()
			if got != tc.want {
				t.Errorf("resolveMode() = %v, want %v", got, tc.want)
			}
			if invalid != tc.wantInvalid {
				t.Errorf("resolveMode() invalid = %q, want %q", invalid, tc.wantInvalid)
			}
			if got := ModeFromEnv(); got != tc.want {
				t.Errorf("ModeFromEnv() = %v, want %v", got, tc.want)
			}
		})
	}

	// Mode off with credentials: Start must not dial at all.
	t.Run("off does not dial", func(t *testing.T) {
		relay := newFakeRelay(t, "success")
		world := testWorld(t)
		startClient(t, world, relay, "off", 0)
		time.Sleep(300 * time.Millisecond)
		connTimes, _ := relay.snapshot()
		if len(connTimes) != 0 {
			t.Errorf("mode off dialed %d times, want 0", len(connTimes))
		}
	})
}
