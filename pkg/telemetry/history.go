// Package telemetry records operator-only observations. Game call sites never
// wait on recording or disk; sampling and persistence run outside the game loop.
package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

var logins, characters, deaths, playerDeaths, warnings, errorEvents atomic.Uint64

func Login()     { logins.Add(1) }
func Character() { characters.Add(1) }
func Death()     { deaths.Add(1) }
func Warning()   { warnings.Add(1) }
func Error()     { errorEvents.Add(1) }

type Counts struct {
	Logins       uint64 `json:"logins"`
	Characters   uint64 `json:"characters"`
	Deaths       uint64 `json:"deaths"`
	Warnings     uint64 `json:"warnings"`
	Errors       uint64 `json:"errors"`
	PlayerDeaths uint64 `json:"player_deaths"`
}

func counters() Counts {
	return Counts{logins.Load(), characters.Load(), deaths.Load(), warnings.Load(), errorEvents.Load(), playerDeaths.Load()}
}

func (c *Counts) add(d Counts) {
	c.Logins += d.Logins
	c.Characters += d.Characters
	c.Deaths += d.Deaths
	c.Warnings += d.Warnings
	c.Errors += d.Errors
	c.PlayerDeaths += d.PlayerDeaths
}

func delta(c, p Counts) Counts {
	return Counts{c.Logins - p.Logins, c.Characters - p.Characters, c.Deaths - p.Deaths, c.Warnings - p.Warnings, c.Errors - p.Errors, c.PlayerDeaths - p.PlayerDeaths}
}

type Point struct {
	Time    time.Time `json:"time"`
	Online  int       `json:"online"`
	Peak    int       `json:"peak"`
	Samples int       `json:"samples"`
	Counts
}
type Release struct {
	Version    string     `json:"version"`
	Commit     string     `json:"commit"`
	Identity   string     `json:"identity"`
	ObservedAt time.Time  `json:"observed_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}
type Snapshot struct {
	StartedAt        time.Time  `json:"started_at"`
	Release          Release    `json:"release"`
	Points           []Point    `json:"points"`
	PersistenceError string     `json:"persistence_error,omitempty"`
	ActivitySince    time.Time  `json:"activity_since"`
	Activity         []Activity `json:"activity"`
	NextActivityID   uint64     `json:"next_activity_id"`
	ActivityDropped  uint64     `json:"activity_dropped"`
	ActivityTrimmed  uint64     `json:"activity_trimmed"`
}
type History struct {
	mu              sync.RWMutex
	state           Snapshot
	previous        Counts
	activityCursor  uint64
	previousDropped uint64
}

func New(now time.Time) *History {
	journal.mu.Lock()
	defer journal.mu.Unlock()
	return &History{state: Snapshot{StartedAt: now.UTC(), ActivitySince: now.UTC(), Points: []Point{}, Activity: []Activity{}}, previous: counters(), activityCursor: journal.sequence, previousDropped: journal.dropped.Load()}
}

// Sample records at most one minute bucket per minute, preserving sample gaps.
// Event deltas belong to the observation interval ending now (up to 10 seconds).
func (h *History) Sample(now time.Time, online int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	t := now.UTC().Truncate(time.Minute)
	current := counters()
	d := delta(current, h.previous)
	h.previous = current
	points := h.state.Points
	cutoff := now.Add(-7 * 24 * time.Hour)
	first := 0
	for first < len(points) && points[first].Time.Before(cutoff) {
		first++
	}
	points = points[first:]
	if len(points) == 0 || !points[len(points)-1].Time.Equal(t) {
		points = append(points, Point{Time: t})
	}
	p := &points[len(points)-1]
	p.Online = online
	if online > p.Peak {
		p.Peak = online
	}
	p.Samples++
	p.add(d)
	h.state.Points = points
	h.sampleActivity(now)
}

func (h *History) Snapshot() Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s := h.state
	s.Points = append([]Point{}, s.Points...)
	s.Activity = append([]Activity{}, s.Activity...)
	return s
}

func (h *History) Load(path string) error {
	b, err := os.ReadFile(filepath.Clean(path))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var s Snapshot
	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.StartedAt.IsZero() || len(s.Points) > 10082 || len(s.Activity) > ActivityCapacity {
		return fmt.Errorf("invalid telemetry history")
	}
	h.mu.Lock()
	// Old files contain combined death counts only. Never relabel those as
	// player deaths, or imply names were recorded before this recorder existed.
	if s.ActivitySince.IsZero() {
		s.ActivitySince = h.state.ActivitySince
	}
	for i := range s.Activity {
		if s.Activity[i].ID > s.NextActivityID {
			s.NextActivityID = s.Activity[i].ID
		}
	}
	h.state = s
	h.mu.Unlock()
	return nil
}

func (h *History) Save(path string) (saveErr error) {
	b, err := json.Marshal(h.Snapshot())
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".history-*")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(f.Name()); err != nil && !os.IsNotExist(err) {
			saveErr = errors.Join(saveErr, err)
		}
	}()
	if _, err = f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

// ObserveRelease records an update only when an observed executable changes.
// On first observation there is no evidence of a prior deployment time.
func (h *History) ObserveRelease(r Release, now time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	old := h.state.Release
	if old.Identity == r.Identity && old.Identity != "" {
		return
	}
	r.ObservedAt = now.UTC()
	if old.Identity != "" {
		t := now.UTC()
		r.UpdatedAt = &t
	}
	h.state.Release = r
}

func CurrentRelease() Release {
	r := Release{Version: "development"}
	if b, ok := debug.ReadBuildInfo(); ok {
		for _, s := range b.Settings {
			if s.Key == "vcs.revision" {
				r.Commit = s.Value
			}
		}
		if b.Main.Version != "" && b.Main.Version != "(devel)" {
			r.Version = b.Main.Version
		}
	}
	exe, err := os.Executable()
	if err != nil {
		return r
	}
	if f, err := os.Open(exe); err == nil {
		hash := sha256.New()
		_, copyErr := io.Copy(hash, f)
		_ = f.Close()
		if copyErr == nil {
			r.Identity = hex.EncodeToString(hash.Sum(nil))
		}
	}
	// Portable bundles carry their release tag alongside the executable.
	if b, err := os.ReadFile(filepath.Join(filepath.Dir(exe), "release-assets", "manifest.json")); err == nil {
		var m struct{ Version, Commit string }
		if json.Unmarshal(b, &m) == nil && m.Version != "" {
			r.Version = m.Version
			r.Commit = m.Commit
		}
	}
	return r
}

// Run immediately samples and then samples every ten seconds. Disk writes occur
// once a minute and on shutdown. A failed write remains visible in the API.
func (h *History) Run(ctx context.Context, path string, online func() int) {
	h.Sample(time.Now(), online())
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	n := 0
	save := func() {
		err := h.Save(path)
		h.mu.Lock()
		h.state.PersistenceError = ""
		if err != nil {
			h.state.PersistenceError = err.Error()
		}
		h.mu.Unlock()
	}
	save()
	for {
		select {
		case <-ctx.Done():
			h.Sample(time.Now(), online())
			save()
			return
		case now := <-tick.C:
			h.Sample(now, online())
			n++
			if n%6 == 0 {
				save()
			}
		}
	}
}
