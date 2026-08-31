package session

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdShow(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: show <players|uptime|stats|reset>")
		return nil
	}
	topic := strings.ToLower(args[0])
	switch topic {
	case "players":
		s.manager.mu.RLock()
		count := len(s.manager.sessions)
		s.manager.mu.RUnlock()
		s.Send(fmt.Sprintf("Players online: %d", count))
	case "uptime":
		s.Send(fmt.Sprintf("Server running since %s", time.Now().Format(time.RFC1123)))
	case "stats":
		s.manager.mu.RLock()
		sessionCount := len(s.manager.sessions)
		s.manager.mu.RUnlock()
		s.Send(fmt.Sprintf("Sessions: %d", sessionCount))
	case "reset":
		zones := s.manager.world.GetAllZones()
		var buf strings.Builder
		fmt.Fprintf(&buf, "Zone Reset Information (%d zones):\r\n", len(zones))
		for _, z := range zones {
			resetInterval := "never"
			if z.Lifespan > 0 {
				resetInterval = fmt.Sprintf("%d min", z.Lifespan)
			}
			resetMode := "never"
			switch z.ResetMode {
			case 1:
				resetMode = "if empty"
			case 2:
				resetMode = "always"
			}
			fmt.Fprintf(&buf, "  [%5d] %-30s reset=%s mode=%s\r\n",
				z.Number, z.Name, resetInterval, resetMode)
		}
		s.Send(buf.String())
	default:
		s.Send(fmt.Sprintf("Unknown topic: %s", topic))
	}
	return nil
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
		s.Send("Your syslog is currently normal.")
		s.Send("Usage: syslog { Off | Brief | Normal | Complete }")
		return nil
	}
	level := strings.ToLower(args[0])
	switch level {
	case "off", "brief", "normal", "complete":
		s.Send(fmt.Sprintf("Your syslog is now %s.", level))
	default:
		s.Send("Usage: syslog { Off | Brief | Normal | Complete }")
	}
	return nil
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
				if cmd.Arg3 == -1 {
					if cmd.Arg2 != vnum {
						continue
					}
				} else if cmd.Arg3 != 0 || cmd.Arg2 != vnum {
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
				if cmd.Arg3 == -1 {
					if cmd.Arg2 != vnum {
						continue
					}
				} else if cmd.Arg3 != 1 || cmd.Arg2 != vnum {
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
func cmdPoofset(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Usage: poofset <in|out> [message]")
		return nil
	}
	direction := strings.ToLower(args[0])
	if direction != "in" && direction != "out" {
		s.Send("Usage: poofset <in|out> [message]")
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
		player.PoofIn = strings.Join(args[1:], " ")
	} else {
		player.PoofOut = strings.Join(args[1:], " ")
	}
	s.Send("Okay.\r\n")
	return nil
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
