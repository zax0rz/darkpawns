package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeExtraPresenceWorld writes seed as zone 30's object file, parses it the
// way boot does, and builds a world around the result. The writer under test
// saves back to the same SourceDir/obj/30.obj path.
func makeExtraPresenceWorld(t *testing.T, seed string) *game.World {
	t.Helper()
	dir := t.TempDir()
	objDir := filepath.Join(dir, "obj")
	if err := os.MkdirAll(objDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(objDir, "30.obj")
	if err := os.WriteFile(path, []byte(seed), 0o666); err != nil {
		t.Fatal(err)
	}
	objs, err := parser.ParseObjFile(path)
	if err != nil {
		t.Fatalf("ParseObjFile: %v", err)
	}
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Presence Room"},
			{VNum: 3000, Name: "Oedit Test Room", Zone: 30},
		},
		Zones:     []parser.Zone{{Number: 30, Name: "Oedit Test Zone", TopRoom: 3099}},
		Objs:      objs,
		SourceDir: dir,
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func readPresenceObjFile(t *testing.T, w *game.World) string {
	t.Helper()
	parsed := w.GetParsedWorld()
	data, err := os.ReadFile(filepath.Join(parsed.SourceDir, "obj", "30.obj"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// zoneLoadedNullSeed is the retained C reproduction's shape
// (docs/fidelity/depth/handoff/2026-10-07-dp-1371-oedit-null-C.py): an empty
// keyword field, which fread_string loads as a NULL pointer (src/db.c:2736-2740).
const zoneLoadedNullSeed = "#3001\n" +
	"sword rusty~\n" +
	"a rusty sword~\n" +
	"A rusty sword lies here.~\n" +
	"~\n" +
	"5 0 0 0 0 8193 0 0 0\n" +
	"0 3 4 11\n" +
	"5 100 50.00\n" +
	"E\n" +
	"~\n" +
	"Broken description~\n" +
	"$~\n"

const zonePlainSeed = "#3001\n" +
	"sword rusty~\n" +
	"a rusty sword~\n" +
	"A rusty sword lies here.~\n" +
	"~\n" +
	"5 0 0 0 0 8193 0 0 0\n" +
	"0 3 4 11\n" +
	"5 100 50.00\n" +
	"$~\n"

const corruptExDescPayload = "SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!"

// TestOeditSaveLoadedNullExtraLogsAndSkips is the DP-1417 producer proof: a
// node whose keyword pointer is NULL (an empty tilde field at load) gets C's
// BRF/LVL_BUILDER/file-TRUE diagnostic and is skipped, while the object itself
// is still written.
func TestOeditSaveLoadedNullExtraLogsAndSkips(t *testing.T) {
	w := makeExtraPresenceWorld(t, zoneLoadedNullSeed)
	m := newTestManager(t, w, nil)
	builder := makeOeditTestSession(t, m, "Nullbuilder", 35)
	watch := mudlogObserver(t, m, "Nullwatch", game.LVL_IMMORT, game.PrfLog1)
	below := mudlogObserver(t, m, "Nullbelow", game.LVL_IMMORT-1, game.PrfLog1)
	// BRF is the lowest syslog type, so the type filter's negative is an
	// observer with syslog off entirely.
	noSyslog := mudlogObserver(t, m, "Nullnosyslog", game.LVL_IMMORT)
	file := captureMudlogFile(t)
	drainSessionText(t, builder)

	if err := cmdOedit(builder, []string{"save", "30"}); err != nil {
		t.Fatalf("oedit save 30: %v", err)
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "[ "+corruptExDescPayload+" ]\r\n" {
		t.Fatalf("observer bytes = %q, want the BRF producer", got)
	}
	if !strings.Contains(file.String(), corruptExDescPayload) {
		t.Fatalf("file flag must be TRUE; log file = %q", file.String())
	}
	if got := strings.Join(drainSessionText(t, below), ""); got != "" {
		t.Fatalf("below-threshold observer saw %q", got)
	}
	if got := strings.Join(drainSessionText(t, noSyslog), ""); got != "" {
		t.Fatalf("syslog-off observer saw %q", got)
	}
	saved := readPresenceObjFile(t, w)
	if strings.Contains(saved, "Broken description") {
		t.Fatalf("the corrupt record must be skipped: %q", saved)
	}
	if !strings.Contains(saved, "#3001\n") {
		t.Fatalf("the object itself must still be written: %q", saved)
	}
}

// driveAllocatedEmptyExtra reproduces the retained C experiment's sequence
// (2026-10-07-dp-1371-oedit-null-C.py): F -> keyword -> description ->
// clear -> one line -> /d1 -> /s -> menu 0 -> q -> y. Deleting the last line
// leaves an allocated empty string, which C writes as "E\n<kw>~\n~\n".
func driveAllocatedEmptyExtra(t *testing.T, s *Session) {
	t.Helper()
	for _, line := range []string{"f", "1", "allocated", "2", "/c", "single line", "/d1", "/s", "0", "q", "y"} {
		s.handleOeditInput(line)
		drainSessionText(t, s)
	}
	if s.oedit != nil {
		t.Fatal("the editor must exit after the internal save")
	}
}

// TestOeditSaveEditedEmptyExtraWrites is the other half of the producer proof:
// an editor-allocated empty description is valid in C, so it must be written
// and must NOT produce the corrupt diagnostic.
func TestOeditSaveEditedEmptyExtraWrites(t *testing.T) {
	w := makeExtraPresenceWorld(t, zonePlainSeed)
	m := newTestManager(t, w, nil)
	builder := makeOeditTestSession(t, m, "Emptybuilder", 35)
	watch := mudlogObserver(t, m, "Emptywatch", game.LVL_IMMORT, game.PrfLog1)
	file := captureMudlogFile(t)

	openOedit(t, builder, "3001")
	driveAllocatedEmptyExtra(t, builder)

	if err := cmdOedit(builder, []string{"save", "30"}); err != nil {
		t.Fatalf("oedit save 30: %v", err)
	}
	saved := readPresenceObjFile(t, w)
	if !strings.Contains(saved, "E\nallocated~\n~\n") {
		t.Fatalf("the allocated-empty record must be written: %q", saved)
	}
	if got := strings.Join(drainSessionText(t, watch), ""); got != "" {
		t.Fatalf("no corrupt diagnostic for an allocated-empty node, got %q", got)
	}
	if strings.Contains(file.String(), corruptExDescPayload) {
		t.Fatalf("corrupt diagnostic reached the file: %q", file.String())
	}
}

// TestOeditReeditAllocatedEmptyIsSetNotNone pins the per-field presence: after
// the commit, re-entering the editor must show the empty description as set
// (C's pointer is non-NULL), not as "<NONE>" / "Not set".
func TestOeditReeditAllocatedEmptyIsSetNotNone(t *testing.T) {
	w := makeExtraPresenceWorld(t, zonePlainSeed)
	m := newTestManager(t, w, nil)
	builder := makeOeditTestSession(t, m, "Reeditbuilder", 35)

	openOedit(t, builder, "3001")
	driveAllocatedEmptyExtra(t, builder)
	drainSessionText(t, builder)

	openOedit(t, builder, "3001")
	builder.handleOeditInput("f")
	menu := readMsgText(t, builder)
	if !strings.Contains(menu, "1) Keyword: allocated\r\n") {
		t.Fatalf("re-edited keyword = %q, want the committed keyword", menu)
	}
	if !strings.Contains(menu, "2) Description:\r\n\r\n") {
		t.Fatalf("re-edited description must render as set but empty: %q", menu)
	}
	if strings.Contains(menu, "<NONE>") {
		t.Fatalf("a committed allocated-empty node must not render as unset: %q", menu)
	}
}

// TestStringObjExtraDescNullsPointerUntilWritten pins C's do_string field-4
// rule: opening the description editor frees and NULLs the pointer
// (src/modify.c:686-709), so field 6 still reports "No field with that keyword"
// (src/modify.c:718) until the editor writes a buffer. A written buffer is
// allocated, so even an empty one is present.
func TestStringObjExtraDescNullsPointerUntilWritten(t *testing.T) {
	w := makeExtraPresenceWorld(t, zonePlainSeed)
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Stringpresence", 35, 3000)
	obj, err := w.SpawnObject(3001, 3000)
	if err != nil {
		t.Fatal(err)
	}

	// An existing field is freed before the editor opens.
	obj.SetLiveExtraDescs([]parser.ExtraDesc{{Keywords: "altar", Description: "an altar"}})
	if err := s.stringObjExtraDescEditor(obj, "altar"); err != nil {
		t.Fatal(err)
	}
	if descs := obj.LiveExtraDescs(); len(descs) != 1 || !descs[0].NullDescription {
		t.Fatalf("opened editor node = %#v, want a NULL description", descs)
	}
	drainSessionText(t, s)
	if err := s.stringObjDeleteExtraDesc(obj, "altar"); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, s); got != stringNoFieldWithKeyword {
		t.Fatalf("field 6 = %q, want C's not-found message", got)
	}

	// A new field starts with a NULL description too.
	if err := s.stringObjExtraDescEditor(obj, "shrine"); err != nil {
		t.Fatal(err)
	}
	if descs := obj.LiveExtraDescs(); !descs[0].NullDescription {
		t.Fatalf("new field node = %#v, want a NULL description", descs)
	}
}
