package portable

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

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
		return status, fmt.Errorf("instance lock is held but its operator endpoint is unavailable; inspect lib/logs/launcher.log: %w", err)
	}
	return RuntimeStatus{State: "stopped", Instance: instance}, nil
}

func instanceLocked(instance string) (locked bool, result error) {
	lock := flock.New(filepath.Join(instance, ".run.lock"))
	defer func() { result = errors.Join(result, lock.Close()) }()
	acquired, err := lock.TryLock()
	return !acquired, err
}

func lifecycleLock(instance string) (func() error, error) {
	lock := flock.New(filepath.Join(instance, ".lifecycle.lock"))
	acquired, err := lock.TryLock()
	if err != nil || !acquired {
		return nil, errors.Join(errors.New("another lifecycle operation is in progress; retry after it finishes"), err, lock.Close())
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
	return start(ctx, bundle, instance)
}

func start(ctx context.Context, bundle, instance string) (RuntimeStatus, error) {
	status, err := Status(ctx, instance)
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
	timeout, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		status, err = Status(timeout, instance)
		if err == nil && status.State == "running" {
			return status, nil
		}
		select {
		case <-timeout.Done():
			return status, fmt.Errorf("background start did not become ready; inspect %s: %w", filepath.Join(instance, "lib", "logs", "launcher.log"), timeout.Err())
		case <-tick.C:
		}
	}
}

func spawnSupervisor(launcher, instance string) (result error) {
	log, err := os.OpenFile(filepath.Join(instance, "lib", "logs", "launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, log.Close()) }()
	cmd := exec.Command(launcher, "_serve", "--instance", instance)
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
			return nil
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
	if err := stopInstance(ctx, instance); err != nil {
		return status, err
	}
	return start(ctx, bundle, instance)
}

// ServeBackground is entered only by the launcher or the registered user job.
func ServeBackground(ctx context.Context, bundle, instance string) error {
	if err := backgroundConsole(); err != nil {
		return err
	}
	log, err := os.OpenFile(filepath.Join(instance, "lib", "logs", "launcher.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer log.Close()
	err = Run(ctx, bundle, instance, nil, log, log)
	if err != nil {
		if _, writeErr := fmt.Fprintln(log, err); writeErr != nil {
			return errors.Join(err, writeErr)
		}
	}
	return err
}
