package spells

import (
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
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
		castCoC(level, ch, world)
	case SpellMentalLapse:
		castMentalLapse(level, ch, cvict)
	case SpellEnchantWeapon:
		castEnchantWeapon(level, ch, ovict)
	case SpellEnchantArmor:
		castEnchantArmor(level, ch, ovict)
	case SpellCreateWater:
		castCreateWater(level, ch, ovict)
	case SpellIdentify:
		castIdentify(level, ch, cvict, ovict)
	case SpellSilkenMissile:
		castSilkenMissile(level, ch, ovict, world)
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
	type getter interface {
		GetHP() int
		GetMaxHP() int
	}
	g, ok := victim.(getter)
	if !ok {
		return
	}
	newHP := g.GetHP() + amount
	if newHP > g.GetMaxHP() {
		newHP = g.GetMaxHP()
	}
	// Player uses SetHealth; fallback to SetHP for other types.
	type healthSetter interface{ SetHealth(int) }
	type hpSetter interface{ SetHP(int) }
	if s, ok := victim.(healthSetter); ok {
		s.SetHealth(newHP)
	} else if s, ok := victim.(hpSetter); ok {
		s.SetHP(newHP)
	}
}

func checkReagents(ch interface{}, spellNum, level int, reagents ...string) int {
	_ = spellNum
	_ = level
	_ = reagents
	return 0
}

func init() {
	// Wave A manual spell registrations — from src/spell_parser.c spello() calls.
	// C spello() arg order: maxMana, minMana, change.
	// setupSpellInfo() arg order: manaMin, manaMax, manaChange.
	setupSpellInfo(SpellSobriety, PosStanding, 20, 35, 5, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellZen, PosFighting, 60, 70, 4, RoutineManual, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellDetectPoison, PosStanding, 10, 20, 2, RoutineManual, false, TarCharRoom|TarObjInv|TarObjRoom)
	setupSpellInfo(SpellCalliope, PosFighting, 50, 100, 10, RoutineManual, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellLycanthropy, PosStanding, 1, 1, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellVampirism, PosStanding, 1, 1, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellControlWeather, PosStanding, 25, 75, 5, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellCoC, PosStanding, 70, 90, 1, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellMentalLapse, PosStanding, 90, 100, 1, RoutineManual, false, TarCharWorld)
	// Wave B manual spell registrations
	setupSpellInfo(SpellCreateWater, PosStanding, 10, 35, 5, RoutineManual, false, TarObjInv|TarObjEquip)
	setupSpellInfo(SpellEnchantWeapon, PosStanding, 150, 200, 10, RoutineManual, false, TarObjInv|TarObjEquip)
	setupSpellInfo(SpellEnchantArmor, PosStanding, 130, 150, 10, RoutineManual, false, TarObjInv|TarObjEquip)
	setupSpellInfo(SpellIdentify, PosStanding, 100, 125, 10, RoutineManual, false, TarCharRoom|TarObjInv|TarObjRoom)
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

// castCoC — circle of summoning. Creates a COC object in the caster's room.
// C source: spells.c:1012-1039. COC_VNUM=64, timer=level/2+rand(-2,1).
func castCoC(level int, ch interface{}, world interface{}) {
	type roomGetter interface{ GetRoomVNum() int }
	type spawner interface {
		SpawnObject(int, int) (interface{}, error)
		AddItemToRoom(interface{}, int)
	}

	s, ok := world.(spawner)
	if !ok {
		slog.Warn("spell_coc: world does not support object spawning")
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	// Get caster's room
	rg, _ := ch.(roomGetter)
	roomVNum := 0
	if rg != nil {
		roomVNum = rg.GetRoomVNum()
	}
	if roomVNum <= 0 {
		sendToCaster(ch, "You can't do that here.\r\n")
		return
	}

	obj, err := s.SpawnObject(CocVnum, roomVNum)
	if err != nil {
		slog.Error("spell_coc: failed to spawn COC object", "error", err, "vnum", CocVnum)
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	s.AddItemToRoom(obj, roomVNum)

	// Set timer
	type timerSetter interface{ SetTimer(int) }
	if ts, ok := obj.(timerSetter); ok {
		timer := level/2 + rand.Intn(4) - 2 // rand(-2, 1)
		if timer < 1 {
			timer = 1
		}
		ts.SetTimer(timer)
	}

	sendToCaster(ch, "You draw a magic circle on the ground.\r\n")
}

// castSilkenMissile — converts armor/clothing into a missile arrow.
// C source: spells.c:883-912. MISSILE VNUM=3.
func castSilkenMissile(level int, ch, ovict interface{}, world interface{}) {
	_ = level
	if ovict == nil {
		return
	}

	type typeFlagger interface{ GetTypeFlag() int }
	tf, ok := ovict.(typeFlagger)
	if !ok {
		sendToCaster(ch, "You can't make anything useful from that.\r\n")
		return
	}

	objType := tf.GetTypeFlag()
	if objType != 11 && objType != 9 { // ITEM_WORN=11, ITEM_ARMOR=9
		sendToCaster(ch, "You can't make anything useful from that.\r\n")
		return
	}

	type spawner interface {
		SpawnObject(int, int) (interface{}, error)
		AddItemToRoom(interface{}, int)
		ExtractObject(interface{}, int)
	}
	type roomGetter interface{ GetRoomVNum() int }
	type inventoryGetter interface{ GetInventory() []interface{} }
	type inventoryAdder interface{ AddItemToInventory(interface{}) error }

	s, ok := world.(spawner)
	if !ok {
		slog.Warn("spell_silken_missile: world does not support object operations")
		return
	}

	// Get caster's room
	rg, _ := ch.(roomGetter)
	roomVNum := 0
	if rg != nil {
		roomVNum = rg.GetRoomVNum()
	}

	// Spawn missile arrow
	missile, err := s.SpawnObject(MissileVnum, roomVNum)
	if err != nil {
		slog.Error("spell_silken_missile: failed to spawn missile", "error", err, "vnum", MissileVnum)
		sendToCaster(ch, "Error, please tell a god.\r\n")
		return
	}

	// Give missile to caster (try inventory, fall back to room)
	added := false
	if ia, ok := ch.(inventoryAdder); ok {
		if err := ia.AddItemToInventory(missile); err == nil {
			added = true
		}
	}
	if !added && roomVNum > 0 {
		s.AddItemToRoom(missile, roomVNum)
	}

	// Extract source object
	s.ExtractObject(ovict, roomVNum)

	sendToCaster(ch, "You create an arrow from it.\r\n")
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

// --- Wave B manual spell implementations ---

// castEnchantWeapon — adds +hitroll/+damroll to a non-magic weapon.
// C source: spells.c:621-662. Sets ITEM_MAGIC, alignment-colored glow.
func castEnchantWeapon(level int, ch, ovict interface{}) {
	if ovict == nil {
		return
	}

	type typeFlagger interface{ GetTypeFlag() int }
	type affectable interface {
		GetAffects() []parser.ObjAffect
		SetAffectsOverride([]parser.ObjAffect)
		SetExtraFlag(int, int)
		HasExtraFlag(int, int) bool
	}

	obj, ok := ovict.(affectable)
	if !ok {
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}

	tf, _ := ovict.(typeFlagger)
	if tf == nil || tf.GetTypeFlag() != 5 { // ITEM_WEAPON
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}

	// ITEM_MAGIC = bit 6 in ExtraFlags[0]
	if obj.HasExtraFlag(0, 6) {
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}

	// Check existing affects — no enchanting already-affect items
	existing := obj.GetAffects()
	for _, a := range existing {
		if a.Location != 0 { // APPLY_NONE = 0
			sendToCaster(ch, "Nothing seems to happen.\r\n")
			return
		}
	}

	// Apply enchantment
	obj.SetExtraFlag(0, 6) // ITEM_MAGIC

	hitroll := 1
	if level >= 18 {
		hitroll = 2
	}
	damroll := 1
	if level >= 20 {
		damroll = 2
	}

	newAffects := make([]parser.ObjAffect, 2)
	newAffects[0] = parser.ObjAffect{Location: 18, Modifier: hitroll}  // APPLY_HITROLL
	newAffects[1] = parser.ObjAffect{Location: 19, Modifier: damroll}  // APPLY_DAMROLL
	obj.SetAffectsOverride(newAffects)

	// Alignment glow
	type aligner interface{ GetAlignment() int }
	if a, ok := ch.(aligner); ok {
		switch {
		case a.GetAlignment() > 0:
			obj.SetExtraFlag(0, 10) // ITEM_ANTI_EVIL
			sendToCaster(ch, "It glows blue.\r\n")
		case a.GetAlignment() < 0:
			obj.SetExtraFlag(0, 9) // ITEM_ANTI_GOOD
			sendToCaster(ch, "It glows red.\r\n")
		default:
			sendToCaster(ch, "It glows yellow.\r\n")
		}
	}
}

// castEnchantArmor — adds AC improvement to non-magic armor/worn items.
// C source: spells.c:836-871. Sets ITEM_MAGIC, alignment-colored glow.
func castEnchantArmor(level int, ch, ovict interface{}) {
	if ovict == nil {
		return
	}

	type typeFlagger interface{ GetTypeFlag() int }
	type affectable interface {
		GetAffects() []parser.ObjAffect
		SetAffectsOverride([]parser.ObjAffect)
		SetExtraFlag(int, int)
		HasExtraFlag(int, int) bool
	}

	obj, ok := ovict.(affectable)
	if !ok {
		return
	}

	tf, _ := ovict.(typeFlagger)
	if tf == nil {
		return
	}

	objType := tf.GetTypeFlag()
	if objType != 9 && objType != 11 { // ITEM_ARMOR=9, ITEM_WORN=11
		return
	}

	// ITEM_MAGIC = bit 6 in ExtraFlags[0]
	if obj.HasExtraFlag(0, 6) {
		return
	}

	// Check existing affects
	existing := obj.GetAffects()
	for _, a := range existing {
		if a.Location != 0 { // APPLY_NONE
			return
		}
	}

	obj.SetExtraFlag(0, 6) // ITEM_MAGIC

	acMod := -1 * ((level - 20) / 2)

	newAffects := make([]parser.ObjAffect, 1)
	newAffects[0] = parser.ObjAffect{Location: 17, Modifier: acMod} // APPLY_AC
	obj.SetAffectsOverride(newAffects)

	// Alignment glow
	type aligner interface{ GetAlignment() int }
	if a, ok := ch.(aligner); ok {
		switch {
		case a.GetAlignment() > 0:
			obj.SetExtraFlag(0, 10) // ITEM_ANTI_EVIL
			sendToCaster(ch, "It glows blue.\r\n")
		case a.GetAlignment() < 0:
			obj.SetExtraFlag(0, 9) // ITEM_ANTI_GOOD
			sendToCaster(ch, "It glows red.\r\n")
		default:
			sendToCaster(ch, "It glows yellow.\r\n")
		}
	}
}

// castCreateWater — fills a drink container with water.
// C source: spells.c:87-120. Poisons non-water liquid. Needs drink name helpers.
func castCreateWater(level int, ch, ovict interface{}) {
	if ovict == nil {
		return
	}

	type typeFlagger interface{ GetTypeFlag() int }
	type valuer interface{ GetValue(int) int }
	type nameable interface{ SetDrinkName(string) }

	tf, ok := ovict.(typeFlagger)
	if !ok || tf.GetTypeFlag() != 17 { // ITEM_DRINKCON
		sendToCaster(ch, "It's not a drink container.\r\n")
		return
	}

	v, ok := ovict.(valuer)
	if !ok {
		return
	}

	// Values[0] = max capacity, Values[1] = current amount, Values[2] = liquid type
	liquidType := v.GetValue(2)
	current := v.GetValue(1)

	// LIQ_WATER = 0, LIQ_SLIME = 2 (from structs.h)
	if liquidType != 0 && current != 0 {
		// Poison non-water liquid
		// TODO: name_from_drinkcon/name_to_drinkcon — needs SetDrinkName interface
		slog.Info("spell_create_water: poisoned non-water liquid")
		sendToCaster(ch, "The water mixes with the liquid...\r\n")
		return
	}

	// Fill with water — need SetDrinkValue interface
	type drinkSetter interface{ SetValue(int, int) }
	if ds, ok := ovict.(drinkSetter); ok {
		maxCap := v.GetValue(0)
		water := maxCap - current
		if water < 0 {
			water = 0
		}
		if water > 0 {
			clampedLevel := level
			if clampedLevel < 1 {
				clampedLevel = 1
			}
			if clampedLevel > 100 {
				clampedLevel = 100
			}
			ds.SetValue(2, 0) // LIQ_WATER
			ds.SetValue(1, current+water)
			sendToCaster(ch, "It is filled.\r\n")
		} else {
			sendToCaster(ch, "You cannot create water in that!\r\n")
		}
	}
}

// castIdentify — reveals item stats or PC stats.
// C source: spells.c:476-621. Simplified Go implementation.
func castIdentify(level int, ch, cvict, ovict interface{}) {
	if ovict != nil {
		castIdentifyObject(level, ch, ovict)
	} else if cvict != nil {
		castIdentifyCharacter(level, ch, cvict)
	}
}

func castIdentifyObject(level int, ch, ovict interface{}) {
	_ = level
	type typeFlagger interface{ GetTypeFlag() int }
	type valuer interface{ GetValue(int) int }
	type affecter interface{ GetAffects() []parser.ObjAffect }
	type namer interface{ GetName() string }
	type weighter interface{ GetWeight() int }
	type coster interface{ GetCost() int }

	sendToCaster(ch, "You feel informed:\r\n")

	if n, ok := ovict.(namer); ok {
		sendToCaster(ch, "Object: "+n.GetName()+"\r\n")
	}

	// Item type
	if tf, ok := ovict.(typeFlagger); ok {
		typeNames := map[int]string{
			1: "container", 2: "liquid container", 3: "key",
			4: "staff", 5: "weapon", 6: "scroll", 7: "ward",
			8: "misc", 9: "armor", 10: "potion", 11: "worn",
			12: "other", 13: "trash", 14: "trap",
			15: "npc corpse", 16: "pc corpse", 17: "drink container",
			18: "fountain", 19: "food", 20: "money",
			22: "boat", 23: "fountain",
		}
		if name, ok := typeNames[tf.GetTypeFlag()]; ok {
			sendToCaster(ch, "Item type: "+name+"\r\n")
		} else {
			sendToCaster(ch, "Item type: unknown\r\n")
		}
	}

	// Weight and cost
	if w, ok := ovict.(weighter); ok {
		sendToCaster(ch, fmt.Sprintf("Weight: %d\r\n", w.GetWeight()))
	}
	if c, ok := ovict.(coster); ok {
		sendToCaster(ch, fmt.Sprintf("Value: %d\r\n", c.GetCost()))
	}

	// Type-specific info
	tf, _ := ovict.(typeFlagger)
	v, _ := ovict.(valuer)
	if tf != nil && v != nil {
		switch tf.GetTypeFlag() {
		case 5: // ITEM_WEAPON
			sendToCaster(ch, fmt.Sprintf("Damage: %dD%d\r\n", v.GetValue(1), v.GetValue(2)))
			avg := float64((v.GetValue(2)+1)/2) * float64(v.GetValue(1))
			sendToCaster(ch, fmt.Sprintf("Average damage: %.1f\r\n", avg))
		case 9: // ITEM_ARMOR
			sendToCaster(ch, fmt.Sprintf("AC-apply: %d\r\n", v.GetValue(0)))
		case 4, 6, 10: // ITEM_STAFF, ITEM_SCROLL, ITEM_POTION
			// Spell contents
			for i := 1; i <= 3; i++ {
				if v.GetValue(i) >= 1 {
					sendToCaster(ch, fmt.Sprintf("Spell slot %d: %d\r\n", i, v.GetValue(i)))
				}
			}
		}
	}

	// Affects
	if a, ok := ovict.(affecter); ok {
		affects := a.GetAffects()
		applyNames := map[int]string{
			17: "AC", 18: "hitroll", 19: "damroll", 1: "strength",
			2: "dexterity", 3: "intelligence", 4: "wisdom",
			5: "constitution", 6: "charisma",
		}
		for _, aff := range affects {
			if aff.Location != 0 && aff.Modifier != 0 {
				name := applyNames[aff.Location]
				if name == "" {
					name = fmt.Sprintf("apply(%d)", aff.Location)
				}
				sendToCaster(ch, fmt.Sprintf("Affects: %s by %d\r\n", name, aff.Modifier))
			}
		}
	}
}

func castIdentifyCharacter(level int, ch, cvict interface{}) {
	_ = level
	type namer interface{ GetName() string }
	type lever interface{ GetLevel() int }
	type stater interface{ GetHP() int; GetMaxHP() int; GetMana() int }
	type hper interface{ GetHitroll() int; GetDamroll() int; GetAC() int }

	sendToCaster(ch, "You feel informed:\r\n")

	if n, ok := cvict.(namer); ok {
		sendToCaster(ch, fmt.Sprintf("Name: %s\r\n", n.GetName()))
	}
	if l, ok := cvict.(lever); ok {
		sendToCaster(ch, fmt.Sprintf("Level: %d\r\n", l.GetLevel()))
	}
	if s, ok := cvict.(stater); ok {
		sendToCaster(ch, fmt.Sprintf("Hits: %d/%d, Mana: %d\r\n", s.GetHP(), s.GetMaxHP(), s.GetMana()))
	}
	if h, ok := cvict.(hper); ok {
		sendToCaster(ch, fmt.Sprintf("AC: %d, Hitroll: %d, Damroll: %d\r\n", h.GetAC(), h.GetHitroll(), h.GetDamroll()))
	}
}


