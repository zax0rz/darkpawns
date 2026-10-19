package notify

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func testService(t *testing.T) *Service {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "notifications.json"))
	if err != nil {
		t.Fatal(err)
	}
	s.ready = func(string) bool { return true }
	return s
}

func TestConfigPersistenceAndIsolation(t *testing.T) {
	s := testService(t)
	c := s.View().Config
	if c.Pushover.Enabled || c.Discord.Enabled || c.Heartbeat || len(c.Pushover.Events) > 0 {
		t.Fatal("must be opt-in")
	}
	c.Pushover = Destination{Enabled: true, Events: []string{Created, Death}}
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(c); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale save: %v", err)
	}
	c.Pushover.Events[0] = "mutated"
	if s.View().Config.Pushover.Events[0] != Created {
		t.Fatal("input aliases live config")
	}
	loaded, err := New(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.View().Config.Pushover.Enabled || loaded.View().Config.Revision != 2 {
		t.Fatal("config lost on restart")
	}
	info, err := os.Stat(s.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatal("settings not private")
	}
	view := s.View()
	view.Config.Pushover.Events[0] = "mutated"
	if s.View().Config.Pushover.Events[0] != Created {
		t.Fatal("view aliases config")
	}
	bad := s.View().Config
	bad.Discord.Events = []string{"raw.log"}
	if !errors.Is(s.Save(bad), ErrInvalid) {
		t.Fatal("unknown event accepted")
	}
	bad = s.View().Config
	bad.Discord.Events = []string{Login, Login}
	if !errors.Is(s.Save(bad), ErrInvalid) {
		t.Fatal("duplicates accepted")
	}
	s.path = filepath.Join(s.path, "impossible.json")
	if s.Save(s.View().Config) == nil || s.View().Config.Revision != 2 {
		t.Fatal("failed save changed live state")
	}
}

func TestMalformedConfigAndMissingCredentials(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(p, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := New(p); err == nil {
		t.Fatal("corrupt config silently ignored")
	}
	s := testService(t)
	s.ready = func(string) bool { return false }
	c := s.View().Config
	c.Discord.Enabled = true
	if !errors.Is(s.Save(c), ErrNotReady) {
		t.Fatal("enabled without credentials")
	}
	if !errors.Is(s.Test(Pushover), ErrNotReady) {
		t.Fatal("test without credentials")
	}
	if err := s.Save(s.View().Config); err != nil {
		t.Fatal("cannot disable without credentials", err)
	}
}

func TestRoutingBoundedDeliveryAndRevision(t *testing.T) {
	s := testService(t)
	c := s.View().Config
	c.Pushover = Destination{true, []string{Created}}
	c.Discord = Destination{true, []string{Level}}
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	s.Publish(Event{Created, "Avery", 0})
	s.Publish(Event{Level, "Avery", 20})
	s.Publish(Event{Login, "Avery", 0})
	if len(s.workers[Pushover].jobs) != 1 || len(s.workers[Discord].jobs) != 1 {
		t.Fatal("event selection ignored")
	}
	got := make(chan string, 4)
	s.send = func(_ context.Context, id, message string) error { got <- id + ":" + message; return nil }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Run(ctx, nil)
	for range 2 {
		select {
		case <-got:
		case <-time.After(time.Second):
			t.Fatal("missing notification")
		}
	}
	// Queued events from before a settings change must not be delivered.
	s2 := testService(t)
	c = s2.View().Config
	c.Discord = Destination{true, []string{Login}}
	if err := s2.Save(c); err != nil {
		t.Fatal(err)
	}
	s2.Publish(Event{Login, "Avery", 0})
	c = s2.View().Config
	c.Discord.Enabled = false
	if err := s2.Save(c); err != nil {
		t.Fatal(err)
	}
	staleProcessed := make(chan struct{}, 1)
	s2.send = func(_ context.Context, _ string, message string) error {
		if !strings.HasPrefix(message, "Test notification.") {
			t.Error("stale job delivered")
		}
		staleProcessed <- struct{}{}
		return nil
	}
	if err := s2.Test(Discord); err != nil {
		t.Fatal(err)
	}
	s2.Run(ctx, nil)
	select {
	case <-staleProcessed:
	case <-time.After(time.Second):
		t.Fatal("worker did not process queued jobs")
	}
	// Exercise full queues independently of rate limiting with no consumer.
	s3 := testService(t)
	c = s3.View().Config
	c.Pushover = Destination{true, []string{Created}}
	if err := s3.Save(c); err != nil {
		t.Fatal(err)
	}
	s3.workers[Pushover].limiter = rate.NewLimiter(rate.Inf, 1)
	start := time.Now()
	for range 1000 {
		s3.Publish(Event{Created, "Avery", 0})
	}
	if time.Since(start) > time.Second {
		t.Fatal("publisher blocked")
	}
	if len(s3.workers[Pushover].jobs) != 16 || s3.View().Status[Pushover].Dropped != 984 {
		t.Fatal("queue capacity or overflow accounting")
	}
	s4 := testService(t)
	c = s4.View().Config
	c.Pushover = Destination{true, []string{Created}}
	if err := s4.Save(c); err != nil {
		t.Fatal(err)
	}
	for range 6 {
		s4.Publish(Event{Created, "Avery", 0})
	}
	if s4.View().Status[Pushover].Queued != 5 || s4.View().Status[Pushover].Dropped != 1 {
		t.Fatal("rate limit not applied")
	}
}

func TestProviderFailureRecovery(t *testing.T) {
	s := testService(t)
	result := errors.New("provider unavailable")
	done := make(chan struct{}, 2)
	s.send = func(context.Context, string, string) error { done <- struct{}{}; return result }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Run(ctx, nil)
	if err := s.Test(Discord); err != nil {
		t.Fatal(err)
	}
	<-done
	waitStatus(t, s, func(st Status) bool { return st.LastFailure != nil && st.LastError != "" })
	// Synchronize the replacement sender through the service lock after the worker
	// has recorded failure, then send a second test.
	s.mu.Lock()
	result = nil
	s.mu.Unlock()
	if err := s.Test(Discord); err != nil {
		t.Fatal(err)
	}
	<-done
	waitStatus(t, s, func(st Status) bool { return st.LastSuccess != nil && st.LastError == "" })
}

func waitStatus(t *testing.T, s *Service, predicate func(Status) bool) {
	t.Helper()
	deadline := time.After(time.Second)
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatal("status was not recorded")
		case <-tick.C:
			if predicate(s.View().Status[Discord]) {
				return
			}
		}
	}
}

func TestHeartbeatRequiresRecentProgress(t *testing.T) {
	s := testService(t)
	c := s.View().Config
	c.Heartbeat = true
	if err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for _, last := range []time.Time{{}, now.Add(-time.Minute), now.Add(time.Minute)} {
		s.heartbeat(now, last)
	}
	if len(s.workers["heartbeat"].jobs) != 0 {
		t.Fatal("dead or stalled loop reported healthy")
	}
	s.heartbeat(now, now.Add(-time.Second))
	if len(s.workers["heartbeat"].jobs) != 1 {
		t.Fatal("live loop not reported")
	}
}

func TestPublicTemplates(t *testing.T) {
	for _, id := range eventIDs {
		msg := format(Event{id, "@everyone_\nAvery<https://bad>", 20})
		if msg == "" || strings.ContainsAny(msg, "@_\n<>:/") {
			t.Fatalf("unsafe template %q", msg)
		}
	}
	if format(Event{"raw.log", "Avery", 0}) != "" {
		t.Fatal("arbitrary logs allowed")
	}
}
