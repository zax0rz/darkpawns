// Package fileedit owns filesystem-safe reads and atomic writes shared by the
// telnet file editor (tedit, luaedit) and webOLC's Lua/tedit controllers.
//
// Every web operation is scoped by an os.Root opened on the caller-supplied
// root, so a symlink or a path that slipped the lexical guard fails closed
// instead of reaching the filesystem. Writes are temp-file + rename in the
// target's directory: a script that runs mid-save sees the old bytes or the
// new bytes, never a truncated file.
package fileedit

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/game"
)

var (
	ErrInvalidPath = errors.New("invalid file path")
	ErrConflict    = errors.New("file changed since it was opened")
	ErrExists      = errors.New("file already exists")
)

// TextField is one row of C's tedit table (src/tedit.c). The table order is
// load-bearing: tedit resolves abbreviated field names with strncmp and takes
// the first match. MinLevel is C's edit threshold for the field.
type TextField struct {
	Name     string
	Filename string
	MinLevel int
	MaxBytes int
}

// TextFields is the single authority for the tedit table; the telnet command
// and the web controller both read it.
var TextFields = []TextField{
	{Name: "credits", Filename: "credits", MinLevel: game.LVL_IMPL, MaxBytes: 2400},
	{Name: "news", Filename: "news", MinLevel: game.LVL_GOD, MaxBytes: 8192},
	{Name: "motd", Filename: "motd", MinLevel: game.LVL_GOD, MaxBytes: 2400},
	{Name: "imotd", Filename: "imotd", MinLevel: game.LVL_IMMORT, MaxBytes: 2400},
	{Name: "help", Filename: "help/screen", MinLevel: game.LVL_GOD, MaxBytes: 2400},
	{Name: "info", Filename: "info", MinLevel: game.LVL_GOD, MaxBytes: 8192},
	{Name: "background", Filename: "background", MinLevel: game.LVL_GRGOD, MaxBytes: 8192},
	{Name: "handbook", Filename: "handbook", MinLevel: game.LVL_GRGOD, MaxBytes: 8192},
	{Name: "policies", Filename: "policies", MinLevel: game.LVL_IMPL, MaxBytes: 8192},
	{Name: "future", Filename: "future", MinLevel: game.LVL_GRGOD, MaxBytes: 8192},
	{Name: "wizlist", Filename: "wizlist", MinLevel: game.LVL_IMPL, MaxBytes: 2400},
	{Name: "immlist", Filename: "immlist", MinLevel: game.LVL_IMPL, MaxBytes: 2400},
}

// TextFieldByName returns the exact-name table row. The web addresses fields
// by their full name; abbreviation matching stays a telnet affordance.
func TextFieldByName(name string) (TextField, bool) {
	for _, field := range TextFields {
		if field.Name == name {
			return field, true
		}
	}
	return TextField{}, false
}

// TextSavedHook refreshes the live copy of a tedit file after a save. The
// session package owns the boot text cache and registers the hook at init, so
// a web save becomes visible to news/motd/etc. exactly like a telnet save.
type TextSavedHook func(world *game.World, filename, text string)

var (
	textSavedMu   sync.RWMutex
	textSavedHook TextSavedHook
)

func RegisterTextSavedHook(hook TextSavedHook) {
	textSavedMu.Lock()
	textSavedHook = hook
	textSavedMu.Unlock()
}

// NotifyTextSaved reports whether a hook was registered to receive the save.
func NotifyTextSaved(world *game.World, filename, text string) bool {
	textSavedMu.RLock()
	hook := textSavedHook
	textSavedMu.RUnlock()
	if hook == nil {
		return false
	}
	hook(world, filename, text)
	return true
}

// luaFilterPattern is luaedit.c's scandir filter (luaedit.c:10-13): numeric
// vnum directories, the typed subtrees, the archive, and .lua files.
var luaFilterPattern = regexp.MustCompile(`^([0-9]+|archive|mob|obj|room|.+[.]lua)$`)

// LuaFilter reports whether luaedit lists name. C's filter sees names only,
// so a directory and a file are matched the same way.
func LuaFilter(name string) bool {
	return luaFilterPattern.MatchString(name)
}

type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	ETag    string `json:"etag"`
}

type Entry struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Directory     bool   `json:"directory"`
	MaxBytes      int    `json:"max_bytes,omitempty"`
	RequiredLevel int    `json:"required_level,omitempty"`
	RequiredLabel string `json:"required_label,omitempty"`
	Allowed       bool   `json:"allowed"`
}

// ETag is a strong, content-derived validator. mtime is useless here: an
// atomic rename can land two different contents inside one timestamp tick.
func ETag(content []byte) string {
	sum := sha256.Sum256(content)
	return fmt.Sprintf(`"%x"`, sum)
}

// writeMu serializes every compare-and-swap and every save. Edits are rare
// and human-paced; one lock keeps a web save and a telnet save from racing
// between the precondition check and the rename.
var writeMu sync.Mutex

// clean validates a root-relative name and returns it in OS form.
func clean(name string) (string, error) {
	if name == "" || filepath.IsAbs(name) || strings.ContainsRune(name, 0) {
		return "", ErrInvalidPath
	}
	c := filepath.Clean(filepath.FromSlash(name))
	if c == "." || c == ".." || strings.HasPrefix(c, ".."+string(filepath.Separator)) {
		return "", ErrInvalidPath
	}
	return c, nil
}

func openRoot(root string) (*os.Root, error) {
	return os.OpenRoot(filepath.Clean(root))
}

// rootError maps os.Root's escape refusal onto ErrInvalidPath.
func rootError(err error) error {
	if err != nil && strings.Contains(err.Error(), "path escapes from parent") {
		return ErrInvalidPath
	}
	return err
}

// regularTarget refuses symlinks and non-regular files at the leaf. Renaming
// over a symlink would replace the link rather than its target, so the web
// surface does not edit through links at all.
func regularTarget(r *os.Root, name string) (fs.FileInfo, error) {
	info, err := r.Lstat(name)
	if err != nil {
		return nil, rootError(err)
	}
	if !info.Mode().IsRegular() {
		return nil, ErrInvalidPath
	}
	return info, nil
}

func Read(root, name string) (File, error) {
	rel, err := clean(name)
	if err != nil {
		return File{}, err
	}
	r, err := openRoot(root)
	if err != nil {
		return File{}, err
	}
	defer func() { _ = r.Close() }()
	if _, err := regularTarget(r, rel); err != nil {
		return File{}, err
	}
	content, err := r.ReadFile(rel)
	if err != nil {
		return File{}, rootError(err)
	}
	return File{Path: filepath.ToSlash(rel), Content: string(content), ETag: ETag(content)}, nil
}

func List(root, directory string, filter func(fs.DirEntry) bool) ([]Entry, error) {
	r, err := openRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	rel := "."
	if directory != "" && directory != "." {
		if rel, err = clean(directory); err != nil {
			return nil, err
		}
	}
	entries, err := fs.ReadDir(r.FS(), filepath.ToSlash(rel))
	if err != nil {
		return nil, rootError(err)
	}
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Type()&fs.ModeSymlink != 0 || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if filter != nil && !filter(entry) {
			continue
		}
		path := entry.Name()
		if rel != "." {
			path = filepath.ToSlash(filepath.Join(rel, entry.Name()))
		}
		result = append(result, Entry{Name: entry.Name(), Path: path, Directory: entry.IsDir()})
	}
	return result, nil
}

// Write replaces an existing file only if its current content still hashes to
// ifMatch. A missing file is ErrNotExist, not a conflict: creation is Create.
func Write(root, name string, content []byte, ifMatch string) (File, error) {
	rel, err := clean(name)
	if err != nil {
		return File{}, err
	}
	if ifMatch == "" {
		return File{}, ErrConflict
	}
	r, err := openRoot(root)
	if err != nil {
		return File{}, err
	}
	defer func() { _ = r.Close() }()

	writeMu.Lock()
	defer writeMu.Unlock()
	info, err := regularTarget(r, rel)
	if err != nil {
		return File{}, err
	}
	current, err := r.ReadFile(rel)
	if err != nil {
		return File{}, rootError(err)
	}
	if ETag(current) != ifMatch {
		return File{}, ErrConflict
	}
	if err := replaceInRoot(r, rel, content, info.Mode().Perm(), true); err != nil {
		return File{}, err
	}
	return File{Path: filepath.ToSlash(rel), Content: string(content), ETag: ETag(content)}, nil
}

// Create writes a new file and refuses if anything already occupies the name
// (If-None-Match: *). The parent directory must already exist.
func Create(root, name string, content []byte, perm fs.FileMode) (File, error) {
	rel, err := clean(name)
	if err != nil {
		return File{}, err
	}
	r, err := openRoot(root)
	if err != nil {
		return File{}, err
	}
	defer func() { _ = r.Close() }()

	writeMu.Lock()
	defer writeMu.Unlock()
	if _, err := r.Lstat(rel); err == nil {
		return File{}, ErrExists
	} else if !errors.Is(err, fs.ErrNotExist) {
		return File{}, rootError(err)
	}
	if err := replaceInRoot(r, rel, content, perm, false); err != nil {
		return File{}, err
	}
	return File{Path: filepath.ToSlash(rel), Content: string(content), ETag: ETag(content)}, nil
}

func Delete(root, name, ifMatch string) error {
	rel, err := clean(name)
	if err != nil {
		return err
	}
	r, err := openRoot(root)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	writeMu.Lock()
	defer writeMu.Unlock()
	if _, err := regularTarget(r, rel); err != nil {
		return err
	}
	current, err := r.ReadFile(rel)
	if err != nil {
		return rootError(err)
	}
	if ifMatch == "" || ETag(current) != ifMatch {
		return ErrConflict
	}
	return rootError(r.Remove(rel))
}

// AtomicWrite is the telnet editor's save: the same temp-file + rename as the
// web path, with no precondition (the in-game editor is C-faithful
// last-writer-wins). A symlinked target is resolved first so the link keeps
// pointing at the edited file, matching the old os.WriteFile write-through.
// An existing file keeps its permission bits; a new file gets perm under the
// process umask, exactly as os.WriteFile did.
func AtomicWrite(path string, content []byte, perm fs.FileMode) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	r, err := openRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	name := filepath.Base(path)

	writeMu.Lock()
	defer writeMu.Unlock()
	info, err := r.Stat(name)
	switch {
	case err == nil:
		return replaceInRoot(r, name, content, info.Mode().Perm(), true)
	case errors.Is(err, fs.ErrNotExist):
		return replaceInRoot(r, name, content, perm, false)
	default:
		return err
	}
}

// replaceInRoot writes content beside rel and renames it into place. When
// preserve is set, perm is the existing file's mode and is applied exactly;
// otherwise perm is a creation mode and the umask applies.
func replaceInRoot(r *os.Root, rel string, content []byte, perm fs.FileMode, preserve bool) error {
	var suffix [8]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return err
	}
	tmpName := filepath.Join(filepath.Dir(rel), ".fileedit-"+hex.EncodeToString(suffix[:]))
	tmp, err := r.OpenFile(tmpName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return rootError(err)
	}
	renamed := false
	defer func() {
		if !renamed {
			_ = r.Remove(tmpName)
		}
	}()
	if preserve {
		if err := tmp.Chmod(perm); err != nil {
			_ = tmp.Close()
			return err
		}
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := r.Rename(tmpName, rel); err != nil {
		return rootError(err)
	}
	renamed = true
	return nil
}
