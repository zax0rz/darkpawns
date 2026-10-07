package session

import (
	"errors"
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

// logOLCOpenParentFailure is called only by the descriptor save-command error
// arms, after save helpers release their zone and snapshot locks. The shared
// admin writers deliberately do not call it. src/redit.c:291-294 is the first
// producer; each caller cites and supplies its own exact C payload.
func logOLCOpenParentFailure(world *game.World, extension, payload string, err error) {
	if world == nil {
		return
	}
	root := world.WorldPath
	if extension != "wld" {
		parsed := world.GetParsedWorld()
		if parsed == nil || parsed.SourceDir == "" {
			return
		}
		root = parsed.SourceDir
	}
	if root != "" && olcOpenParentObstruction(err, filepath.Join(root, extension)) {
		game.MudLog(payload, game.MudlogBrief, game.LVL_IMMORT, true)
	}
}
