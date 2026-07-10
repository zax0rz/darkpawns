package agentcli

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// trackedConn records whether Close was called on the underlying net.Conn.
type trackedConn struct {
	net.Conn
	closed *int32
}

func (c *trackedConn) Close() error {
	atomic.StoreInt32(c.closed, 1)
	return c.Conn.Close()
}

// TestWSConnCloseReleasesSocket guards against fd exhaustion: WSConn.Close must
// close the underlying TCP connection, not merely send a close frame. Without
// the explicit w.conn.Close(), the socket leaks on every reconnect/shutdown.
func TestWSConnCloseReleasesSocket(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	var closed int32
	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			c, err := net.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			return &trackedConn{Conn: c, closed: &closed}, nil
		},
	}
	c, _, err := dialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	w := &WSConn{conn: c}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if atomic.LoadInt32(&closed) != 1 {
		t.Fatal("WSConn.Close did not close the underlying TCP connection (fd leak)")
	}
}

func TestWSConnConcurrentWrites(t *testing.T) {
	var received int32
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer c.Close()

		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				close(done)
				return
			}
			atomic.AddInt32(&received, 1)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ws, err := Dial(ctx, "ws://"+strings.TrimPrefix(srv.URL, "http://")+"/ws")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	const n = 50
	var wg sync.WaitGroup
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := ws.WriteJSON(map[string]int{"i": i}); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("write error: %v", err)
	}

	_ = ws.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("server did not finish reading")
	}

	if got := atomic.LoadInt32(&received); got != n {
		t.Fatalf("expected %d messages, got %d", n, got)
	}
}
