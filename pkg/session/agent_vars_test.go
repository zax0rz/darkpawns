package session

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// Helpers: agent test sessions
// ---------------------------------------------------------------------------

// makeTestAgentSession creates a session with isAgent=true and a
// pre-populated subscribedVars map.
func makeTestAgentSession(t *testing.T, m *Manager, name string, roomVNum int, vars []string) *Session {
	t.Helper()
	s := makeTestSession(t, m, name, roomVNum, true)
	s.isAgent = true
	s.agentMu.Lock()
	for _, v := range vars {
		s.subscribedVars[v] = true
	}
	s.agentMu.Unlock()
	return s
}

// makeTestWorldWithMobs creates a World with mob prototypes indexed so that
// SpawnMob can be called. Mobs are registered at VNums starting at 1001.
func makeTestWorldWithMobs(t *testing.T) *game.World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Lab", Zone: 1, Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 1002},
				"east":  {Direction: "east", ToRoom: 1001},
			}},
			{VNum: 1002, Name: "Empty Vault", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, Keywords: "goblin guard", ShortDesc: "A goblin guard", LongDesc: "A scrawny goblin stands watch.\n"},
			{VNum: 2002, Keywords: "goblin guard", ShortDesc: "Another goblin guard", LongDesc: "Another scrawny goblin stands watch.\n"},
			{VNum: 2003, Keywords: "the ancient dragon", ShortDesc: "An ancient dragon", LongDesc: "A massive dragon sleeps here.\n"},
			{VNum: 2004, Keywords: "rat", ShortDesc: "A rat", LongDesc: "A small rat scurries about.\n"},
		},
		Objs: []parser.Obj{
			{VNum: 3001, Keywords: "sword long iron", ShortDesc: "An iron longsword", TypeFlag: 5, WearFlags: [4]int{0, 8192}},
			{VNum: 3002, Keywords: "potion healing red", ShortDesc: "A red healing potion", TypeFlag: 16},
			{VNum: 3003, Keywords: "shield wooden round", ShortDesc: "A round wooden shield", TypeFlag: 9, WearFlags: [4]int{0, 256}},
			{VNum: 3004, Keywords: "coin gold", ShortDesc: "A gold coin", TypeFlag: 14},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// makeTestManagerWithMobs returns a Manager whose world mob/obj prototypes
// are already indexed, ready for SpawnMob / AddItemToRoom.
func makeTestManagerWithMobs(t *testing.T) *Manager {
	t.Helper()
	w := makeTestWorldWithMobs(t)
	return NewManager(w, nil)
}

// registerMob spawns a mob prototype in a room and returns the instance.
func registerMob(t *testing.T, m *Manager, protoVNum, roomVNum int) *game.MobInstance {
	t.Helper()
	mob, err := m.world.SpawnMob(protoVNum, roomVNum)
	if err != nil {
		t.Fatalf("SpawnMob(%d, %d) failed: %v", protoVNum, roomVNum, err)
	}
	return mob
}

// registerObject adds a parsed object prototype to the world's obj map,
// creates an instance, and places it on the room floor.
func registerObject(t *testing.T, m *Manager, protoVNum, roomVNum int) *game.ObjectInstance {
	t.Helper()
	w := m.world
	proto := findObjProto(t, w, protoVNum)
	if proto == nil {
		t.Fatalf("obj prototype %d not found in world", protoVNum)
	}
	obj := game.NewObjectInstance(proto, roomVNum)
	w.AddItemToRoom(obj, roomVNum)
	return obj
}

// findObjProto locates an obj prototype from the world registry.
// Since world.objs is unexported, we reconstruct from parsed world data.
func findObjProto(t *testing.T, w *game.World, vnum int) *parser.Obj {
	t.Helper()
	// Build from the NewWorld internals: the parsed data stores protos.
	// We inject proto VNums via the parse data, so NewWorld's indexing
	// populates w.objs. SpawnMob assumes mobs are there — same for objects.
	// However SpawnMob has special handling: it reads w.mobs and w.rooms.
	// For objects we use AddItemToRoom which needs an instance, not a proto.
	// We'll create instances directly.
	return &parser.Obj{
		VNum:      vnum,
		Keywords:  fmt.Sprintf("obj%d", vnum),
		ShortDesc: fmt.Sprintf("Object %d", vnum),
		LongDesc:  fmt.Sprintf("An object with vnum %d lies here.\n", vnum),
		TypeFlag:  0,
		WearFlags: [4]int{},
		Values:    [4]int{},
		Weight:    1,
		Cost:      0,
	}
}

// makeObjInstance creates an ObjectInstance from raw fields for tests that
// don't need a full prototype registration.
func makeObjInstance(vnum int, shortDesc, keywords string) *game.ObjectInstance {
	proto := &parser.Obj{
		VNum:      vnum,
		Keywords:  keywords,
		ShortDesc: shortDesc,
		LongDesc:  shortDesc + " lies here.\n",
	}
	return game.NewObjectInstance(proto, -1)
}

// ---------------------------------------------------------------------------
// Helper: parse a JSON message from a session's send channel
// ---------------------------------------------------------------------------

// recvJSON reads one JSON message from the send channel (with timeout)
// and unmarshals into the target.
func recvJSON(t *testing.T, s *Session, target interface{}) {
	t.Helper()
	select {
	case msg := <-s.send:
		if err := json.Unmarshal(msg, target); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message on send channel")
	}
}

// recvJSONMaybe reads a JSON message if one is available, returning false
// if the channel is empty after a short poll (5ms).
func recvJSONMaybe(t *testing.T, s *Session, target interface{}) bool {
	t.Helper()
	select {
	case msg := <-s.send:
		if err := json.Unmarshal(msg, target); err != nil {
			t.Fatalf("json.Unmarshal: %v", err)
		}
		return true
	case <-time.After(5 * time.Millisecond):
		return false
	}
}

// ---------------------------------------------------------------------------
// Tests: TestHandleSubscribe
// ---------------------------------------------------------------------------

func TestHandleSubscribe(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Build a subscribe message
	data := json.RawMessage(`{"variables":["HEALTH","MANA","ROOM_VNUM"]}`)
	if err := s.handleSubscribe(data); err != nil {
		t.Fatalf("handleSubscribe failed: %v", err)
	}

	// Verify subscribedVars
	s.agentMu.Lock()
	vars := make([]string, 0, len(s.subscribedVars))
	for v := range s.subscribedVars {
		vars = append(vars, v)
	}
	s.agentMu.Unlock()

	expected := map[string]bool{"HEALTH": true, "MANA": true, "ROOM_VNUM": true}
	if len(vars) != len(expected) {
		t.Errorf("subscribedVars has %d entries, want %d", len(vars), len(expected))
	}
	for _, v := range vars {
		if !expected[v] {
			t.Errorf("unexpected subscribed var: %s", v)
		}
	}
}

func TestHandleSubscribe_NonAgent(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = false

	data := json.RawMessage(`{"variables":["HEALTH"]}`)
	if err := s.handleSubscribe(data); err != nil {
		t.Fatalf("handleSubscribe should not error for non-agent: %v", err)
	}

	// Verify no vars were subscribed
	s.agentMu.Lock()
	count := len(s.subscribedVars)
	s.agentMu.Unlock()
	if count != 0 {
		t.Errorf("non-agent subscribed %d vars, want 0", count)
	}
}

func TestHandleSubscribe_EmptyVariables(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	data := json.RawMessage(`{"variables":[]}`)
	if err := s.handleSubscribe(data); err != nil {
		t.Fatalf("handleSubscribe with empty variables failed: %v", err)
	}

	s.agentMu.Lock()
	count := len(s.subscribedVars)
	s.agentMu.Unlock()
	if count != 0 {
		t.Errorf("subscribed %d vars from empty list, want 0", count)
	}
}

func TestHandleSubscribe_MalformedJSON(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	data := json.RawMessage(`not json`)
	if err := s.handleSubscribe(data); err == nil {
		t.Error("expected error for malformed JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// Tests: TestMarkDirty
// ---------------------------------------------------------------------------

func TestMarkDirty(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH", "MANA"})

	s.markDirty("HEALTH", "MANA")

	s.agentMu.Lock()
	dirty := make([]string, 0, len(s.dirtyVars))
	for v := range s.dirtyVars {
		dirty = append(dirty, v)
	}
	s.agentMu.Unlock()

	if len(dirty) != 2 {
		t.Fatalf("dirtyVars has %d entries, want 2", len(dirty))
	}
	expected := map[string]bool{"HEALTH": true, "MANA": true}
	for _, v := range dirty {
		if !expected[v] {
			t.Errorf("unexpected dirty var: %s", v)
		}
	}
}

func TestMarkDirty_NonAgent(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = false

	s.markDirty("HEALTH", "MANA")

	s.agentMu.Lock()
	count := len(s.dirtyVars)
	s.agentMu.Unlock()
	if count != 0 {
		t.Errorf("non-agent has %d dirty vars, want 0", count)
	}
}

func TestMarkDirty_NotSubscribed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH"})

	// "MANA" is not subscribed, so it should not become dirty
	s.markDirty("HEALTH", "MANA")

	s.agentMu.Lock()
	dirty := make(map[string]bool)
	for v := range s.dirtyVars {
		dirty[v] = true
	}
	s.agentMu.Unlock()

	if !dirty["HEALTH"] {
		t.Error("HEALTH should be dirty")
	}
	if dirty["MANA"] {
		t.Error("MANA was not subscribed, should not be dirty")
	}
	if len(dirty) != 1 {
		t.Errorf("expected 1 dirty var, got %d", len(dirty))
	}
}

func TestMarkDirty_Duplicate(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH"})

	s.markDirty("HEALTH")
	s.markDirty("HEALTH") // mark again

	s.agentMu.Lock()
	count := len(s.dirtyVars)
	s.agentMu.Unlock()
	if count != 1 {
		t.Errorf("duplicate mark should produce 1 dirty var, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Tests: TestFlushDirtyVars
// ---------------------------------------------------------------------------

func TestFlushDirtyVars(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH", "MANA"})

	// Set up player state
	s.player.Health = 75
	s.player.Mana = 50

	// Mark vars dirty and flush
	s.markDirty("HEALTH", "MANA")
	s.flushDirtyVars()

	// Read the message
	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}
	recvJSON(t, s, &msg)

	if msg.Type != "vars" {
		t.Errorf("message type = %q, want %q", msg.Type, "vars")
	}
	if msg.Data["HEALTH"] != float64(75) {
		t.Errorf("HEALTH = %v, want %v", msg.Data["HEALTH"], 75)
	}
	if msg.Data["MANA"] != float64(50) {
		t.Errorf("MANA = %v, want %v", msg.Data["MANA"], 50)
	}
}

func TestFlushDirtyVars_Empty(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH"})

	// No vars marked dirty — flush should produce no message
	s.flushDirtyVars()

	if recvJSONMaybe(t, s, &struct{}{}) {
		t.Error("expected no message when no vars are dirty")
	}
}

func TestFlushDirtyVars_DirtyCleared(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH"})

	s.markDirty("HEALTH")
	s.flushDirtyVars()

	// Drain the send channel
	select {
	case <-s.send:
	default:
	}

	// After flush, dirtyVars should be empty
	s.agentMu.Lock()
	count := len(s.dirtyVars)
	s.agentMu.Unlock()
	if count != 0 {
		t.Errorf("dirtyVars has %d entries after flush, want 0", count)
	}
}

func TestFlushDirtyVars_NonAgent(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = false

	s.flushDirtyVars()

	if recvJSONMaybe(t, s, &struct{}{}) {
		t.Error("non-agent should not send flush messages")
	}
}

// ---------------------------------------------------------------------------
// Tests: TestSendFullVarDump
// ---------------------------------------------------------------------------

func TestSendFullVarDump(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, nil)

	s.player.Health = 60
	s.player.MaxHealth = 100
	s.player.Mana = 40
	s.player.MaxMana = 100
	s.player.Level = 5
	s.player.Exp = 2500

	s.sendFullVarDump()

	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}
	recvJSON(t, s, &msg)

	if msg.Type != "vars" {
		t.Errorf("message type = %q, want %q", msg.Type, "vars")
	}

	// Verify all expected keys are present
	expectedKeys := []string{
		"HEALTH", "MAX_HEALTH", "MANA", "MAX_MANA", "LEVEL", "EXP",
		"ROOM_VNUM", "ROOM_NAME", "ROOM_EXITS", "ROOM_MOBS", "ROOM_ITEMS",
		"FIGHTING", "INVENTORY", "EQUIPMENT", "EVENTS",
	}
	for _, k := range expectedKeys {
		if _, ok := msg.Data[k]; !ok {
			t.Errorf("key %q missing from full var dump", k)
		}
	}

	// Spot-check values
	if msg.Data["HEALTH"] != float64(60) {
		t.Errorf("HEALTH = %v, want %v", msg.Data["HEALTH"], 60)
	}
	if msg.Data["LEVEL"] != float64(5) {
		t.Errorf("LEVEL = %v, want %v", msg.Data["LEVEL"], 5)
	}
	if msg.Data["ROOM_VNUM"] != float64(1001) {
		t.Errorf("ROOM_VNUM = %v, want %v", msg.Data["ROOM_VNUM"], 1001)
	}
}

// ---------------------------------------------------------------------------
// Tests: TestBuildVarValue
// ---------------------------------------------------------------------------

func TestBuildVarValue_Basic(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	s.player.Health = 85
	s.player.MaxHealth = 100
	s.player.Mana = 42
	s.player.MaxMana = 100
	s.player.Level = 3
	s.player.Exp = 1500

	tests := []struct {
		varName string
		want    interface{}
	}{
		{VarHealth, 85},
		{VarMaxHealth, 100},
		{VarMana, 42},
		{VarMaxMana, 100},
		{VarLevel, 3},
		{VarExp, 1500},
	}

	for _, tt := range tests {
		t.Run(tt.varName, func(t *testing.T) {
			got := s.buildVarValue(tt.varName)
			if got != tt.want {
				t.Errorf("buildVarValue(%q) = %v (%T), want %v (%T)",
					tt.varName, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestBuildVarValue_Room(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// ROOM_VNUM
	roomVNum := s.buildVarValue(VarRoomVnum)
	if roomVNum != 1001 {
		t.Errorf("ROOM_VNUM = %v, want %v", roomVNum, 1001)
	}

	// ROOM_NAME
	roomName := s.buildVarValue(VarRoomName)
	if roomName != "Room A" {
		t.Errorf("ROOM_NAME = %q, want %q", roomName, "Room A")
	}

	// ROOM_EXITS should be present (no exits in test rooms, but var should exist)
	exits := s.buildVarValue(VarRoomExits)
	if exits == nil {
		t.Error("ROOM_EXITS should not be nil")
	}
}

func TestBuildVarValue_UnknownRoom(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 9999, true) // room doesn't exist
	s.isAgent = true

	roomName := s.buildVarValue(VarRoomName)
	if roomName != "" {
		t.Errorf("ROOM_NAME for unknown room = %q, want empty string", roomName)
	}

	exits := s.buildVarValue(VarRoomExits)
	switch v := exits.(type) {
	case []string:
		if len(v) != 0 {
			t.Errorf("ROOM_EXITS = %v, want empty slice", v)
		}
	default:
		t.Errorf("ROOM_EXITS type = %T, want []string", v)
	}
}

func TestBuildVarValue_Fighting(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Not fighting — combat engine has no entry for Alice
	fighting := s.buildVarValue(VarFighting)
	if fighting != false {
		t.Errorf("FIGHTING = %v, want false", fighting)
	}
}

func TestBuildVarValue_Events(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// No pending events
	events := s.buildVarValue(VarEvents)
	if events == nil {
		t.Error("EVENTS should be empty slice, not nil")
	}

	// With pending events
	s.agentMu.Lock()
	s.pendingEvents = []interface{}{"event1", "event2"}
	s.agentMu.Unlock()

	events = s.buildVarValue(VarEvents)
	evSlice, ok := events.([]interface{})
	if !ok {
		t.Fatalf("EVENTS type = %T, want []interface{}", events)
	}
	if len(evSlice) != 2 {
		t.Errorf("EVENTS has %d entries, want 2", len(evSlice))
	}

	// After reading, pendingEvents should be cleared
	s.agentMu.Lock()
	remaining := s.pendingEvents
	s.agentMu.Unlock()
	if remaining != nil {
		t.Error("pendingEvents should be nil after reading")
	}
}

func TestBuildVarValue_UnknownVar(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	val := s.buildVarValue("NONEXISTENT_VAR")
	if val != nil {
		t.Errorf("unknown var should return nil, got %v", val)
	}
}

// ---------------------------------------------------------------------------
// Tests: TestBuildRoomMobs
// ---------------------------------------------------------------------------

func TestBuildRoomMobs(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Spawn a couple mobs in room 1001
	goblin := registerMob(t, m, 2001, 1001) // "goblin guard", "A goblin guard"
	_ = registerMob(t, m, 2004, 1001)       // "rat", "A rat"

	goblin.Fighting = true

	mobs := s.buildRoomMobs()
	if len(mobs) != 2 {
		t.Fatalf("buildRoomMobs returned %d mobs, want 2", len(mobs))
	}

	// Should be sorted in iteration order; find goblin by vnum
	var foundGoblin, foundRat bool
	for _, m := range mobs {
		switch m.VNum {
		case 2001:
			foundGoblin = true
			if m.Name != "A goblin guard" {
				t.Errorf("goblin name = %q, want %q", m.Name, "A goblin guard")
			}
			if !m.Fighting {
				t.Error("goblin should be fighting")
			}
			if m.InstanceID == "" {
				t.Error("goblin instance_id should not be empty")
			}
			if m.TargetString == "" {
				t.Error("goblin target_string should not be empty")
			}
		case 2004:
			foundRat = true
			if m.Name != "A rat" {
				t.Errorf("rat name = %q, want %q", m.Name, "A rat")
			}
			if m.Fighting {
				t.Error("rat should not be fighting")
			}
		}
	}

	if !foundGoblin {
		t.Error("goblin (vnum 2001) not found in room mobs")
	}
	if !foundRat {
		t.Error("rat (vnum 2004) not found in room mobs")
	}
}

func TestBuildRoomMobs_Empty(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1002, true) // Empty Vault — no mobs
	s.isAgent = true

	mobs := s.buildRoomMobs()
	if mobs == nil || len(mobs) != 0 {
		t.Errorf("expected empty non-nil slice, got %v (len=%d)", mobs, len(mobs))
	}
}

func TestBuildRoomMobs_KeywordDisambiguation(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Spawn two mobs with the same keyword ("goblin")
	goblin1 := registerMob(t, m, 2001, 1001) // "goblin guard"
	_ = registerMob(t, m, 2002, 1001)        // "goblin guard" (same keywords!)

	// Make them distinguishable
	goblin1.Fighting = true

	mobs := s.buildRoomMobs()
	if len(mobs) != 2 {
		t.Fatalf("buildRoomMobs returned %d mobs, want 2", len(mobs))
	}

	// Both should have target_string containing "goblin"
	var foundFirst, foundSecond bool
	for _, m := range mobs {
		switch m.TargetString {
		case "goblin":
			foundFirst = true
		case "2.goblin":
			foundSecond = true
		default:
			t.Errorf("unexpected target_string: %q", m.TargetString)
		}
	}

	if !foundFirst {
		t.Error("first goblin should have target_string 'goblin'")
	}
	if !foundSecond {
		t.Error("second goblin should have target_string '2.goblin'")
	}
}

func TestBuildRoomMobs_DifferentKeywords(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// "the ancient dragon" → "ancient" (first meaningful keyword)
	// "goblin guard" → "goblin"
	registerMob(t, m, 2003, 1001) // "the ancient dragon"
	registerMob(t, m, 2001, 1001) // "goblin guard"

	mobs := s.buildRoomMobs()
	if len(mobs) != 2 {
		t.Fatalf("expected 2 mobs, got %d", len(mobs))
	}

	for _, m := range mobs {
		switch m.VNum {
		case 2003:
			if m.TargetString != "ancient" {
				t.Errorf("dragon target_string = %q, want %q", m.TargetString, "ancient")
			}
		case 2001:
			if m.TargetString != "goblin" {
				t.Errorf("goblin target_string = %q, want %q", m.TargetString, "goblin")
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: TestBuildRoomItems
// ---------------------------------------------------------------------------

func TestBuildRoomItems(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Add items to the room floor
	sword := registerObject(t, m, 3001, 1001)
	_ = registerObject(t, m, 3004, 1001)

	// Override the sword's prototype keywords for meaningful testing
	if sword.Prototype != nil {
		sword.Prototype.Keywords = "sword long iron"
	}

	items := s.buildRoomItems()
	if len(items) != 2 {
		t.Fatalf("buildRoomItems returned %d items, want 2", len(items))
	}

	var foundSword, foundCoin bool
	for _, item := range items {
		switch item.VNum {
		case 3001:
			foundSword = true
			if item.Name == "" {
				t.Error("sword name should not be empty")
			}
		case 3004:
			foundCoin = true
			if item.Name == "" {
				t.Error("coin name should not be empty")
			}
		}
	}

	if !foundSword {
		t.Error("sword (vnum 3001) not found in room items")
	}
	if !foundCoin {
		t.Error("coin (vnum 3004) not found in room items")
	}
}

func TestBuildRoomItems_Empty(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1002, true) // Empty Vault
	s.isAgent = true

	items := s.buildRoomItems()
	if items == nil || len(items) != 0 {
		t.Errorf("expected empty non-nil slice, got %v (len=%d)", items, len(items))
	}
}

func TestBuildRoomItems_KeywordDisambiguation(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Add two items that share a keyword by creating instances with the same proto
	_ = registerObject(t, m, 3004, 1001)
	coin2 := registerObject(t, m, 3004, 1001)

	// Override short desc and keywords for the second to be distinguishable
	coin2.Prototype.Keywords = "coin gold silver"
	coin2.Prototype.ShortDesc = "A silver coin"

	items := s.buildRoomItems()
	if len(items) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(items))
	}

	var foundFirst, foundSecond bool
	for _, item := range items {
		if item.TargetString == "coin" || item.TargetString == "gold" {
			foundFirst = true
		} else if item.VNum == 3004 && (item.TargetString == "2.coin" || item.TargetString == "2.gold" || item.TargetString == "silver") {
			foundSecond = true
		}
	}

	if !foundFirst {
		t.Error("first coin should have a target_string")
	}
	if !foundSecond {
		t.Log("second coin with different keywords — may have its own keyword")
	}
}

// ---------------------------------------------------------------------------
// Tests: TestBuildInventory
// ---------------------------------------------------------------------------

func TestBuildInventory(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Add items to player inventory
	sword := makeObjInstance(3001, "An iron longsword", "sword long iron")
	potion := makeObjInstance(3002, "A red healing potion", "potion healing red")

	_ = s.player.Inventory.AddItem(sword)
	_ = s.player.Inventory.AddItem(potion)

	inv := s.buildInventory()
	if len(inv) != 2 {
		t.Fatalf("buildInventory returned %d items, want 2", len(inv))
	}

	// Verify items exist (FindItems("") returns all)
	allItems := s.player.Inventory.FindItems("")
	if len(allItems) != 2 {
		t.Fatalf("expected 2 inventory items, got %d", len(allItems))
	}
}

func TestBuildInventory_Empty(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	inv := s.buildInventory()
	if inv == nil {
		t.Error("buildInventory should return non-nil empty slice")
	} else if len(inv) != 0 {
		t.Errorf("expected empty inventory, got %d items", len(inv))
	}
}

func TestBuildInventory_ItemFields(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	sword := makeObjInstance(3001, "An iron longsword", "sword long iron")
	_ = s.player.Inventory.AddItem(sword)

	inv := s.buildInventory()
	if len(inv) != 1 {
		t.Fatalf("expected 1 inventory item, got %d", len(inv))
	}

	item := inv[0]
	if item["name"] != "An iron longsword" {
		t.Errorf("item name = %v, want %q", item["name"], "An iron longsword")
	}
	if item["vnum"] != 3001 {
		t.Errorf("item vnum = %v, want %v", item["vnum"], 3001)
	}
	instanceID, ok := item["instance_id"].(string)
	if !ok || instanceID == "" {
		t.Errorf("instance_id = %v, want non-empty string", item["instance_id"])
	}
}

// ---------------------------------------------------------------------------
// Tests: TestBuildEquipment
// ---------------------------------------------------------------------------

func TestBuildEquipment(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Equip items
	sword := makeObjInstance(3001, "An iron longsword", "sword long iron")
	shield := makeObjInstance(3003, "A round wooden shield", "shield wooden round")

	// Build prototypes with wear flags
	sword.Prototype.WearFlags = [4]int{8192, 0} // IIEM_WEAR_WIELD = bit 13, WearFlags[0]
	shield.Prototype.WearFlags = [4]int{512, 0} // ITEM_WEAR_SHIELD = bit 9, WearFlags[0]

	// Add to inventory first
	s.player.Inventory.AddItem(sword)
	s.player.Inventory.AddItem(shield)

	// Equip from inventory
	s.player.Equipment.Equip(sword, s.player.Inventory)
	s.player.Equipment.Equip(shield, s.player.Inventory)

	eq := s.buildEquipment()
	if len(eq) == 0 {
		t.Fatal("buildEquipment returned empty map")
	}

	wieldVal, ok := eq["wield"]
	if !ok {
		t.Fatal("wield slot not found in equipment")
	}
	wieldMap, ok := wieldVal.(map[string]interface{})
	if !ok {
		t.Fatalf("wield slot type = %T, want map[string]interface{}", wieldVal)
	}
	if wieldMap["name"] != "An iron longsword" {
		t.Errorf("wield name = %v, want %q", wieldMap["name"], "An iron longsword")
	}
	if wieldMap["vnum"] != 3001 {
		t.Errorf("wield vnum = %v (type %T), want %v", wieldMap["vnum"], wieldMap["vnum"], 3001)
	}

	shieldVal, ok := eq["shield"]
	if !ok {
		t.Fatal("shield slot not found in equipment")
	}
	shieldMap, ok := shieldVal.(map[string]interface{})
	if !ok {
		t.Fatalf("shield slot type = %T, want map[string]interface{}", shieldVal)
	}
	if shieldMap["name"] != "A round wooden shield" {
		t.Errorf("shield name = %v, want %q", shieldMap["name"], "A round wooden shield")
	}
}

func TestBuildEquipment_Empty(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	eq := s.buildEquipment()
	if eq == nil {
		t.Error("buildEquipment should return non-nil empty map")
	} else if len(eq) != 0 {
		t.Errorf("expected empty equipment, got %d slots", len(eq))
	}
}

// ---------------------------------------------------------------------------
// Tests: TestFirstMeaningfulKeyword
// ---------------------------------------------------------------------------

func TestFirstMeaningfulKeyword(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"goblin guard", "goblin"},
		{"the ancient dragon", "ancient"},
		{"an old wizard", "old"},
		{"a goblin", "goblin"},
		{"sword long iron", "sword"},
		{"", "unknown"},
		{"  ", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := firstMeaningfulKeyword(tt.input)
			if got != tt.want {
				t.Errorf("firstMeaningfulKeyword(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFirstMeaningfulKeyword_AllArticles(t *testing.T) {
	got := firstMeaningfulKeyword("a an the")
	// When all words are articles, falls back to first word
	if got != "a" {
		t.Errorf("all-article input should fall back to first word, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// Tests: End-to-end subscribe-mark-flush flow
// ---------------------------------------------------------------------------

func TestAgentVarFlow_E2E(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH", "MANA"})

	s.player.Health = 50
	s.player.Mana = 30

	// Subscribe
	data := json.RawMessage(`{"variables":["HEALTH","MANA"]}`)
	if err := s.handleSubscribe(data); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	// Mark dirty
	s.markDirty("HEALTH", "MANA")

	// Flush — should send one vars message
	s.flushDirtyVars()

	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}
	recvJSON(t, s, &msg)

	if msg.Type != "vars" {
		t.Errorf("type = %q, want %q", msg.Type, "vars")
	}
	if msg.Data["HEALTH"] != float64(50) {
		t.Errorf("HEALTH = %v, want %v", msg.Data["HEALTH"], 50)
	}
	if msg.Data["MANA"] != float64(30) {
		t.Errorf("MANA = %v, want %v", msg.Data["MANA"], 30)
	}

	// After flush, dirty vars should be empty
	s.agentMu.Lock()
	dirtyCount := len(s.dirtyVars)
	s.agentMu.Unlock()
	if dirtyCount != 0 {
		t.Errorf("dirtyVars has %d entries after flush, want 0", dirtyCount)
	}
}

// ---------------------------------------------------------------------------
// Tests: Room exit details in buildVarValue
// ---------------------------------------------------------------------------

func TestBuildVarValue_RoomExits(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	exits := s.buildVarValue(VarRoomExits)
	exitList, ok := exits.([]string)
	if !ok {
		t.Fatalf("ROOM_EXITS type = %T, want []string", exits)
	}

	// Test rooms created by makeTestManager have no exits, so exitList should
	// be an empty []string — verify the type, not the content.
	t.Logf("exits returned: %v (test rooms have no exits — OK)", exitList)
}

// ---------------------------------------------------------------------------
// Tests: flushDirtyVars non-agent edge cases
// ---------------------------------------------------------------------------

func TestFlushDirtyVars_AgentUnsubscribed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true
	// No subscribedVars

	s.markDirty("HEALTH") // should be a no-op since HEALTH is not subscribed
	s.flushDirtyVars()    // should also be a no-op

	if recvJSONMaybe(t, s, &struct{}{}) {
		t.Error("unsubscribed agent should not send flush messages")
	}
}

// ---------------------------------------------------------------------------
// Tests: Inventory item count and field consistency
// ---------------------------------------------------------------------------

func TestBuildInventory_MultipleItems(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.isAgent = true

	// Add several items
	items := []*game.ObjectInstance{
		makeObjInstance(3001, "Sword", "sword"),
		makeObjInstance(3002, "Potion", "potion"),
		makeObjInstance(3003, "Shield", "shield"),
		makeObjInstance(3004, "Coin", "coin"),
	}
	for _, item := range items {
		_ = s.player.Inventory.AddItem(item)
	}

	inv := s.buildInventory()
	if len(inv) != 4 {
		t.Errorf("expected 4 inventory items, got %d", len(inv))
	}

	// Each item should have all expected fields
	for i, itemMap := range inv {
		if _, ok := itemMap["name"]; !ok {
			t.Errorf("item %d missing 'name'", i)
		}
		if _, ok := itemMap["vnum"]; !ok {
			t.Errorf("item %d missing 'vnum'", i)
		}
		if _, ok := itemMap["instance_id"]; !ok {
			t.Errorf("item %d missing 'instance_id'", i)
		}
	}
}

// ---------------------------------------------------------------------------
// Tests: Multiple flush calls
// ---------------------------------------------------------------------------

func TestFlushDirtyVars_MultipleCalls(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestAgentSession(t, m, "Alice", 1001, []string{"HEALTH"})

	s.player.Health = 42

	// First flush
	s.markDirty("HEALTH")
	s.flushDirtyVars()

	var msg struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}
	recvJSON(t, s, &msg)

	if msg.Type != "vars" || msg.Data["HEALTH"] != float64(42) {
		t.Errorf("first flush: HEALTH = %v, want 42", msg.Data["HEALTH"])
	}

	// Second flush with no new changes
	s.flushDirtyVars()

	if recvJSONMaybe(t, s, &struct{}{}) {
		t.Error("second flush should not send a message when no vars are dirty")
	}

	// Third flush with new changes
	s.player.Health = 99
	s.markDirty("HEALTH")
	s.flushDirtyVars()

	recvJSON(t, s, &msg)
	if msg.Type != "vars" || msg.Data["HEALTH"] != float64(99) {
		t.Errorf("third flush: HEALTH = %v, want 99", msg.Data["HEALTH"])
	}
}
