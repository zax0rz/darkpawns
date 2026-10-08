package session

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// syncLogSink is a race-safe sink for game.SetLogWriter: the refusal producer
// runs on the HTTP handler's goroutine while the test polls.
type syncLogSink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *syncLogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncLogSink) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// waitForLogLine polls the sink until it holds want.
func waitForLogLine(t *testing.T, sink *syncLogSink, want string) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		got := sink.String()
		if strings.Contains(got, want) || time.Now().After(deadline) {
			return got
		}
		time.Sleep(time.Millisecond)
	}
}

// comm.c:1573-1574 on the WebSocket transport: the transport keeps its own
// policy-violation close frame — it is a Go-only transport with no C bytes —
// and C's producer still fires with the host d->host would hold.
func TestWebSocketBannedRefusalLogs(t *testing.T) {
	database := entryDatabase(t)
	manager := entryTransportManager(t, database)
	// C's wildhost string for 127.0.0.1 (src/comm.c:1541-1544).
	if err := manager.GetBanManager().AddBan("127.000.000.*", game.BanAll, "God"); err != nil {
		t.Fatal(err)
	}
	sink := &syncLogSink{}
	game.SetLogWriter(sink)
	t.Cleanup(func() { game.SetLogWriter(os.Stderr) })

	server := httptest.NewServer(http.HandlerFunc(manager.HandleWebSocket))
	t.Cleanup(server.Close)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"Origin": []string{"https://darkpawns.org"}})
	if err != nil {
		t.Fatalf("upgrade refused before the ban check: %v", err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, _, readErr := conn.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(readErr, &closeErr) || closeErr.Code != websocket.ClosePolicyViolation {
		t.Fatalf("WebSocket close = %v, want a policy-violation close frame", readErr)
	}

	want := "Connection attempt denied from [127.000.000.001]"
	if got := waitForLogLine(t, sink, want); !strings.Contains(got, want) {
		t.Fatalf("file log = %q, want it to contain %q", got, want)
	}
}
