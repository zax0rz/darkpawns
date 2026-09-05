package session

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// setField is the ordered table from src/act.wizard.c:2537-2596. The order is
// observable: do_set accepts a prefix and the first matching row wins.
type setField struct {
	name  string
	level int
	pcnpc uint8
	typ   setFieldType
}

type setFieldType uint8

const (
	setMisc setFieldType = iota
	setNumber
	setBinary
)

const (
	setPC  = 1
	setNPC = 2
)

var setFields = []setField{
	{"brief", LVL_GOD, setPC, setBinary},
	{"invstart", LVL_GOD, setPC, setBinary},
	{"title", LVL_GOD, setPC, setMisc},
	{"nosummon", LVL_GRGOD, setPC, setBinary},
	{"maxhit", LVL_GRGOD, setPC | setNPC, setNumber},
	{"maxmana", LVL_GRGOD, setPC | setNPC, setNumber},
	{"maxmove", LVL_GRGOD, setPC | setNPC, setNumber},
	{"hit", LVL_GRGOD, setPC | setNPC, setNumber},
	{"mana", LVL_GRGOD, setPC | setNPC, setNumber},
	{"move", LVL_GRGOD, setPC | setNPC, setNumber},
	{"align", LVL_GOD, setPC | setNPC, setNumber},
	{"str", LVL_GRGOD, setPC | setNPC, setNumber},
	{"stradd", LVL_GRGOD, setPC | setNPC, setNumber},
	{"int", LVL_GRGOD, setPC | setNPC, setNumber},
	{"wis", LVL_GRGOD, setPC | setNPC, setNumber},
	{"dex", LVL_GRGOD, setPC | setNPC, setNumber},
	{"con", LVL_GRGOD, setPC | setNPC, setNumber},
	{"sex", LVL_GRGOD, setPC | setNPC, setMisc},
	{"ac", LVL_GRGOD, setPC | setNPC, setNumber},
	{"gold", LVL_GOD, setPC | setNPC, setNumber},
	{"bank", LVL_GOD, setPC, setNumber},
	{"exp", LVL_GRGOD, setPC | setNPC, setNumber},
	{"hitroll", LVL_GRGOD, setPC | setNPC, setNumber},
	{"damroll", LVL_GRGOD, setPC | setNPC, setNumber},
	{"invis", LVL_IMPL, setPC, setNumber},
	{"nohassle", LVL_GRGOD, setPC, setBinary},
	{"frozen", LVL_GRGOD, setPC, setBinary},
	{"practices", LVL_GRGOD, setPC, setNumber},
	{"lessons", LVL_GRGOD, setPC, setNumber},
	{"drunk", LVL_GRGOD, setPC | setNPC, setMisc},
	{"hunger", LVL_GRGOD, setPC | setNPC, setMisc},
	{"thirst", LVL_GRGOD, setPC | setNPC, setMisc},
	{"outlaw", LVL_GOD, setPC, setBinary},
	{"name", LVL_GRGOD, setPC, setMisc},
	{"level", LVL_GRGOD, setPC | setNPC, setNumber},
	{"room", LVL_IMPL, setPC | setNPC, setNumber},
	{"roomflag", LVL_GRGOD, setPC, setBinary},
	{"siteok", LVL_GRGOD, setPC, setBinary},
	{"deleted", LVL_GRGOD, setPC, setBinary},
	{"class", LVL_GRGOD, setPC | setNPC, setMisc},
	{"nowizlist", LVL_GOD, setPC, setBinary},
	{"quest", LVL_GOD, setPC, setBinary},
	{"loadroom", LVL_GRGOD, setPC, setMisc},
	{"color", LVL_GOD, setPC, setBinary},
	{"idnum", LVL_IMPL - 1, setPC, setNumber},
	{"passwd", LVL_IMPL - 1, setPC, setMisc},
	{"nodelete", LVL_GOD, setPC, setBinary},
	{"cha", LVL_GRGOD, setPC | setNPC, setNumber},
	{"olc", LVL_GOD + 1, setPC, setNumber},
	{"race", LVL_GOD, setPC, setMisc},
	{"kills", LVL_GRGOD, setPC | setNPC, setNumber},
	{"pks", LVL_GRGOD, setPC | setNPC, setNumber},
	{"deaths", LVL_GRGOD, setPC | setNPC, setNumber},
	{"home", LVL_GRGOD, setPC, setNumber},
	{"tattoo", LVL_GRGOD, setPC, setNumber},
	{"origcon", LVL_GRGOD, setPC, setNumber},
	{"chosen", LVL_GRGOD, setPC, setBinary},
	{"clan", LVL_GRGOD, setPC, setNumber},
	{"played", LVL_IMPL, setPC, setNumber},
}

type setTarget struct {
	player  *game.Player
	mob     *game.MobInstance
	session *Session
	file    bool
}

// cmdSet is the tokenized entry point retained for direct callers and tests.
// The transport path calls cmdSetText so title/name values retain C's original
// post-field spacing.
func cmdSet(s *Session, args []string) error {
	return cmdSetText(s, args, "")
}

// cmdSetText ports do_set() (src/act.wizard.c:2523-3069). In particular, the
// command itself is available at LVL_GOD (interpreter.c:682); authority is
// checked again against the selected C field row below.
func cmdSetText(s *Session, args []string, rawArgs string) error {
	name, fieldText, value, isFile, playerOnly := parseSetArgs(args, rawArgs)
	if name == "" || fieldText == "" {
		s.Send("Usage: set <victim> <field> <value>\r\n")
		return nil
	}

	target, ok := resolveSetTarget(s, name, playerOnly)
	if isFile {
		var err error
		target.player, err = game.LoadPlayer(name)
		target.file = true
		if err != nil {
			s.Send("There is no such player.\r\n")
			return nil
		}
		ok = true
	}
	if !ok {
		if strings.EqualFold(fieldText, "player") {
			s.Send("There is no such player.\r\n")
		} else {
			s.Send("There is no such creature.\r\n")
		}
		return nil
	}

	fieldName := fieldText
	field, found := findSetField(fieldName)
	if !found {
		field = setField{name: "", typ: setMisc, pcnpc: setPC | setNPC}
	}

	actorLevel := getEffectiveLevel(s)
	// C's safety check runs before field lookup and before the field-specific
	// privilege/type checks (act.wizard.c:2612-2621).
	if target.player != nil && actorLevel != LVL_IMPL && target.player != s.player && target.player.GetLevel() >= actorLevel {
		s.Send("Maybe that's not such a great idea...\r\n")
		return nil
	}
	if actorLevel < field.level {
		s.Send("You are not godly enough for that!\r\n")
		return nil
	}
	isNPC := target.mob != nil
	if isNPC && field.pcnpc&setNPC == 0 {
		s.Send("You can't do that to a beast!\r\n")
		return nil
	}
	if !isNPC && field.pcnpc&setPC == 0 {
		s.Send("That can only be done to a beast!\r\n")
		return nil
	}

	on, binaryOK := false, true
	switch field.typ {
	case setBinary:
		switch value {
		case "on", "yes":
			on = true
		case "off", "no":
		default:
			binaryOK = false
		}
		if !binaryOK {
			s.Send("Value must be on or off.\r\n")
			return nil
		}
	}

	valueInt := cAtoi(value)
	if field.typ == setNumber {
		valueInt = clampSetValue(field.name, valueInt, target.player, target.mob)
	}
	ack, changed, early := applySetField(s, target, field, value, valueInt, on)
	if early {
		if ack != "" {
			s.Send(ack)
		}
		return nil
	}
	if !changed && !found {
		s.Send("Can't set that!\r\n")
		return nil
	}

	switch field.typ {
	case setBinary:
		// C intentionally inverts the acknowledgement for the negative
		// nosummon field (act.wizard.c:2674).
		ackOn := on
		if field.name == "nosummon" {
			ackOn = !on
		}
		ack = fmt.Sprintf("%s %s for %s.\r\n", setCap(field.name), onOff(ackOn), setTargetName(target))
	case setNumber:
		ack = fmt.Sprintf("%s's %s set to %d.\r\n", setTargetName(target), field.name, valueInt)
	default:
		ack += "\r\n"
	}
	// C calls CAP(buf) immediately before send_to_char for every successful
	// field, so a mob's lower-case short description is capitalized too.
	ack = setCap(ack)
	s.Send(ack)

	if target.file && target.player != nil {
		if err := game.SavePlayer(target.player); err != nil {
			slog.Error("set file: save failed", "target", target.player.Name, "error", err)
		} else {
			s.Send("Saved in file.\r\n")
		}
	}
	slog.Warn("wizard set", "by", s.player.Name, "target", setTargetName(target), "field", field.name, "value", value)
	return nil
}

func parseSetArgs(args []string, rawArgs string) (name, field, value string, isFile, playerOnly bool) {
	remainder := rawArgs
	if remainder == "" {
		remainder = strings.Join(args, " ")
	}
	name, remainder = setHalfChop(remainder)
	if name == "file" {
		isFile = true
		name, remainder = setHalfChop(remainder)
	} else if strings.EqualFold(name, "player") {
		playerOnly = true
		name, remainder = setHalfChop(remainder)
	} else if strings.EqualFold(name, "mob") {
		name, remainder = setHalfChop(remainder)
	}
	field, value = setHalfChop(remainder)
	return name, field, value, isFile, playerOnly
}

func setHalfChop(input string) (first, remainder string) {
	input = strings.TrimLeft(input, cCommandWhitespace)
	if input == "" {
		return "", ""
	}
	idx := strings.IndexAny(input, cCommandWhitespace)
	if idx < 0 {
		return input, ""
	}
	return input[:idx], strings.TrimLeft(input[idx+1:], cCommandWhitespace)
}

func findSetField(input string) (setField, bool) {
	for _, field := range setFields {
		if len(input) <= len(field.name) && strings.HasPrefix(field.name, input) {
			return field, true
		}
	}
	return setField{}, false
}

func resolveSetTarget(s *Session, name string, playerOnly bool) (setTarget, bool) {
	if s == nil || s.player == nil || s.manager == nil {
		return setTarget{}, false
	}
	if playerOnly {
		for _, p := range s.manager.world.GetAllPlayers() {
			if strings.EqualFold(p.GetName(), name) {
				return setTarget{player: p, session: findSessionForPlayer(s.manager, p)}, true
			}
		}
		if sess := findSessionByName(s.manager, name); sess != nil && sess.player != nil {
			return setTarget{player: sess.player, session: sess}, true
		}
		return setTarget{}, false
	}
	if s.manager.world != nil {
		if resolved, ok := s.manager.world.ResolveCharWorld(s.player, name); ok {
			if resolved.Player != nil {
				return setTarget{player: resolved.Player, session: findSessionForPlayer(s.manager, resolved.Player)}, true
			}
			if resolved.Mob != nil {
				return setTarget{mob: resolved.Mob}, true
			}
		}
	}
	// Unit fixtures construct sessions without registering their players in the
	// world. Keep this fallback exact-name only; live prefix/visibility behavior
	// remains owned by ResolveCharWorld above.
	if sess := findSessionByName(s.manager, name); sess != nil && sess.player != nil {
		return setTarget{player: sess.player, session: sess}, true
	}
	return setTarget{}, false
}

func findSessionForPlayer(m *Manager, p *game.Player) *Session {
	if m == nil || p == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sess := range m.sessions {
		if sess != nil && sess.player == p {
			return sess
		}
	}
	return nil
}

func setTargetName(target setTarget) string {
	if target.player != nil {
		return target.player.GetName()
	}
	if target.mob != nil {
		return target.mob.GetName()
	}
	return ""
}

func setCap(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func cAtoi(input string) int {
	input = strings.TrimLeft(input, cCommandWhitespace)
	if input == "" {
		return 0
	}
	start := 0
	if input[0] == '+' || input[0] == '-' {
		start = 1
	}
	end := start
	for end < len(input) && input[end] >= '0' && input[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	v, err := strconv.Atoi(input[:end])
	if err != nil {
		if input[0] == '-' {
			return -int(^uint(0)>>1) - 1
		}
		return int(^uint(0) >> 1)
	}
	return v
}

func cIsNumber(input string) bool {
	if input == "" {
		return true
	}
	for i := 0; i < len(input); i++ {
		if input[i] < '0' || input[i] > '9' {
			return false
		}
	}
	return true
}

func setCPlayerFlag(p *game.Player, bit int, enabled bool) {
	// The port has both the legacy PLR storage and the typed PLR accessor. Keep
	// them synchronized while the save/runtime boundary is being consolidated.
	p.SetPlrFlag(bit, enabled)
	p.SetPLRFlag(bit)
	if !enabled {
		p.ClearPLRFlag(bit)
	}
}

func setCPrfFlag(p *game.Player, bit int, enabled bool) {
	p.SetPlrFlag(bit, enabled)
}

func applySetField(s *Session, target setTarget, field setField, value string, valueInt int, on bool) (ack string, changed, early bool) {
	p, m := target.player, target.mob
	name := setTargetName(target)
	ack = "Okay."

	if !changed && field.name == "" {
		return "Can't set that!", false, false
	}
	if field.typ == setNumber {
		// C's RANGE macro mutates the local `value`, and the common final
		// acknowledgement prints that clamped value.
		valueInt = clampSetValue(field.name, valueInt, p, m)
	}

	switch field.name {
	case "brief":
		setCPrfFlag(p, game.PrfBrief, on)
	case "invstart":
		setCPlayerFlag(p, game.PlrInvstart, on)
	case "title":
		game.SetTitle(p, value)
		ack = fmt.Sprintf("%s's title is now: %s", name, p.GetTitle())
	case "nosummon":
		setCPrfFlag(p, game.PrfSummonable, on)
	case "maxhit":
		if p != nil {
			p.SetMaxHP(valueInt)
		} else {
			m.MaxHP = valueInt
		}
	case "maxmana":
		if p != nil {
			p.SetMaxMana(valueInt)
		} else {
			m.SetMaxMana(valueInt)
		}
	case "maxmove":
		if p != nil {
			p.SetMaxMove(valueInt)
		} else {
			m.MaxMove = valueInt
		}
	case "hit":
		if p != nil {
			p.SetHP(valueInt)
		} else {
			m.SetHealth(valueInt)
		}
	case "mana":
		if p != nil {
			p.SetMana(valueInt)
		} else {
			m.SetMana(valueInt)
		}
	case "move":
		if p != nil {
			p.SetMove(valueInt)
		} else {
			m.SetMove(valueInt)
		}
	case "align":
		if p != nil {
			p.SetAlignment(valueInt)
		} else {
			m.Runtime.AlignmentOverride = &valueInt
		}
	case "str", "int", "wis", "dex", "con", "cha":
		if p != nil {
			p.SetStat(strings.ToUpper(field.name), valueInt)
			if field.name == "str" {
				p.Strength = valueInt
				p.SetStat("StrAdd", 0)
			}
		} else {
			setMobStat(m, field.name, valueInt)
		}
	case "stradd":
		if p != nil {
			p.SetStat("StrAdd", valueInt)
			if valueInt > 0 {
				p.SetStat("STR", 18)
				p.Strength = 18
			}
		} else {
			m.Runtime.StrAddOverride = &valueInt
			if valueInt > 0 {
				m.Str = 18
			}
		}
	case "sex":
		sex, ok := parseSetSex(value)
		if !ok {
			return "Must be 'male', 'female', or 'neutral'.\r\n", false, true
		}
		if p != nil {
			p.SetSex(sex)
		} else {
			// Parser mob data uses the C encoding (neutral=0, male=1,
			// female=2); preserve that at the oracle boundary.
			m.Runtime.SexOverride = &sex
		}
	case "ac":
		if p != nil {
			p.SetAC(valueInt)
		} else {
			m.Runtime.ACOverride = &valueInt
		}
	case "gold":
		if p != nil {
			p.SetGold(valueInt)
		} else {
			m.SetGold(valueInt)
		}
	case "bank":
		p.SetBankGold(valueInt)
	case "exp":
		if p != nil {
			p.SetExp(valueInt)
		}
	case "hitroll":
		if p != nil {
			p.SetHitroll(valueInt)
		} else {
			m.Runtime.HitrollOverride = &valueInt
		}
	case "damroll":
		if p != nil {
			p.SetDamroll(valueInt)
		} else {
			m.SetDamroll(valueInt)
		}
	case "invis":
		if getEffectiveLevel(s) < LVL_IMPL && (p == nil || p != s.player) {
			return "You aren't godly enough for that!\r\n", false, true
		}
		p.SetInvisLevel(valueInt)
	case "nohassle":
		if getEffectiveLevel(s) < LVL_IMPL && p != s.player {
			return "You aren't godly enough for that!\r\n", false, true
		}
		setCPrfFlag(p, game.PrfNohassle, on)
	case "frozen":
		if p == s.player {
			return "Better not -- could be a long winter!\r\n", false, true
		}
		setCPlayerFlag(p, game.PlrFrozen, on)
	case "practices", "lessons":
		p.SetPractices(valueInt)
	case "drunk", "hunger", "thirst":
		cond := map[string]int{"drunk": game.CondDrunk, "hunger": game.CondFull, "thirst": game.CondThirst}[field.name]
		if value == "off" {
			p.SetCondition(cond, -1)
			ack = fmt.Sprintf("%s's %s now off.", name, field.name)
		} else if !cIsNumber(value) {
			return "Must be 'off' or a value from 0 to 48.\r\n", false, true
		} else {
			valueInt = clamp(valueInt, 0, 48)
			p.SetCondition(cond, valueInt)
			ack = fmt.Sprintf("%s's %s set to %d.", name, field.name, valueInt)
		}
	case "outlaw":
		setCPlayerFlag(p, game.PlrOutlaw, on)
	case "name":
		old := p.Name
		p.Name = value
		if target.session != nil {
			target.session.playerName = value
		}
		ack = "Okay."
		slog.Warn("set renamed live player", "old_name", old, "new_name", value)
	case "level":
		if valueInt >= getEffectiveLevel(s) || valueInt > LVL_IMPL {
			return "You can't do that!\r\n", false, true
		}
		valueInt = clamp(valueInt, 0, LVL_IMPL)
		if p != nil {
			p.SetLevel(valueInt)
		} else {
			m.SetLevel(valueInt)
		}
	case "room":
		if s.manager.world == nil || s.manager.world.GetRoomInWorld(valueInt) == nil {
			return "No room exists with that number.\r\n", false, true
		}
		var err error
		if p != nil {
			err = s.manager.world.PlayerTransfer(p, valueInt)
		} else {
			err = s.manager.world.MobTransfer(m, valueInt)
		}
		if err != nil {
			return "No room exists with that number.\r\n", false, true
		}
	case "roomflag":
		setCPrfFlag(p, game.PrfRoomFlags, on)
	case "siteok":
		setCPlayerFlag(p, game.PlrSiteok, on)
	case "deleted":
		setCPlayerFlag(p, game.PlrDeleted, on)
	case "class":
		class, ok := parseSetClass(value)
		if !ok {
			return "That is not a class.\r\n", false, true
		}
		if p != nil {
			p.SetClass(class)
		} else {
			m.Runtime.ClassOverride = &class
		}
	case "nowizlist":
		setCPlayerFlag(p, game.PlrNowizlist, on)
	case "quest":
		setCPrfFlag(p, game.PrfQuest, on)
	case "loadroom":
		if value == "off" {
			setCPlayerFlag(p, game.PlrLoadroom, false)
		} else if !cIsNumber(value) {
			ack = "Must be 'off' or a room's virtual number.\r\n"
		} else if s.manager.world.GetRoomInWorld(valueInt) == nil {
			ack = "That room does not exist!"
		} else {
			setCPlayerFlag(p, game.PlrLoadroom, true)
			p.SetLoadRoom(valueInt)
			ack = fmt.Sprintf("%s will enter at room #%d.", name, valueInt)
		}
	case "color":
		setCPrfFlag(p, game.PrfColor1, on)
		setCPrfFlag(p, game.PrfColor2, on)
	case "idnum":
		if s.player.GetID() != 1 || m == nil {
			return "", false, true
		}
		m.ID = valueInt
	case "passwd":
		if !target.file {
			return "You must use set file with this command.\r\nThe player *must* not be logged in when this command is run.\r\n", false, true
		}
		return "Assuming the player is not logged in, this will not take effect if they are.\r\n", false, true
	case "nodelete":
		setCPlayerFlag(p, game.PlrNODELETE, on)
	case "olc":
		if target.session != nil {
			target.session.olcZone = valueInt
		}
	case "race":
		race, ok := parseSetRace(value)
		if !ok {
			return "That is not a race.\r\n", false, true
		}
		if p != nil {
			p.Race = race
		}
	case "kills":
		if p != nil {
			p.Kills = valueInt
		}
	case "pks":
		if p != nil {
			p.PKs = valueInt
		}
	case "deaths":
		if p != nil {
			p.Deaths = valueInt
		}
	case "home":
		p.Hometown = valueInt
	case "tattoo":
		if target.session != nil {
			tattooAf(target.session, false)
		}
		p.Tattoo = valueInt
		if target.session != nil {
			tattooAf(target.session, true)
		}
	case "origcon":
		p.SetOrigCon(valueInt)
	case "chosen":
		setCPlayerFlag(p, game.PlrChosen, on)
	case "clan":
		p.ClanID = valueInt
		if valueInt == 0 {
			p.ClanRank = 0
		}
	case "played":
		s.player.PlayedDuration = int64(valueInt)
	default:
		return "Can't set that!", false, false
	}

	if field.typ == setNumber {
		return "", true, false
	}
	return ack, true, false
}

func clampSetValue(field string, value int, p *game.Player, m *game.MobInstance) int {
	level := 0
	maxHP, maxMana, maxMove := 0, 0, 0
	if p != nil {
		level = p.GetLevel()
		maxHP, maxMana, maxMove = p.MaxHealth, p.MaxMana, p.MaxMove
	} else if m != nil {
		level = m.GetLevel()
		maxHP, maxMana, maxMove = m.MaxHP, m.MaxMana, m.MaxMove
	}
	switch field {
	case "maxhit", "maxmana", "maxmove":
		return clamp(value, 1, 5000)
	case "hit":
		return clamp(value, -9, maxHP)
	case "mana":
		return clamp(value, 0, maxMana)
	case "move":
		return clamp(value, 0, maxMove)
	case "align":
		return clamp(value, -1000, 1000)
	case "str", "int", "wis", "dex", "con", "cha":
		max := 18
		if m != nil || level >= LVL_GRGOD {
			max = 25
		}
		return clamp(value, 0, max)
	case "stradd":
		return clamp(value, 0, 100)
	case "ac":
		return clamp(value, -200, 100)
	case "gold", "bank":
		return clamp(value, 0, 100000000)
	case "exp":
		return clamp(value, 0, 50000000)
	case "hitroll", "damroll":
		return clamp(value, -20, 200)
	case "invis":
		return clamp(value, 0, level)
	case "practices", "lessons":
		return clamp(value, 0, 100)
	case "level":
		return value
	case "kills", "pks", "deaths":
		return clamp(value, -100, 65534)
	case "tattoo":
		return clamp(value, 0, TatOwl)
	case "origcon":
		return clamp(value, 1, 18)
	default:
		return value
	}
}

func setMobStat(m *game.MobInstance, field string, value int) {
	switch field {
	case "str":
		m.Str = value
	case "int":
		m.Intel = value
	case "wis":
		m.Wis = value
	case "dex":
		m.Dex = value
	case "con":
		m.Con = value
	case "cha":
		m.Cha = value
	}
}

func parseSetSex(value string) (int, bool) {
	switch strings.ToLower(value) {
	case "male":
		return game.SexMale, true
	case "female":
		return game.SexFemale, true
	case "neutral":
		return game.SexNeutral, true
	default:
		return 0, false
	}
}

func parseSetClass(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	switch strings.ToLower(value[:1]) {
	case "m":
		return game.ClassMageUser, true
	case "c":
		return game.ClassCleric, true
	case "w":
		return game.ClassWarrior, true
	case "t":
		return game.ClassThief, true
	case "a":
		return game.ClassAvatar, true
	case "v":
		return game.ClassMagus, true
	case "s":
		return game.ClassAssassin, true
	case "p":
		return game.ClassPaladin, true
	case "n":
		return game.ClassNinja, true
	case "i":
		return game.ClassPsionic, true
	case "r":
		return game.ClassRanger, true
	case "y":
		return game.ClassMystic, true
	default:
		return 0, false
	}
}

func parseSetRace(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	switch strings.ToLower(value[:1]) {
	case "h":
		return game.RaceHuman, true
	case "e":
		return game.RaceElf, true
	case "d":
		return game.RaceDwarf, true
	case "k":
		return game.RaceKender, true
	case "m":
		return game.RaceMinotaur, true
	case "r":
		return game.RaceRakshasa, true
	case "s":
		return game.RaceSsaur, true
	default:
		return 0, false
	}
}
