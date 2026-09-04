package session

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdStat(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Stats on who or what?\r\n")
		return nil
	}
	selector := strings.ToLower(args[0])
	target := strings.Join(args, " ")
	if selector == "room" {
		s.sendStatRoom()
		return nil
	}
	switch selector {
	case "mob":
		if len(args) == 1 {
			s.Send("Stats on which mobile?\r\n")
			return nil
		}
		if target, ok := s.manager.world.ResolveCharWorld(s.player, args[1]); ok && target.Combatant != nil && target.Combatant.IsNPC() {
			s.sendStatMob(target.Combatant.(*game.MobInstance))
			return nil
		}
		s.Send("No such mobile around.\r\n")
		return nil
	case "player":
		if len(args) == 1 {
			s.Send("Stats on which player?\r\n")
			return nil
		}
		if sess := findSessionByName(s.manager, args[1]); sess != nil && sess.player != nil {
			s.sendStatPlayer(sess.player)
			return nil
		}
		s.Send("No such player around.\r\n")
		return nil
	case "file":
		if len(args) == 1 {
			s.Send("Stats on which player?\r\n")
			return nil
		}
		if !game.PlayerSaveExists(args[1]) {
			s.Send("There is no such player.\r\n")
			return nil
		}
		player, err := game.LoadPlayer(args[1])
		if err != nil {
			s.Send("There is no such player.\r\n")
			return nil
		}
		s.sendStatPlayerFile(player)
		return nil
	case "obj", "object":
		if len(args) == 1 {
			s.Send("Stats on which object?\r\n")
			return nil
		}
		s.sendStatObject(args[1])
		return nil
	}
	if object, ok := s.manager.world.ResolveObjectWorld(s.player, args[0]); ok {
		s.sendStatObjectInstance(object)
		return nil
	}
	if sess := findSessionByName(s.manager, target); sess != nil && sess.player != nil {
		s.sendStatPlayer(sess.player)
		return nil
	}
	s.Send("Nothing around by that name.\r\n")
	return nil
}

func (s *Session) sendStatMob(mob *game.MobInstance) {
	if mob == nil || mob.Prototype == nil {
		return
	}
	proto := mob.Prototype
	rnum := -1
	if parsed := s.manager.world.GetParsedWorld(); parsed != nil {
		for i := range parsed.Mobs {
			if parsed.Mobs[i].VNum == proto.VNum {
				rnum = i
				break
			}
		}
	}

	s.sendStatMobLine(fmt.Sprintf("%s MOB '%s'  IDNum: [%5d], In room [%5d]",
		wizardSexName(mob.GetSex()), mob.GetName(), 0, mob.GetRoomVNum()))
	s.sendStatMobLine(fmt.Sprintf("Alias: %s, VNum: [%5d], RNum: [%5d]", proto.Keywords, proto.VNum, rnum))
	longDesc := proto.LongDesc
	if longDesc == "" {
		longDesc = "<None>"
	}
	s.sendStatMobLine("L-Des: " + longDesc)
	s.sendStatMobLine(fmt.Sprintf("Monster Class: Normal, Lev: [%2d], XP: [%7d], Align: [%4d]",
		mob.GetLevel(), proto.Exp, proto.Alignment))
	race := proto.Race
	if proto.RaceStr == "" {
		// The C parser's absent Race: extra field means RACE_OTHER, which is
		// entry 16 in mob_races[]. The Go parser keeps its internal default at
		// 7 for compatibility with the broader mob model.
		race = 16
	}
	s.sendStatMobLine(fmt.Sprintf("Race: [%d] %s", race, wizardMobRaceName(race)))
	s.sendStatMobLine("*************-------------*************-------------*************")
	s.sendStatMobLine(fmt.Sprintf("Coins: [%9d], Bank: [%9d] (Total: %d)", mob.GetGold(), 0, mob.GetGold()))
	s.sendStatMobLine(fmt.Sprintf("AC: [%d/10], Hitroll: [%2d], Damroll: [%2d], Saving throws: [0/0/0/0/0]",
		mob.GetAC(), mob.GetHitroll(), proto.Damage.Plus))
	s.sendStatMobLine(fmt.Sprintf("Pos: %s, Fighting: %s, Attack type: hit",
		wizardPositionName(mob.GetPosition()), wizardFightingName(mob.GetFighting())))
	s.sendStatMobLine(fmt.Sprintf("Default position: %s, Idle Timer (in tics) [%d]",
		wizardPositionName(proto.DefaultPos), 0))
	s.sendStatMobLine("NPC flags: " + wizardMobFlags(mob.GetMobFlags()))
	s.sendStatMobLine(fmt.Sprintf("Mob Spec-Proc: None, NPC Bare Hand Dam: %dd%d", proto.Damage.Num, proto.Damage.Sides))
	s.sendStatMobLine(fmt.Sprintf("Items in: inventory: %d, eq: %d", len(mob.Inventory), len(mob.Equipment)))
	s.sendStatMobLine("Master is: <none>, Followers are:")
	s.sendStatMobLine("AFF: " + wizardAffectFlags(mob.GetAffects()))
}

func (s *Session) sendStatMobLine(line string) {
	s.Send(line + "\r\n")
}

func (s *Session) sendStatPlayerFile(player *game.Player) {
	s.sendStatPlayerReport(player, -1, false, true)
}

func (s *Session) sendStatPlayerReport(p *game.Player, room int, connected, file bool) {
	if p == nil {
		return
	}

	flags := p.GetFlags()
	id := p.GetID() + 1
	if file {
		// The temporary C char_data is populated from the saved player record.
		// Go's disk snapshot may predate the runtime ID field, so prefer the
		// matching live session's C-facing ID when that session is available.
		if live := findSessionByName(s.manager, p.GetName()); live != nil && live.player != nil {
			id = live.player.GetID() + 1
		}
	}
	s.Send(fmt.Sprintf("%s PC '%s'  IDNum: [%5d], In room [%5d]\r\n",
		wizardSexName(p.GetSex()), p.GetName(), id, room))
	title := p.GetTitle()
	if title == "" {
		title = "<None>"
	}
	s.Send("Title: " + title + "\r\n")
	description := p.GetDescription()
	if description == "" {
		description = "<None>"
	}
	s.Send("L-Des: " + description + "\r\n")
	s.Send(fmt.Sprintf("Class: %s, Lev: [%2d], XP: [%7d], Align: [%4d]\r\n",
		wizardClassName(p.GetClass()), p.GetLevel(), p.GetExp(), p.GetAlignment()))
	pt := game.PlayingTime(p.ConnectedAt, p.PlayedDuration)
	age := game.Age(p.Birth).Year
	if file && p.Birth == 0 {
		age = 17
	}
	s.Send(fmt.Sprintf("Played [%dd %dh], Age [%d]\r\n", pt.Day, pt.Hours, age))
	if p.GetLastDeath() == 0 {
		s.Send("Last Death: [NONE]\r\n")
	}
	s.Send("*************-------------*************-------------*************\r\n")
	s.Send(fmt.Sprintf("Kills: [%9d], PKills: [%9d], Deaths: [%9d]\r\n", p.Kills, p.PKs, p.Deaths))
	s.Send(fmt.Sprintf("Coins: [%9d], Bank: [%9d] (Total: %d)\r\n", p.GetGold(), p.GetBankGold(), p.GetGold()+p.GetBankGold()))
	s.Send(fmt.Sprintf("AC: [%d/10], Hitroll: [%2d], Damroll: [%2d], Saving throws: [%d/%d/%d/%d/%d]\r\n",
		p.GetAC(), p.GetHitroll(), p.GetDamroll(), p.GetSavingThrow(0), p.GetSavingThrow(1), p.GetSavingThrow(2), p.GetSavingThrow(3), p.GetSavingThrow(4)))
	position := wizardPositionName(p.GetPosition())
	fighting := wizardFightingName(p.GetFighting())
	line := fmt.Sprintf("Pos: %s, Fighting: %s", position, fighting)
	if connected {
		line += ", Connected: Playing"
	}
	s.Send(line + "\r\n")
	s.Send(fmt.Sprintf("Default position: %s, Idle Timer (in tics) [%d]\r\n", position, p.GetIdleTimer()))
	s.Send("PLR: " + wizardPlayerFlags(p, flags) + "\r\n")
	preferenceFlags := flags
	autoExit := p.GetAutoExit()
	if file {
		// C's file path materializes a freshly loaded char_data with no runtime
		// preference bits. Go's save record stores the shifted preference mask,
		// so do not leak that implementation detail into the C report.
		preferenceFlags = 0
		autoExit = false
	}
	s.Send("PRF: " + wizardPreferenceFlags(preferenceFlags, autoExit) + "\r\n")
	s.Send("Racial Hatreds: None None None None None\r\n")
	eqCount := 0
	if p.Equipment != nil {
		for _, item := range p.Equipment.Slots {
			if item != nil {
				eqCount++
			}
		}
	}
	invCount := len(p.GetInventory())
	s.Send(fmt.Sprintf("Items in: inventory: %d, eq: %d\r\n", invCount, eqCount))
	full := p.GetCondition(game.CondFull)
	thirst := p.GetCondition(game.CondThirst)
	drunk := p.GetCondition(game.CondDrunk)
	if file {
		// load_char() initializes the transient condition slots to 24 on the
		// file-stat path; those values are not the active character's runtime
		// conditions.
		full, thirst, drunk = 24, 24, 24
	}
	s.Send(fmt.Sprintf("Hunger: %d, Thirst: %d, Drunk: %d, Tattoo: None (%d)\r\n", full, thirst, drunk, p.TatTimer))
	s.Send("Master is: <none>, Followers are:\r\n")
	s.Send("AFF: " + wizardAffectFlags(p.GetAffectBitVector()) + "\r\n")
}

func wizardSexName(sex int) string {
	switch sex {
	case game.SexMale:
		return "MALE"
	case game.SexFemale:
		return "FEMALE"
	default:
		return "NEUTRAL-SEX"
	}
}

func wizardClassName(class int) string {
	if name, ok := game.ClassNames[class]; ok {
		return name
	}
	return "UNDEFINED"
}

func wizardPositionName(position int) string {
	names := []string{"Dead", "Mortally wounded", "Incapacitated", "Stunned", "Sleeping", "Resting", "Sitting", "Fighting", "Standing"}
	if position >= 0 && position < len(names) {
		return names[position]
	}
	return "Standing"
}

func wizardFightingName(name string) string {
	if name == "" {
		return "Nobody"
	}
	return name
}

var wizardMobRaceNames = []string{
	"Human", "Elf", "Dwarf", "Kender", "Centaur", "Rakshasa", "Troll", "Lycanthrope",
	"Vampire", "Undead", "Dragon", "Demon", "Horse", "Reptile", "Arachnid", "Rodent",
	"Other", "Vegetable", "Giant", "Demi-god", "Ogre", "Insect", "Mammal", "Fish", "Avian",
	"Magical Construct", "Amphibian", "Humanoid", "Faery", "Ssaur", "Minotaur",
}

func wizardMobRaceName(race int) string {
	if race >= 0 && race < len(wizardMobRaceNames) {
		return wizardMobRaceNames[race]
	}
	return "None"
}

var wizardMobFlagNames = []string{
	"SPEC", "SENTINEL", "SCAVENGER", "ISNPC", "AWARE", "AGGR", "STAY-ZONE", "WIMPY",
	"AGGR_EVIL", "AGGR_GOOD", "AGGR_NEUTRAL", "MEMORY", "HELPER", "!CHARM", "!SUMMN",
	"!SLEEP", "!BASH", "!BLIND", "HUNTER", "AGGR24", "RNDLD_ZONE", "MOUNTABLE", "RARE", "LOOTS", "OKGIVE",
}

var wizardPlayerFlagNames = []string{
	"OUTLAW", "NOTHIN", "FROZEN", "DONTSET", "WRITING", "MAILING", "CSH", "SITEOK", "NOSHOUT", "NOTITLE",
	"DELETED", "LOADRM", "!WIZL", "!DEL", "INVST", "CRYO", "WEREW", "VAMP", "IT", "CHOSEN", "REMORT",
}

var wizardPreferenceFlagNames = []string{
	"BRIEF", "COMPACT", "DEAF", "!TELL", "D_HP", "D_MANA", "D_MOVE", "AUTOEX", "!HASS", "QUEST", "SUMN",
	"!REP", "LIGHT", "C1", "C2", "!WIZ", "L1", "L2", "!AUC", "!GOS", "!GTZ", "RMFLG", "AFK", "AUTOLOOT",
	"AUTOGOLD", "AUTOSPLIT", "D_TANK", "D_TARGET", "!NEW", "INACTIVE", "!CTELL", "!BROAD",
}

func wizardMobFlags(flags uint64) string {
	return strings.TrimSpace(game.SprintbitArray([]uint32{uint32(flags)}, wizardMobFlagNames, 1))
}

func wizardAffectFlags(flags uint64) string {
	return strings.TrimSpace(game.SprintbitArray([]uint32{uint32(flags)}, wizardCaffectNames, 1))
}

func wizardPlayerFlags(p *game.Player, flags uint64) string {
	plr := p.PlayerFlags | (flags & ((1 << 20) - 1))
	if plr == 0 && flags != 0 {
		// Object movement sets C's crash-save bit. The Go runtime keeps the
		// preference half of this legacy word in Flags, so preserve the visible
		// CSH marker when only that half is present.
		plr = 1 << game.PlrCrash
	}
	return strings.TrimSpace(game.SprintbitArray([]uint32{uint32(plr)}, wizardPlayerFlagNames, 1))
}

func wizardPreferenceFlags(flags uint64, autoExit bool) string {
	goBits := []int{
		game.PrfBrief, game.PrfCompact, game.PrfDeaf, game.PrfNotell, game.PrfDisphp, game.PrfDispmmana,
		game.PrfDispmove, game.PrfAutoexit, game.PrfNohassle, game.PrfQuest, game.PrfSummonable, game.PrfNoRepeat,
		game.PrfHolyLight, game.PrfColor1, game.PrfColor2, game.PrfNowiz, game.PrfLog1, game.PrfLog2,
		game.PrfNoAuctions, game.PrfNoGossip, game.PrfNoGratz, game.PrfRoomFlags, game.PrfAFK, game.PrfAutoLoot,
		game.PrfAutoGold, game.PrfAutoSplit, game.PrfDispTank, game.PrfDispTarget, game.PrfNoNewbie, game.PrfInactive,
		game.PrfNoCTell, game.PrfNoBroad,
	}
	var cflags uint64
	for cbit, gobit := range goBits {
		if cbit == 7 {
			if autoExit {
				cflags |= 1 << uint(cbit)
			}
		} else if flags&(1<<uint(gobit)) != 0 {
			cflags |= 1 << uint(cbit)
		}
	}
	return strings.TrimSpace(game.SprintbitArray([]uint32{uint32(cflags)}, wizardPreferenceFlagNames, 1))
}

var wizardCaffectNames = []string{
	"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE", "WATERWALK", "SANCT", "GROUP", "CURSE",
	"INFRA", "POISON", "PROT-EVIL", "PROT-GOOD", "SLEEP", "!TRACK", "FLESH-ALTER", "DODGE", "SNEAK", "HIDE",
	"BERSERK", "CHARM", "FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY", "WEREWOLF", "VAMPIRE", "MOUNTED",
	"INVULN", "FLAMING", "NOTHING", "HASTE", "SLOW", "DREAM", "WATERBREATHE", "METALSKIN", "ROBBED",
}

func (s *Session) sendStatRoom() {
	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return
	}
	w := s.manager.world
	room := w.GetRoomInWorld(s.player.GetRoom())
	if room == nil {
		s.Send("Room data not found.")
		return
	}

	zoneIndex := 0
	if parsed := w.GetParsedWorld(); parsed != nil {
		for i := range parsed.Zones {
			if parsed.Zones[i].Number == room.Zone {
				zoneIndex = i
				break
			}
		}
	}
	rnum, _ := w.RealRoomIndex(room.VNum)
	s.Send(fmt.Sprintf("Room name: %s\r\n", room.Name))
	s.Send(fmt.Sprintf("Zone: [%3d], VNum: [%5d], RNum: [%5d], Type: %s\r\n", zoneIndex, room.VNum, rnum, wizardSectorName(room.Sector)))
	s.Send(fmt.Sprintf("SpecProc: None, Flags: %s\r\n", wizardRoomFlags(room.Flags)))
	s.Send("Description:\r\n")
	if room.Description != "" {
		s.Send(strings.ReplaceAll(room.Description, "\n", "\r\n"))
	}

	players := w.GetPlayersInRoom(room.VNum)
	sort.Slice(players, func(i, j int) bool { return players[i].GetID() > players[j].GetID() })
	mobs := w.GetMobsInRoom(room.VNum)
	sort.Slice(mobs, func(i, j int) bool { return mobs[i].GetID() < mobs[j].GetID() })
	var chars []string
	for _, player := range players {
		chars = append(chars, fmt.Sprintf("%s(PC)", player.GetName()))
	}
	for _, mob := range mobs {
		chars = append(chars, fmt.Sprintf("%s(MOB)", mob.GetName()))
	}
	s.Send(fmt.Sprintf("Chars present: %s\r\n", strings.Join(chars, ", ")))

	items := w.GetItemsInRoom(room.VNum)
	var itemNames []string
	for _, item := range items {
		itemNames = append(itemNames, item.GetShortDesc())
	}
	if len(itemNames) > 0 {
		s.Send(fmt.Sprintf("Contents: %s\r\n", strings.Join(itemNames, ", ")))
	}
	for _, direction := range []string{"north", "east", "south", "west", "up", "down"} {
		exit, ok := room.Exits[direction]
		if !ok {
			continue
		}
		exitBits := wizardExitFlags(exit.ExitInfo)
		s.Send(fmt.Sprintf("Exit %s:  To: [%5d], Key: [%5d], Keywrd: %s, Type: %s\r\n", direction, exit.ToRoom, exit.Key, wizardExitKeyword(exit.Keywords), exitBits))
		if exit.Description != "" {
			s.Send(strings.ReplaceAll(exit.Description, "\n", "\r\n"))
		} else {
			s.Send("   No exit description.\r\n")
		}
	}
}

func wizardSectorName(sector int) string {
	sectorNames := []string{
		"Inside", "City", "Field", "Forest", "Hills", "Mountains", "Water (Swim)",
		"Water (No Swim)", "Underwater", "In Flight", "Desert", "Fire", "Earth", "Wind",
		"Water", "Swamp",
	}
	if sector >= 0 && sector < len(sectorNames) {
		return sectorNames[sector]
	}
	return "Unknown"
}

func wizardRoomFlags(flags []string) string {
	const roomFlagNames = "DARK DEATH !MOB INDOORS PEACEFUL SOUNDPROOF !TRACK !MAGIC TUNNEL PRIVATE GODROOM HOUSE HCRSH ATRIUM OLC * NEUTRAL BFR REGENROOM NO_WHO_ROOM ** FLOW_NORTH FLOW_SOUTH FLOW_EAST FLOW_WEST FLOW_UP FLOW_DOWN ARENA"
	names := strings.Fields(roomFlagNames)
	words := make([]uint32, 4)
	for i := range words {
		if i < len(flags) {
			value, err := strconv.ParseUint(flags[i], 10, 32)
			if err == nil {
				words[i] = uint32(value)
			}
		}
	}
	return strings.TrimSpace(game.SprintbitArray(words, names, len(words)))
}

func wizardExitFlags(flags int) string {
	const exitFlagNames = "DOOR CLOSED LOCKED PICKPROOF"
	words := []uint32{uint32(flags), 0, 0, 0}
	return strings.TrimSpace(game.SprintbitArray(words, strings.Fields(exitFlagNames), len(words)))
}

func wizardExitKeyword(keyword string) string {
	if keyword == "" {
		return "None"
	}
	return keyword
}

func wizardNameMatches(query, names string) bool {
	query = strings.ToLower(query)
	if query == "" {
		return false
	}
	for _, name := range strings.Fields(strings.ToLower(names)) {
		if strings.HasPrefix(name, query) {
			return true
		}
	}
	return false
}

func (s *Session) sendStatPlayer(p *game.Player) {
	if p != nil {
		s.sendStatPlayerReport(p, p.GetRoom(), true, false)
	}
}

func (s *Session) sendStatObject(name string) {
	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return
	}
	object, ok := s.manager.world.ResolveObjectWorld(s.player, name)
	if !ok {
		s.Send("No such object around.\r\n")
		return
	}
	s.sendStatObjectInstance(object)
}

func (s *Session) sendStatObjectInstance(object *game.ObjectInstance) {
	if object == nil || object.Prototype == nil {
		return
	}
	w := s.manager.world
	proto := object.Prototype
	rnum := -1
	if parsed := w.GetParsedWorld(); parsed != nil {
		for i := range parsed.Objs {
			if parsed.Objs[i].VNum == object.GetVNum() {
				rnum = i
				break
			}
		}
	}

	s.Send(fmt.Sprintf("Name: '%s', Aliases: %s\r\n", object.GetShortDesc(), object.GetKeywords()))
	s.Send(fmt.Sprintf("VNum: [%5d], RNum: [%5d], Type: %s, SpecProc: None\r\n", object.GetVNum(), rnum, wizardItemTypeName(object.GetTypeFlag())))
	s.Send(fmt.Sprintf("L-Des: %s\r\n", object.GetLongDesc()))
	if extra := object.GetExtraDescs(); len(extra) > 0 {
		var names []string
		for _, desc := range extra {
			names = append(names, desc.Keywords)
		}
		s.Send(fmt.Sprintf("Extra descs: %s\r\n", strings.Join(names, " ")))
	}

	words := objectFlagWords(proto.WearFlags)
	s.Send(fmt.Sprintf("Can be worn on: %s\r\n", strings.TrimSpace(game.SprintbitArray(words, wizardWearBitNames, len(words)))))
	s.Send(fmt.Sprintf("Set char bits : %s\r\n", strings.TrimSpace(game.SprintbitArray([]uint32{0, 0, 0, 0}, game.AffectedBitNames, 4))))
	s.Send(fmt.Sprintf("Extra flags   : %s\r\n", strings.TrimSpace(game.SprintbitArray(objectFlagWords(object.GetExtraFlags()), wizardExtraBitNames, 4))))
	s.Send(fmt.Sprintf("Encumbrance: %d, Value: %d, Percent Load: %.2f%%, Timer: %d\r\n", object.GetWeight(), object.GetCost(), proto.LoadPercent, object.GetTimer()))

	inRoom, inObject, carriedBy, wornBy := s.objectLocation(object)
	s.Send(fmt.Sprintf("In room: %s, In object: %s, Carried by: %s, Worn by: %s\r\n", inRoom, inObject, carriedBy, wornBy))
	s.Send(fmt.Sprintf("%s\r\n", wizardObjectValues(object)))
	if len(object.GetContents()) > 0 {
		var contents []string
		for _, child := range object.GetContents() {
			contents = append(contents, child.GetShortDesc())
		}
		s.Send(fmt.Sprintf("\r\nContents: %s\r\n", strings.Join(contents, ", ")))
	}
	if affects := object.GetAffects(); len(affects) > 0 {
		var rendered []string
		for _, affect := range affects {
			if affect.Location != 0 {
				rendered = append(rendered, fmt.Sprintf("%+d to %s", affect.Modifier, wizardApplyTypeName(affect.Location)))
			}
		}
		if len(rendered) == 0 {
			s.Send("Affections: None\r\n")
		} else {
			s.Send(fmt.Sprintf("Affections: %s\r\n", strings.Join(rendered, ",")))
		}
		return
	}
	s.Send("Affections: None\r\n")
}

func objectFlagWords(flags [4]int) []uint32 {
	words := make([]uint32, len(flags))
	for i, flag := range flags {
		words[i] = uint32(flag)
	}
	return words
}

func wizardItemTypeName(itemType int) string {
	if itemType >= 0 && itemType < len(game.ItemTypeNames) {
		return game.ItemTypeNames[itemType]
	}
	return "UNDEFINED"
}

var wizardWearBitNames = []string{
	"TAKE", "FINGER", "NECK", "BODY", "HEAD", "LEGS", "FEET", "HANDS", "ARMS", "SHIELD",
	"ABOUT", "WAIST", "WRIST", "WIELD", "HOLD", "THROW", "ABLEGS", "FACE", "HOVER",
}

var wizardExtraBitNames = []string{
	"GLOW", "HUM", "!RENT", "!DONATE", "!INVIS", "INVIS", "MAGIC", "!DROP", "BLESS", "!GOOD",
	"!EVIL", "!NEU", "!MAGE", "!CLE", "!THI", "!WAR", "!SELL", "NAMED", "!PSI", "!NIN",
	"!PAL", "!MAGUS", "!ASS", "!AVA", "RARE", "!LOCATE", "!RAN", "!MYS", "TWOHANDS",
}

func (s *Session) objectLocation(object *game.ObjectInstance) (string, string, string, string) {
	inRoom, inObject, carriedBy, wornBy := "Nowhere", "None", "Nobody", "Nobody"
	location := object.Location
	switch location.Kind {
	case game.ObjInRoom:
		inRoom = strconv.Itoa(location.RoomVNum)
	case game.ObjInContainer:
		for _, candidate := range s.manager.world.GetAllObjects() {
			if candidate.ID == location.ContainerObjID {
				inObject = candidate.GetShortDesc()
				break
			}
		}
	case game.ObjInInventory:
		if location.OwnerKind == game.OwnerPlayer {
			carriedBy = location.PlayerName
		} else {
			for _, mob := range s.manager.world.GetAllMobs() {
				if mob.GetID() == location.MobID {
					carriedBy = mob.GetName()
					break
				}
			}
		}
	case game.ObjEquipped:
		if location.OwnerKind == game.OwnerPlayer {
			wornBy = location.PlayerName
		} else {
			for _, mob := range s.manager.world.GetAllMobs() {
				if mob.GetID() == location.MobID {
					wornBy = mob.GetName()
					break
				}
			}
		}
	}
	return inRoom, inObject, carriedBy, wornBy
}

func wizardObjectValues(object *game.ObjectInstance) string {
	values := [4]int{object.GetValue(0), object.GetValue(1), object.GetValue(2), object.GetValue(3)}
	switch object.GetTypeFlag() {
	case game.ITEM_FOOD:
		return fmt.Sprintf("Makes full: %d, Poisoned: %s", values[0], wizardYesNo(values[3]))
	default:
		return fmt.Sprintf("Values 0-3: [%d] [%d] [%d] [%d]", values[0], values[1], values[2], values[3])
	}
}

func wizardYesNo(value int) string {
	if value != 0 {
		return "YES"
	}
	return "NO"
}

func wizardApplyTypeName(location int) string {
	if location >= 0 && location < len(game.ApplyTypeNames) {
		return game.ApplyTypeNames[location]
	}
	return "UNDEFINED"
}

// cmdVnum — find vnums by keyword (LVL_IMMORT)
func cmdVnum(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: vnum { obj | mob } <name>\r\n")
		return nil
	}

	category := strings.ToLower(args[0])
	keyword := strings.ToLower(strings.Join(args[1:], " "))
	if category != "mob" && category != "obj" && category != "object" {
		s.Send("Usage: vnum { obj | mob } <name>\r\n")
		return nil
	}

	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return nil
	}

	parsed := s.manager.world.GetParsedWorld()
	if parsed == nil {
		s.Send("Parsed world data not available.")
		return nil
	}

	results := make([]string, 0, 30)
	switch category {
	case "mob":
		found := 0
		for i := range parsed.Mobs {
			m := &parsed.Mobs[i]
			if wizardNameMatches(keyword, m.Keywords) {
				found++
				results = append(results, fmt.Sprintf("%3d. [%5d] %s\r\n", found, m.VNum, m.ShortDesc))
			}
		}
	case "obj", "object":
		found := 0
		for i := range parsed.Objs {
			o := &parsed.Objs[i]
			if wizardNameMatches(keyword, o.Keywords) {
				found++
				results = append(results, fmt.Sprintf("%3d. [%5d] %s\r\n", found, o.VNum, o.ShortDesc))
			}
		}
	}

	if len(results) == 0 {
		if category == "mob" {
			s.Send("No mobiles by that name.\r\n")
		} else {
			s.Send("No objects by that name.\r\n")
		}
		return nil
	}
	for _, r := range results {
		s.Send(r)
	}
	return nil
}

// cmdVstat — detailed vnum info for prototypes (LVL_IMMORT)
func cmdVstat(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: vstat { obj | mob } <number>\r\n")
		return nil
	}

	cat := strings.ToLower(args[0])
	vnum, err := strconv.Atoi(args[1])
	if err != nil {
		s.Send("Usage: vstat { obj | mob } <number>\r\n")
		return nil
	}

	w := s.manager.world

	switch cat {
	case "mob":
		proto, ok := w.GetMobPrototype(vnum)
		if !ok {
			s.Send("There is no monster with that number.\r\n")
			return nil
		}
		s.sendStatMob(game.NewMob(proto, 0))

	case "obj":
		proto, ok := w.GetObjPrototype(vnum)
		if !ok {
			s.Send("There is no object with that number.\r\n")
			return nil
		}
		s.sendStatObjectInstance(game.NewObjectInstance(proto, -1))

	default:
		s.Send("That'll have to be either 'obj' or 'mob'.\r\n")
	}
	return nil
}

// cmdWizlock — toggle wizard-only login (LVL_IMPL)
