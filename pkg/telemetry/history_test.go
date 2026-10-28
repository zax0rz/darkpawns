package telemetry

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestHistoryEventsGapsRetentionAndRestart(t *testing.T) {
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	h := New(now)
	Login()
	Character()
	Death()
	Warning()
	Error()
	h.Sample(now, 3)
	h.Sample(now.Add(10*time.Second), 5)
	s := h.Snapshot()
	if len(s.Points) != 1 || s.Points[0].Online != 5 || s.Points[0].Peak != 5 || s.Points[0].Samples != 2 || s.Points[0].Logins != 1 || s.Points[0].Characters != 1 || s.Points[0].Deaths != 1 || s.Points[0].Warnings != 1 || s.Points[0].Errors != 1 {
		t.Fatalf("wrong recorded observations: %+v", s)
	}
	path := filepath.Join(t.TempDir(), "history.json")
	h.ObserveRelease(Release{Identity: "one", Version: "v1"}, now)
	if err := h.Save(path); err != nil {
		t.Fatal(err)
	}
	restored := New(now.Add(time.Hour))
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	restored.ObserveRelease(Release{Identity: "one", Version: "v1"}, now.Add(time.Hour))
	if restored.Snapshot().Release.UpdatedAt != nil {
		t.Fatal("restart invented an update")
	}
	restored.ObserveRelease(Release{Identity: "two", Version: "v2"}, now.Add(2*time.Hour))
	if restored.Snapshot().Release.UpdatedAt == nil {
		t.Fatal("missed actual executable change")
	}
	restored.Sample(now.Add(2*time.Hour), 0)
	s = restored.Snapshot()
	if len(s.Points) != 2 || s.Points[1].Logins != 0 {
		t.Fatalf("gap fabricated or counters replayed: %+v", s.Points)
	}
	restored.Sample(now.Add(8*24*time.Hour), 1)
	if len(restored.Snapshot().Points) != 1 {
		t.Fatal("retention failed")
	}
}

func TestCorruptHistoryDoesNotLookLikeLoadedData(t *testing.T) {
	h := New(time.Now())
	if err := h.Load(t.TempDir()); err == nil {
		t.Fatal("expected read failure")
	}
}

func TestHistoryShutdownPersistsAndFailuresAreVisible(t *testing.T) {
	now := time.Now()
	h := New(now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(t.TempDir(), "history.json")
	h.Run(ctx, path, func() int { return 2 })
	restored := New(now)
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	if len(restored.Snapshot().Points) != 1 {
		t.Fatal("shutdown failed to persist sample")
	}
	blocked := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	h.Run(ctx, filepath.Join(blocked, "history.json"), func() int { return 1 })
	if h.Snapshot().PersistenceError == "" {
		t.Fatal("write failure hidden")
	}
}
