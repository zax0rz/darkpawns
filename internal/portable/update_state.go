package portable

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofrs/flock"
)

type (
	activeRelease struct {
		Bundle string `json:"bundle"`
	}
	updateRecord struct {
		ID         string `json:"id"`
		Phase      string `json:"phase"`
		Previous   string `json:"previous"`
		Candidate  string `json:"candidate"`
		SHA256     string `json:"sha256"`
		WasRunning bool   `json:"was_running"`
		Error      string `json:"error,omitempty"`
	}
)

func validateUpdatesDir(instance string) error {
	info, err := os.Lstat(filepath.Join(instance, ".updates"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New(".updates must be a real private directory, not a link")
	}
	return nil
}

func updatePath(instance string) string { return filepath.Join(instance, ".updates") }
func readUpdate(instance string) (updateRecord, error) {
	var record updateRecord
	canonical, canonicalErr := canonicalInstance(instance)
	if canonicalErr != nil {
		return record, canonicalErr
	}
	instance = canonical
	if err := validateUpdatesDir(instance); err != nil {
		return record, err
	}
	err := readJSON(filepath.Join(updatePath(instance), "transaction.json"), &record)
	if err != nil {
		return record, err
	}
	if len(record.ID) != 32 || strings.ContainsAny(record.ID, "/\\") || !filepath.IsAbs(record.Previous) || !filepath.IsAbs(record.Candidate) {
		return record, errors.New("invalid update journal; retain .updates for inspection")
	}
	if rel, err := filepath.Rel(filepath.Join(updatePath(instance), record.ID, "release"), record.Candidate); err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return record, errors.New("candidate release is outside its transaction directory")
	}
	if _, err := hex.DecodeString(record.ID); err != nil {
		return record, errors.New("invalid update transaction identity")
	}
	switch record.Phase {
	case "prepared", "backing-up", "backed-up", "testing", "recovering", "committed", "rolled-back":
	default:
		return record, errors.New("unknown update phase")
	}
	return record, nil
}

func PendingUpdate(instance string) error {
	return pendingUpdate(instance)
}

func pendingUpdate(instance string) error {
	record, err := readUpdate(instance)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if record.Phase != "committed" && record.Phase != "rolled-back" {
		return fmt.Errorf("update %s needs recovery; use %s", record.ID, OperatorCommand(instance, "update", "recover"))
	}
	return nil
}

func saveUpdate(instance string, record updateRecord) error {
	return updateJSON(filepath.Join(updatePath(instance), "transaction.json"), record)
}

func updateJSON(path string, value any) (result error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return updateBytes(path, append(data, '\n'))
}

func updateBytes(path string, data []byte) (result error) {
	f, err := os.CreateTemp(filepath.Dir(path), ".update-record-*")
	if err != nil {
		return err
	}
	defer func() {
		err := os.Remove(f.Name())
		if !os.IsNotExist(err) {
			result = errors.Join(result, err)
		}
	}()
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Sync(), f.Close()); err != nil {
		return err
	}
	return replaceUpdateRecord(f.Name(), path)
}

// ActiveBundle keeps the original launcher and native job as a stable entry
// point. Releases are installed side by side; no running executable is replaced.
func ActiveBundle(fallback, instance string) (string, error) {
	var active activeRelease
	if err := validateUpdatesDir(instance); err != nil {
		return "", err
	}
	err := readJSON(filepath.Join(updatePath(instance), "active.json"), &active)
	if os.IsNotExist(err) {
		return fallback, nil
	}
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(active.Bundle) {
		return "", errors.New("invalid active release path")
	}
	info, err := os.Stat(active.Bundle)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("active release is not a directory")
	}
	return active.Bundle, nil
}

func setActive(instance, bundle string) error {
	return updateJSON(filepath.Join(updatePath(instance), "active.json"), activeRelease{Bundle: bundle})
}

func updateRunAllowed(instance, bundle, id string) error {
	if id == "" {
		return pendingUpdate(instance)
	}
	record, err := readUpdate(instance)
	if err != nil {
		return err
	}
	if record.ID != id || record.Phase != "testing" || record.Candidate != bundle {
		return errors.New("update supervisor does not match the pending transaction")
	}
	return nil
}

func stoppedLock(instance string) (func() error, error) {
	lock := flock.New(filepath.Join(instance, ".run.lock"))
	ok, err := lock.TryLock()
	if err != nil || !ok {
		return nil, errors.Join(errors.New("instance is not fully stopped; state was not replaced"), err, lock.Close())
	}
	return lock.Close, nil
}

// Refuse symlinks/special files anywhere in the stopped instance. Copy the whole
// regular-file tree, except updater history and ephemeral process ownership.
func snapshotInstance(instance, destination string) error {
	skip := map[string]bool{".updates": true, ".run.lock": true, ".lifecycle.lock": true, ".control.json": true, ".last-exit.json": true, ".supervision.lock": true}
	return copyTree(instance, destination, skip, true)
}

func copyTree(source, destination string, skip map[string]bool, recordHashes bool) error {
	hashes := map[string]string{}
	if err := os.MkdirAll(destination, 0o700); err != nil {
		return err
	}
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skip[rel] {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() && !entry.IsDir() {
			return fmt.Errorf("backup refuses link or special file: %s", rel)
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		if err := copyUpdateFile(path, target); err != nil {
			return err
		}
		sum, err := fileDigest(target)
		if err != nil {
			return err
		}
		hashes[filepath.ToSlash(rel)] = sum
		return nil
	})
	if err != nil {
		return err
	}
	if err := syncTreeDirectories(destination); err != nil {
		return err
	}
	if recordHashes {
		return updateJSON(filepath.Join(filepath.Dir(destination), "backup-manifest.json"), hashes)
	}
	return nil
}

func copyUpdateFile(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	return errors.Join(copyErr, out.Sync(), out.Close())
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func verifySnapshot(work string) error {
	var hashes map[string]string
	if err := readJSON(filepath.Join(work, "backup-manifest.json"), &hashes); err != nil {
		return err
	}
	if len(hashes) == 0 {
		return errors.New("empty backup manifest")
	}
	root := filepath.Join(work, "backup")
	got := 0
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.IsDir() {
			return nil
		}
		if !e.Type().IsRegular() {
			return errors.New("backup contains nonregular file")
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum, err := fileDigest(path)
		if err != nil {
			return err
		}
		if hashes[filepath.ToSlash(rel)] != sum {
			return fmt.Errorf("backup checksum mismatch: %s", rel)
		}
		got++
		return nil
	})
	if err != nil {
		return err
	}
	if got != len(hashes) {
		return errors.New("backup is incomplete")
	}
	return nil
}

func databaseIfPresent(ctx context.Context, root string) error {
	path := filepath.Join(root, "lib", "data", "darkpawns.db")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return err
	}
	return checkDatabase(ctx, path)
}

func syncTreeDirectories(root string) error {
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, e fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if e.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := syncUpdateDirectory(dirs[i]); err != nil {
			return err
		}
	}
	return nil
}
