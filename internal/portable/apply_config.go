package portable

// The network apply transaction (docs/design/admin-network-settings.md):
// stop the world gracefully, start it on the proposed configuration, then
// prove every requested listener actually bound — HTTP health alone is not
// enough, because the game can serve HTTP while a telnet listener failed.
// If any listener fails, restore the previous configuration and start that
// instead. The transaction runs in its own detached launcher process
// (`_apply-config`) spawned by the control server, exactly like `restart`,
// so the dying game process never drives its own restart.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ConfigApplyReceipt is the durable transaction record the console polls.
// States: applying (stop and start), verifying (dialing listeners),
// committed, rolled-back, refused (the configuration changed after the
// request, so nothing was restarted).
type ConfigApplyReceipt struct {
	State          string `json:"state"`
	StartedAt      string `json:"started_at"`
	WasRunning     bool   `json:"was_running"`
	Error          string `json:"error,omitempty"`
	FailedListener string `json:"failed_listener,omitempty"`
}

const (
	configApplyFile     = ".config-apply.json"
	applyRollbackFile   = ".config-apply-rollback.json"
	applyRefusedMessage = "the configuration changed after this apply was requested; nothing was restarted; reload and apply again"
)

func writeApplyReceipt(instance string, receipt ConfigApplyReceipt) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return writePrivate(filepath.Join(instance, configApplyFile), append(data, '\n'))
}

// ReadConfigApplyStatus returns the last transaction receipt, or a zero
// receipt (state "idle") when no transaction ever ran.
func ReadConfigApplyStatus(instance string) (ConfigApplyReceipt, error) {
	canonical, err := canonicalInstance(instance)
	if err != nil {
		return ConfigApplyReceipt{}, err
	}
	var receipt ConfigApplyReceipt
	if err := readJSON(filepath.Join(canonical, configApplyFile), &receipt); err != nil {
		if os.IsNotExist(err) {
			return ConfigApplyReceipt{State: "idle"}, nil
		}
		return ConfigApplyReceipt{}, err
	}
	return receipt, nil
}

// ApplyNetworkConfig runs the whole transaction. It is the body of the
// hidden `_apply-config` command: the console's scoped control request
// spawns this as a detached process, and the receipt file is the single
// source of truth the console polls across the restart. The revision is the
// one the operator reviewed: the transaction refuses to run on a
// configuration that changed underneath it, so a save by anyone else is
// never silently discarded by a failed apply.
func ApplyNetworkConfig(ctx context.Context, instance string, revision int64) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	if err := pendingUpdate(instance); err != nil {
		return err
	}
	// Authoritative revision check, under the lifecycle lock: the console's
	// control server checked too, but a save can land in between. A refusal
	// leaves the world exactly as it is and says so in the receipt.
	if cfg, err := Load(instance); err != nil {
		return err
	} else if cfg.ConfigRevision != revision {
		receipt := ConfigApplyReceipt{State: "refused", StartedAt: time.Now().UTC().Format(time.RFC3339), Error: applyRefusedMessage}
		if err := writeApplyReceipt(instance, receipt); err != nil {
			return err
		}
		return ErrConfigConflict
	}
	// Freeze the rollback target this transaction owns: the configuration
	// that was live before the reviewed proposal was saved. A later save
	// rewrites the save-time snapshot; this private copy does not move, so
	// the rollback restores exactly what the operator saw before the apply.
	if err := takeApplySnapshot(instance); err != nil {
		return err
	}
	// The bundle is this very launcher's release directory: the control
	// server spawned this process from the running binary, and an empty
	// fallback would resolve release assets against the working directory
	// instead of the bundle.
	self, selfErr := os.Executable()
	if selfErr != nil {
		return selfErr
	}
	bundle := filepath.Dir(self)
	status, err := Status(ctx, instance)
	if err != nil {
		return err
	}
	running := status.State == "running"
	receipt := ConfigApplyReceipt{State: "applying", StartedAt: time.Now().UTC().Format(time.RFC3339), WasRunning: running}
	if err := writeApplyReceipt(instance, receipt); err != nil {
		return err
	}
	fail := func(listener, message string) error {
		receipt.State = "rolling-back"
		receipt.FailedListener = listener
		receipt.Error = message
		if err := writeApplyReceipt(instance, receipt); err != nil {
			return err
		}
		// A save that landed mid-transaction is about to be replaced by the
		// rollback; the receipt must say so instead of discarding it silently.
		if current, err := Load(instance); err == nil && current.ConfigRevision > revision {
			message = fmt.Sprintf("%s; a newer saved change (revision %d) was replaced by this rollback", message, current.ConfigRevision)
			receipt.Error = message
			if err := writeApplyReceipt(instance, receipt); err != nil {
				return err
			}
		}
		restored, restoreErr := restoreApplyRollback(instance)
		if restoreErr != nil {
			receipt.State = "rolled-back"
			receipt.Error = fmt.Sprintf("%s; restoring the previous configuration failed: %v (the snapshot is retained at %s)", message, restoreErr, applyRollbackFile)
			return writeApplyReceipt(instance, receipt)
		}
		if !restored {
			// Nothing was ever proposed: there is no previous
			// configuration to fall back to, so report and stop.
			receipt.State = "rolled-back"
			receipt.Error = fmt.Sprintf("%s; no previous configuration snapshot exists to restore", message)
			return writeApplyReceipt(instance, receipt)
		}
		// The rollback target is installed; its private copy is spent.
		discardApplySnapshot(instance)
		if running {
			if err := stopInstance(ctx, instance); err != nil {
				return errors.Join(fmt.Errorf("%s; stopping the failed world: %w", message, err))
			}
		}
		if _, err := start(ctx, bundle, instance); err != nil {
			receipt.State = "rolled-back"
			receipt.Error = fmt.Sprintf("%s; the previous configuration also failed to start: %v; recover with `darkpawns doctor`", message, err)
			return writeApplyReceipt(instance, receipt)
		}
		receipt.State = "rolled-back"
		return writeApplyReceipt(instance, receipt)
	}

	// Stop the world if it is running; a stopped world just starts on the
	// new configuration.
	if running {
		if err := stopInstance(ctx, instance); err != nil {
			return fmt.Errorf("stop the world before applying network changes: %w", err)
		}
	}
	if _, err := start(ctx, bundle, instance); err != nil {
		// HTTP readiness is the floor: if even that failed, some listener
		// or the server itself refused the configuration.
		return fail("server", fmt.Sprintf("the world did not come back with the new settings: %v", err))
	}

	receipt.State = "verifying"
	if err := writeApplyReceipt(instance, receipt); err != nil {
		return err
	}
	cfg, err := Load(instance)
	if err != nil {
		return err
	}
	for _, target := range listenerTargets(cfg) {
		if err := verifyListener(ctx, target.Address); err != nil {
			return fail(target.Name, fmt.Sprintf("the %s listener did not answer on %s: %v; the previous settings were restored", target.Name, target.Address, err))
		}
	}
	receipt.State = "committed"
	receipt.Error = ""
	if err := writeApplyReceipt(instance, receipt); err != nil {
		return err
	}
	discardApplySnapshot(instance)
	return nil
}

// verifyListener dials until the listener answers or the budget runs out.
// A TCP connect proves the bind; no protocol exchange is needed, and a
// public bind answers on loopback too, which is where this transaction
// always looks.
func verifyListener(ctx context.Context, address string) error {
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		dialer := net.Dialer{Timeout: time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err == nil {
			return conn.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("no answer within 15s")
}

// discardApplySnapshot drops the transaction's private rollback copy once
// its job is done: after a commit, or after a rollback installed it. A copy
// that could not be installed is left in place for manual recovery.
func discardApplySnapshot(instance string) {
	_ = os.Remove(filepath.Join(instance, applyRollbackFile))
}

// takeApplySnapshot copies the save-time rollback snapshot into a file this
// transaction owns. The save-time snapshot is rewritten by every later save;
// this copy holds still, so a rollback restores the configuration that was
// live before the reviewed proposal no matter what was saved since. A
// missing save-time snapshot means nothing was ever proposed through the
// console, and the transaction simply has no rollback target.
func takeApplySnapshot(instance string) error {
	previous, err := os.ReadFile(filepath.Join(instance, configRollbackFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return writePrivate(filepath.Join(instance, applyRollbackFile), previous)
}

// restoreApplyRollback puts the transaction's snapshot back as the live
// configuration, byte-identical to what ran before the failed change. A
// missing snapshot means nothing was ever proposed; the caller treats that
// as "nothing to restore".
func restoreApplyRollback(instance string) (restored bool, err error) {
	path := filepath.Join(instance, applyRollbackFile)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// Prove the snapshot is a configuration before installing it.
	var check Config
	if err := json.Unmarshal(data, &check); err != nil || check.Format != Format {
		return false, errors.New("rollback snapshot is not a valid configuration; it is retained for manual recovery")
	}
	if err := updateBytes(filepath.Join(instance, "instance.json"), data); err != nil {
		return false, err
	}
	return true, nil
}
