package session

import (
	"fmt"
	"github.com/zax0rz/darkpawns/pkg/game"
	"strings"
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
		fmt.Fprintf(&buf, "%-15s %-6s %-20s\n", "Name", "Level", "Remote Addr")
		buf.WriteString(strings.Repeat("-", 45) + "\n")
	} else {
		fmt.Fprintf(&buf, "%-15s %-6s\n", "Name", "Level")
		buf.WriteString(strings.Repeat("-", 25) + "\n")
	}

	count := 0
	s.manager.mu.RLock()
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
			fmt.Fprintf(&buf, "%-15s %-6d %-20s\n", name, level, ip)
		} else {
			fmt.Fprintf(&buf, "%-15s %-6d\n", name, level)
		}
		count++
	}
	s.manager.mu.RUnlock()

	fmt.Fprintf(&buf, "\n%d player(s) connected.\n", count)
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
	switch count {
	case 0:
		out += "\nNo-one at all!\n"
	case 1:
		out += "\nOne character displayed.\n"
	default:
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

// Help topics provides category-based help beyond individual commands.
// Each entry describes a theme or system within the game.
var helpTopics = map[string]string{
	"commands":      "Type 'commands' to see all available commands.",
	"movement":      "Move around the world using: north, south, east, west, up, down (or n/s/e/w/u/d).",
	"combat":        "Fighting commands: hit <target>, flee, bash <target>, kick <target>, backstab <target>, rescue <target>, trip <target>, headbutt <target>, disembowel <target>, dragonkick <target>, tigerpunch <target>, subdue <target>, shoot <target>, parry, sneak, hide, ambush, neckbreak, sleeper, consider <target>, diagnose <target>",
	"communication": "Chat commands: say <text>, tell <player> <text>, reply <text>, whisper <player> <text>, emote <text>, shout <text>, gossip <text>, gtell/gsay <text>",
	"items":         "Item commands: get <item>, drop <item>, inventory/i, equipment/eq, wear <item>, remove <item>, wield <item>, hold <item>, give <item> <player>, put <item> <container>, eat <item>, drink <container>, quaff <item>",
	"doors":         "Door commands: open <dir>, close <dir>, lock <dir>, unlock <dir>, pick <dir>, bashdoor <dir>, knock <dir>",
	"social":        "Social commands: wave, nod, grin, laugh, bow, curtsey, hug, kiss, cheer, cry, dance, smile, frown, shrug, clap, salute, yawn, stretch, scratch, sit, rest, sleep, wake, stand",
	"info":          "Information commands: score, who, where, review, whois, consider, examine, time, weather, affects, autoexit, title, describe, spells, commands",
	"account":       "Account commands: password <old> <new> (change password), prompt [string|on|off|all], save, quit",
	"groups":        "Group commands: follow <player>, group <player>, ungroup/disband, gtell/gsay <text>, split <amount>, assist <target>",
	"reporting":     "Player help: report <player> <type> [desc] — report abusive behavior. Types: harassment, spam, cheating, hate_speech, exploit, other. Also: bug, typo, idea, todo",
	"admin":         "Admin commands (admin/mod only): warn <player> <reason>, mute <player> <duration> [reason], kick <player> <reason>, ban <player> <duration/permanent> [reason], reports [pending], penalties, investigate <player>, filter <add|remove|list>, spamconfig <threshold> [action], users, wizlock, last, snoop, force",
	"wizard":        "Wizard commands (immortal+): goto <vnum/player>, at <vnum> <cmd>, stat <target>, vnum <keyword>, vstat <vnum>, load <vnum>, purge, heal <player>, restore <player>, set <target> <field> <value>, switch <target>, return, invis, vis, gecho <msg>, echo <msg>, send <target> <msg>, reload, shut down, advance <target>, poofset, wiznet, zreset, zlist, rlist, olist, mlist, home, date, last, wizutil, show, dark, syslog",
	"mounts":        "Mount commands: ride <mount>, dismount, yank <player> from mount",
	"ignore":        "Ignore system: ignore <player> to toggle ignoring someone, ignore alone shows your ignore list.",
	"skills":        "Skill commands: skills (show learned skills), practice <skill>, learn <skill>, listskills (list available skills), skillinfo <skill> (show skill details), use <skill> <target>",
	"shops":         "Shop commands: list (show items for sale), buy <item>, sell <item>, appraise <item>",
	"clans":         "Clan commands: clan (shows clan info), clan create <name>, clan join <name>, clan leave, clan members, clan info",
	"houses":        "House commands: house (manage your house), hcontrol (admin house control)",
}

// cmdHelp provides help on commands and game systems.
func cmdHelp(s *Session, args []string) error {
	if len(args) == 0 {
		// Show overview of categories
		s.sendText("Help Topics\n===========\n" +
			"Type 'help <topic>' for detailed help.\n\n" +
			"Game topics: commands, movement, combat, communication, items, doors, social, info, account, groups, reporting, skills, shops, mounts, ignore, clans, houses\n" +
			"Admin: admin, wizard\n" +
			"Type 'help <topic>' for details, or 'help <command>' for a specific command.")
		return nil
	}

	topic := strings.ToLower(strings.Join(args, " "))

	// Check helpTopics first
	if text, ok := helpTopics[topic]; ok {
		s.sendText("[" + topic + "]\n" + strings.Repeat("-", len(topic)+4) + "\n" + text)
		return nil
	}

	// Check registered commands
	if entry, ok := cmdRegistry.Lookup(topic); ok {
		desc := entry.HelpText
		if desc == "" {
			desc = topic + " (command)"
		}
		aliases := ""
		if len(entry.Aliases) > 0 {
			aliases = "\nAliases: /" + strings.Join(entry.Aliases, ", /")
		}
		levelInfo := ""
		if entry.MinLevel > 0 {
			levelInfo = fmt.Sprintf("\nMinimum level: %d", entry.MinLevel)
		}
		s.sendText(fmt.Sprintf("%s%s%s", desc, aliases, levelInfo))
		return nil
	}

	// Try fuzzy match on help topics
	var suggestions []string
	for helpTopic := range helpTopics {
		if strings.Contains(helpTopic, topic) || strings.Contains(topic, helpTopic) {
			suggestions = append(suggestions, helpTopic)
		}
	}
	// Also fuzzy match on commands
	for _, entry := range cmdRegistry.GetAll() {
		if strings.Contains(entry.Name, topic) || strings.Contains(topic, entry.Name) {
			suggestions = append(suggestions, entry.Name)
		}
	}

	if len(suggestions) > 0 {
		s.sendText(fmt.Sprintf("No exact match for '%s'. Did you mean:\n  %s",
			topic, strings.Join(suggestions, ", ")))
	} else {
		s.sendText(fmt.Sprintf("No help available for '%s'. Try 'help' for a list of topics.", topic))
	}
	return nil
}

// cmdReview shows recent gossip history.
// Matches C: do_review() in new_cmds.c.
func cmdReview(s *Session) error {
	if s.player == nil {
		return nil
	}
	result := game.DoReview(s.player, s.manager.world)
	if result.MessageToCh != "" {
		s.sendText(result.MessageToCh)
	}
	return nil
}

// cmdWhois looks up a player by name in the database.
// Matches C: do_whois() in new_cmds.c — loads player save file.
func cmdWhois(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	if len(args) == 0 {
		s.sendText("For whom do you wish to search?\r\n")
		return nil
	}
	targetName := strings.Join(args, " ")

	// Check online players first
	for _, p := range s.manager.world.AllPlayers() {
		if strings.EqualFold(p.Name, targetName) {
			classAbbr := "??"
			if p.Class >= 0 && p.Class < len(game.ClassAbbrevs) {
				classAbbr = game.ClassAbbrevs[p.Class]
			}
			s.sendText(fmt.Sprintf("[%2d %s] %s\r\n", p.Level, classAbbr, p.Name))
			return nil
		}
	}

	// Check database for offline players
	if s.manager.hasDB {
		rec, err := s.manager.db.GetPlayer(targetName)
		if err != nil {
			s.sendText("Error looking up player.\r\n")
			return nil
		}
		if rec == nil {
			s.sendText("There is no such player.\r\n")
			return nil
		}
		classAbbr := "??"
		if rec.Class >= 0 && rec.Class < len(game.ClassAbbrevs) {
			classAbbr = game.ClassAbbrevs[rec.Class]
		}
		s.sendText(fmt.Sprintf("[%2d %s] %s\r\n", rec.Level, classAbbr, rec.Name))
		return nil
	}

	s.sendText("There is no such player.\r\n")
	return nil
}

// directions maps abbreviated direction names to full names.
