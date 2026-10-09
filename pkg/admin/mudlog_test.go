package admin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/apidoc"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/mudlog"
)

func newMudlogFeed(t *testing.T) *mudlog.Feed {
	t.Helper()
	f := mudlog.NewFeed(64, 128)
	f.Start()
	t.Cleanup(f.Stop)
	return f
}

func mudlogAPI(t *testing.T, f *mudlog.Feed) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	api := apidoc.New().NewInternalAPI(mux)
	registerMudlog(api, f)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func waitForSeq(t *testing.T, f *mudlog.Feed, seq uint64) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for f.LastSeq() != seq {
		if time.Now().After(deadline) {
			t.Fatalf("feed stalled at seq %d, want %d", f.LastSeq(), seq)
		}
		time.Sleep(time.Millisecond)
	}
}

// The list endpoint replays ring history and applies the virtual-immortal
// filters the same way MudLog's own delivery check does.
func TestMudlogListEndpoint(t *testing.T) {
	f := newMudlogFeed(t)
	f.Tap("", "boot line", 1, 31, false)
	f.Tap("", "cmp line", 3, 31, false)
	f.Tap("", "level 35 line", 1, 35, false)
	waitForSeq(t, f, 3)
	srv := mudlogAPI(t, f)

	get := func(path string) (struct {
		Events  []mudlog.Event `json:"events"`
		LastSeq uint64         `json:"last_seq"`
	}, *http.Response,
	) {
		t.Helper()
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %s", path, resp.Status)
		}
		var out struct {
			Events  []mudlog.Event `json:"events"`
			LastSeq uint64         `json:"last_seq"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out, resp
	}

	all, _ := get("/admin/mudlog")
	if len(all.Events) != 3 || all.LastSeq != 3 {
		t.Fatalf("all = %d events, last %d", len(all.Events), all.LastSeq)
	}

	// A BRF (type 1) virtual immortal misses the CMP line.
	brief, _ := get("/admin/mudlog?type=1")
	if len(brief.Events) != 2 || brief.Events[1].Text != "level 35 line" {
		t.Fatalf("brief filter = %+v", brief.Events)
	}

	// A level-31 virtual immortal misses the level-35 line.
	lvl, _ := get("/admin/mudlog?level=31")
	if len(lvl.Events) != 2 || lvl.Events[1].Text != "cmp line" {
		t.Fatalf("level filter = %+v", lvl.Events)
	}

	// since= replays only the tail.
	tail, _ := get("/admin/mudlog?since=2")
	if len(tail.Events) != 1 || tail.Events[0].Seq != 3 {
		t.Fatalf("since filter = %+v", tail.Events)
	}
}

// The SSE stream replays from the cursor, then tails live events; the event
// id is the ring seq, which is also the Last-Event-ID resume point.
func TestMudlogStreamReplaysAndTails(t *testing.T) {
	f := newMudlogFeed(t)
	f.Tap("", "history", 1, 31, false)
	waitForSeq(t, f, 1)

	srv := httptest.NewServer(mudlogStream(f))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/admin/mudlog/stream?since=0", nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("content-type = %q", ct)
	}

	lines := make(chan string, 64)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

	// Replay arrives first: the history event with its seq as id.
	expectData := func(t *testing.T, want string) {
		t.Helper()
		for {
			select {
			case l, ok := <-lines:
				if !ok {
					t.Fatal("stream closed early")
				}
				if strings.HasPrefix(l, "data: ") && strings.Contains(l, want) {
					return
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("never saw %q", want)
			}
		}
	}
	expectData(t, `"text":"history"`)

	f.Tap("", "live", 1, 31, false)
	waitForSeq(t, f, 2)
	expectData(t, `"text":"live"`)
	cancel()
}

// The production path, end to end through the process feed: main's one
// helper starts the feed game.MudLog taps and hands the same feed to the
// admin router. If the tap side and the reader side ever drift onto two
// different feeds again (PR #1893 review, bug 1: a NewFeed in main left
// /admin/mudlog, the SSE tail and ntfy permanently empty), the default
// feed fills to its cap and drops, and this list comes back empty.
func TestMudlogEndToEndUsesTheProcessFeed(t *testing.T) {
	feed := mudlog.StartDefault(context.Background(), nil)
	defer feed.Stop()
	srv := mudlogAPI(t, feed)

	game.MudLog("e2e default-feed line", mudlog.DefaultBriefType, mudlog.DefaultImmortalLevel, false)

	deadline := time.Now().Add(3 * time.Second)
	for {
		resp, err := srv.Client().Get(srv.URL + "/admin/mudlog")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if bytes.Contains(body, []byte("e2e default-feed line")) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("game.MudLog never reached the admin endpoint; body: %s", body)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
