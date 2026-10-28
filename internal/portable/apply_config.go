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
// committed, rolled-back.
type ConfigApplyReceipt struct {
	State          string `json:"state"`
	StartedAt      string `json:"started_at"`
	WasRunning     bool   `json:"was_running"`
	Error          string `json:"error,omitempty"`
	FailedListener string `json:"failed_listener,omitempty"`
}

const configApplyFile = ".config-apply.json"

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
// source of truth the console polls across the restart.
func ApplyNetworkConfig(ctx context.Context, instance string) (result error) {
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
		restored, restoreErr := restoreRollbackConfig(instance)
		if restoreErr != nil {
			receipt.State = "rolled-back"
			receipt.Error = fmt.Sprintf("%s; restoring the previous configuration failed: %v (the snapshot is retained at %s)", message, restoreErr, configRollbackFile)
			return writeApplyReceipt(instance, receipt)
		}
		if !restored {
			// Nothing was ever proposed: there is no previous
			// configuration to fall back to, so report and stop.
			receipt.State = "rolled-back"
			receipt.Error = fmt.Sprintf("%s; no previous configuration snapshot exists to restore", message)
			return writeApplyReceipt(instance, receipt)
		}
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
	return writeApplyReceipt(instance, receipt)
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
