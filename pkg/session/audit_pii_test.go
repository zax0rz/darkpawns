package session

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/audit"
)

type auditRecord struct {
	Action  string `json:"action"`
	User    string `json:"user"`
	Details string `json:"details"`
}

func readAuditFile(t *testing.T, path string) (string, []auditRecord) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	var records []auditRecord
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var rec auditRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("unmarshal audit line %q: %v", line, err)
		}
		records = append(records, rec)
	}
	return string(data), records
}

// Players often type a password at the name prompt, so rejected name input is
// a secret in plain text. None of the pre-validation audit sites may ever
// write it: invalid names log no user (length only), and the rate-limit and
// lockout sites log the name only when it passed the entry gate.
func TestAuditNeverRecordsRejectedNameInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	if err := audit.Init(path); err != nil {
		t.Fatalf("audit.Init: %v", err)
	}
	m := makeTestManager(t)

	newSession := func(remote string) *Session {
		s := makeCharSession(t, m)
		s.request = &http.Request{RemoteAddr: remote}
		return s
	}

	// invalid_player_name: name fails the entry gate (digit + punctuation).
	s1 := newSession("10.9.9.1:1000")
	_, _ = callHandleLogin(s1, loginMsg("sup3rsecr3tpw!", "x"))

	// rate_limit_exceeded: drain that IP's token bucket first.
	s2 := newSession("10.9.9.2:1000")
	for i := 0; i < 20; i++ {
		_ = m.loginLimiter.GetLimiter("10.9.9.2").Allow()
	}
	_, _ = callHandleLogin(s2, loginMsg("passw0rd!hunter2", "x"))

	// login_locked_out: lock that IP out first.
	s3 := newSession("10.9.9.3:1000")
	for i := 0; i < 12; i++ {
		m.loginAttempts.RecordFailure("10.9.9.3")
	}
	_, _ = callHandleLogin(s3, loginMsg("an0thersecretpw", "x"))

	raw, records := readAuditFile(t, path)
	for _, secret := range []string{"sup3rsecr3tpw", "passw0rd", "hunter2", "an0thersecretpw"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("audit file contains rejected input %q:\n%s", secret, raw)
		}
	}

	seen := map[string]string{} // action -> user
	for _, rec := range records {
		seen[rec.Action] = rec.User
	}
	for _, action := range []string{"invalid_player_name", "rate_limit_exceeded", "login_locked_out"} {
		user, ok := seen[action]
		if !ok {
			t.Fatalf("missing %s event; records = %+v", action, records)
		}
		if user != "" {
			t.Fatalf("%s event recorded user %q; rejected input must log no user", action, user)
		}
	}
}

// A name that passes the entry gate is logged normally — the guard must not
// silence legitimate lockout attribution.
func TestAuditRecordsGatePassedName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	if err := audit.Init(path); err != nil {
		t.Fatalf("audit.Init: %v", err)
	}
	m := makeTestManager(t)

	s := makeCharSession(t, m)
	s.request = &http.Request{RemoteAddr: "10.9.9.4:1000"}
	for i := 0; i < 20; i++ {
		_ = m.loginLimiter.GetLimiter("10.9.9.4").Allow()
	}
	_, _ = callHandleLogin(s, loginMsg("Realtypone", "x"))

	_, records := readAuditFile(t, path)
	found := false
	for _, rec := range records {
		if rec.Action == "rate_limit_exceeded" {
			found = true
			if rec.User != "Realtypone" {
				t.Fatalf("rate_limit_exceeded user = %q, want the gate-passed name", rec.User)
			}
		}
	}
	if !found {
		t.Fatalf("missing rate_limit_exceeded event; records = %+v", records)
	}
}
