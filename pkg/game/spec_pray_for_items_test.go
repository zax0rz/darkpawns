package game

import "testing"

// TestSpecPrayForItems_StaffNamesPatched pins the DP-1373 security
// divergence: C's pray immortality made Serapis/Orodreth level 40 and
// Frontline 39 (spec_procs.c:2081-2126). The port refuses with "Nice try."
// and changes nothing; every other name keeps C's silent TRUE.
func TestSpecPrayForItems_StaffNamesPatched(t *testing.T) {
	tests := []struct {
		name       string
		wantOutput string
	}{
		{name: "Serapis", wantOutput: "Nice try.\r\n"},
		{name: "Orodreth", wantOutput: "Nice try.\r\n"},
		{name: "Frontline", wantOutput: "Nice try.\r\n"},
		{name: "Unlisted", wantOutput: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			player.Name = tt.name
			startLevel := player.GetLevel()
			_ = lastMsg() // discard setup output

			if got := specPrayForItems(w, player, nil, "pray", "immortality"); !got {
				t.Fatal("immortality should be intercepted")
			}
			if got := player.GetLevel(); got != startLevel {
				t.Errorf("level = %d, want unchanged %d", got, startLevel)
			}
			if got := lastMsg(); got != tt.wantOutput {
				t.Errorf("output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}

func TestSpecPrayForItems_FallsThroughOrdinarySocial(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	_ = lastMsg() // discard setup output

	if got := specPrayForItems(w, player, nil, "look", ""); got {
		t.Fatal("non-pray commands should not be intercepted")
	}
	if got := lastMsg(); got != "" {
		t.Errorf("non-pray output = %q, want empty output", got)
	}

	if got := specPrayForItems(w, player, nil, "pray", "nobody"); got {
		t.Fatal("pray without a matching item should fall through")
	}
	if got := lastMsg(); got != "" {
		t.Errorf("output = %q, want empty output", got)
	}
}

func TestSpecPrayForItems_ItemRewardBranch(t *testing.T) {
	w, player, lastMsg := newSpecProcTestWorld(t)
	observer := NewPlayer(2, "Observer", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	_ = lastMsg() // discard setup output

	altar, err := w.SpawnObject(3001, -1)
	if err != nil {
		t.Fatalf("SpawnObject altar: %v", err)
	}
	if err := w.MoveObjectToRoom(altar, 1001); err != nil {
		t.Fatalf("MoveObjectToRoom altar: %v", err)
	}
	if !w.SetObjectExtraDesc(3001, "item_for_Tester", "a test reward") {
		t.Fatal("SetObjectExtraDesc should find the altar object")
	}
	player.SetGold(50)

	if got := specPrayForItems(w, player, nil, "pray", ""); !got {
		t.Fatal("matching item_for_ extra description should be intercepted")
	}
	if got := player.GetGold(); got != 0 {
		t.Errorf("gold = %d, want 0 after the 100-gold reward cost", got)
	}
	if got := len(w.GetItemsInRoom(1001)); got != 2 {
		t.Errorf("room item count = %d, want generated reward plus altar", got)
	}
	if got := lastMsg(); got == "" {
		t.Error("item branch should emit actor and observer messages")
	}
}
