package notify

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var gossipANSI = regexp.MustCompile("\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\x07\x1b]*(?:\x07|\x1b\\\\)|[@-_])")

// GossipEnabled reports whether the mirror is active. Credentials alone
// never enable the mirror, and the event-alert Discord destination is separate.
func (s *Service) GossipEnabled() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Gossip && s.ready(Gossip)
}

// GossipNoticeEnabled reports whether the active mirror needs an automatic MOTD notice.
func (s *Service) GossipNoticeEnabled() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Gossip && s.cfg.GossipNotice && s.ready(Gossip)
}

// PublishGossip accepts only the game's already rendered public observer line.
// It performs no disk/network/log I/O and never waits for queue space. Overlong
// or invalid UTF-8 lines are dropped whole rather than rewritten or truncated.
func (s *Service) PublishGossip(line string) {
	if s == nil {
		return
	}
	line = strings.TrimSuffix(plainGossip(line), "\r\n")
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.cfg.Gossip || !s.ready(Gossip) {
		return
	}
	units := 0
	for _, r := range line {
		units++
		if r > 0xffff {
			units++
		}
	}
	if line == "" || !utf8.ValidString(line) || units > 2000 {
		s.drop(Gossip)
		return
	}
	_ = s.enqueue(Gossip, job{message: line, revision: s.cfg.Revision})
}

// One worker owns this webhook's response-driven cooldown. Requests are never
// retried: a rejected line and lines arriving during cooldown are dropped.
// Overflow logging is aggregated here, never on the game loop.
func (s *Service) runGossipWorker(ctx context.Context, w *worker) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var nextAttempt time.Time
	var destination string
	var logged uint64
	logDrops := func() {
		s.mu.Lock()
		dropped := s.status[Gossip].Dropped
		s.mu.Unlock()
		if dropped < logged {
			logged = 0
		}
		if dropped > logged {
			slog.Warn("Discord gossip messages dropped", "count", dropped-logged)
			logged = dropped
		}
	}
	defer logDrops()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logDrops()
		case j := <-w.jobs:
			s.mu.Lock()
			// A test is delivered while the mirror is off, matching the event
			// destinations: the point of a test is an unverified channel.
			allowed := j.revision == s.cfg.Revision && (s.cfg.Gossip || j.test)
			credentials := s.vault.snapshot.Load().values[Gossip]
			s.mu.Unlock()
			if !allowed || ctx.Err() != nil {
				continue
			}
			if destination != credentials.URL {
				destination = credentials.URL
				nextAttempt = time.Time{}
			}
			if time.Now().Before(nextAttempt) {
				s.mu.Lock()
				s.drop(Gossip)
				s.mu.Unlock()
				continue
			}
			deliveryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			cooldown, err := deliverProvider(deliveryCtx, Gossip, j.message, credentials)
			cancel()
			if cooldown > 0 {
				nextAttempt = time.Now().Add(cooldown)
			}
			s.mu.Lock()
			if j.revision != s.cfg.Revision {
				s.mu.Unlock()
				continue
			}
			st := s.status[Gossip]
			now := time.Now().UTC()
			if err != nil {
				st.LastFailure = &now
				st.LastError = err.Error()
				st.Dropped++
			} else {
				st.LastSuccess = &now
				st.LastError = ""
			}
			s.status[Gossip] = st
			s.mu.Unlock()
		}
	}
}

// Discord documents decimal-second reset/retry headers and a JSON retry_after
// fallback. Honor the longest supplied delay, including successful responses
// that exhaust the bucket. Never expose the response body in logs or errors.
func discordCooldown(resp *http.Response, now time.Time) time.Duration {
	exhausted := resp.StatusCode == http.StatusTooManyRequests || resp.Header.Get("X-RateLimit-Remaining") == "0"
	if !exhausted {
		return 0
	}
	delay := secondsDelay(resp.Header.Get("X-RateLimit-Reset-After"))
	retry := resp.Header.Get("Retry-After")
	delay = max(delay, secondsDelay(retry))
	if when, err := http.ParseTime(retry); err == nil {
		delay = max(delay, when.Sub(now))
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		var body struct {
			RetryAfter float64 `json:"retry_after"`
		}
		if json.NewDecoder(io.LimitReader(resp.Body, 4096)).Decode(&body) == nil {
			delay = max(delay, secondsDelay(strconv.FormatFloat(body.RetryAfter, 'f', -1, 64)))
		}
	}
	if delay <= 0 {
		return time.Minute
	}
	return delay
}

func secondsDelay(raw string) time.Duration {
	seconds, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(seconds) || seconds <= 0 {
		return 0
	}
	if seconds >= float64(math.MaxInt64)/float64(time.Second) {
		return time.Duration(math.MaxInt64)
	}
	return time.Duration(math.Ceil(seconds * float64(time.Second)))
}

// Strip the same built-in markers expanded by the terminal (src/comm.c:1288-1336),
// as well as raw ANSI. Preserve unknown markers and escaped literal ampersands.
func plainGossip(line string) string {
	line = gossipANSI.ReplaceAllString(line, "")
	var out strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '&' && i+1 < len(line) {
			if line[i+1] == '&' {
				out.WriteByte('&')
				i++
				continue
			}
			if strings.ContainsRune("ndbgcrmywDBGCRMYW", rune(line[i+1])) {
				i++
				continue
			}
		}
		out.WriteByte(line[i])
	}
	return out.String()
}
