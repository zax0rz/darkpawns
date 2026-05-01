package session

import (
	"fmt"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdScore(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}
	s.Send(fmt.Sprintf("Name: %s  Level: %d  XP: %d/%d", p.Name, p.Level, p.Exp, 1000))
	s.Send(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d", p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove))
	s.Send(fmt.Sprintf("STR:%d  INT:%d  WIS:%d  DEX:%d  CON:%d  CHA:%d", p.Stats.Str, p.Stats.Int, p.Stats.Wis, p.Stats.Dex, p.Stats.Con, p.Stats.Cha))
	s.Send(fmt.Sprintf("AC:%d  Hitroll:%d  Damroll:%d  Align:%d  Gold:%d", p.AC, p.Hitroll, p.Damroll, p.Alignment, p.Gold))
	return nil
}

// cmdUsersSafe replaces cmdUsers to gate IP display behind LVL_GOD+.
// Regular immortals see name/level only; gods and above see IPs.
func cmdUsersSafe(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.sendText("Huh?!?")
		return nil
	}

	showIPs := s.player.Level >= LVL_GOD

	filter := ""
	if len(args) > 0 {
		filter = strings.ToLower(args[0])
	}

	var buf strings.Builder
	if showIPs {
		buf.WriteString(fmt.Sprintf("%-15s %-6s %-20s\n", "Name", "Level", "Remote Addr"))
		buf.WriteString(strings.Repeat("-", 45) + "\n")
	} else {
		buf.WriteString(fmt.Sprintf("%-15s %-6s\n", "Name", "Level"))
		buf.WriteString(strings.Repeat("-", 25) + "\n")
	}

	count := 0
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		name := sess.player.Name
		level := sess.player.GetLevel()

		if filter != "" && !strings.Contains(strings.ToLower(name), filter) {
			continue
		}

		if showIPs {
			ip := "unknown"
			if sess.request != nil {
				ip = sess.request.RemoteAddr
				if fwd := sess.request.Header.Get("X-Forwarded-For"); fwd != "" {
					ip = fwd
				}
			}
			buf.WriteString(fmt.Sprintf("%-15s %-6d %-20s\n", name, level, ip))
		} else {
			buf.WriteString(fmt.Sprintf("%-15s %-6d\n", name, level))
		}
		count++
	}

	buf.WriteString(fmt.Sprintf("\n%d player(s) connected.\n", count))
	s.sendText(buf.String())
	return nil
}

func cmdWho(s *Session) error {
	s.manager.mu.RLock()
	sessions := make([]*Session, 0, len(s.manager.sessions))
	for _, sess := range s.manager.sessions {
		sessions = append(sessions, sess)
	}
	s.manager.mu.RUnlock()

	isImm := s.player != nil && s.player.Level >= LVL_IMMORT

	out := "Players\n-------\n"
	count := 0
	for _, sess := range sessions {
		if sess.player == nil {
			continue
		}
		p := sess.player
		className := game.ClassNames[p.Class]
		raceName := game.RaceNames[p.Race]
		// Format: [ LV  Class ] Name Race — act.informative.c line 1874
		tag := "player"
		if sess.isAgent && isImm {
			tag = "agent"
		}
		out += fmt.Sprintf("[ %2d  %-8s] %-15s (%s, %s, %s)\n",
			p.Level, className, p.Name, raceName, className, tag)
		count++
	}
	if count == 0 {
		out += "\nNo-one at all!\n"
	} else if count == 1 {
		out += "\nOne character displayed.\n"
	} else {
		out += fmt.Sprintf("\n%d characters displayed.\n", count)
	}
	s.sendText(out)
	return nil
}

// cmdTell sends a private message to another player.
// Source: act.comm.c do_tell() lines 901-931, perform_tell()

// cmdEmote broadcasts a roleplay action to the room.
// Source: act.comm.c do_emote() — "$n laughs." style

// cmdShout broadcasts a message to all players in the same zone.
// Source: act.comm.c do_gen_comm() SCMD_SHOUT lines 1286-1289
// Original: zone-scoped; receivers must be POS_RESTING or higher.

// cmdWhere lists all online players and their locations.
// Source: act.informative.c do_where() lines 2244-2307
func cmdWhere(s *Session) error {
	s.manager.mu.RLock()
	sessions := make([]*Session, 0, len(s.manager.sessions))
	for _, sess := range s.manager.sessions {
		sessions = append(sessions, sess)
	}
	s.manager.mu.RUnlock()

	out := "Players\n-------\n"
	found := false
	for _, sess := range sessions {
		if sess.player == nil {
			continue
		}
		p := sess.player
		room, ok := s.manager.world.GetRoom(p.GetRoom())
		if !ok {
			continue
		}
		// Format mirrors do_where() line 2272: name - [vnum] room name
		out += fmt.Sprintf("%-20s - [%5d] %s\n", p.Name, room.VNum, room.Name)
		found = true
	}
	if !found {
		out += "No-one visible.\n"
	}
	s.sendText(out)
	return nil
}

// cmdSummon pulls a named player into your current room. Debug/admin convenience.
func cmdSummon(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Summon who?")
		return nil
	}
	targetName := strings.ToLower(args[0])
	s.manager.mu.RLock()
	defer s.manager.mu.RUnlock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		if strings.ToLower(sess.player.Name) == targetName {
			old := sess.player.RoomVNum
			sess.player.RoomVNum = s.player.RoomVNum
			s.sendText(fmt.Sprintf("%s materializes before you.", sess.player.Name))
			sess.sendText(fmt.Sprintf("You are summoned by %s.", s.player.Name))
			_ = old
			return nil
		}
	}
	s.sendText("No one by that name online.")
	return nil
}

// cmdHelp provides a basic help stub.
// Full implementation deferred to a later phase.
func cmdHelp(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Available commands: look, north/south/east/west/up/down, say, hit, flee, " +
			"inventory, equipment, wear, remove, wield, hold, get, drop, " +
			"score, who, tell, emote, shout, where, quit, " +
			"open, close, lock, unlock, pick, bashdoor\n" +
			"Type 'help <topic>' for more info (stub — full help coming later).")
		return nil
	}
	s.sendText(fmt.Sprintf("No help available for '%s' yet.", strings.Join(args, " ")))
	return nil
}

// directions maps abbreviated direction names to full names.
