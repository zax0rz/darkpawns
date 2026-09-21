package session

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// textEditField is the C tedit table from src/tedit.c. The table order is
// load-bearing: tedit resolves abbreviated field names with strncmp and takes
// the first match.
type textEditField struct {
	name     string
	filename string
	minLevel int
	maxBytes int
}

var textEditFields = []textEditField{
	{name: "credits", filename: "credits", minLevel: LVL_IMPL, maxBytes: 2400},
	{name: "news", filename: "news", minLevel: LVL_GOD, maxBytes: 8192},
	{name: "motd", filename: "motd", minLevel: LVL_GOD, maxBytes: 2400},
	{name: "imotd", filename: "imotd", minLevel: LVL_IMMORT, maxBytes: 2400},
	{name: "help", filename: "help/screen", minLevel: LVL_GOD, maxBytes: 2400},
	{name: "info", filename: "info", minLevel: LVL_GOD, maxBytes: 8192},
	{name: "background", filename: "background", minLevel: LVL_GRGOD, maxBytes: 8192},
	{name: "handbook", filename: "handbook", minLevel: LVL_GRGOD, maxBytes: 8192},
	{name: "policies", filename: "policies", minLevel: LVL_IMPL, maxBytes: 8192},
	{name: "future", filename: "future", minLevel: LVL_GRGOD, maxBytes: 8192},
	{name: "wizlist", filename: "wizlist", minLevel: LVL_IMPL, maxBytes: 2400},
	{name: "immlist", filename: "immlist", minLevel: LVL_IMPL, maxBytes: 2400},
}

const textEditHelp = "Editor command formats: /<letter>\r\n\r\n" +
	"/a         -  aborts editor\r\n" +
	"/c         -  clears buffer\r\n" +
	"/d#        -  deletes a line #\r\n" +
	"/e# <text> -  changes the line at # with <text>\r\n" +
	"/f         -  formats text\r\n" +
	"/fi        -  indented formatting of text\r\n" +
	"/h         -  list text editor commands\r\n" +
	"/i# <text> -  inserts <text> before line #\r\n" +
	"/l         -  lists buffer\r\n" +
	"/n         -  lists buffer with line numbers\r\n" +
	"/r 'a' 'b' -  replace 1st occurance of text <a> in buffer with text <b>\r\n" +
	"/ra 'a' 'b'-  replace all occurances of text <a> within buffer with text <b>\r\n" +
	"              usage: /r[a] 'pattern' 'replacement'\r\n" +
	"/s         -  saves text\r\n"

type textEditState struct {
	field    textEditField
	path     string
	original string
	buffer   string
	// cacheKey is non-empty for tedit's process-global static text buffers.
	// luaedit uses the same editor state and save path without a live cache.
	cacheKey    string
	killOnEmpty bool
	// roomEditor keeps the bounded REDIT string_write buffer descriptor-local.
	// Ordinary tedit intentionally shares the C process-global text pointer;
	// room descriptions and exit/extra descriptions do not.
	roomEditor bool
	// onComplete is used by the bounded room OLC string fields. The improved
	// editor has one descriptor-owned line buffer, but its save/abort target is
	// supplied by the owning OLC state rather than a text file.
	onComplete func(action textEditAction, buffer, original string)
}

type textEditAction uint8

const (
	textEditContinue textEditAction = iota
	textEditSave
	textEditAbort
)

// liveTextEditMu serializes access to the process-global text pointers
// represented by cachedText and World.HelpScreen. C's d->str points directly
// at those globals, so overlapping descriptors observe each other's edits.
// Ordinary help-screen reads take the shared lock; editor/reload mutations take
// the exclusive lock.
var liveTextEditMu sync.RWMutex

// cmdTedit ports do_tedit/general_file_edit. It intentionally owns only the
// finite C text-file editor; object, room, mob, shop, and zone OLC remain
// separate future work rather than being hidden behind a generic editor.
func cmdTedit(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}

	fieldName := ""
	if len(args) > 0 {
		// C's one_argument lowercases the field token before tedit's
		// case-sensitive strncmp table scan.
		fieldName = strings.ToLower(args[0])
	}
	if fieldName == "" {
		var out strings.Builder
		out.WriteString("Files available to be edited:\r\n")
		count := 0
		for _, field := range textEditFields {
			if s.player.GetLevel() < field.minLevel {
				continue
			}
			fmt.Fprintf(&out, "%-11.11s ", field.name)
			count++
			if count%6 == 0 {
				out.WriteString("\r\n")
			}
		}
		if count%6 != 0 {
			out.WriteString("\r\n")
		}
		if count == 0 {
			out.WriteString("None.\r\n")
		}
		s.sendTextEditor(out.String())
		return nil
	}

	var field *textEditField
	for i := range textEditFields {
		if strings.HasPrefix(textEditFields[i].name, fieldName) {
			field = &textEditFields[i]
			break
		}
	}
	if field == nil {
		s.sendTextEditor("Invalid text editor option.\r\n")
		return nil
	}
	if s.player.GetLevel() < field.minLevel {
		s.sendTextEditor("You are not godly enough for that!\r\n")
		return nil
	}
	return s.startTextEdit(*field)
}

func (s *Session) startTextEdit(field textEditField) error {
	s.textEditMu.Lock()
	liveTextEditMu.Lock()
	var text string
	var err error
	if field.filename == "help/screen" && s.manager.world.HelpScreen != "" {
		text = editorCRLF(s.manager.world.HelpScreen)
	} else {
		text, err = cachedTextForFile(s, field.filename)
		if err != nil {
			// C's boot-time file_to_string_alloc leaves a missing global NULL;
			// general_file_edit still opens an empty editor and permits a save.
			text = ""
		}
	}

	state := &textEditState{
		field:    field,
		path:     filepath.Join(s.manager.world.LibTextDir, field.filename),
		original: text,
		buffer:   text,
		cacheKey: field.filename,
	}
	s.textEdit = state
	liveTextEditMu.Unlock()
	s.textEditMu.Unlock()

	s.finishFileEditStart(state)
	return nil
}

// startFileEdit installs the common general_file_edit state used by tedit and
// luaedit. Keeping the editor entry in one path is important: the improved
// editor, descriptor state, writing flag, room act, and save cleanup are one C
// lifecycle even though the two commands choose different files.
func (s *Session) startFileEdit(state textEditState) error {
	s.textEditMu.Lock()
	liveTextEditMu.Lock()
	if s.textEdit != nil {
		liveTextEditMu.Unlock()
		s.textEditMu.Unlock()
		return fmt.Errorf("already editing a file")
	}
	s.textEdit = &state
	liveTextEditMu.Unlock()
	s.textEditMu.Unlock()

	s.finishFileEditStart(&state)
	return nil
}

func (s *Session) finishFileEditStart(state *textEditState) {
	s.sendTextEditor("Instructions: /s or @ to save, /h for more options.\r\n" +
		"Edit file below:\r\n\r\n" + state.buffer)
	game.Act(s.manager.world, true, s.player, nil, nil, nil,
		"$n begins editing a scroll.", "", game.ToRoom)
	s.player.SetPlrFlag(game.PlrWriting, true)
}

func (s *Session) sendTextEditor(text string) {
	if err := s.SendMessage(text); err != nil {
		slog.Error("tedit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) IsTextEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.textEdit != nil
}

func (s *Session) isTextEditing() bool {
	return s.IsTextEditing()
}

// handleTextEditInput is the CON_TEDIT/string_add route. It runs before the
// ordinary command interpreter so editor lines such as /h and @ cannot become
// game commands (R2/R5e).
func (s *Session) handleTextEditInput(line string) {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	s.handleTextEditInputLocked(line)
}

// handleTextEditInputLocked is the common CON_TEDIT/REDIT string_add route.
// The caller owns textEditMu; room OLC uses it while retaining its enclosing
// room state, so save/abort cannot race disconnect cleanup.
func (s *Session) handleTextEditInputLocked(line string) {
	if s.textEdit == nil {
		return
	}
	liveTextEditMu.Lock()
	defer liveTextEditMu.Unlock()
	if !s.textEdit.roomEditor {
		s.refreshTextEditBufferLocked()
	}

	line = editorSanitizeInput(line)
	if strings.HasPrefix(line, "@") {
		s.finishTextEditLocked(textEditSave)
		return
	}
	if strings.HasPrefix(line, "/") {
		switch s.executeTextEditorLocked(line) {
		case textEditSave:
			s.finishTextEditLocked(textEditSave)
		case textEditAbort:
			s.finishTextEditLocked(textEditAbort)
		}
		return
	}
	s.appendTextEditorLineLocked(line)
	s.commitTextEditBufferLocked()
}

func editorSanitizeInput(line string) string {
	// modify.c calls delete_doubledollar() followed by smash_tilde().
	var out strings.Builder
	for i := 0; i < len(line); i++ {
		if line[i] == '$' && i+1 < len(line) && line[i+1] == '$' {
			i++
		}
		if line[i] == '~' && (i+1 == len(line) || line[i+1] == '\r' || line[i+1] == '\n') {
			out.WriteByte(' ')
		} else {
			out.WriteByte(line[i])
		}
	}
	return out.String()
}

func (s *Session) appendTextEditorLineLocked(line string) {
	state := s.textEdit
	if state.buffer == "" {
		if len(line)+3 > state.field.maxBytes {
			if state.field.maxBytes >= 3 {
				line = line[:state.field.maxBytes-3] + "\r\n"
			}
			state.buffer = line
			s.sendTextEditor("String too long - Truncated.\r\n")
			return
		}
		state.buffer = line
	} else {
		if len(line)+len(state.buffer)+3 > state.field.maxBytes {
			s.sendTextEditor("String too long.  Last line skipped.\r\n")
			return
		}
		state.buffer += line
	}
	if len(state.buffer)+3 <= state.field.maxBytes {
		state.buffer += "\r\n"
	}
}

func (s *Session) finishTextEditLocked(action textEditAction) {
	state := s.textEdit
	if state == nil {
		return
	}
	if state.onComplete != nil {
		// OLC string_write callbacks (redit, medit D-description) return to
		// their owning menu. They do not emit tedit's file "Saved." or "Edit
		// aborted." text and they keep the descriptor in PLR_WRITING until
		// the OLC session itself exits.
		s.textEdit = nil
		state.onComplete(action, state.buffer, state.original)
		return
	}

	switch action {
	case textEditSave:
		// C's strip_string removes carriage returns in the same buffer that is
		// passed to fputs. That means both the disk file and the live global
		// become LF-delimited until a later boot/reload reconstructs CRLF.
		if state.killOnEmpty && state.buffer == "" {
			err := os.Remove(state.path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				slog.Error("luaedit delete failed", "player", s.playerName, "file", state.path, "error", err)
			} else {
				slog.Info(fmt.Sprintf("OLC: %s deletes '%s'.", s.playerName, state.path))
				s.sendTextEditor("Deleted.\r\n")
			}
		} else {
			savedText := strings.ReplaceAll(state.buffer, "\r", "")
			if err := os.WriteFile(state.path, []byte(savedText), 0o666); err != nil {
				slog.Error("file edit save failed", "player", s.playerName, "file", state.path, "error", err)
			} else {
				if state.cacheKey != "" {
					setTextEditCache(s, state.cacheKey, savedText)
				}
				slog.Info(fmt.Sprintf("OLC: %s saves '%s'.", s.playerName, state.path))
				s.sendTextEditor("Saved.\r\n")
			}
		}
	case textEditAbort:
		if state.cacheKey != "" {
			setTextEditCache(s, state.cacheKey, state.original)
		}
		s.sendTextEditor("Edit aborted.\r\n")
		game.Act(s.manager.world, true, s.player, nil, nil, nil,
			"$n stops editing some scrolls.", "", game.ToRoom)
	}

	// tedit_string_cleanup calls cleanup_olc after its own save/abort output;
	// cleanup_olc emits this second room-visible transition for both paths.
	s.textEdit = nil
	s.player.SetPlrFlag(game.PlrWriting, false)
	game.Act(s.manager.world, true, s.player, nil, nil, nil,
		"$n stops using OLC.", "", game.ToRoom)
}

func (s *Session) cancelTextEdit() {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.textEdit == nil {
		return
	}
	if s.textEdit.onComplete != nil {
		// cleanup_olc drops an active REDIT string buffer without invoking its
		// menu callback. cancelRoomEdit owns the enclosing OLC transition.
		s.textEdit = nil
		return
	}
	liveTextEditMu.Lock()
	defer liveTextEditMu.Unlock()
	// close_socket calls cleanup_olc directly, not tedit_string_cleanup. That
	// drops descriptor state but leaves the already-mutated boot-cache pointer
	// live; the unsaved bytes are therefore visible in memory but never reach
	// disk. The buffer has already been committed after each input line/action;
	// do not write this descriptor's stale snapshot back over another editor.
	s.textEdit = nil
	s.player.SetPlrFlag(game.PlrWriting, false)
	game.Act(s.manager.world, true, s.player, nil, nil, nil,
		"$n stops using OLC.", "", game.ToRoom)
}

func setTextEditCache(s *Session, filename, text string) {
	if filename == "help/screen" {
		s.manager.world.HelpScreen = text
		return
	}
	cacheMu.Lock()
	cachedText[filename] = text
	cacheMu.Unlock()
}

// refreshTextEditBufferLocked mirrors d->str: each descriptor action starts
// from the current process-global text, not from a private session copy. The
// caller holds liveTextEditMu and s.textEditMu.
func (s *Session) refreshTextEditBufferLocked() {
	state := s.textEdit
	if state == nil {
		return
	}
	if state.cacheKey == "" {
		return
	}
	if state.field.filename == "help/screen" {
		state.buffer = s.manager.world.HelpScreen
		return
	}
	cacheMu.RLock()
	if text, ok := cachedText[state.field.filename]; ok {
		state.buffer = text
	}
	cacheMu.RUnlock()
}

func (s *Session) commitTextEditBufferLocked() {
	state := s.textEdit
	if state != nil && !state.roomEditor && state.cacheKey != "" {
		setTextEditCache(s, state.cacheKey, state.buffer)
	}
}

func editorCRLF(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n\r", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.ReplaceAll(text, "\n", "\r\n")
}

func (s *Session) executeTextEditorLocked(line string) textEditAction {
	option := byte(0)
	if len(line) > 1 {
		option = line[1]
	}
	actions := ""
	if len(line) > 2 {
		actions = line[2:]
	}

	switch option {
	case 'a':
		return textEditAbort
	case 'c':
		if s.textEdit.buffer != "" {
			s.textEdit.buffer = ""
			s.sendTextEditor("Current buffer cleared.\r\n")
		} else {
			s.sendTextEditor("Current buffer empty.\r\n")
		}
	case 'd':
		s.parseTextEditorDelete(actions)
	case 'e':
		s.parseTextEditorEdit(actions)
	case 'f':
		if s.textEdit.buffer == "" {
			s.sendTextEditor("Current buffer empty.\r\n")
		} else {
			indent := false
			for i := 0; i < len(actions) && i < 2 && isASCIILetter(actions[i]); i++ {
				if actions[i] == 'i' && !indent {
					indent = true
				}
			}
			s.textEdit.buffer = formatTextEditor(s.textEdit.buffer, indent, s.textEdit.field.maxBytes)
			if indent {
				s.sendTextEditor("Text formatted with indent.\r\n")
			} else {
				s.sendTextEditor("Text formatted without indent.\r\n")
			}
		}
	case 'i':
		if s.textEdit.buffer == "" {
			s.sendTextEditor("Current buffer empty.\r\n")
		} else {
			s.parseTextEditorInsert(actions)
		}
	case 'h':
		s.sendTextEditor(textEditHelp)
	case 'l':
		if s.textEdit.buffer == "" {
			s.sendTextEditor("Current buffer empty.\r\n")
		} else {
			s.listTextEditor(actions, false)
		}
	case 'n':
		if s.textEdit.buffer == "" {
			s.sendTextEditor("Current buffer empty.\r\n")
		} else {
			s.listTextEditor(actions, true)
		}
	case 'r':
		s.parseTextEditorReplace(actions)
	case 's':
		return textEditSave
	default:
		s.sendTextEditor("Invalid option.\r\n")
	}
	s.commitTextEditBufferLocked()
	return textEditContinue
}

func (s *Session) parseTextEditorDelete(actions string) {
	low, high, ok := parseEditorRange(actions)
	if !ok {
		if strings.TrimSpace(actions) == "" {
			// sscanf returns EOF for an empty input in the oracle libc; the C
			// handler then falls through its uninitialized line-number branch.
			// Preserve that observed byte rather than normalizing it.
			s.sendTextEditor("Invalid, line numbers to delete must be higher than 0.\r\n")
		} else {
			// A non-empty malformed argument is sscanf's case 0.
			s.sendTextEditor("You must specify a line number or range to delete.\r\n")
		}
		return
	}
	if high < low {
		s.sendTextEditor("That range is invalid.\r\n")
		return
	}
	if low <= 0 {
		s.sendTextEditor("Invalid, line numbers to delete must be higher than 0.\r\n")
		return
	}
	if s.textEdit.buffer == "" {
		s.sendTextEditor("Buffer is empty.\r\n")
		return
	}
	starts := editorLineStarts(s.textEdit.buffer)
	if low > len(starts) {
		s.sendTextEditor("Line(s) out of range; not deleting.\r\n")
		return
	}
	endLine := high
	if endLine > len(starts) {
		endLine = len(starts)
	}
	start := starts[low-1]
	end := len(s.textEdit.buffer)
	if endLine < len(starts) {
		end = starts[endLine]
	}
	count := 0
	for i := low - 1; i < endLine && i < len(starts); i++ {
		if strings.Contains(s.textEdit.buffer[starts[i]:editorLineEnd(s.textEdit.buffer, starts, i)], "\n") {
			count++
		}
	}
	s.textEdit.buffer = s.textEdit.buffer[:start] + s.textEdit.buffer[end:]
	s.sendTextEditor(fmt.Sprintf("%d line%sdeleted.\r\n", count, pluralEditorSpace(count)))
}

func (s *Session) parseTextEditorInsert(actions string) {
	lineText, text := oneSpaceHalfChopGo(actions)
	if lineText == "" {
		s.sendTextEditor("You must specify a line number before which to insert text.\r\n")
		return
	}
	lineNumber := atoiEditor(lineText)
	text += "\r\n"
	starts := editorLineStarts(s.textEdit.buffer)
	if lineNumber <= 0 {
		s.sendTextEditor("Line number must be higher than 0.\r\n")
		return
	}
	if s.textEdit.buffer == "" {
		s.sendTextEditor("Buffer is empty, nowhere to insert.\r\n")
		return
	}
	if lineNumber > len(starts) {
		s.sendTextEditor("Line number out of range; insert aborted.\r\n")
		return
	}
	pos := starts[lineNumber-1]
	updated := s.textEdit.buffer[:pos] + text + s.textEdit.buffer[pos:]
	if len(updated) > s.textEdit.field.maxBytes {
		s.sendTextEditor("Insert text pushes buffer over maximum size, insert aborted.\r\n")
		return
	}
	s.textEdit.buffer = updated
	s.sendTextEditor("Line inserted.\r\n")
}

func (s *Session) parseTextEditorEdit(actions string) {
	lineText, text := oneSpaceHalfChopGo(actions)
	if lineText == "" {
		s.sendTextEditor("You must specify a line number at which to change text.\r\n")
		return
	}
	lineNumber := atoiEditor(lineText)
	text += "\r\n"
	starts := editorLineStarts(s.textEdit.buffer)
	if lineNumber <= 0 {
		s.sendTextEditor("Line number must be higher than 0.\r\n")
		return
	}
	if s.textEdit.buffer == "" {
		s.sendTextEditor("Buffer is empty, nothing to change.\r\n")
		return
	}
	if lineNumber > len(starts) {
		s.sendTextEditor("Line number out of range; change aborted.\r\n")
		return
	}
	lineIndex := lineNumber - 1
	start := starts[lineIndex]
	end := editorLineEnd(s.textEdit.buffer, starts, lineIndex)
	updated := s.textEdit.buffer[:start] + text + s.textEdit.buffer[end:]
	if len(updated) > s.textEdit.field.maxBytes {
		s.sendTextEditor("Change causes new length to exceed buffer maximum size, aborted.\r\n")
		return
	}
	s.textEdit.buffer = updated
	s.sendTextEditor("Line changed.\r\n")
}

func (s *Session) listTextEditor(actions string, numbered bool) {
	low, high, ok := parseEditorRange(actions)
	if !ok {
		low, high = 1, 999999
	}
	if low < 1 {
		s.sendTextEditor("Line numbers must be greater than 0.\r\n")
		return
	}
	if high < low {
		s.sendTextEditor("That range is invalid.\r\n")
		return
	}
	starts := editorLineStarts(s.textEdit.buffer)
	if low > len(starts) {
		s.sendTextEditor("Line(s) out of range; no buffer listing.\r\n")
		return
	}
	var out strings.Builder
	if !numbered && (high < 999999 || low > 1) {
		fmt.Fprintf(&out, "Current buffer range [%d - %d]:\r\n", low, high)
	}
	if numbered {
		// improved-edit.c's numbered path uses sprintf(buf, "%s...", buf,
		// ...), with overlapping source and destination. On the oracle libc a
		// multi-line range retains only its final numbered line; preserve that
		// observed C behavior instead of silently presenting a cleaner list.
		lastActual := len(starts)
		if strings.HasSuffix(s.textEdit.buffer, "\n") {
			lastActual--
		}
		target := low
		if high > low && high < lastActual {
			target = high
		} else if high > low && high >= lastActual {
			target = lastActual
		}
		if target >= low && target >= 1 && target <= len(starts) {
			start := starts[target-1]
			end := editorLineEnd(s.textEdit.buffer, starts, target-1)
			line := s.textEdit.buffer[start:end]
			if strings.Contains(line, "\n") {
				fmt.Fprintf(&out, "%4d:\r\n", target)
				out.WriteString(line)
			}
		}
		s.sendTextEditor(out.String())
		return
	}
	last := high
	if last > len(starts) {
		last = len(starts)
	}
	for i := low - 1; i < last; i++ {
		start := starts[i]
		end := editorLineEnd(s.textEdit.buffer, starts, i)
		out.WriteString(s.textEdit.buffer[start:end])
	}
	shown := 0
	for i := low - 1; i < last; i++ {
		if strings.Contains(s.textEdit.buffer[starts[i]:editorLineEnd(s.textEdit.buffer, starts, i)], "\n") {
			shown++
		}
	}
	fmt.Fprintf(&out, "\r\n%d line%sshown.\r\n", shown, pluralEditorSpace(shown))
	s.sendTextEditor(out.String())
}

func (s *Session) parseTextEditorReplace(actions string) {
	repAll := len(actions) > 0 && actions[0] == 'a'
	tokens := editorQuoteTokens(actions)
	if len(tokens) == 0 {
		s.sendTextEditor("Invalid format.\r\n")
		return
	}
	if len(tokens) < 2 {
		s.sendTextEditor("Target string must be enclosed in single quotes.\r\n")
		return
	}
	if len(tokens) < 3 {
		s.sendTextEditor("No replacement string.\r\n")
		return
	}
	if len(tokens) < 4 {
		s.sendTextEditor("Replacement string must be enclosed in single quotes.\r\n")
		return
	}
	pattern := tokens[1]
	replacement := tokens[3]
	if s.textEdit.buffer == "" {
		return
	}

	// improved-edit.c evaluates this expression as unsigned size_t. For a
	// shorter replacement, the subtraction wraps before the existing buffer
	// length is added, yielding the ordinary resulting length.
	totalLen := uint(len(replacement)) - uint(len(pattern)) + uint(len(s.textEdit.buffer))
	if totalLen > uint(s.textEdit.field.maxBytes) {
		s.sendTextEditor("Not enough space left in buffer.\r\n")
		return
	}
	updated, count := replaceEditorString(s.textEdit.buffer, pattern, replacement, repAll, s.textEdit.field.maxBytes)
	if count < 0 {
		s.sendTextEditor("ERROR: Replacement string causes buffer overflow, aborted replace.\r\n")
	} else if count == 0 {
		s.textEdit.buffer = updated
		s.sendTextEditor(fmt.Sprintf("String '%s' not found.\r\n", pattern))
	} else {
		s.textEdit.buffer = updated
		s.sendTextEditor(fmt.Sprintf("Replaced %d occurance%sof '%s' with '%s'.\r\n",
			count, pluralEditorSpace(count), pattern, replacement))
	}
}

// editorQuoteTokens mirrors strtok(actions, "'"): delimiters are removed and
// adjacent/leading/trailing delimiters do not produce empty tokens.
func editorQuoteTokens(actions string) []string {
	return strings.FieldsFunc(actions, func(r rune) bool { return r == '\'' })
}

func replaceEditorString(input, pattern, replacement string, all bool, maxBytes int) (string, int) {
	if pattern == "" {
		return input, 0
	}
	if !all {
		idx := strings.Index(input, pattern)
		if idx < 0 {
			return input, 0
		}
		return input[:idx] + replacement + input[idx+len(pattern):], 1
	}
	count := strings.Count(input, pattern)
	if count == 0 {
		return input, 0
	}
	var out strings.Builder
	flow := input
	flowOffset := 0
	for {
		idx := strings.Index(flow, pattern)
		if idx < 0 {
			out.WriteString(flow)
			break
		}
		// improved-edit.c temporarily NUL-terminates flow at the match before
		// measuring jetsam. strlen(jetsam) is therefore idx, the prefix before
		// the current match, not the untouched suffix after it. out already holds
		// the prefixes and replacements accepted by earlier iterations. If this
		// inner guard trips, C sets i to -1 but the later i <= 0 branch returns 0,
		// so the caller reports "not found". C does not restore the temporary
		// NUL on this break, so d->str is left truncated at the failed match.
		if maxBytes > 0 && out.Len()+idx+len(replacement) > maxBytes {
			return input[:flowOffset+idx], 0
		}
		out.WriteString(flow[:idx])
		out.WriteString(replacement)
		flowOffset += idx + len(pattern)
		flow = flow[idx+len(pattern):]
	}
	return out.String(), count
}

func formatTextEditor(input string, indent bool, maxBytes int) string {
	flow := []byte(input)
	var out strings.Builder
	lineChars := 0
	capNext, capNextNext := true, false
	if indent {
		out.WriteString("   ")
		lineChars = 3
	}
	for pos := 0; pos < len(flow); {
		for pos < len(flow) && isEditorFormatSpace(flow[pos]) {
			pos++
		}
		if pos >= len(flow) {
			break
		}
		start := pos
		pos++
		for pos < len(flow) && !isEditorFormatStop(flow[pos]) {
			pos++
		}
		if capNextNext {
			capNextNext = false
			capNext = true
		}
		for pos < len(flow) && isEditorSentenceStop(flow[pos]) {
			capNextNext = true
			pos++
		}
		word := append([]byte(nil), flow[start:pos]...)
		if lineChars+len(word)+1 > 78 {
			out.WriteString("\r\n")
			lineChars = 0
		}
		if !capNext {
			if lineChars > 0 {
				out.WriteByte(' ')
				lineChars++
			}
		} else {
			capNext = false
			if len(word) > 0 && word[0] >= 'a' && word[0] <= 'z' {
				word[0] -= 'a' - 'A'
			}
		}
		lineChars += len(word)
		out.Write(word)
		if capNextNext && pos < len(flow) {
			if lineChars+3 > 78 {
				out.WriteString("\r\n")
				lineChars = 0
			} else {
				out.WriteString("  ")
				lineChars += 2
			}
		}
	}
	out.WriteString("\r\n")
	formatted := out.String()
	if len(formatted)+1 > maxBytes && maxBytes > 0 {
		formatted = formatted[:maxBytes-1]
	}
	return formatted
}

func isEditorFormatSpace(b byte) bool {
	switch b {
	case '\n', '\r', '\f', '\t', '\v', ' ':
		return true
	default:
		return false
	}
}

func isEditorFormatStop(b byte) bool {
	return isEditorFormatSpace(b) || isEditorSentenceStop(b)
}

func isEditorSentenceStop(b byte) bool {
	return b == '.' || b == '!' || b == '?'
}

func isASCIILetter(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

func editorLineStarts(buffer string) []int {
	starts := []int{0}
	for i := 0; i < len(buffer); i++ {
		if buffer[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

func editorLineEnd(buffer string, starts []int, index int) int {
	if index+1 < len(starts) {
		return starts[index+1]
	}
	return len(buffer)
}

func parseEditorRange(input string) (low, high int, ok bool) {
	trimmed := strings.TrimLeft(input, " \t\r\n\v\f")
	if trimmed == "" {
		return 0, 0, false
	}
	first, used := parseEditorInt(trimmed)
	if used == 0 {
		return 0, 0, false
	}
	rest := strings.TrimLeft(trimmed[used:], " \t\r\n\v\f")
	if strings.HasPrefix(rest, "-") {
		second, secondUsed := parseEditorInt(strings.TrimLeft(rest[1:], " \t\r\n\v\f"))
		if secondUsed > 0 {
			return first, second, true
		}
	}
	return first, first, true
}

func parseEditorInt(input string) (int, int) {
	if input == "" {
		return 0, 0
	}
	end := 0
	if input[0] == '+' || input[0] == '-' {
		end++
	}
	startDigits := end
	for end < len(input) && input[end] >= '0' && input[end] <= '9' {
		end++
	}
	if end == startDigits {
		return 0, 0
	}
	value, err := strconv.Atoi(input[:end])
	if err != nil {
		return 0, end
	}
	return value, end
}

func oneSpaceHalfChopGo(input string) (token, rest string) {
	input = strings.TrimLeft(input, " \t\r\n\v\f")
	end := 0
	for end < len(input) && !isEditorFormatSpace(input[end]) {
		end++
	}
	token = input[:end]
	if end < len(input) {
		end++ // interpreter.c: one_space_half_chop skips exactly one space
	}
	return token, input[end:]
}

func atoiEditor(input string) int {
	value, _ := parseEditorInt(input)
	return value
}

func pluralEditorSpace(count int) string {
	if count != 1 {
		return "s "
	}
	return " "
}
