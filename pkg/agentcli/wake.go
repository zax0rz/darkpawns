package agentcli

import (
	"log/slog"
	"sync"
	"time"
)

// TriggerType identifies the reason the LLM mind should wake.
type TriggerType int

const (
	TriggerCombatStart       TriggerType = iota // A mob started fighting us
	TriggerLowHP                                // HP below threshold
	TriggerTellReceived                         // Someone sent us a tell
	TriggerEventBufferFull                      // Event buffer exceeds size limit
	TriggerScheduledInterval                    // Periodic wake
	TriggerGoalCompleted                        // A waypoint or goal was reached
	TriggerPlayerEntered                        // A player entered the room
	TriggerDeath                                // We died
)

// TriggerName returns the string name of a TriggerType.
func (t TriggerType) TriggerName() string {
	switch t {
	case TriggerCombatStart:
		return "combat_start"
	case TriggerLowHP:
		return "low_hp"
	case TriggerTellReceived:
		return "tell_received"
	case TriggerEventBufferFull:
		return "event_buffer_full"
	case TriggerScheduledInterval:
		return "scheduled_interval"
	case TriggerGoalCompleted:
		return "goal_completed"
	case TriggerPlayerEntered:
		return "player_entered"
	case TriggerDeath:
		return "death"
	default:
		return "unknown"
	}
}

// WakeTrigger defines which events should cause the mind to wake.
type WakeTrigger struct {
	CombatStart   bool          `json:"combat_start"`
	LowHP         int           `json:"low_hp_threshold"` // percentage (e.g. 30 = 30%)
	TellReceived  bool          `json:"tell_received"`
	BufferSize    int           `json:"event_buffer_size"` // wake when buffer exceeds this
	Interval      time.Duration `json:"interval"`          // periodic wake interval
	PlayerEntered bool          `json:"player_entered"`
	Death         bool          `json:"death"`
}

// WakeEvent describes why the mind was woken.
type WakeEvent struct {
	Type      TriggerType
	Timestamp time.Time
	Detail    string // optional context
}

// WakeEngine monitors state and events, firing a callback when triggers fire.
type WakeEngine struct {
	trigger  WakeTrigger
	callback func(WakeEvent)
	lastWake time.Time
	lastSeq  uint64 // highest Seq we've already processed
	mu       sync.Mutex
}

// NewWakeEngine creates a WakeEngine. The callback is called (within Check)
// for each trigger that fires.
func NewWakeEngine(trigger WakeTrigger, callback func(WakeEvent)) *WakeEngine {
	return &WakeEngine{
		trigger:  trigger,
		callback: callback,
		lastWake: time.Now(),
	}
}

// Reset clears the engine state (e.g. after the mind reconnects).
func (we *WakeEngine) Reset() {
	we.mu.Lock()
	defer we.mu.Unlock()
	we.lastWake = time.Now()
	we.lastSeq = 0
}

// Check evaluates the current state and recent events against configured triggers.
// It is safe to call from multiple goroutines. The callback is invoked synchronously
// within the lock for each trigger that fires.
func (we *WakeEngine) Check(state *GameState, events []AgentEvent) {
	we.mu.Lock()
	defer we.mu.Unlock()

	if we.callback == nil {
		return
	}

	now := time.Now()

	// Filter to events we haven't seen yet.
	newEvents := filterNewEvents(events, we.lastSeq)

	// Update lastSeq to the highest Seq we've seen.
	if len(events) > 0 {
		maxSeq := we.lastSeq
		for _, e := range events {
			if e.Seq > maxSeq {
				maxSeq = e.Seq
			}
		}
		we.lastSeq = maxSeq
	}

	// 1. CombatStart: Fighting changed from "" to non-empty.
	//    Fire if there's a combat_start event in the new batch.
	if we.trigger.CombatStart && state != nil && state.Fighting != "" {
		for _, e := range newEvents {
			if e.Type == "combat_start" {
				we.callback(WakeEvent{
					Type:      TriggerCombatStart,
					Timestamp: now,
					Detail:    state.Fighting,
				})
				break
			}
		}
	}

	// 2. LowHP: HP/MaxHP * 100 < threshold.
	if we.trigger.LowHP > 0 && state != nil && state.Player.MaxHealth > 0 {
		pct := (state.Player.Health * 100) / state.Player.MaxHealth
		if pct < we.trigger.LowHP {
			we.callback(WakeEvent{
				Type:      TriggerLowHP,
				Timestamp: now,
				Detail:    itoa(state.Player.Health) + "/" + itoa(state.Player.MaxHealth) + " (" + itoa(pct) + "%)",
			})
		}
	}

	// 3. TellReceived: any event with Type "tell".
	if we.trigger.TellReceived {
		for _, e := range newEvents {
			if e.Type == "tell" {
				we.callback(WakeEvent{
					Type:      TriggerTellReceived,
					Timestamp: now,
					Detail:    string(e.Data),
				})
				break
			}
		}
	}

	// 4. BufferFull: len(events) exceeds BufferSize.
	if we.trigger.BufferSize > 0 && len(events) > we.trigger.BufferSize {
		we.callback(WakeEvent{
			Type:      TriggerEventBufferFull,
			Timestamp: now,
			Detail:    "buffer size " + itoa(len(events)) + " exceeds " + itoa(we.trigger.BufferSize),
		})
	}

	// 5. ScheduledInterval: time since lastWake exceeds Interval.
	if we.trigger.Interval > 0 && now.Sub(we.lastWake) >= we.trigger.Interval {
		we.callback(WakeEvent{
			Type:      TriggerScheduledInterval,
			Timestamp: now,
			Detail:    "periodic wake after " + now.Sub(we.lastWake).String(),
		})
		we.lastWake = now
	}

	// 6. PlayerEntered: any event with Type "entity_enter".
	if we.trigger.PlayerEntered {
		for _, e := range newEvents {
			if e.Type == "entity_enter" {
				we.callback(WakeEvent{
					Type:      TriggerPlayerEntered,
					Timestamp: now,
					Detail:    string(e.Data),
				})
				break
			}
		}
	}

	// 7. Death: HP is 0.
	if we.trigger.Death && state != nil && state.Player.Health <= 0 && state.Player.MaxHealth > 0 {
		we.callback(WakeEvent{
			Type:      TriggerDeath,
			Timestamp: now,
			Detail:    "health reached 0",
		})
	}

	slog.Debug("wake engine check", "events_new", len(newEvents), "hp", hpStr(state))
}

// filterNewEvents returns events with Seq > afterSeq.
func filterNewEvents(events []AgentEvent, afterSeq uint64) []AgentEvent {
	var out []AgentEvent
	for _, e := range events {
		if e.Seq > afterSeq {
			out = append(out, e)
		}
	}
	return out
}

// itoa is a minimal int-to-string for logging. Not general purpose.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := false
	if n < 0 {
		negative = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func hpStr(state *GameState) string {
	if state == nil {
		return "nil"
	}
	return itoa(state.Player.Health) + "/" + itoa(state.Player.MaxHealth)
}
