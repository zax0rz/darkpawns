package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

var loadObjectLimits = map[int]int{
	81:   LVL_GRGOD,
	82:   LVL_GRGOD,
	8095: LVL_GRGOD,
}

var loadMessages = [...]string{
	"Suddenly the walls run red with blood and a neon '666' sign flashes.",
	"The flames of Hell roar up then fade, leaving something behind...",
	"A sound not unlike the shriek of a dying dragon fills the room...",
}

func cmdLoad(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	kind, rest := game.OneArgument(strings.Join(args, " "))
	value, _ := game.OneArgument(rest)
	if kind == "" {
		s.Send("Usage: load { obj | mob } <number or name>\r\n")
		return nil
	}
	if value == "" {
		if loadIsAbbrev(kind, "mob") {
			s.Send("Mob name not found.\r\n")
		} else if loadIsAbbrev(kind, "object") {
			s.Send("Object name not found.\r\n")
		}
		s.Send("Usage: load { obj | mob } <number or name>\r\n")
		return nil
	}

	var (
		vnum   int
		isName bool
	)
	if !loadStartsWithDigit(value) {
		if loadIsAbbrev(kind, "mob") {
			vnum, isName = findMobLoadVNum(s, value)
			if !isName {
				s.Send("Mob name not found.\r\n")
			}
		} else if loadIsAbbrev(kind, "object") {
			vnum, isName = findObjectLoadVNum(s, value)
			if !isName {
				s.Send("Object name not found.\r\n")
			}
		}
	}
	if !loadStartsWithDigit(value) && !isName {
		s.Send("Usage: load { obj | mob } <number or name>\r\n")
		return nil
	}
	if !isName {
		vnum = loadAtoi(value)
		if vnum < 0 {
			s.Send("A NEGATIVE number??\r\n")
			return nil
		}
	}
	roomVNum := s.player.GetRoom()

	if loadIsAbbrev(kind, "mob") {
		mob, err := s.manager.world.SpawnMobQuiet(vnum, roomVNum)
		if err != nil {
			s.Send("There is no monster with that number.\r\n")
			return nil
		}
		slog.Info("(GC) load mob", "who", s.player.Name, "mob", mob.GetShortDesc(), "room", roomVNum)
		sendLoadNarration(s, roomVNum, fmt.Sprintf("%s has created %s!", s.player.Name, mob.GetShortDesc()), fmt.Sprintf("You create %s.", mob.GetShortDesc()))
	} else if loadIsAbbrev(kind, "obj") {
		obj, err := s.manager.world.SpawnObject(vnum, -1)
		if err != nil {
			s.Send("There is no object with that number.\r\n")
			return nil
		}
		if minLevel, limited := loadObjectLimits[vnum]; limited && s.player.GetLevel() < minLevel {
			s.manager.world.ExtractObject(obj, roomVNum)
			s.Send("You're not godly enough to load that!\r\n")
			return nil
		}
		if maxCost, limited := loadCostLimit(s.player.GetLevel()); limited && obj.GetCost() > maxCost {
			s.manager.world.ExtractObject(obj, roomVNum)
			s.Send("That is beyond your godly powers...\r\n")
			return nil
		}
		if err := s.manager.world.PlaceWizardLoadedObjectInInventory(obj, s.player); err != nil {
			slog.Error("wizard load object transfer failed", "who", s.player.Name, "obj_vnum", vnum, "error", err)
			s.manager.world.ExtractObject(obj, roomVNum)
			s.Send("You can't carry that right now.\r\n")
			return nil
		}
		slog.Info("(GC) load obj", "who", s.player.Name, "obj", obj.GetShortDesc(), "room", roomVNum)
		sendLoadNarration(s, roomVNum, fmt.Sprintf("%s has created %s!", s.player.Name, obj.GetShortDesc()), fmt.Sprintf("You create %s.", obj.GetShortDesc()))
	} else {
		s.Send("That'll have to be either 'obj' or 'mob'.\r\n")
	}
	return nil
}

func loadIsAbbrev(arg, word string) bool {
	return arg != "" && strings.HasPrefix(strings.ToLower(word), strings.ToLower(arg))
}

func loadStartsWithDigit(value string) bool {
	return value != "" && value[0] >= '0' && value[0] <= '9'
}

func loadAtoi(value string) int {
	end := 0
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	parsed, err := strconv.Atoi(value[:end])
	if err != nil {
		return 0
	}
	return parsed
}

func findMobLoadVNum(s *Session, name string) (int, bool) {
	parsed := s.manager.world.GetParsedWorld()
	if parsed == nil {
		return 0, false
	}
	for i := range parsed.Mobs {
		if loadNameMatches(name, parsed.Mobs[i].Keywords) {
			return parsed.Mobs[i].VNum, true
		}
	}
	return 0, false
}

func findObjectLoadVNum(s *Session, name string) (int, bool) {
	parsed := s.manager.world.GetParsedWorld()
	if parsed == nil {
		return 0, false
	}
	for i := range parsed.Objs {
		if loadNameMatches(name, parsed.Objs[i].Keywords) {
			return parsed.Objs[i].VNum, true
		}
	}
	return 0, false
}

func loadNameMatches(name, keywords string) bool {
	name = strings.ToLower(name)
	for _, keyword := range strings.Fields(strings.ToLower(keywords)) {
		if strings.HasPrefix(keyword, name) {
			return true
		}
	}
	return false
}

func loadCostLimit(level int) (int, bool) {
	switch level {
	case 31:
		return 4000, true
	case 32:
		return 5000, true
	case 33:
		return 5500, true
	case 34:
		return 6000, true
	case 35:
		return 7000, true
	case 36:
		return 7500, true
	case 37:
		return 8000, true
	default:
		return 0, false
	}
}

func sendLoadNarration(s *Session, roomVNum int, roomCreation, actorCreation string) {
	gesture := fmt.Sprintf("%s makes a strange magickal gesture.\r\n", s.player.Name)
	s.manager.BroadcastToRoom(roomVNum, []byte(gesture), s.playerName)
	message := loadMessages[dprng.Number(0, len(loadMessages)-1)]
	s.manager.BroadcastToRoom(roomVNum, []byte(message+"\r\n"), s.playerName)
	s.Send(message + "\r\n")
	s.manager.BroadcastToRoom(roomVNum, []byte(roomCreation+"\r\n"), s.playerName)
	s.Send(actorCreation + "\r\n")
}

// ---------------------------------------------------------------------------
// purge — remove all mobs/objects from room (LVL_IMMORT)
// ---------------------------------------------------------------------------
func cmdPurge(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	roomVNum := s.player.GetRoom()
	if len(args) >= 1 && args[0] != "" {
		// C's one_argument() supplies only the first token to both character
		// and object lookup. ResolveCharInRoom mirrors get_char_room_vis(),
		// including keyword abbreviation and visibility semantics.
		targetName, _ := game.OneArgument(strings.Join(args, " "))
		if target, found := s.manager.world.ResolveCharInRoom(s.player, targetName); found {
			if target.Player != nil {
				victim := target.Player
				if s.player.GetLevel() <= victim.GetLevel() {
					s.Send("Fuuuuuuuuu!\r\n")
					return nil
				}

				// close_socket() announces the lost link before do_purge()
				// extracts the lower-level player. Suppress the generic Go
				// "has left" cleanup broadcast so only C's line is visible.
				if victimSession, ok := s.manager.GetSession(victim.GetName()); ok {
					game.Act(s.manager.world, true, victim, nil, nil, nil, "$n has lost $s link.", "", game.ToRoom)
					victimSession.leaveBroadcastHandled = true
					s.manager.UnregisterAndClose(victim.GetName())
				} else {
					game.ExtractChar(victim)
				}
			} else if target.Mob != nil {
				game.Act(s.manager.world, false, s.player, target.Mob, nil, nil, "$n disintegrates $N.", "", game.ToNotVict)
				s.manager.world.ExtractMob(target.Mob)
			}
			s.Send("Okay.\r\n")
			return nil
		}

		if item, found := s.manager.world.ResolveObjectInRoom(s.player, targetName); found {
			game.Act(s.manager.world, false, s.player, nil, item, nil, "$n destroys $p.", "", game.ToRoom)
			s.manager.world.ExtractObject(item, roomVNum)
			s.Send("Okay.\r\n")
			return nil
		}
		s.Send("Nothing here by that name.\r\n")
		return nil
	}
	// No argument — purge entire room
	game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n gestures... You are surrounded by scorching flames!", "", game.ToRoom)
	sendPurgeRoomLine(s, roomVNum, "The world seems a little cleaner.\r\n")
	for _, mob := range s.manager.world.GetMobsInRoom(roomVNum) {
		s.manager.world.ExtractMob(mob)
	}
	for _, item := range s.manager.world.GetItemsInRoom(roomVNum) {
		s.manager.world.ExtractObject(item, roomVNum)
	}
	return nil
}

// sendPurgeRoomLine mirrors send_to_room(): deliver to every connected player
// in the room who is awake, including the issuing immortal.
func sendPurgeRoomLine(s *Session, roomVNum int, message string) {
	for _, player := range s.manager.world.GetPlayersInRoom(roomVNum) {
		if player.GetPosition() <= combat.PosSleeping {
			continue
		}
		if player == s.player {
			s.Send(message)
			continue
		}
		if peer, ok := s.manager.GetSession(player.GetName()); ok {
			peer.Send(message)
		}
	}
}

// ---------------------------------------------------------------------------
// teleport — teleport a player (LVL_GOD)
// ---------------------------------------------------------------------------
