package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// zeditMode mirrors the ZEDIT_* values in src/olc.h. ZEDIT works on a
// room-filtered reset-command list, not on a single prototype entity.
type zeditMode uint8

const (
	zeditMainMenu zeditMode = iota + 1
	zeditConfirmSaveString
	zeditNewEntry
	zeditDeleteEntry
	zeditChangeEntry
	zeditCommandType
	zeditIfFlag
	zeditArg1
	zeditArg2
	zeditArg3
	zeditZoneNameMode
	zeditZoneReset
	zeditZoneLife
	zeditZoneTop
)

type zeditState struct {
	zone          parser.Zone
	roomVNum      int
	roomRNum      int
	zoneIndex     int
	zoneNumber    int
	position      int // C OLC_VAL while a reset command is selected.
	dirtyHeader   bool
	dirtyCommands bool
	mode          zeditMode
	pendingOutput string
}

// cmdZedit ports do_olc's SCMD_OLC_ZEDIT entry path (src/olc.c:80-277).
// The editor key is a room VNUM: the containing zone is selected from that
// room, and the working reset list contains only commands scoped to it.
func cmdZedit(s *Session, args []string) error {
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return fmt.Errorf("not logged in")
	}

	var first, second string
	if len(args) > 0 {
		first = args[0]
	}
	if len(args) > 1 {
		second = args[1]
	}
	save := strings.HasPrefix(first, "save")

	if first == "" {
		if save {
			s.zeditSend("Save which zone?\r\n")
			return nil
		}
		first = strconv.Itoa(s.player.GetRoomVNum())
	}

	var number int
	if !isASCIIDigit(firstByte(first)) {
		if save {
			if second == "" {
				s.zeditSend("Save which zone?\r\n")
				return nil
			}
			number = atoiC(second) * 100
		} else if getEffectiveLevel(s) >= LVL_HIGOD {
			// C routes every nonnumeric ZEDIT argument through the
			// high-god branch: only the exact `new <zone>` shape creates
			// a zone, and every other spelling gets this same prompt.
			if strings.HasPrefix(first, "new") && second != "" {
				return zeditNewZone(s, atoiC(second))
			}
			s.zeditSend("Specify a new zone number.\r\n")
			return nil
		} else {
			s.zeditSend("Yikes!  Stop that, someone will get hurt!\r\n")
			return nil
		}
	} else {
		number = atoiC(first)
	}

	if save {
		// C's duplicate scan runs on the save path too, against the room
		// number supplied to do_olc; saving never reserves that room.
		if other := s.manager.zoneEditHolder(number); other != "" {
			s.zeditSend(fmt.Sprintf("That room is currently being edited by %s.\r\n", other))
			return nil
		}
		zone, ok := olcZoneForVNum(s.manager.world, number)
		if !ok {
			s.zeditSend("Sorry, there is no zone for that number!\r\n")
			return nil
		}
		if !olcAuthorized(s, zone.Number) {
			s.zeditSend("You do not have permission to edit this zone.\r\n")
			return nil
		}
		s.zeditSend("Saving all zone information.\r\n")
		if err := saveZeditZone(s.manager.world, zone); err != nil {
			slog.Error("zedit disk save failed", "player", s.playerName, "zone", zone.Number, "error", err)
		}
		return nil
	}

	// The duplicate-editor gate runs before zone lookup and authorization,
	// matching do_olc's descriptor scan. The atomic claim replaces C's
	// single-threaded scan, and every refusal/failure below releases it.
	if other, claimed := s.manager.claimZoneEdit(number, s); !claimed {
		s.zeditSend(fmt.Sprintf("That room is currently being edited by %s.\r\n", other))
		return nil
	}
	zone, ok := olcZoneForVNum(s.manager.world, number)
	if !ok {
		s.manager.releaseZoneEdit(number, s)
		s.zeditSend("Sorry, there is no zone for that number!\r\n")
		return nil
	}
	if !olcAuthorized(s, zone.Number) {
		s.manager.releaseZoneEdit(number, s)
		s.zeditSend("You do not have permission to edit this zone.\r\n")
		return nil
	}
	if _, exists := s.manager.world.SnapshotRoom(number); !exists {
		s.manager.releaseZoneEdit(number, s)
		s.zeditSend("That room does not exist.\r\n")
		return nil
	}
	if err := s.startZedit(number, zone); err != nil {
		s.manager.releaseZoneEdit(number, s)
		return err
	}
	return nil
}

func (s *Session) startZedit(number int, zone *parser.Zone) error {
	roomRNum, ok := s.manager.world.RealRoomIndex(number)
	if !ok {
		return fmt.Errorf("room %d has no rnum", number)
	}
	zoneCopy, ok := s.manager.world.SnapshotZone(zone.Number)
	if !ok {
		return fmt.Errorf("zone %d disappeared", zone.Number)
	}
	zoneCopy.Commands = zeditSetupCommands(zoneCopy.Commands, number)
	zoneIndex := 0
	for i, candidate := range s.manager.world.GetAllZones() {
		if candidate.Number == zone.Number {
			zoneIndex = i
			break
		}
	}

	s.textEditMu.Lock()
	s.zedit = &zeditState{
		zone:       zoneCopy,
		roomVNum:   number,
		roomRNum:   roomRNum,
		zoneIndex:  zoneIndex,
		zoneNumber: zone.Number,
		mode:       zeditMainMenu,
	}
	s.setPlayerWritingLocked(true)
	s.zeditShowMenuLocked()
	s.flushZeditOutputLocked()
	s.textEditMu.Unlock()

	if s.player != nil {
		game.Act(s.manager.world, true, s.player, nil, nil, nil, "$n starts using OLC.", "", game.ToRoom)
	}
	return nil
}

func (s *Session) finishZeditLocked(save bool) {
	state := s.zedit
	if state == nil {
		return
	}
	if save {
		saveMu := zoneSaveLock(state.zoneNumber)
		saveMu.Lock()
		if s.manager.world.CommitEditedZone(state.zoneNumber, state.roomVNum, state.zone) {
			markOLCDirty(olcKindZone, state.zoneNumber)
		}
		saveMu.Unlock()
	}
	s.flushZeditOutputLocked()
	s.setPlayerWritingLocked(false)
	if s.olcActor() != nil {
		game.Act(s.manager.world, true, s.player, nil, nil, nil, "$n stops using OLC.", "", game.ToRoom)
	}
	s.zedit = nil
	if s.manager != nil {
		s.manager.releaseZoneEdit(state.roomVNum, s)
	}
}

func (s *Session) zeditSend(text string) {
	if s.zedit != nil {
		s.zedit.pendingOutput += text
		return
	}
	if err := s.SendMessage(text); err != nil {
		slog.Error("zedit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) flushZeditOutputLocked() {
	if s.zedit == nil || s.zedit.pendingOutput == "" {
		return
	}
	text := s.zedit.pendingOutput
	s.zedit.pendingOutput = ""
	if err := s.SendMessage(text); err != nil {
		slog.Error("zedit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) handleZeditInput(line string) {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.zedit == nil {
		return
	}
	s.parseZeditLocked(strings.TrimLeft(line, " \t\r\n\v\f"))
	s.flushZeditOutputLocked()
}

// IsZoneEditing reports whether this descriptor owns CON_ZEDIT.
func (s *Session) IsZoneEditing() bool { return s.isZoneEditing() }

func (s *Session) isZoneEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.zedit != nil
}

func (s *Session) zeditCols() (nrm, grn, cyn, yel string) {
	return s.reditCols()
}

// zeditSetupCommands ports zedit_setup's stale-carry filter. The parser's
// command fields are VNUMs, so setup compares the carried room value directly
// with the requested room VNUM. G/E/P/* deliberately do not reset cmdRoom.
func zeditSetupCommands(commands []parser.ZoneCommand, roomVNum int) []parser.ZoneCommand {
	return olc.ZoneCommandsForRoom(commands, roomVNum)
}

func zeditEquipmentName(index int) string {
	if index >= 0 && index < len(olc.ZoneEquipmentNames) {
		return olc.ZoneEquipmentNames[index]
	}
	return "\n"
}

func zeditCommandName(w *game.World, cmd parser.ZoneCommand) string {
	switch cmd.Command {
	case "M":
		if mob, ok := w.GetMobPrototype(cmd.Arg1); ok {
			return mob.ShortDesc
		}
	case "G", "O", "E", "P":
		if obj, ok := w.GetObjPrototype(cmd.Arg1); ok {
			return obj.ShortDesc
		}
	}
	return ""
}

func zeditContainerName(w *game.World, vnum int) string {
	if obj, ok := w.GetObjPrototype(vnum); ok {
		return obj.ShortDesc
	}
	return ""
}

func zeditRemoveName(w *game.World, cmd parser.ZoneCommand) (string, int) {
	if cmd.Arg2 != 0 {
		if obj, ok := w.GetObjPrototype(cmd.Arg3); ok {
			return obj.ShortDesc, cmd.Arg3
		}
		return "", cmd.Arg3
	}
	if mob, ok := w.GetMobPrototype(cmd.Arg3); ok {
		return mob.ShortDesc, cmd.Arg3
	}
	return "", cmd.Arg3
}

func (s *Session) zeditShowMenuLocked() {
	state := s.zedit
	if state == nil {
		return
	}
	nrm, grn, cyn, yel := s.zeditCols()
	var out strings.Builder
	fmt.Fprintf(&out, "\r\nRoom number: %s%d%s        Room zone: %s%d\r\n",
		cyn, state.roomVNum, nrm, cyn, state.zoneNumber)
	fmt.Fprintf(&out, "%sZ%s) Zone name   : %s%s\r\n", grn, nrm, yel, zeditZoneName(state.zone.Name))
	fmt.Fprintf(&out, "%sL%s) Lifespan    : %s%d minutes\r\n", grn, nrm, yel, state.zone.Lifespan)
	fmt.Fprintf(&out, "%sT%s) Top of zone : %s%d\r\n", grn, nrm, yel, state.zone.TopRoom)
	fmt.Fprintf(&out, "%sR%s) Reset Mode  : %s%s%s\r\n", grn, nrm, yel,
		zeditResetMode(state.zone.ResetMode), nrm)
	out.WriteString("[Command list]\r\n")

	repeat := false
	for index, cmd := range state.zone.Commands {
		text := s.zeditCommandText(cmd, repeat)
		fmt.Fprintf(&out, "%s%d - %s%s\r\n", nrm, index, yel, text)
		if cmd.Command == "L" {
			repeat = cmd.Arg2 == 0
		}
	}
	fmt.Fprintf(&out, "%s%d - <END OF LIST>\r\n%sN%s) New command.\r\n%sE%s) Edit a command.\r\n%sD%s) Delete a command.\r\n%sQ%s) Quit\r\nEnter your choice : ",
		nrm, len(state.zone.Commands), grn, nrm, grn, nrm, grn, nrm, grn, nrm)
	s.zeditSend(out.String())
	state.mode = zeditMainMenu
}

func zeditZoneName(name string) string {
	if name == "" {
		return "<NONE!>"
	}
	return name
}

func zeditResetMode(mode int) string {
	switch mode {
	case 0:
		return "Never reset"
	case 1:
		return "Reset when no players are in zone."
	default:
		return "Normal reset."
	}
}

func (s *Session) zeditCommandText(cmd parser.ZoneCommand, repeat bool) string {
	state := s.zedit
	if state == nil {
		return "<Unknown Command>"
	}
	indent := ""
	if repeat {
		indent = "  "
	}
	then := ""
	if cmd.IfFlag != 0 {
		then = " then "
	}
	w := s.manager.world
	switch cmd.Command {
	case "M":
		return fmt.Sprintf("%s%sLoad %s [%s%d%s], Max : %d", indent, then,
			zeditCommandName(w, cmd), s.zeditColsCyan(), cmd.Arg1, s.zeditColsYellow(), cmd.Arg2)
	case "G":
		return fmt.Sprintf("%s%sGive it %s [%s%d%s], Max : %d", indent, then,
			zeditCommandName(w, cmd), s.zeditColsCyan(), cmd.Arg1, s.zeditColsYellow(), cmd.Arg2)
	case "O":
		return fmt.Sprintf("%s%sLoad %s [%s%d%s], Max : %d", indent, then,
			zeditCommandName(w, cmd), s.zeditColsCyan(), cmd.Arg1, s.zeditColsYellow(), cmd.Arg2)
	case "E":
		return fmt.Sprintf("%s%sEquip with %s [%s%d%s], %s, Max : %d", indent, then,
			zeditCommandName(w, cmd), s.zeditColsCyan(), cmd.Arg1, s.zeditColsYellow(),
			zeditEquipmentName(cmd.Arg3), cmd.Arg2)
	case "P":
		return fmt.Sprintf("%s%sPut %s [%s%d%s] in %s [%s%d%s], Max : %d", indent, then,
			zeditCommandName(w, cmd), s.zeditColsCyan(), cmd.Arg1, s.zeditColsYellow(),
			zeditContainerName(w, cmd.Arg3), s.zeditColsCyan(), cmd.Arg3, s.zeditColsYellow(), cmd.Arg2)
	case "R":
		name, vnum := zeditRemoveName(w, cmd)
		return fmt.Sprintf("%s%sRemove %s [%s%d%s] from room.", indent, then, name,
			s.zeditColsCyan(), vnum, s.zeditColsYellow())
	case "D":
		direction := ""
		if cmd.Arg2 >= 0 && cmd.Arg2 < len(olc.ZoneDirections) {
			direction = olc.ZoneDirections[cmd.Arg2]
		}
		doorState := "open"
		if cmd.Arg3 == 1 {
			doorState = "closed"
		} else if cmd.Arg3 != 0 {
			doorState = "locked"
		}
		return fmt.Sprintf("%s%sSet door %s as %s.", indent, then, direction, doorState)
	case "L":
		if cmd.Arg2 == 0 {
			return fmt.Sprintf("%sRepeat for %d times...", then, cmd.Arg3)
		}
		return fmt.Sprintf("%s...End Repeat", then)
	default:
		return "<Unknown Command>"
	}
}

func (s *Session) zeditColsCyan() string {
	_, _, cyan, _ := s.zeditCols()
	return cyan
}

func (s *Session) zeditColsYellow() string {
	_, _, _, yellow := s.zeditCols()
	return yellow
}

func (s *Session) zeditShowCommandTypeLocked() {
	nrm, grn, _, _ := s.zeditCols()
	var out strings.Builder
	fmt.Fprintf(&out, "\r\n%sM%s) Load Mobile to room              %sO%s) Load Object to room\r\n",
		grn, nrm, grn, nrm)
	fmt.Fprintf(&out, "%sE%s) Equip mobile with object         %sG%s) Give an object to a mobile\r\n",
		grn, nrm, grn, nrm)
	fmt.Fprintf(&out, "%sP%s) Put object in another object     %sD%s) Open/Close/Lock a Door\r\n",
		grn, nrm, grn, nrm)
	fmt.Fprintf(&out, "%sR%s) Remove a mobile/object from room %sL%s) Begin/End Looping\r\n",
		grn, nrm, grn, nrm)
	out.WriteString("What sort of command will this be? : ")
	s.zeditSend(out.String())
	s.zedit.mode = zeditCommandType
}

func (s *Session) zeditShowArg1Locked() {
	cmd := &s.zedit.zone.Commands[s.zedit.position]
	switch cmd.Command {
	case "M":
		s.zeditSend("Input mob's vnum : ")
		s.zedit.mode = zeditArg1
	case "O", "E", "P", "G":
		s.zeditSend("Input object vnum : ")
		s.zedit.mode = zeditArg1
	case "D", "R", "L":
		updated := *cmd
		updated.Arg1 = s.zedit.roomVNum
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &s.zedit.zone, Index: s.zedit.position, Command: &updated})
		s.zeditShowArg2Locked()
	default:
		s.finishZeditLocked(false)
	}
}

func (s *Session) zeditShowArg2Locked() {
	cmd := &s.zedit.zone.Commands[s.zedit.position]
	switch cmd.Command {
	case "M", "O", "E", "P", "G":
		s.zeditSend("Input the maximum number that can exist on the mud : ")
	case "D":
		var out strings.Builder
		for i, direction := range olc.ZoneDirections[:len(olc.ZoneDirections)-1] {
			fmt.Fprintf(&out, "%d) Exit %s.\r\n", i, direction)
		}
		out.WriteString("Enter exit number for door : ")
		s.zeditSend(out.String())
	case "R":
		s.zeditSend("Input 0 for a mobile, 1 for an object: ")
	case "L":
		s.zeditSend("Input 0 for loop start, 1 for loop finish: ")
	default:
		s.finishZeditLocked(false)
		return
	}
	s.zedit.mode = zeditArg2
}

func (s *Session) zeditShowArg3Locked() {
	cmd := &s.zedit.zone.Commands[s.zedit.position]
	switch cmd.Command {
	case "E":
		var out strings.Builder
		for i := 0; i < len(olc.ZoneEquipmentNames); i += 2 {
			second := ""
			if i+1 < len(olc.ZoneEquipmentNames) {
				second = olc.ZoneEquipmentNames[i+1]
			}
			fmt.Fprintf(&out, "%2d) %26.26s %2d) %26.26s\r\n", i, olc.ZoneEquipmentNames[i], i+1, second)
		}
		out.WriteString("Input location to equip : ")
		s.zeditSend(out.String())
	case "P":
		s.zeditSend("Input the vnum of the container : ")
	case "D":
		s.zeditSend("0)  Door open\r\n1)  Door closed\r\n2)  Door locked\r\nEnter state of the door : ")
	case "L":
		if cmd.Arg2 == 0 {
			s.zeditSend("Input the number of times to repeat the loop : ")
		} else {
			updated := *cmd
			updated.Arg3 = -1
			applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &s.zedit.zone, Index: s.zedit.position, Command: &updated})
		}
	case "R":
		if cmd.Arg2 != 0 {
			s.zeditSend("Input object's vnum : ")
		} else {
			s.zeditSend("Input mobile's vnum : ")
		}
	default:
		s.finishZeditLocked(false)
		return
	}
	s.zedit.mode = zeditArg3
}

func (s *Session) parseZeditLocked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	switch state.mode {
	case zeditConfirmSaveString:
		switch firstByte(line) {
		case 'y', 'Y':
			s.zeditSend("Saving zone info in memory.\r\n")
			s.finishZeditLocked(true)
		case 'n', 'N':
			s.finishZeditLocked(false)
		default:
			s.zeditSend("Invalid choice!\r\n")
			s.zeditSend("Do you wish to save the zone info? : ")
		}
	case zeditMainMenu:
		s.parseZeditMainLocked(line)
	case zeditNewEntry:
		s.parseZeditNewLocked(line)
	case zeditDeleteEntry:
		s.parseZeditDeleteLocked(line)
	case zeditChangeEntry:
		s.parseZeditChangeLocked(line)
	case zeditCommandType:
		s.parseZeditCommandTypeLocked(line)
	case zeditIfFlag:
		s.parseZeditIfFlagLocked(line)
	case zeditArg1:
		s.parseZeditArg1Locked(line)
	case zeditArg2:
		s.parseZeditArg2Locked(line)
	case zeditArg3:
		s.parseZeditArg3Locked(line)
	case zeditZoneNameMode:
		applyOLC(olc.Operation{Kind: olc.OpSetZoneName, Zone: &state.zone, Text: line})
		state.dirtyHeader = true
		s.zeditShowMenuLocked()
	case zeditZoneReset:
		number := atoiC(line)
		if !isASCIIDigit(firstByte(line)) || number < 0 || number > 2 {
			s.zeditSend("Try again (0-2) : ")
			return
		}
		applyOLC(olc.Operation{Kind: olc.OpSetZoneResetMode, Zone: &state.zone, Value: number})
		state.dirtyHeader = true
		s.zeditShowMenuLocked()
	case zeditZoneLife:
		number := atoiC(line)
		if !isASCIIDigit(firstByte(line)) || number < 0 || number > 240 {
			s.zeditSend("Try again (0-240) : ")
			return
		}
		applyOLC(olc.Operation{Kind: olc.OpSetZoneLifespan, Zone: &state.zone, Value: number})
		state.dirtyHeader = true
		s.zeditShowMenuLocked()
	case zeditZoneTop:
		oldTop := state.zone.TopRoom
		zones := s.manager.world.GetAllZones()
		if state.zoneIndex == len(zones)-1 {
			applyOLC(olc.Operation{
				Kind:  olc.OpSetZoneTopRoom,
				Zone:  &state.zone,
				Value: atoiC(line),
				Low:   state.zone.Number * 100,
				High:  32000,
			})
		} else if state.zoneIndex >= 0 && state.zoneIndex+1 < len(zones) {
			applyOLC(olc.Operation{
				Kind:  olc.OpSetZoneTopRoom,
				Zone:  &state.zone,
				Value: atoiC(line),
				Low:   state.zone.Number * 100,
				High:  zones[state.zoneIndex+1].Number*100 - 1,
			})
		}
		if oldTop != state.zone.TopRoom {
			state.dirtyHeader = true
		}
		s.zeditShowMenuLocked()
	}
}

func (s *Session) parseZeditMainLocked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	switch firstByte(line) {
	case 'q', 'Q':
		if state.dirtyCommands || state.dirtyHeader {
			state.mode = zeditConfirmSaveString
			s.zeditSend("Do you wish to save the changes to the zone info? (y/n) : ")
		} else {
			s.zeditSend("No changes made.\r\n")
			s.finishZeditLocked(false)
		}
	case 'n', 'N':
		state.mode = zeditNewEntry
		s.zeditSend("What number in the list should the new command be? : ")
	case 'e', 'E':
		state.mode = zeditChangeEntry
		s.zeditSend("Which command do you wish to change? : ")
	case 'd', 'D':
		state.mode = zeditDeleteEntry
		s.zeditSend("Which command do you wish to delete? : ")
	case 'z', 'Z':
		state.mode = zeditZoneNameMode
		s.zeditSend("Enter new zone name : ")
	case 't', 'T':
		if getEffectiveLevel(s) < game.LVL_IMPL {
			s.zeditShowMenuLocked()
		} else {
			state.mode = zeditZoneTop
			s.zeditSend("Enter new top of zone : ")
		}
	case 'l', 'L':
		state.mode = zeditZoneLife
		s.zeditSend("Enter new zone lifespan : ")
	case 'r', 'R':
		s.zeditSend("\r\n0) Never reset\r\n1) Reset only when no players in zone\r\n2) Normal reset\r\nEnter new zone reset type : ")
		state.mode = zeditZoneReset
	default:
		s.zeditShowMenuLocked()
	}
}

func (s *Session) parseZeditNewLocked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	pos := atoiC(line)
	if !isASCIIDigit(firstByte(line)) || pos < 0 || pos > len(state.zone.Commands) {
		s.zeditShowMenuLocked()
		return
	}
	applyOLC(olc.Operation{
		Kind:    olc.OpAddZoneCommand,
		Zone:    &state.zone,
		Index:   pos,
		Command: &parser.ZoneCommand{Command: "N"},
	})
	state.position = pos
	state.dirtyCommands = true
	s.zeditShowCommandTypeLocked()
}

func (s *Session) parseZeditDeleteLocked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	pos := atoiC(line)
	if isASCIIDigit(firstByte(line)) {
		state.dirtyCommands = true
		if pos >= 0 && pos < len(state.zone.Commands) {
			applyOLC(olc.Operation{Kind: olc.OpRemoveZoneCommand, Zone: &state.zone, Index: pos})
		}
	}
	s.zeditShowMenuLocked()
}

func (s *Session) parseZeditChangeLocked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	pos := atoiC(line)
	if !isASCIIDigit(firstByte(line)) || pos < 0 || pos >= len(state.zone.Commands) {
		s.zeditShowMenuLocked()
		return
	}
	state.position = pos
	state.dirtyCommands = true
	s.zeditShowCommandTypeLocked()
}

func (s *Session) parseZeditCommandTypeLocked(line string) {
	state := s.zedit
	if state == nil || state.position < 0 || state.position >= len(state.zone.Commands) {
		return
	}
	command := firstByte(line)
	if command >= 'a' && command <= 'z' {
		command -= 'a' - 'A'
	}
	if !strings.ContainsRune("MOPEDGRL", rune(command)) {
		s.zeditSend("Invalid choice, try again : ")
		return
	}
	updated := state.zone.Commands[state.position]
	updated.Command = string(command)
	applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
	if state.position != 0 {
		s.zeditSend("Is this command dependent on the success of the previous one? (y/n)\r\n")
		state.mode = zeditIfFlag
		return
	}
	updated.IfFlag = 0
	applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
	s.zeditShowArg1Locked()
}

func (s *Session) parseZeditIfFlagLocked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	switch firstByte(line) {
	case 'y', 'Y':
		updated := state.zone.Commands[state.position]
		updated.IfFlag = 1
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
	case 'n', 'N':
		updated := state.zone.Commands[state.position]
		updated.IfFlag = 0
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
	default:
		s.zeditSend("Try again : ")
		return
	}
	s.zeditShowArg1Locked()
}

func (s *Session) parseZeditArg1Locked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	if !isASCIIDigit(firstByte(line)) {
		s.zeditSend("Must be a numeric value, try again : ")
		return
	}
	number := atoiC(line)
	cmd := &state.zone.Commands[state.position]
	switch cmd.Command {
	case "M":
		if _, ok := s.manager.world.GetMobPrototype(number); !ok {
			s.zeditSend("That mobile does not exist, try again : ")
			return
		}
	case "O", "P", "E", "G":
		if _, ok := s.manager.world.GetObjPrototype(number); !ok {
			s.zeditSend("That object does not exist, try again : ")
			return
		}
	default:
		s.finishZeditLocked(false)
		return
	}
	updated := *cmd
	updated.Arg1 = number
	applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
	s.zeditShowArg2Locked()
}

func (s *Session) parseZeditArg2Locked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	if !isASCIIDigit(firstByte(line)) {
		s.zeditSend("Must be a numeric value, try again : ")
		return
	}
	number := atoiC(line)
	cmd := &state.zone.Commands[state.position]
	switch cmd.Command {
	case "M", "O":
		updated := *cmd
		updated.Arg2 = number
		updated.Arg3 = state.roomVNum
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowMenuLocked()
	case "G":
		updated := *cmd
		updated.Arg2 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowMenuLocked()
	case "P", "E":
		updated := *cmd
		updated.Arg2 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowArg3Locked()
	case "D":
		// C counts six real directions but accepts pos == i, i.e. 6.
		if number < 0 || number > len(olc.ZoneDirections)-1 {
			s.zeditSend("Try again : ")
			return
		}
		updated := *cmd
		updated.Arg2 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowArg3Locked()
	case "R":
		if number < 0 || number > 1 {
			s.zeditSend("Try again : ")
			return
		}
		updated := *cmd
		updated.Arg2 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowArg3Locked()
	case "L":
		if number < 0 || number > 1 {
			s.zeditSend("Try again : ")
			return
		}
		updated := *cmd
		updated.Arg2 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		if number == 0 {
			s.zeditShowArg3Locked()
		} else {
			// zedit_parse handles loop finish directly: C stores -1 and
			// redraws the main menu without entering ZEDIT_ARG3.
			updated := *cmd
			updated.Arg3 = -1
			applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
			s.zeditShowMenuLocked()
		}
	}
}

func (s *Session) parseZeditArg3Locked(line string) {
	state := s.zedit
	if state == nil {
		return
	}
	if !isASCIIDigit(firstByte(line)) {
		s.zeditSend("Must be a numeric value, try again : ")
		return
	}
	number := atoiC(line)
	cmd := &state.zone.Commands[state.position]
	switch cmd.Command {
	case "E":
		if number < 0 || number > len(olc.ZoneEquipmentNames) {
			s.zeditSend("Try again : ")
			return
		}
		updated := *cmd
		updated.Arg3 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowMenuLocked()
	case "P":
		if _, ok := s.manager.world.GetObjPrototype(number); !ok {
			s.zeditSend("That object does not exist, try again : ")
			return
		}
		updated := *cmd
		updated.Arg3 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowMenuLocked()
	case "D":
		if number < 0 || number > 2 {
			s.zeditSend("Try again : ")
			return
		}
		updated := *cmd
		updated.Arg3 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowMenuLocked()
	case "L":
		if cmd.Arg2 == 0 {
			if number <= 0 {
				s.zeditSend("The loop must repeat at least once, try again : ")
				return
			}
			updated := *cmd
			updated.Arg3 = number
			applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		} else {
			updated := *cmd
			updated.Arg3 = -1
			applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		}
		s.zeditShowMenuLocked()
	case "R":
		if cmd.Arg2 != 0 {
			if _, ok := s.manager.world.GetObjPrototype(number); !ok {
				s.zeditSend("That object does not exist, try again : ")
				return
			}
		} else if _, ok := s.manager.world.GetMobPrototype(number); !ok {
			s.zeditSend("That mobile does not exist, try again : ")
			return
		}
		updated := *cmd
		updated.Arg3 = number
		applyOLC(olc.Operation{Kind: olc.OpModifyZoneCommand, Zone: &state.zone, Index: state.position, Command: &updated})
		s.zeditShowMenuLocked()
	}
}

func zeditRemoveSaveZone(zone int) {
	clearOLCDirty(olcKindZone, zone)
}

func saveZeditZone(world *game.World, zone *parser.Zone) error {
	saveMu := zoneSaveLock(zone.Number)
	saveMu.Lock()
	defer saveMu.Unlock()
	return saveZeditZoneLocked(world, zone)
}

func saveZeditZoneLocked(world *game.World, zone *parser.Zone) error {
	snapshot, ok := world.SnapshotZone(zone.Number)
	if !ok {
		return fmt.Errorf("zone %d not found", zone.Number)
	}
	var out strings.Builder
	fmt.Fprintf(&out, "#%d\n%s~\n%d %d %d\n", snapshot.Number,
		zeditZoneName(snapshot.Name), snapshot.TopRoom, snapshot.Lifespan, snapshot.ResetMode)
	for _, cmd := range snapshot.Commands {
		arg1, arg2, arg3, ok := zeditDiskArgs(cmd)
		if !ok {
			continue
		}
		fmt.Fprintf(&out, "%s %d %d %d %d\n", cmd.Command, cmd.IfFlag, arg1, arg2, arg3)
	}
	out.WriteString("S\n$\n")

	path := filepath.Join(world.WorldPath, "zon", fmt.Sprintf("%d.zon", snapshot.Number))
	if err := atomicWriteFile(path, []byte(out.String()), 0o666); err != nil {
		return err
	}
	zeditRemoveSaveZone(snapshot.Number)
	return nil
}

// parser.ZoneCommand stores VNUMs, so the C writer's rnum-to-vnum translation
// collapses to identity for every supported command. The explicit switch is
// retained as the audited translation table: G has no arg3 and R chooses an
// object or mobile from arg2; E and L retain their raw numeric fields.
func zeditDiskArgs(cmd parser.ZoneCommand) (int, int, int, bool) {
	switch cmd.Command {
	case "M":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "O":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "G":
		return cmd.Arg1, cmd.Arg2, -1, true
	case "E":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "P":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "D":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "L":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "R":
		return cmd.Arg1, cmd.Arg2, cmd.Arg3, true
	case "*":
		return 0, 0, 0, false
	default:
		slog.Error("zedit writer skipped unknown command", "command", cmd.Command)
		return 0, 0, 0, false
	}
}

func zeditNewZone(s *Session, number int) error {
	if number > olc.NewZoneMax {
		s.zeditSend("326 is the highest zone allowed.\r\n")
		return nil
	}
	world := s.manager.world
	start := number * 100
	if _, ok := olcZoneForVNum(world, start); ok {
		s.zeditSend("A zone already covers that area.\r\n")
		return nil
	}

	if err := olc.WriteNewZoneFiles(world.WorldPath, number); err != nil {
		slog.Error("zedit new-zone file creation failed", "zone", number, "error", err)
		return nil
	}
	if _, ok := world.CreateZone(number); !ok {
		// A concurrent creator may have won after the preflight check. The C
		// implementation is single-threaded; report the same user-facing
		// refusal rather than silently presenting a duplicate in memory.
		s.zeditSend("A zone already covers that area.\r\n")
		return nil
	}
	for _, ext := range []string{"zon", "wld", "mob", "obj", "shp"} {
		if err := zeditCreateIndex(world.WorldPath, number, ext); err != nil {
			slog.Error("zedit index update failed", "zone", number, "type", ext, "error", err)
		}
	}
	s.zeditSend("Zone created.\r\n")
	slog.Info("OLC: new zone created", "player", s.playerName, "zone", number)
	return nil
}

func zeditCreateIndex(worldPath string, number int, ext string) error {
	directory := filepath.Join(worldPath, ext)
	oldPath := filepath.Join(directory, "index")
	newPath := filepath.Join(directory, "newindex")
	data, err := os.ReadFile(filepath.Clean(oldPath))
	if err != nil {
		// C's sprintf(buf, "...%s", buf) uses the same buffer as source and
		// destination on this failure path. Go cannot make that undefined
		// overlap meaningful; retain the resulting diagnostic shape and the
		// original path bytes deterministically.
		return fmt.Errorf("SYSERR: OLC: Failed to open %s", oldPath)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var out strings.Builder
	found := false
	for _, line := range lines {
		if line == "" {
			continue
		}
		if line == "$" {
			if !found {
				fmt.Fprintf(&out, "%d.%s\n", number, ext)
			}
			out.WriteString("$\n")
			break
		}
		if !found {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				if existing, parseErr := strconv.Atoi(fields[0]); parseErr == nil && existing >= number {
					found = true
					if existing > number {
						fmt.Fprintf(&out, "%d.%s\n", number, ext)
					}
				}
			}
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	if err := atomicWriteFile(newPath, []byte(out.String()), 0o666); err != nil {
		return err
	}
	if err := os.Remove(filepath.Clean(oldPath)); err != nil {
		_ = os.Remove(filepath.Clean(newPath))
		return err
	}
	return os.Rename(filepath.Clean(newPath), filepath.Clean(oldPath))
}
