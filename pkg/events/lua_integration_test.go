package events

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// mockWorldForEvents is a minimal ScriptableWorld that supports event creation.
type mockWorldForEvents struct {
	scripting.ScriptableWorld
	eq        *EventQueue
	lastEvent uint64
}

func (m *mockWorldForEvents) CreateEvent(delay int, source, target, obj, argument int, trigger string, eventType int) uint64 {
	if m.eq == nil {
		return 0
	}
	m.lastEvent = m.eq.Create(int64(delay), source, target, obj, argument, trigger, eventType,
		func(_ context.Context, src, tgt, o, arg int, trig string, et int) int64 {
			return 0
		})
	return m.lastEvent
}

// findScriptsDir locates the test scripts directory.
func findScriptsDir(t *testing.T) string {
	scriptsDir := filepath.Join("..", "..", "test_scripts")
	if _, err := os.Stat(scriptsDir); !os.IsNotExist(err) {
		return scriptsDir
	}
	scriptsDir = filepath.Join("..", "..", "pkg", "scripting", "test_scripts")
	if _, err := os.Stat(scriptsDir); !os.IsNotExist(err) {
		return scriptsDir
	}
	t.Skip("test_scripts directory not found")
	return ""
}

// TestLuaCreateEventBinding verifies that the Lua create_event function
// correctly parses arguments and schedules an event.
func TestLuaCreateEventBinding(t *testing.T) {
	scriptsDir := findScriptsDir(t)

	eq := NewEventQueue(10 * time.Millisecond)
	mockWorld := &mockWorldForEvents{eq: eq}

	engine := scripting.NewEngine(scriptsDir, mockWorld)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	// Load a test script that calls create_event
	script := `
		-- Simulate sandstorm.lua pattern: create_event(me, NIL, NIL, NIL, "port", 1, LT_MOB)
		me = { vnum = 6102, name = "sandstorm" }
		LT_MOB = 1
		result = create_event(me, NIL, NIL, NIL, "port", 1, LT_MOB)
	`
	if err := engine.LState().DoString(script); err != nil {
		t.Fatalf("Failed to run test script: %v", err)
	}

	// Verify an event was created
	if mockWorld.lastEvent == 0 {
		t.Fatal("create_event did not return an event ID")
	}

	// Verify the event is pending
	if eq.Pending() != 1 {
		t.Fatalf("expected 1 pending event, got %d", eq.Pending())
	}

	// Process the event queue until the event fires
	ctx := context.Background()
	fired := false
	for i := 0; i < 50; i++ {
		eq.Process(ctx)
		if eq.Pending() == 0 {
			fired = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	if !fired {
		t.Fatal("event did not fire within expected time")
	}
}

// TestLuaCreateEventWithTarget verifies create_event with a target parameter.
func TestLuaCreateEventWithTarget(t *testing.T) {
	scriptsDir := findScriptsDir(t)

	eq := NewEventQueue(10 * time.Millisecond)
	mockWorld := &mockWorldForEvents{eq: eq}

	engine := scripting.NewEngine(scriptsDir, mockWorld)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	// Simulate bane.lua pattern: create_event(me, NIL, NIL, NIL, "bane_one", 10, LT_MOB)
	script := `
		me = { vnum = 1234, name = "bane" }
		ch = { vnum = 0, name = "player" }
		LT_MOB = 1
		result = create_event(me, ch, NIL, NIL, "bane_one", 10, LT_MOB)
	`
	if err := engine.LState().DoString(script); err != nil {
		t.Fatalf("Failed to run test script: %v", err)
	}

	if mockWorld.lastEvent == 0 {
		t.Fatal("create_event did not return an event ID")
	}

	// Event should be pending with delay=10 (10 * 20 = 200 pulses = 20 seconds at 100ms/pulse)
	// With 10ms pulse duration, that's 2 seconds. Let's verify it's scheduled.
	if eq.Pending() != 1 {
		t.Fatalf("expected 1 pending event, got %d", eq.Pending())
	}
}

// trackingWorld wraps mockWorldForEvents with a custom CreateEvent.
type trackingWorld struct {
	*mockWorldForEvents
	onCreate func(delay int, source, target, obj, argument int, trigger string, eventType int) uint64
}

func (t *trackingWorld) CreateEvent(delay int, source, target, obj, argument int, trigger string, eventType int) uint64 {
	return t.onCreate(delay, source, target, obj, argument, trigger, eventType)
}

// TestLuaCreateEventCancel verifies event cancellation via the queue.
func TestLuaCreateEventCancel(t *testing.T) {
	scriptsDir := findScriptsDir(t)

	var fired int64
	eq := NewEventQueue(10 * time.Millisecond)
	baseWorld := &mockWorldForEvents{eq: eq}

	mockWorld := &trackingWorld{
		mockWorldForEvents: baseWorld,
		onCreate: func(delay int, source, target, obj, argument int, trigger string, eventType int) uint64 {
			id := eq.Create(int64(delay), source, target, obj, argument, trigger, eventType,
				func(_ context.Context, src, tgt, o, arg int, trig string, et int) int64 {
					atomic.AddInt64(&fired, 1)
					return 0
				})
			baseWorld.lastEvent = id
			return id
		},
	}

	engine := scripting.NewEngine(scriptsDir, mockWorld)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	script := `
		me = { vnum = 999, name = "testmob" }
		LT_MOB = 1
		event_id = create_event(me, NIL, NIL, NIL, "test_trigger", 5, LT_MOB)
	`
	if err := engine.LState().DoString(script); err != nil {
		t.Fatalf("Failed to run test script: %v", err)
	}

	// Cancel the event
	if baseWorld.lastEvent > 0 {
		eq.Cancel(baseWorld.lastEvent)
	}

	// Process past the event time
	ctx := context.Background()
	for i := 0; i < 20; i++ {
		eq.Process(ctx)
	}

	if atomic.LoadInt64(&fired) != 0 {
		t.Fatal("cancelled event should not have fired")
	}
}

// TestLuaCreateEventDelayClamping verifies delay < 1 is clamped.
func TestLuaCreateEventDelayClamping(t *testing.T) {
	scriptsDir := findScriptsDir(t)

	var fired int64
	eq := NewEventQueue(10 * time.Millisecond)
	baseWorld := &mockWorldForEvents{eq: eq}

	mockWorld := &trackingWorld{
		mockWorldForEvents: baseWorld,
		onCreate: func(delay int, source, target, obj, argument int, trigger string, eventType int) uint64 {
			id := eq.Create(int64(delay), source, target, obj, argument, trigger, eventType,
				func(_ context.Context, src, tgt, o, arg int, trig string, et int) int64 {
					atomic.AddInt64(&fired, 1)
					return 0
				})
			return id
		},
	}

	engine := scripting.NewEngine(scriptsDir, mockWorld)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	// Pass delay 0 — should be clamped to 1
	script := `
		me = { vnum = 999, name = "testmob" }
		LT_MOB = 1
		create_event(me, NIL, NIL, NIL, "now", 0, LT_MOB)
	`
	if err := engine.LState().DoString(script); err != nil {
		t.Fatalf("Failed to run test script: %v", err)
	}

	// Event should fire on first Process (delay clamped to 1)
	ctx := context.Background()
	eq.Process(ctx)

	if atomic.LoadInt64(&fired) != 1 {
		t.Fatalf("event with delay=0 should fire after clamping, fired=%d", atomic.LoadInt64(&fired))
	}
}

// TestLuaCreateEventReturnsID verifies create_event returns the event ID to Lua.
func TestLuaCreateEventReturnsID(t *testing.T) {
	scriptsDir := findScriptsDir(t)

	eq := NewEventQueue(10 * time.Millisecond)
	mockWorld := &mockWorldForEvents{eq: eq}

	engine := scripting.NewEngine(scriptsDir, mockWorld)
	if engine == nil {
		t.Fatal("Failed to create engine")
	}

	script := `
		me = { vnum = 999, name = "testmob" }
		LT_MOB = 1
		event_id = create_event(me, NIL, NIL, NIL, "test", 1, LT_MOB)
		return event_id
	`
	fn, err := engine.LState().LoadString(script)
	if err != nil {
		t.Fatalf("Failed to load script: %v", err)
	}

	L := engine.LState()
	L.Push(fn)
	if err := L.PCall(0, 1, nil); err != nil {
		t.Fatalf("Failed to call script: %v", err)
	}

	ret := L.Get(-1)
	L.Pop(1)

	if ret.Type() != lua.LTNumber {
		t.Fatalf("expected number return, got %s", ret.Type().String())
	}

	id := uint64(lua.LVAsNumber(ret))
	if id == 0 {
		t.Fatal("create_event returned 0, expected non-zero event ID")
	}
}
