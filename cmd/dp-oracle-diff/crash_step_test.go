package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// TestProcessKillIsImmediateSIGKILL pins the <CRASH> step's signal contract
// at the process layer: kill() terminates a process that ignores SIGINT,
// which only an immediate SIGKILL can do, and it returns only after the exit
// is observed. If kill() ever degrades to stop()'s SIGINT-first escalation,
// the SIGINT-ignoring child survives and this test fails on the timeout.
//
// The child is this test binary re-executed (the classic re-exec pattern):
// it ignores SIGINT and sleeps, holds the harness output pipe itself, and
// spawns nothing — a shell with a `sleep` grandchild would keep the pipe
// open after the kill and stall cmd.Wait for the grandchild's lifetime.
func TestProcessKillIsImmediateSIGKILL(t *testing.T) {
	if os.Getenv("DP_KILLTEST_CHILD") == "1" {
		signal.Ignore(syscall.SIGINT)
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "DP_KILLTEST_CHILD=1")
	p, err := startProcess(context.Background(), "sigint-proof", t.TempDir(), env,
		exe, "-test.run=TestProcessKillIsImmediateSIGKILL", "-test.count=1")
	if err != nil {
		t.Fatal(err)
	}
	// Give the child a moment to install its SIGINT handler.
	time.Sleep(time.Second)

	done := make(chan struct{})
	go func() {
		p.kill()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("kill() did not terminate a SIGINT-ignoring process; the crash step no longer SIGKILLs immediately")
	}
	select {
	case <-p.done:
	default:
		t.Fatal("kill() returned before the process exit was observed")
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
