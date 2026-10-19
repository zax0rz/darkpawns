//go:build !windows

package session

import (
	"os"

	"golang.org/x/sys/unix"
)

// killEditorFile ports src/file-edit.c:19-25. Keep the real-uid access(R_OK)
// probe on POSIX: unreadable files and dangling links succeed without removal.
func killEditorFile(path string) error {
	if unix.Access(path, unix.R_OK) != nil {
		return nil
	}
	return os.Remove(path)
}
