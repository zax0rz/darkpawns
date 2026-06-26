package agentcli

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

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
