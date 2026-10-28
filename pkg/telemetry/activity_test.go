package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestActivityRecordingLossRetentionAndRestart(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	h := New(now)
	RecordLogin(Activity{Player: "Avery", Class: "Warrior", MinLevel: 31})
	RecordCharacter(Activity{Player: "Mira", Class: "Mage", MinLevel: 31})
	RecordPlayerDeath(Activity{Player: "Avery", Cause: "mob", MobName: "a guard", MobVNum: 300, RoomName: "Arena", RoomVNum: 1001, MinLevel: 31})
	h.Sample(now.Add(time.Second), 1)
	s := h.Snapshot()
	if len(s.Activity) != 3 || s.Points[0].PlayerDeaths != 1 || s.Activity[2].MobVNum != 300 || s.Activity[0].Kind != "login" {
		t.Fatalf("missing events: %+v", s)
	}
	// A busy recorder cannot stall the game. Counts survive a detail drop.
	journal.mu.Lock()
	RecordPlayerDeath(Activity{Player: "Dropped"})
	journal.mu.Unlock()
	h.Sample(now.Add(2*time.Second), 0)
	s = h.Snapshot()
	if s.ActivityDropped != 1 || s.Points[0].PlayerDeaths != 2 || len(s.Activity) != 3 {
		t.Fatalf("drop hidden: %+v", s)
	}
	path := filepath.Join(t.TempDir(), "history.json")
	if err := h.Save(path); err != nil {
		t.Fatal(err)
	}
	restored := New(now)
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	RecordLogin(Activity{Player: "Returner"})
	restored.Sample(now.Add(3*time.Second), 0)
	s = restored.Snapshot()
	if len(s.Activity) != 4 || s.Activity[3].ID <= s.Activity[2].ID || !s.ActivitySince.Equal(now) {
		t.Fatalf("restart lost IDs or coverage: %+v", s)
	}
	restored.Sample(now.Add(8*24*time.Hour), 0)
	if len(restored.Snapshot().Activity) != 0 {
		t.Fatal("events outlived seven-day retention")
	}
	for i := range journal.entries {
		if journal.entries[i].Player != "" {
			t.Fatal("expired name kept in staging buffer")
		}
	}
	for i := range restored.state.Activity[:cap(restored.state.Activity)] {
		if restored.state.Activity[:cap(restored.state.Activity)][i].Player != "" {
			t.Fatal("expired name kept behind history slice")
		}
	}
}

func TestActivityBoundedJournalAndHistory(t *testing.T) {
	h := New(time.Now())
	for i := 0; i < journalCapacity+10; i++ {
		RecordLogin(Activity{Player: "Avery"})
	}
	h.Sample(time.Now(), 0)
	if s := h.Snapshot(); len(s.Activity) != journalCapacity || s.ActivityDropped != 10 {
		t.Fatalf("journal limit/loss: %d/%d", len(s.Activity), s.ActivityDropped)
	}
	for i := 0; i < 21; i++ {
		for j := 0; j < 1000; j++ {
			RecordCharacter(Activity{Player: strings.Repeat("x", 200) + "\n", RoomName: strings.Repeat("r", 300)})
		}
		h.Sample(time.Now(), 0)
	}
	s := h.Snapshot()
	if len(s.Activity) != ActivityCapacity || s.ActivityTrimmed == 0 {
		t.Fatalf("unbounded history: %d", len(s.Activity))
	}
	e := s.Activity[len(s.Activity)-1]
	if len(e.Player) != 32 || strings.Contains(e.Player, "\n") || len(e.RoomName) != 160 {
		t.Fatalf("unbounded fields: %+v", e)
	}
}

func TestLegacyHistoryCannotInventNamedOrPlayerDeathCoverage(t *testing.T) {
	now := time.Now().UTC()
	h := New(now)
	path := filepath.Join(t.TempDir(), "legacy.json")
	legacy := map[string]any{"started_at": now.Add(-time.Hour), "points": []map[string]any{{"time": now, "deaths": 11, "samples": 1}}}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := h.Load(path); err != nil {
		t.Fatal(err)
	}
	s := h.Snapshot()
	if !s.ActivitySince.Equal(now) || s.Points[0].PlayerDeaths != 0 || s.Points[0].Deaths != 11 || len(s.Activity) != 0 {
		t.Fatalf("invented legacy data: %+v", s)
	}
}

func TestShutdownSavesActivityAfterLastRegularSample(t *testing.T) {
	h := New(time.Now())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path := filepath.Join(t.TempDir(), "history.json")
	done := make(chan struct{})
	go func() { defer close(done); h.Run(ctx, path, func() int { return 1 }) }()
	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for len(h.Snapshot().Points) == 0 {
		select {
		case <-deadline.C:
			t.Fatal("recorder did not start")
		case <-tick.C:
		}
	}
	RecordLogin(Activity{Player: "Lastarrival", Class: "Thief", MinLevel: 31})
	cancel()
	select {
	case <-done:
	case <-deadline.C:
		t.Fatal("recorder did not stop")
	}
	restored := New(time.Now())
	if err := restored.Load(path); err != nil {
		t.Fatal(err)
	}
	events := restored.Snapshot().Activity
	if len(events) != 1 || events[0].Player != "Lastarrival" {
		t.Fatalf("shutdown lost pending activity: %+v", events)
	}
}
