package session

import (
	"fmt"
	"sort"
	"strconv"
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
	// 30 lines > PAGE_LENGTH(22) → paginated on plain-text clients; agents and
	// structured-data clients receive the whole table (DP-1195).
	PageString(s, buf.String())
	return nil
}

func cmdAbils(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}
	s.Send("Your current ability scores:\r\n")
	s.Send(fmt.Sprintf("Strength:      (%s)\r\n", getAbilName(cDisplayAbility(p.GetStr()))))
	s.Send(fmt.Sprintf("Dexterity:     (%s)\r\n", getAbilName(cDisplayAbility(p.GetDex()))))
	s.Send(fmt.Sprintf("Intelligence:  (%s)\r\n", getAbilName(cDisplayAbility(p.GetInt()))))
	s.Send(fmt.Sprintf("Wisdom:        (%s)\r\n", getAbilName(cDisplayAbility(p.GetWis()))))
	s.Send(fmt.Sprintf("Constitution:  (%s)\r\n", getAbilName(cDisplayAbility(p.GetCon()))))
	s.Send(fmt.Sprintf("Charisma:      (%s)\r\n", getAbilName(p.GetCha())))
	return nil
}

// cDisplayAbility mirrors affect_total's player-stat normalization
// (handler.c:352-366). C keeps these five displayed ability values in [0,18]
// after applying equipment and spell affects; exceptional strength is carried
// separately in GET_ADD and does not change do_abils' GET_STR output.
func cDisplayAbility(score int) int {
	if score < 0 {
		return 0
	}
	if score > 18 {
		return 18
	}
	return score
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
	fmt.Fprintf(&buf, "%s                           Age: %d years", p.Name, age.Year)
	if age.Month == 0 && age.Day == 0 {
		buf.WriteString(" (It's your birthday today.)")
	}
	buf.WriteString("\r\n")

	// 2. HP/Mana/Move (from C line 1196)
	manaLabel := scoreManaLabel(p.Class)
	fmt.Fprintf(&buf, "Hit points: %d(%d)  %s points: %d(%d)  Movement points: %d(%d)\r\n",
		p.GetHP(), p.GetMaxHP(), manaLabel, p.GetMana(), p.GetMaxMana(), p.GetMove(), p.GetMaxMove())

	// 3. Alignment text (from C lines 1213-1238)
	buf.WriteString(alignmentText(p.Alignment))
	buf.WriteString("\r\n")

	// 4. AC text (from C lines 1240-1265). acText already returns "You are ..."
	// sentences, so do not prepend another "You" — that produced "You You are well armored."
	buf.WriteString(acText(p.GetAC()))
	buf.WriteString("\r\n")

	// 5. Experience (from C line 1267)
	fmt.Fprintf(&buf, "Experience:    %d points\r\n", p.Exp)

	// 6. Gold (from C lines 1268-1269)
	fmt.Fprintf(&buf, "Coins carried: %d gold coins    ", p.Gold)
	fmt.Fprintf(&buf, "Coins in bank: %d gold coins\r\n", p.BankGold)

	// 7. Kills/PKs/Deaths (from C line 1270)
	fmt.Fprintf(&buf, "Kills: %d  Pks: %d  Deaths: %d\r\n", p.Kills, p.PKs, p.Deaths)

	// 8. XP to next level (from C line 1275)
	if p.Level < game.LVL_IMMORT-1 {
		needed := game.FindExp(p.Class, p.Level) - p.Exp
		if needed < 0 {
			needed = 0
		}
		fmt.Fprintf(&buf, "You need %d exp to reach your next level.\r\n", needed)
	}

	// 9. Play time (from C line 1279)
	pt := game.PlayingTime(p.ConnectedAt, p.PlayedDuration)
	fmt.Fprintf(&buf, "You have been playing for %d days and %d hours.\r\n", pt.Day, pt.Hours)

	// 10. Veteran status (from C line 1281)
	if pt.Day >= 30 && p.Kills >= 10000 {
		buf.WriteString("You are a veteran of many battles.\r\n")
	}

	// 11. Citizenship (from C line 1283)
	fmt.Fprintf(&buf, "You are a citizen of %s.\r\n", game.HometownName(p.Hometown))

	// 12. Clan info (from C lines 1284-1292)
	if p.ClanID != 0 && p.ClanRank != 0 {
		_, clan := s.manager.world.Clans.FindClanByID(p.ClanID)
		if clan != nil && p.ClanRank > 0 && p.ClanRank <= len(clan.RankName) && clan.RankName[p.ClanRank-1] != "" {
			fmt.Fprintf(&buf, "You are a %s of %s.\r\n", clan.RankName[p.ClanRank-1], clan.Name)
		}
	}

	// 13. Title line (from C lines 1293-1294)
	className := game.ClassNames[p.Class]
	fmt.Fprintf(&buf, "This ranks you as %s %s (level %d).\r\n", p.Name, p.Title, p.Level)

	// 14. Race + Class (from C lines 1295-1298)
	raceName := game.RaceNames[p.Race]
	fmt.Fprintf(&buf, "You are %s %s %s.\r\n", articleFor(raceName), raceName, className)

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

	// PLR_CHOSEN check (from C line 1316).
	if p.HasPLRFlag(game.PlrChosen) {
		buf.WriteString("You are a chosen of the gods.(BadMuthaFucker)\r\n")
	}

	// 16. Position (from C lines 1318-1340)
	buf.WriteString(positionText(p.Position))
	buf.WriteString("\r\n")

	// 17. Status conditions (from C lines 1342-1347)
	if p.Drunk > 10 {
		buf.WriteString("You are intoxicated.\r\n")
	}
	if p.Hunger == 0 {
		buf.WriteString("You are hungry.\r\n")
	}
	if p.Thirst == 0 {
		buf.WriteString("You are thirsty.\r\n")
	}

	// 18. Active affects (from C lines 1349-1369)
	if p.IsAffected(game.AffBlind) {
		buf.WriteString("You have been blinded!\r\n")
	}
	// Check PRF_SUMMONABLE flag on Player.Flags (PRF bit 48)
	if p.Flags&(1<<game.PrfSummonable) != 0 {
		buf.WriteString("You are summonable by other players.\r\n")
	}
	// AFF_WEREWOLF = bit 27 (src/structs.h).
	if p.IsAffected(game.AffWerewolf) {
		buf.WriteString("You're a lycanthrope!\r\n")
	}
	// AFF_VAMPIRE = bit 28 (src/structs.h).
	if p.IsAffected(game.AffVampire) {
		buf.WriteString("You're a vampire!\r\n")
	}
	// AFF_MOUNT — check via IsAffected or flags
	// Source: structs.h AFF_MOUNT bit
	if p.IsAffected(game.AffMount) {
		buf.WriteString("You're mounted.\r\n")
	}
	// AFF_FLESH_ALTER = bit 16
	if p.IsAffected(game.AffFleshAlter) {
		fmt.Fprintf(&buf, "Your hand is a %s!\r\n", fleshAlterWeapon(p.GetLevel()))
	}

	// 19. Spell affects list (from C lines 1371-1397)
	// C bit positions for filtering (structs.h):
	// AFF_SNEAK = 18, AFF_DODGE = 17, AFF_BERSERK = 20, AFF_ROBBED = 38
	// AFF_KUJI_KIRI = 24
	const (
		affSneakBit    uint64 = 1 << 18
		affDodgeBit    uint64 = 1 << 17
		affBerserkBit  uint64 = 1 << 20
		affRobbedBit   uint64 = 1 << 38
		affKujiKiriBit uint64 = 1 << 24
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

	// C's second score affect pass (act.informative.c:1416-1448) reports
	// active affect bits that are present without a matching ch->affected node.
	// These are normally equipment/status bits; spell-backed nodes belong in
	// the preceding list. The exclusion set is copied from the C condition.
	var equipmentAffects []string
	for bit, name := range scoreEquipmentAffectNames {
		if bit == 8 || bit == 25 || bit == game.AffWerewolf || bit == game.AffMount || bit == 28 || bit == 32 {
			continue
		}
		if !p.IsAffected(bit) || scoreAffectHasNode(p, bit) {
			continue
		}
		equipmentAffects = append(equipmentAffects, name)
	}
	if len(equipmentAffects) > 0 {
		buf.WriteString("\r\nEquipment spells affecting you:")
		for _, name := range equipmentAffects {
			buf.WriteString("\r\n" + name)
		}
		buf.WriteString("\r\n")
	}

	s.Send(buf.String())
	return nil
}

// scoreEquipmentAffectNames mirrors C's affected_names[] table in
// src/constants.c:646-686. The C score loop indexes this table by the C
// affect bit, so keeping the exact order and lowercase spelling matters.
var scoreEquipmentAffectNames = []string{
	"blind", "invisibility", "detect alignment", "detect invisibility",
	"detect magic", "sense life", "waterwalk", "sanctuary", "group", "curse",
	"infravision", "poison", "protection from evil", "protection from good",
	"sleep", "no track", "flesh alter", "dodge", "sneak", "hide", "berserk",
	"charm", "follow", "wimpy", "kuji-kiri", "cutthroat", "fly", "werewolf",
	"vampire", "mounted", "invulnerability", "flaming", "nothing", "haste",
	"slow", "dream", "waterbreathe", "metalskin", "robbed",
}

func scoreAffectHasNode(p *game.Player, bit int) bool {
	cMask := uint64(1) << uint(bit)
	engineMask := game.AffBitToEngineFlag[bit]
	for _, aff := range p.MasterAffects {
		if aff != nil && (aff.Bitvector&cMask != 0 || engineMask != 0 && aff.Bitvector&engineMask != 0) {
			return true
		}
	}
	for _, aff := range p.ActiveAffects {
		if aff != nil && (aff.Flags&cMask != 0 || engineMask != 0 && aff.Flags&engineMask != 0) {
			return true
		}
	}
	return false
}

// fleshAlterWeapon mirrors flesh_alter_weapon() in src/new_cmds.c:1836-1870.
func fleshAlterWeapon(level int) string {
	switch {
	case level <= 3:
		return "studded wooden club"
	case level <= 6:
		return "razor-sharp dagger"
	case level <= 9:
		return "steel-shafted axe"
	case level <= 12:
		return "studded steel mace"
	case level <= 15:
		return "battle flail"
	case level <= 18:
		return "steel-shafted battle axe"
	case level <= 21:
		return "double-headed battle axe"
	case level <= 24:
		return "studded morning-star"
	case level <= 27:
		return "gleaming broad sword"
	case level <= 29:
		return "gleaming long sword"
	default:
		return "gleaming scythe"
	}
}

func scoreManaLabel(class int) string {
	if class == game.ClassPsionic || class == game.ClassMystic {
		return "Mind/Psi"
	}
	return "Mana"
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

const (
	whoFormat        = "format: who [minlev[-maxlev]] [-n name] [-c classlist] [-s] [-o] [-q] \r\n"
	roomNoWhoFlagBit = 19 // ROOM_NO_WHO_ROOM in src/structs.h
)

type whoOptions struct {
	low       int
	high      int
	name      string
	classMask int64
	short     bool
	quest     bool
	outlaws   bool
	sameZone  bool
	sameRoom  bool
}

func parseWhoLevelRange(arg string, low, high int) (int, int, bool) {
	parts := strings.SplitN(arg, "-", 2)
	parsedLow, err := strconv.Atoi(parts[0])
	if err != nil {
		return low, high, false
	}
	low = parsedLow
	if len(parts) == 2 {
		parsedHigh, err := strconv.Atoi(parts[1])
		if err != nil {
			return low, high, false
		}
		high = parsedHigh
	}
	return low, high, true
}

func parseWhoArgs(args []string) (whoOptions, bool) {
	opts := whoOptions{low: 1, high: LVL_IMPL}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "" {
			continue
		}

		if arg[0] >= '0' && arg[0] <= '9' {
			low, high, ok := parseWhoLevelRange(arg, opts.low, opts.high)
			if !ok {
				return opts, false
			}
			opts.low, opts.high = low, high
			// Accept the brief's space-separated spelling as well as C's
			// min-max token. A second bare number is the upper bound.
			if !strings.Contains(arg, "-") && i+1 < len(args) {
				if upper, err := strconv.Atoi(args[i+1]); err == nil {
					opts.high = upper
					i++
				}
			}
			continue
		}

		if !strings.HasPrefix(arg, "-") || len(arg) < 2 {
			return opts, false
		}
		switch arg[1] {
		case 'o', 'k':
			opts.outlaws = true
		case 's':
			opts.short = true
		case 'q':
			opts.quest = true
		case 'r':
			opts.sameZone = true
		case 'z':
			opts.sameRoom = true
		case 'l':
			if i+1 >= len(args) {
				continue
			}
			i++
			low, high, ok := parseWhoLevelRange(args[i], opts.low, opts.high)
			if !ok {
				return opts, false
			}
			opts.low, opts.high = low, high
			if !strings.Contains(args[i], "-") && i+1 < len(args) {
				if upper, err := strconv.Atoi(args[i+1]); err == nil {
					opts.high = upper
					i++
				}
			}
		case 'n':
			if i+1 < len(args) {
				i++
				opts.name = args[i]
			}
		case 'c':
			if i+1 < len(args) {
				i++
				for _, classLetter := range strings.ToLower(args[i]) {
					opts.classMask |= game.FindClassBitvector(byte(classLetter)) // #nosec G115 -- class filter letters are ASCII; a truncated non-ASCII rune only fails the lookup, no security impact
				}
			}
		default:
			return opts, false
		}
	}
	return opts, true
}

func whoClassAbbrev(class int) string {
	if class >= 0 && class < len(game.ClassAbbrevs) {
		return game.ClassAbbrevs[class]
	}
	return "--"
}

func whoShortRank(level int) string {
	switch {
	case level >= LVL_IMPL:
		return "[ *IMP*  ]"
	case level >= LVL_GRGOD:
		return "[ GRGOD  ]"
	case level >= LVL_HIGOD:
		return "[ HIGOD  ]"
	case level >= LVL_LEGEND:
		return "[ LEGEND ]"
	case level >= LVL_GOD:
		return "[  GOD   ]"
	case level >= LVL_IMMORT+1:
		return "[ TITAN  ]"
	case level >= LVL_IMMORT:
		return "[ IMMORT ]"
	default:
		return ""
	}
}

func whoHasFlag(player *game.Player, bit int) bool {
	return player.GetFlags()&(1<<uint(bit)) != 0
}

func whoStatus(player *game.Player) string {
	var status strings.Builder
	flags := player.GetFlags()
	if flags&(1<<uint(game.PlrMailing)) != 0 {
		status.WriteString(" (mailing)")
	} else if flags&(1<<uint(game.PlrWriting)) != 0 {
		status.WriteString(" (writing)")
	}
	if flags&(1<<uint(game.PrfDeaf)) != 0 {
		status.WriteString(" (deaf)")
	}
	if flags&(1<<uint(game.PrfNotell)) != 0 {
		status.WriteString(" (notell)")
	}
	if flags&(1<<uint(game.PrfQuest)) != 0 {
		status.WriteString(" (quest)")
	}
	if flags&(1<<uint(game.PrfAFK)) != 0 {
		status.WriteString(" (AFK)")
	}
	if flags&(1<<uint(game.PrfInactive)) != 0 {
		status.WriteString(" (INACTIVE)")
	}
	if flags&(1<<uint(game.PlrIt)) != 0 {
		status.WriteString(" (IT)")
	}
	if flags&(1<<uint(game.PlrOutlaw)) != 0 {
		status.WriteString(" (OUTLAW)")
	}
	return status.String()
}

func whoTargetVisible(s *Session, target *game.Player, opts whoOptions) bool {
	viewer := s.player
	if viewer == nil || target == nil || !game.CanSee(viewer, target) {
		return false
	}
	level := target.GetLevel()
	if level < opts.low || level > opts.high {
		return false
	}
	if opts.quest && !whoHasFlag(target, game.PrfQuest) {
		return false
	}
	if opts.outlaws && !whoHasFlag(target, game.PlrOutlaw) {
		return false
	}
	if room, ok := s.manager.world.GetRoom(target.GetRoom()); ok && room != nil &&
		(room.HasFlag(roomNoWhoFlagBit) || roomFlagNamed(room.Flags, "no_who_room", "no_who")) &&
		viewer.GetLevel() < LVL_IMPL {
		return false
	}
	if opts.sameZone && s.manager.world.GetRoomZone(viewer.GetRoom()) != s.manager.world.GetRoomZone(target.GetRoom()) {
		return false
	}
	if opts.sameRoom && viewer.GetRoom() != target.GetRoom() {
		return false
	}
	class := target.GetClass()
	if opts.classMask != 0 && (class < 0 || opts.classMask&(1<<uint(class)) == 0) {
		return false
	}
	if opts.name != "" && !strings.EqualFold(target.GetName(), opts.name) && !strings.Contains(target.GetTitle(), opts.name) {
		return false
	}
	return true
}

func roomFlagNamed(flags []string, names ...string) bool {
	for _, flag := range flags {
		for _, name := range names {
			if strings.EqualFold(flag, name) {
				return true
			}
		}
	}
	return false
}

func whoColorEnabled(viewer *game.Player) bool {
	if viewer == nil {
		return false
	}
	flags := viewer.GetFlags()
	return flags&(1<<uint(game.PrfColor1)) != 0 || flags&(1<<uint(game.PrfColor2)) != 0
}

func renderWhoLong(viewer, player *game.Player) string {
	level := player.GetLevel()
	name, title := player.GetName(), player.GetTitle()
	var line string
	var color string
	switch {
	case level >= LVL_IMMORT:
		line = fmt.Sprintf("[ Wizard ] %s %s", name, title)
		if whoColorEnabled(viewer) {
			color = "\x1b[0;31m"
		}
	case whoHasFlag(player, game.PlrChosen):
		line = fmt.Sprintf("[ Chosen ] %s %s", name, title)
		if whoColorEnabled(viewer) {
			color = "\x1b[0;33m"
		}
	default:
		line = fmt.Sprintf("[ %2d  %s ] %s %s", level, whoClassAbbrev(player.GetClass()), name, title)
	}
	line += whoStatus(player)
	if color != "" {
		line = color + line + "\x1b[0m"
	}
	return line + "\r\n"
}

func renderWhoShort(viewer, player *game.Player) string {
	rank := whoShortRank(player.GetLevel())
	if rank == "" {
		rank = fmt.Sprintf("[ %2d  %s ]", player.GetLevel(), whoClassAbbrev(player.GetClass()))
	}
	line := fmt.Sprintf("%s %-12.12s", rank, player.GetName())
	if whoColorEnabled(viewer) {
		return "\x1b[33m" + line + "\x1b[0m"
	}
	return line
}

func cmdWho(s *Session, args []string) error {
	opts, ok := parseWhoArgs(args)
	if !ok {
		s.sendText(whoFormat)
		return nil
	}

	s.manager.mu.RLock()
	sessions := make([]*Session, 0, len(s.manager.sessions))
	for _, sess := range s.manager.sessions {
		sessions = append(sessions, sess)
	}
	s.manager.mu.RUnlock()
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].connectedAt.After(sessions[j].connectedAt)
	})

	var out strings.Builder
	out.WriteString("Players\r\n-------\r\n")
	count := 0
	lastShort := ""
	for _, sess := range sessions {
		if !whoTargetVisible(s, sess.player, opts) {
			continue
		}
		if opts.short {
			lastShort = renderWhoShort(s.player, sess.player)
			out.WriteString(lastShort)
			count++
			if count%4 == 0 {
				out.WriteString("\r\n")
			}
			continue
		}
		out.WriteString(renderWhoLong(s.player, sess.player))
		count++
	}
	if opts.short {
		// The C implementation builds the trailer in the same scratch buffer
		// as the final short-list cell. For a partial four-column row it sends
		// that buffer again, so the last cell is observably repeated.
		if count%4 != 0 {
			out.WriteString(lastShort)
			switch count {
			case 1:
				out.WriteString("\r\nOne character displayed.\r\n\r\n")
			default:
				fmt.Fprintf(&out, "\r\n%d characters displayed.\r\n\r\n", count)
			}
		}
		s.sendText(out.String())
		return nil
	}

	switch count {
	case 0:
		out.WriteString("\r\nNo-one at all!\r\n")
	case 1:
		out.WriteString("\r\nOne character displayed.\r\n")
	default:
		fmt.Fprintf(&out, "\r\n%d characters displayed.\r\n", count)
	}
	s.sendText(out.String())
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
// The critical mortal vnum exposure is closed by the LVL_IMMORT command gate;
// the optional immortal target-search branch mirrors C's one_argument path.
func cmdWhere(s *Session, args []string) error {
	s.manager.mu.RLock()
	sessions := make([]*Session, 0, len(s.manager.sessions))
	for _, sess := range s.manager.sessions {
		sessions = append(sessions, sess)
	}
	s.manager.mu.RUnlock()
	// C iterates descriptor_list, which prepends new connections — the most
	// recently connected player is listed first (same order cmdWhere's who
	// sibling uses).
	sort.SliceStable(sessions, func(i, j int) bool {
		return sessions[i].connectedAt.After(sessions[j].connectedAt)
	})

	// C's one_argument branch searches the visible character list by name
	// (act.informative.c:2283-2301); it does not interpret the argument as a
	// zone filter. Keep the optional argument's player-facing format separate
	// from the no-argument Players page.
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		query := strings.ToLower(strings.TrimSpace(args[0]))
		count := 0
		var out strings.Builder
		for _, sess := range sessions {
			if sess.player == nil || !strings.HasPrefix(strings.ToLower(sess.player.Name), query) {
				continue
			}
			room, ok := s.manager.world.GetRoom(sess.player.GetRoom())
			if !ok || !game.CanSee(s.player, sess.player) {
				continue
			}
			count++
			fmt.Fprintf(&out, "M%3d. %-25s - [%5d]%s\r\n", count, sess.player.Name, room.VNum, room.Name)
			if count >= 30 {
				break
			}
		}
		if count == 0 {
			s.sendText("Couldn't find any such thing.\r\n")
		} else {
			s.sendText(out.String())
		}
		return nil
	}

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
			old := sess.player.GetRoom()
			sess.player.SetRoom(s.player.GetRoom())
			s.sendText(fmt.Sprintf("%s materializes before you.", sess.player.Name))
			sess.sendText(fmt.Sprintf("You are summoned by %s.", s.player.Name))
			_ = old
			return nil
		}
	}
	s.sendText("No one by that name online.")
	return nil
}

// Help ANSI codes (CCGRN/CCRED/CCCYN/CCNRM, C_CMP mode — act.informative.c).
const (
	helpGreen  = "\x1b[32m"
	helpRed    = "\x1b[31m"
	helpCyan   = "\x1b[36m"
	helpNormal = "\x1b[0m"
)

// helpSeparator is the red 75-dash rule do_help prints under the topic header.
// C sends two concatenated string chunks that total 75 dashes (a leading space
// + 74 dashes in the source), act.informative.c:1643-1645.
const helpSeparator = helpRed +
	" ---------------------------------------------------------------------------" +
	helpNormal

// cmdHelp serves ONLY what the help files serve (R4). It is a faithful port of
// C do_help (act.informative.c:1566-1674): no-arg page_strings the help screen;
// otherwise a prefix lookup (game.SearchHelp) into the keyword-sorted table,
// with wizonly entries hidden from mortals and misses reporting the exact C
// line. The invented helpTopics map, registry-description fallback, and fuzzy
// "did you mean" surface are all removed — a command with no help entry is, per
// C, "There is no help on:".
func cmdHelp(s *Session, args []string) error {
	// no argument → page_string the help screen (C: page_string(ch->desc, help, 0)).
	if len(args) == 0 {
		PageString(s, s.manager.world.HelpScreen)
		return nil
	}

	table := s.manager.world.HelpTable
	if len(table) == 0 {
		// C: "No help available.\r\n" when help_table is empty.
		s.sendText("No help available.\r\n")
		return nil
	}

	argument := strings.Join(args, " ")
	entry := game.SearchHelp(table, argument)
	if entry == nil {
		// C: "There is no help on: %s\r\n" + mudlog + append to misc/help file.
		// The mudlog and the misc/help usage file are server-side only (not
		// player-facing); the file write is intentionally skipped here.
		s.sendText(fmt.Sprintf("There is no help on: %s\r\n", argument))
		return nil
	}

	// Mortals cannot see wizonly entries — existence is hidden, same as a miss
	// (C: GET_LEVEL(ch) < LVL_IMMORT && strstr(entry, "wizonly")).
	if s.player != nil && s.player.GetLevel() < LVL_IMMORT &&
		strings.Contains(entry.Entry, "wizonly") {
		s.sendText(fmt.Sprintf("There is no help on: %s\r\n", argument))
		return nil
	}

	// Header: green "\r\n[ " cyan TOPIC-UPPERCASED green " ]\r\n" normal
	// (C uppercases the whole keyword string).
	topic := strings.ToUpper(entry.Keyword)
	header := helpGreen + "\r\n[ " + helpCyan + topic + helpGreen + " ]\r\n" + helpNormal
	// Header + separator go as ONE message: the transport appends a line
	// break per message that doesn't end in a newline (both header and
	// separator end in ANSI-normal codes), so per-message emission injected
	// blank lines C doesn't have. One message = C's contiguous stc stream.
	s.sendText(header + helpSeparator)

	// Body: after the keyword line. C's pointer stops AT the first '\n' and
	// that newline terminates the dash line; here the message framing supplies
	// that break, so the body starts past the newline.
	body := entry.Entry
	if i := strings.IndexByte(body, '\n'); i >= 0 {
		body = body[i+1:]
	}
	PageString(s, body)
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
			// C do_whois (new_cmds.c:1418): "[%2d %s] %s %s" — level, class
			// abbrev, name, AND title. The title was previously dropped.
			s.sendText(fmt.Sprintf("[%2d %s] %s %s\r\n", p.Level, classAbbr, p.Name, p.Title))
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
		// TODO(port): PlayerRecord does not persist Title (C chdata.title), so
		// the offline path omits the trailing title field C prints here. The
		// online path above is faithful; closing this gap needs a schema change.
		s.sendText(fmt.Sprintf("[%2d %s] %s\r\n", rec.Level, classAbbr, rec.Name))
		return nil
	}

	s.sendText("There is no such player.\r\n")
	return nil
}

// directions maps abbreviated direction names to full names.
