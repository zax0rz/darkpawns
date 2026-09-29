package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// crashTestChild is the re-exec child of TestProcessKillIsImmediateSIGKILL:
// it announces readiness through a marker file, records every SIGINT it
// receives in another marker file (so the parent can prove none was sent),
// and sleeps. It spawns nothing — a shell with a `sleep` grandchild would
// hold the harness output pipe open after the kill and stall cmd.Wait for
// the grandchild's lifetime.
func crashTestChild(t *testing.T) {
	dir := os.Getenv("DP_KILLTEST_DIR")
	// Install the SIGINT recorder BEFORE announcing readiness: a parent that
	// sees the marker and signals immediately must hit a registered handler,
	// or a SIGINT-first mutation could kill the child unrecorded and pass
	// the test on scheduling luck. The channel is buffered, so a signal
	// arriving between Notify and the recorder goroutine's first drain is
	// still captured and written.
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGINT)
	go func() {
		for range sigs {
			f, err := os.OpenFile(filepath.Join(dir, "sigint"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				continue
			}
			_, _ = f.WriteString("sigint\n")
			_ = f.Close()
		}
	}()
	if err := os.WriteFile(filepath.Join(dir, "ready"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Minute)
	os.Exit(0)
}

// waitForFile polls for a file's existence, the explicit readiness sync for
// the re-exec child.
func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s never appeared within %s", filepath.Base(path), timeout)
}

// TestProcessKillIsImmediateSIGKILL pins the <CRASH> step's signal contract
// at the process layer:
//
//   - kill() sends no SIGINT at all — the child records every SIGINT it
//     gets, and none may arrive. stop()'s SIGINT-first escalation (even one
//     that still ends in SIGKILL within the timeout) therefore fails this
//     test, not only a pure-SIGINT mutation;
//   - kill() terminates the process and returns only after the exit is
//     observed;
//   - the child's readiness is synchronized through a marker file, not a
//     sleep, so the kill cannot race the handler installation.
func TestProcessKillIsImmediateSIGKILL(t *testing.T) {
	if os.Getenv("DP_KILLTEST_CHILD") == "1" {
		crashTestChild(t)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	env := append(os.Environ(),
		"DP_KILLTEST_CHILD=1",
		"DP_KILLTEST_DIR="+dir)
	p, err := startProcess(context.Background(), "sigint-proof", dir, env,
		exe, "-test.run=TestProcessKillIsImmediateSIGKILL", "-test.count=1", "-test.v")
	if err != nil {
		t.Fatal(err)
	}
	waitForFile(t, filepath.Join(dir, "ready"), 10*time.Second)

	done := make(chan struct{})
	go func() {
		p.kill()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("kill() did not terminate the process within the timeout")
	}
	select {
	case <-p.done:
	default:
		t.Fatal("kill() returned before the process exit was observed")
	}
	if _, err := os.Stat(filepath.Join(dir, "sigint")); err == nil {
		t.Fatal("the child received a SIGINT before it died; kill() no longer crashes without signalling first (stop()'s SIGINT-first escalation is not the crash contract)")
	}
}

// TestEngineCrashKillClearsProcessForRestart: crashKill waits for the killed
// process's exit and clears the engine's handle, so the restart that follows
// is a genuine fresh start on the same start function (same disposable data
// directory and ports in production).
func TestEngineCrashKillClearsProcessForRestart(t *testing.T) {
	starts := 0
	e := &engine{name: "crash-test", start: func() (*process, error) {
		starts++
		return startProcess(context.Background(), "crash-test-proc", t.TempDir(), os.Environ(), "sleep", "60")
	}}
	t.Cleanup(e.stop)
	if _, err := e.ensure(); err != nil {
		t.Fatal(err)
	}
	first := e.proc

	e.crashKill()

	select {
	case <-first.done:
	default:
		t.Fatal("crashKill returned before the killed process exited")
	}
	if e.proc != nil {
		t.Fatal("crashKill did not clear the engine's process handle")
	}
	if err := e.restartAfterCrash(); err != nil {
		t.Fatal(err)
	}
	if starts != 2 || e.proc == first {
		t.Fatal("restartAfterCrash did not start a fresh process")
	}
}

// TestCrashRestartGatesOnFreshReadiness pins the production composition the
// crash wiring uses: the restart callback starts the engine and then blocks
// on the fresh process's readiness marker before returning, and the await
// observes the fresh process, not the killed one. A delayed boot (the
// marker arrives well after the process starts) fails any composition that
// skips or races the wait.
func TestCrashRestartGatesOnFreshReadiness(t *testing.T) {
	dir := t.TempDir()
	starts := 0
	e := &engine{name: "crash-ready-test", start: func() (*process, error) {
		starts++
		// Boot 2: the marker only appears well after the process is alive.
		delay := "1"
		if starts == 1 {
			delay = "0"
		}
		return startProcess(context.Background(), "slow-boot", dir, os.Environ(),
			"/bin/sh", "-c", "sleep "+delay+"; echo crash-ready")
	}}
	t.Cleanup(e.stop)
	if _, err := e.ensure(); err != nil {
		t.Fatal(err)
	}
	first := e.proc

	await := func() error {
		if e.proc == first {
			return errAwaitSawKilledProcess
		}
		// The same wait the production closures use: poll the fresh
		// process's output for its marker.
		return waitForLog(e.proc, "crash-ready", 10*time.Second)
	}
	kill, restart := crashRestart(e, await)
	if err := kill(); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	if err := restart(); err != nil {
		t.Fatal(err)
	}
	if starts != 2 {
		t.Fatalf("starts = %d, want a fresh second start", starts)
	}
	// The second boot delays its marker by a second; a restart that skipped
	// the readiness wait would return in milliseconds.
	if elapsed := time.Since(started); elapsed < 900*time.Millisecond {
		t.Fatalf("restart returned in %s without waiting for the fresh readiness marker", elapsed)
	}
}

var errAwaitSawKilledProcess = &staticError{"await observed the killed process, not the fresh one"}

type staticError struct{ msg string }

func (e *staticError) Error() string { return e.msg }
