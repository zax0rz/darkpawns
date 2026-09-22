package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/fileedit"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// These expressions are the file-edit.c expressions verbatim. The path
// containment check below is an additional Go-side traversal guard; it does
// not replace the C checks that define the player-facing command surface.
var (
	scriptsRootPattern    = regexp.MustCompile(`^(mob|obj|room)`)
	validDirectoryPattern = regexp.MustCompile(`^[/a-zA-Z0-9_-]+$`)
	validFilenamePattern  = regexp.MustCompile(`^[a-zA-Z0-9_-]+.?[a-zA-Z0-9_-]*$`)
)

const luaEditError = "Invalid lua edit option.\r\n"

// cmdLuaEdit is the registered wrapper for do_luaedit.
func cmdLuaEdit(s *Session, args []string) error {
	return do_luaedit(s, args)
}

// do_luaedit ports src/luaedit.c:28-58. The second argument is deliberately
// ignored when the first token is not mob/obj/room: that is the C arg dance,
// even though the helper name is_scripts_root suggests the opposite.
func do_luaedit(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}

	arg1, arg2 := luaEditTwoArguments(args)
	root := luaScriptsDir(s.manager.world)
	dir := root
	if arg1 == "" {
		// root already selected
	} else if is_scripts_root(arg1) {
		arg2 = arg1
	} else {
		// Keep the raw .. components visible to valid_directory and the
		// traversal guard. filepath.Join would clean them before validation.
		dir = root + "/" + arg1
	}

	if arg2 == "" {
		if err := list_directory(s, dir); err != nil {
			s.sendTextEditor(luaEditError)
		}
		return nil
	}

	// C uses strstr, not a suffix test: my.lua.txt is not changed before the
	// valid_filename check rejects it.
	if !strings.Contains(arg2, ".lua") {
		arg2 += ".lua"
	}

	var err error
	if s.player.GetLevel() < LVL_HIGOD {
		err = view_file(s, dir, arg2)
	} else {
		err = edit_file(s, dir, arg2, true)
	}
	if err != nil {
		s.sendTextEditor(luaEditError)
	}
	return nil
}

func luaEditTwoArguments(args []string) (string, string) {
	arg1, rest := game.OneArgument(strings.Join(args, " "))
	arg2, _ := game.OneArgument(rest)
	return arg1, arg2
}

// luafilter ports luaedit.c:10-13. C's scandir filter sees names only, so
// matching a directory is intentional.
func luafilter(name string) bool {
	return fileedit.LuaFilter(name)
}

// is_scripts_root ports luaedit.c:15-18. It returns true for a root filename;
// the unanchored tail is intentional, so names beginning with mob, obj, or
// room select a subdirectory instead.
func is_scripts_root(pathname string) bool {
	return !scriptsRootPattern.MatchString(pathname)
}

// valid_directory ports file-edit.c:78-81.
func valid_directory(pathname string) bool {
	return validDirectoryPattern.MatchString(pathname)
}

// valid_filename ports file-edit.c:118-121, including its unescaped optional
// wildcard character between the basename and suffix.
func valid_filename(filename string) bool {
	return validFilenamePattern.MatchString(filename)
}

// luaScriptsDir returns the same tree passed to scripting.NewEngine. Tests and
// embedders may set World.ScriptsDir explicitly; the production default is
// the running world's <world>/scripts directory.
func luaScriptsDir(w *game.World) string {
	if w.ScriptsDir != "" {
		return filepath.Clean(w.ScriptsDir)
	}
	if w.WorldPath != "" {
		return filepath.Clean(filepath.Join(w.WorldPath, "scripts"))
	}
	return "scripts"
}

// pathWithinLuaRoot is the Go-side traversal guard. The C regexes reject the
// ordinary ../ spellings, but checking the resolved lexical path keeps the
// safety invariant true if a future caller supplies a different directory.
func pathWithinLuaRoot(root, target string) bool {
	if hasParentComponent(target) {
		return false
	}
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}

	// Also reject a symlinked directory or file whose resolved target leaves
	// the scripts tree. Missing files are allowed here because edit_file must
	// be able to create them; their existing parent is still checked.
	resolvedRoot, rootErr := filepath.EvalSymlinks(rootAbs)
	if rootErr != nil {
		return true
	}
	resolvedTarget, targetErr := filepath.EvalSymlinks(targetAbs)
	if targetErr == nil {
		return pathWithinResolvedRoot(resolvedRoot, resolvedTarget)
	}
	resolvedParent, parentErr := filepath.EvalSymlinks(filepath.Dir(targetAbs))
	if parentErr != nil {
		return true
	}
	return pathWithinResolvedRoot(resolvedRoot, filepath.Join(resolvedParent, filepath.Base(targetAbs)))
}

func hasParentComponent(path string) bool {
	for _, component := range strings.Split(filepath.ToSlash(path), "/") {
		if component == ".." {
			return true
		}
	}
	return false
}

func pathWithinResolvedRoot(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// list_directory ports file-edit.c:83-114. os.ReadDir returns entries in
// filename order, matching scandir(..., alphasort) for the ASCII command
// names admitted by luafilter.
func list_directory(s *Session, dir string) error {
	root := luaScriptsDir(s.manager.world)
	if !valid_directory(dir) || !pathWithinLuaRoot(root, dir) {
		return fmt.Errorf("invalid script directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var out strings.Builder
	count := 0
	for _, entry := range entries {
		if !luafilter(entry.Name()) {
			continue
		}
		if count > 0 && count%3 == 0 {
			out.WriteString("\r\n")
		}
		fmt.Fprintf(&out, "%-26.26s", entry.Name())
		count++
	}
	if count == 0 {
		out.WriteString("None.\r\n")
	} else {
		out.WriteString("\r\n")
	}
	s.sendTextEditor(out.String())
	return nil
}

// view_file ports file-edit.c:128-147. file_to_string_alloc's CRLF
// conversion and MAX_STRING_LENGTH cap are shared with tedit through
// readCTextFile.
func view_file(s *Session, dir, filename string) error {
	if !valid_filename(filename) {
		return fmt.Errorf("invalid script filename")
	}
	if hasParentComponent(dir) {
		return fmt.Errorf("script path escaped root")
	}
	path := filepath.Join(dir, filename)
	if !pathWithinLuaRoot(luaScriptsDir(s.manager.world), path) {
		return fmt.Errorf("script path escaped root")
	}
	text, err := readCTextFile(path)
	if err != nil {
		return err
	}
	s.sendTextEditor(text)
	return nil
}

// edit_file ports file-edit.c:176-198 and enters the shared improved editor.
// A file without simultaneous read/write access intentionally starts as an
// empty buffer, allowing a new file to be created on save.
func edit_file(s *Session, dir, filename string, killOnEmpty bool) error {
	if !valid_filename(filename) {
		return fmt.Errorf("invalid script filename")
	}
	if hasParentComponent(dir) {
		return fmt.Errorf("script path escaped root")
	}
	root := luaScriptsDir(s.manager.world)
	path := filepath.Join(dir, filename)
	if !pathWithinLuaRoot(root, path) {
		return fmt.Errorf("script path escaped root")
	}

	text := ""
	// os.Root scopes the open (and the read) under the scripts tree so a
	// path that slipped the lexical guards fails closed here instead of
	// reaching the filesystem. C's valid_filename stays the behavior law;
	// the root is the security boundary gosec can see.
	if luaRoot, rootErr := os.OpenRoot(root); rootErr == nil {
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil {
			if file, err := luaRoot.OpenFile(rel, os.O_RDWR, 0); err == nil {
				if closeErr := file.Close(); closeErr != nil {
					slog.Error("luaedit file close failed", "player", s.playerName, "file", path, "error", closeErr)
				}
				if loaded, readErr := readCTextFile(path); readErr == nil {
					text = loaded
				}
			}
		}
		if closeErr := luaRoot.Close(); closeErr != nil {
			slog.Error("luaedit scripts root close failed", "error", closeErr)
		}
	}

	return s.startFileEdit(textEditState{
		field:       textEditField{name: "luaedit", maxBytes: cMaxStringLength},
		path:        path,
		original:    text,
		buffer:      text,
		killOnEmpty: killOnEmpty,
	})
}

// forgetScriptFailures clears the scripting engine's negative cache after a
// save or delete under the scripts tree. The cache assumes failures are
// stable per file (DP-903); an in-game edit invalidates that assumption, so
// a fixed script runs on its next trigger instead of after a reboot.
// Non-script files (tedit's news/motd) leave the cache untouched.
func (s *Session) forgetScriptFailures(path string) {
	if game.ScriptEngine == nil || s.manager == nil || s.manager.world == nil {
		return
	}
	if !pathWithinLuaRoot(luaScriptsDir(s.manager.world), path) {
		return
	}
	game.ScriptEngine.ForgetFailures()
}
