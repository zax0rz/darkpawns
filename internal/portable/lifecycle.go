package portable

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gofrs/flock"
)

// errStartupWindow marks the instant in which a supervisor owns the run lock
// but has not published its operator record yet. It is not a failure by
// itself: Start waits it out (see startStatus).
var errStartupWindow = errors.New("instance lock is held but its operator endpoint is unavailable")

// Status never infers ownership from a PID or an arbitrary healthy game port.
func Status(ctx context.Context, instance string) (RuntimeStatus, error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return RuntimeStatus{}, err
	}
	status, err := controlRequest(ctx, instance, http.MethodGet, "/status")
	if err == nil {
		return status, nil
	}
	if _, loadErr := Load(instance); loadErr != nil {
		return status, loadErr
	}
	running, lockErr := instanceLocked(instance)
	if lockErr != nil {
		return status, lockErr
	}
	if running {
		return status, fmt.Errorf("%w; inspect lib/logs/launcher.log: %w", errStartupWindow, err)
	}
	return RuntimeStatus{State: "stopped", Instance: instance}, nil
}

// startupWindowWait bounds how long Start waits for a supervisor that is still
// starting to publish its operator record. A variable so tests need not wait
// the full window out.
var startupWindowWait = 5 * time.Second

// The readiness budget keeps the launcher's historical 40-second WORST CASE:
// a first attempt of readinessFirstWait, then diagnostics, then one retry of
// readinessRetryWait. Callers time `start` against that budget (the smokes
// cap the subprocess at 60s and one of them asserts a REFUSED start returns
// within it), so the retry must split the budget, not double it: an instance
// that never becomes ready still fails by the same wall-clock bound as
// before. A slow-but-successful supervisor is covered either way — the retry
// polls the same live status, so a record published at any point inside the
// budget is seen immediately.
var (
	readinessFirstWait = 30 * time.Second
	readinessRetryWait = 10 * time.Second
)

// readinessDiagnosticLines is how many trailing lines of the launcher log the
// retry diagnostics capture. A variable so tests can shrink the capture.
var readinessDiagnosticLines = 20

// startStatus reads the instance state, tolerating a supervisor that is still
// starting. Registration can begin the instance itself — launchd's RunAtLoad
// job boots it the moment `service install` bootstraps it, where systemd only
// enables the unit — so for the instant between taking the run lock and
// publishing the operator record the lock is held with no endpoint to ask.
// Start is idempotent, so it waits for that record instead of reporting a
// spurious failure (PR #2197). A lock that never publishes a record still fails
// with Status's own diagnosis once the window expires.
func startStatus(ctx context.Context, instance string) (RuntimeStatus, error) {
	status, err := Status(ctx, instance)
	if err == nil || !errors.Is(err, errStartupWindow) {
		return status, err
	}
	wait, cancel := context.WithTimeout(ctx, startupWindowWait)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-wait.Done():
			return status, err
		case <-tick.C:
			status, err = Status(ctx, instance)
			if err == nil || !errors.Is(err, errStartupWindow) {
				return status, err
			}
		}
	}
}

func instanceLocked(instance string) (locked bool, result error) {
	lock := flock.New(filepath.Join(instance, ".run.lock"))
	defer func() { result = errors.Join(result, lock.Close()) }()
	acquired, err := lock.TryLock()
	return !acquired, err
}

var errLifecycleBusy = errors.New("another lifecycle operation is in progress; retry after it finishes")

func lifecycleLock(instance string) (func() error, error) {
	lock := flock.New(filepath.Join(instance, ".lifecycle.lock"))
	acquired, err := lock.TryLock()
	if err != nil || !acquired {
		return nil, errors.Join(errLifecycleBusy, err, lock.Close())
	}
	return lock.Close, nil
}

func Start(ctx context.Context, bundle, instance string) (status RuntimeStatus, result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return status, err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return status, err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	if err := setSupervisionIntent(instance, true); err != nil {
		return status, err
	}
	return start(ctx, bundle, instance)
}

func start(ctx context.Context, bundle, instance string) (RuntimeStatus, error) {
	if err := pendingUpdate(instance); err != nil {
		return RuntimeStatus{}, err
	}
	var resolveErr error
	bundle, resolveErr = ActiveBundle(bundle, instance)
	if resolveErr != nil {
		return RuntimeStatus{}, resolveErr
	}
	status, err := startStatus(ctx, instance)
	if err != nil {
		return status, err
	}
	if status.State == "running" {
		return status, nil
	}
	if status.State != "stopped" && status.State != "starting" {
		return status, fmt.Errorf("instance is %s; wait before starting it", status.State)
	}
	if status.State == "stopped" {
		cfg, err := Load(instance)
		if err != nil {
			return status, err
		}
		// Refuse an occupied port before opening any game database. This also avoids
		// launching over an orphaned game after an abrupt supervisor failure.
		probe, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", cfg.HTTPPort))
		if err != nil {
			return status, fmt.Errorf("web port is occupied; diagnose with %s before starting: %w", OperatorCommand(instance, "doctor"), err)
		}
		if err := probe.Close(); err != nil {
			return status, err
		}
		manifest, err := ReadManifest(bundle)
		if err != nil {
			return status, err
		}
		if err := Verify(bundle, manifest); err != nil {
			return status, err
		}
		if installed, err := ServiceInstalled(instance); err != nil {
			return status, err
		} else if installed {
			if err := startService(ctx, instance); err != nil {
				return status, err
			}
		} else {
			launcher, err := os.Executable()
			if err != nil {
				return status, err
			}
			launcher, err = filepath.EvalSymlinks(launcher)
			if err != nil {
				return status, err
			}
			if err := spawnSupervisor(launcher, instance); err != nil {
				return status, err
			}
		}
	}
	return waitReadyWithRetry(ctx, instance)
}

// waitReadyWithRetry runs the readiness wait with exactly one retry. CI
// runners have shown transient first-attempt failures (a supervisor that
// publishes its record late on loaded macos/windows runners, a startup-task
// query that exits 1 once). Before retrying, the failure is logged with the
// evidence a diagnosis needs — process state, the launcher log tail, and the
// automatic-start task's query output — so a real failure never passes
// silently: the retry only re-runs the WAIT, and a second failure returns an
// error carrying both attempts' terminal state.
func waitReadyWithRetry(ctx context.Context, instance string) (RuntimeStatus, error) {
	status, waitErr := waitReady(ctx, instance, readinessFirstWait)
	if waitErr == nil {
		return status, nil
	}
	firstStatus, firstErr := status, waitErr
	logReadinessFailure(instance, "first readiness attempt failed", firstStatus, firstErr)

	status, waitErr = waitReady(ctx, instance, readinessRetryWait)
	if waitErr == nil {
		slog.Warn("background start became ready on the retry; the first attempt failed",
			"instance", instance,
			"first_state", firstStatus.State,
			"first_error", firstErr.Error())
		return status, nil
	}
	return status, fmt.Errorf("background start did not become ready after two attempts (first: %v; second: %w); inspect %s",
		firstErr, waitErr, filepath.Join(instance, "lib", "logs", "launcher.log"))
}

// waitReady polls the instance status until it reports running.
func waitReady(ctx context.Context, instance string, wait time.Duration) (RuntimeStatus, error) {
	timeout, cancel := context.WithTimeout(ctx, wait)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	var status RuntimeStatus
	var err error
	for {
		status, err = Status(timeout, instance)
		if err == nil && status.State == "running" {
			return status, nil
		}
		select {
		case <-timeout.Done():
			return status, fmt.Errorf("did not become ready within %s: %w", wait, timeout.Err())
		case <-tick.C:
		}
	}
}

// logReadinessFailure records why a readiness attempt failed: the process
// state Status reports, the tail of the launcher log, and the automatic-start
// task's query output where the platform has one. Evidence only — the caller
// decides whether to retry.
func logReadinessFailure(instance, reason string, status RuntimeStatus, waitErr error) {
	attrs := []any{
		"instance", instance,
		"reason", reason,
		"state", status.State,
		"wait_error", waitErr.Error(),
	}
	if locked, lockErr := instanceLocked(instance); lockErr != nil {
		attrs = append(attrs, "run_lock", fmt.Sprintf("error: %v", lockErr))
	} else {
		attrs = append(attrs, "run_lock", locked)
	}
	if tail, err := tailLauncherLog(instance); err != nil {
		attrs = append(attrs, "launcher_log_tail", fmt.Sprintf("unavailable: %v", err))
	} else {
		attrs = append(attrs, "launcher_log_tail", tail)
	}
	attrs = append(attrs, "startup_task_query", startupTaskQueryOutput(instance))
	slog.Warn("readiness attempt failed", attrs...)
}

// tailLauncherLog returns the last readinessDiagnosticLines lines of the
// instance's launcher log, the file Start's own errors point at.
func tailLauncherLog(instance string) (string, error) {
	path := filepath.Join(instance, "lib", "logs", "launcher.log")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > readinessDiagnosticLines {
		lines = lines[len(lines)-readinessDiagnosticLines:]
	}
	return strings.Join(lines, "\n"), nil
}

// startupTaskQueryOutput runs the platform's automatic-start task query and
// returns its output, for diagnostics only: failures are reported, never
// acted on.
func startupTaskQueryOutput(instance string) string {
	record, err := readService(instance)
	if err != nil {
		if os.IsNotExist(err) {
			return "no automatic-start record"
		}
		return fmt.Sprintf("record unavailable: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	switch runtime.GOOS {
	case "windows":
		out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive",
			"-Command", "[Console]::OutputEncoding=[System.Text.UTF8Encoding]::new($false); $s=New-Object -ComObject Schedule.Service; $s.Connect(); $t=$s.GetFolder('\\').GetTask('"+record.Name+"'); $t | Select-Object State,LastRunTime,LastTaskResult | ConvertTo-Json -Compress").CombinedOutput()
		return describeQueryOutput(out, err)
	case "darwin":
		out, err := exec.CommandContext(ctx, "launchctl", "print", "gui/"+fmt.Sprint(os.Getuid())+"/"+record.Name).CombinedOutput()
		return describeQueryOutput(out, err)
	case "linux":
		out, err := exec.CommandContext(ctx, "systemctl", "--user", "status", record.Name+".service", "--no-pager", "--lines=20").CombinedOutput()
		return describeQueryOutput(out, err)
	default:
		return "no query on " + runtime.GOOS
	}
}

func describeQueryOutput(out []byte, err error) string {
	text := strings.TrimSpace(string(out))
	if len(text) > 2048 {
		text = text[len(text)-2048:]
	}
	if err != nil {
		return fmt.Sprintf("error: %v; output: %s", err, text)
	}
	return text
}

func spawnSupervisor(launcher, instance string, arguments ...string) (result error) {
	log, err := os.OpenFile(filepath.Join(instance, "lib", "logs", "launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Close()) }()
	if len(arguments) == 0 {
		arguments = []string{"_serve"}
	}
	arguments = append(arguments, "--instance", instance)
	cmd := exec.Command(launcher, arguments...)
	cmd.Dir = instance
	cmd.Stdout, cmd.Stderr = log, log
	detachProcess(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func Stop(ctx context.Context, instance string) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	if err := setSupervisionIntent(instance, false); err != nil {
		return err
	}
	return stopInstance(ctx, instance)
}

func stopInstance(ctx context.Context, instance string) error {
	status, err := Status(ctx, instance)
	if err != nil {
		return err
	}
	if status.State == "stopped" {
		return nil
	}
	if err := func() error { _, err := controlRequest(ctx, instance, http.MethodPost, "/stop"); return err }(); err != nil {
		return fmt.Errorf("request graceful stop: %w", err)
	}
	timeout, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		locked, err := instanceLocked(instance)
		if err != nil {
			return err
		}
		if !locked {
			return verifyExit(instance, status.RunID)
		}
		select {
		case <-timeout.Done():
			return fmt.Errorf("shutdown is still in progress; no process was force-killed: %w", timeout.Err())
		case <-tick.C:
		}
	}
}

func Restart(ctx context.Context, bundle, instance string) (status RuntimeStatus, result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return status, err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return status, err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	if err := setSupervisionIntent(instance, true); err != nil {
		return status, err
	}
	if err := stopInstance(ctx, instance); err != nil {
		return status, err
	}
	return start(ctx, bundle, instance)
}

// ServeBackground is entered only by the launcher or the registered user job.
func ServeBackground(ctx context.Context, bundle, instance string) error {
	return ServeUpdateBackground(ctx, bundle, instance, "")
}

func ServeUpdateBackground(ctx context.Context, bundle, instance, transaction string) error {
	if err := backgroundConsole(); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(instance, "lib", "logs", "launcher.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer log.Close()
	err = run(ctx, bundle, instance, nil, log, log, nil, transaction)
	if err != nil {
		if _, writeErr := fmt.Fprintln(log, err); writeErr != nil {
			return errors.Join(err, writeErr)
		}
	}
	return err
}
