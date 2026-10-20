package portable

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"
)

type supervisionState struct {
	Wanted bool `json:"wanted"`
}

// Intent is changed only by explicit lifecycle commands. Internal update stops
// leave it alone so the monitor cannot undo either a manual stop or an update.
func setSupervisionIntent(instance string, wanted bool) error {
	var state supervisionState
	if err := readJSON(filepath.Join(instance, ".supervision.json"), &state); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return updateJSON(filepath.Join(instance, ".supervision.json"), supervisionState{Wanted: wanted})
}

// Supervise is the stable foreground process for an external service manager.
// The authenticated game supervisor may change during update; this process stays
// alive, respects operator stop/start intent and restarts a crashed game only
// outside a pending update transaction.
func Supervise(ctx context.Context, bundle, instance string, out io.Writer) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	if _, err := Load(instance); err != nil {
		return err
	}
	if installed, err := ServiceInstalled(instance); err != nil {
		return err
	} else if installed {
		return errors.New("remove sign-in startup registration before using system supervision")
	}
	owner := flock.New(filepath.Join(instance, ".supervision.lock"))
	ok, err := owner.TryLock()
	if err != nil || !ok {
		return errors.Join(errors.New("another system supervisor owns this instance"), err, owner.Close())
	}
	defer func() { result = errors.Join(result, owner.Close()) }()
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	installed, installErr := ServiceInstalled(instance)
	if installErr != nil || installed {
		return errors.Join(errors.New("sign-in startup cannot share a system supervisor"), installErr, unlock())
	}
	err = updateJSON(filepath.Join(instance, ".supervision.json"), supervisionState{Wanted: true})
	if err := errors.Join(err, unlock()); err != nil {
		return err
	}
	defer func() {
		shutdown, cancel := context.WithTimeout(context.Background(), 70*time.Second)
		defer cancel()
		for {
			err := Stop(shutdown, instance)
			if !errors.Is(err, errLifecycleBusy) {
				result = errors.Join(result, err)
				return
			}
			select {
			case <-shutdown.Done():
				result = errors.Join(result, shutdown.Err(), err)
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}()
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		if err := ensureSupervised(ctx, bundle, instance); err != nil && !errors.Is(err, errLifecycleBusy) && ctx.Err() == nil {
			if _, err := fmt.Fprintf(out, "System supervision: %v\n", err); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}

func ensureSupervised(ctx context.Context, bundle, instance string) (result error) {
	needed, err := supervisedStartNeeded(ctx, instance)
	if err != nil || !needed {
		return err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	// Recheck under the lock: an operator stop or an update may have won the
	// race. Healthy-world observation never takes the mutation lock.
	needed, err = supervisedStartNeeded(ctx, instance)
	if err != nil || !needed {
		return err
	}
	_, err = start(ctx, bundle, instance)
	return err
}

func supervisedStartNeeded(ctx context.Context, instance string) (bool, error) {
	var state supervisionState
	if err := readJSON(filepath.Join(instance, ".supervision.json"), &state); err != nil {
		return false, err
	}
	if !state.Wanted {
		return false, nil
	}
	if err := pendingUpdate(instance); err != nil {
		return false, nil // the updater/recovery owns the transition, including on failure
	}
	status, err := Status(ctx, instance)
	return status.State == "stopped", err
}

func supervisionActive(instance string) (active bool, result error) {
	path := filepath.Join(instance, ".supervision.lock")
	if info, err := os.Lstat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else if !info.Mode().IsRegular() {
		return false, errors.New("system supervision lock is not a regular file")
	}
	lock := flock.New(path)
	defer func() { result = errors.Join(result, lock.Close()) }()
	ok, err := lock.TryLock()
	return !ok, err
}
