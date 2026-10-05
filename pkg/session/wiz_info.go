package session

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// showFields is C's show subcommand table (src/act.wizard.c:2251-2267): the
// `show options` listing and the subcommand lookup read the same rows in this
// order, and levels gate both. Row 0 is C's "nothing": the listing loop skips
// it (it starts at index 1), but the lookup does not, so a prefix such as
// `show n` resolves to it and falls through to C's default reply. C's "\n"
// sentinel is the no-match case, which reaches the same reply.
var showFields = []struct {
	name  string
	level int
}{
	{name: "nothing", level: 0},
	{name: "zones", level: LVL_IMMORT},
	{name: "player", level: LVL_GOD},
	{name: "rent", level: LVL_GOD},
	{name: "stats", level: LVL_IMMORT},
	{name: "errors", level: LVL_IMPL - 1},
	{name: "death", level: LVL_GOD},
	{name: "godrooms", level: LVL_GOD},
	{name: "shops", level: LVL_IMMORT},
	{name: "houses", level: LVL_GOD},
	{name: "tattoos", level: LVL_IMMORT},
	{name: "aggr", level: LVL_IMPL - 1},
	{name: "reagents", level: LVL_IMMORT},
	{name: "hooks", level: LVL_IMMORT},
	{name: "neutral", level: LVL_IMMORT},
}

func cmdShow(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		// C (act.wizard.c:2271-2277) prints the rows this character may use,
		// five per line, then one trailing CRLF; a shown count that is a
		// multiple of five therefore ends the output with a blank line.
		var b strings.Builder
		b.WriteString("Show options:\r\n")
		shown := 0
		for _, f := range showFields[1:] {
			if f.level > getEffectiveLevel(s) {
				continue
			}
			shown++
			fmt.Fprintf(&b, "%-15s", f.name)
			if shown%5 == 0 {
				b.WriteString("\r\n")
			}
		}
		b.WriteString("\r\n")
		s.Send(b.String())
		return nil
	}

	// This is C's case-sensitive prefix lookup (act.wizard.c:2285-2287:
	// strncmp over strlen(field)), so preserve the raw first argument. It
	// scans from row 0, so "nothing" wins every prefix it shares with a later
	// row ("n", "no" beat "neutral").
	field := args[0]
	value := ""
	if len(args) > 1 {
		value = args[1]
	}
	fields := showFields
	fieldIndex := -1
	for i, candidate := range fields {
		if strings.HasPrefix(candidate.name, field) {
			fieldIndex = i
			break
		}
	}
	if fieldIndex < 0 {
		s.Send("Sorry, I don't understand that.")
		return nil
	}
	if !checkLevel(s, fields[fieldIndex].level) {
		s.Send("You are not godly enough for that!\r\n")
		return nil
	}
	if fieldIndex == 0 {
		// Row 0 has no case in C's switch; its default replies.
		s.Send("Sorry, I don't understand that.")
		return nil
	}

	switch fields[fieldIndex].name {
	case "zones":
		// src/act.wizard.c:2225-2231, 2300-2319: live ages and C pager.
		zones := s.manager.world.GetAllZones()
		ages := s.manager.world.ZoneClockSnapshot().Ages
		var report strings.Builder
		selected := -1
		if value == "." {
			selected = s.manager.world.GetRoomZone(s.player.GetRoomVNum())
		} else if value != "" && cIsNumber(value) {
			selected = cAtoi(value)
		}
		matched := false
		for _, zone := range zones {
			if selected >= 0 && zone.Number != selected {
				continue
			}
			matched = true
			fmt.Fprintf(&report, "%3d %-30.30s Age: %3d; Reset: %3d (%1d); Top: %5d\r\n", zone.Number, zone.Name, ages[zone.Number], zone.Lifespan, zone.ResetMode, zone.TopRoom)
		}
		if selected >= 0 && !matched {
			s.Send("That is not a valid zone.\r\n")
			return nil
		}
		PageString(s, report.String())
	case "player":
		if value == "" {
			s.Send("A name would help.\r\n")
			return nil
		}
		if _, online := s.manager.world.GetPlayer(value); !online && !s.storedPlayerExists(value) {
			s.Send("There is no such player.\r\n")
		}
	case "rent":
		if value == "" {
			s.Send("A name would help.\r\n")
			return nil
		}
		if !s.storedPlayerExists(value) {
			s.Send(fmt.Sprintf("%s has no rent file.\r\n", strings.ToLower(value)))
		}
	case "stats":
		parsed := s.manager.world.GetParsedWorld()
		mobPrototypes, objectPrototypes := 0, 0
		if parsed != nil {
			mobPrototypes = len(parsed.Mobs)
			objectPrototypes = len(parsed.Objs)
		}
		// C walks character_list (act.wizard.c:2356-2378): non-NPC characters
		// the caller can see, and of those the ones holding a descriptor — a
		// linkdead player counts in the first number only. `registered` is the
		// player-file index size (top_of_p_table + 1), which the port reads from
		// the player store.
		playersInGame, connected := 0, 0
		for _, p := range s.manager.world.GetAllPlayers() {
			if !game.CanSee(s.player, p) {
				continue
			}
			playersInGame++
			if live := findSessionByName(s.manager, p.GetName()); live != nil && live.hasTransport() {
				connected++
			}
		}
		registered := 0
		if s.manager.hasDB && s.manager.db != nil {
			if n, err := s.manager.db.CountPlayers(); err == nil {
				registered = n
			}
		}
		s.Send(fmt.Sprintf("Current stats:\r\n"+
			"  %5d players in game  %5d connected\r\n"+
			"  %5d registered\r\n"+
			"  %5d mobiles          %5d prototypes\r\n"+
			"  %5d objects          %5d prototypes\r\n"+
			"  %5d rooms            %5d zones\r\n"+
			"  %5d large bufs\r\n"+
			"  %5d buf switches     %5d overflows\r\n",
			playersInGame, connected, registered,
			len(s.manager.world.GetAllMobs()), mobPrototypes,
			len(s.manager.world.GetAllObjects()), objectPrototypes,
			len(s.manager.world.Rooms()), len(s.manager.world.GetAllZones()),
			// large bufs, buf switches and overflows are C's output-buffer
			// counters; the port has no counterpart, so they stay zero.
			0, 0, 0))
	case "errors":
		s.Send(showErrantRooms(s.manager.world))
	case "death":
		s.Send(showFlaggedRooms(s.manager.world, "ROOM_DEATH", "Death Traps\r\n-----------\r\n"))
	case "godrooms":
		s.Send(showFlaggedRooms(s.manager.world, "ROOM_GODROOM", "Godrooms\r\n--------------------------\r\n"))
	case "shops":
		// C's show_shops() consumes the complete parsed .shp database. The Go
		// world currently indexes only shopkeepers, so this branch remains
		// explicitly unproven rather than inventing a partial listing.
		return nil
	case "houses":
		if len(s.manager.world.HouseControl) == 0 {
			s.Send("No houses have been defined.\r\n")
		}
	case "tattoos":
		s.Send(showTattooListing())
	case "aggr":
		mobs := s.manager.world.GetAllMobs()
		sort.SliceStable(mobs, func(i, j int) bool {
			if mobs[i].GetVNum() != mobs[j].GetVNum() {
				return mobs[i].GetVNum() < mobs[j].GetVNum()
			}
			return mobs[i].GetName() < mobs[j].GetName()
		})
		for _, mob := range mobs {
			if mob.HasFlag("AGGR24") {
				s.Send(fmt.Sprintf("%d %s\r\n", mob.GetVNum(), mob.GetName()))
			}
		}
	case "reagents":
		s.Send(showReagentListing())
	case "hooks":
		if value == "" {
			s.Send("You must supply a zone number!\r\n")
			return nil
		}
		if !cIsNumber(value) {
			return nil
		}
		number := cAtoi(value)
		for _, zone := range s.manager.world.GetAllZones() {
			if zone.Number == number {
				s.Send(showZoneHooks(s.manager.world, number))
				return nil
			}
		}
		s.Send("That is not a valid zone.\r\n")
	case "neutral":
		s.Send(showFlaggedRooms(s.manager.world, "ROOM_NEUTRAL", "Neutral Rooms\r\n-------------\r\n"))
	}
	return nil
}

func showErrantRooms(w *game.World) string {
	count := 0
	var result strings.Builder
	result.WriteString("Errant Rooms\r\n------------\r\n")
	rooms := w.Rooms()
	for i := range rooms {
		room := &rooms[i]
		for _, exit := range room.Exits {
			if exit.ToRoom == 0 {
				count++
				fmt.Fprintf(&result, "%2d: [%5d] %s\r\n", count, room.VNum, room.Name)
			}
		}
	}
	return result.String()
}

func showFlaggedRooms(w *game.World, flag, header string) string {
	count := 0
	var result strings.Builder
	result.WriteString(header)
	rooms := w.Rooms()
	for i := range rooms {
		room := &rooms[i]
		if game.HasRoomFlag(room, flag) {
			count++
			fmt.Fprintf(&result, "%2d: [%5d] %s\r\n", count, room.VNum, room.Name)
		}
	}
	return result.String()
}

func showTattooListing() string {
	return strings.Join([]string{
		"[ 0]                                None  : no tattoo\r\n",
		"[ 1]                   of a green dragon  : damroll+2 str+2\r\n",
		"[ 2]                  in a tribal design  : dex+1\r\n",
		"[ 3]                  of a flaming skull  : summon skull\r\n",
		"[ 4]                  of a leaping tiger  : dex+1 mv+10\r\n",
		"[ 5]                      of an ice worm  : dam+2\r\n",
		"[ 6]                      of an open eye  : greater percept\r\n",
		"[ 7]                   of crossed swords  : hit and dam+1\r\n",
		"[ 8]                of a screaming eagle  : moves+20\r\n",
		"[ 9]                          of a heart  : hp+20\r\n",
		"[10]                           of a star  : mana+20\r\n",
		"[11]                           of a ship  : change density\r\n",
		"[12]                         of a spider  : dex+3\r\n",
		"[13]          of the symbol of the Jyhad  : dam+1\r\n",
		"[14]                   of the word 'MOM'  : wis+3\r\n",
		"[15]                         of an angel  : bless\r\n",
		"[16]                            of a fox  : int+1\r\n",
		"[17]                           of an owl  : wis+1\r\n",
	}, "")
}

func showReagentListing() string {
	return strings.Join([]string{
		"blindness:  a small, clouded lens\r\n",
		"charm person:  a small, glittering crystal\r\n",
		"color spray:  a prism\r\n",
		"curse:  a raven feather\r\n",
		"energy drain:  some vampire dust\r\n",
		"fireball:  a bit of ash\r\n",
		"flame arrow:  a shard of obsidian\r\n",
		"sleep:  a pinch of sand\r\n",
		"waterwalk:  the leg of a frog\r\n",
		"metalskin:  a small chunk of iron\r\n",
		"disintegration:  the eye of a beholder\r\n",
	}, "")
}

// cmdDark — stop all combat in the room (LVL_IMMORT)
func cmdDark(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	roomVNum := s.player.GetRoom()
	// C walks the room list and stops every fighting character. A fighter is
	// told about the peace before the command's actor/room messages; fighting
	// mobs also forget their combat memory.
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	fightingMobs := make([]*game.MobInstance, 0, len(mobs))
	for _, mob := range mobs {
		if mob.IsFighting() {
			fightingMobs = append(fightingMobs, mob)
		}
	}
	for _, p := range players {
		if !p.IsFighting() {
			continue
		}
		s.manager.combatEngine.StopCombat(p.GetName())
		// C uses the mixed \n\r terminator here. Send a lone LF so the
		// telnet transport's canonicalizer produces one line terminator rather
		// than appending a second CRLF to C's already-terminated text.
		p.SendMessage("The peace of the ancients fills your soul.\n")
	}
	for _, mob := range fightingMobs {
		s.manager.combatEngine.StopCombat(mob.GetName())
		if len(mob.GetMemory()) > 0 {
			mob.ClearMemory()
		}
	}

	s.Send("You stop the senseless violence in the room with a wave of your hand.\r\n")
	game.Act(s.manager.world, true, s.player, nil, nil, nil,
		"$n stops the senseless violence in the room with a wave of $s hand.", "", game.ToRoom)
	return nil
}

// cmdSyslog — toggle system logging level (LVL_IMMORT)
func cmdSyslog(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send(fmt.Sprintf("Your syslog is currently %s.\r\n", syslogLevel(s.player.GetFlags())))
		return nil
	}
	level, ok := parseSyslogLevel(args[0])
	if !ok {
		s.Send("Usage: syslog { Off | Brief | Normal | Complete }\r\n")
		return nil
	}
	setSyslogFlags(s.player, level)
	s.Send(fmt.Sprintf("Your syslog is now %s.\r\n", syslogNames[level]))
	return nil
}

var syslogNames = [...]string{"off", "brief", "normal", "complete"}

// parseSyslogLevel mirrors C search_block(..., FALSE): the first argument is
// lowercased and may be any non-empty prefix of one of the four level names.
func parseSyslogLevel(arg string) (int, bool) {
	arg = strings.ToLower(arg)
	if arg == "" {
		return 0, false
	}
	for level, name := range syslogNames {
		if strings.HasPrefix(name, arg) {
			return level, true
		}
	}
	return 0, false
}

func setSyslogFlags(p *game.Player, level int) {
	p.SetPlrFlag(game.PrfLog1, false)
	p.SetPlrFlag(game.PrfLog2, false)
	if level&1 != 0 {
		p.SetPlrFlag(game.PrfLog1, true)
	}
	if level&2 != 0 {
		p.SetPlrFlag(game.PrfLog2, true)
	}
}

func syslogLevel(flags uint64) string {
	level := (flags & (1 << uint(game.PrfLog1))) >> uint(game.PrfLog1)
	level |= ((flags & (1 << uint(game.PrfLog2))) >> uint(game.PrfLog2)) << 1
	if level < uint64(len(syslogNames)) {
		return syslogNames[level]
	}
	return syslogNames[0]
}

// cmdIdlist — dump object ID list to the fixed C report file (LVL_GRGOD).
func cmdIdlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	// C ignores command arguments and always opens "object_idlist" relative to
	// the server process directory (src/act.wizard.c:3582-3589).
	f, err := os.Create("object_idlist")
	if err != nil {
		s.Send("Could not open id list file, cannot complete operation!\r\n")
		return nil
	}
	pw := s.manager.world.GetParsedWorld()
	count, writeErr := writeObjectIDList(f, pw.Objs)
	closeErr := f.Close()
	if writeErr != nil {
		slog.Error("idlist report write failed", "error", writeErr)
		return writeErr
	}
	if closeErr != nil {
		slog.Error("idlist report close failed", "error", closeErr)
		return closeErr
	}
	s.Send("Ok. Id list complete.\r\n")
	slog.Info("(GC) idlist", "who", s.player.Name, "file", "object_idlist", "count", count)
	return nil
}

// cmdCheckload — check zone load info for a mob/obj (LVL_IMMORT)
func cmdCheckload(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	const usage = "Usage: checkload { obj | mob } <number>\r\n"
	if len(args) < 2 || args[1] == "" || args[1][0] < '0' || args[1][0] > '9' {
		s.Send(usage)
		return nil
	}

	vnum := checkloadAtoi(args[1])
	w := s.manager.world
	switch strings.ToLower(args[0][:1]) {
	case "m":
		mob, ok := w.GetMobPrototype(vnum)
		if !ok {
			s.Send("That mob does not exist.\r\n")
			return nil
		}
		s.Send(checkloadMobReport(w, vnum, mob.ShortDesc))
	case "o":
		obj, ok := w.GetObjPrototype(vnum)
		if !ok {
			s.Send("That object does not exist.\r\n")
			return nil
		}
		s.Send(checkloadObjectReport(w, vnum, obj.ShortDesc))
	default:
		s.Send("Usage: checkload { obj | mob } <number>\r\n")
	}
	return nil
}

// checkloadAtoi mirrors the decimal-prefix behavior of C's atoi after
// do_checkload's first-byte isdigit gate.
func checkloadAtoi(value string) int {
	n := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func checkloadRoomName(w *game.World, vnum int) (string, bool) {
	room := w.GetRoomInWorld(vnum)
	if room == nil {
		return "", false
	}
	return room.Name, true
}

func checkloadMobReport(w *game.World, vnum int, name string) string {
	var report strings.Builder
	_, _ = fmt.Fprintf(&report, "Checking load info for %s...\r\n", name)
	found := false
	for _, zone := range w.GetAllZones() {
		for _, cmd := range zone.Commands {
			switch cmd.Command {
			case "M":
				if cmd.Arg1 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, cmd.Arg3)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         %d Max\r\n", cmd.Arg3, roomName, cmd.Arg2)
			case "R":
				// After renum_zone_table, R is Arg2 = kind (0 mobile) and
				// Arg3 = vnum (act.wizard.c:3709-3718).
				if cmd.Arg2 != 0 || cmd.Arg3 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, cmd.Arg1)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         Removed from room\r\n", cmd.Arg1, roomName)
			}
		}
	}
	if !found {
		report.WriteString(" Doesn't load anywhere.\r\n")
	}
	return report.String()
}

func checkloadObjectReport(w *game.World, vnum int, name string) string {
	var report strings.Builder
	_, _ = fmt.Fprintf(&report, "Checking load info for %s...\r\n", name)
	found := false
	lastRoomVNum := 0
	lastObjectVNum := 0
	lastObjectName := ""
	lastMobVNum := 0
	lastMobName := ""
	for _, zone := range w.GetAllZones() {
		for _, cmd := range zone.Commands {
			if cmd.Command == "O" || cmd.Command == "E" || cmd.Command == "G" {
				obj, ok := w.GetObjPrototype(cmd.Arg1)
				if !ok {
					continue
				}
				lastObjectVNum = obj.VNum
				lastObjectName = obj.ShortDesc
			}
			if cmd.Command == "M" || cmd.Command == "O" {
				lastRoomVNum = cmd.Arg3
			}

			switch cmd.Command {
			case "M":
				mob, ok := w.GetMobPrototype(cmd.Arg1)
				if ok {
					lastMobVNum = mob.VNum
					lastMobName = mob.ShortDesc
				}
			case "O":
				if cmd.Arg1 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, lastRoomVNum)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         Loaded to room\r\n         %.2f%% Load, %d Max\r\n", lastRoomVNum, roomName, mustCheckloadObjectPercent(w, lastObjectVNum), cmd.Arg2)
			case "P":
				if cmd.Arg1 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, lastRoomVNum)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         Put in %s [%d]\r\n         %.2f%% Load, %d Max\r\n", lastRoomVNum, roomName, lastObjectName, lastObjectVNum, mustCheckloadObjectPercent(w, lastObjectVNum), cmd.Arg2)
			case "E":
				if cmd.Arg1 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, lastRoomVNum)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         Equipped to %s [%d]\r\n         %.2f%% Load, %d Max\r\n", lastRoomVNum, roomName, lastMobName, lastMobVNum, mustCheckloadObjectPercent(w, lastObjectVNum), cmd.Arg2)
			case "G":
				if cmd.Arg1 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, lastRoomVNum)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         Given to %s [%d]\r\n         %.2f%% Load, %d Max\r\n", lastRoomVNum, roomName, lastMobName, lastMobVNum, mustCheckloadObjectPercent(w, lastObjectVNum), cmd.Arg2)
			case "R":
				// Every R moves the report's last room, matched or not, so a
				// following G/E/P names it; an object row is Arg2 nonzero
				// with Arg3 = vnum (act.wizard.c:3798-3807).
				lastRoomVNum = cmd.Arg1
				if cmd.Arg2 == 0 || cmd.Arg3 != vnum {
					continue
				}
				roomName, ok := checkloadRoomName(w, cmd.Arg1)
				if !ok {
					continue
				}
				found = true
				_, _ = fmt.Fprintf(&report, " [%5d] %s\r\n         Removed from room\r\n", cmd.Arg1, roomName)
			}
		}
	}
	if !found {
		report.WriteString(" Doesn't load anywhere.\r\n")
	}
	return report.String()
}

func mustCheckloadObjectPercent(w *game.World, vnum int) float64 {
	obj, ok := w.GetObjPrototype(vnum)
	if !ok {
		return 0
	}
	return obj.LoadPercent
}

// cmdPoofset — set poof in/out messages (LVL_IMMORT)
// Original: act.wizard.c do_poofset() (1711).
func cmdPoofsetText(s *Session, direction, message string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	// C do_poofset: with an argument, str_dup it into the poof slot; with no
	// argument, clear the slot. Either way it replies with the global OK
	// ("Okay.\r\n", config.c:92) — never an invented set/cleared message.
	player := s.GetPlayer()
	if player == nil {
		s.Send("You don't seem to exist.")
		return nil
	}

	if direction == "in" {
		player.PoofIn = message
	} else {
		player.PoofOut = message
	}
	s.Send("Okay.\r\n")
	return nil
}

func cmdPoofset(s *Session, args []string) error {
	if len(args) < 1 {
		s.Send("Usage: poofset <in|out> [message]")
		return nil
	}
	direction := strings.ToLower(args[0])
	if direction != "in" && direction != "out" {
		s.Send("Usage: poofset <in|out> [message]")
		return nil
	}
	return cmdPoofsetText(s, direction, strings.Join(args[1:], " "))
}

// cmdPoofin — standalone "poofin [message]" command.
// Source: src/interpreter.c do_poofset/SCMD_POOFIN, gated at LVL_IMMORT.
// Thin wrapper over cmdPoofset that fixes the direction to "in" (the C subcmd).
func cmdPoofin(s *Session, args []string) error {
	return cmdPoofset(s, append([]string{"in"}, args...))
}

// cmdPoofout — standalone "poofout [message]" command.
// Source: src/interpreter.c do_poofset/SCMD_POOFOUT, gated at LVL_IMMORT.
// Thin wrapper over cmdPoofset that fixes the direction to "out" (the C subcmd).
func cmdPoofout(s *Session, args []string) error {
	return cmdPoofset(s, append([]string{"out"}, args...))
}

// cmdWiznet — send message on wizard net (LVL_IMMORT)
// Original: act.wizard.c do_wiznet() — supports level-tagged, emote, and @list variants

// showZoneHooks follows src/act.wizard.c:2461-2487: C RNUM then direction order.
func showZoneHooks(w *game.World, zoneNumber int) string {
	names := make(map[int]string)
	for _, zone := range w.GetAllZones() {
		names[zone.Number] = zone.Name
	}
	var report strings.Builder
	fmt.Fprintf(&report, "Connections in zone %d.\r\n========================\r\n", zoneNumber)
	rooms := w.Rooms()
	for i := range rooms {
		room, ok := w.SnapshotRoom(rooms[i].VNum)
		if !ok || room.Zone != zoneNumber {
			continue
		}
		for _, direction := range game.DirectionNames {
			exit, exists := room.Exits[direction]
			if !exists {
				continue
			}
			destination, exists := w.SnapshotRoom(exit.ToRoom)
			if !exists || destination.Zone == zoneNumber {
				continue
			}
			fmt.Fprintf(&report, "%5d leads %s to %-5d -- %s\r\n", room.VNum, direction, destination.VNum, names[destination.Zone])
		}
	}
	return report.String()
}
