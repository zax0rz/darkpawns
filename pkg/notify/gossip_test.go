package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func gossipService(t *testing.T) *Service {
	t.Helper()
	emptyEnv(t)
	t.Setenv("DP_NOTIFY_GOSSIP_WEBHOOK", "https://discord.com/api/webhooks/123/gossip-private-token")
	s, err := New(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func enableGossip(t *testing.T, s *Service) {
	t.Helper()
	c := s.View().Config
	c.Gossip = true
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
}

func gossipWorker(t *testing.T, s *Service) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); s.runGossipWorker(ctx, s.workers[Gossip]) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("worker did not stop")
		}
	})
}

func awaitGossipDrops(t *testing.T, s *Service, want uint64) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		if s.View().Status[Gossip].Dropped == want {
			return
		}
		select {
		case <-deadline.C:
			t.Fatalf("drops = %d, want %d", s.View().Status[Gossip].Dropped, want)
		case <-tick.C:
		}
	}
}

func gossipTransport(t *testing.T, f roundTripFunc) {
	t.Helper()
	old := providerClient.Transport
	providerClient.Transport = f
	t.Cleanup(func() { providerClient.Transport = old })
}

func TestGossipOptInCredentialsAndBoundedQueue(t *testing.T) {
	s := gossipService(t)
	s.PublishGossip("Actor gossips, 'hello'\r\n")
	if s.GossipEnabled() || len(s.workers[Gossip].jobs) != 0 {
		t.Fatal("credentials enabled gossip by default")
	}
	enableGossip(t, s)
	s.workers[Gossip].limiter = rate.NewLimiter(rate.Inf, 1)
	start := time.Now()
	for range 1000 {
		s.PublishGossip("Actor gossips, 'hello'\r\n")
	}
	if time.Since(start) > time.Second || len(s.workers[Gossip].jobs) != 16 || s.View().Status[Gossip].Dropped != 984 {
		t.Fatal("queue is blocking or unbounded")
	}
	if len(s.workers[Discord].jobs) != 0 {
		t.Fatal("gossip reached event-alert destination")
	}
	if err := s.Test(Gossip); !errors.Is(err, ErrInvalid) {
		t.Fatal("non-channel test message allowed in gossip feed")
	}
	if err := s.SaveCredentials(Gossip, s.View().Config.Revision, nil); err != nil {
		t.Fatal(err)
	}
	if s.GossipEnabled() || s.View().Config.Gossip || s.View().Status[Gossip].Ready {
		t.Fatal("credential removal did not disable gossip")
	}
	restarted, err := New(s.path)
	if err != nil || restarted.View().Status[Gossip].Ready {
		t.Fatal("removed credential fell back to environment", err)
	}
}

func TestGossipWireRenderingMentionsAndPrivateCredentials(t *testing.T) {
	s := gossipService(t)
	secret := Credentials{URL: "https://discord.com/api/webhooks/456/encrypted-gossip-token"}
	if err := s.SaveCredentials(Gossip, s.View().Config.Revision, &secret); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	view, err := json.Marshal(s.View())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "encrypted-gossip-token") || strings.Contains(string(view), "encrypted-gossip-token") {
		t.Fatal("secret exposed")
	}
	enableGossip(t, s)
	requests := make(chan string, 1)
	gossipTransport(t, func(r *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Content         string `json:"content"`
			AllowedMentions struct {
				Parse []string `json:"parse"`
			} `json:"allowed_mentions"`
		}
		if err = json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		if r.URL.Path != "/api/webhooks/456/encrypted-gossip-token" || r.URL.Query().Get("wait") != "true" || payload.AllowedMentions.Parse == nil || len(payload.AllowedMentions.Parse) != 0 {
			t.Error("unsafe route or allowed_mentions", string(body))
		}
		requests <- payload.Content
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	// Cleanup joins worker before restoring its transport.
	gossipWorker(t, s)
	s.PublishGossip("&rSomeone&n gossips, '\x1b[31mred\x1b[0m @everyone @here <@123> <@&456>'\r\n")
	select {
	case got := <-requests:
		if got != "Someone gossips, 'red @everyone @here <@123> <@&456>'" {
			t.Fatal("rewrote the visible line", got)
		}
	case <-time.After(time.Second):
		t.Fatal("missing request")
	}
}

func TestGossipCooldownAndNonBlockingInFlightRequest(t *testing.T) {
	s := gossipService(t)
	enableGossip(t, s)
	s.workers[Gossip].limiter = rate.NewLimiter(rate.Inf, 1)
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	gossipTransport(t, func(*http.Request) (*http.Response, error) {
		select {
		case entered <- struct{}{}:
		default:
			t.Error("sent during cooldown")
		}
		<-release
		return &http.Response{StatusCode: 429, Header: http.Header{"Retry-After": []string{"3600"}}, Body: io.NopCloser(strings.NewReader(`{"retry_after":3600}`))}, nil
	})
	gossipWorker(t, s)
	s.PublishGossip("Actor gossips, 'first'")
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("worker never started")
	}
	// Ensure a failure cannot leave the HTTP fake hung in cleanup.
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	start := time.Now()
	for range 1000 {
		s.PublishGossip("Actor gossips, 'burst'")
	}
	if time.Since(start) > time.Second {
		t.Fatal("HTTP stalled publisher")
	}
	close(release)
	awaitGossipDrops(t, s, 1001) // rejected request + 16 queued + 984 overflow
	if s.View().Status[Gossip].LastError != "provider returned HTTP 429" {
		t.Fatal("missing redacted rate-limit status")
	}
}

func TestGossipDiscordCooldownHeaders(t *testing.T) {
	for _, tc := range []struct {
		name    string
		code    int
		headers http.Header
		body    string
		want    time.Duration
	}{
		{"success exhausted", 200, http.Header{"X-Ratelimit-Remaining": []string{"0"}, "X-Ratelimit-Reset-After": []string{"1.25"}}, "{}", 1250 * time.Millisecond},
		{"429 body", 429, http.Header{}, `{"retry_after":2.5}`, 2500 * time.Millisecond},
		{"longest delay", 429, http.Header{"Retry-After": []string{"3"}, "X-Ratelimit-Reset-After": []string{"4"}}, `{"retry_after":2}`, 4 * time.Second},
		{"malformed", 429, http.Header{"Retry-After": []string{"NaN"}}, "private response", time.Minute},
		{"not exhausted", 200, http.Header{"X-Ratelimit-Remaining": []string{"1"}}, "{}", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.code, Header: tc.headers, Body: io.NopCloser(strings.NewReader(tc.body))}
			if got := discordCooldown(resp, time.Now()); got != tc.want {
				t.Fatalf("delay=%s want=%s", got, tc.want)
			}
		})
	}
}

func TestGossipDropsUnrepresentableLinesAndStaleJobs(t *testing.T) {
	s := gossipService(t)
	enableGossip(t, s)
	s.PublishGossip(strings.Repeat("🙂", 1001))
	s.PublishGossip(string([]byte{0xff}))
	if s.View().Status[Gossip].Dropped != 2 || len(s.workers[Gossip].jobs) != 0 {
		t.Fatal("overlong or invalid UTF8 line queued")
	}
	s.PublishGossip("Someone gossips, 'stale'")
	c := s.View().Config
	c.Gossip = false
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	// A fresh job behind the stale job is a FIFO processing barrier: receiving
	// it proves the old epoch was consumed without relying on a sleep.
	enableGossip(t, s)
	requests := make(chan string, 2)
	gossipTransport(t, func(r *http.Request) (*http.Response, error) {
		var payload struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return nil, err
		}
		requests <- payload.Content
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("{}"))}, nil
	})
	s.PublishGossip("Someone gossips, 'fresh'")
	gossipWorker(t, s)
	select {
	case got := <-requests:
		if got != "Someone gossips, 'fresh'" {
			t.Fatalf("old epoch leaked: %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("fresh processing barrier not reached")
	}
}

func TestGossipOverflowLogsOnlyAggregateCounts(t *testing.T) {
	s := gossipService(t)
	enableGossip(t, s)
	for range 10 {
		s.PublishGossip("Actor gossips, 'private-test-marker'")
	}
	var output bytes.Buffer
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	defer slog.SetDefault(old)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.runGossipWorker(ctx, s.workers[Gossip])
	got := output.String()
	if !strings.Contains(got, "Discord gossip messages dropped") || !strings.Contains(got, "count=5") || strings.Contains(got, "private-test-marker") || strings.Contains(got, "gossip-private-token") {
		t.Fatalf("unsafe or missing drop log: %q", got)
	}
}

func TestGossipNoticeDefaultsAndPersistence(t *testing.T) {
	emptyEnv(t)
	for _, legacy := range []bool{false, true} {
		name := "fresh"
		if legacy {
			name = "existing settings without notice field"
		}
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "notifications.json")
			if legacy {
				if err := os.WriteFile(path, []byte(`{"revision":1,"gossip":false}`), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			s, err := New(path)
			if err != nil {
				t.Fatal(err)
			}
			config := s.View().Config
			if !config.GossipNotice || config.Gossip || s.GossipNoticeEnabled() {
				t.Fatalf("notice must default on without enabling mirroring: %+v", config)
			}
			config.GossipNotice = false
			if err := s.Save(config); err != nil {
				t.Fatal(err)
			}
			reloaded, err := New(path)
			if err != nil {
				t.Fatal(err)
			}
			if reloaded.View().Config.GossipNotice {
				t.Fatal("explicit notice opt-out did not survive reload")
			}
		})
	}
}
