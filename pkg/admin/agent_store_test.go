package admin

import (
	"testing"
)

func TestAgentStore_New(t *testing.T) {
	store := NewAgentStore()
	agents := store.GetAgents()
	if len(agents) != 2 {
		t.Errorf("expected 2 agents (daeron, reek), got %d", len(agents))
	}
}

func TestAgentStore_UpdateStatus(t *testing.T) {
	store := NewAgentStore()

	// Update daeron to "active"
	agent, ok := store.UpdateAgentStatus("daeron", "active")
	if !ok {
		t.Fatal("UpdateAgentStatus returned false for daeron")
	}
	if agent.Status != "active" {
		t.Errorf("status = %q, want %q", agent.Status, "active")
	}

	// Verify via GetAgents
	agents := store.GetAgents()
	found := false
	for _, a := range agents {
		if a.AgentID == "daeron" && a.Status == "active" {
			found = true
			break
		}
	}
	if !found {
		t.Error("daeron status not reflected in GetAgents")
	}
}

func TestAgentStore_UpdateStatusUnknown(t *testing.T) {
	store := NewAgentStore()
	_, ok := store.UpdateAgentStatus("nonexistent", "active")
	if ok {
		t.Error("UpdateAgentStatus should return false for unknown agent")
	}
}

func TestAgentStore_AddAndGetFindings(t *testing.T) {
	store := NewAgentStore()

	// Add findings
	f1 := store.AddFinding("reek", "high", "nil panic", "handlers.go", 42, "potential nil dereference")
	if f1.ID != 1 {
		t.Errorf("first finding ID = %d, want 1", f1.ID)
	}
	if f1.Status != "open" {
		t.Errorf("default status = %q, want %q", f1.Status, "open")
	}

	f2 := store.AddFinding("daeron", "medium", "unused variable", "server.go", 10, "unused param")
	if f2.ID != 2 {
		t.Errorf("second finding ID = %d, want 2", f2.ID)
	}

	// Get all findings — no filters
	all := store.GetFindings("", "", "")
	if len(all) != 2 {
		t.Errorf("GetFindings() returned %d, want 2", len(all))
	}
}

func TestAgentStore_FindingsFiltered(t *testing.T) {
	store := NewAgentStore()

	store.AddFinding("reek", "critical", "CRIT-001", "panic.go", 1, "critical bug")
	store.AddFinding("reek", "high", "HIGH-001", "server.go", 2, "high bug")
	store.AddFinding("daeron", "low", "LOW-001", "config.go", 3, "low bug")

	// Filter by severity
	highFindings := store.GetFindings("", "high", "")
	if len(highFindings) != 1 {
		t.Errorf("filter severity=high returned %d, want 1", len(highFindings))
	}

	// Filter by source
	reekFindings := store.GetFindings("", "", "reek")
	if len(reekFindings) != 2 {
		t.Errorf("filter source=reek returned %d, want 2", len(reekFindings))
	}

	// Filter by status
	openFindings := store.GetFindings("open", "", "")
	if len(openFindings) != 3 {
		t.Errorf("filter status=open returned %d, want 3", len(openFindings))
	}
}

func TestAgentStore_UpdateFindingStatus(t *testing.T) {
	store := NewAgentStore()

	f := store.AddFinding("reek", "high", "test", "file.go", 1, "desc")
	if f.ID != 1 {
		t.Fatalf("expected ID 1, got %d", f.ID)
	}

	// Update status to confirmed
	updated, ok := store.UpdateFindingStatus(1, "confirmed")
	if !ok {
		t.Fatal("UpdateFindingStatus returned false")
	}
	if updated.Status != "confirmed" {
		t.Errorf("status = %q, want %q", updated.Status, "confirmed")
	}

	// Verify filter
	confirmed := store.GetFindings("confirmed", "", "")
	if len(confirmed) != 1 {
		t.Errorf("found %d confirmed, want 1", len(confirmed))
	}

	// Update unknown ID
	_, ok = store.UpdateFindingStatus(999, "fixed")
	if ok {
		t.Error("UpdateFindingStatus for unknown ID should return false")
	}
}

func TestAgentStore_TriageSummaries(t *testing.T) {
	store := NewAgentStore()

	// Add summaries
	s1 := store.AddTriageSummary("2026-05-01", "Good day", 5, 1, 2)
	if s1.ID != 1 {
		t.Errorf("first triage ID = %d, want 1", s1.ID)
	}

	s2 := store.AddTriageSummary("2026-05-02", "Bad day", 2, 4, 0)
	if s2.ID != 2 {
		t.Errorf("second triage ID = %d, want 2", s2.ID)
	}

	summaries := store.GetTriageSummaries()
	if len(summaries) != 2 {
		t.Errorf("got %d summaries, want 2", len(summaries))
	}
	if summaries[0].Date != "2026-05-01" {
		t.Errorf("first summary date = %q, want %q", summaries[0].Date, "2026-05-01")
	}
}

func TestAgentStore_EmptyFindings(t *testing.T) {
	store := NewAgentStore()
	findings := store.GetFindings("", "", "")
	if len(findings) != 0 {
		t.Errorf("new store should have 0 findings, got %d", len(findings))
	}
}

func TestAgentStore_ThreadSafety(t *testing.T) {
	store := NewAgentStore()

	done := make(chan bool)
	go func() {
		for i := 0; i < 50; i++ {
			store.AddFinding("reek", "high", "test", "f.go", i, "desc")
			store.UpdateFindingStatus(i+1, "confirmed")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 50; i++ {
			store.GetFindings("", "", "")
			store.GetAgents()
			store.UpdateAgentStatus("daeron", "active")
		}
		done <- true
	}()

	<-done
	<-done

	// Should have 50 findings (2 pre-seeded + 50 new)
	findings := store.GetFindings("", "", "")
	if len(findings) != 50 {
		t.Errorf("expected 50 findings, got %d (race)", len(findings))
	}
}
