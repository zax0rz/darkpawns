package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
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
		count := len(s.manager.sessions)
		s.Send(fmt.Sprintf("Players online: %d", count))
	case "uptime":
		s.Send(fmt.Sprintf("Server running since %s", time.Now().Format(time.RFC1123)))
	case "stats":
		s.Send(fmt.Sprintf("Sessions: %d", len(s.manager.sessions)))
	case "reset":
		s.Send("Show reset is not yet implemented.")
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
	// Stop combat for everyone in the room
	roomVNum := s.player.GetRoom()
	s.Send("You stop the senseless violence in the room with a wave of your hand.\r\n")
	s.manager.BroadcastToRoom(roomVNum, []byte(fmt.Sprintf("%s raises a hand and combat freezes!\r\n", s.player.Name)), s.playerName)
	// Stop fighting for all mobs in the room
	for _, mob := range s.manager.world.GetMobsInRoom(roomVNum) {
		if stopper, ok := interface{}(mob).(interface{ StopFighting() }); ok {
			stopper.StopFighting()
		}
	}
	// Stop fighting for all players in the room
	for _, p := range s.manager.world.GetPlayersInRoom(roomVNum) {
		if p != s.player {
			p.StopFighting()
		}
	}
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

// cmdIdlist — dump object ID list to file (LVL_IMPL)
// Security: filename is always sanitized via filepath.Base() to prevent path traversal,
// and output is restricted to the data/ directory. Even though the filename is currently
// hardcoded, this defense-in-depth guard prevents regression if user args are re-enabled.
func cmdIdlist(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMPL) {
		s.Send("Huh?!?")
		return nil
	}
	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return nil
	}
	pw := s.manager.world.GetParsedWorld()
	if pw == nil {
		s.Send("World data not loaded.")
		return nil
	}

	// Sanitize filename — strip any path components to prevent directory traversal
	filename := "idlist.txt"
	if len(args) > 0 {
		filename = filepath.Base(args[0])
	}

	// Force output into data/ directory
	const safeDir = "data"
	if err := os.MkdirAll(safeDir, 0755); err != nil {
		s.Send(fmt.Sprintf("Could not create %s directory: %v\r\n", safeDir, err))
		return nil
	}
	safePath := filepath.Join(safeDir, filename)

	f, err := os.Create(safePath)
	if err != nil {
		s.Send(fmt.Sprintf("Could not create %s: %v\r\n", safePath, err))
		return nil
	}
	defer func() { _ = f.Close() }()
	for _, obj := range pw.Objs {
		_, _ = fmt.Fprintf(f, "[%d] %s\n", obj.VNum, obj.ShortDesc)
		_, _ = fmt.Fprintf(f, "  Keywords: %s  Type: %d  Cost: %d\n", obj.Keywords, obj.TypeFlag, obj.Cost)
		_, _ = fmt.Fprintf(f, "  Values: [%d] [%d] [%d] [%d]\n", obj.Values[0], obj.Values[1], obj.Values[2], obj.Values[3])
	}
	s.Send(fmt.Sprintf("Wrote %d objects to %s\r\n", len(pw.Objs), safePath))
	slog.Info("(GC) idlist", "who", s.player.Name, "file", safePath, "count", len(pw.Objs))
	return nil
}

// cmdCheckload — check zone load info for a mob/obj (LVL_IMMORT)
func cmdCheckload(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: checkload <mob|obj> <vnum>")
		return nil
	}
	s.Send(fmt.Sprintf("Checkload %s %s — not yet implemented (requires zone table).", args[0], args[1]))
	return nil
}

// cmdPoofset — set poof in/out messages (LVL_IMMORT)
// Original: act.wizard.c do_poofset()
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
	var msg string
	if len(args) >= 2 {
		msg = strings.Join(args[1:], " ")
	}
	if direction == "in" {
		if msg == "" {
			s.SetTempData("poofin", nil)
			s.Send("Poofin cleared.")
		} else {
			s.SetTempData("poofin", msg)
			s.Send("Ok.")
		}
	} else {
		if msg == "" {
			s.SetTempData("poofout", nil)
			s.Send("Poofout cleared.")
		} else {
			s.SetTempData("poofout", msg)
			s.Send("Ok.")
		}
	}
	return nil
}

// cmdWiznet — send message on wizard net (LVL_IMMORT)
// Original: act.wizard.c do_wiznet() — supports level-tagged, emote, and @list variants
