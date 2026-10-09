package mudlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// The virtual-immortal predicate is MudLog's own delivery check: level and
// syslog type both gate, and file-only lines never push (design §3.3).
func TestNtfyVirtualImmortalFilter(t *testing.T) {
	cfg := &NtfyConfig{MinLevel: 31, MinType: 1}
	cases := []struct {
		e    Event
		want bool
		why  string
	}{
		{Event{Level: 31, Typ: 1}, true, "immortal brief sees the standard line"},
		{Event{Level: 30, Typ: 1}, true, "higher immortal also sees lower-level lines"},
		{Event{Level: 31, Typ: 0}, true, "OFF (0) lines are below every syslog setting"},
		{Event{Level: 32, Typ: 1}, false, "sub at 31 does not see level-32 lines"},
		{Event{Level: 31, Typ: 3}, false, "CMP (3) needs syslog >= 3; a BRF (1) sub misses it"},
		{Event{Level: -1, Typ: 1}, false, "file-only line"},
	}
	for _, c := range cases {
		if got := cfg.Visible(c.e); got != c.want {
			t.Errorf("Visible(%+v) = %v, want %v (%s)", c.e, got, c.want, c.why)
		}
	}
}

// A matching event is posted once with its text; a non-matching event never
// reaches the wire.
func TestNtfyPostsMatchingEventsOnly(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		mu.Lock()
		bodies = append(bodies, string(buf[:n]))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFeed(64, 64)
	f.Start()
	defer f.Stop()
	cfg := &NtfyConfig{URL: srv.URL, MinLevel: 31, MinType: 1, RatePerMin: 1000}
	sub := f.SubscribeNtfy(context.Background(), cfg, srv.Client())
	f.tap("", "advance: Zach is now level 2.", 1, 31, false)
	f.tap("", "file-only whisper", 1, -1, true)
	f.tap("", "cmp line invisible to brief", 3, 31, false) // Typ 3 > MinType 1
	waitFor(t, "one post", func() bool { return sub.Stats().Posted == 1 })
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 1 || !strings.Contains(bodies[0], "advance: Zach is now level 2.") {
		t.Fatalf("bodies = %q", bodies)
	}
}

// Beyond the rate, events drop and count instead of queueing (design §7).
func TestNtfyRateLimitDrops(t *testing.T) {
	f := NewFeed(512, 512)
	f.Start()
	defer f.Stop()
	cfg := &NtfyConfig{URL: "http://127.0.0.1:1/unreachable", MinLevel: 31, MinType: 1, RatePerMin: 1}
	// Errors back off; use a successful sink so only the limiter acts.
	var mu sync.Mutex
	posted := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		posted++
		mu.Unlock()
	}))
	defer srv.Close()
	cfg.URL = srv.URL
	sub := f.SubscribeNtfy(context.Background(), cfg, srv.Client())
	for i := 0; i < 50; i++ {
		f.tap("", "flood", 1, 31, false)
	}
	waitFor(t, "limiter settles", func() bool {
		st := sub.Stats()
		return st.Matched == 50 && st.Posted+st.RateDrops == 50
	})
	st := sub.Stats()
	if st.Posted > 6 { // burst 5 + a little refill
		t.Fatalf("posted %d through a 1/min limiter", st.Posted)
	}
	if st.RateDrops == 0 {
		t.Fatal("flood was not rate-dropped")
	}
}

// A failing endpoint counts errors and does not kill the worker.
func TestNtfyErrorCountedWorkerSurvives(t *testing.T) {
	fails := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fails++
		if fails == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	f := NewFeed(64, 64)
	f.Start()
	defer f.Stop()
	cfg := &NtfyConfig{URL: srv.URL, MinLevel: 31, MinType: 1, RatePerMin: 1000}
	sub := f.SubscribeNtfy(context.Background(), cfg, srv.Client())
	f.tap("", "first", 1, 31, false)
	waitFor(t, "first error recorded", func() bool { return sub.Stats().Errors >= 1 })
	// Backoff passes (1s); a later event still delivers.
	time.Sleep(1100 * time.Millisecond)
	f.tap("", "after recovery", 1, 31, false)
	waitFor(t, "recovered post", func() bool { return sub.Stats().Posted >= 1 })
}

// NtfyFromEnv is off without the URL, and reads the documented knobs.
func TestNtfyFromEnv(t *testing.T) {
	t.Setenv("DP_MUDLOG_NTFY_URL", "")
	if cfg := NtfyFromEnv(); cfg != nil {
		t.Fatal("subscription must be off without DP_MUDLOG_NTFY_URL")
	}
	t.Setenv("DP_MUDLOG_NTFY_URL", "https://ntfy.example.org/ops")
	t.Setenv("DP_MUDLOG_NTFY_TOKEN", "tk_x")
	t.Setenv("DP_MUDLOG_NTFY_LEVEL", "30")
	t.Setenv("DP_MUDLOG_NTFY_TYPE", "2")
	cfg := NtfyFromEnv()
	if cfg.URL != "https://ntfy.example.org/ops" || cfg.Token != "tk_x" || cfg.MinLevel != 30 || cfg.MinType != 2 {
		t.Fatalf("cfg = %+v", cfg)
	}
}
