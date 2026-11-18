package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// olcOpenParentObstruction recognizes only the common C fopen/Go atomic-open
// boundary approved in #1825: a missing or non-directory parent. Temporary
// file permissions, target permissions and later atomic-save stages are not
// interchangeable with fopen, so an operation name alone is insufficient.
func olcOpenParentObstruction(err error, parent string) bool {
	var failure *os.PathError
	if parent == "" || !errors.As(err, &failure) || (failure.Op != "open" && failure.Op != "mkdir") {
		return false
	}
	if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
		return false
	}
	info, statErr := os.Stat(parent)
	if statErr == nil {
		return !info.IsDir()
	}
	return errors.Is(statErr, os.ErrNotExist) || errors.Is(statErr, syscall.ENOTDIR)
}

// olcSaveTargetDirectoryFailure recognizes atomic replacement failing because
// the exact C fopen target is a directory. C opens that target directly and
// emits its open-failure diagnostic; Go first writes a sibling temporary file,
// then gets a directory-target error from rename. Other rename/write failures
// remain distinct.
func olcSaveTargetDirectoryFailure(err error, target string) bool {
	var failure *os.LinkError
	targetKindError := errors.Is(err, syscall.EISDIR) || errors.Is(err, syscall.EEXIST) || errors.Is(err, syscall.ENOTEMPTY)
	if target == "" || !errors.As(err, &failure) || failure.Op != "rename" ||
		filepath.Clean(failure.New) != filepath.Clean(target) || !targetKindError {
		return false
	}
	info, statErr := os.Stat(target)
	return statErr == nil && info.IsDir()
}

// logOLCSaveFailure is called by descriptor save-command error arms after
// save helpers release their zone and snapshot locks. It preserves the proven
// missing/non-directory-parent boundary and also maps an exact target-directory
// rename failure to C's fopen failure. Each caller supplies its own C payload.
func logOLCSaveFailure(world *game.World, extension string, zone int, payload string, err error) {
	if world == nil {
		return
	}
	root := world.WorldPath
	if extension != "wld" && extension != "zon" {
		parsed := world.GetParsedWorld()
		if parsed == nil || parsed.SourceDir == "" {
			return
		}
		root = parsed.SourceDir
	}
	parent := filepath.Join(root, extension)
	target := filepath.Join(parent, fmt.Sprintf("%d.%s", zone, extension))
	if root != "" && (olcOpenParentObstruction(err, parent) || olcSaveTargetDirectoryFailure(err, target)) {
		game.MudLog(payload, game.MudlogBrief, game.LVL_IMMORT, true)
	}
}
