package agentcli

import (
	"testing"
	"time"
)

// testState creates a GameState with the given HP for testing.
func testState(hp, maxHP int) *GameState {
	s := &GameState{}
	s.Player.Health = hp
	s.Player.MaxHealth = maxHP
	return s
}

func TestCombatStartFires(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{CombatStart: true}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	state := testState(100, 100)
	state.Fighting = "a angry goblin"
	events := []AgentEvent{
		{Seq: 1, Type: "combat_start", Timestamp: time.Now()},
	}

	eng.Check(state, events)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerCombatStart {
		t.Errorf("expected TriggerCombatStart, got %v", fired[0].Type)
	}
}

func TestCombatStartDoesNotFireWhenAlreadyFighting(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{CombatStart: true}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	state := testState(100, 100)
	state.Fighting = "a angry goblin"
	events := []AgentEvent{
		{Seq: 1, Type: "vars", Timestamp: time.Now()},
	}

	eng.Check(state, events)

	if len(fired) != 0 {
		t.Fatalf("expected 0 fires, got %d", len(fired))
	}
}

func TestLowHP(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{LowHP: 30}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	state := testState(25, 100)
	eng.Check(state, nil)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerLowHP {
		t.Errorf("expected TriggerLowHP, got %v", fired[0].Type)
	}
}

func TestLowHPDoesNotFireAboveThreshold(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{LowHP: 30}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	state := testState(50, 100)
	eng.Check(state, nil)

	if len(fired) != 0 {
		t.Fatalf("expected 0 fires, got %d", len(fired))
	}
}

func TestTellReceived(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{TellReceived: true}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	events := []AgentEvent{
		{Seq: 1, Type: "tell", Data: []byte(`{"from":"bob","message":"hello"}`)},
	}

	eng.Check(nil, events)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerTellReceived {
		t.Errorf("expected TriggerTellReceived, got %v", fired[0].Type)
	}
}

func TestBufferSize(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{BufferSize: 3}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	events := make([]AgentEvent, 5)
	for i := range events {
		events[i] = AgentEvent{Seq: uint64(i + 1), Type: "vars"}
	}

	eng.Check(nil, events)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerEventBufferFull {
		t.Errorf("expected TriggerEventBufferFull, got %v", fired[0].Type)
	}
}

func TestBufferSizeDoesNotFireBelowLimit(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{BufferSize: 10}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	events := make([]AgentEvent, 5)
	for i := range events {
		events[i] = AgentEvent{Seq: uint64(i + 1), Type: "vars"}
	}

	eng.Check(nil, events)

	if len(fired) != 0 {
		t.Fatalf("expected 0 fires, got %d", len(fired))
	}
}

func TestScheduledInterval(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{Interval: 100 * time.Millisecond}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	eng.mu.Lock()
	eng.lastWake = time.Now().Add(-200 * time.Millisecond)
	eng.mu.Unlock()

	eng.Check(nil, nil)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerScheduledInterval {
		t.Errorf("expected TriggerScheduledInterval, got %v", fired[0].Type)
	}
}

func TestScheduledIntervalDoesNotFireEarly(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{Interval: 10 * time.Second}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	eng.Check(nil, nil)

	if len(fired) != 0 {
		t.Fatalf("expected 0 fires, got %d", len(fired))
	}
}

func TestReset(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{Interval: 100 * time.Millisecond}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	eng.mu.Lock()
	eng.lastWake = time.Now().Add(-200 * time.Millisecond)
	eng.mu.Unlock()

	eng.Check(nil, nil)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire before reset, got %d", len(fired))
	}

	eng.Reset()

	fired = nil
	eng.Check(nil, nil)

	if len(fired) != 0 {
		t.Fatalf("expected 0 fires after reset, got %d", len(fired))
	}
}

func TestMultipleTriggers(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{
		CombatStart:  true,
		LowHP:        30,
		TellReceived: true,
	}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	state := testState(10, 100)
	state.Fighting = "a angry goblin"
	events := []AgentEvent{
		{Seq: 1, Type: "combat_start"},
		{Seq: 2, Type: "tell", Data: []byte(`{"from":"bob"}`)},
	}

	eng.Check(state, events)

	if len(fired) < 2 {
		t.Fatalf("expected at least 2 fires, got %d", len(fired))
	}

	types := map[TriggerType]bool{}
	for _, f := range fired {
		types[f.Type] = true
	}
	if !types[TriggerCombatStart] {
		t.Error("expected TriggerCombatStart to fire")
	}
	if !types[TriggerLowHP] {
		t.Error("expected TriggerLowHP to fire")
	}
	if !types[TriggerTellReceived] {
		t.Error("expected TriggerTellReceived to fire")
	}
}

func TestPlayerEntered(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{PlayerEntered: true}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	events := []AgentEvent{
		{Seq: 1, Type: "entity_enter", Data: []byte(`{"name":"Bob","type":"player"}`)},
	}

	eng.Check(nil, events)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerPlayerEntered {
		t.Errorf("expected TriggerPlayerEntered, got %v", fired[0].Type)
	}
}

func TestDeath(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{Death: true}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	state := testState(0, 100)

	eng.Check(state, nil)

	if len(fired) != 1 {
		t.Fatalf("expected 1 fire, got %d", len(fired))
	}
	if fired[0].Type != TriggerDeath {
		t.Errorf("expected TriggerDeath, got %v", fired[0].Type)
	}
}

func TestNilCallback(t *testing.T) {
	eng := NewWakeEngine(WakeTrigger{CombatStart: true, LowHP: 30}, nil)

	state := testState(10, 100)
	state.Fighting = "a goblin"
	events := []AgentEvent{
		{Seq: 1, Type: "combat_start"},
	}

	eng.Check(state, events)
}

func TestNilState(t *testing.T) {
	var fired []WakeEvent
	eng := NewWakeEngine(WakeTrigger{CombatStart: true, LowHP: 30, Death: true}, func(e WakeEvent) {
		fired = append(fired, e)
	})

	eng.Check(nil, nil)

	if len(fired) != 0 {
		t.Fatalf("expected 0 fires with nil state, got %d", len(fired))
	}
}

func TestEventsOnlyProcessedOnce(t *testing.T) {
	var count int
	eng := NewWakeEngine(WakeTrigger{TellReceived: true}, func(e WakeEvent) {
		count++
	})

	events := []AgentEvent{
		{Seq: 1, Type: "tell", Data: []byte(`{"from":"bob"}`)},
	}

	eng.Check(nil, events)
	if count != 1 {
		t.Fatalf("expected 1 fire on first check, got %d", count)
	}

	eng.Check(nil, events)
	if count != 1 {
		t.Fatalf("expected still 1 fire after re-check, got %d", count)
	}
}
