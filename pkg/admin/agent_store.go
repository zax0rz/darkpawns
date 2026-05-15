package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AgentStatus represents the status of an AI agent.
type AgentStatus struct {
	AgentID     string    `json:"agent_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"` // "active", "idle", "error"
	LastRun     time.Time `json:"last_run"`
	Model       string    `json:"model"`
	Description string    `json:"description"`
}

// Finding represents a code review finding from Reek or Daeron.
type Finding struct {
	ID            int       `json:"id"`
	Source        string    `json:"source"`                 // "reek" or "daeron"
	Severity      string    `json:"severity"`               // "critical", "high", "medium", "low"
	Status        string    `json:"status"`                 // "open", "confirmed", "rejected", "fixed"
	Title         string    `json:"title"`
	File          string    `json:"file"`
	Line          int       `json:"line"`
	Description   string    `json:"description"`
	LinearIssueID string    `json:"linear_issue_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TriageSummary represents a daily triage summary.
type TriageSummary struct {
	ID        int       `json:"id"`
	Date      string    `json:"date"`
	Confirmed int       `json:"confirmed"`
	Rejected  int       `json:"rejected"`
	Pending   int       `json:"pending"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// storeJSON is the on-disk persistence format.
type storeJSON struct {
	Agents        map[string]*AgentStatus `json:"agents"`
	Findings      []Finding              `json:"findings"`
	Triages       []TriageSummary        `json:"triages"`
	NextFindingID int                    `json:"next_finding_id"`
	NextTriageID  int                    `json:"next_triage_id"`
}

// AgentStore is a store for agent statuses, findings, and triage summaries.
// Data is persisted to a JSON file on every mutation.
type AgentStore struct {
	mu             sync.RWMutex
	filePath       string
	agents         map[string]*AgentStatus
	findings        []Finding
	triages         []TriageSummary
	nextFindingID  int
	nextTriageID   int
}

// defaultAgents returns the seeded agent defaults.
func defaultAgents() map[string]*AgentStatus {
	return map[string]*AgentStatus{
		"daeron": {
			AgentID:     "daeron",
			Name:        "Daeron",
			Status:      "idle",
			Model:       "mimo-v2.5-base",
			Description: "Loremaster — triage, verification, monitoring",
			LastRun:     time.Now(),
		},
		"reek": {
			AgentID:     "reek",
			Name:        "Reek",
			Status:      "idle",
			Model:       "deepseek-v4-flash",
			Description: "Code Crawler — nightly code review",
			LastRun:     time.Now(),
		},
	}
}

// NewAgentStore creates an AgentStore, loading from filePath if it exists.
// If the file doesn't exist, a fresh store with seeded defaults is created.
func NewAgentStore(filePath string) *AgentStore {
	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	s := &AgentStore{
		filePath: filePath,
	}

	data, err := os.ReadFile(filePath)
	if err == nil {
		var sj storeJSON
		if json.Unmarshal(data, &sj) == nil {
			s.agents = sj.Agents
			s.findings = sj.Findings
			s.triages = sj.Triages
			s.nextFindingID = sj.NextFindingID
			s.nextTriageID = sj.NextTriageID
			return s
		}
	}

	// Fresh store with seeded defaults
	s.agents = defaultAgents()
	s.nextFindingID = 1
	s.nextTriageID = 1
	return s
}

// Save persists the store to the JSON file atomically.
func (s *AgentStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sj := storeJSON{
		Agents:        s.agents,
		Findings:      s.findings,
		Triages:       s.triages,
		NextFindingID: s.nextFindingID,
		NextTriageID:  s.nextTriageID,
	}

	data, err := json.MarshalIndent(sj, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}

// GetAgents returns all agent statuses.
func (s *AgentStore) GetAgents() []*AgentStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*AgentStatus, 0, len(s.agents))
	for _, a := range s.agents {
		result = append(result, a)
	}
	return result
}

// UpdateAgentStatus updates the status for a given agent.
func (s *AgentStore) UpdateAgentStatus(agentID, status string) (*AgentStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agent, ok := s.agents[agentID]
	if !ok {
		return nil, false
	}
	agent.Status = status
	agent.LastRun = time.Now()
	_ = s.save()
	return agent, true
}

// GetFindings returns findings, optionally filtered.
func (s *AgentStore) GetFindings(status, severity, source string) []Finding {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Finding, 0, len(s.findings))
	for _, f := range s.findings {
		if status != "" && f.Status != status {
			continue
		}
		if severity != "" && f.Severity != severity {
			continue
		}
		if source != "" && f.Source != source {
			continue
		}
		result = append(result, f)
	}
	return result
}

// AddFinding adds a new finding and returns it with its assigned ID.
func (s *AgentStore) AddFinding(source, severity, title, file string, line int, description string) Finding {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	f := Finding{
		ID:          s.nextFindingID,
		Source:      source,
		Severity:    severity,
		Status:      "open",
		Title:       title,
		File:        file,
		Line:        line,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.findings = append(s.findings, f)
	s.nextFindingID++
	_ = s.save()
	return f
}

// UpdateFindingStatus updates the status of a finding by ID.
func (s *AgentStore) UpdateFindingStatus(id int, status string) (*Finding, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.findings {
		if s.findings[i].ID == id {
			s.findings[i].Status = status
			s.findings[i].UpdatedAt = time.Now()
			_ = s.save()
			return &s.findings[i], true
		}
	}
	return nil, false
}

// UpdateFinding updates both the status and linear_issue_id of a finding by ID.
func (s *AgentStore) UpdateFinding(id int, status, linearIssueID string) (*Finding, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.findings {
		if s.findings[i].ID == id {
			if status != "" {
				s.findings[i].Status = status
			}
			if linearIssueID != "" {
				s.findings[i].LinearIssueID = linearIssueID
			}
			s.findings[i].UpdatedAt = time.Now()
			_ = s.save()
			return &s.findings[i], true
		}
	}
	return nil, false
}

// GetTriageSummaries returns all triage summaries.
func (s *AgentStore) GetTriageSummaries() []TriageSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]TriageSummary, len(s.triages))
	copy(result, s.triages)
	return result
}

// AddTriageSummary adds a new triage summary.
func (s *AgentStore) AddTriageSummary(date, summary string, confirmed, rejected, pending int) TriageSummary {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := TriageSummary{
		ID:        s.nextTriageID,
		Date:      date,
		Confirmed: confirmed,
		Rejected:  rejected,
		Pending:   pending,
		Summary:   summary,
		CreatedAt: time.Now(),
	}
	s.triages = append(s.triages, t)
	s.nextTriageID++
	_ = s.save()
	return t
}

// save writes the store to disk. Caller must hold s.mu (at least a write lock).
func (s *AgentStore) save() error {
	sj := storeJSON{
		Agents:        s.agents,
		Findings:      s.findings,
		Triages:       s.triages,
		NextFindingID: s.nextFindingID,
		NextTriageID:  s.nextTriageID,
	}

	data, err := json.MarshalIndent(sj, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.filePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.filePath)
}
