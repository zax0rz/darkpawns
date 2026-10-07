package session

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// killEditorFile ports src/file-edit.c:19-25. access(R_OK) uses the real uid
// and follows symlinks. Any access failure means success without removal,
// including unreadable files and dangling links; remove failure is separate.
func killEditorFile(path string) error {
	if unix.Access(path, unix.R_OK) != nil {
		return nil
	}
	return os.Remove(path)
}

// The caller holds textEditMu and liveTextEditMu. Snapshot attached-body
// identity under the manager lock, then release it before MudLog delivery.
func (s *Session) fileEditorActorName() string {
	s.manager.mu.RLock()
	defer s.manager.mu.RUnlock()
	if s.isSwitched && s.switchedMob != nil {
		return s.switchedMob.GetName()
	}
	return s.player.GetName()
}

// os.OpenRoot returns a non-errno "not a directory" error for a regular
// root on Linux. Recognize this exact root-open stage by its path, plus
// the inspected non-directory parent. Other atomic stages stay unclassified.
func fileEditorOpenParentFailure(err error, path string) bool {
	parent := filepath.Dir(path)
	if olcOpenParentObstruction(err, parent) {
		return true
	}
	var failure *os.PathError
	if !errors.As(err, &failure) || failure.Op != "open" || failure.Path != parent {
		return false
	}
	info, statErr := os.Stat(parent)
	return statErr == nil && !info.IsDir()
}
