package session

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/olc"
)

// Zone-file save infrastructure shared by the OLC editors (oedit, medit,
// redit). The three editors write different files per zone (N.obj, N.mob,
// N.wld) but share two hazards, both introduced by running C's
// single-threaded save path under Go's per-session goroutines:
//
//  1. A direct write to the final path can leave a truncated zone file if
//     the process dies mid-write. atomicWriteFile stages the content in a
//     temporary file in the same directory and renames it into place, so
//     readers never observe a partial file.
//  2. Two builders saving the same zone concurrently interleave their
//     writes, and a working-copy commit landing between a save's snapshot
//     and its save-list cleanup gets its dirty marker cleared without being
//     persisted. zoneSaveLock serializes the whole snapshot -> write ->
//     cleanup sequence per zone, and the commit paths hold the same lock
//     across world-commit -> marker-set, so a commit is either fully inside
//     the snapshot or keeps its marker — never silently marked saved.

// atomicWriteFile writes data to path atomically: the bytes go to a new
// temporary file in path's directory, which is then renamed over path. The
// rename is atomic on POSIX filesystems, so a crash or failed write can
// never leave a truncated file at path. The delivered bytes are identical
// to a direct write; only the delivery mechanism changes.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	path = filepath.Clean(path)
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup: a successful rename removes the temp name, so
	// this only fires on the error paths below.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	// Close before rename so the content is fully flushed to the temp
	// file first.
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// zoneSaveLock returns the mutex serializing zone-file saves for one zone
// number, creating it on first use. The three editors share one lock per
// zone: a single lock keeps the save/commit ordering argument simple, and
// zone saves are rare enough that the extra serialization is unmeasurable.
func zoneSaveLock(zone int) *sync.Mutex {
	return olc.ZoneSaveLock(zone)
}
