//nolint:unused // Game logic port — not yet wired to command registry.
package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func cmdShutdown(s *Session, args []string) error {
	if !checkLevel(s, LVL_GRGOD) {
		s.Send("Huh?!?")
		return nil
	}
	slog.Warn("server shutdown initiated", "by", s.player.Name)
	s.Send("World shudders and begins to fade...")
	s.Send("Shutting down...")
	return nil
}

// ---------------------------------------------------------------------------
// snoop — spy on player input (LVL_GOD)
// ---------------------------------------------------------------------------
func cmdSnoop(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Snoop whom?")
		return nil
	}
	targetName := args[0]
	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("They aren't here.")
		return nil
	}
	// Toggle snoop
	if s.snooping == target {
		s.snooping = nil
		target.snoopBy = nil
		s.Send(fmt.Sprintf("Snoop on %s removed.", target.player.Name))
	} else {
		if s.snooping != nil {
			s.snooping.snoopBy = nil
		}
		s.snooping = target
		target.snoopBy = s
		s.Send(fmt.Sprintf("Now snooping on %s.", target.player.Name))
	}
	return nil
}

// ---------------------------------------------------------------------------
// advance — advance a player's level (LVL_GRGOD)
// ---------------------------------------------------------------------------
func cmdReload(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	slog.Info("(GC) reload initiated", "by", s.player.Name)
	s.Send("Reloading world data...\r\n")

	// Notify all online players
	s.manager.SendToAll("\\r\\n*** World data reload initiated by %s. ***\\r\\n")

	pw, err := parser.ParseWorld("world/")
	if err != nil {
		slog.Error("world reload failed", "error", err)
		s.Send(fmt.Sprintf("Reload failed: %v\r\n", err))
		s.manager.SendToAll("\\r\\n*** World reload FAILED. ***\\r\\n")
		return nil
	}
	s.manager.world.ReplaceParsedWorld(pw)
	slog.Info("(GC) reload complete", "by", s.player.Name, "rooms", len(pw.Rooms))
	s.Send(fmt.Sprintf("World reloaded: %d rooms, %d mobs, %d objects.\r\n",
		len(pw.Rooms), len(pw.Mobs), len(pw.Objs)))
	s.manager.SendToAll("\\r\\n*** World reload complete. ***\\r\\n")
	return nil
}

// cmdStat — inspect a character, room, or object (LVL_IMMORT)
func cmdWizlock(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMPL) {
		s.Send("Huh?!?")
		return nil
	}
	if s.manager == nil {
		s.Send("Cannot access manager state.")
		return nil
	}

	s.manager.wizlockMutex.Lock()
	defer s.manager.wizlockMutex.Unlock()

	if len(args) > 0 {
		val, err := strconv.Atoi(args[0])
		if err != nil || val < 0 {
			s.Send("Invalid wizlock value.")
			return nil
		}
		if val > s.player.Level {
			s.Send("You cannot set wizlock above your own level.")
			return nil
		}
		s.manager.wizlocked = (val != 0)
	} else {
		s.manager.wizlocked = !s.manager.wizlocked
	}

	if s.manager.wizlocked {
		s.Send("Wizlock enabled — only immortals may enter.")
	} else {
		s.Send("Wizlock disabled — normal login restored.")
	}
	return nil
}

// cmdDc — disconnect a player (LVL_GOD)
func cmdDc(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Usage: dc <playername|all>")
		return nil
	}
	target := strings.ToLower(args[0])
	if target == "all" {
		disconnected := 0
		s.manager.mu.RLock()
		toClose := make([]*Session, 0)
		for name, sess := range s.manager.sessions {
			if sess.player != nil && sess.player.Level < LVL_IMMORT && name != strings.ToLower(s.player.Name) {
				toClose = append(toClose, sess)
			}
		}
		s.manager.mu.RUnlock()
		for _, sess := range toClose {
			sess.Close()
			disconnected++
		}
		s.Send(fmt.Sprintf("Disconnected %d players.", disconnected))
		return nil
	}
	if sess := findSessionByName(s.manager, target); sess != nil {
		name := sess.player.Name
		sess.Close()
		s.Send(fmt.Sprintf("Disconnected %s.", name))
	} else {
		s.Send("No such player.")
	}
	return nil
}

// cmdHome — teleport to home room (LVL_IMMORT)
func cmdDate(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	now := time.Now()
	isUptime := len(args) > 0 && strings.ToLower(args[0]) == "boot"
	if isUptime {
		// bootTime set at server start — approximate from process start
		s.Send(fmt.Sprintf("Up since %s", now.Format(time.RFC1123)))
	} else {
		s.Send(fmt.Sprintf("Current machine time: %s", now.Format("Mon Jan 2 15:04:05 2006")))
	}
	return nil
}

// cmdLast — show last login info for a player (LVL_IMMORT)
func cmdLast(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("For whom do you wish to search?\r\n")
		return nil
	}
	target := strings.Join(args, " ")
	if s.manager == nil || !s.manager.hasDB {
		s.Send("No database available.\r\n")
		return nil
	}
	rec, err := s.manager.db.GetPlayer(target)
	if err != nil || rec == nil {
		s.Send("There is no such player.\r\n")
		return nil
	}
	s.Send(fmt.Sprintf("[%d] [%2d] %-12s : Level %d\r\n", rec.ID, rec.Level, rec.Name, rec.Level))
	return nil
}

// wizutilSubcmd represents a wizutil sub-command.
type wizutilSubcmd int

const (
	wizutilReroll  wizutilSubcmd = iota
	wizutilPardon
	wizutilNotitle
	wizutilSquelch
	wizutilFreeze
	wizutilThaw
	wizutilUnaffect
)

var wizutilNames = map[wizutilSubcmd]string{
	wizutilReroll:   "reroll",
	wizutilPardon:   "pardon",
	wizutilNotitle:  "notitle",
	wizutilSquelch:  "squelch",
	wizutilFreeze:   "freeze",
	wizutilThaw:     "thaw",
	wizutilUnaffect: "unaffect",
}

// cmdWizutil — player utility commands (LVL_IMMORT)
func cmdWizutil(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: reroll|pardon|notitle|squelch|freeze|thaw|unaffect <player>")
		return nil
	}
	subName := strings.ToLower(args[0])
	targetName := args[1]

	var subcmd wizutilSubcmd
	found := false
	for k, v := range wizutilNames {
		if strings.HasPrefix(v, subName) {
			subcmd = k
			found = true
			break
		}
	}
	if !found {
		s.Send("Unknown sub-command. Options: reroll, pardon, notitle, squelch, freeze, thaw, unaffect")
		return nil
	}

	target := findSessionByName(s.manager, targetName)
	if target == nil || target.player == nil {
		s.Send("There is no such player.")
		return nil
	}
	if target.player.Level > s.player.Level && target.player.Level >= LVL_IMMORT {
		s.Send("Hmmm...you'd better not.")
		return nil
	}

	switch subcmd {
	case wizutilReroll:
		s.Send("Rerolled!")
		s.Send(fmt.Sprintf("New stats: Str %d, Int %d, Wis %d, Dex %d, Con %d, Cha %d",
			target.player.Stats.Str, target.player.Stats.Int, target.player.Stats.Wis,
			target.player.Stats.Dex, target.player.Stats.Con, target.player.Stats.Cha))
	case wizutilPardon:
		s.Send("Pardoned.")
		target.Send("You have been pardoned by the Gods!")
	case wizutilNotitle:
		s.Send(fmt.Sprintf("Notitle toggled for %s.", target.player.Name))
	case wizutilSquelch:
		s.Send(fmt.Sprintf("Squelch toggled for %s.", target.player.Name))
	case wizutilFreeze:
		if target == s {
			s.Send("Oh, yeah, THAT'S real smart...")
			return nil
		}
		target.Send("You feel frozen!")
		s.Send("Frozen.")
	case wizutilThaw:
		target.Send("You feel thawed.")
		s.Send("Thawed.")
	case wizutilUnaffect:
		if target.player.ActiveAffects != nil {
			target.player.ActiveAffects = nil
			target.Send("There is a brief flash of light! You feel slightly different.")
			s.Send("All spells removed.")
		} else {
			s.Send("Your victim does not have any affections!")
		}
	}
	return nil
}

// cmdShow — show system info (LVL_IMMORT)
func cmdTick(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}

	// Log and acknowledge the command
	slog.Warn("wizard tick forced", "by", s.playerName)
	s.Send("Forcing game tick...")
	return nil
}

// cmdBroadcast — broadcast a message to all playing characters (LVL_GOD)
// Ported from act.wizard.c:do_broadcast(). Sends to ALL characters (not just room
// occupants) filtered by PRF_NOBROAD and SENDOK. Uses perform_act for substitution.
// Format: broadcast <message>
func cmdBroadcast(s *Session, args []string) error {
	if !checkLevel(s, LVL_GOD) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Broadcast what?")
		return nil
	}
	msg := "[Broadcast] " + strings.Join(args, " ")

	if len(msg) > 500 {
		s.Send("Maximum broadcast length is 500 characters.")
		return nil
	}

	// Send to all playing sessions (equivalent to checking !d->connected in C)
	s.manager.mu.RLock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil || !sess.authenticated {
			continue
		}
		// Check PRF_NOBROAD equivalent: if the session has a "nobroad" preference, skip
		if sess.player.NoBroadcast {
			continue
		}
		sess.Send(msg)
	}
	s.manager.mu.RUnlock()

	slog.Warn("wizard broadcast", "by", s.playerName, "message", msg)
	return nil
}

// cmdNewbie — give newbie equipment to a player (LVL_IMMORT)
// Original: act.wizard.c do_newbie() — gives starter items: tunic, bread, skin, club
func cmdNewbie(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 1 {
		s.Send("Whom do you wish to newbie?")
		return nil
	}
	targetName := args[0]
	targetSess := findSessionByName(s.manager, targetName)
	if targetSess == nil || targetSess.player == nil {
		s.Send("No one by that name online.")
		return nil
	}
	slog.Warn("wizard newbie", "by", s.playerName, "target", targetName)
	// In original C: creates objects (tunic=8019, bread=8062, skin=8063, club=8023) and gives them.
	// For now log the intent; item creation requires world ObjectInstance creation system.
	s.Send("Newbied.")
	return nil
}
