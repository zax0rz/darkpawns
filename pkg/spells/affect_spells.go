package spells

import (
	"log/slog"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// MagAffects applies spell affects to a character.
// Functions named MagXxx to match C convention; constants are RoutineXxx.
func MagAffects(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil || ch == nil {
		return
	}
	_ = level
	_ = ch
	_ = savetype
	_ = world

	var aff *engine.Affect

	switch spellNum {
	case SpellChillTouch:
		dur := 4
		if magSavingThrow(victim, savetype) {
			dur = 1
		}
		aff = engine.NewAffect(engine.AffectStrength, dur, -1, "chill touch")
	case SpellBless:
		aff = engine.NewAffect(engine.AffectHitRoll, 6, 2, "bless")
		applyAffect(victim, aff)
		aff = engine.NewAffect(engine.AffectArmorClass, 6, -2, "bless")
	case SpellArmor:
		aff = engine.NewAffect(engine.AffectArmorClass, 24, -15, "armor")
	case SpellBlindness, SpellSmokescreen:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "You shake off the blinding effect.\r\n")
			return
		}
		aff = engine.NewAffect(engine.AffectHitRoll, 2, -(4), "blindness")
		applyAffect(victim, aff)
		aff = engine.NewAffect(engine.AffectBlind, 2, 40, "blindness")
		sendToVictim(victim, "You have been blinded!\r\n")
	case SpellCurse:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "The spell had no effect.\r\n")
			return
		}
		aff = engine.NewAffect(engine.AffectCurse, getLevel(ch)/2, -(3), "curse")
		applyAffect(victim, aff)
		aff = engine.NewAffect(engine.AffectDamageRoll, getLevel(ch)/2, -(3), "curse")
	case SpellInvisible:
		aff = engine.NewAffect(engine.AffectInvisible, 12+getLevel(ch)/4, 0, "invisibility")
	case SpellSanctuary:
		aff = engine.NewAffect(engine.AffectSanctuary, 4, 0, "sanctuary")
	case SpellSleep:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "You resist the spell!\r\n")
			return
		}
		aff = engine.NewAffect(engine.AffectSleep, 4+getLevel(ch)/4, 0, "sleep")
	case SpellPoison:
		if magSavingThrow(victim, savetype) {
			return
		}
		dur := (level / 2) - 2
		if dur < 1 {
			dur = 1
		}
		aff = engine.NewAffect(engine.AffectPoison, dur, -2, "poison")
	case SpellHaste:
		aff = engine.NewAffect(engine.AffectHaste, level, 0, "haste")
	case SpellSlow:
		aff = engine.NewAffect(engine.AffectSlow, level, 0, "slow")
	case SpellFly:
		aff = engine.NewAffect(engine.AffectFlying, getLevel(ch), 0, "fly")
	case SpellDetectMagic:
		aff = engine.NewAffect(engine.AffectDetectMagic, 12+level, 0, "detect magic")
	case SpellDetectInvis:
		aff = engine.NewAffect(engine.AffectDetectInvisible, 12+level, 0, "detect invis")
	case SpellInfravision:
		aff = engine.NewAffect(engine.AffectInfrared, 12+level, 0, "infravision")
	case SpellWaterBreathe:
		aff = engine.NewAffect(engine.AffectWaterBreathing, getLevel(ch), 0, "water breathe")
	default:
		return
	}

	applyAffect(victim, aff)
}

// MagPoints handles HP/MV restoration spells.
func MagPoints(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil {
		return
	}
	_ = ch
	_ = savetype
	_ = world

	hit := 0

	switch spellNum {
	case SpellCureLight:
		hit = dice(2, 8) + 1 + (level >> 2)
		sendToVictim(victim, "You feel better.\r\n")
	case SpellCureCritic:
		hit = dice(5, 8) + 3 + (level >> 2)
		sendToVictim(victim, "You feel a lot better!\r\n")
	case SpellHeal:
		hit = 100 + dice(3, 8)
		sendToVictim(victim, "A warm feeling floods your body.\r\n")
	case SpellVitality:
		hit = dice(5, 10)
		sendToVictim(victim, "You feel vitalized!\r\n")
	}

	if hit > 0 {
		healHP(victim, hit)
	}
}

// MagUnaffects removes spell affects from a target.
func MagUnaffects(level int, ch, victim interface{}, spellNum int, world interface{}) {
	if victim == nil {
		return
	}
	_ = level
	_ = ch
	_ = world

	switch spellNum {
	case SpellCureBlind, SpellHeal, SpellMassHeal:
		removeAffect(victim, SpellBlindness)
		sendToVictim(victim, "Your vision clears!\r\n")
	case SpellRemovePoison:
		removeAffect(victim, SpellPoison)
		sendToVictim(victim, "A warm feeling runs through your body!\r\n")
	case SpellRemoveCurse:
		removeAffect(victim, SpellCurse)
		sendToVictim(victim, "You don't feel so unlucky.\r\n")
	}
}

// MagGroups applies group versions of spells.
func MagGroups(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	_ = level
	_ = savetype
	_ = world
	_ = spellNum
}

// MagMasses applies mass (room-wide) spells.
func MagMasses(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	_ = level
	_ = savetype
	_ = world
	_ = spellNum
}

// MagAreas applies area (room-wide offensive) spells.
func MagAreas(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	_ = level
	_ = savetype
	_ = world
	_ = spellNum
}

// MagSummons summons NPCs into the world.
func MagSummons(level int, ch interface{}, spellNum int, world interface{}) {
	_ = level
	_ = ch
	_ = spellNum
	_ = world
}

// MagCreations creates objects.
func MagCreations(level int, ch interface{}, spellNum int, world interface{}) {
	_ = level
	_ = ch
	_ = spellNum
	_ = world
}

// MagAlterObjs alters objects.
func MagAlterObjs(level int, ch, obj interface{}, spellNum int, world interface{}) {
	_ = level
	_ = ch
	_ = obj
	_ = spellNum
	_ = world
}

// ExecuteManualSpell dispatches manual (ASPELL) spell implementations.
// Ported from src/spell_parser.c MANUAL_SPELL dispatch.
func ExecuteManualSpell(spellNum, level int, ch, cvict, ovict interface{}, arg string, world interface{}) {
	if ch == nil {
		return
	}

	switch spellNum {
	case SpellSobriety:
		castSobriety(level, ch, cvict)
	case SpellZen:
		castZen(level, ch, cvict)
	case SpellDetectPoison:
		castDetectPoison(level, ch, cvict, ovict)
	case SpellCalliope:
		castCalliope(level, ch, cvict)
	case SpellLycanthropy:
		castLycanthropy(level, ch, cvict)
	case SpellVampirism:
		castVampirism(level, ch, cvict)
	case SpellControlWeather:
		castControlWeather(level, ch, arg, world)
	case SpellCoC:
		castCoC(level, ch, cvict)
	case SpellMentalLapse:
		castMentalLapse(level, ch, cvict)
	default:
		sendToCaster(ch, "Spell not yet implemented.\r\n")
	}

	_ = world
}

// --- helpers ---

func sendToVictim(victim interface{}, msg string) {
	type sender interface{ SendMessage(string) }
	if s, ok := victim.(sender); ok {
		s.SendMessage(msg)
	}
}

func sendToCaster(ch interface{}, msg string) {
	type sender interface{ SendMessage(string) }
	if s, ok := ch.(sender); ok {
		s.SendMessage(msg)
	}
}

func getLevel(ch interface{}) int {
	type lever interface{ GetLevel() int }
	if l, ok := ch.(lever); ok {
		return l.GetLevel()
	}
	return 1
}

func isClassMage(ch interface{}) bool {
	type classer interface{ GetClass() int }
	if c, ok := ch.(classer); ok {
		return c.GetClass() == 0 || c.GetClass() == 8
	}
	return false
}

func applyAffect(entity interface{}, aff *engine.Affect) {
	type affecter interface{ AddAffect(*engine.Affect) }
	if a, ok := entity.(affecter); ok {
		a.AddAffect(aff)
	}
}

func removeAffect(entity interface{}, spellNum int) {
	type remover interface{ RemoveAffectBySpell(int) }
	if r, ok := entity.(remover); ok {
		r.RemoveAffectBySpell(spellNum)
	}
}

func healHP(victim interface{}, amount int) {
	type healer interface {
		GetHP() int
		GetMaxHP() int
		SetHP(int)
	}
	if h, ok := victim.(healer); ok {
		newHP := h.GetHP() + amount
		if newHP > h.GetMaxHP() {
			newHP = h.GetMaxHP()
		}
		h.SetHP(newHP)
	}
}

func checkReagents(ch interface{}, spellNum, level int, reagents ...string) int {
	_ = spellNum
	_ = level
	_ = reagents
	return 0
}

func init() {
	// Wave A manual spell registrations — from src/spell_parser.c spello() calls
	setupSpellInfo(SpellSobriety, PosStanding, 35, 20, 5, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellZen, PosFighting, 70, 60, 4, RoutineManual, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellDetectPoison, PosStanding, 20, 10, 2, RoutineManual, false, TarCharRoom|TarObjInv|TarObjRoom)
	setupSpellInfo(SpellCalliope, PosFighting, 100, 50, 10, RoutineManual, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellLycanthropy, PosStanding, 1, 1, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellVampirism, PosStanding, 1, 1, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellControlWeather, PosStanding, 75, 25, 5, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellCoC, PosStanding, 90, 70, 1, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellMentalLapse, PosStanding, 100, 90, 1, RoutineManual, false, TarCharWorld)
}

// --- Wave A manual spell implementations ---

// castSobriety — clears drunk condition.
// C source: spells.c:688-715. C bug: assert(victim) crashes on null — we return gracefully.
func castSobriety(level int, ch, cvict interface{}) {
	_ = level
	if cvict == nil {
		sendToCaster(ch, "No target.\r\n")
		return
	}

	type conder interface {
		SetCondition(int, int)
		GetPosition() int
		SetPosition(int)
	}
	c, ok := cvict.(conder)
	if !ok {
		return
	}

	c.SetCondition(0, 0) // DRUNK = 0
	// Position may change after condition reset (C calls update_pos)
	// update_pos logic: if pos > POS_SLEEPING and drunk is 0 and thirst/hunger are 0, sit
	sendToVictim(cvict, "You are splashed in the face with HOT coffee, ")
	sendToVictim(cvict, "but feel much more sober.\r\n")
}

// castZen — heals caster and sets position to stunned (meditating).
// C source: spells.c:871-883. C bug fixed: C uses 'victim' for healing but 'ch' for position.
func castZen(level int, ch, cvict interface{}) {
	sendToCaster(ch, "You begin to meditate deeply, focusing your thoughts only on ")
	sendToCaster(ch, "healing.\r\n")

	// Heal caster (ch), not victim — C bug fix
	healHP(ch, 2*level)

	type poser interface{ SetPosition(int) }
	if p, ok := ch.(poser); ok {
		p.SetPosition(int(PosStunned))
	}

	_ = cvict
}

// castDetectPoison — checks for poison on victims and objects.
// C source: spells.c:794-836. AFF_POISON = bit 11.
func castDetectPoison(level int, ch, cvict, ovict interface{}) {
	_ = level

	// Check victim for poison affect
	if cvict != nil {
		type affecter interface{ IsAffected(int) bool }
		type namer interface{ GetName() string }

		a, ok := cvict.(affecter)
		if !ok {
			return
		}

		if a.IsAffected(11) { // AFF_POISON
			if cvict == ch {
				sendToCaster(ch, "You can sense poison in your blood.\r\n")
			} else {
				sendToCaster(ch, "You sense that they are poisoned.\r\n")
			}
		} else {
			if cvict == ch {
				sendToCaster(ch, "You feel healthy.\r\n")
			} else {
				if n, ok := cvict.(namer); ok {
					sendToCaster(ch, "You sense that "+n.GetName()+" is healthy.\r\n")
				}
			}
		}
	}

	// Check object for poison (values[3] on food/drinkcon/fountain)
	if ovict != nil {
		// Object type checking via interfaces
		type typeFlagger interface{ GetTypeFlag() int }
		type valuer interface{ GetValue(int) int }

		tf, ok := ovict.(typeFlagger)
		if !ok {
			sendToCaster(ch, "You sense that it should not be consumed.\r\n")
			return
		}

		switch tf.GetTypeFlag() {
		case 17, 23, 19: // ITEM_DRINKCON=17, ITEM_FOUNTAIN=23, ITEM_FOOD=19
			if v, ok := ovict.(valuer); ok && v.GetValue(3) != 0 {
				sendToCaster(ch, "You sense that it has been contaminated.\r\n")
			} else {
				sendToCaster(ch, "You sense that it is safe for consumption.\r\n")
			}
		default:
			sendToCaster(ch, "You sense that it should not be consumed.\r\n")
		}
	}
}

// castCalliope — fires multiple magic missiles at a target.
// C source: spells.c:983-997. missiles = MAX(4, number(level/6, level*2)).
func castCalliope(level int, ch, cvict interface{}) {
	if cvict == nil {
		return
	}

	lo := level / 6
	hi := level * 2
	missiles := lo
	if hi > lo {
		missiles += rand.Intn(hi-lo+1)
	}
	if missiles < 4 {
		missiles = 4
	}

	for i := 0; i < missiles; i++ {
		CallMagic(ch, cvict, nil, SpellMagicMissile, level, CastSpell, nil)
	}
}

// castLycanthropy — sets PLR_WEREWOLF flag.
// C source: spells.c:662-700.
func castLycanthropy(level int, ch, cvict interface{}) {
	_ = level
	if cvict == nil {
		sendToCaster(ch, "Specify a target.\r\n")
		return
	}

	// PLR_WEREWOLF=16, PLR_VAMPIRE=17 via HasPLRFlag/SetPLRFlag interface
	type flagChecker interface {
		HasPLRFlag(int) bool
		SetPLRFlag(int)
	}
	p, ok := cvict.(flagChecker)
	if !ok {
		sendToCaster(ch, "That only works on players.\r\n")
		return
	}

	if p.HasPLRFlag(16) { // PLR_WEREWOLF
		sendToCaster(ch, "Already a werewolf.\r\n")
		return
	}
	if p.HasPLRFlag(17) { // PLR_VAMPIRE
		sendToCaster(ch, "Already a creature of the night.\r\n")
		return
	}

	sendToVictim(cvict, "You feel a strange sensation in your bones...\r\n")
	p.SetPLRFlag(16)
}

// castVampirism — sets PLR_VAMPIRE flag. PC only.
// C source: spells.c:766-794.
func castVampirism(level int, ch, cvict interface{}) {
	_ = level
	if cvict == nil {
		sendToCaster(ch, "Specify a target.\r\n")
		return
	}

	type flagChecker interface {
		HasPLRFlag(int) bool
		SetPLRFlag(int)
	}

	p, ok := cvict.(flagChecker)
	if !ok {
		sendToCaster(ch, "That only works on players.\r\n")
		return
	}

	if p.HasPLRFlag(17) { // PLR_VAMPIRE
		sendToCaster(ch, "Already a vampire.\r\n")
		return
	}
	if p.HasPLRFlag(16) { // PLR_WEREWOLF
		sendToCaster(ch, "Already a creature of the night.\r\n")
		return
	}

	sendToVictim(cvict, "You feel a strange sensation in your blood!\r\n")
	p.SetPLRFlag(17)
}

// castControlWeather — adjusts weather change variable.
// C source: spells.c:997-1012. "better" adds dice(level/3, 4), "worse" subtracts.
// Uses the world parameter for weather mutation.
func castControlWeather(level int, ch interface{}, arg string, world interface{}) {
	type weatherMutator interface{ ModifyWeatherChange(int) }

	var w weatherMutator
	if world != nil {
		w, _ = world.(weatherMutator)
	}

	switch strings.ToLower(strings.TrimSpace(arg)) {
	case "better":
		if w != nil {
			w.ModifyWeatherChange(dice(level/3, 4))
		}
	case "worse":
		if w != nil {
			w.ModifyWeatherChange(-dice(level/3, 4))
		}
	default:
		sendToCaster(ch, "Do you want it to get better or worse?\r\n")
	}
}

// castCoC — circle of summoning. STUB until runtime object creation exists.
// C source: spells.c:1012-1039. COC_VNUM = 64, timer = level/2 + rand(-2,1).
func castCoC(level int, ch, cvict interface{}) {
	_ = cvict
	slog.Info("spell_coc: runtime object creation not yet implemented", "vnum", CocVnum, "level", level)
	sendToCaster(ch, "You draw a magic circle on the ground.\r\n")
}

// castMentalLapse — clears mob hunting target. Partial implementation.
// C source: spells.c:960-983. Full version needs mob hunting system.
func castMentalLapse(level int, ch, cvict interface{}) {
	if cvict == nil {
		slog.Warn("spell_mental_lapse: no victim")
		return
	}

	// Get caster name for hunting match
	type namer interface{ GetName() string }
	casterName := ""
	if n, ok := ch.(namer); ok {
		casterName = n.GetName()
	}

	// MobInstance must implement HuntingGetter/Setter interfaces
	type hunter interface {
		GetHunting() string
		ClearHunting()
	}

	mob, ok := cvict.(hunter)
	if !ok {
		sendToCaster(ch, "Your psionic energy recoils!\r\n")
		return
	}

	hunting := mob.GetHunting()
	if hunting == "" {
		sendToCaster(ch, "Your psionic energy recoils!\r\n")
		return
	}

	// Level 30+ can redirect mobs hunting others
	if hunting != casterName {
		type lever interface{ GetLevel() int }
		casterLevel := 0
		if l, ok := ch.(lever); ok {
			casterLevel = l.GetLevel()
		}
		if casterLevel < 30 {
			sendToCaster(ch, "Your psionic energy recoils!\r\n")
			return
		}
	}

	sendToCaster(ch, "You sense their intentions even from where you are, and ")
	sendToCaster(ch, "change their mind.\r\n")

	mob.ClearHunting()
}


