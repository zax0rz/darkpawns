package session

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// cmdString ports do_string (src/modify.c:594-772), with quad_arg
// (src/modify.c:562-592) and old_search_block (src/whod.c:496-527) inlined.
//
// `string` edits strings on LIVE entities: an NPC's name/short/long/
// description/title, and one carried object's name/short/long plus its extra
// descriptions (create, modify, delete). Nothing is written to the world files;
// the edit lands on the instance and dies with it (see pkg/game/live_strings.go).
//
// C's get_obj (src/modify.c:538-558) is NOT ported. It has zero callers anywhere
// under src/ — the command's call path goes through get_char_vis
// (modify.c:620) and get_obj_in_list_vis (modify.c:668) — so porting it would
// be invention (R4). The exclusion is recorded in docs/fidelity/depth/string.tsv.

const (
	stringTypeMob = iota
	stringTypeObj
	stringTypeError
)

// stringFields is C's string_fields[] (src/modify.c:60-69); the trailing "\n"
// sentinel is the end of the slice. The order is load-bearing: old_search_block
// mode 0 is a prefix match, so "de" resolves to description and "del" to
// delete-description.
var stringFields = []string{
	"name",
	"short",
	"long",
	"description",
	"title",
	"delete-description",
}

// stringFieldLengths is C's length[] (src/modify.c:73-80): maximum length for
// text field x+1.
var stringFieldLengths = []int{15, 60, 256, 240, 60}

// stringMaxLength is MAX_STRING_LENGTH (src/structs.h:643), the bound the
// extra-description editor runs with (modify.c:710).
const stringMaxLength = 8192

// Every player-facing byte do_string can emit, with C's exact terminator. Most
// end "\n\r" (this codebase's reversed LFCR quirk); the usage line and the
// player-field refusal have no terminator at all, and the inline truncation
// notice is lower-case "truncated" where the editor's is capital-T (that one
// lives in tedit.go, from string_add itself).
const (
	stringUsage                = "Usage: string ('obj'|'mob') <name> <field> [<string>]"
	stringNoField              = "No field by that name. Try 'help string'.\n\r"
	stringNoTargetMob          = "I don't know anyone by that name...\n\r"
	stringPlayerFieldRefused   = "You can't change that field for players."
	stringPlayerNameWarning    = "WARNING: You have changed the name of a player.\n\r"
	stringMonstersOnly         = "That field is for monsters only.\n\r"
	stringMonstersNoTitles     = "Monsters have no titles.\n\r"
	stringUndefinedForMonsters = "That field is undefined for monsters.\n\r"
	stringNoTargetObj          = "Nothing by that name..\n\r"
	stringKeywordRequired      = "You have to supply a keyword.\n\r"
	stringNewField             = "New field.\n\r"
	stringModifyingDescription = "Modifying description.\n\r"
	stringFieldNameRequired    = "You must supply a field name.\n\r"
	stringNoFieldWithKeyword   = "No field with that keyword.\n\r"
	stringFieldDeleted         = "Field deleted.\n\r"
	stringUndefinedForObjects  = "That field is undefined for objects.\n\r"
	stringTooLongTruncated     = "String too long - truncated.\n\r"
	stringInlineOk             = "Ok.\n\r"
	stringEnterString          = "Enter string. terminate with '@'.\n\r"

	// stringAbortSyserr is C's log() line for an abort from a state with no
	// backstr row (modify.c:173). A `string` edit runs with STATE(d) ==
	// CON_PLAYING, which the abort switch does not handle.
	stringAbortSyserr = "SYSERR: string_add: Aborting write from unknown origin."
	// stringDeleteFallthroughSyserr records the port's forced divergence where
	// C segfaults (modify.c:722-746): see stringObjDeleteExtraDesc.
	stringDeleteFallthroughSyserr = "SYSERR: do_string: delete-description fell through the head-node guard; C dereferences NULL here"
)

// stringQuadArg is quad_arg's output. Name, Field and Str are only meaningful
// when Kind != stringTypeError: C returns from quad_arg before writing them and
// do_string checks the type before touching them.
type stringQuadArg struct {
	Kind  int
	Name  string
	Field int
	Str   string
}

// stringOldSearchBlock ports old_search_block(token, 0, strlen(token),
// string_fields, 0) (src/whod.c:496-527). Mode 0 compares only the first
// strlen(token) bytes of each entry, so the token is a prefix of the entry; the
// loop increments the guess before returning, so a match on entry i yields the
// 1-based field number i+1. An empty token matches before the loop
// (found = length < 1) and returns 0; running off the "\n" sentinel returns -1.
func stringOldSearchBlock(token string) int {
	if token == "" {
		return 0
	}
	for i, field := range stringFields {
		if strings.HasPrefix(field, token) {
			return i + 1
		}
	}
	return -1
}

// parseStringQuadArg ports quad_arg (src/modify.c:562-592): one_argument for the
// type, name and field tokens (each lowercased with fill words skipped), then a
// verbatim copy of the rest of the line as the inline string.
func parseStringQuadArg(argument string) stringQuadArg {
	var qa stringQuadArg

	// determine type: is_abbrev(buf, "mob") / is_abbrev(buf, "obj")
	// (src/interpreter.c:1356-1370) — the token must be a non-empty prefix of
	// the table word, so "mobs" is not a type and falls through to TP_ERROR.
	token, rest := game.OneArgument(argument)
	switch {
	case game.IsAbbrev(token, "mob"):
		qa.Kind = stringTypeMob
	case game.IsAbbrev(token, "obj"):
		qa.Kind = stringTypeObj
	default:
		qa.Kind = stringTypeError
		return qa
	}

	// find name
	qa.Name, rest = game.OneArgument(rest)

	// field name and number
	var fieldToken string
	fieldToken, rest = game.OneArgument(rest)
	qa.Field = stringOldSearchBlock(fieldToken)
	if qa.Field == 0 {
		// C: if (!(*field = old_search_block(...))) return; — the string stays
		// empty and do_string prints the no-field message before any lookup.
		return qa
	}

	// string: skip leading whitespace, then copy to the end of the line.
	qa.Str = strings.TrimLeft(rest, cCommandWhitespace)
	return qa
}

// cmdString is the tokenized entry point retained for direct callers and tests.
// The transport path calls cmdStringText so the inline string keeps C's original
// interior spacing.
func cmdString(s *Session, args []string) error {
	return cmdStringText(s, args, "")
}

// cmdStringText ports do_string (src/modify.c:594-772).
func cmdStringText(s *Session, args []string, rawArgs string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	// C opens with `if (IS_NPC(ch)) return;`. A switched immortal is refused by
	// the immortal-command gate before any handler runs (commandGateRejected:
	// gate.MinLevel 32 >= LVL_IMMORT), so that guard has no reachable case in
	// the port; the disposition is recorded in docs/fidelity/depth/string.tsv.

	argument := rawArgs
	if argument == "" {
		argument = strings.Join(args, " ")
	}

	qa := parseStringQuadArg(argument)
	if qa.Kind == stringTypeError {
		s.Send(stringUsage)
		return nil
	}
	if qa.Field == 0 {
		s.Send(stringNoField)
		return nil
	}

	if qa.Kind == stringTypeMob {
		return s.stringMobBranch(qa)
	}
	return s.stringObjBranch(qa)
}

// stringMobBranch is C's TP_MOB arm (src/modify.c:619-665). get_char_vis
// resolves a mob or a player anywhere the actor can see; the field switch then
// either refuses outright or points d->str at one live field.
func (s *Session) stringMobBranch(qa stringQuadArg) error {
	target, ok := s.manager.world.ResolveCharWorld(s.player, qa.Name)
	if !ok {
		s.Send(stringNoTargetMob)
		return nil
	}

	var apply func(string)
	switch {
	case target.Player != nil:
		player := target.Player
		switch qa.Field {
		case 1:
			// Per-field gate inside the command: renaming a player needs
			// LEVEL_IMPL-1.
			if s.player.GetLevel() < LVL_IMPL-1 {
				s.Send(stringPlayerFieldRefused)
				return nil
			}
			s.Send(stringPlayerNameWarning)
			apply = func(value string) { s.applyLivePlayerName(player, value) }
		case 2, 3:
			s.Send(stringMonstersOnly)
			return nil
		case 4:
			apply = func(value string) { player.Description = value }
		case 5:
			apply = func(value string) { game.SetTitle(player, value) }
		default:
			// Field 6, and any value old_search_block rejected (-1), land here.
			s.Send(stringUndefinedForMonsters)
			return nil
		}
	case target.Mob != nil:
		mob := target.Mob
		switch qa.Field {
		case 1:
			apply = func(value string) { mob.SetLiveKeywords(value) }
		case 2:
			apply = func(value string) { mob.SetLiveShortDesc(value) }
		case 3:
			// C appends a real CRLF inside the switch, before the inline/editor
			// test (modify.c:648-651). A bare `string mob x long` therefore
			// writes "\r\n" and reports Ok instead of opening the editor.
			qa.Str += "\r\n"
			apply = func(value string) { mob.SetLiveLongDesc(value) }
		case 4:
			apply = func(value string) { mob.SetLiveDetailedDesc(value) }
		case 5:
			s.Send(stringMonstersNoTitles)
			return nil
		default:
			s.Send(stringUndefinedForMonsters)
			return nil
		}
	default:
		s.Send(stringNoTargetMob)
		return nil
	}

	return s.finishStringField(qa, apply, stringFieldBytes(qa.Field))
}

// stringObjBranch is C's TP_OBJ arm (src/modify.c:666-752). Only carried objects
// are searched (get_obj_in_list_vis over ch->carrying).
func (s *Session) stringObjBranch(qa stringQuadArg) error {
	obj, ok := s.manager.world.ResolveObjectInInventory(s.player, qa.Name)
	if !ok {
		s.Send(stringNoTargetObj)
		return nil
	}

	switch qa.Field {
	case 1:
		return s.finishStringField(qa, func(value string) { obj.SetLiveKeywords(value) }, stringFieldBytes(qa.Field))
	case 2:
		return s.finishStringField(qa, func(value string) { obj.SetLiveShortDesc(value) }, stringFieldBytes(qa.Field))
	case 3:
		return s.finishStringField(qa, func(value string) { obj.SetLiveLongDesc(value) }, stringFieldBytes(qa.Field))
	case 4:
		// Create-or-modify an extra description. C returns from the case, past
		// the standard tail, so the inline string is a keyword and never a
		// value (modify.c:676-712).
		return s.stringObjExtraDescEditor(obj, qa.Str)
	case 6:
		return s.stringObjDeleteExtraDesc(obj, qa.Str)
	default:
		// Field 5 (title) and any rejected value are undefined for objects.
		s.Send(stringUndefinedForObjects)
		return nil
	}
}

// stringObjExtraDescEditor is C's object field 4: given an inline keyword, it
// either resets an existing extra description or creates one at the head of the
// list, then opens the editor with MAX_STRING_LENGTH and returns.
func (s *Session) stringObjExtraDescEditor(obj *game.ObjectInstance, keyword string) error {
	if keyword == "" {
		s.Send(stringKeywordRequired)
		return nil
	}

	descs := obj.LiveExtraDescs()
	index := -1
	for i := range descs {
		// C: str_cmp(ed->keyword, string) — case-insensitive exact equality
		// (src/utils.c:107).
		if strings.EqualFold(descs[i].Keywords, keyword) {
			index = i
			break
		}
	}

	if index >= 0 {
		// C frees the old description and points d->str at the field, so the
		// text is gone before the editor starts.
		descs[index].Description = ""
		obj.SetLiveExtraDescs(descs)
		s.Send(stringModifyingDescription)
	} else {
		// C creates the node at the head of obj->ex_description.
		descs = append([]parser.ExtraDesc{{Keywords: keyword}}, descs...)
		index = 0
		obj.SetLiveExtraDescs(descs)
		s.Send(stringNewField)
	}

	apply := func(value string) {
		current := obj.LiveExtraDescs()
		if index < len(current) {
			current[index].Description = value
			obj.SetLiveExtraDescs(current)
		}
	}
	s.startLiveStringEditor(apply, stringMaxLength)
	return nil
}

// stringObjDeleteExtraDesc is C's object field 6.
//
// C looks the keyword up with find_exdesc (src/act.informative.c:987-996), then
// re-tests isname_with_abbrevs against the list HEAD only (modify.c:722). When
// the match is not the head, control leaves the switch at modify.c:746 and runs
// the standard tail with field == 6: length[5] is read past the end of the
// five-element array and CREATE() writes through d->str, which do_string never
// set for this field — a NULL dereference, i.e. a segfault. A port cannot
// reproduce a crash, so this path reproduces C's observable output (nothing)
// and records the refusal in the SYSERR shape. Forced divergence, documented in
// the PR and the depth manifest.
func (s *Session) stringObjDeleteExtraDesc(obj *game.ObjectInstance, keyword string) error {
	if keyword == "" {
		s.Send(stringFieldNameRequired)
		return nil
	}

	descs := obj.LiveExtraDescs()
	index := game.FindExtraDescIndex(descs, keyword)
	if index < 0 || descs[index].Description == "" {
		// find_exdesc returns the entry's description pointer, so a field whose
		// description was never written behaves as "not found" even though the
		// keyword exists.
		s.Send(stringNoFieldWithKeyword)
		return nil
	}
	if !game.IsNameWithAbbrevs(keyword, descs[0].Keywords) {
		slog.Error(stringDeleteFallthroughSyserr,
			"player", s.playerName, "keyword", keyword, "object", obj.GetShortDesc())
		return nil
	}

	obj.SetLiveExtraDescs(descs[1:])
	s.Send(stringFieldDeleted)
	return nil
}

// stringFieldBytes is length[field-1] (src/modify.c:73-80). Out-of-range fields
// never reach a caller, so 0 means "no bound" rather than a real limit.
func stringFieldBytes(field int) int {
	if field < 1 || field > len(stringFieldLengths) {
		return 0
	}
	return stringFieldLengths[field-1]
}

// finishStringField is the code after C's field switches (modify.c:754-771). An
// inline string is written immediately; otherwise the field is cleared and the
// improved editor takes over with the field's length as its bound.
func (s *Session) finishStringField(qa stringQuadArg, apply func(string), maxBytes int) error {
	if qa.Str != "" {
		if maxBytes > 0 && len(qa.Str) > maxBytes {
			s.Send(stringTooLongTruncated)
			qa.Str = qa.Str[:maxBytes]
		}
		apply(qa.Str)
		s.Send(stringInlineOk)
		return nil
	}

	s.Send(stringEnterString)
	// C clears the live field on entry (*ch->desc->str = 0), so the previous
	// text is already gone — and, with no backstr, an abort cannot restore it.
	apply("")
	s.startLiveStringEditor(apply, maxBytes)
	return nil
}

// applyLivePlayerName writes a renamed player's name field and repoints the
// connected session, mirroring do_set's live rename handling (wiz_set.go).
func (s *Session) applyLivePlayerName(player *game.Player, value string) {
	if player == nil {
		return
	}
	old := player.Name
	player.Name = value
	if target := findSessionForPlayer(s.manager, player); target != nil {
		target.playerName = value
	}
	slog.Warn("string renamed live player", "actor", s.playerName, "old_name", old, "new_name", value)
}

// startLiveStringEditor installs do_string's editor. C reaches it without
// string_write: d->str points at the live field, d->max_str is the field bound,
// PLR_WRITING is never set, there is no d->backstr, and STATE(d) stays
// CON_PLAYING. That shape drives three behaviors the port must keep:
//
//   - every accepted line and every /action writes straight into the live field
//     (liveString), because the buffer *is* the field;
//   - save is silent — playing_string_cleanup has no non-mail branch, so no
//     "Saved." line and no room act;
//   - abort restores nothing and logs the unhandled-origin SYSERR, because the
//     abort switch has no CON_PLAYING row.
func (s *Session) startLiveStringEditor(apply func(string), maxBytes int) {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	// C assigns d->str unconditionally, so a second editor clobbers the first.
	// The input path makes that unreachable (editor lines cannot also be a
	// command), so replacing the state here just keeps the descriptor usable.
	s.textEdit = &textEditState{
		field:         textEditField{name: "string", maxBytes: maxBytes},
		liveString:    apply,
		playingEditor: true,
		onComplete: func(action textEditAction, _, _ string) {
			if action == textEditAbort {
				slog.Error(stringAbortSyserr, "player", s.playerName)
			}
		},
	}
}
