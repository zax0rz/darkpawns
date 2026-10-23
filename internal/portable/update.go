package portable

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type (
	UpdateOptions struct{ Archive, SHA256 string }
	// UpdatePlan is staged and verified without stopping or changing the active world.
	UpdatePlan struct {
		Version                                 string
		PreviousVersion                         string
		SHA256                                  string
		UpToDate                                bool
		retained                                bool
		instance, work, candidate, previous, id string
	}
)

func PrepareUpdate(ctx context.Context, bundle, instance string, options UpdateOptions) (plan *UpdatePlan, result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return nil, err
	}
	if _, err := Load(instance); err != nil {
		return nil, err
	}
	if err := pendingUpdate(instance); err != nil {
		return nil, err
	}
	bundle, err = ActiveBundle(bundle, instance)
	if err != nil {
		return nil, err
	}
	bundle, err = canonicalInstance(bundle)
	if err != nil {
		return nil, err
	}
	current, err := ReadManifest(bundle)
	if err != nil {
		return nil, err
	}
	// The first updater can adopt slice1-3 data (all instance format1). Every new
	// target must explicitly declare compatibility. Unknown epochs fail closed.
	if current.DataFormat != 0 && current.DataFormat != 1 {
		return nil, errors.New("current data format requires a separate migration")
	}
	if err := Verify(bundle, current); err != nil {
		return nil, err
	}
	if installed, err := ServiceInstalled(instance); err != nil {
		return nil, err
	} else if installed {
		service, err := readService(instance)
		if err != nil {
			return nil, err
		}
		anchor, err := ReadManifest(filepath.Dir(service.Launcher))
		if err != nil {
			return nil, err
		}
		if anchor.UpdateFormat != 1 {
			return nil, errors.New("sign-in startup uses a pre-updater launcher; remove and reinstall it with this release before updating")
		}
	}
	if err := os.MkdirAll(updatePath(instance), 0o700); err != nil {
		return nil, err
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(idBytes)
	work := filepath.Join(updatePath(instance), id)
	if err := os.Mkdir(work, 0o700); err != nil {
		return nil, err
	}
	defer func() {
		if result != nil {
			result = errors.Join(result, os.RemoveAll(work))
		}
	}()
	archive := filepath.Join(work, "download")
	digest, err := fetchUpdate(ctx, options.Archive, options.SHA256, archive)
	if err != nil {
		return nil, err
	}
	destination := filepath.Join(work, "release")
	if err := os.Mkdir(destination, 0o700); err != nil {
		return nil, err
	}
	candidate, err := extractUpdate(archive, destination)
	if err != nil {
		return nil, err
	}
	target, err := verifyUpdateBundle(candidate)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(instance)
	if err != nil {
		return nil, err
	}
	if cfg.MudletDirectory != "" {
		if _, ok := target.Files["mudlet/darkpawns.xml"]; !ok {
			return nil, errors.New("release lacks the configured hosted Mudlet package; your world was not stopped")
		}
	}
	if err := syncTreeDirectories(destination); err != nil {
		return nil, err
	}
	if err := os.Remove(archive); err != nil {
		return nil, err
	}
	same := target.Version == current.Version && target.Commit == current.Commit && maps.Equal(target.Files, current.Files)
	if same {
		for _, name := range []string{"darkpawns", "darkpawns-server"} {
			if runtime.GOOS == "windows" {
				name += ".exe"
			}
			before, err := fileDigest(filepath.Join(bundle, name))
			if err != nil {
				return nil, err
			}
			after, err := fileDigest(filepath.Join(candidate, name))
			if err != nil {
				return nil, err
			}
			if before != after {
				same = false
			}
		}
	}
	return &UpdatePlan{Version: target.Version, PreviousVersion: current.Version, SHA256: digest, UpToDate: same, instance: instance, work: work, candidate: candidate, previous: bundle, id: id}, nil
}

// Close discards an unapplied plan; durable transactions and their backups stay.
func (p *UpdatePlan) Close() error {
	if p.retained {
		return nil
	}
	record, err := readUpdate(p.instance)
	if err == nil && record.ID == p.id {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.RemoveAll(p.work)
}

func (p *UpdatePlan) Apply(ctx context.Context) (result error) {
	if p.UpToDate {
		return nil
	}
	instance := p.instance
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	if err := pendingUpdate(instance); err != nil {
		return err
	}
	active, err := ActiveBundle(p.previous, instance)
	if err != nil {
		return err
	}
	if active != p.previous {
		return errors.New("active release changed after staging; run update again")
	}
	if _, err := verifyUpdateBundle(p.candidate); err != nil {
		return err
	}
	status, err := Status(ctx, instance)
	if err != nil {
		return err
	}
	if status.State != "stopped" && (status.State != "running" || status.Bundle != p.previous) {
		return errors.New("wait for the selected release to finish starting or stopping before updating")
	}
	record := updateRecord{ID: p.id, Phase: "prepared", Previous: p.previous, Candidate: p.candidate, SHA256: p.SHA256, WasRunning: status.State == "running"}
	if err := saveUpdate(instance, record); err != nil {
		return err
	}
	p.retained = true
	// From this point retain the journal and backups on every failure. Recovery is
	// explicit after an interruption; automatic recovery only follows our own error.
	fail := func(cause error) error {
		recoveryCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		record.Error = cause.Error()
		if err := saveUpdate(instance, record); err != nil {
			return errors.Join(cause, err)
		}
		if err := recoverUpdate(recoveryCtx, instance, &record); err != nil {
			return errors.Join(cause, fmt.Errorf("recovery incomplete; use %s: %w", OperatorCommand(instance, "update", "recover"), err))
		}
		return fmt.Errorf("update failed; previous release restored (backup and failed state retained in %s): %w", p.work, cause)
	}
	if err := stopInstance(ctx, instance); err != nil {
		return fmt.Errorf("update stopped before backup; inspect shutdown, then use %s: %w", OperatorCommand(instance, "update", "recover"), err)
	}
	lock, err := stoppedLock(instance)
	if err != nil {
		return err
	}
	err = func() (backupErr error) {
		defer func() { backupErr = errors.Join(backupErr, lock()) }()
		if err := databaseIfPresent(ctx, instance); err != nil {
			return err
		}
		record.Phase = "backing-up"
		if err := saveUpdate(instance, record); err != nil {
			return err
		}
		if err := snapshotInstance(instance, filepath.Join(p.work, "backup")); err != nil {
			return err
		}
		if err := verifySnapshot(p.work); err != nil {
			return err
		}
		record.Phase = "backed-up"
		return saveUpdate(instance, record)
	}()
	if err != nil {
		return fail(err)
	}
	// Acquire the run lock again before pointer activation. Direct foreground runs
	// are rejected by the durable pending journal after they obtain that lock.
	lock, err = stoppedLock(instance)
	if err != nil {
		return fail(err)
	}
	err = setActive(instance, p.candidate)
	if err == nil {
		record.Phase = "testing"
		err = saveUpdate(instance, record)
	}
	err = errors.Join(err, lock())
	if err != nil {
		return fail(err)
	}
	if record.WasRunning {
		if err := startUpdateCandidate(ctx, instance, record); err != nil {
			return fail(err)
		}
	}
	record.Phase = "committed"
	if err := saveUpdate(instance, record); err != nil {
		record.Phase = "testing"
		return fail(err)
	}
	return updateJSON(filepath.Join(p.work, "receipt.json"), record)
}

func startUpdateCandidate(ctx context.Context, instance string, record updateRecord) error {
	launcher := filepath.Join(record.Candidate, "darkpawns")
	if runtime.GOOS == "windows" {
		launcher += ".exe"
	}
	// Strict publisher authentication (L1-006/007/021): nothing spawns from
	// a candidate — a recovery journal's Previous included — whose manifest
	// is not publisher-signed with matching checksums.
	if err := PublisherVerifyExecutables(record.Candidate, "darkpawns", "darkpawns-server"); err != nil {
		return fmt.Errorf("candidate release is not publisher-authenticated: %w", err)
	}
	var previousExit exitRecord
	_ = readJSON(filepath.Join(instance, ".last-exit.json"), &previousExit)
	if err := spawnSupervisor(launcher, instance, "_serve-update", "--transaction", record.ID); err != nil {
		return err
	}
	wait, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, err := Status(wait, instance)
		if err == nil && status.State == "running" {
			if status.Bundle != record.Candidate {
				return errors.New("a different release became active during the update")
			}
			return nil
		}
		if err == nil && status.State == "stopped" {
			var exited exitRecord
			if readJSON(filepath.Join(instance, ".last-exit.json"), &exited) == nil && exited.RunID != "" && exited.RunID != previousExit.RunID {
				return fmt.Errorf("new release exited before readiness: %s", exited.Error)
			}
		}
		select {
		case <-wait.Done():
			return fmt.Errorf("new release did not become ready; inspect lib/logs/launcher.log: %w", wait.Err())
		case <-ticker.C:
		}
	}
}

// RecoverUpdate resumes an interrupted recovery, retaining failed candidate data.
func RecoverUpdate(ctx context.Context, instance string) (result error) {
	instance, err := canonicalInstance(instance)
	if err != nil {
		return err
	}
	unlock, err := lifecycleLock(instance)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, unlock()) }()
	record, err := readUpdate(instance)
	if err != nil {
		return err
	}
	if record.Phase == "committed" || record.Phase == "rolled-back" {
		return errors.New("no interrupted update needs recovery")
	}
	return recoverUpdate(ctx, instance, &record)
}

func recoverUpdate(ctx context.Context, instance string, record *updateRecord) (result error) {
	if stopErr := stopInstance(ctx, instance); stopErr != nil {
		// A failed candidate can have exited with a failure receipt. Restoration is
		// still safe only after its process lock has actually been released.
		locked, lockErr := instanceLocked(instance)
		if lockErr != nil || locked {
			return errors.Join(stopErr, lockErr)
		}
	}
	lock, err := stoppedLock(instance)
	if err != nil {
		return err
	}
	locked := true
	defer func() {
		if locked {
			result = errors.Join(result, lock())
		}
	}()
	work := filepath.Join(updatePath(instance), record.ID)
	// No candidate was allowed to run before testing. A partial backup is not
	// usable and is never restored. Existing data stays untouched in that case.
	if record.Phase == "testing" || record.Phase == "recovering" {
		if err := verifySnapshot(work); err != nil {
			return err
		}
		record.Phase = "recovering"
		if err := saveUpdate(instance, *record); err != nil {
			return err
		}
		failed := filepath.Join(work, "failed-lib")
		if _, err := os.Stat(failed); os.IsNotExist(err) {
			if err := os.Rename(filepath.Join(instance, "lib"), failed); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		// An interrupted restore can be retried without recopying over partial data.
		restore := filepath.Join(work, "restore-lib")
		if err := os.RemoveAll(restore); err != nil {
			return err
		}
		if _, err := os.Stat(filepath.Join(instance, "lib")); os.IsNotExist(err) {
			if err := copyTree(filepath.Join(work, "backup", "lib"), restore, nil, false); err != nil {
				return err
			}
			if err := os.Rename(restore, filepath.Join(instance, "lib")); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		data, err := os.ReadFile(filepath.Join(work, "backup", "instance.json"))
		if err != nil {
			return err
		}
		if err := updateBytes(filepath.Join(instance, "instance.json"), data); err != nil {
			return err
		}
		if err := databaseIfPresent(ctx, instance); err != nil {
			return err
		}
	}
	if err := setActive(instance, record.Previous); err != nil {
		return err
	}
	record.Phase = "rolled-back"
	if err := saveUpdate(instance, *record); err != nil {
		return err
	}
	locked = false
	if err := lock(); err != nil {
		return err
	}
	if record.WasRunning {
		if _, err := start(ctx, record.Previous, instance); err != nil {
			return fmt.Errorf("previous state restored but restart failed: %w", err)
		}
	}
	return updateJSON(filepath.Join(work, "receipt.json"), record)
}
