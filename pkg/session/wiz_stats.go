package session

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
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
		if victim, ok := s.manager.world.ResolveCharWorld(s.player, args[1]); ok && victim.Combatant != nil {
			s.sendStatCharacter(victim)
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
		player, record, err := s.storedPlayer(args[1])
		if err != nil || player == nil {
			s.Send("There is no such player.\r\n")
			return nil
		}
		if player.GetLevel() > getEffectiveLevel(s) {
			s.Send("Sorry, you can't do that.\r\n") // act.wizard.c:1036-1037.
			return nil
		}
		s.sendStatPlayerFile(player, record.OlcZone)
		return nil
	case "obj", "object":
		if len(args) == 1 {
			s.Send("Stats on which object?\r\n")
			return nil
		}
		s.sendStatObject(args[1])
		return nil
	}
	// C do_stat fallback chain (src/act.wizard.c:1055-1070):
	// equip -> inventory -> room char -> room obj -> world char -> world obj.
	if object, ok := s.manager.world.ResolveObjectInEquipmentVis(s.player, args[0]); ok {
		s.sendStatObjectInstance(object)
		return nil
	}
	if object, ok := s.manager.world.ResolveObjectInInventory(s.player, args[0]); ok {
		s.sendStatObjectInstance(object)
		return nil
	}
	if victim, ok := s.manager.world.ResolveCharInRoom(s.player, args[0]); ok {
		s.sendStatCharacter(victim)
		return nil
	}
	if object, ok := s.manager.world.ResolveObjectInRoom(s.player, args[0]); ok {
		s.sendStatObjectInstance(object)
		return nil
	}
	if victim, ok := s.manager.world.ResolveCharWorld(s.player, args[0]); ok {
		s.sendStatCharacter(victim)
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

func (s *Session) sendStatCharacter(target game.CharTarget) {
	if target.Player != nil {
		s.sendStatPlayer(target.Player)
		return
	}
	if target.Mob != nil {
		s.sendStatMob(target.Mob)
	}
}

func (s *Session) sendStatMob(mob *game.MobInstance) {
	if mob == nil || mob.Proto() == nil {
		return
	}
	proto := mob.Proto()
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
		mob.GetLevel(), proto.Exp, mob.GetAlignment()))
	race := proto.Race
	if proto.RaceStr == "" {
		// The C parser's absent Race: extra field means RACE_OTHER, which is
		// entry 16 in mob_races[]. The Go parser keeps its internal default at
		// 7 for compatibility with the broader mob model.
		race = 16
	}
	s.sendStatMobLine(fmt.Sprintf("Race: [%d] %s", race, wizardMobRaceName(race)))
	s.sendStatMobLine("*************-------------*************-------------*************")
	s.sendStatMobLine(fmt.Sprintf("Str: [%d/%d]\tInt: [%d]\t*\tHit p.:  [%d/%d+%d]",
		mob.GetStr(), mob.GetStrAdd(), mob.GetInt(), mob.GetHP(), mob.GetMaxHP(), game.MobHitGain(mob)))
	s.sendStatMobLine(fmt.Sprintf("Dex: [%d]\tWis: [%d]\t*\tMana p.: [%d/%d+%d]",
		mob.GetDex(), mob.GetWis(), mob.GetMana(), mob.GetMaxMana(), game.ManaGainNPC(mob)))
	s.sendStatMobLine(fmt.Sprintf("Con: [%d/%d]\tCha: [%d]\t*\tMove p.: [%d/%d+%d]",
		mob.GetCon(), mob.GetCon(), mob.GetCha(), mob.GetMove(), mob.GetMaxMove(), game.MoveGainNPC(mob)))
	s.sendStatMobLine("*************-------------*************-------------*************")
	s.sendStatMobLine(fmt.Sprintf("Coins: [%9d], Bank: [%9d] (Total: %d)", mob.GetGold(), 0, mob.GetGold()))
	s.sendStatMobLine(fmt.Sprintf("AC: [%d/10], Hitroll: [%2d], Damroll: [%2d], Saving throws: [%d/%d/%d/%d/%d]",
		mob.GetAC(), mob.GetHitroll(), mob.GetDamrollPoint(), mob.GetSavingThrow(0), mob.GetSavingThrow(1), mob.GetSavingThrow(2), mob.GetSavingThrow(3), mob.GetSavingThrow(4)))
	attackType := proto.BareHandAttack
	attackName := combat.AttackHitTexts[0].Singular
	if attackType >= 0 && attackType < len(combat.AttackHitTexts) {
		attackName = combat.AttackHitTexts[attackType].Singular
	}
	s.sendStatMobLine(fmt.Sprintf("Pos: %s, Fighting: %s, Attack type: %s",
		wizardPositionName(mob.GetPosition()), wizardFightingName(mob.GetFighting()), attackName))
	if proto.Noise != "" {
		color, reset := "", ""
		if s.player != nil && s.player.GetFlags()&(1<<uint(game.PrfColor2)) != 0 {
			color, reset = "\x1b[33m", "\x1b[0m"
		}
		s.sendStatMobLine("Noise: " + color + proto.Noise + reset)
	}
	s.sendStatMobLine(fmt.Sprintf("Default position: %s, Idle Timer (in tics) [%d]",
		wizardPositionName(proto.DefaultPos), 0))
	s.sendStatMobLine("NPC flags: " + wizardMobFlags(mob.GetMobFlags()))
	s.sendStatMobLine(fmt.Sprintf("Mob Spec-Proc: None, NPC Bare Hand Dam: %dd%d", proto.Damage.Num, proto.Damage.Sides))
	carriedWeight := 0
	for _, item := range mob.Inventory {
		carriedWeight += item.GetTotalWeight()
	}
	s.sendStatMobLine(fmt.Sprintf("Carried: weight: %d, items: %d; Items in: inventory: %d, eq: %d",
		carriedWeight, len(mob.Inventory), len(mob.Inventory), len(mob.Equipment)))
	if hunting := mob.GetHunting(); hunting != "" {
		s.sendStatMobLine("Hunting: " + hunting)
	}
	s.sendStatMobLine("Master is: <none>, Followers are:")
	s.sendStatMobLine("AFF: " + wizardAffectFlags(mobEffectiveAffectBits(mob)))
	s.sendStatAffectLines(mob.OrderedAffects())
}

func (s *Session) sendStatMobLine(line string) {
	s.Send(line + "\r\n")
}

func (s *Session) sendStatPlayerFile(player *game.Player, olcZone int) {
	// store_to_char clears carried objects and resets combat scalars
	// (db.c:2452-2463); Crash_load does not run for stat file.
	player.Inventory = game.NewInventory()
	player.Equipment = game.NewEquipment()
	player.AC, player.Hitroll, player.Damroll = 100, 0, 0
	player.RestoreEffectiveAttributes() // store_to_char, without Crash_load equipment
	s.sendStatPlayerReport(player, -1, false, olcZone)
}

// statOlcZone is the live descriptor-owned representation of C's saved
// olc_zone. File reports receive the zone from their original SQLite read.
func (s *Session) statOlcZone(name string) int {
	if _, involved := s.manager.switchDescriptorByName(name); involved {
		if owner, ok := s.manager.GetSession(name); ok {
			s.manager.mu.RLock()
			zone := owner.olcZone
			s.manager.mu.RUnlock()
			return zone
		}
	}
	if live := findSessionByName(s.manager, name); live != nil {
		return live.olcZone
	}
	return 0
}

func (s *Session) sendStatPlayerReport(p *game.Player, room int, connected bool, olcZone int) {
	if p == nil {
		return
	}

	flags := p.GetFlags()
	// Player IDs assigned by the no-database runtime already use C's
	// one-based idnum sequence (manager.go:1422-1427). Do not add another
	// offset when rendering the C-facing report.
	id := p.GetID()

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
	pt := p.PlayingTime()
	age := game.Age(p.Birth).Year
	created := time.Unix(p.Birth, 0).Format("Mon Jan _2 15:04:05 2006")
	lastLogon := p.ConnectedAt.Format("Mon Jan _2 15:04:05 2006")
	s.Send(fmt.Sprintf("Created: [%s], Last Logon: [%s]\r\n", created, lastLogon))
	s.Send(fmt.Sprintf("Played [%dd %dh], Age [%d]\r\n", pt.Day, pt.Hours, age))
	xpToLevel := 0
	if p.GetLevel() < LVL_IMMORT {
		xpToLevel = game.ExpNeededForLevel(p) - p.GetExp()
	}
	raceName := "Undefined"
	if race := p.GetRace(); race >= 0 && race < len(game.PCRaceTypes) {
		raceName = game.PCRaceTypes[race]
	}
	lastDeath := "NONE"
	if death := p.GetLastDeath(); death != 0 {
		lastDeath = time.Unix(death, 0).Format("Mon Jan _2 15:04")
	}
	hometown, practices := p.GetHometown(), p.GetPractices()
	hitPoints, originalCon := p.GetHP(), p.GetOrigCon()
	olc := ""
	if p.GetLevel() >= LVL_IMMORT {
		// C appends the OLC zone for immortals only (act.wizard.c:767-776).
		olc = fmt.Sprintf(", OLC[%d]", olcZone)
	}
	s.Send(fmt.Sprintf("Hometown: [%d], Speaks: [0/0/0], (STL[%d]/per[%d]/NSTL[%d])%s\r\n",
		hometown, practices, game.IntAppLearn(p.GetInt()), game.WisAppBonus(p.GetWis()), olc))
	s.Send(fmt.Sprintf("Race: [%d] %s  XP to level: [%d]  Last Death: [%s]\r\n",
		p.GetRace(), raceName, xpToLevel, lastDeath))
	s.Send("*************-------------*************-------------*************\r\n")
	s.Send(fmt.Sprintf("Str: [%d/%d]\tInt: [%d]\t*\tHit p.:  [%d/%d+%d]\r\n",
		p.GetStr(), p.GetStrAdd(), p.GetInt(), hitPoints, p.GetMaxHP(), s.manager.world.HitGain(p)))
	s.Send(fmt.Sprintf("Dex: [%d]\tWis: [%d]\t*\tMana p.: [%d/%d+%d]\r\n",
		p.GetDex(), p.GetWis(), p.GetMana(), p.GetMaxMana(), s.manager.world.ManaGain(p)))
	s.Send(fmt.Sprintf("Con: [%d/%d]\tCha: [%d]\t*\tMove p.: [%d/%d+%d]\r\n",
		p.GetCon(), originalCon, p.GetCha(), p.GetMove(), p.GetMaxMove(), s.manager.world.MoveGain(p)))
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
	s.Send(fmt.Sprintf("Default position: %s, Idle Timer (in tics) [%d]\r\n", wizardPositionName(combat.PosStanding), p.GetIdleTimer()))
	s.Send("PLR: " + wizardPlayerFlags(p, flags) + "\r\n")
	preferenceFlags := flags
	autoExit := p.GetAutoExit()

	s.Send("PRF: " + wizardPreferenceFlags(preferenceFlags, autoExit) + "\r\n")
	s.Send("Racial Hatreds: None None None None None\r\n")
	if p.GetLevel() >= LVL_IMMORT {
		poofIn, poofOut := p.PoofIn, p.PoofOut
		if poofIn == "" {
			poofIn = "None"
		}
		if poofOut == "" {
			poofOut = "None"
		}
		// C prints the configured arrival/departure strings for immortal PCs
		// (src/act.wizard.c:905), including the literal fallback "None".
		s.Send(fmt.Sprintf("Poofin:%s\r\nPoofout:%s\r\n", poofIn, poofOut))
	}
	eqCount := 0
	if p.Equipment != nil {
		for _, item := range p.Equipment.Slots {
			if item != nil {
				eqCount++
			}
		}
	}
	invCount := len(p.GetInventory())
	s.Send(fmt.Sprintf("Carried: weight: %d, items: %d; Items in: inventory: %d, eq: %d\r\n",
		p.CarriedWeight(), invCount, invCount, eqCount))
	full := p.GetCondition(game.CondFull)
	thirst := p.GetCondition(game.CondThirst)
	drunk := p.GetCondition(game.CondDrunk)

	s.Send(fmt.Sprintf("Hunger: %d, Thirst: %d, Drunk: %d, Tattoo: None (%d)\r\n", full, thirst, drunk, p.TatTimer))
	s.Send("Master is: <none>, Followers are:\r\n")
	s.Send("AFF: " + wizardAffectFlags(playerEffectiveAffectBits(p)) + "\r\n")
	// C affect_to_char/affect_to_char2 prepends to ch->affected (src/handler.c:391,415),
	// whereas Go appends to p.ActiveAffects (pkg/game/player_affects.go:405).
	ordered := make([]*engine.Affect, 0, len(p.ActiveAffects))
	for i := len(p.ActiveAffects) - 1; i >= 0; i-- {
		ordered = append(ordered, p.ActiveAffects[i])
	}
	s.sendStatAffectLines(ordered)
}

func playerEffectiveAffectBits(p *game.Player) uint64 {
	if p == nil {
		return 0
	}
	flags := p.GetAffectBitVector()
	for bit := 0; bit < len(wizardCaffectNames); bit++ {
		if p.IsAffected(bit) {
			flags |= 1 << uint(bit)
		}
	}
	for _, aff := range p.ActiveAffects {
		if aff != nil && (aff.Flags == 0 || aff.Flags == engine.AFFNothing) {
			flags |= 1 << 32 // AFF_NOTHING (src/structs.h:348)
		}
	}
	return flags
}

func mobEffectiveAffectBits(mob *game.MobInstance) uint64 {
	if mob == nil {
		return 0
	}
	flags := mob.GetAffects()
	for _, aff := range mob.OrderedAffects() {
		if aff != nil && (aff.Flags == 0 || aff.Flags == engine.AFFNothing) {
			flags |= 1 << 32 // AFF_NOTHING (src/structs.h:348)
		}
	}
	return flags
}

// sendStatAffectLines renders do_stat_character's SPL: rows (src/act.wizard.c:971-991).
func (s *Session) sendStatAffectLines(affects []*engine.Affect) {
	for _, aff := range affects {
		if aff == nil {
			continue
		}
		spellName := spells.SpellRawName(aff.SpellID)
		if spellName == "" {
			spellName = aff.Source
		}
		line := fmt.Sprintf("SPL: (%3dhr) %-21s ", aff.Duration+1, spellName)
		hasMod := false
		if aff.Magnitude != 0 {
			line += fmt.Sprintf("%+d to %s", aff.Magnitude, wizardApplyTypeName(aff.Location))
			hasMod = true
		}
		if bitName, ok := wizardAffectBitName(aff); ok {
			if hasMod {
				line += ", sets " + bitName
			} else {
				line += "sets " + bitName
			}
		}
		s.Send(line + "\r\n")
	}
}

func wizardAffectBitName(aff *engine.Affect) (string, bool) {
	if aff == nil {
		return "", false
	}
	// In C (src/structs.h:310,348; src/act.wizard.c:980):
	// AFF_BLIND is bit 0, so `if (aff->bitvector)` is false for AFF_BLIND.
	// Pure-stat spells set `af.bitvector = AFF_NOTHING` (32), which is non-zero.
	if aff.Flags == engine.AFFBlind {
		return "", false
	}
	if aff.Flags == 0 || aff.Flags == engine.AFFNothing {
		return "NOTHING", true
	}
	for cBit := 1; cBit < len(wizardCaffectNames); cBit++ {
		if engFlag, ok := game.AffBitToEngineFlag[cBit]; ok && aff.Flags&engFlag != 0 {
			return wizardCaffectNames[cBit], true
		}
	}
	return "", false
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
	if class >= 0 && class < len(game.PCClassTypes) {
		return game.PCClassTypes[class]
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
	return strings.TrimSpace(game.SprintbitArray([]uint32{wizardFlagWord(flags)}, wizardMobFlagNames, 1))
}

func wizardAffectFlags(flags uint64) string {
	return strings.TrimSpace(game.SprintbitArray([]uint32{wizardFlagWord(flags), wizardFlagWord(flags >> 32), 0, 0}, wizardCaffectNames, 4))
}

func wizardFlagWord(flags uint64) uint32 {
	if flags > uint64(^uint32(0)) {
		flags &= uint64(^uint32(0))
	}
	return uint32(flags)
}

func wizardPlayerFlags(p *game.Player, flags uint64) string {
	plr := flags & (1<<uint(game.PrfBrief) - 1) // PLR bits sit below the PRF range
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
	if parsed := w.GetParsedWorld(); parsed != nil {
		for i := range parsed.Rooms {
			prototype := &parsed.Rooms[i]
			if prototype.VNum != room.VNum || len(prototype.ExtraDescs) == 0 {
				continue
			}
			keywords := make([]string, 0, len(prototype.ExtraDescs))
			for _, desc := range prototype.ExtraDescs {
				keywords = append(keywords, desc.Keywords)
			}
			s.Send(wizardExtraDescriptionLine(keywords, wizardHasNormalColor(s.player.GetFlags())))
			break
		}
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
	colored := wizardHasNormalColor(s.player.GetFlags())
	s.Send(wizardWrappedListOutput("Chars present:", chars, "\x1b[33m", colored))

	items := w.GetItemsInRoom(room.VNum)
	var itemNames []string
	for _, item := range items {
		itemNames = append(itemNames, item.GetShortDesc())
	}
	if len(itemNames) > 0 {
		s.Send(wizardWrappedListOutput("Contents:", itemNames, "\x1b[32m", colored))
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
	words := []uint32{wizardIntFlagWord(flags), 0, 0, 0}
	return strings.TrimSpace(game.SprintbitArray(words, strings.Fields(exitFlagNames), len(words)))
}

func wizardIntFlagWord(flags int) uint32 {
	if flags < 0 || uint64(flags) > uint64(^uint32(0)) {
		return 0
	}
	return uint32(flags)
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
		s.sendStatPlayerReport(p, p.GetRoom(), true, s.statOlcZone(p.GetName()))
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
		s.Send(wizardExtraDescriptionLine(names, wizardHasNormalColor(s.player.GetFlags())))
	}

	words := objectFlagWords(proto.WearFlags)
	s.Send(fmt.Sprintf("Can be worn on: %s\r\n", strings.TrimSpace(game.SprintbitArray(words, wizardWearBitNames, len(words)))))
	s.Send(fmt.Sprintf("Set char bits : %s\r\n", strings.TrimSpace(game.SprintbitArray([]uint32{0, 0, 0, 0}, game.AffectedBitNames, 4))))
	s.Send(fmt.Sprintf("Extra flags   : %s\r\n", strings.TrimSpace(game.SprintbitArray(objectFlagWords(object.GetExtraFlags()), wizardExtraBitNames, 4))))
	s.Send(fmt.Sprintf("Encumbrance: %d, Value: %d, Percent Load: %.2f%%, Timer: %d\r\n", object.GetTotalWeight(), object.GetCost(), proto.LoadPercent, object.GetTimer()))

	inRoom, inObject, carriedBy, wornBy := s.objectLocation(object)
	s.Send(fmt.Sprintf("In room: %s, In object: %s, Carried by: %s, Worn by: %s\r\n", inRoom, inObject, carriedBy, wornBy))
	s.Send(fmt.Sprintf("%s\r\n", wizardObjectValues(object)))
	if contents := wizardObjectContentsOutput(object.GetContents(), wizardHasNormalColor(s.player.GetFlags())); contents != "" {
		s.Send(contents)
	}
	s.Send(wizardObjectAffectOutput(object.GetAffects()))
}

func wizardObjectAffectOutput(affects []parser.ObjAffect) string {
	var output strings.Builder
	output.WriteString("Affections:")
	found := 0
	for _, affect := range affects {
		if affect.Location == 0 || affect.Location == game.ApplySpell {
			continue
		}
		if found > 0 {
			output.WriteByte(',')
		}
		fmt.Fprintf(&output, " %+d to %s", affect.Modifier, wizardApplyTypeName(affect.Location))
		found++
	}
	for _, affect := range affects {
		if affect.Location != game.ApplySpell {
			continue
		}
		if found > 0 {
			output.WriteByte(',')
		}
		fmt.Fprintf(&output, " casts permanent %s", wizardObjectAffectedBitName(affect.Modifier))
		found++
	}
	if found == 0 {
		output.WriteString(" None")
	}
	output.WriteString("\r\n")
	return output.String()
}

func wizardObjectAffectedBitName(index int) string {
	// C's affected_bits[] names differ from the general Go bit labels.
	// Preserve the stat report spelling and index order from src/constants.c:596-635.
	// C indexes the table unchecked (src/act.wizard.c:680); an out-of-range
	// modifier is undefined behaviour there and would panic here (R1a), so it
	// renders as sprinttype's "UNDEFINED".
	names := [...]string{
		"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE", "WATERWALK", "SANCT",
		"GROUP", "CURSE", "INFRA", "POISON", "PROT-EVIL", "PROT-GOOD", "SLEEP", "!TRACK",
		"FLESH-ALTER", "DODGE", "SNEAK", "HIDE", "BERSERK", "CHARM", "FOLLOW", "WIMPY",
		"KUJI-KIRI", "CUTTHROAT", "FLY", "WEREWOLF", "VAMPIRE", "MOUNTED", "INVULN", "FLAMING",
		"NOTHING", "HASTE", "SLOW", "DREAM", "WATERBREATHE", "METALSKIN", "ROBBED",
	}
	if index < 0 || index >= len(names) {
		return "UNDEFINED"
	}
	return names[index]
}

func objectFlagWords(flags [4]int) []uint32 {
	words := make([]uint32, len(flags))
	for i, flag := range flags {
		words[i] = wizardIntFlagWord(flag)
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
	case game.ITEM_LIGHT:
		if values[2] == -1 {
			return "Hours left: Infinite"
		}
		return fmt.Sprintf("Hours left: [%d]", values[2])
	case game.ITEM_SCROLL, game.ITEM_POTION:
		return fmt.Sprintf("Spells: (Level %d) %s, %s, %s", values[0], wizardSkillName(values[1]), wizardSkillName(values[2]), wizardSkillName(values[3]))
	case game.ITEM_WAND, game.ITEM_STAFF:
		return fmt.Sprintf("Spell: %s at level %d, %d (of %d) charges remaining", wizardSkillName(values[3]), values[0], values[2], values[1])
	case game.ITEM_WEAPON:
		average := (float64(values[2]+1) / 2.0) * float64(values[1])
		return fmt.Sprintf("Todam: %dd%d, Message type: %d Apr: %.1f", values[1], values[2], values[3], average)
	case game.ITEM_ARMOR:
		return fmt.Sprintf("AC-apply: [%d]", values[0])
	case game.ITEM_TRAP:
		return fmt.Sprintf("Spell: %d, - Hitpoints: %d", values[0], values[1])
	case game.ITEM_CONTAINER:
		lockType := "<INVALID BITVECTOR>"
		if values[1] >= 0 {
			lockType = game.SprintbitArray([]uint32{wizardIntFlagWord(values[1])}, game.ContainerBitNames, 1)
		}
		return fmt.Sprintf("Capacity: %d, Lock Type: %s, Key Num: %d, Corpse: %s", values[0], lockType, values[2], wizardYesNo(values[3]))
	case game.ITEM_DRINKCON, game.ITEM_FOUNTAIN:
		return fmt.Sprintf("Capacity: %d, Contains: %d, Poisoned: %s, Liquid: %s", values[0], values[1], wizardYesNo(values[3]), game.DrinkName(values[2]))
	case game.ITEM_NOTE:
		return fmt.Sprintf("Tongue: %d", values[0])
	case game.ITEM_KEY:
		return ""
	case game.ITEM_FOOD:
		return fmt.Sprintf("Makes full: %d, Poisoned: %s", values[0], wizardYesNo(values[3]))
	case game.ITEM_MONEY:
		return fmt.Sprintf("Coins: %d", values[0])
	default:
		return fmt.Sprintf("Values 0-3: [%d] [%d] [%d] [%d]", values[0], values[1], values[2], values[3])
	}
}

func wizardSkillName(skill int) string {
	if skill <= 0 {
		if skill == -1 {
			return "UNUSED"
		}
		return "UNDEFINED"
	}
	name := spells.SpellRawName(skill)
	if name == "" || name == "\n" {
		return "UNDEFINED"
	}
	return name
}

func wizardExtraDescriptionLine(keywords []string, colored bool) string {
	var line strings.Builder
	line.WriteString("Extra descs:")
	if colored {
		line.WriteString("\x1b[36m") // CCCYN(ch, C_NRM), src/screen.h:45-56.
	}
	for _, keyword := range keywords {
		line.WriteByte(' ')
		line.WriteString(keyword)
	}
	if colored {
		line.WriteString("\x1b[0m") // CCNRM(ch, C_NRM), src/screen.h:45-56.
	}
	line.WriteString("\r\n")
	return line.String()
}

func wizardHasNormalColor(flags uint64) bool {
	// C's clr(ch, C_NRM) is _clrlevel(ch) >= 2, where PRF_COLOR_1 counts 1 and
	// PRF_COLOR_2 counts 2 (src/screen.h:45-49): PRF_COLOR_2 alone ("color
	// normal") is enough.
	return flags&(1<<uint(game.PrfColor2)) != 0
}

// wizardWrappedListOutput mirrors do_stat_room's 62-byte buffer flushes.
// C starts each list with its label and optional color, then clears the whole
// buffer after a wrapped line; continuation lines therefore have no label or
// color prefix (src/act.wizard.c:451-490; src/screen.h:45-56).
func wizardWrappedListOutput(label string, entries []string, colorCode string, colored bool) string {
	var output, line strings.Builder
	line.WriteString(label)
	if colored {
		line.WriteString(colorCode)
	}

	found := 0
	for i, entry := range entries {
		if found > 0 {
			line.WriteByte(',')
		}
		line.WriteByte(' ')
		line.WriteString(entry)
		found++
		if line.Len() >= 62 {
			if i+1 < len(entries) {
				line.WriteString(",\r\n")
			} else {
				line.WriteString("\r\n")
			}
			output.WriteString(line.String())
			line.Reset()
			found = 0
		}
	}
	if line.Len() > 0 {
		line.WriteString("\r\n")
		output.WriteString(line.String())
	}
	if colored {
		output.WriteString("\x1b[0m")
	}
	return output.String()
}

func wizardObjectContentsOutput(contents []*game.ObjectInstance, colored bool) string {
	if len(contents) == 0 {
		return ""
	}

	var output, line strings.Builder
	line.WriteString("\r\nContents:")
	if colored {
		line.WriteString("\x1b[32m") // CCGRN(ch, C_NRM), src/screen.h:45-56.
	}
	found := 0
	for i, child := range contents {
		if found > 0 {
			line.WriteByte(',')
		}
		line.WriteByte(' ')
		line.WriteString(child.GetShortDesc())
		found++
		if line.Len() >= 62 {
			if i+1 < len(contents) {
				line.WriteString(",\r\n")
			} else {
				line.WriteString("\r\n")
			}
			output.WriteString(line.String())
			line.Reset()
			found = 0
		}
	}
	if line.Len() > 0 {
		line.WriteString("\r\n")
		output.WriteString(line.String())
	}
	if colored {
		output.WriteString("\x1b[0m") // CCNRM(ch, C_NRM), src/screen.h:45-56.
	}
	return output.String()
}

func wizardYesNo(value int) string {
	if value != 0 {
		return "YES"
	}
	return "NO"
}

func wizardApplyTypeName(location int) string {
	if location >= 0 && location < len(wizardCApplyTypeNames) {
		return wizardCApplyTypeNames[location]
	}
	return "UNDEFINED"
}

var wizardCApplyTypeNames = [...]string{
	"NONE", "STR", "DEX", "INT", "WIS", "CON", "CHA", "CLASS", "LEVEL", "AGE",
	"CHAR_WEIGHT", "CHAR_HEIGHT", "MAXMANA", "MAXHIT", "MAXMOVE", "GOLD", "EXP", "ARMOR", "HITROLL", "DAMROLL",
	"SAVING_PARA", "SAVING_ROD", "SAVING_PETRI", "SAVING_BREATH", "SAVING_SPELL", "RACE_HATE", "HIT_REGEN", "MANA_REGEN", "MOVE_REGEN", "PERM_SPELL",
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
