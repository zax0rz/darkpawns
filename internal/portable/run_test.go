package portable

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// PR #2197: the launcher's graceful stop is a SIGINT to the bundled server.
// Before that child reaches readiness it has not installed a signal handler —
// the port calls signal.Notify only after the world is loaded, exactly as C
// installs signal_setup() after boot_db() (src/comm.c:269,277) — so the
// interrupt kills it with the default action and wait(2) reports "signal:
// interrupt". Recording that as a shutdown failure made `darkpawns stop` exit
// non-zero for a stop that actually stopped the instance, and turned the macOS
// portable job red. These tests run a stand-in child that never handles SIGINT,
// which is the state the real child is in for its whole boot.

// freePort returns a loopback port nothing is listening on.
func freePort(t *testing.T) int {
	t.Helper()
	probe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("probe loopback port: %v", err)
	}
	defer probe.Close()
	return probe.Addr().(*net.TCPAddr).Port
}

// interruptedServerFixture builds a signed bundle whose bundled server never
// installs a signal handler, plus an instance on the given web port. The
// publisher gate itself is covered by verify_publisher_test.go; here it is
// pointed at the fixture's disposable key.
func interruptedServerFixture(t *testing.T, port int) (bundle, instance string) {
	t.Helper()
	key := generatePublisherKey(t)
	bundle = writeSignedBundleWithServer(t, t.TempDir(), key, "#!/bin/sh\nexec sleep 300\n")
	previous := PublisherVerifyExecutables
	PublisherVerifyExecutables = func(bundle string, names ...string) error {
		return verifyBundlePublisherWithKey(bundle, key.public, names)
	}
	t.Cleanup(func() { PublisherVerifyExecutables = previous })
	instance = filepath.Join(t.TempDir(), "world")
	if err := InitWithTelnet(bundle, instance, port, 0); err != nil {
		t.Fatalf("initialize instance: %v", err)
	}
	return bundle, instance
}

// runUntilStopped starts run in the background and returns the stop request and
// the channel its result arrives on.
func runUntilStopped(t *testing.T, bundle, instance string) (context.CancelFunc, <-chan error) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- run(ctx, bundle, instance, nil, io.Discard, io.Discard, nil, "") }()
	return cancel, done
}

// awaitResult waits for run to return, with a bound well above the child's
// shutdown timer.
func awaitResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("run did not return after the stop request")
		return nil
	}
}

func waitForPublishedControlRecord(t *testing.T, instance string) {
	t.Helper()
	path := filepath.Join(instance, ".control.json")
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s was never published", path)
}

func waitForInstanceState(t *testing.T, instance, want string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		status, err := Status(context.Background(), instance)
		if err == nil && status.State == want {
			return
		}
		last = err
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("instance never reported %q: %v", want, last)
}

// TestStopOutcomeOnlyAcceptsTheInterruptItAskedFor pins the classifier itself:
// a death from our interrupt is clean only before readiness, and any death the
// launcher did not cause stays a failure.
func TestStopOutcomeOnlyAcceptsTheInterruptItAskedFor(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("interrupt delivery is a POSIX signal; Windows uses a console control event")
	}
	interrupted := interruptedExit(t)
	if err := stopOutcome(false, nil, interrupted); err != nil {
		t.Fatalf("requested interrupt before readiness = %v, want a clean stop", err)
	}
	if err := stopOutcome(true, nil, interrupted); !errors.Is(err, interrupted) {
		t.Fatalf("interrupt after readiness = %v, want the child's own failure", err)
	}
	undelivered := errors.New("interrupt not delivered")
	if err := stopOutcome(false, undelivered, interrupted); !errors.Is(err, interrupted) {
		t.Fatalf("death with no delivered interrupt = %v, want the child's own failure", err)
	}
	if err := stopOutcome(false, nil, errors.New("exit status 2")); err == nil {
		t.Fatal("an ordinary exit failure before readiness was accepted as a clean stop")
	}
}

// interruptedExit runs a child that dies from SIGINT and returns its wait error.
func interruptedExit(t *testing.T) error {
	t.Helper()
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("no POSIX sleep to interrupt: %v", err)
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("interrupt the child: %v", err)
	}
	err := cmd.Wait()
	if err == nil {
		t.Fatal("the interrupted child reported success")
	}
	return err
}

// TestStopRequestedBeforeReadinessIsNotAShutdownFailure is the #2197
// regression: a stop that lands in the child's startup window must be recorded
// as the stop the operator asked for, not as "signal: interrupt".
func TestStopRequestedBeforeReadinessIsNotAShutdownFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("interrupt delivery is a POSIX signal; Windows uses a console control event")
	}
	bundle, instance := interruptedServerFixture(t, freePort(t))
	cancel, done := runUntilStopped(t, bundle, instance)
	waitForPublishedControlRecord(t, instance)
	cancel()
	if err := awaitResult(t, done); err != nil {
		t.Fatalf("stop requested before readiness = %v, want a clean stop", err)
	}
}

// TestSignalDeathAfterReadinessStaysAShutdownFailure keeps the fix honest: once
// the child has answered readiness it holds a signal handler, so a death from
// the same SIGINT means it died without running its shutdown path — lost saves
// — and must still fail.
func TestSignalDeathAfterReadinessStaysAShutdownFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("interrupt delivery is a POSIX signal; Windows uses a console control event")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for the readiness gate: %v", err)
	}
	health := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	health.Listener = listener
	health.Start()
	defer health.Close()
	bundle, instance := interruptedServerFixture(t, listener.Addr().(*net.TCPAddr).Port)
	cancel, done := runUntilStopped(t, bundle, instance)
	waitForInstanceState(t, instance, "running")
	cancel()
	err = awaitResult(t, done)
	if err == nil || !strings.Contains(err.Error(), "signal: interrupt") {
		t.Fatalf("signal death after readiness = %v, want a shutdown failure", err)
	}
}
