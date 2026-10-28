package telemetry

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ActivityCapacity = 20000
	journalCapacity  = 4096
)

// Activity is deliberately restricted to game identities and world metadata.
// No addresses, accounts, credentials, chat or arbitrary log attributes belong here.
type Activity struct {
	ID       uint64    `json:"id"`
	Time     time.Time `json:"time"`
	Kind     string    `json:"kind" enum:"login,character,death"`
	Player   string    `json:"player"`
	Class    string    `json:"class"`
	Level    int       `json:"level"`
	RoomVNum int       `json:"room_vnum"`
	RoomName string    `json:"room_name"`
	Cause    string    `json:"cause,omitempty" enum:"mob,player,trap,other,unknown"`
	MobName  string    `json:"mob_name"`
	MobVNum  int       `json:"mob_vnum"`
	MinLevel int       `json:"min_level"`
}

var journal = struct {
	mu       sync.Mutex
	sequence uint64
	entries  [journalCapacity]Activity
	dropped  atomic.Uint64
}{}

func RecordLogin(e Activity)       { Login(); e.Kind = "login"; recordActivity(e) }
func RecordCharacter(e Activity)   { Character(); e.Kind = "character"; recordActivity(e) }
func RecordPlayerDeath(e Activity) { playerDeaths.Add(1); e.Kind = "death"; recordActivity(e) }

func bounded(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)
	r := []rune(s)
	if len(r) > max {
		r = r[:max]
	}
	return string(r)
}

func recordActivity(e Activity) {
	// Observation must never stall a combat or entry goroutine (R1/R3/R4).
	if !journal.mu.TryLock() {
		journal.dropped.Add(1)
		return
	}
	defer journal.mu.Unlock()
	e.Time = time.Now().UTC()
	e.Player = bounded(e.Player, 32)
	e.Class = bounded(e.Class, 32)
	e.RoomName = bounded(e.RoomName, 160)
	e.MobName = bounded(e.MobName, 160)
	journal.sequence++
	journal.entries[(journal.sequence-1)%journalCapacity] = e
}

// Called under h.mu by the ten-second sampler. Each History has its own cursor;
// readers cannot steal events from another recorder or block game writers.
func (h *History) sampleActivity(now time.Time) {
	cutoff := now.Add(-7 * 24 * time.Hour)
	journal.mu.Lock()
	end := journal.sequence
	start := h.activityCursor + 1
	if end >= journalCapacity && start <= end-journalCapacity {
		h.state.ActivityDropped += end - journalCapacity + 1 - start
		start = end - journalCapacity + 1
	}
	for seq := start; seq <= end; seq++ {
		e := journal.entries[(seq-1)%journalCapacity]
		if e.Time.Before(cutoff) {
			continue
		}
		h.state.NextActivityID++
		e.ID = h.state.NextActivityID
		h.state.Activity = append(h.state.Activity, e)
	}
	h.activityCursor = end
	dropped := journal.dropped.Load()
	h.state.ActivityDropped += dropped - h.previousDropped
	h.previousDropped = dropped
	// Retention also releases names from the process-wide staging buffer.
	for i := range journal.entries {
		if journal.entries[i].Time.Before(cutoff) {
			journal.entries[i] = Activity{}
		}
	}
	journal.mu.Unlock()
	kept := h.state.Activity[:0]
	for i := range h.state.Activity {
		e := &h.state.Activity[i]
		if !e.Time.Before(cutoff) {
			kept = append(kept, *e)
		}
	}
	if len(kept) > ActivityCapacity {
		h.state.ActivityTrimmed += uint64(len(kept) - ActivityCapacity)
		copy(kept, kept[len(kept)-ActivityCapacity:])
		kept = kept[:ActivityCapacity]
	}
	clear(h.state.Activity[len(kept):])
	h.state.Activity = kept
}
