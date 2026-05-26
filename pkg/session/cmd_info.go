package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func cmdLevels(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}
	var buf strings.Builder
	for i := 1; i < 31; i++ { // LVL_IMMORT = 31
		xpNeeded := game.FindExp(p.Class, i)
		xpPrev := game.FindExp(p.Class, i-1)
		fmt.Fprintf(&buf, "[%2d] %8d-%-8d    (%6d)\r\n", i, xpPrev, xpNeeded, xpNeeded-xpPrev)
	}
	s.Send(buf.String())
	return nil
}

func cmdAbils(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}
	s.Send("Your current ability scores:\r\n")
	s.Send(fmt.Sprintf("Strength:      (%s)\r\n", getAbilName(p.Stats.Str)))
	s.Send(fmt.Sprintf("Dexterity:     (%s)\r\n", getAbilName(p.Stats.Dex)))
	s.Send(fmt.Sprintf("Intelligence:  (%s)\r\n", getAbilName(p.Stats.Int)))
	s.Send(fmt.Sprintf("Wisdom:        (%s)\r\n", getAbilName(p.Stats.Wis)))
	s.Send(fmt.Sprintf("Constitution:  (%s)\r\n", getAbilName(p.Stats.Con)))
	s.Send(fmt.Sprintf("Charisma:      (%s)\r\n", getAbilName(p.Stats.Cha)))
	return nil
}

func cmdCoins(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}
	s.Send(fmt.Sprintf("You are currently carrying %d coins,\r\n", p.Gold))
	s.Send(fmt.Sprintf("and in your bank account, you have %d coins.\r\n", p.BankGold))
	s.Send(fmt.Sprintf("Your current net-worth is %d coins.\r\n", p.Gold+p.BankGold))
	return nil
}

// alignmentText returns a text description for an alignment value (-1000 to 1000).
// Source: act.informative.c do_score() lines 1213-1238
func alignmentText(alignment int) string {
	switch {
	case alignment == 1000:
		return "You are the Epitome of Righteousness!"
	case alignment >= 900:
		return "You're so good, you make the angels jealous."
	case alignment >= 750:
		return "You are feeling pretty righteous."
	case alignment >= 500:
		return "You are aligned with the path of right."
	case alignment >= 350:
		return "You are feeling pretty good today."
	case alignment >= 100:
		return "You are a little more good than neutral, but yet still bland."
	case alignment > -100:
		return "You are neutral, how boring."
	case alignment > -350:
		return "You are little more evil than neutral, but not very exciting."
	case alignment > -500:
		return "I actually think you would kill your own mother."
	case alignment > -750:
		return "You are so evil it hurts."
	case alignment > -900:
		return "Charles Manson is in your fan club."
	default:
		return "You are the Epitome of Evil!"
	}
}

// acText returns a text description for an AC value.
// Source: act.informative.c do_score() lines 1240-1265
func acText(ac int) string {
	switch {
	case ac >= 100:
		return "You are naked, have you no shame?"
	case ac > 70:
		return "You are lightly clothed."
	case ac > 40:
		return "You are pretty well clothed."
	case ac > 10:
		return "You are lightly armored."
	case ac > -10:
		return "You are well armored."
	case ac > -40:
		return "You are getting pretty sweaty with all that armor on."
	case ac > -20:
		return "You are extremely well armored." // Note: this case is after -40, so -20..-40 hits here
	case ac > -50:
		return "You are extremely well armored."
	case ac > -75:
		return "You are decked out in full battle armor."
	case ac > -125:
		return "You are armored like a wyvern!"
	case ac > -150:
		return "You are armored like a dragon!"
	case ac > -175:
		return "You could walk through the gates of Hell in all that armor!"
	default:
		return "You are armored like a god!"
	}
}

// packWeightLabel returns a text label for the carried weight ratio.
func packWeightLabel(carriedWeight, maxCarryWeight int) string {
	if maxCarryWeight <= 0 {
		return ""
	}
	ratio := maxCarryWeight / carriedWeight
	if carriedWeight == 0 {
		ratio = -1
	}
	if ratio > -1 {
		ratio = -1
	}
	if ratio < 0 {
		ratio = 0
	}
	ratio = max(-1, ratio)

	// Actually let's just use the simpler logic from C
	// weight = CAN_CARRY_W / IS_CARRYING_W, then MAX(-1, weight)
	var weight int
	if carriedWeight > 0 {
		weight = maxCarryWeight / carriedWeight
	}
	if carriedWeight == 0 {
		weight = -1
	}
	if weight < -1 {
		weight = -1
	}
	if weight > 4 {
		weight = 4
	}

	switch {
	case weight >= 4:
		return "Your pack is light."
	case weight == 3:
		return "Your pack is fairly light."
	case weight == 2:
		return "Your pack is fairly heavy."
	case weight == 1:
		return "Your pack is heavy."
	case weight == 0:
		return "Your pack is almost too heavy to lift."
	case weight == -1:
		return "Your pack is empty."
	default:
		return ""
	}
}

// positionText returns a text description for a position value.
func positionText(pos int) string {
	switch pos {
	case game.PosDead:
		return "You are DEAD!(Tell a god)"
	case game.PosMortally:
		return "You are mortally wounded!  You should seek help!"
	case game.PosIncap:
		return "You are incapacitated, slowly fading away..."
	case game.PosStunned:
		return "You are stunned!  You can't move!"
	case game.PosSleeping:
		return "You are sleeping."
	case game.PosResting:
		return "You are resting."
	case game.PosSitting:
		return "You are sitting."
	case game.PosFighting:
		return "You are fighting."
	case game.PosStanding:
		return "You are standing."
	default:
		return "You are floating."
	}
}

func cmdScore(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}

	var buf strings.Builder

	// 1. Name + Age (from C line 1185)
	// GET_AGE(ch) calculation
	age := game.Age(p.Birth)
	buf.WriteString(fmt.Sprintf("%s                           Age: %d years", p.Name, age.Year))
	if age.Month == 0 && age.Day == 0 {
		buf.WriteString(" (It's your birthday today.)")
	}
	buf.WriteString("\r\n")

	// 2. HP/Mana/Move (from C line 1196)
	manaLabel := "Mana"
	buf.WriteString(fmt.Sprintf("Hit points: %d(%d)  %s: %d(%d)  Movement points: %d(%d)\r\n",
		p.Health, p.MaxHealth, manaLabel, p.Mana, p.MaxMana, p.Move, p.MaxMove))

	// 3. Alignment text (from C lines 1213-1238)
	buf.WriteString(alignmentText(p.Alignment))
	buf.WriteString("\r\n")

	// 4. AC text (from C lines 1240-1265)
	buf.WriteString(fmt.Sprintf("You %s\r\n", acText(p.AC)))

	// 5. Experience (from C line 1267)
	buf.WriteString(fmt.Sprintf("Experience:    %d points\r\n", p.Exp))

	// 6. Gold (from C lines 1268-1269)
	buf.WriteString(fmt.Sprintf("Coins carried: %d gold coins    ", p.Gold))
	buf.WriteString(fmt.Sprintf("Coins in bank: %d gold coins\r\n", p.BankGold))

	// 7. Kills/PKs/Deaths (from C line 1270)
	buf.WriteString(fmt.Sprintf("Kills: %d  Pks: %d  Deaths: %d\r\n", p.Kills, p.PKs, p.Deaths))

	// 8. XP to next level (from C line 1275)
	if p.Level < game.LVL_IMMORT-1 {
		needed := game.FindExp(p.Class, p.Level) - p.Exp
		if needed < 0 {
			needed = 0
		}
		buf.WriteString(fmt.Sprintf("You need %d exp to reach your next level.\r\n", needed))
	}

	// 9. Play time (from C line 1279)
	pt := game.PlayingTime(p.ConnectedAt, p.PlayedDuration)
	buf.WriteString(fmt.Sprintf("You have been playing for %d days and %d hours.\r\n", pt.Day, pt.Hours))

	// 10. Veteran status (from C line 1281)
	if pt.Day >= 30 && p.Kills >= 10000 {
		buf.WriteString("You are a veteran of many battles.\r\n")
	}

	// 11. Citizenship (from C line 1283)
	hometown := "Unknown"
	if p.Hometown >= 0 && p.Hometown < len(game.Hometowns) {
		hometown = game.Hometowns[p.Hometown]
	}
	buf.WriteString(fmt.Sprintf("You are a citizen of %s.\r\n", hometown))

	// 12. Clan info (from C lines 1284-1292)
	if p.ClanID != 0 && p.ClanRank != 0 {
		_, clan := s.manager.world.Clans.FindClanByID(p.ClanID)
		if clan != nil && p.ClanRank > 0 && p.ClanRank <= len(clan.RankName) && clan.RankName[p.ClanRank-1] != "" {
			buf.WriteString(fmt.Sprintf("You are a %s of %s.\r\n", clan.RankName[p.ClanRank-1], clan.Name))
		}
	}

	// 13. Title line (from C lines 1293-1294)
	className := game.ClassNames[p.Class]
	buf.WriteString(fmt.Sprintf("This ranks you as %s %s (level %d).\r\n", p.Name, p.Title, p.Level))

	// 14. Race + Class (from C lines 1295-1298)
	raceName := game.RaceNames[p.Race]
	buf.WriteString(fmt.Sprintf("You are %s %s %s.\r\n", articleFor(raceName), raceName, className))

	// 15. Pack weight (from C lines 1304-1315)
	carriedW := p.CarriedWeight()
	maxW := p.MaxCarryWeight()
	var weightText string
	if maxW > 0 {
		rat := -1
		if carriedW > 0 {
			rat = maxW / carriedW
			if rat > 4 {
				rat = 4
			}
		}
		if rat < -1 {
			rat = -1
		}
		switch {
		case rat >= 4:
			weightText = "Your pack is light."
		case rat == 3:
			weightText = "Your pack is fairly light."
		case rat == 2:
			weightText = "Your pack is fairly heavy."
		case rat == 1:
			weightText = "Your pack is heavy."
		case rat == 0:
			weightText = "Your pack is almost too heavy to lift."
		case rat == -1:
			weightText = "Your pack is empty."
		}
	}
	if weightText != "" {
		buf.WriteString(weightText + "\r\n")
	}

	// PLR_CHOSEN check (from C line 1316)
	// Skip — PLR_CHOSEN is bit 19

	// 16. Position (from C lines 1318-1340)
	buf.WriteString(positionText(p.Position))
	buf.WriteString("\r\n")

	// 17. Status conditions (from C lines 1342-1347)
	if p.Drunk > 0 {
		buf.WriteString("You are intoxicated.\r\n")
	}
	if p.Hunger <= 0 && p.Level < game.LVL_IMMORT {
		buf.WriteString("You are hungry.\r\n")
	}
	if p.Thirst <= 0 && p.Level < game.LVL_IMMORT {
		buf.WriteString("You are thirsty.\r\n")
	}

	// 18. Active affects (from C lines 1349-1369)
	if p.Affects&(1<<0) != 0 { // AFF_BLIND = bit 0
		buf.WriteString("You have been blinded!\r\n")
	}
	// Check PRF_SUMMONABLE flag on Player.Flags (PRF bit 48)
	if p.Flags&(1<<game.PrfSummonable) != 0 {
		buf.WriteString("You are summonable by other players.\r\n")
	}
	// AFF_WEREWOLF = bit 32 (in C structs.h: AFF_WEREWOLF bit 32)
	if p.Affects&(1<<32) != 0 {
		buf.WriteString("You're a lycanthrope!\r\n")
	}
	// AFF_VAMPIRE = bit 33
	if p.Affects&(1<<33) != 0 {
		buf.WriteString("You're a vampire!\r\n")
	}
	// AFF_MOUNT — check via IsAffected or flags
	// Source: structs.h AFF_MOUNT bit
	if p.Affects&(1<<10) != 0 { // AFF_MOUNT is bit 10 in structs.h
		buf.WriteString("You're mounted.\r\n")
	}
	// AFF_FLESH_ALTER = bit 16
	if p.Affects&(1<<16) != 0 {
		buf.WriteString("Your hand is a weapon!\r\n")
	}

	// 19. Spell affects list (from C lines 1371-1397)
	// C bit positions for filtering (structs.h):
	// AFF_SNEAK = 18, AFF_DODGE = 17, AFF_BERSERK = 20, AFF_ROBBED = 38
	// AFF_KUJI_KIRI = 24
	const (
		affSneakBit     uint64 = 1 << 18
		affDodgeBit     uint64 = 1 << 17
		affBerserkBit   uint64 = 1 << 20
		affRobbedBit    uint64 = 1 << 38
		affKujiKiriBit  uint64 = 1 << 24
	)

	if len(p.MasterAffects) > 0 || len(p.ActiveAffects) > 0 {
		var visibleAffects []string

		// Check MasterAffects first
		for _, aff := range p.MasterAffects {
			bv := uint64(aff.Bitvector)
			if bv == affSneakBit || bv == affDodgeBit || bv == affBerserkBit || bv == affRobbedBit {
				continue
			}
			// AFF_KUJI_KIRI with no modifier: skip if modifier == 0 and bitvector matches
			if bv == affKujiKiriBit && aff.Modifier == 0 {
				continue
			}
			visibleAffects = append(visibleAffects, fmt.Sprintf("%-24s", spells.GetSpellName(aff.Type)))
		}

		// Also check ActiveAffects (for spells that don't have MasterAffects)
		for _, aff := range p.ActiveAffects {
			bv := uint64(aff.Flags)
			if bv == affSneakBit || bv == affDodgeBit || bv == affBerserkBit || bv == affRobbedBit {
				continue
			}
			if bv == affKujiKiriBit && aff.Magnitude == 0 {
				continue
			}
			visibleAffects = append(visibleAffects, fmt.Sprintf("%-24s", spells.GetSpellName(aff.SpellID)))
		}

		if len(visibleAffects) > 0 {
			buf.WriteString("Spells affecting you:\r\n")
			for _, sa := range visibleAffects {
				buf.WriteString("     " + sa + "\r\n")
			}
		}
	}

	s.Send(buf.String())
	return nil
}

// articleFor returns "a" or "an" for a given word.
func articleFor(word string) string {
	if word == "" {
		return "a"
	}
	first := strings.ToLower(string(word[0]))
	switch first {
	case "a", "e", "i", "o", "u":
		return "an"
	default:
		return "a"
	}
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
