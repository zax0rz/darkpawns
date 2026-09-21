package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestTeditRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["tedit"]
	if !ok {
		t.Fatal("tedit command has no C gate")
	}
	if gate.MinLevel != game.LVL_GRGOD || gate.MinPosition != combat.PosDead {
		t.Fatalf("tedit gate = (%d,%d), want (%d,%d)",
			gate.MinLevel, gate.MinPosition, game.LVL_GRGOD, combat.PosDead)
	}
	entry, ok := cmdRegistry.Lookup("tedit")
	if !ok {
		t.Fatal("tedit command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("tedit registry gate = (%d,%d), want (%d,%d)",
			entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestTeditAuthorizationAndCaseNormalization(t *testing.T) {
	m := makeTestManager(t)

	// The command row is hidden from mortals by interpreter.c before the
	// handler can reveal which text fields exist.
	mortal := makeCommandTestSession(t, m, "Teditmortal", game.LVL_GOD, 1001)
	if err := ExecuteCommand(mortal, "tedit", nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, mortal); got != "Huh?!?\r\n" {
		t.Fatalf("mortal tedit gate = %q", got)
	}

	// At the command gate, a GRGOD actor may enter tedit, but the credits
	// field remains reserved for LVL_IMPL in src/tedit.c's field table.
	grgod := makeCommandTestSession(t, m, "Teditgrgod", game.LVL_GRGOD, 1001)
	if err := cmdTedit(grgod, []string{"credits"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, grgod); got != "You are not godly enough for that!\r\n" {
		t.Fatalf("credits field authorization = %q", got)
	}

	if err := cmdTedit(grgod, []string{"NEWS"}); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, grgod); !strings.HasPrefix(got, "Instructions: /s or @ to save") {
		t.Fatalf("one_argument-style case normalization = %q", got)
	}
	grgod.cancelTextEdit()
}

func TestReadCTextFileUsesBootCRLFRepresentation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "text")
	if err := os.WriteFile(path, []byte("first\nsecond\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readCTextFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "first\r\nsecond\r\n" {
		t.Fatalf("readCTextFile = %q, want CRLF boot buffer", got)
	}
}

func TestReadCTextFileDropsShortFinalFragmentAtEOF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "text")
	if err := os.WriteFile(path, []byte("first\nsecond"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readCTextFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "first\r\n" {
		t.Fatalf("readCTextFile = %q, want C's dropped final fragment", got)
	}
}

func TestTeditAbortRestoresCacheAndLeavesDiskUntouched(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Teditgod", game.LVL_IMPL, 1001)
	path := filepath.Join(t.TempDir(), "news")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(path)
	setTeditTestCache(t, "news", "")

	if err := cmdTedit(s, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	start := readMsgText(t, s)
	if !strings.Contains(start, "Instructions: /s or @ to save, /h for more options.\r\n") {
		t.Fatalf("start output missing editor instructions: %q", start)
	}
	s.handleTextEditInput("draft line")
	s.handleTextEditInput("/a")
	if got := readMsgText(t, s); got != "Edit aborted.\r\n" {
		t.Fatalf("abort output = %q", got)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "old line\n" {
		t.Fatalf("disk after abort = %q, err=%v", got, err)
	}
	cacheMu.RLock()
	got := cachedText["news"]
	cacheMu.RUnlock()
	if got != "old line\r\n" {
		t.Fatalf("cache after abort = %q", got)
	}
	if s.isTextEditing() || s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
		t.Fatal("abort left the descriptor in writing state")
	}
}

func TestTeditSaveUpdatesDiskCacheAndStaticCommand(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Teditgod", game.LVL_IMPL, 1001)
	path := filepath.Join(t.TempDir(), "news")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(path)
	setTeditTestCache(t, "news", "")

	if err := cmdTedit(s, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleTextEditInput("new line")
	s.handleTextEditInput("/s")
	if got := readMsgText(t, s); got != "Saved.\r\n" {
		t.Fatalf("save output = %q", got)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != "old line\nnew line\n" {
		t.Fatalf("disk after save = %q, err=%v", got, err)
	}
	cacheMu.RLock()
	got := cachedText["news"]
	cacheMu.RUnlock()
	if got != "old line\nnew line\n" {
		t.Fatalf("live cache after C strip_string = %q", got)
	}
	if err := cmdNews(s, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != "old line\nnew line\n" {
		t.Fatalf("news after save from live cache = %q", got)
	}

	// Drop the live cache and read through a fresh manager/session to prove the
	// persisted LF file is a valid C-style boot reload, not just a live-cache
	// mutation on the saving descriptor.
	cacheMu.Lock()
	delete(cachedText, "news")
	cacheMu.Unlock()
	reloadedManager := makeTestManager(t)
	reloadedManager.world.LibTextDir = filepath.Dir(path)
	reloaded := makeCommandTestSession(t, reloadedManager, "Teditreload", game.LVL_IMPL, 1001)
	if err := cmdNews(reloaded, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, reloaded); got != "old line\r\nnew line\r\n" {
		t.Fatalf("news after disk reload = %q", got)
	}
}

func TestTeditRawLineRoutesSlashInputBeforeCommands(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Teditgod", game.LVL_IMPL, 1001)
	path := filepath.Join(t.TempDir(), "news")
	if err := os.WriteFile(path, []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(path)
	setTeditTestCache(t, "news", "")
	if err := cmdTedit(s, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)

	data, err := json.Marshal(CommandData{Command: "/", Args: []string{"h"}, RawLine: "/h"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleCommand(data); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); !strings.HasPrefix(got, "Editor command formats: /<letter>\r\n") {
		t.Fatalf("raw /h was not routed to editor: %q", got)
	}
}

func TestTeditBlankLineAppendsToSharedBuffer(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Teditblank", game.LVL_IMPL, 1001)
	path := filepath.Join(t.TempDir(), "news")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(path)
	setTeditTestCache(t, "news", "")

	if err := cmdTedit(s, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleTextEditInput("")
	cacheMu.RLock()
	got := cachedText["news"]
	cacheMu.RUnlock()
	if got != "old line\r\n\r\n" {
		t.Fatalf("blank editor line = %q, want shared CRLF blank line", got)
	}
	s.cancelTextEdit()
}

func TestTeditOverlappingEditorsShareLiveBufferAndBackstr(t *testing.T) {
	m := makeTestManager(t)
	first := makeCommandTestSession(t, m, "Teditfirst", game.LVL_IMPL, 1001)
	second := makeCommandTestSession(t, m, "Teditsecond", game.LVL_IMPL, 1001)
	observer := makeCommandTestSession(t, m, "Teditobserver", 1, 1001)
	path := filepath.Join(t.TempDir(), "news")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(path)
	setTeditTestCache(t, "news", "")

	if err := cmdTedit(first, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, first)
	first.handleTextEditInput("first live line")

	if err := cmdNews(observer, nil); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, observer); got != "old line\r\nfirst live line\r\n" {
		t.Fatalf("observer news during edit = %q", got)
	}

	if err := cmdTedit(second, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, second)
	second.handleTextEditInput("second live line")
	if got := textEditTestCache("news"); got != "old line\r\nfirst live line\r\nsecond live line\r\n" {
		t.Fatalf("overlapping live buffer = %q", got)
	}

	// C's second descriptor restores its own backstr on abort, even though the
	// first descriptor is still in CON_TEDIT against the same global pointer.
	second.handleTextEditInput("/a")
	if got := textEditTestCache("news"); got != "old line\r\nfirst live line\r\n" {
		t.Fatalf("second abort restored = %q", got)
	}
	first.handleTextEditInput("/a")
	if got := textEditTestCache("news"); got != "old line\r\n" {
		t.Fatalf("first abort restored = %q", got)
	}
}

func TestTeditReplacementMatchesCUnsignedSizeAndStrtokEdges(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Teditreplace", game.LVL_IMPL, 1001)
	s.textEdit = &textEditState{
		field:  textEditFields[1],
		buffer: "Lots\r\n",
	}

	s.parseTextEditorReplace(" 'Lots' 'Few'")
	if got := readMsgText(t, s); got != "Replaced 1 occurance of 'Lots' with 'Few'.\r\n" {
		t.Fatalf("short replacement output = %q", got)
	}
	if s.textEdit.buffer != "Few\r\n" {
		t.Fatalf("short replacement buffer = %q", s.textEdit.buffer)
	}

	for _, test := range []struct {
		actions string
		want    string
	}{
		{actions: "bad", want: "Target string must be enclosed in single quotes.\r\n"},
		{actions: " 'Few'", want: "No replacement string.\r\n"},
		{actions: " 'Few' bad", want: "Replacement string must be enclosed in single quotes.\r\n"},
	} {
		s.parseTextEditorReplace(test.actions)
		if got := readMsgText(t, s); got != test.want {
			t.Errorf("replace %q = %q, want %q", test.actions, got, test.want)
		}
	}

	s.textEdit.buffer = "aaaa\r\n"
	s.textEdit.field.maxBytes = 10
	s.parseTextEditorReplace("a 'a' 'bbb'")
	if got := readMsgText(t, s); got != "String 'a' not found.\r\n" {
		t.Fatalf("replace-all overflow output = %q", got)
	}
	if s.textEdit.buffer != "aaa" {
		t.Fatalf("replace-all overflow buffer = %q, want C's failed-match truncation", s.textEdit.buffer)
	}

	// C NUL-terminates each match before its inner size check, so only the
	// prefix before the match contributes to that check. The exact final result
	// fits and /ra succeeds at this boundary.
	s.textEdit.buffer = "aab"
	s.textEdit.field.maxBytes = 5
	s.parseTextEditorReplace("a 'a' 'bb'")
	if got := readMsgText(t, s); got != "Replaced 2 occurances of 'a' with 'bb'.\r\n" {
		t.Fatalf("replace-all prefix boundary output = %q", got)
	}
	if s.textEdit.buffer != "bbbbb" {
		t.Fatalf("replace-all prefix boundary buffer = %q", s.textEdit.buffer)
	}

	// This is the compiled-C boundary reported for the editor: replacing the
	// final character in xxa with a one-byte replacement succeeds with max=4,
	// even though the untouched suffix would make Go's old check reject it.
	s.textEdit.buffer = "xxa"
	s.textEdit.field.maxBytes = 4
	s.parseTextEditorReplace("a 'a' 'b'")
	if got := readMsgText(t, s); got != "Replaced 1 occurance of 'a' with 'b'.\r\n" {
		t.Fatalf("replace-all trailing-match boundary output = %q", got)
	}
	if s.textEdit.buffer != "xxb" {
		t.Fatalf("replace-all trailing-match boundary buffer = %q", s.textEdit.buffer)
	}
	s.textEdit = nil
}

func TestTeditHelpScreenSynchronizesWithHelpAndReload(t *testing.T) {
	m := makeTestManager(t)
	editor := makeCommandTestSession(t, m, "Tedithelp", game.LVL_IMPL, 1001)
	reader := makeCommandTestSession(t, m, "Helpreader", game.LVL_IMPL, 1001)
	reloader := makeCommandTestSession(t, m, "Helpreloader", game.LVL_IMPL, 1001)

	helpDir := filepath.Join(t.TempDir(), "help")
	if err := os.MkdirAll(helpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(helpDir, "screen"), []byte("disk help\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(helpDir)
	m.world.HelpScreen = "initial help\r\n"

	if err := cmdTedit(editor, []string{"help"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, editor)

	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			if err := cmdHelp(reader, nil); err != nil {
				t.Errorf("help read failed: %v", err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			reloadHelpScreen(reloader)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			editor.handleTextEditInput("draft")
		}
	}()
	wg.Wait()
	editor.cancelTextEdit()
}

func textEditTestCache(filename string) string {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return cachedText[filename]
}

func TestTeditDisconnectDropsDescriptorWithoutSaving(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Teditgod", game.LVL_IMPL, 1001)
	path := filepath.Join(t.TempDir(), "news")
	if err := os.WriteFile(path, []byte("old line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.world.LibTextDir = filepath.Dir(path)
	setTeditTestCache(t, "news", "")
	if err := cmdTedit(s, []string{"news"}); err != nil {
		t.Fatal(err)
	}
	_ = readMsgText(t, s)
	s.handleTextEditInput("unsaved line")
	s.cancelTextEdit()
	if got, err := os.ReadFile(path); err != nil || string(got) != "old line\n" {
		t.Fatalf("disk after disconnect cleanup = %q, err=%v", got, err)
	}
	cacheMu.RLock()
	got := cachedText["news"]
	cacheMu.RUnlock()
	if got != "old line\r\nunsaved line\r\n" {
		t.Fatalf("cache after disconnect cleanup = %q", got)
	}
	if s.isTextEditing() || s.player.GetFlags()&(1<<uint(game.PlrWriting)) != 0 {
		t.Fatal("disconnect cleanup left the descriptor in writing state")
	}
}

func setTeditTestCache(t *testing.T, filename, value string) {
	t.Helper()
	cacheMu.Lock()
	previous, existed := cachedText[filename]
	if value == "" {
		delete(cachedText, filename)
	} else {
		cachedText[filename] = value
	}
	cacheMu.Unlock()
	t.Cleanup(func() {
		cacheMu.Lock()
		if existed {
			cachedText[filename] = previous
		} else {
			delete(cachedText, filename)
		}
		cacheMu.Unlock()
	})
}
