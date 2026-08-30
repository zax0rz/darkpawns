package game

import "testing"

func TestSpecFlyExitUp_EntryGatesAndAudience(t *testing.T) {
	w, actor, _ := newSpecProcTestWorld(t)
	peer := NewPlayer(2, "Peer", actor.GetRoomVNum())
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer(peer): %v", err)
	}

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	clearMessages := func() {
		for name := range messages {
			delete(messages, name)
		}
	}

	if specFlyExitUp(nil, nil, nil, "up", "") {
		t.Fatal("nil invocation should fall through")
	}
	if specFlyExitUp(w, actor, nil, "look", "") {
		t.Fatal("non-up command should fall through")
	}
	if len(messages) != 0 {
		t.Fatalf("non-up command emitted messages: %#v", messages)
	}

	actor.SetLevel(LVL_IMMORT + 1)
	if specFlyExitUp(w, actor, nil, "up", "") {
		t.Fatal("level above LVL_IMMORT should pass through")
	}
	if len(messages) != 0 {
		t.Fatalf("high-level pass-through emitted messages: %#v", messages)
	}

	actor.SetLevel(LVL_IMMORT)
	actor.SetAffect(affFly, true)
	if specFlyExitUp(w, actor, nil, "up", "") {
		t.Fatal("flying player should pass through")
	}
	if len(messages) != 0 {
		t.Fatalf("flying pass-through emitted messages: %#v", messages)
	}

	actor.SetAffect(affFly, false)
	blockedRoom := actor.GetRoomVNum()
	if !specFlyExitUp(w, actor, nil, "up", "ignored") {
		t.Fatal("mortal non-flying player should be blocked")
	}
	if got := actor.GetRoomVNum(); got != blockedRoom {
		t.Errorf("blocked player room = %d, want unchanged room %d", got, blockedRoom)
	}
	if got, want := messages[actor.Name], "You try and jump up there but it's just too high.\r\n"; got != want {
		t.Errorf("actor message = %q, want %q", got, want)
	}
	if got, want := messages[peer.Name], "Tester jumps up and down in a vain attempt to travel upwards.\r\n"; got != want {
		t.Errorf("peer message = %q, want %q", got, want)
	}

	clearMessages()
	actor.SetLevel(LVL_IMMORT - 1)
	if !specFlyExitUp(w, actor, nil, "up", "") {
		t.Fatal("level at LVL_IMMORT-1 should be blocked")
	}
	if messages[actor.Name] == "" || messages[peer.Name] == "" {
		t.Fatalf("boundary blocked arm missing output: %#v", messages)
	}
}
