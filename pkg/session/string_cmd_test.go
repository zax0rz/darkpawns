package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// newStringTestSession builds an immortal standing in room 1001 of the shared
// mob/obj test world (mob 2001 is the "goblin guard", objs come from
// makeObjInstance). The player is registered in the world so world-scope target
// resolution behaves as it does for a logged-in immortal.
func newStringTestSession(t *testing.T, level int) (*Session, *Manager) {
	t.Helper()
	m := makeTestManagerWithMobs(t)
	s := makeCommandTestSession(t, m, "Stringtest", level, 1001)
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return s, m
}

// carriedObject puts one freshly built object in the actor's inventory, which is
// the only scope do_string searches (get_obj_in_list_vis over ch->carrying).
func carriedObject(t *testing.T, s *Session, keywords, shortDesc string) *game.ObjectInstance {
	t.Helper()
	obj := makeObjInstance(9001, shortDesc, keywords)
	if s.player.Inventory == nil {
		s.player.Inventory = game.NewInventory()
	}
	if err := s.player.Inventory.AddItem(obj); err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	return obj
}

// runString drives the transport entry point exactly as the command dispatcher
// does: the raw argument remainder plus its tokenized form.
func runString(t *testing.T, s *Session, line string) {
	t.Helper()
	if err := cmdStringText(s, strings.Fields(line), line); err != nil {
		t.Fatalf("cmdStringText(%q): %v", line, err)
	}
}

// TestStringRegistrationUsesTheCGate pins the interpreter.c:745 row.
func TestStringRegistrationUsesTheCGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("string")
	if !ok {
		t.Fatal("'string' command not found in registry")
	}
	if entry.MinLevel != LVL_IMMORT+1 {
		t.Errorf("string MinLevel = %d, want LVL_IMMORT+1 (%d)", entry.MinLevel, LVL_IMMORT+1)
	}
	if entry.MinPosition != combat.PosResting {
		t.Errorf("string MinPosition = %d, want POS_RESTING", entry.MinPosition)
	}
}

// TestStringLiteralsMatchC drives every non-editor message do_string can emit
// and compares the exact bytes, covering all three terminator styles: "\n\r",
// CRLF, and no terminator at all.
func TestStringLiteralsMatchC(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"no arguments", "", stringUsage},
		{"unknown type", "blah guard name x", stringUsage},
		{"empty field token", "mob guard", stringNoField},
		{"unknown field on a mob", "mob guard bogus x", stringUndefinedForMonsters},
		{"unknown field on an object", "obj sword bogus x", stringUndefinedForObjects},
		{"unknown mob target", "mob nobody name x", stringNoTargetMob},
		{"unknown object target", "obj nobody name x", stringNoTargetObj},
		{"npc title", "mob guard title x", stringMonstersNoTitles},
		{"object title is undefined", "obj sword title x", stringUndefinedForObjects},
		{"extra description without a keyword", "obj sword description", stringKeywordRequired},
		{"delete without a keyword", "obj sword delete-description", stringFieldNameRequired},
		{"delete with an unknown keyword", "obj sword delete-description nope", stringNoFieldWithKeyword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, m := newStringTestSession(t, LVL_IMPL)
			registerMob(t, m, 2001, 1001)
			carriedObject(t, s, "sword long iron", "An iron longsword")

			runString(t, s, tt.line)
			if got := readMsgText(t, s); got != tt.want {
				t.Errorf("%q output = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

// TestStringTypeTokenMatchesIsAbbrev pins C's is_abbrev direction for the type
// word (interpreter.c:1356): the token must be a non-empty prefix of "mob"/"obj".
// A longer word such as "mobs" is not a type and falls through to TP_ERROR —
// tp_error means the usage line, and field/name are never read (quad_arg returns
// before writing them, so do_string checks the type first).
func TestStringTypeTokenMatchesIsAbbrev(t *testing.T) {
	tests := []struct {
		argument string
		kind     int
	}{
		{"", stringTypeError},
		{"m guard name x", stringTypeMob},
		{"mo guard name x", stringTypeMob},
		{"mob guard name x", stringTypeMob},
		{"mobs guard name x", stringTypeError},
		{"mobx guard name x", stringTypeError},
		{"o bread name x", stringTypeObj},
		{"ob bread name x", stringTypeObj},
		{"obj bread name x", stringTypeObj},
		{"objs bread name x", stringTypeError},
		{"MOB guard name x", stringTypeMob},
		// one_argument lowercases the token, but is_abbrev still needs the token
		// to be a prefix of "obj": "object" is longer, so C rejects it here even
		// though the load command accepts "object".
		{"OBJECT bread name x", stringTypeError},
		{"blah guard name x", stringTypeError},
	}
	for _, tt := range tests {
		if got := parseStringQuadArg(tt.argument).Kind; got != tt.kind {
			t.Errorf("parseStringQuadArg(%q).Kind = %d, want %d", tt.argument, got, tt.kind)
		}
	}
}

// TestStringQuadArgSkipsFillWordsAndKeepsRawString covers one_argument's fill
// words (in/from/with/the/on/at/to) and the verbatim remainder that becomes the
// inline string.
func TestStringQuadArgSkipsFillWordsAndKeepsRawString(t *testing.T) {
	qa := parseStringQuadArg("mob in the guard long a  guard   stands watch.  ")
	if qa.Kind != stringTypeMob || qa.Name != "guard" || qa.Field != 3 {
		t.Fatalf("quad arg = %+v, want mob/guard/long(3)", qa)
	}
	// Interior spacing survives: C copies from the first non-space character to
	// the end of the line.
	if want := "a  guard   stands watch.  "; qa.Str != want {
		t.Errorf("inline string = %q, want %q", qa.Str, want)
	}

	// An omitted field token resolves to field 0, which is the only way C reaches
	// the "No field by that name" branch.
	if got := parseStringQuadArg("mob guard").Field; got != 0 {
		t.Errorf("field for a missing token = %d, want 0", got)
	}
	if got := parseStringQuadArg("mob guard bogus x").Field; got != -1 {
		t.Errorf("field for an unknown token = %d, want -1", got)
	}
}

// (src/whod.c:496-527) contract: a prefix match over the field table returning a
// 1-based index, 0 for an empty token, -1 for no match.
func TestStringFieldResolutionMatchesOldSearchBlock(t *testing.T) {
	tests := []struct {
		token string
		want  int
	}{
		{"", 0},
		{"n", 1},
		{"name", 1},
		{"s", 2},
		{"short", 2},
		{"l", 3},
		{"long", 3},
		{"d", 4},
		{"de", 4},   // "de" is a prefix of description, not delete-description
		{"desc", 4}, // abbreviation semantics
		{"del", 6},
		{"delete-description", 6},
		{"t", 5},
		{"title", 5},
		{"bogus", -1},
		{"names", -1}, // longer than the entry never matches
	}
	for _, tt := range tests {
		if got := stringOldSearchBlock(tt.token); got != tt.want {
			t.Errorf("stringOldSearchBlock(%q) = %d, want %d", tt.token, got, tt.want)
		}
	}
}

// TestStringMobInlineFields verifies each live write on an NPC, plus the exact
// "Ok.\n\r" acknowledgement. Field 3 additionally proves C's in-switch CRLF
// append (modify.c:648-651).
func TestStringMobInlineFields(t *testing.T) {
	tests := []struct {
		name string
		line string
		read func(*game.MobInstance) string
		want string
	}{
		{"name", "mob guard name goblin sentinel", func(m *game.MobInstance) string { return m.Proto().Keywords }, "goblin sentinel"},
		{"short", "mob guard short A watchful goblin", func(m *game.MobInstance) string { return m.GetShortDesc() }, "A watchful goblin"},
		{"long", "mob guard long A goblin stands watch.", func(m *game.MobInstance) string { return m.GetLongDesc() }, "A goblin stands watch.\r\n"},
		{"description", "mob guard description A scrawny goblin watches you.", func(m *game.MobInstance) string { return m.Proto().DetailedDesc }, "A scrawny goblin watches you."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, m := newStringTestSession(t, LVL_IMPL)
			mob := registerMob(t, m, 2001, 1001)

			runString(t, s, tt.line)
			if got := readMsgText(t, s); got != stringInlineOk {
				t.Errorf("%q ack = %q, want %q", tt.line, got, stringInlineOk)
			}
			if got := tt.read(mob); got != tt.want {
				t.Errorf("%q wrote %q, want %q", tt.line, got, tt.want)
			}
			if s.IsTextEditing() {
				t.Errorf("%q opened the editor for an inline string", tt.line)
			}
		})
	}
}

// TestStringInlineTruncationIsLowercaseLFCR pins the inline form of the length
// notice — "truncated", LFCR — against the editor's capital-T CRLF variant that
// tedit.go owns.
func TestStringInlineTruncationIsLowercaseLFCR(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	mob := registerMob(t, m, 2001, 1001)

	// length[0] is 15, so a 26-byte name keeps its first 15 bytes.
	runString(t, s, "mob guard name abcdefghijklmnopqrstuvwxyz")
	if got := readMsgText(t, s); got != stringTooLongTruncated {
		t.Fatalf("truncation notice = %q, want %q", got, stringTooLongTruncated)
	}
	if got := readMsgText(t, s); got != stringInlineOk {
		t.Fatalf("ack after truncation = %q, want %q", got, stringInlineOk)
	}
	if got := mob.Proto().Keywords; got != "abcdefghijklmno" {
		t.Errorf("truncated name = %q, want 15 bytes", got)
	}
}

// TestStringMobLongWithoutInlineTextMatchesC covers the consequence of C's
// in-switch append: a bare `string mob x long` writes a real CRLF instead of
// opening the editor, and reports Ok.
func TestStringMobLongWithoutInlineTextMatchesC(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	mob := registerMob(t, m, 2001, 1001)

	runString(t, s, "mob guard long")
	if got := readMsgText(t, s); got != stringInlineOk {
		t.Fatalf("bare long ack = %q, want %q (no editor)", got, stringInlineOk)
	}
	if got := mob.GetLongDesc(); got != "\r\n" {
		t.Errorf("bare long write = %q, want %q", got, "\r\n")
	}
	if s.IsTextEditing() {
		t.Error("bare `string mob x long` opened the editor; C writes \"\\r\\n\" inline")
	}
}

// TestStringMobLongAppendInteractsWithTruncation proves the append happens
// before the length check: length[2] is 256, so a 255-byte value plus "\r\n" is
// reported and cut to 256 bytes, leaving the bare '\r'.
func TestStringMobLongAppendInteractsWithTruncation(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	mob := registerMob(t, m, 2001, 1001)

	value := strings.Repeat("x", 255)
	runString(t, s, "mob guard long "+value)
	if got := readMsgText(t, s); got != stringTooLongTruncated {
		t.Fatalf("truncation notice = %q, want %q", got, stringTooLongTruncated)
	}
	if got := readMsgText(t, s); got != stringInlineOk {
		t.Fatalf("ack = %q, want %q", got, stringInlineOk)
	}
	want := value + "\r" // 255 bytes + the leading CRLF byte = length[2]
	if got := mob.GetLongDesc(); got != want {
		t.Errorf("long value = %d bytes, want %d bytes ending %q", len(got), len(want), got[len(got)-1:])
	}
}

// TestStringObjectInlineFields covers the carried-object writes. Unlike the mob
// long field, the object fields have no CRLF append.
func TestStringObjectInlineFields(t *testing.T) {
	tests := []struct {
		name string
		line string
		read func(*game.ObjectInstance) string
		want string
	}{
		{"name", "obj sword name blade katana", func(o *game.ObjectInstance) string { return o.GetKeywords() }, "blade katana"},
		{"short", "obj sword short A runed katana", func(o *game.ObjectInstance) string { return o.GetShortDesc() }, "A runed katana"},
		{"long", "obj sword long A katana lies here.", func(o *game.ObjectInstance) string { return o.GetLongDesc() }, "A katana lies here."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newStringTestSession(t, LVL_IMPL)
			obj := carriedObject(t, s, "sword long iron", "An iron longsword")

			runString(t, s, tt.line)
			if got := readMsgText(t, s); got != stringInlineOk {
				t.Errorf("%q ack = %q, want %q", tt.line, got, stringInlineOk)
			}
			if got := tt.read(obj); got != tt.want {
				t.Errorf("%q wrote %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

// newStringTargetPlayer registers a second player in the actor's room so
// world-scope character resolution finds them, exactly like a logged-in peer.
func newStringTargetPlayer(t *testing.T, m *Manager, name string, roomVNum int) *game.Player {
	t.Helper()
	p := game.NewPlayer(77, name, roomVNum)
	if err := m.world.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	return p
}

// TestStringPlayerTargetFields covers the branch where the `mob` target is a
// player: fields 2 and 3 are monsters-only, 4 and 5 write the player fields.
func TestStringPlayerTargetFields(t *testing.T) {
	t.Run("short is monsters only", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL)
		newStringTargetPlayer(t, m, "victim", 1001)
		runString(t, s, "mob victim short A victim")
		if got := readMsgText(t, s); got != stringMonstersOnly {
			t.Errorf("output = %q, want %q", got, stringMonstersOnly)
		}
	})

	t.Run("long is monsters only", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL)
		newStringTargetPlayer(t, m, "victim", 1001)
		runString(t, s, "mob victim long A victim stands here.")
		if got := readMsgText(t, s); got != stringMonstersOnly {
			t.Errorf("output = %q, want %q", got, stringMonstersOnly)
		}
	})

	t.Run("description", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL)
		victim := newStringTargetPlayer(t, m, "victim", 1001)
		runString(t, s, "mob victim description A tall figure.")
		if got := readMsgText(t, s); got != stringInlineOk {
			t.Errorf("ack = %q, want %q", got, stringInlineOk)
		}
		if victim.Description != "A tall figure." {
			t.Errorf("player description = %q", victim.Description)
		}
	})

	t.Run("title", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL)
		victim := newStringTargetPlayer(t, m, "victim", 1001)
		runString(t, s, "mob victim title the Unlucky")
		if got := readMsgText(t, s); got != stringInlineOk {
			t.Errorf("ack = %q, want %q", got, stringInlineOk)
		}
		if got := victim.GetTitle(); got != "the Unlucky" {
			t.Errorf("player title = %q", got)
		}
	})
}

// TestStringPlayerRenameGate covers the per-field LEVEL_IMPL-1 gate and the
// WARNING line C emits immediately before the standard tail. The refusal literal
// has no terminator, so it is compared exactly. C's test is
// `GET_LEVEL(ch) < LEVEL_IMPL-1`, so 38 is refused and 39 is allowed.
func TestStringPlayerRenameGate(t *testing.T) {
	t.Run("refused below LEVEL_IMPL-1", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL-2)
		victim := newStringTargetPlayer(t, m, "victim", 1001)
		runString(t, s, "mob victim name renamed")
		if got := readMsgText(t, s); got != stringPlayerFieldRefused {
			t.Errorf("output = %q, want %q", got, stringPlayerFieldRefused)
		}
		if victim.Name != "victim" {
			t.Errorf("refused rename still changed the name to %q", victim.Name)
		}
	})

	t.Run("allowed at LEVEL_IMPL-1", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL-1)
		victim := newStringTargetPlayer(t, m, "victim", 1001)
		runString(t, s, "mob victim name renamed")
		if got := readMsgText(t, s); got != stringPlayerNameWarning {
			t.Fatalf("warning = %q, want %q", got, stringPlayerNameWarning)
		}
		if got := readMsgText(t, s); got != stringInlineOk {
			t.Fatalf("ack = %q, want %q", got, stringInlineOk)
		}
		if victim.Name != "renamed" {
			t.Errorf("player name = %q, want %q", victim.Name, "renamed")
		}
	})
}

// createExtraDesc runs the field-4 create path and writes one line of text.
func createExtraDesc(t *testing.T, s *Session, keyword, text string) {
	t.Helper()
	runString(t, s, "obj sword description "+keyword)
	if got := readMsgText(t, s); got != stringNewField {
		t.Fatalf("create %q = %q, want %q", keyword, got, stringNewField)
	}
	if !s.IsTextEditing() {
		t.Fatalf("create %q did not open the editor", keyword)
	}
	s.handleTextEditInput(text)
	s.handleTextEditInput("@")
	if s.IsTextEditing() {
		t.Fatalf("editor still active after saving %q", keyword)
	}
	if got := readMsgTextOrEmpty(t, s); got != "" {
		t.Fatalf("saving %q emitted %q; C's save is silent for a live string", keyword, got)
	}
}

// TestStringObjectExtraDescLifecycle covers create, in-place modify (str_cmp is
// case-insensitive and must not duplicate the field) and head delete.
func TestStringObjectExtraDescLifecycle(t *testing.T) {
	s, _ := newStringTestSession(t, LVL_IMPL)
	obj := carriedObject(t, s, "sword long iron", "An iron longsword")

	createExtraDesc(t, s, "blade", "A gleaming blade.")
	descs := obj.GetExtraDescs()
	if len(descs) != 1 || descs[0].Keywords != "blade" || descs[0].Description != "A gleaming blade.\r\n" {
		t.Fatalf("after create: %+v", descs)
	}

	// Modify: C matches with str_cmp, so a different case hits the existing node.
	runString(t, s, "obj sword description BLADE")
	if got := readMsgText(t, s); got != stringModifyingDescription {
		t.Fatalf("modify notice = %q, want %q", got, stringModifyingDescription)
	}
	if got := obj.GetExtraDescs()[0].Description; got != "" {
		t.Fatalf("modify must clear the field before editing, got %q", got)
	}
	s.handleTextEditInput("A dull blade.")
	s.handleTextEditInput("@")
	descs = obj.GetExtraDescs()
	if len(descs) != 1 {
		t.Fatalf("modify created a duplicate: %+v", descs)
	}
	if descs[0].Description != "A dull blade.\r\n" {
		t.Errorf("modified description = %q", descs[0].Description)
	}

	runString(t, s, "obj sword delete-description blade")
	if got := readMsgText(t, s); got != stringFieldDeleted {
		t.Fatalf("delete = %q, want %q", got, stringFieldDeleted)
	}
	if got := obj.GetExtraDescs(); len(got) != 0 {
		t.Errorf("after delete: %+v", got)
	}
}

// TestStringDeleteUnwrittenExtraDescReportsNoField is the NULL-description case:
// find_exdesc returns the entry's description pointer, so a field that was
// created but never written reads as "not found" (src/act.informative.c:987-996).
func TestStringDeleteUnwrittenExtraDescReportsNoField(t *testing.T) {
	s, _ := newStringTestSession(t, LVL_IMPL)
	obj := carriedObject(t, s, "sword long iron", "An iron longsword")

	runString(t, s, "obj sword description ghost")
	if got := readMsgText(t, s); got != stringNewField {
		t.Fatalf("create = %q, want %q", got, stringNewField)
	}
	s.handleTextEditInput("@") // leave without writing a line

	runString(t, s, "obj sword delete-description ghost")
	if got := readMsgText(t, s); got != stringNoFieldWithKeyword {
		t.Errorf("delete of an unwritten field = %q, want %q", got, stringNoFieldWithKeyword)
	}
	if got := obj.GetExtraDescs(); len(got) != 1 {
		t.Errorf("field list changed: %+v", got)
	}
}

// TestStringDeleteNonHeadExtraDescRefusesInsteadOfCrashing covers modify.c:722's
// head-only guard, where C reads length[5] out of bounds and then dereferences a
// NULL d->str — a segfault. The port must refuse without output, without
// deleting, and without panicking (the forced divergence documented in the PR).
func TestStringDeleteNonHeadExtraDescRefusesInsteadOfCrashing(t *testing.T) {
	s, _ := newStringTestSession(t, LVL_IMPL)
	obj := carriedObject(t, s, "sword long iron", "An iron longsword")

	// Each create head-inserts, so "first" is the non-head entry.
	createExtraDesc(t, s, "first", "one")
	createExtraDesc(t, s, "second", "two")

	runString(t, s, "obj sword delete-description first")
	if got := readMsgTextOrEmpty(t, s); got != "" {
		t.Errorf("non-head delete emitted %q; C produces no output before crashing", got)
	}
	descs := obj.GetExtraDescs()
	if len(descs) != 2 {
		t.Fatalf("non-head delete changed the list: %+v", descs)
	}
	if descs[0].Keywords != "second" || descs[1].Keywords != "first" {
		t.Errorf("list order changed: %+v", descs)
	}
}

// TestStringEditorWriteThroughAndAbortRestoresNothing is the core editor-shape
// test: every accepted line lands in the live field immediately (C's d->str is
// the field), entering the editor already cleared it, and an abort restores
// nothing because do_string never set d->backstr.
func TestStringEditorWriteThroughAndAbortRestoresNothing(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	mob := registerMob(t, m, 2001, 1001)
	mob.SetLiveShortDesc("the original short description")

	runString(t, s, "mob guard short")
	if got := readMsgText(t, s); got != stringEnterString {
		t.Fatalf("editor entry = %q, want %q", got, stringEnterString)
	}
	if !s.IsTextEditing() {
		t.Fatal("editor is not active after the no-string form")
	}
	if got := mob.GetShortDesc(); got != "" {
		t.Fatalf("editor entry left %q in the field; C clears it (*d->str = 0)", got)
	}

	s.handleTextEditInput("first line")
	if got := mob.GetShortDesc(); got != "first line\r\n" {
		t.Fatalf("after line 1 the live field = %q", got)
	}
	s.handleTextEditInput("second line")
	if got := mob.GetShortDesc(); got != "first line\r\nsecond line\r\n" {
		t.Fatalf("after line 2 the live field = %q", got)
	}

	s.handleTextEditInput("/a")
	if s.IsTextEditing() {
		t.Error("editor still active after /a")
	}
	if got := mob.GetShortDesc(); got != "first line\r\nsecond line\r\n" {
		t.Errorf("abort changed the live field to %q; C restores nothing", got)
	}
	if got := readMsgTextOrEmpty(t, s); got != "" {
		t.Errorf("abort emitted %q; C's abort path only logs", got)
	}
}

// TestStringEditorSaveIsSilent covers both exits C treats as a save: "@" and
// "/s". playing_string_cleanup has no non-mail branch, so neither prints.
func TestStringEditorSaveIsSilent(t *testing.T) {
	for _, terminator := range []string{"@", "/s"} {
		t.Run(terminator, func(t *testing.T) {
			s, m := newStringTestSession(t, LVL_IMPL)
			mob := registerMob(t, m, 2001, 1001)

			runString(t, s, "mob guard description")
			if got := readMsgText(t, s); got != stringEnterString {
				t.Fatalf("entry = %q, want %q", got, stringEnterString)
			}
			s.handleTextEditInput("A scrawny goblin.")
			if got := mob.Proto().DetailedDesc; got != "A scrawny goblin.\r\n" {
				t.Fatalf("live field before save = %q", got)
			}

			s.handleTextEditInput(terminator)
			if s.IsTextEditing() {
				t.Fatal("editor still active after save")
			}
			if got := readMsgTextOrEmpty(t, s); got != "" {
				t.Errorf("save emitted %q; C's live-string save is silent", got)
			}
			if got := mob.Proto().DetailedDesc; got != "A scrawny goblin.\r\n" {
				t.Errorf("saved field = %q", got)
			}
		})
	}
}

// TestStringEditorClearBufferEmptiesTheLiveField proves /c writes the cleared
// buffer back through to the live field, and that the next line then takes
// string_add's first-line branch again.
func TestStringEditorClearBufferEmptiesTheLiveField(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	mob := registerMob(t, m, 2001, 1001)

	runString(t, s, "mob guard short")
	if got := readMsgText(t, s); got != stringEnterString {
		t.Fatalf("entry = %q", got)
	}
	s.handleTextEditInput("line one")
	if got := mob.GetShortDesc(); got != "line one\r\n" {
		t.Fatalf("live field = %q", got)
	}

	s.handleTextEditInput("/c")
	if got := readMsgText(t, s); got != "Current buffer cleared.\r\n" {
		t.Fatalf("clear notice = %q", got)
	}
	if got := mob.GetShortDesc(); got != "" {
		t.Errorf("cleared buffer left %q in the live field", got)
	}

	s.handleTextEditInput("fresh")
	if got := mob.GetShortDesc(); got != "fresh\r\n" {
		t.Errorf("after the first line following /c, field = %q", got)
	}
}

// TestStringEditorDisconnectKeepsWrittenText covers close_socket's path: C drops
// the descriptor without a string cleanup for CON_PLAYING, so the bytes already
// written through the live field survive while the editor state goes away.
func TestStringEditorDisconnectKeepsWrittenText(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	mob := registerMob(t, m, 2001, 1001)

	runString(t, s, "mob guard description")
	if got := readMsgText(t, s); got != stringEnterString {
		t.Fatalf("entry = %q", got)
	}
	s.handleTextEditInput("half typed")

	s.cancelTextEdit()
	if s.IsTextEditing() {
		t.Fatal("editor survived cancelTextEdit")
	}
	if got := mob.Proto().DetailedDesc; got != "half typed\r\n" {
		t.Errorf("live field after disconnect = %q, want the written-through text", got)
	}
}

// TestStringEditorTruncationMessagesMatchStringAdd pins string_add's own
// notices, which differ from the inline ones in case and terminator.
func TestStringEditorTruncationMessagesMatchStringAdd(t *testing.T) {
	t.Run("first line too long", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL)
		mob := registerMob(t, m, 2001, 1001)

		runString(t, s, "mob guard name") // length[0] == 15
		if got := readMsgText(t, s); got != stringEnterString {
			t.Fatalf("entry = %q", got)
		}
		s.handleTextEditInput("abcdefghijklmnopqrst") // 20 bytes
		if got := readMsgText(t, s); got != "String too long - Truncated.\r\n" {
			t.Fatalf("notice = %q, want capital-T CRLF variant", got)
		}
		if got := mob.Proto().Keywords; got != "abcdefghijkl\r\n" {
			t.Errorf("truncated field = %q, want 12 bytes + CRLF", got)
		}
	})

	t.Run("later line too long", func(t *testing.T) {
		s, m := newStringTestSession(t, LVL_IMPL)
		mob := registerMob(t, m, 2001, 1001)

		runString(t, s, "mob guard name")
		if got := readMsgText(t, s); got != stringEnterString {
			t.Fatalf("entry = %q", got)
		}
		s.handleTextEditInput("abc")
		s.handleTextEditInput("abcdefghij") // 10 + 5 + 3 > 15
		if got := readMsgText(t, s); got != "String too long.  Last line skipped.\r\n" {
			t.Fatalf("notice = %q, want two spaces and CRLF", got)
		}
		if got := mob.Proto().Keywords; got != "abc\r\n" {
			t.Errorf("field = %q, want the previous line unchanged", got)
		}
	})
}

// TestStringEditorPromptMatchesMakePrompt pins C's prompt while the live-string
// editor is active: make_prompt returns a bare "] " whenever d->str is set
// (comm.c:1038-1039), and process_output frames a CON_PLAYING flush with an
// extra "\r\n" (comm.c:1633-1636) — so the first prompt after editor output
// carries that blank line and a later bare prompt does not.
func TestStringEditorPromptMatchesMakePrompt(t *testing.T) {
	s, m := newStringTestSession(t, LVL_IMPL)
	registerMob(t, m, 2001, 1001)

	runString(t, s, "mob guard short")
	if got := readMsgText(t, s); got != stringEnterString {
		t.Fatalf("entry = %q", got)
	}

	if got := promptTextOf(t, s); got != "\r\n] " {
		t.Errorf("prompt after editor output = %q, want %q", got, "\r\n] ")
	}
	if got := promptTextOf(t, s); got != "] " {
		t.Errorf("bare editor prompt = %q, want %q", got, "] ")
	}
}

// promptTextOf sends a prompt and returns its text.
func promptTextOf(t *testing.T, s *Session) string {
	t.Helper()
	s.SendPrompt()
	msg := <-s.send
	var sm ServerMessage
	if err := json.Unmarshal(msg, &sm); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if sm.Type != MsgPrompt {
		t.Fatalf("message type = %q, want %q", sm.Type, MsgPrompt)
	}
	data, err := json.Marshal(sm.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	var prompt map[string]interface{}
	if err := json.Unmarshal(data, &prompt); err != nil {
		t.Fatalf("unmarshal prompt: %v", err)
	}
	text, _ := prompt["text"].(string)
	return text
}
