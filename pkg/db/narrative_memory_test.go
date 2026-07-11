package db

import (
	"strings"
	"testing"
	"time"
)

func TestSalienceScore(t *testing.T) {
	tests := []struct {
		name           string
		valence        int
		hasSocialEvent bool
		isNovel        bool
		wantMin        float64
		wantMax        float64
	}{
		{"neutral", 0, false, false, 0.5, 0.5},
		{"positive valence", 3, false, false, 0.8, 0.8},
		{"negative valence", -3, false, false, 0.2, 0.2},
		{"social bonus", 0, true, false, 0.65, 0.65},
		{"novelty bonus", 0, false, true, 0.6, 0.6},
		{"all bonuses positive", 3, true, true, 1.0, 1.0},
		{"all bonuses negative", -3, true, true, 0.45, 0.45},
		{"clamped at floor", -3, false, false, 0.2, 0.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SalienceScore(tt.valence, tt.hasSocialEvent, tt.isNovel)
			const epsilon = 1e-9
			if got < tt.wantMin-epsilon || got > tt.wantMax+epsilon {
				t.Errorf("SalienceScore(%d, %v, %v) = %v, want between %v and %v",
					tt.valence, tt.hasSocialEvent, tt.isNovel, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestBootstrapBlock(t *testing.T) {
	t.Run("empty input returns empty string", func(t *testing.T) {
		if got := BootstrapBlock(nil, nil); got != "" {
			t.Errorf("BootstrapBlock(nil, nil) = %q, want empty string", got)
		}
	})

	t.Run("renders history and warnings", func(t *testing.T) {
		memories := []*NarrativeMemory{
			{Summary: "Killed an orc", Valence: 0},
			{Summary: "Found a gold coin", Valence: 1},
			{Summary: "Keldor took your gear", Valence: -3},
		}
		summaries := []string{"Session one summary"}

		block := BootstrapBlock(memories, summaries)

		if !strings.Contains(block, "CHARACTER HISTORY") {
			t.Errorf("expected CHARACTER HISTORY header, got:\n%s", block)
		}
		if !strings.Contains(block, "Recent sessions") {
			t.Errorf("expected session summaries, got:\n%s", block)
		}
		if !strings.Contains(block, "WORLD KNOWLEDGE") {
			t.Errorf("expected WORLD KNOWLEDGE section, got:\n%s", block)
		}
		if !strings.Contains(block, "ACTIVE WARNINGS") {
			t.Errorf("expected ACTIVE WARNINGS section, got:\n%s", block)
		}
		if !strings.Contains(block, "Keldor took your gear") {
			t.Errorf("expected warning summary, got:\n%s", block)
		}
	})

	t.Run("negative valence below threshold is history", func(t *testing.T) {
		memories := []*NarrativeMemory{
			{Summary: "Minor inconvenience", Valence: -1},
		}

		block := BootstrapBlock(memories, nil)

		if strings.Contains(block, "ACTIVE WARNINGS") {
			t.Errorf("expected no ACTIVE WARNINGS for valence -1, got:\n%s", block)
		}
		if !strings.Contains(block, "Minor inconvenience") {
			t.Errorf("expected summary in history, got:\n%s", block)
		}
	})
}

func TestNarrativeMemory_ValenceConstraintLogic(t *testing.T) {
	// The valence field is documented as -3 to +3; guard that SalienceScore
	// clamps even when called with out-of-range values.
	got := SalienceScore(10, false, false)
	if got != 1.0 {
		t.Errorf("SalienceScore(10, false, false) = %v, want 1.0 (clamped)", got)
	}
	got = SalienceScore(-10, false, false)
	if got != 0.1 {
		t.Errorf("SalienceScore(-10, false, false) = %v, want 0.1 (clamped)", got)
	}
}

func TestNarrativeMemory_ZeroValue(t *testing.T) {
	m := &NarrativeMemory{}
	if m.CreatedAt.IsZero() {
		// CreatedAt defaults to zero time; this is expected for a fresh struct.
		m.CreatedAt = time.Now()
	}
	if m.Salience == 0 {
		m.Salience = SalienceScore(m.Valence, m.SocialEventID != "", false)
	}
	if m.Salience < 0.1 || m.Salience > 1.0 {
		t.Errorf("default salience %v out of range", m.Salience)
	}
}
