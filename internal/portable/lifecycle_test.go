package portable

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

// PR #2197, second half: `service install` registers a launchd RunAtLoad job on
// macOS, and bootstrap starts that job immediately — systemd only enables the
// unit — so the explicit `darkpawns start` that follows can arrive in the
// instant where the job owns the run lock but has not published its operator
// record yet. Start is idempotent, so it waits for that record rather than
// reporting "instance lock is held but its operator endpoint is unavailable"
// for an instance that is simply still starting.

// writeInstanceConfig writes the minimal valid configuration Status needs
// before it can report the startup window.
func writeInstanceConfig(t *testing.T, instance string, port int) {
	t.Helper()
	secret := make([]byte, 32)
	config := Config{Format: Format, HTTPPort: port, JWTSecret: hex.EncodeToString(secret), TelnetBind: "127.0.0.1"}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("encode instance configuration: %v", err)
	}
	if err := os.WriteFile(filepath.Join(instance, "instance.json"), data, 0o600); err != nil {
		t.Fatalf("write instance configuration: %v", err)
	}
}

// holdRunLock takes the instance's run lock the way a supervisor does for the
// instant between acquiring it and publishing its operator record.
func holdRunLock(t *testing.T, instance string) {
	t.Helper()
	lock := flock.New(filepath.Join(instance, ".run.lock"))
	acquired, err := lock.TryLock()
	if err != nil || !acquired {
		t.Fatalf("take the run lock: acquired=%v err=%v", acquired, err)
	}
	t.Cleanup(func() { _ = lock.Close() })
}

// runningEndpoint serves the operator endpoint a started instance publishes.
func runningEndpoint(t *testing.T, instance string) *httptest.Server {
	t.Helper()
	canonical, err := canonicalInstance(instance)
	if err != nil {
		t.Fatalf("canonical instance: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(RuntimeStatus{State: "running", RunID: "fixture", Instance: canonical}); err != nil {
			return
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// publishControlRecord writes the operator record for an endpoint, as the
// supervisor does once it is ready to be commanded.
func publishControlRecord(instance, endpoint string) error {
	record := controlRecord{Address: strings.TrimPrefix(endpoint, "http://"), Token: strings.Repeat("ab", 32)}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return writePrivate(filepath.Join(instance, ".control.json"), data)
}

func TestStartWaitsForAConcurrentSupervisorStartup(t *testing.T) {
	instance := t.TempDir()
	writeInstanceConfig(t, instance, freePort(t))
	holdRunLock(t, instance)
	endpoint := runningEndpoint(t, instance)
	published := make(chan error, 1)
	go func() {
		time.Sleep(250 * time.Millisecond)
		published <- publishControlRecord(instance, endpoint.URL)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	status, err := Start(ctx, "unused-bundle", instance)
	if err != nil {
		t.Fatalf("start during a concurrent supervisor startup = %v, want the published status", err)
	}
	if err := <-published; err != nil {
		t.Fatalf("publish the operator record: %v", err)
	}
	if status.State != "running" {
		t.Fatalf("start reported state %q, want running", status.State)
	}
}

func TestStartStillFailsWhenTheRunLockNeverPublishesAnEndpoint(t *testing.T) {
	defer func(previous time.Duration) { startupWindowWait = previous }(startupWindowWait)
	startupWindowWait = 300 * time.Millisecond
	instance := t.TempDir()
	writeInstanceConfig(t, instance, freePort(t))
	holdRunLock(t, instance)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, err := Start(ctx, "unused-bundle", instance); !errors.Is(err, errStartupWindow) {
		t.Fatalf("start with a run lock that never publishes an endpoint = %v, want the startup-window diagnosis", err)
	}
}

// TestStartRetriesTheReadinessWaitOnce covers the CI flake shape: a
// supervisor that publishes its operator record just after the first
// readiness attempt expires. Start must retry the wait once, succeed, and
// log that the first attempt failed (the recovered path stays visible).
func TestStartRetriesTheReadinessWaitOnce(t *testing.T) {
	defer func(previous time.Duration) { readinessFirstWait = previous }(readinessFirstWait)
	readinessFirstWait = 150 * time.Millisecond
	instance := t.TempDir()
	writeInstanceConfig(t, instance, freePort(t))
	holdRunLock(t, instance)
	endpoint := runningEndpoint(t, instance)

	go func() {
		// Publish after the first attempt has expired but well before the
		// second gives up: the window only the retry covers.
		time.Sleep(500 * time.Millisecond)
		_ = publishControlRecord(instance, endpoint.URL)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	status, err := Start(ctx, "unused-bundle", instance)
	if err != nil {
		t.Fatalf("start with a late publisher = %v, want the retry to succeed", err)
	}
	if status.State != "running" {
		t.Fatalf("start reported state %q, want running", status.State)
	}
}

// TestWaitReadyWithRetryFailsAfterBothAttempts keeps real failures loud: an
// instance that never reaches running must fail, with the error naming both
// attempts and the launcher log to inspect.
func TestWaitReadyWithRetryFailsAfterBothAttempts(t *testing.T) {
	defer func(ready, retry time.Duration) { readinessFirstWait, readinessRetryWait = ready, retry }(readinessFirstWait, readinessRetryWait)
	readinessFirstWait, readinessRetryWait = 100*time.Millisecond, 100*time.Millisecond
	instance := t.TempDir()
	writeInstanceConfig(t, instance, freePort(t))
	// A held run lock with no published record keeps Status in the
	// startup-window error state: never running, without launching anything.
	holdRunLock(t, instance)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := waitReadyWithRetry(ctx, instance)
	if err == nil {
		t.Fatal("readiness retry succeeded against an instance that never runs, want a loud failure")
	}
	if !strings.Contains(err.Error(), "two attempts") {
		t.Fatalf("error = %v, want the two-attempt diagnosis", err)
	}
	if !strings.Contains(err.Error(), "launcher.log") {
		t.Fatalf("error = %v, want the launcher.log pointer", err)
	}
}
