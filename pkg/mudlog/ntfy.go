package mudlog

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// NtfyConfig is the slice-1 subscription: one ntfy topic from the ops
// environment (design §12.1: secrets live only in the environment and are
// referred to by name — nothing user-entered, so the provider preset is
// trusted for outbound addressing; §6 SSRF strictness applies to
// user-entered destinations, which arrive with slice 3's CRUD).
type NtfyConfig struct {
	URL string // full topic URL, e.g. https://ntfy.example.org/darkpawns-ops
	// Token is an optional bearer token ("tk_...").
	Token string
	// MinLevel and MinType are the virtual immortal (design §3.3): the
	// subscription sees an event exactly when an immortal with this level
	// and syslog type would — level >= event level AND type >= event type.
	MinLevel int
	MinType  int
	// Title prefixes every push.
	Title string
	// RatePerMin caps outbound posts; events beyond it are dropped+counted
	// (design §7: the game never waits on delivery).
	RatePerMin int
}

// NtfyFromEnv reads DP_MUDLOG_NTFY_URL (required; the subscription is off
// when unset — private by default, design §3.4), DP_MUDLOG_NTFY_TOKEN,
// DP_MUDLOG_NTFY_LEVEL (default 31, LVL_IMMORT), DP_MUDLOG_NTFY_TYPE
// (default 1, BRF) and DP_MUDLOG_NTFY_RATE (default 30/min).
func NtfyFromEnv() *NtfyConfig {
	url := strings.TrimSpace(os.Getenv("DP_MUDLOG_NTFY_URL"))
	if url == "" {
		return nil
	}
	cfg := &NtfyConfig{
		URL:        url,
		Token:      strings.TrimSpace(os.Getenv("DP_MUDLOG_NTFY_TOKEN")),
		MinLevel:   DefaultImmortalLevel,
		MinType:    DefaultBriefType,
		Title:      "Dark Pawns",
		RatePerMin: 30,
	}
	if v := strings.TrimSpace(os.Getenv("DP_MUDLOG_NTFY_LEVEL")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MinLevel = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("DP_MUDLOG_NTFY_TYPE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MinType = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("DP_MUDLOG_NTFY_RATE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RatePerMin = n
		}
	}
	return cfg
}

// Visible implements the virtual-immortal predicate shared with MudLog's
// own delivery check: an immortal of MinLevel with syslog MinType sees the
// line. File-only events (negative level) are never pushed.
func (c *NtfyConfig) Visible(e Event) bool {
	return e.Level >= 0 && c.MinLevel >= e.Level && c.MinType >= e.Typ
}

// NtfyStats are the subscription's delivery counters.
type NtfyStats struct {
	Matched   int64  `json:"matched"`
	Posted    int64  `json:"posted"`
	RateDrops int64  `json:"rate_drops"`
	Errors    int64  `json:"errors"`
	LastError string `json:"last_error,omitempty"`
}

// NtfySubscription is the delivery worker for one topic.
type NtfySubscription struct {
	cfg  *NtfyConfig
	feed *Feed
	post func(context.Context, *NtfyConfig, string) error // injected for tests

	mu       sync.Mutex
	stats    NtfyStats
	lastFail time.Time
	backoff  time.Duration
}

// SubscribeNtfy starts the worker. It consumes events live from the feed's
// current position (no replay of pre-boot history: a restart must not
// re-notify old lines).
func (f *Feed) SubscribeNtfy(ctx context.Context, cfg *NtfyConfig, client *http.Client) *NtfySubscription {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	sub := &NtfySubscription{
		cfg:  cfg,
		feed: f,
		post: func(ctx context.Context, cfg *NtfyConfig, text string) error {
			return ntfyPost(ctx, client, cfg, text)
		},
	}
	events, kicked, cancel := f.Subscribe(f.LastSeq())
	go func() {
		defer cancel()
		limiter := newTokenBucket(cfg.RatePerMin)
		for {
			select {
			case <-ctx.Done():
				return
			case <-kicked:
				// Too slow: re-sync from the ring's newest event and go on.
				events, kicked, cancel = f.Subscribe(f.LastSeq())
				continue
			case e, ok := <-events:
				if !ok {
					return
				}
				if !cfg.Visible(e) {
					continue
				}
				sub.mu.Lock()
				sub.stats.Matched++
				sub.mu.Unlock()
				if sub.paused(time.Now()) || !limiter.allow(time.Now()) {
					sub.mu.Lock()
					sub.stats.RateDrops++
					sub.mu.Unlock()
					continue
				}
				if err := sub.post(ctx, cfg, e.Text); err != nil {
					sub.recordError(err)
					continue
				}
				sub.mu.Lock()
				sub.stats.Posted++
				sub.backoff = 0
				sub.mu.Unlock()
			}
		}
	}()
	return sub
}

func (s *NtfySubscription) recordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats.Errors++
	s.stats.LastError = err.Error()
	// One failure slows the next post (1s, 2s, 4s … capped at 60s); the
	// feed is never backpressured, events during the pause are skipped.
	s.backoff *= 2
	if s.backoff == 0 {
		s.backoff = time.Second
	}
	if s.backoff > time.Minute {
		s.backoff = time.Minute
	}
	s.lastFail = time.Now()
}

// paused reports whether the worker is in post-failure backoff.
func (s *NtfySubscription) paused(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.backoff == 0 {
		return false
	}
	if now.Sub(s.lastFail) >= s.backoff {
		s.backoff = 0
		return false
	}
	return true
}

// Stats snapshots the delivery counters.
func (s *NtfySubscription) Stats() NtfyStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func ntfyPost(ctx context.Context, client *http.Client, cfg *NtfyConfig, text string) error {
	body := bytes.NewReader([]byte(text))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Title", cfg.Title)
	req.Header.Set("Content-Type", "text/plain")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("ntfy: %s", resp.Status)
	}
	return nil
}

// tokenBucket is a minimal allow/deny limiter (design §7). Burst is small
// (5) so a flood coalesces into drops rather than a queue.
type tokenBucket struct {
	rate   float64 // tokens per second
	tokens float64
	last   time.Time
}

func newTokenBucket(perMin int) *tokenBucket {
	perSec := float64(perMin) / 60
	return &tokenBucket{rate: perSec, tokens: 5, last: time.Now()}
}

func (b *tokenBucket) allow(now time.Time) bool {
	dt := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += dt * b.rate
	if b.tokens > 5 {
		b.tokens = 5
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
