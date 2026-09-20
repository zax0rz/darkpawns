package session

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// These values mirror the REDIT_MODE constants in src/olc.h. Keeping the
// state names local makes the descriptor transition table auditable without
// pretending that the other OLC families share this implementation.
type reditMode uint8

const (
	reditMainMenu reditMode = iota + 1
	reditName
	reditDescription
	reditFlags
	reditSector
	reditExitMenu
	reditConfirmSave
	reditConfirmSaveString
	reditExitNumber
	reditExitDescription
	reditExitKeyword
	reditExitKey
	reditExitDoorFlags
	reditExtraMenu
	reditExtraKey
	reditExtraDescription
	reditCopy
	reditScriptMenu
	reditScriptName
	reditScriptFlags
)

type reditExtraMeta struct {
	keywordSet     bool
	descriptionSet bool
}

type reditState struct {
	room       parser.Room
	number     int
	zoneNumber int
	isNew      bool
	mode       reditMode
	value      int // C OLC_VAL while an exit is selected.
	olcVal     int // C OLC_VAL's dirty/quit-prompt value.

	currentExtra  int
	extraMeta     []reditExtraMeta
	pendingOutput string
}

type reditStringField uint8

const (
	reditRoomDescription reditStringField = iota + 1
	reditExitDescriptionField
	reditExtraDescriptionField
)

var reditRoomFlagNames = []string{
	"DARK", "DEATH", "!MOB", "INDOORS", "PEACEFUL", "SOUNDPROOF", "!TRACK",
	"!MAGIC", "TUNNEL", "PRIVATE", "GODROOM", "HOUSE", "HCRSH", "ATRIUM",
	"OLC", "*", "NEUTRAL", "BFR", "REGENROOM", "NO_WHO_ROOM", "**",
	"FLOW_NORTH", "FLOW_SOUTH", "FLOW_EAST", "FLOW_WEST", "FLOW_UP",
	"FLOW_DOWN", "ARENA",
}

var reditSectorNames = []string{
	"Inside", "City", "Field", "Forest", "Hills", "Mountains", "Water (Swim)",
	"Water (No Swim)", "Underwater", "In Flight", "Desert", "Fire", "Earth",
	"Wind", "Water", "Swamp",
}

var reditScriptFlagNames = []string{"NONE", "ENTER", "ONPULSE", "ONDROP", "ONGET", "ONCMD"}

// cmdRedit ports the reachable SCMD_OLC_REDIT entry in do_olc. The other OLC
// command names remain unregistered; their shared surface is not part of this
// bounded room-editor goal.
func cmdRedit(s *Session, args []string) error {
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return fmt.Errorf("not logged in")
	}

	first := ""
	if len(args) > 0 {
		first = strings.ToLower(args[0])
	}
	if first == "" {
		first = strconv.Itoa(s.player.GetRoomVNum())
	}

	if strings.HasPrefix(first, "save") {
		if len(args) < 2 || args[1] == "" {
			s.reditSend("Save which zone?\r\n")
			return nil
		}
		zoneNumber := atoiC(args[1])
		zone, ok := olcZoneForVNum(s.manager.world, zoneNumber*100)
		if !ok {
			s.reditSend("Sorry, there is no zone for that number!\r\n")
			return nil
		}
		if other := s.manager.roomEditHolder(zoneNumber * 100); other != "" {
			s.reditSend(fmt.Sprintf("That room is currently being edited by %s.\r\n", other))
			return nil
		}
		if !olcAuthorized(s, zone.Number) {
			s.reditSend("You do not have permission to edit this zone.\r\n")
			return nil
		}
		s.reditSend("Saving all rooms in zone.\r\n")
		if err := saveReditZone(s.manager.world, zone); err != nil {
			slog.Error("redit disk save failed", "player", s.playerName, "zone", zone.Number, "error", err)
		}
		return nil
	}

	if !isASCIIDigit(first[0]) {
		s.reditSend("Yikes!  Stop that, someone will get hurt!\r\n")
		return nil
	}
	number := atoiC(first)
	// The duplicate-editor gate runs before zone lookup and authorization,
	// matching do_olc's descriptor scan order. C relies on its single-threaded
	// interpreter for atomicity; here the claim itself is the check, so two
	// sessions racing through cmdRedit cannot both be admitted. Every refusal
	// and failure path below releases the claim.
	if other, ok := s.manager.claimRoomEdit(number, s); !ok {
		s.reditSend(fmt.Sprintf("That room is currently being edited by %s.\r\n", other))
		return nil
	}
	zone, ok := olcZoneForVNum(s.manager.world, number)
	if !ok {
		s.manager.releaseRoomEdit(number, s)
		s.reditSend("Sorry, there is no zone for that number!\r\n")
		return nil
	}
	if !olcAuthorized(s, zone.Number) {
		s.manager.releaseRoomEdit(number, s)
		s.reditSend("You do not have permission to edit this zone.\r\n")
		return nil
	}
	if err := s.startRedit(number, zone); err != nil {
		s.manager.releaseRoomEdit(number, s)
		return err
	}
	return nil
}

func (s *Session) startRedit(number int, zone *parser.Zone) error {
	room, exists := s.manager.world.SnapshotRoom(number)
	if !exists {
		room = parser.Room{
			VNum:        number,
			Name:        "An unfinished room",
			Description: "You are in an unfinished room.\n",
			Zone:        zone.Number,
			Flags:       []string{"0", "0", "0", "0"},
			Exits:       make(map[string]parser.Exit),
		}
	}
	state := &reditState{
		room:       game.CloneRoom(room),
		number:     number,
		zoneNumber: zone.Number,
		isNew:      !exists,
		mode:       reditMainMenu,
	}
	state.extraMeta = make([]reditExtraMeta, len(state.room.ExtraDescs))
	for i, extra := range state.room.ExtraDescs {
		state.extraMeta[i] = reditExtraMeta{
			keywordSet:     extra.Keywords != "",
			descriptionSet: extra.Description != "",
		}
	}

	s.textEditMu.Lock()
	s.roomEdit = state
	s.reditDisplayMainLocked()
	s.flushReditOutputLocked()
	s.textEditMu.Unlock()

	game.Act(s.manager.world, true, s.player, nil, nil, nil,
		"$n starts using OLC.", "", game.ToRoom)
	s.player.SetPlrFlag(game.PlrWriting, true)
	return nil
}

func (s *Session) reditSend(text string) {
	if s.roomEdit != nil {
		s.roomEdit.pendingOutput += text
		return
	}
	s.reditSendDirect(text)
}

func (s *Session) reditSendDirect(text string) {
	if err := s.SendMessage(text); err != nil {
		slog.Error("redit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) flushReditOutputLocked() {
	if s.roomEdit == nil || s.roomEdit.pendingOutput == "" {
		return
	}
	text := s.roomEdit.pendingOutput
	s.roomEdit.pendingOutput = ""
	s.reditSendDirect(text)
}

func (s *Session) IsRoomEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.roomEdit != nil
}

func (s *Session) isRoomEditing() bool {
	return s.IsRoomEditing()
}

// handleReditInput is the CON_REDIT route. The dispatcher has already removed
// C's leading input whitespace before redit_parse sees the line.
func (s *Session) handleReditInput(line string) {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.roomEdit == nil {
		return
	}
	line = strings.TrimLeft(line, " \t\r\n\v\f")
	if s.textEdit != nil {
		s.handleTextEditInputLocked(line)
		s.flushReditOutputLocked()
		return
	}
	s.parseReditLocked(line)
	s.flushReditOutputLocked()
}

func (s *Session) cancelRoomEdit() {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.roomEdit == nil {
		return
	}
	// A disconnect follows cleanup_olc(CLEANUP_ALL): no room commit and no
	// menu callback. Live script fields already changed through the C-shallow
	// script pointer intentionally remain changed.
	number := s.roomEdit.number
	s.textEdit = nil
	s.roomEdit = nil
	if s.manager != nil {
		s.manager.releaseRoomEdit(number, s)
	}
	if s.player != nil {
		s.player.SetPlrFlag(game.PlrWriting, false)
		game.Act(s.manager.world, true, s.player, nil, nil, nil,
			"$n stops using OLC.", "", game.ToRoom)
	}
}

func (s *Session) finishReditLocked(save bool) {
	state := s.roomEdit
	if state == nil {
		return
	}
	if save {
		state.room.VNum = state.number
		if !state.isNew {
			// C's OLC room copy shallow-copies the live script pointer. Script
			// menu changes therefore survive a later whole-room assignment.
			if live, ok := s.manager.world.SnapshotRoom(state.number); ok {
				state.room.ScriptName = live.ScriptName
				state.room.ScriptFunctions = live.ScriptFunctions
			}
		}
		// Hold the zone save lock across the world commit and the dirty-marker
		// set so the pair is atomic against a concurrent saveReditZone:
		// without this, a commit landing between the save's snapshot and its
		// marker cleanup would be cleared from the save list without being
		// persisted.
		saveMu := zoneSaveLock(state.zoneNumber)
		saveMu.Lock()
		committed := s.manager.world.CommitEditedRoom(state.room)
		if committed {
			reditAddSaveRoom(state.zoneNumber)
		}
		saveMu.Unlock()
	}
	// Both the save and abort exits of REDIT_CONFIRM_SAVESTRING end the
	// editor; the duplicate-editor reservation must not outlive either.
	s.manager.releaseRoomEdit(state.number, s)
	s.textEdit = nil
	s.roomEdit = nil
	if s.player != nil {
		s.player.SetPlrFlag(game.PlrWriting, false)
		game.Act(s.manager.world, true, s.player, nil, nil, nil,
			"$n stops using OLC.", "", game.ToRoom)
	}
	if save {
		s.reditSend("Room saved to memory.\r\n")
	}
}

func (s *Session) parseReditLocked(line string) {
	state := s.roomEdit
	if state == nil {
		return
	}
	switch state.mode {
	case reditConfirmSaveString:
		switch firstByte(line) {
		case 'y', 'Y':
			s.finishReditLocked(true)
		case 'n', 'N':
			s.finishReditLocked(false)
		default:
			s.reditSend("Invalid choice!\r\n")
			s.reditSend("Do you wish to save this room internally? : ")
		}
		return
	case reditMainMenu:
		s.parseReditMainLocked(line)
	case reditName:
		applyOLC(olc.Operation{Kind: olc.OpSetRoomName, Room: &state.room, Text: line})
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	case reditFlags:
		s.parseReditFlagsLocked(line)
	case reditSector:
		s.parseReditSectorLocked(line)
	case reditExitMenu:
		s.parseReditExitMenuLocked(line)
	case reditExitNumber:
		s.parseReditExitNumberLocked(line)
	case reditExitKeyword:
		exit := s.reditEnsureExitLocked(state.value)
		exit.Keywords = line
		state.room.Exits[game.DirectionNames[state.value]] = exit
		state.mode = reditExitMenu
		s.reditDisplayExitMenuLocked()
	case reditExitKey:
		exit := s.reditEnsureExitLocked(state.value)
		exit.Key = atoiC(line)
		state.room.Exits[game.DirectionNames[state.value]] = exit
		state.mode = reditExitMenu
		s.reditDisplayExitMenuLocked()
	case reditExitDoorFlags:
		s.parseReditDoorFlagsLocked(line)
	case reditExtraKey:
		if state.currentExtra < len(state.room.ExtraDescs) {
			state.room.ExtraDescs[state.currentExtra].Keywords = line
			state.extraMeta[state.currentExtra].keywordSet = true
		}
		state.mode = reditExtraMenu
		s.reditDisplayExtraMenuLocked()
	case reditExtraMenu:
		s.parseReditExtraMenuLocked(line)
	case reditCopy:
		s.parseReditCopyLocked(line)
	case reditScriptMenu:
		s.parseReditScriptMenuLocked(line)
	case reditScriptName:
		current, ok := s.manager.world.SnapshotRoom(state.number)
		if ok {
			if err := s.manager.world.SetRoomScript(state.number, current.ScriptFunctions, line); !err {
				slog.Warn("redit script name update lost room", "room", state.number)
			}
		}
		s.reditDisplayScriptMenuLocked()
	case reditScriptFlags:
		s.parseReditScriptFlagsLocked(line)
	default:
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	}
}

func (s *Session) parseReditMainLocked(line string) {
	state := s.roomEdit
	if state == nil {
		return
	}
	switch firstByte(line) {
	case 'q', 'Q':
		if state.olcVal != 0 {
			state.mode = reditConfirmSaveString
			s.reditSend("Do you wish to save this room internally? : ")
		} else {
			s.finishReditLocked(false)
		}
	case '1':
		state.mode = reditName
		s.reditSend("Enter room name:-\r\n| ")
	case '2':
		state.mode = reditDescription
		state.olcVal = 1
		s.startReditStringLocked(reditRoomDescription)
	case '3':
		state.mode = reditFlags
		s.reditDisplayFlagsLocked()
	case '4':
		state.mode = reditSector
		s.reditDisplaySectorLocked()
	case '5', '6', '7', '8', '9', 'a', 'A':
		direction := int(firstByte(line) - '5')
		if firstByte(line) == 'a' || firstByte(line) == 'A' {
			direction = 5
		}
		state.value = direction
		state.olcVal = direction
		state.mode = reditExitMenu
		s.reditDisplayExitMenuLocked()
	case 'b', 'B':
		if len(state.room.ExtraDescs) == 0 {
			state.room.ExtraDescs = append(state.room.ExtraDescs, parser.ExtraDesc{})
			state.extraMeta = append(state.extraMeta, reditExtraMeta{})
		}
		state.currentExtra = 0
		state.olcVal = 1
		state.mode = reditExtraMenu
		s.reditDisplayExtraMenuLocked()
	case 'c', 'C':
		state.mode = reditCopy
		s.reditSend("Enter virtual number of room to copy : ")
	case 's', 'S':
		state.mode = reditScriptMenu
		s.reditDisplayScriptMenuLocked()
	default:
		s.reditSend("Invalid choice!")
		s.reditDisplayMainLocked()
	}
}

func (s *Session) parseReditFlagsLocked(line string) {
	state := s.roomEdit
	number := atoiC(line)
	if number < 0 || number > len(reditRoomFlagNames) {
		s.reditSend("That's not a valid choice!\r\n")
		s.reditDisplayFlagsLocked()
		return
	}
	if number == 0 {
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
		return
	}
	reditToggleRoomFlag(&state.room, number-1)
	s.reditDisplayFlagsLocked()
}

func (s *Session) parseReditSectorLocked(line string) {
	state := s.roomEdit
	number := atoiC(line)
	if number < 0 || number >= len(reditSectorNames) {
		s.reditSend("Invalid choice!")
		s.reditDisplaySectorLocked()
		return
	}
	state.room.Sector = number
	state.olcVal = 1
	state.mode = reditMainMenu
	s.reditDisplayMainLocked()
}

func (s *Session) parseReditExitMenuLocked(line string) {
	state := s.roomEdit
	switch firstByte(line) {
	case '0':
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	case '1':
		state.mode = reditExitNumber
		s.reditSend("Exit to room number : ")
	case '2':
		state.mode = reditExitDescription
		s.startReditStringLocked(reditExitDescriptionField)
	case '3':
		state.mode = reditExitKeyword
		s.reditSend("Enter keywords : ")
	case '4':
		state.mode = reditExitKey
		s.reditSend("Enter key number : ")
	case '5':
		state.mode = reditExitDoorFlags
		s.reditDisplayExitFlagLocked()
	case '6':
		delete(state.room.Exits, game.DirectionNames[state.value])
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	default:
		s.reditSend("Try again : ")
	}
}

func (s *Session) parseReditExitNumberLocked(line string) {
	state := s.roomEdit
	number := atoiC(line)
	if number != -1 {
		if _, ok := s.manager.world.SnapshotRoom(number); !ok {
			s.reditSend("That room does not exist, try again : ")
			return
		}
	}
	exit := s.reditEnsureExitLocked(state.value)
	exit.ToRoom = number
	state.room.Exits[game.DirectionNames[state.value]] = exit
	state.mode = reditExitMenu
	s.reditDisplayExitMenuLocked()
}

func (s *Session) parseReditDoorFlagsLocked(line string) {
	state := s.roomEdit
	number := atoiC(line)
	if number < 0 || number > 2 {
		s.reditSend("That's not a valid choice!\r\n")
		s.reditDisplayExitFlagLocked()
		return
	}
	exit := s.reditEnsureExitLocked(state.value)
	switch number {
	case 0:
		exit.ExitInfo = 0
		exit.DoorState = 0
	case 1:
		exit.ExitInfo = parser.ExitIsDoor
		exit.DoorState = 1
	case 2:
		exit.ExitInfo = parser.ExitIsDoor | parser.ExitPickproof
		exit.DoorState = 2
	}
	state.room.Exits[game.DirectionNames[state.value]] = exit
	state.mode = reditExitMenu
	s.reditDisplayExitMenuLocked()
}

func (s *Session) parseReditExtraMenuLocked(line string) {
	state := s.roomEdit
	number := atoiC(line)
	if number == 0 {
		if state.currentExtra < len(state.room.ExtraDescs) &&
			(!state.extraMeta[state.currentExtra].keywordSet || !state.extraMeta[state.currentExtra].descriptionSet) {
			// C unlinks the current node by writing NULL through the predecessor,
			// dropping the remainder of the linked list as a side effect.
			state.room.ExtraDescs = state.room.ExtraDescs[:state.currentExtra]
			state.extraMeta = state.extraMeta[:state.currentExtra]
			state.currentExtra = 0
		}
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
		return
	}
	switch number {
	case 1:
		state.mode = reditExtraKey
		s.reditSend("Enter keywords, separated by spaces : ")
	case 2:
		state.mode = reditExtraDescription
		s.startReditStringLocked(reditExtraDescriptionField)
	case 3:
		if state.currentExtra >= len(state.room.ExtraDescs) ||
			!state.extraMeta[state.currentExtra].keywordSet ||
			!state.extraMeta[state.currentExtra].descriptionSet {
			s.reditSend("You can't edit the next extra desc without completing this one.\r\n")
			s.reditDisplayExtraMenuLocked()
			return
		}
		if state.currentExtra+1 < len(state.room.ExtraDescs) {
			state.currentExtra++
		} else {
			state.room.ExtraDescs = append(state.room.ExtraDescs, parser.ExtraDesc{})
			state.extraMeta = append(state.extraMeta, reditExtraMeta{})
			state.currentExtra++
		}
		state.mode = reditExtraMenu
		s.reditDisplayExtraMenuLocked()
	default:
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	}
}

func (s *Session) parseReditCopyLocked(line string) {
	state := s.roomEdit
	number := atoiC(line)
	if number != -1 {
		room, ok := s.manager.world.SnapshotRoom(number)
		if !ok {
			s.reditSend("That room does not exist, try again : ")
			return
		}
		applyOLC(olc.Operation{Kind: olc.OpSetRoomName, Room: &state.room, Text: room.Name})
		applyOLC(olc.Operation{Kind: olc.OpSetRoomDescription, Room: &state.room, Text: room.Description})
	}
	state.olcVal = 1
	state.mode = reditMainMenu
	s.reditDisplayMainLocked()
}

func (s *Session) parseReditScriptMenuLocked(line string) {
	state := s.roomEdit
	switch atoiC(line) {
	case 0:
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	case 1:
		state.mode = reditScriptName
		s.reditSend("Enter script name: ")
	case 2:
		state.mode = reditScriptFlags
		s.reditDisplayScriptFlagsLocked()
	default:
		state.olcVal = 1
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	}
}

func (s *Session) parseReditScriptFlagsLocked(line string) {
	number := atoiC(line)
	if number == 0 {
		s.roomEdit.mode = reditScriptMenu
		s.reditDisplayScriptMenuLocked()
		return
	}
	if number > 0 && number <= len(reditScriptFlagNames) {
		room, ok := s.manager.world.SnapshotRoom(s.roomEdit.number)
		if ok {
			flags := room.ScriptFunctions ^ (1 << uint(number-1))
			if !s.manager.world.SetRoomScript(s.roomEdit.number, flags, room.ScriptName) {
				slog.Warn("redit script flag update lost room", "room", s.roomEdit.number)
			}
		}
	}
	s.reditDisplayScriptFlagsLocked()
}

func (s *Session) reditEnsureExitLocked(direction int) parser.Exit {
	state := s.roomEdit
	name := game.DirectionNames[direction]
	if exit, ok := state.room.Exits[name]; ok {
		return exit
	}
	toRoom := 0
	if first, ok := s.manager.world.RoomVNumByIndex(0); ok {
		toRoom = first
	}
	exit := parser.Exit{Direction: name, ToRoom: toRoom}
	state.room.Exits[name] = exit
	return exit
}

func (s *Session) startReditStringLocked(field reditStringField) {
	state := s.roomEdit
	initial := ""
	maxBytes := 0
	switch field {
	case reditRoomDescription:
		initial = state.room.Description
		maxBytes = olc.MaxRoomDesc
	case reditExitDescriptionField:
		exit := s.reditEnsureExitLocked(state.value)
		initial = exit.Description
		maxBytes = olc.MaxExitDesc
	case reditExtraDescriptionField:
		if state.currentExtra < len(state.room.ExtraDescs) && state.extraMeta[state.currentExtra].descriptionSet {
			initial = state.room.ExtraDescs[state.currentExtra].Description
		}
		maxBytes = olc.MaxExtraDesc
	}
	initial = editorCRLF(initial)
	s.textEdit = &textEditState{
		field:      textEditField{maxBytes: maxBytes},
		original:   initial,
		buffer:     initial,
		roomEditor: true,
		onComplete: func(action textEditAction, buffer, original string) {
			s.finishReditStringLocked(field, action, buffer, original)
		},
	}
	switch field {
	case reditRoomDescription:
		s.reditSend("Instructions: /s or @ to save, /h for more options.\r\n" +
			"Enter room description:\r\n\r\n" + initial)
	case reditExitDescriptionField:
		s.reditSend("Instructions: /s or @ to save, /h for more options.\r\n" +
			"Enter exit description:\r\n\r\n" + initial)
	case reditExtraDescriptionField:
		s.reditSend("Instructions: /s or @ to save, /h for more options.\r\n" +
			"Enter extra description:\r\n\r\n" + initial)
	}
}

func (s *Session) finishReditStringLocked(field reditStringField, action textEditAction, buffer, original string) {
	state := s.roomEdit
	if state == nil {
		return
	}
	if action == textEditSave {
		value := editorToRoomText(buffer)
		switch field {
		case reditRoomDescription:
			applyOLC(olc.Operation{Kind: olc.OpSetRoomDescription, Room: &state.room, Text: value})
		case reditExitDescriptionField:
			exit := s.reditEnsureExitLocked(state.value)
			applyOLC(olc.Operation{Kind: olc.OpSetExitDescription, Exit: &exit, Text: value})
			state.room.Exits[game.DirectionNames[state.value]] = exit
		case reditExtraDescriptionField:
			if state.currentExtra < len(state.room.ExtraDescs) {
				applyOLC(olc.Operation{
					Kind:  olc.OpSetExtraDescription,
					Extra: &state.room.ExtraDescs[state.currentExtra],
					Text:  value,
				})
				state.extraMeta[state.currentExtra].descriptionSet = true
			}
		}
	}
	// Abort intentionally leaves the working target untouched: C string_add
	// restored d->backstr before redit_string_cleanup was called.
	switch field {
	case reditRoomDescription:
		state.mode = reditMainMenu
		s.reditDisplayMainLocked()
	case reditExitDescriptionField:
		state.mode = reditExitMenu
		s.reditDisplayExitMenuLocked()
	case reditExtraDescriptionField:
		state.mode = reditExtraMenu
		s.reditDisplayExtraMenuLocked()
	}
}

func editorToRoomText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func reditToggleRoomFlag(room *parser.Room, bit int) {
	if room == nil || bit < 0 || bit >= 28 {
		return
	}
	for len(room.Flags) < 4 {
		room.Flags = append(room.Flags, "0")
	}
	word := bit / 32
	bitInWord := uint(bit % 32)
	value, err := strconv.ParseUint(room.Flags[word], 10, 32)
	if err != nil {
		value = 0
	}
	value ^= 1 << bitInWord
	room.Flags[word] = strconv.FormatUint(value, 10)
}

func reditRoomFlagSet(room parser.Room, bit int) bool {
	if bit < 0 || bit >= 28 || bit/32 >= len(room.Flags) {
		return false
	}
	value, err := strconv.ParseUint(room.Flags[bit/32], 10, 32)
	return err == nil && value&(1<<uint(bit%32)) != 0
}

func reditRoomFlags(room parser.Room) string {
	var out strings.Builder
	for bit, name := range reditRoomFlagNames {
		if reditRoomFlagSet(room, bit) {
			out.WriteString(name)
			out.WriteByte(' ')
		}
	}
	if out.Len() == 0 {
		return "NOBITS "
	}
	return out.String()
}

func reditScriptFlagsText(flags int) string {
	var out strings.Builder
	for bit, name := range reditScriptFlagNames {
		if flags&(1<<uint(bit)) != 0 {
			out.WriteString(name)
			out.WriteByte(' ')
		}
	}
	if out.Len() == 0 {
		return "NOBITS "
	}
	return out.String()
}

// reditCols mirrors get_char_cols (src/olc.c:416): the OLC menu color strings
// for this character. screen.h's CC* macros emit the K* ANSI codes only when
// the character's color level (PRF_COLOR_1=1 + PRF_COLOR_2=2) reaches C_NRM
// (2), i.e. exactly when the PRF_COLOR_2 bit is set; creation's single Y sets
// both bits (interpreter.c CON_COLOR).
func (s *Session) reditCols() (nrm, grn, cyn, yel string) {
	if s.player != nil && s.player.GetFlags()&(1<<uint(game.PrfColor2)) != 0 {
		return "\x1b[0m", "\x1b[32m", "\x1b[36m", "\x1b[33m"
	}
	return "", "", "", ""
}

func (s *Session) reditDisplayMainLocked() {
	state := s.roomEdit
	if state == nil {
		return
	}
	nrm, grn, cyn, yel := s.reditCols()
	var out strings.Builder
	fmt.Fprintf(&out, "\r\n-- Room number : [%s%d%s]      Room zone: [%s%d%s]\r\n",
		cyn, state.number, nrm, cyn, state.zoneNumber, nrm)
	fmt.Fprintf(&out, "%s1%s) Name        : %s%s\r\n", grn, nrm, yel, state.room.Name)
	fmt.Fprintf(&out, "%s2%s) Description :\r\n%s%s", grn, nrm, yel, editorCRLF(state.room.Description))
	fmt.Fprintf(&out, "%s3%s) Room flags  : %s%s\r\n", grn, nrm, cyn, reditRoomFlags(state.room))
	sector := "<INVALID>"
	if state.room.Sector >= 0 && state.room.Sector < len(reditSectorNames) {
		sector = reditSectorNames[state.room.Sector]
	}
	fmt.Fprintf(&out, "%s4%s) Sector type : %s%s\r\n", grn, nrm, cyn, sector)
	exitLabels := []string{
		") Exit north  : ",
		") Exit east   : ",
		") Exit south  : ",
		") Exit west   : ",
		") Exit up     : ",
		") Exit down   : ",
	}
	exitKeys := []byte{'5', '6', '7', '8', '9', 'A'}
	for i, direction := range game.DirectionNames {
		target := -1
		if exit, ok := state.room.Exits[direction]; ok {
			target = s.reditExitTargetDisplay(exit.ToRoom)
		}
		fmt.Fprintf(&out, "%s%c%s%s%s%d\r\n", grn, exitKeys[i], nrm, exitLabels[i], cyn, target)
	}
	fmt.Fprintf(&out, "%sB%s) Extra descriptions menu\r\n", grn, nrm)
	fmt.Fprintf(&out, "%sC%s) Copy another room description\r\n", grn, nrm)
	fmt.Fprintf(&out, "%sS%s) Script menu\r\n", grn, nrm)
	fmt.Fprintf(&out, "%sQ%s) Quit\r\n", grn, nrm)
	out.WriteString("Enter choice : ")
	s.reditSend(out.String())
	state.mode = reditMainMenu
}

func (s *Session) reditExitTargetDisplay(target int) int {
	if target == -1 {
		return -1
	}
	if _, ok := s.manager.world.SnapshotRoom(target); !ok {
		return -1
	}
	return target
}

func (s *Session) reditDisplayFlagsLocked() {
	state := s.roomEdit
	nrm, grn, cyn, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	for i, name := range reditRoomFlagNames {
		fmt.Fprintf(&out, "%s%2d%s) %-20.20s ", grn, i+1, nrm, name)
		if (i+1)%2 == 0 {
			out.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&out, "\r\nRoom flags: %s%s%s\r\nEnter room flags, 0 to quit : ", cyn, reditRoomFlags(state.room), nrm)
	s.reditSend(out.String())
	state.mode = reditFlags
}

func (s *Session) reditDisplaySectorLocked() {
	state := s.roomEdit
	// redit_disp_sector_menu (redit.c:541-556) reads the grn/nrm globals
	// without refreshing them through get_char_cols; in the reachable flow the
	// main menu just set them for this same character, so the bytes equal this
	// character's own colors.
	nrm, grn, _, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	for i, name := range reditSectorNames {
		fmt.Fprintf(&out, "%s%2d%s) %-20.20s ", grn, i, nrm, name)
		if (i+1)%2 == 0 {
			out.WriteString("\r\n")
		}
	}
	out.WriteString("\r\nEnter sector type : ")
	s.reditSend(out.String())
	state.mode = reditSector
}

func (s *Session) reditDisplayExitMenuLocked() {
	state := s.roomEdit
	exit := s.reditEnsureExitLocked(state.value)
	toRoom := s.reditExitTargetDisplay(exit.ToRoom)
	door := "No door"
	if exit.ExitInfo&parser.ExitIsDoor != 0 {
		door = "Is a door"
		if exit.ExitInfo&parser.ExitPickproof != 0 {
			door = "Pickproof"
		}
	}
	nrm, grn, cyn, yel := s.reditCols()
	var out strings.Builder
	fmt.Fprintf(&out, "\r\n%s1%s) Exit to     : %s%d\r\n", grn, nrm, cyn, toRoom)
	fmt.Fprintf(&out, "%s2%s) Description :-\r\n%s%s\r\n", grn, nrm, yel, reditDisplayValue(exit.Description))
	fmt.Fprintf(&out, "%s3%s) Door name   : %s%s\r\n", grn, nrm, yel, reditDisplayValue(exit.Keywords))
	fmt.Fprintf(&out, "%s4%s) Key         : %s%d\r\n", grn, nrm, cyn, exit.Key)
	fmt.Fprintf(&out, "%s5%s) Door flags  : %s%s\r\n", grn, nrm, cyn, door)
	fmt.Fprintf(&out, "%s6%s) Purge exit.\r\n", grn, nrm)
	out.WriteString("Enter choice, 0 to quit : ")
	s.reditSend(out.String())
	state.mode = reditExitMenu
}

func reditDisplayValue(value string) string {
	if value == "" {
		return "<NONE>"
	}
	return editorCRLF(value)
}

func (s *Session) reditDisplayExitFlagLocked() {
	nrm, grn, _, _ := s.reditCols()
	s.reditSend(fmt.Sprintf("%s0%s) No door\r\n%s1%s) Closeable door\r\n%s2%s) Pickproof\r\nEnter choice : ",
		grn, nrm, grn, nrm, grn, nrm))
}

func (s *Session) reditDisplayExtraMenuLocked() {
	state := s.roomEdit
	if state.currentExtra >= len(state.room.ExtraDescs) {
		state.currentExtra = 0
	}
	extra := state.room.ExtraDescs[state.currentExtra]
	meta := state.extraMeta[state.currentExtra]
	keyword := "<NONE>"
	if meta.keywordSet {
		keyword = extra.Keywords
	}
	description := "<NONE>"
	if meta.descriptionSet {
		description = reditDisplayValue(extra.Description)
	}
	nrm, grn, _, yel := s.reditCols()
	var out strings.Builder
	fmt.Fprintf(&out, "\r\n%s1%s) Keyword: %s%s\r\n", grn, nrm, yel, keyword)
	fmt.Fprintf(&out, "%s2%s) Description:\r\n%s%s\r\n", grn, nrm, yel, description)
	fmt.Fprintf(&out, "%s3%s) Goto next description: ", grn, nrm)
	if state.currentExtra+1 < len(state.room.ExtraDescs) {
		out.WriteString("Set.\r\n")
	} else {
		out.WriteString("<NOT SET>\r\n")
	}
	out.WriteString("Enter choice (0 to quit) : ")
	s.reditSend(out.String())
	state.mode = reditExtraMenu
}

func (s *Session) reditDisplayScriptFlagsLocked() {
	state := s.roomEdit
	room, ok := s.manager.world.SnapshotRoom(state.number)
	if !ok {
		room = state.room
	}
	nrm, grn, cyn, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[J")
	for i, name := range reditScriptFlagNames {
		fmt.Fprintf(&out, "%s%2d%s) %-20.20s  ", grn, i+1, nrm, name)
		if (i+1)%2 == 0 {
			out.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&out, "\r\nCurrent flags   : %s%s%s\r\nEnter script flags (0 to quit) : ", cyn, reditScriptFlagsText(room.ScriptFunctions), nrm)
	s.reditSend(out.String())
	state.mode = reditScriptFlags
}

func (s *Session) reditDisplayScriptMenuLocked() {
	state := s.roomEdit
	if state.isNew {
		s.reditSend("\r\nCannot assign a script until the room is saved at least once.\r\n")
		s.reditDisplayMainLocked()
		return
	}
	room, ok := s.manager.world.SnapshotRoom(state.number)
	if !ok {
		room = state.room
	}
	name := room.ScriptName
	if name == "" {
		name = "None"
	}
	nrm, grn, _, yel := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	fmt.Fprintf(&out, "%s1%s) Name: %s%s\r\n", grn, nrm, yel, name)
	fmt.Fprintf(&out, "%s2%s) Script Flags: %s%s%s\r\n", grn, nrm, yel, reditScriptFlagsText(room.ScriptFunctions), nrm)
	out.WriteString("Enter choice (0 to quit) : ")
	s.reditSend(out.String())
	state.mode = reditScriptMenu
}

func reditAddSaveRoom(zone int) {
	markOLCDirty(olcKindRoom, zone)
}

func reditRemoveSaveRoom(zone int) {
	clearOLCDirty(olcKindRoom, zone)
}

func saveReditZone(world *game.World, zone *parser.Zone) error {
	// Serialize the whole snapshot -> write -> save-list cleanup against
	// concurrent saves of this zone and against working-copy commits (see
	// finishReditLocked): a commit landing mid-save is either fully inside
	// the snapshot or keeps its dirty marker, never silently marked saved.
	saveMu := zoneSaveLock(zone.Number)
	saveMu.Lock()
	defer saveMu.Unlock()

	rooms := world.SnapshotRooms()
	var out strings.Builder
	minimum := zone.Number * 100
	for i := range rooms {
		room := rooms[i]
		if room.VNum < minimum || room.VNum > zone.TopRoom {
			continue
		}
		fmt.Fprintf(&out, "#%d\n%s~\n%s~\n%d %s %s %s %s %d\n",
			room.VNum,
			room.Name,
			diskReditString(room.Description),
			zone.Number,
			reditFlagWord(room, 0),
			reditFlagWord(room, 1),
			reditFlagWord(room, 2),
			reditFlagWord(room, 3),
			room.Sector,
		)
		if room.ScriptName != "" {
			fmt.Fprintf(&out, "R %s %d\n", room.ScriptName, room.ScriptFunctions)
		}
		for direction, name := range game.DirectionNames {
			exit, ok := room.Exits[name]
			if !ok {
				continue
			}
			doorState := 0
			if exit.ExitInfo&parser.ExitIsDoor != 0 {
				doorState = 1
				if exit.ExitInfo&parser.ExitPickproof != 0 {
					doorState = 2
				}
			}
			fmt.Fprintf(&out, "D%d\n%s~\n%s~\n%d %d %d\n",
				direction,
				diskReditString(exit.Description),
				diskReditString(exit.Keywords),
				doorState,
				exit.Key,
				exit.ToRoom,
			)
		}
		for _, extra := range room.ExtraDescs {
			fmt.Fprintf(&out, "E\n%s~\n%s~\n", extra.Keywords, diskReditString(extra.Description))
		}
		out.WriteString("S\n")
	}
	out.WriteString("$~\n")

	path := filepath.Join(world.WorldPath, "wld", fmt.Sprintf("%d.wld", zone.Number))
	if err := atomicWriteFile(path, []byte(out.String()), 0o666); err != nil {
		return err
	}
	reditRemoveSaveRoom(zone.Number)
	return nil
}

func reditFlagWord(room parser.Room, index int) string {
	if index >= 0 && index < len(room.Flags) && room.Flags[index] != "" {
		return room.Flags[index]
	}
	return "0"
}

func diskReditString(text string) string {
	return strings.ReplaceAll(text, "\r", "")
}
