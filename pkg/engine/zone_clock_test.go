package engine

import "testing"

// src/structs.h PULSE_ZONE=10 RL_SEC; comm.c:805-808 extracts first.
func TestZoneHeartbeatTenSecondBoundary(t *testing.T) {
	var calls int
	var order []string
	gl := NewGameLoop(GameLoopCallbacks{OnExtractPending: func() { order = append(order, "extract") }, OnZoneUpdate: func() { calls++; order = append(order, "zone") }})
	for p := int64(1); p < 100; p++ {
		gl.heartbeat(p)
	}
	if calls != 0 {
		t.Fatalf("early resets: %d", calls)
	}
	gl.heartbeat(100)
	if calls != 1 {
		t.Fatalf("zone calls at C pulse 100 = %d, want 1", calls)
	}
	if order[len(order)-2] != "extract" || order[len(order)-1] != "zone" {
		t.Fatalf("order: %v", order)
	}
	for p := int64(101); p <= 600; p++ {
		gl.heartbeat(p)
	}
	if calls != 6 {
		t.Fatalf("zone calls per minute = %d, want 6", calls)
	}
}
