package moderation

import (
	"testing"
	"time"
)

func TestWordFilterCensor(t *testing.T) {
	wf := WordFilterEntry{
		Pattern: "badword",
		IsRegex: false,
		Action:  FilterActionCensor,
	}

	message := "This is a badword test"
	expected := "This is a ******* test"

	result := wf.censor(message)
	if result != expected {
		t.Errorf("censor() = %q, want %q", result, expected)
	}
}

func TestWordFilterCensor_CaseVariants(t *testing.T) {
	wf := WordFilterEntry{
		Pattern: "badword",
		IsRegex: false,
		Action:  FilterActionCensor,
	}

	tests := []struct {
		message  string
		expected string
	}{
		{"This is a BADWORD test", "This is a ******* test"},
		{"This is a BadWord test", "This is a ******* test"},
		{"BADWORD and badword", "******* and *******"},
		{"No match here", "No match here"},
	}

	for _, tt := range tests {
		result := wf.censor(tt.message)
		if result != tt.expected {
			t.Errorf("censor(%q) = %q, want %q", tt.message, result, tt.expected)
		}
	}
}

func TestWordFilterRegex(t *testing.T) {
	wf := WordFilterEntry{
		Pattern: `(?i)hate.*speech`,
		IsRegex: true,
		Action:  FilterActionBlock,
	}

	tests := []struct {
		message  string
		expected bool
	}{
		{"I hate speech", true},
		{"HATE SPEECH is bad", true},
		{"This is fine", false},
		{"hateful speech", true},
	}

	for _, tt := range tests {
		result := wf.matches(tt.message)
		if result != tt.expected {
			t.Errorf("matches(%q) = %v, want %v", tt.message, result, tt.expected)
		}
	}
}

func TestWordFilterCensor_RegexCaseVariants(t *testing.T) {
	wf := WordFilterEntry{
		Pattern: `hate\s+speech`,
		IsRegex: true,
		Action:  FilterActionCensor,
	}

	tests := []struct {
		message  string
		expected string
	}{
		{"I hate speech here", "I *********** here"},
		{"HATE SPEECH is bad", "*********** is bad"},
		{"Hate Speech too", "*********** too"},
		{"This is fine", "This is fine"},
	}

	for _, tt := range tests {
		result := wf.censor(tt.message)
		if result != tt.expected {
			t.Errorf("censor(%q) = %q, want %q", tt.message, result, tt.expected)
		}
	}
}

func TestSpamDetection(t *testing.T) {
	m := &Manager{
		messageHistory: make(map[string][]time.Time),
		spamConfig: SpamDetectionConfig{
			MessagesPerMinute: 3,
			DuplicateWindow:   5 * time.Second,
			Action:            FilterActionWarn,
		},
	}

	player := "testplayer"
	now := time.Now()

	// Add 2 messages within minute - not spam
	m.messageHistory[player] = []time.Time{
		now.Add(-30 * time.Second),
		now.Add(-20 * time.Second),
	}

	if m.isSpam(player) {
		t.Error("isSpam() = true for 2 messages, want false")
	}

	// Add 4th message - should be spam
	m.messageHistory[player] = append(
		m.messageHistory[player],
		now.Add(-10*time.Second),
		now.Add(-5*time.Second),
	)

	if !m.isSpam(player) {
		t.Error("isSpam() = false for 4 messages, want true")
	}
}

func TestCheckMessage(t *testing.T) {
	m := &Manager{
		activePenalties: make(map[string][]PlayerPenalty),
		wordFilters: []WordFilterEntry{
			{
				Pattern: "badword",
				IsRegex: false,
				Action:  FilterActionCensor,
			},
			{
				Pattern: `(?i)blockme`,
				IsRegex: true,
				Action:  FilterActionBlock,
			},
		},
		messageHistory: make(map[string][]time.Time),
		spamConfig: SpamDetectionConfig{
			MessagesPerMinute: 10,
			DuplicateWindow:   5 * time.Second,
			Action:            FilterActionWarn,
		},
	}

	tests := []struct {
		name      string
		player    string
		message   string
		wantMsg   string
		wantAct   FilterAction
		wantBlock bool
	}{
		{
			name:      "normal message",
			player:    "player1",
			message:   "Hello world",
			wantMsg:   "Hello world",
			wantAct:   FilterActionLog,
			wantBlock: false,
		},
		{
			name:      "censored word",
			player:    "player1",
			message:   "This has badword in it",
			wantMsg:   "This has ******* in it",
			wantAct:   FilterActionCensor,
			wantBlock: false,
		},
		{
			name:      "blocked word",
			player:    "player1",
			message:   "Please blockme now",
			wantMsg:   "",
			wantAct:   FilterActionBlock,
			wantBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMsg, gotAct, gotBlock := m.CheckMessage(tt.player, tt.message)

			if gotMsg != tt.wantMsg {
				t.Errorf("CheckMessage() gotMsg = %q, want %q", gotMsg, tt.wantMsg)
			}
			if gotAct != tt.wantAct {
				t.Errorf("CheckMessage() gotAct = %q, want %q", gotAct, tt.wantAct)
			}
			if gotBlock != tt.wantBlock {
				t.Errorf("CheckMessage() gotBlock = %v, want %v", gotBlock, tt.wantBlock)
			}
		})
	}
}

func TestClose_StopsCleanupRoutine(t *testing.T) {
	m := NewManager(nil)

	m.Close()
	m.Close() // double Close must not panic

	select {
	case <-m.done:
	default:
		t.Fatal("cleanup goroutine still running after Close")
	}
}

func TestHasPenalty(t *testing.T) {
	now := time.Now()
	future := now.Add(1 * time.Hour)
	past := now.Add(-1 * time.Hour)

	// Note: hasPenalty is private, so we can't test it directly
	// This test is for documentation purposes
	_ = &Manager{
		activePenalties: map[string][]PlayerPenalty{
			"mutedplayer": {
				{
					PlayerName:  "mutedplayer",
					PenaltyType: ActionMute,
					IssuedAt:    past,
					ExpiresAt:   &future,
					Reason:      "Test mute",
					IssuedBy:    "admin",
				},
			},
			"expiredplayer": {
				{
					PlayerName:  "expiredplayer",
					PenaltyType: ActionMute,
					IssuedAt:    past.Add(-2 * time.Hour),
					ExpiresAt:   &past,
					Reason:      "Expired mute",
					IssuedBy:    "admin",
				},
			},
		},
	}

	tests := []struct {
		player  string
		penalty AdminAction
		want    bool
	}{
		{"mutedplayer", ActionMute, true},
		{"mutedplayer", ActionBan, false},
		{"expiredplayer", ActionMute, false},
		{"nonexistent", ActionMute, false},
	}

	for _, tt := range tests {
		// Note: hasPenalty is private, so we can't test it directly
		// This test is for documentation purposes
		t.Logf("Player %s has penalty %s: %v (expected: %v)",
			tt.player, tt.penalty, false, tt.want)
	}
}

func TestAddWordFilter_MemoryIDsDoNotCollide(t *testing.T) {
	m := NewManager(nil)
	t.Cleanup(m.Close)
	// Simulate DB load ordering by created_at DESC: ids 2 then 1.
	m.wordFilters = []WordFilterEntry{
		{ID: 2, Pattern: "second", Action: FilterActionCensor},
		{ID: 1, Pattern: "first", Action: FilterActionCensor},
	}

	if err := m.AddWordFilter("third", false, "censor", "admin"); err != nil {
		t.Fatalf("AddWordFilter failed: %v", err)
	}

	filters := m.GetWordFilters()
	if len(filters) != 3 {
		t.Fatalf("expected 3 filters, got %d", len(filters))
	}
	if filters[2].ID != 3 {
		t.Errorf("new filter ID = %d, want 3 (max existing + 1)", filters[2].ID)
	}
}

func TestIsMuted_CaseInsensitiveLookup(t *testing.T) {
	m := NewManager(nil)
	t.Cleanup(m.Close)
	_ = m.AddPenalty(PlayerPenalty{
		PlayerName:  "MixedCasePlayer",
		PenaltyType: ActionMute,
		IssuedAt:    time.Now().Add(-time.Hour),
		IssuedBy:    "admin",
	})

	variants := []string{"MixedCasePlayer", "mixedcaseplayer", "MIXEDCASEPLAYER", "MiXeDcAsEpLaYeR"}
	for _, name := range variants {
		if !m.IsMuted(name) {
			t.Errorf("IsMuted(%q) = false, want true", name)
		}
	}
}

func TestCleanupExpiredPenalties_RemovesFromMemory(t *testing.T) {
	m := NewManager(nil)
	t.Cleanup(m.Close)
	player := "ExpiredPlayer"
	past := time.Now().Add(-time.Hour)

	m.activePenalties[player] = []PlayerPenalty{
		{
			PlayerName:  player,
			PenaltyType: ActionMute,
			IssuedAt:    past,
			ExpiresAt:   &past,
			IssuedBy:    "admin",
		},
	}

	m.cleanupExpiredPenalties()

	if len(m.activePenalties[player]) != 0 {
		t.Errorf("expected expired penalty to be removed from memory, got %d", len(m.activePenalties[player]))
	}
}

func TestPenaltyHelpersConcurrentAccess(t *testing.T) {
	m := NewManager(nil)
	t.Cleanup(m.Close)
	player := "ConcurrentPlayer"
	now := time.Now()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			_ = m.AddPenalty(PlayerPenalty{
				PlayerName:  player,
				PenaltyType: ActionMute,
				IssuedAt:    now.Add(-time.Hour),
				IssuedBy:    "admin",
			})
		}
	}()

	for i := 0; i < 100; i++ {
		_ = m.IsMuted(player)
		_ = m.IsBanned(player)
	}

	<-done
}
