//go:build !windows

package portable

import (
	"errors"
	"os"
	"path/filepath"
)

func replaceUpdateRecord(from, to string) error {
	if err := os.Rename(from, to); err != nil {
		return err
	}
	return syncUpdateDirectory(filepath.Dir(to))
}

func syncUpdateDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
