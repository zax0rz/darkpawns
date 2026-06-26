package dreaming

import (
	"strings"
	"testing"
	"time"
)

func TestBuildSummaryUsesEventOccurredAt(t *testing.T) {
	g := NewMemoryGraph(DefaultGraphConfig())

	// Simulate two events that happened yesterday and today, but were both
	// inserted into the graph right now. The summary should reflect the
	// original event times, not the insertion time.
	yesterday := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	today := time.Date(2026, 6, 26, 14, 30, 0, 0, time.UTC)

	g.AddOrReinforceNode("agent-combat-1", NodeKindEvent, "fought a rat", 1, yesterday)
	g.AddOrReinforceNode("agent-movement-2", NodeKindEvent, "moved east", 0, today)

	summary := g.BuildSummary(1000)

	if !strings.Contains(summary, "Jun 25") {
		t.Errorf("expected summary to contain original date Jun 25, got:\n%s", summary)
	}
	if !strings.Contains(summary, "Jun 26") {
		t.Errorf("expected summary to contain original date Jun 26, got:\n%s", summary)
	}

	// The earlier event should appear before the later event.
	idx25 := strings.Index(summary, "Jun 25")
	idx26 := strings.Index(summary, "Jun 26")
	if idx25 == -1 || idx26 == -1 || idx25 > idx26 {
		t.Errorf("expected Jun 25 to appear before Jun 26 in summary, got:\n%s", summary)
	}
}

func TestBuildSummaryGroupsByOccurredAtGap(t *testing.T) {
	g := NewMemoryGraph(DefaultGraphConfig())

	// Two events on the same day but more than 30 minutes apart should
	// produce two sessions.
	t1 := time.Date(2026, 6, 26, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 26, 12, 0, 0, 0, time.UTC)

	g.AddOrReinforceNode("agent-combat-1", NodeKindEvent, "fought a rat", 1, t1)
	g.AddOrReinforceNode("agent-movement-2", NodeKindEvent, "moved east", 0, t2)

	summary := g.BuildSummary(1000)

	// With two sessions separated by >30 min, there should be two session headers.
	count := strings.Count(summary, "### Session")
	if count != 2 {
		t.Errorf("expected 2 session headers, got %d; summary:\n%s", count, summary)
	}
}
