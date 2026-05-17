package spells

import (
	"fmt"
	"log/slog"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
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
		aff = engine.NewAffect(SpellChillTouch, engine.ApplyStr, dur, -1, "chill touch")
	case SpellBless:
		aff = engine.NewAffect(SpellBless, engine.ApplyHitroll, 6, 2, "bless")
		applyAffect(victim, aff)
		aff = engine.NewAffect(SpellBless, engine.ApplySavingSpell, 6, -2, "bless")
		applyAffect(victim, aff)
	case SpellArmor:
		aff = engine.NewAffect(SpellArmor, engine.ApplyAC, 24, -15, "armor")
	case SpellBlindness, SpellSmokescreen:
		reag := 0
		if isClassMage(ch) {
			reag = checkReagents(ch, SpellBlindness, level)
			if reag > 0 {
				sendToCaster(ch, "You crush a small lens under your heel.\r\n")
			}
		}
		if magSavingThrow(victim, savetype) {
			if spellNum == SpellBlindness {
				sendToCaster(ch, "Your magic fades, then dies out totally.\r\n")
			}
			npcRetaliate(victim, ch)
			return
		}
		aff = engine.NewAffect(SpellBlindness, engine.ApplyHitroll, 2, -(4+reag), "blindness")
		applyAffect(victim, aff)
		aff = engine.NewAffectDirect(SpellBlindness, engine.ApplyNone, 2+reag, 40, engine.AFFBlind, "blindness")
		sendToVictim(victim, "You have been blinded!\r\n")
	case SpellCurse:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "The spell had no effect.\r\n")
			npcRetaliate(victim, ch)
			return
		}
		curseDur := 1 + (getLevel(ch) >> 1)
		aff = engine.NewAffectDirect(SpellCurse, engine.ApplyNone, curseDur, -3, engine.AFFCurse, "curse")
		applyAffect(victim, aff)
		aff = engine.NewAffect(SpellCurse, engine.ApplyDamroll, curseDur, -3, "curse")
	case SpellInvisible:
		aff = engine.NewAffectDirect(SpellInvisible, engine.ApplyNone, 12+getLevel(ch)/4, 0, engine.AFFInvisible, "invisibility")
	case SpellSanctuary:
		aff = engine.NewAffectDirect(SpellSanctuary, engine.ApplyNone, 4, 0, engine.AFFSanctuary, "sanctuary")
	case SpellSleep:
		if magSavingThrow(victim, savetype) {
			sendToVictim(victim, "You resist the spell!\r\n")
			return
		}
		aff = engine.NewAffectDirect(SpellSleep, engine.ApplyNone, 4+getLevel(ch)/4, 0, engine.AFFSleep, "sleep")
	case SpellFlameStrike:
		// C source: magic.c:1109-1129 — outdoor-only DOT with saving throw
		if magSavingThrow(victim, savetype) {
			sendToCaster(ch, "Nothing seems to happen.\r\n")
			return
		}
		// Must be outdoors
		w, wOk := world.(worldAoe)
		if wOk {
			chRG, _ := ch.(roomGetter2)
			if chRG != nil {
				roomData := w.GetRoomInWorld(chRG.GetRoomVNum())
				if roomData != nil && roomData.HasFlag(3) && roomData.Sector == 0 {
					sendToCaster(ch, "You can only do this outdoors!\r\n")
					return
				}
			}
		}
		// Duration: level * 0.17, max ~5 hours for mortals
		dur := int(float64(level) * 0.17)
		if dur < 1 {
			dur = 1
		}
		aff = engine.NewAffectDirect(SpellFlameStrike, engine.ApplyNone, dur, 0, engine.AFFFlaming, "flamestrike")
	case SpellPoison:
		if magSavingThrow(victim, savetype) {
			npcRetaliate(victim, ch)
			return
		}
		dur := (level / 2) - 2
		if dur < 1 {
			dur = 1
		}
		aff = engine.NewAffectDirect(SpellPoison, engine.ApplyNone, dur, -2, engine.AFFPoison, "poison")
	case SpellHaste:
		aff = engine.NewAffectDirect(SpellHaste, engine.ApplyNone, level, 0, engine.AFFHaste, "haste")
	case SpellSlow:
		aff = engine.NewAffectDirect(SpellSlow, engine.ApplyNone, level, 0, engine.AFFSlow, "slow")
	case SpellFly:
		aff = engine.NewAffectDirect(SpellFly, engine.ApplyNone, getLevel(ch), 0, engine.AFFFlying, "fly")
	case SpellDetectMagic:
		aff = engine.NewAffectDirect(SpellDetectMagic, engine.ApplyNone, 12+level, 0, engine.AFFDetectMagic, "detect magic")
	case SpellDetectInvis:
		aff = engine.NewAffectDirect(SpellDetectInvis, engine.ApplyNone, 12+level, 0, engine.AFFDetectInvisible, "detect invis")
	case SpellInfravision:
		aff = engine.NewAffectDirect(SpellInfravision, engine.ApplyNone, 12+level, 0, engine.AFFInfrared, "infravision")
	case SpellWaterBreathe:
		aff = engine.NewAffectDirect(SpellWaterBreathe, engine.ApplyNone, getLevel(ch), 0, engine.AFFWaterBreathing, "water breathe")
	case SpellDetectAlign, SpellKnowAlign:
		aff = engine.NewAffectDirect(SpellDetectAlign, engine.ApplyNone, 12+level, 0, engine.AFFDetectAlign, "detect align")
	case SpellDreamTravel:
		aff = engine.NewAffectDirect(SpellDreamTravel, engine.ApplyNone, 6, 0, engine.AFFDream, "dream travel")
	case SpellLevitate:
		// Levitate uses same AFF flag as Fly
		aff = engine.NewAffectDirect(SpellLevitate, engine.ApplyNone, getLevel(ch), 0, engine.AFFFlying, "levitate")
	case SpellProtFromEvil:
		if isEvil(victim) {
			sendToCaster(ch, "You cannot protect yourself from the Evil inside you!\r\n")
			// C source: magic.c:1142-1148 — raw_kill(ch, TYPE_BLAST) on alignment violation
			if c, ok := ch.(combat.Combatant); ok {
				combat.RawKill(c, combat.TYPE_BLAST)
			}
			return
		}
		// Set flag only — no stat modifier
		applyAffect(victim, engine.NewAffectDirect(SpellProtFromEvil, engine.ApplyNone, 24, 0, engine.AFFProtectionEvil, "prot from evil"))
		return
	case SpellProtFromGood:
		if isGood(victim) {
			sendToCaster(ch, "The forces of Light destroy you for your betrayal!\r\n")
			// C source: magic.c:1162-1168 — raw_kill(ch, TYPE_BLAST) on alignment violation
			if c, ok := ch.(combat.Combatant); ok {
				combat.RawKill(c, combat.TYPE_BLAST)
			}
			return
		}
		applyAffect(victim, engine.NewAffectDirect(SpellProtFromGood, engine.ApplyNone, 24, 0, engine.AFFProtectionGood, "prot from good"))
		return
	case SpellAdrenaline, SpellStrength:
		mag := 1 + boolToInt(level > 18)
		if ch == victim && spellNum == SpellAdrenaline {
			mag++
		}
		aff = engine.NewAffect(SpellStrength, engine.ApplyStr, (getLevel(ch)>>1)+4, mag, "strength")
	case SpellSenseLife:
		aff = engine.NewAffectDirect(SpellSenseLife, engine.ApplyNone, getLevel(ch), 0, engine.AFFSenseLife, "sense life")
	case SpellWaterwalk:
		aff = engine.NewAffectDirect(SpellWaterwalk, engine.ApplyNone, 4+getLevel(ch)/5, 0, engine.AFFWaterwalk, "waterwalk")
	case SpellChangeDensity:
		aff = engine.NewAffectDirect(SpellChangeDensity, engine.ApplyNone, 4+getLevel(ch)/5, 0, engine.AFFWaterwalk, "change density")
	case SpellChameleon:
		aff = engine.NewAffectDirect(SpellChameleon, engine.ApplyNone, getLevel(ch), 0, engine.AFFHide, "chameleon")
	case SpellMetalskin:
		reag := checkReagents(ch, SpellMetalskin, getLevel(ch), "chunk of iron",
			"A small chunk of iron melts in your palm as you cast the spell...", "flat:1")
		applyAffect(victim, engine.NewAffectDirect(SpellMetalskin, engine.ApplyNone, 5, -(15+getLevel(ch)/2+reag), engine.AFFMetalskin, "metalskin"))
		applyAffect(victim, engine.NewAffect(SpellMetalskin, engine.ApplyAC, 5, -(15+getLevel(ch)/2+reag), "metalskin"))
	case SpellInvulnerability:
		applyAffect(victim, engine.NewAffectDirect(SpellInvulnerability, engine.ApplyNone, 7, -100, engine.AFFInvuln, "invulnerability"))
		aff = engine.NewAffect(SpellInvulnerability, engine.ApplySavingSpell, 7, -7, "invulnerability")
	case SpellPsyshield:
		aff = engine.NewAffect(SpellPsyshield, engine.ApplyAC, getLevel(ch)/2, -15, "psyshield")
	case SpellGreatPercept:
		applyAffect(victim, engine.NewAffectDirect(SpellGreatPercept, engine.ApplyNone, level/2+4, 0, engine.AFFDetectInvisible, "great percept"))
		aff = engine.NewAffectDirect(SpellGreatPercept, engine.ApplyNone, level/2+4, 0, engine.AFFSenseLife, "great percept")
	case SpellLessPercept:
		applyAffect(victim, engine.NewAffectDirect(SpellLessPercept, engine.ApplyNone, level/2+4, 0, engine.AFFDetectAlign, "lesser percept"))
		aff = engine.NewAffectDirect(SpellLessPercept, engine.ApplyNone, level/2+4, 0, engine.AFFInfrared, "lesser percept")
	case SpellIntellect:
		aff = engine.NewAffect(SpellIntellect, engine.ApplyInt, 8, 1, "intellect")
	case SpellMindBar:
		aff = engine.NewAffectDirect(SpellMindBar, engine.ApplyNone, (level/2)-2, -18, engine.AFFMindBar, "mind bar")
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
	case SpellCellAdjustment:
		hit = 90 + dice(2, 8)
		sendToVictim(victim, "You focus your mind on healing your body..\r\n")
	case SpellMassHeal:
		hit = 200
		sendToVictim(victim, "A warm feeling floods your body.\r\n")
	case SpellInvigorate:
		sendToVictim(victim, "You feel invigorated!\r\n")
		// Movement restoration — healHP handles hit, we handle move separately
		type mover interface{ GetMove() int; GetMaxMove() int; SetMove(int) }
		if m, ok := victim.(mover); ok {
			move := dice(10, 10)
			newMove := m.GetMove() + move
			if newMove > m.GetMaxMove() {
				newMove = m.GetMaxMove()
			}
			m.SetMove(newMove)
		}
	case SpellLayHands:
		if ch == victim {
			sendToVictim(victim, "Your wounds mend beneath your hands!\r\n")
		} else {
			sendToCaster(ch, "Your wounds start to heal beneath $n's hands!\r\n")
		}
		hit = dice(3, getLevel(ch))
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
// MagGroups applies group spells to all grouped characters in the room.

func MagGroups(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}

	// Get all characters in the room
	type roomGetter interface{ GetRoomVNum() int }
	type worldChars interface{ GetAllCharsInRoom(roomVNum int) []interface{} }

	rg, ok := ch.(roomGetter)
	if !ok {
		return
	}
	roomVNum := rg.GetRoomVNum()

	w, ok := world.(worldChars)
	if !ok {
		return
	}

	chars := w.GetAllCharsInRoom(roomVNum)

	// Apply spell to each grouped character (including caster)
	for _, c := range chars {
		if !areGrouped(ch, c) && c != ch {
			continue
		}
		switch spellNum {
		case SpellGroupHeal:
			MagPoints(level, ch, c, SpellHeal, savetype, world)
		case SpellHolyShield:
			MagAffects(level, ch, c, SpellArmor, savetype, world)
		case SpellGroupInvis:
			MagAffects(level, ch, c, SpellInvisible, savetype, world)
		case SpellMassDominate:
			// Mass dominate — charm each grouped NPC
			ExecuteManualSpell(SpellCharm, level, ch, c, nil, "", world)
		}
	}
}

// MagMasses applies mass (room-wide) spells.
// MagMasses applies mass spells to all non-grouped characters in the room.

func MagMasses(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}

	type roomGetter interface{ GetRoomVNum() int }
	type worldChars interface{ GetAllCharsInRoom(roomVNum int) []interface{} }
	type npcChecker interface{ IsNPC() bool }
	type lever interface{ GetLevel() int }

	rg, ok := ch.(roomGetter)
	if !ok {
		return
	}
	roomVNum := rg.GetRoomVNum()

	w, ok := world.(worldChars)
	if !ok {
		return
	}

	// Send messages
	switch spellNum {
	case SpellSmokescreen:
		sendToCaster(ch, "As you quickly mumble the incantation a cloud of thick, acrid black smoke forms around you.\r\n")
	}

	// Apply to each non-grouped character in the room
	chars := w.GetAllCharsInRoom(roomVNum)
	for _, c := range chars {
		if c == ch {
			continue
		}
		// Skip immortals
		if nc, ok := c.(npcChecker); ok && !nc.IsNPC() {
			if l, ok := c.(lever); ok && l.GetLevel() >= 100 {
				continue
			}
		}
		// Skip charmed NPCs
		if nc, ok := c.(npcChecker); ok && nc.IsNPC() {
			type affectChecker interface{ IsAffected(int) bool }
			if ac, ok := c.(affectChecker); ok && ac.IsAffected(1 << 2) { // AFF_CHARM
				continue
			}
		}
		// Skip grouped
		if areGrouped(ch, c) {
			continue
		}
		// Apply affect
		MagAffects(level, ch, c, spellNum, savetype, world)
	}
}

// MagAreas applies area (room-wide offensive) spells.
// MagAreas applies area damage to all non-grouped characters in the room.

func MagAreas(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}

	type roomGetter interface{ GetRoomVNum() int }
	type worldChars interface{ GetAllCharsInRoom(roomVNum int) []interface{} }
	type npcChecker interface{ IsNPC() bool }
	type lever interface{ GetLevel() int }

	rg, ok := ch.(roomGetter)
	if !ok {
		return
	}
	roomVNum := rg.GetRoomVNum()

	w, ok := world.(worldChars)
	if !ok {
		return
	}

	// Send area messages
	switch spellNum {
	case SpellEarthquake:
		sendToCaster(ch, "You gesture and the earth begins to shake all around you!\r\n")
	case SpellAcidBlast:
		sendToCaster(ch, "A spray of acid flows from your fingertips!\r\n")
	case SpellFireBreath:
		// Fire breath is handled by the manual cast function
		return
	}

	chars := w.GetAllCharsInRoom(roomVNum)
	for _, c := range chars {
		if c == ch {
			continue
		}
		// Skip immortals
		if nc, ok := c.(npcChecker); ok && !nc.IsNPC() {
			if l, ok := c.(lever); ok && l.GetLevel() >= 100 {
				continue
			}
		}
		// Skip charmed NPCs
		if nc, ok := c.(npcChecker); ok && nc.IsNPC() {
			type affectChecker interface{ IsAffected(int) bool }
			if ac, ok := c.(affectChecker); ok && ac.IsAffected(1 << 2) { // AFF_CHARM
				continue
			}
		}
		// Skip grouped
		if areGrouped(ch, c) {
			continue
		}
		// Deal damage
		MagDamage(level, ch, c, spellNum, 1, world)
	}
}

// MagSummons summons NPCs into the world.
// MagSummons spawns NPCs based on spell type (e.g. Animate Dead creates zombies).

func MagSummons(level int, ch interface{}, spellNum int, world interface{}) {
	if ch == nil {
		return
	}

	switch spellNum {
	case SpellAnimateDead:
		// Animate dead requires a corpse object in the room
		// This is handled by the manual cast function (ExecuteManualSpell)
		// For the routine path, we spawn a zombie
		type roomGetter interface{ GetRoomVNum() int }
		rg, ok := ch.(roomGetter)
		if !ok {
			return
		}
		roomVNum := rg.GetRoomVNum()

		type mobSpawner interface {
			SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error)
		}
		w, ok := world.(mobSpawner)
		if !ok {
			return
		}

		// MOB_ZOMBIE vnum — check C source for the value
		mobLevel := getLevel(ch) / 2
		if mobLevel < 1 {
			mobLevel = 1
		}
		mob, err := w.SpawnMobWithLevelI(10, roomVNum, mobLevel) // MOB_ZOMBIE = 10
		if err != nil {
			sendToCaster(ch, "The corpse refuses to come alive!\r\n")
			return
		}

		// Add as follower
		type followerAdder interface{ AddFollowerQuiet(ch, leader interface{}) }
		if fa, ok := world.(followerAdder); ok {
			fa.AddFollowerQuiet(mob, ch)
		}

		sendToCaster(ch, "The corpse starts to twitch, then stands with a life of its own!\r\n")
	}
}

// MagCreations creates items based on spell type (e.g. Create Food spawns mushrooms).
func MagCreations(level int, ch interface{}, spellNum int, world interface{}) {
	if ch == nil {
		return
	}

	switch spellNum {
	case SpellCreateFood:
		// Create food spawns magic mushrooms (vnum 8062)
		type roomGetter interface{ GetRoomVNum() int }
		rg, ok := ch.(roomGetter)
		if !ok {
			return
		}
		roomVNum := rg.GetRoomVNum()

		type objSpawner interface {
			SpawnObject(vnum, roomVNum int) (interface{}, error)
		}
		w, ok := world.(objSpawner)
		if !ok {
			return
		}

		obj, err := w.SpawnObject(8062, roomVNum) // OBJ_MAGIC_MUSHROOMS = 8062
		if err != nil {
			sendToCaster(ch, "I seem to have goofed.\r\n")
			return
		}

		// Give to caster
		type inventoryAdder interface{ AddItemToInventory(interface{}) error }
		if ia, ok := ch.(inventoryAdder); ok {
			if err := ia.AddItemToInventory(obj); err != nil {
				sendToCaster(ch, "You can't carry any more!\r\n")
				return
			}
		}

		sendToCaster(ch, "You create some magic mushrooms.\r\n")
	}
}

// MagAlterObjs alters objects.
// MagAlterObjs modifies target object attributes based on spell type.
func MagAlterObjs(level int, ch, obj interface{}, spellNum int, world interface{}) {
	if obj == nil || ch == nil {
		return
	}
	_ = level
	_ = world

	// Object interfaces for flag manipulation
	type extraFlagGetter interface{ GetExtraFlags() int }
	type extraFlagSetter interface{ SetExtraFlags(int) }
	type objTypeGetter interface{ GetObjType() int }
	type objValGetter interface{ GetObjVal(idx int) int }
	type objValSetter interface{ SetObjVal(idx, val int) }
	type objWeightGetter interface{ GetWeight() int }

	getExtraFlags := func(o interface{}) int {
		if f, ok := o.(extraFlagGetter); ok {
			return f.GetExtraFlags()
		}
		return 0
	}
	setExtraFlags := func(o interface{}, flags int) {
		if f, ok := o.(extraFlagSetter); ok {
			f.SetExtraFlags(flags)
		}
	}

	switch spellNum {
	case SpellBless:
		flags := getExtraFlags(obj)
		// ITEM_BLESS = 1<<0, ITEM_MAGIC = 1<<4
		if flags&(1<<0) != 0 || flags&(1<<4) != 0 {
			sendToCaster(ch, "It doesn't seem to have any effect.\r\n")
			return
		}
		if wg, ok := obj.(objWeightGetter); ok {
			cl := 0
			if clGetter, ok := ch.(interface{ GetLevel() int }); ok {
				cl = clGetter.GetLevel()
			}
			if wg.GetWeight() > 5*cl {
				sendToCaster(ch, "It doesn't seem to have any effect.\r\n")
				return
			}
		}
		setExtraFlags(obj, flags|(1<<0)) // ITEM_BLESS
		sendToCaster(ch, "$p glows briefly with an ethereal light.\r\n")

	case SpellCurse:
		flags := getExtraFlags(obj)
		if flags&(1<<1) != 0 { // ITEM_NODROP
			sendToCaster(ch, "It doesn't seem to have any effect.\r\n")
			return
		}
		setExtraFlags(obj, flags|(1<<1)) // ITEM_NODROP
		if ot, ok := obj.(objTypeGetter); ok && ot.GetObjType() == 3 { // ITEM_WEAPON
			if vs, ok := obj.(objValSetter); ok {
				if vg, ok := obj.(objValGetter); ok {
					vs.SetObjVal(2, vg.GetObjVal(2)-1)
				}
			}
		}
		sendToCaster(ch, "$p briefly glows red.\r\n")

	case SpellInvisible:
		flags := getExtraFlags(obj)
		// ITEM_NOINVIS = 1<<5 — don't make noinvis items invisible
		if flags&(1<<5) != 0 {
			sendToCaster(ch, "It doesn't seem to have any effect.\r\n")
			return
		}
		setExtraFlags(obj, flags|(1<<2)) // ITEM_INVISIBLE
		sendToCaster(ch, "$p vanishes.\r\n")

	case SpellPoison:
		if ot, ok := obj.(objTypeGetter); ok {
			objType := ot.GetObjType()
			// ITEM_DRINKCON=17, ITEM_FOUNTAIN=18, ITEM_FOOD=19
			if objType == 17 || objType == 18 || objType == 19 {
				if vs, ok := obj.(objValSetter); ok {
					if vg, ok := obj.(objValGetter); ok {
						if vg.GetObjVal(3) == 0 {
							vs.SetObjVal(3, 1) // poison val
							sendToCaster(ch, "$p steams briefly.\r\n")
						}
					}
				}
			}
		}

	case SpellRemoveCurse:
		flags := getExtraFlags(obj)
		if flags&(1<<1) == 0 { // not NODROP
			sendToCaster(ch, "It doesn't seem to have any effect.\r\n")
			return
		}
		setExtraFlags(obj, flags &^ (1<<1)) // remove NODROP
		if ot, ok := obj.(objTypeGetter); ok && ot.GetObjType() == 3 { // ITEM_WEAPON
			if vs, ok := obj.(objValSetter); ok {
				if vg, ok := obj.(objValGetter); ok {
					vs.SetObjVal(2, vg.GetObjVal(2)+1)
				}
			}
		}
		sendToCaster(ch, "$p briefly glows blue.\r\n")

	default:
		sendToCaster(ch, " spell not yet implemented.\r\n")
	}
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
	case SpellWordOfRecall:
		castWordOfRecall(level, ch, cvict, world)
	case SpellTeleport:
		castTeleport(level, ch, cvict, world)
	case SpellMeteorSwarm:
		castMeteorSwarm(level, ch, world)
	case SpellHellfire:
		castHellfire(level, ch, world)
	case SpellCharm:
		castCharm(level, ch, cvict, world)
	case SpellSummon:
		castSummon(level, ch, cvict, world)
	case SpellDivineInt:
		castDivineInt(level, ch, world)
	case SpellConjureElemental:
		castConjureElemental(level, ch, world)
	case SpellMindsight:
		castMindsight(level, ch, cvict, world)
	case SpellGate:
		castGate(level, ch, world)
	case SpellLocateObject:
		castLocateObject(level, ch, ovict, world)
	case SpellMirrorImage:
		castMirrorImage(level, ch, world)
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

// boolToInt converts a boolean to 1 (true) or 0 (false).
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
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

// npcRetaliate makes an NPC victim auto-attack the caster on a failed spell save.
// Matches C: if (IS_NPC(victim)) hit(victim, ch, TYPE_UNDEFINED);
func npcRetaliate(victim, ch interface{}) {
	type npcCheck interface{ IsNPC() bool }
	type attacker interface{ StartCombatWith(target string) }
	type namer interface{ GetName() string }
	if nc, ok := victim.(npcCheck); ok && nc.IsNPC() {
		if a, ok := victim.(attacker); ok {
			if n, ok := ch.(namer); ok {
				a.StartCombatWith(n.GetName())
			}
		}
	}
}

// ReagentItem is the interface a reagent object must satisfy.
// Extracted from function-local to package scope for testability.
type ReagentItem interface {
	GetShortDesc() string
}

// ReagentInventory is the interface for an inventory that can find and remove reagents.
type ReagentInventory interface {
	FindItem(string) (ReagentItem, bool)
	RemoveItem(ReagentItem) bool
}

// InventoryHolder is the interface for a caster that has an inventory.
type InventoryHolder interface {
	GetInventory() ReagentInventory
}

func checkReagents(ch interface{}, spellNum, level int, reagents ...string) int {
	_ = spellNum
	if len(reagents) == 0 {
		return 0
	}

	reagentName := reagents[0]

	// Look for and consume the reagent from the caster's inventory.
	// Uses interface assertions to avoid circular imports (spells → game).
	type messageSender interface { SendMessage(string) }

	var found bool
	if holder, ok := ch.(InventoryHolder); ok {
		if item, ok := holder.GetInventory().FindItem(reagentName); ok {
			holder.GetInventory().RemoveItem(item)
			found = true
		}
	}

	if !found {
		return 0
	}

	// Send caster message (second arg if provided)
	if len(reagents) > 1 && reagents[1] != "" {
		if sender, ok := ch.(messageSender); ok {
			sender.SendMessage(reagents[1] + "\r\n")
		}
	}

	// Damage bonus: scales with level, matching C behavior.
	// Original C: bonus is approximately level/2 (varies by spell).
	// metalskin reagent adds flat +1 (C: magic.c:1300 reag is boolean 0/1).
	bonus := level / 2
	if bonus < 1 {
		bonus = 1
	}

	// Check for flat bonus override ("flat:1" in reagent args)
	for _, r := range reagents {
		if r == "flat:1" {
			return 1
		}
	}

	return bonus
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
	setupSpellInfo(SpellWordOfRecall, PosFighting, 50, 50, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellTeleport, PosFighting, 60, 50, 3, RoutineManual, false, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMeteorSwarm, PosFighting, 80, 120, 8, RoutineManual, true, TarIgnore)
	setupSpellInfo(SpellHellfire, PosFighting, 100, 150, 10, RoutineManual, true, TarIgnore)
	setupSpellInfo(SpellCharm, PosFighting, 50, 50, 1, RoutineManual, false, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellSummon, PosFighting, 100, 100, 5, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellDivineInt, PosFighting, 200, 200, 10, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellConjureElemental, PosFighting, 150, 150, 5, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellMindsight, PosFighting, 80, 120, 5, RoutineManual, false, TarCharRoom)

	// --- Routine-based spell registrations (MagAffects/MagDamage/MagPoints/etc.) ---
	// From C source spello() calls. These use the generic framework, not custom cast functions.
	// MAG_AFFECTS spells
	setupSpellInfo(SpellArmor, PosStanding, 15, 30, 3, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellBless, PosStanding, 15, 30, 3, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellBlindness, PosStanding, 25, 50, 2, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellChillTouch, PosFighting, 15, 30, 3, RoutineAffects, false, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDetectInvis, PosStanding, 20, 30, 2, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellDetectMagic, PosStanding, 15, 30, 2, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellHaste, PosFighting, 50, 80, 3, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellSlow, PosFighting, 50, 80, 2, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellInvisible, PosStanding, 25, 50, 4, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellPoison, PosFighting, 25, 50, 3, RoutineAffects, false, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellSanctuary, PosStanding, 50, 80, 4, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellSleep, PosStanding, 25, 50, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellStrength, PosStanding, 20, 35, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellSenseLife, PosStanding, 20, 30, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellInfravision, PosStanding, 20, 40, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellWaterBreathe, PosStanding, 58, 92, 6, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellFly, PosStanding, 80, 100, 5, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellLevitate, PosStanding, 40, 60, 3, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellProtFromEvil, PosStanding, 25, 50, 3, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellProtFromGood, PosStanding, 25, 50, 3, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellMetalskin, PosStanding, 60, 75, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellInvulnerability, PosStanding, 85, 85, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellPsyshield, PosStanding, 20, 30, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellAdrenaline, PosStanding, 30, 35, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellLessPercept, PosStanding, 30, 40, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellGreatPercept, PosStanding, 45, 65, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellChangeDensity, PosStanding, 55, 70, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellTransparency, PosStanding, 25, 35, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellKnowAlign, PosStanding, 20, 20, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellIntellect, PosStanding, 60, 60, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellMindBar, PosStanding, 100, 115, 1, RoutineAffects, true, TarCharRoom)
	setupSpellInfo(SpellSmokescreen, PosFighting, 100, 100, 1, RoutineMasses, true, TarIgnore)
	// MAG_POINTS spells
	setupSpellInfo(SpellCureLight, PosFighting, 10, 20, 2, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellCureCritic, PosFighting, 30, 50, 2, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellHeal, PosFighting, 50, 100, 5, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellVitality, PosFighting, 100, 110, 1, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellInvigorate, PosFighting, 95, 110, 1, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellLayHands, PosStanding, 90, 90, 1, RoutinePoints, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellCellAdjustment, PosFighting, 75, 85, 1, RoutinePoints, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellCureBlind, PosStanding, 25, 50, 2, RoutineUnaffects, false, TarCharRoom)
	setupSpellInfo(SpellRemoveCurse, PosStanding, 25, 50, 2, RoutineUnaffects, false, TarCharRoom)
	setupSpellInfo(SpellRemovePoison, PosStanding, 30, 40, 1, RoutineUnaffects, false, TarCharRoom|TarObjInv|TarObjRoom)
	// MAG_GROUPS spells
	setupSpellInfo(SpellGroupHeal, PosFighting, 150, 210, 5, RoutineGroups, false, TarCharRoom)
	setupSpellInfo(SpellGroupRecall, PosFighting, 125, 155, 5, RoutineGroups, false, TarCharRoom)
	setupSpellInfo(SpellGroupInvis, PosStanding, 50, 100, 5, RoutineGroups, false, TarCharRoom)
	setupSpellInfo(SpellMassHeal, PosFighting, 100, 130, 1, RoutineGroups, false, TarCharRoom)
	// MAG_AREAS spells
	setupSpellInfo(SpellAcidBlast, PosFighting, 20, 35, 1, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellEarthquake, PosFighting, 50, 80, 4, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellFireBreath, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
	// MAG_SUMMONS spells
	setupSpellInfo(SpellAnimateDead, PosStanding, 100, 120, 10, RoutineSummons, false, TarObjRoom)
	// MAG_CREATIONS spells
	setupSpellInfo(SpellCreateFood, PosStanding, 10, 25, 1, RoutineCreations, false, TarIgnore)
	// MAG_DAMAGE spells (registered separately for damage routing)
	setupSpellInfo(SpellMagicMissile, PosFighting, 15, 35, 3, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellBurningHands, PosFighting, 15, 35, 3, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellLightningBolt, PosFighting, 15, 40, 3, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellColorSpray, PosFighting, 15, 45, 3, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellFireball, PosFighting, 15, 60, 3, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDisrupt, PosFighting, 165, 175, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDisintegrate, PosFighting, 120, 120, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDispelEvil, PosFighting, 65, 95, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDispelGood, PosFighting, 65, 95, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellCallLightning, PosFighting, 15, 40, 3, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellHarm, PosFighting, 50, 100, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellSoulLeech, PosFighting, 55, 60, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellEnergyDrain, PosFighting, 50, 100, 2, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellPsiblast, PosFighting, 150, 180, 10, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMindPoke, PosFighting, 15, 30, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMindBlast, PosFighting, 40, 70, 2, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMindAttack, PosFighting, 25, 55, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellFlameStrike, PosFighting, 100, 105, 1, RoutineAffects, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellGate, PosStanding, 95, 95, 1, RoutineManual, true, TarCharWorld|TarNotSelf)
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
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
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
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
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
// C source: spells.c:87-120. Poisons non-water liquid. Drink name helpers are cosmetic.
func castCreateWater(level int, ch, ovict interface{}) {
	_ = level
	if ovict == nil {
		return
	}

	type typeFlagger interface{ GetTypeFlag() int }
	tf, ok := ovict.(typeFlagger)
	if !ok || tf.GetTypeFlag() != 17 { // ITEM_DRINKCON
		sendToCaster(ch, "It's not a drink container.\r\n")
		return
	}

	type valuer interface{ GetValue(int) int }
	v, ok := ovict.(valuer)
	if !ok {
		return
	}

	// Values[0] = max capacity, Values[1] = current amount, Values[2] = liquid type
	liquidType := v.GetValue(2)
	current := v.GetValue(1)

	type drinkSetter interface{ SetValue(int, int) }
	ds, ok := ovict.(drinkSetter)
	if !ok {
		return
	}

	// LIQ_WATER = 0, LIQ_SLIME = 9 (from structs.h)
	if liquidType != 0 && current != 0 {
		// Poison non-water liquid — set to slime
		ds.SetValue(2, 9) // LIQ_SLIME
		sendToCaster(ch, "The water mixes with the liquid...\r\n")
		return
	}

	// Fill with water up to capacity
	maxCap := v.GetValue(0)
	water := maxCap - current
	if water > 0 {
		ds.SetValue(2, 0) // LIQ_WATER
		ds.SetValue(1, current+water)
		sendToCaster(ch, "It is filled.\r\n")
	} else {
		sendToCaster(ch, "You cannot create water in that!\r\n")
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
	type npcChecker interface{ IsNPC() bool }

	// C: identify on NPCs fails and aggros
	if nc, ok := cvict.(npcChecker); ok && nc.IsNPC() {
		sendToCaster(ch, "The magicks fail horribly!\r\n")
		return
	}

	// C: identify on PCs level <= 5 is blocked
	if l, ok := cvict.(lever); ok {
		if nc, ok := cvict.(npcChecker); ok && !nc.IsNPC() && l.GetLevel() <= 5 {
			sendToCaster(ch, "You cannot identify them yet.\r\n")
			return
		}
	}

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

	// Full stat line
	type strGetter interface{ GetStr() int; GetStrAdd() int }
	type intGetter interface{ GetInt() int }
	type wisGetter interface{ GetWis() int }
	type dexGetter interface{ GetDex() int }
	type conGetter interface{ GetCon() int }
	type chaGetter interface{ GetCha() int }

	if sg, ok := cvict.(strGetter); ok {
		str, strAdd := sg.GetStr(), sg.GetStrAdd()
		var strStr string
		if str == 18 && strAdd > 0 {
			strStr = fmt.Sprintf("%d/%d", str, strAdd)
		} else {
			strStr = fmt.Sprintf("%d", str)
		}
		line := fmt.Sprintf("Str: %s", strStr)

		if ig, ok := cvict.(intGetter); ok {
			line += fmt.Sprintf(", Int: %d", ig.GetInt())
		}
		if wg, ok := cvict.(wisGetter); ok {
			line += fmt.Sprintf(", Wis: %d", wg.GetWis())
		}
		if dg, ok := cvict.(dexGetter); ok {
			line += fmt.Sprintf(", Dex: %d", dg.GetDex())
		}
		if cg, ok := cvict.(conGetter); ok {
			line += fmt.Sprintf(", Con: %d", cg.GetCon())
		}
		if chg, ok := cvict.(chaGetter); ok {
			line += fmt.Sprintf(", Cha: %d", chg.GetCha())
		}
		line += "\r\n"
		sendToCaster(ch, line)
	}
}

// --- Room transfer spells ---

// Interfaces for room transfer spells.
type (
	roomGetter2 interface{ GetRoomVNum() int }
	fighter2   interface{ IsFighting() bool }
	hometowner interface{ GetHometown() int }

	worldTransfer interface {
		PlayerTransfer(ch interface{}, toRoomVNum int) error
		MobTransfer(m interface{}, toRoomVNum int) error
		GetRoomInWorld(vnum int) interface { HasFlag(bit int) bool }
		GetRoomCount() int
	}

	// grouper for are_grouped checks
	grouper interface{ IsInGroup() bool; GetFollowing() string; GetName() string }

	// Room AoE interface for meteor_swarm / hellfire
	// charInRoom for AoE iteration — all methods needed by meteor_swarm/hellfire
	charInRoom interface {
		GetName() string
		GetRoomVNum() int
		GetLevel() int
		GetDex() int
		GetMaxHP() int
		SetPosition(int)
		IsInGroup() bool
		GetFollowing() string
	}

	// worldAoe for AoE damage spells — get chars + deal damage
	worldAoe interface {
		GetAllCharsInRoom(roomVNum int) []interface{}
		DoSpellDamage(attacker, victim interface{}, dam int, skill string) bool
		GetRoomInWorld(vnum int) *parser.Room
	}

	npcChecker2 interface{ IsNPC() bool }

	// Interfaces for follower operations via type assertion on interface{}
	// These match signatures in follow.go but use interface{} to avoid circular imports.
	followerWorld interface {
		AddFollowerQuiet(ch interface{}, leader interface{})
		StopFollowerByName(name string)
		CircleFollowByName(followerName, leaderName string) bool
		NumFollowers(leaderName string) int
	}
)

// areGrouped returns true if two characters are in the same group.
// C source: utils.c are_grouped() — checks AFF_GROUP, then walks follower chain.
// Simplified Go: both InGroup + same master or follower relationship.
func areGrouped(ch, victim interface{}) bool {
	cg, ok := ch.(grouper)
	if !ok {
		return false
	}
	vg, ok := victim.(grouper)
	if !ok {
		return false
	}
	if !cg.IsInGroup() || !vg.IsInGroup() {
		return false
	}
	if cg.GetName() == vg.GetName() {
		return true
	}
	chMaster := cg.GetFollowing()
	if chMaster == "" {
		chMaster = cg.GetName()
	}
	vMaster := vg.GetFollowing()
	if vMaster == "" {
		vMaster = vg.GetName()
	}
	if chMaster == vMaster {
		return true
	}
	if chMaster == vg.GetName() || vMaster == cg.GetName() {
		return true
	}
	return false
}

// spell_recall ports src/spells.c spell_recall (lines 124–165).
// Teleports the victim to their hometown. Can't use while fighting or in BFR rooms.
// Unmounts on arrival. Uses room transfer system.
func castWordOfRecall(level int, ch, cvict, world interface{}) {
	_ = level

	// Only works on player victims
	type npcChecker interface{ IsNPC() bool }
	if v, ok := cvict.(npcChecker); ok && v.IsNPC() {
		return
	}
	if cvict == nil {
		return
	}

	w, ok := world.(worldTransfer)
	if !ok {
		sendToCaster(ch, "Recall failed: world interface not available.\r\n")
		return
	}

	// Check BFR flag on caster's room and victim's room
	chRoom := ch.(roomGetter2).GetRoomVNum()
	chRoomData := w.GetRoomInWorld(chRoom)
	if chRoomData != nil && chRoomData.HasFlag(RoomBFR) {
		sendToCaster(ch, "Your magic ebbs and dissolves as you lose your concentration.\r\n")
		return
	}
	victRoom := cvict.(roomGetter2).GetRoomVNum()
	victRoomData := w.GetRoomInWorld(victRoom)
	if victRoomData != nil && victRoomData.HasFlag(RoomBFR) {
		sendToVictim(cvict, "Your magic ebbs and dissolves as you lose your concentration.\r\n")
		return
	}

	// Can't recall while fighting
	if f, ok := ch.(fighter2); ok && f.IsFighting() {
		sendToCaster(ch, "Your concentration is broken by your fighting!\r\n")
		return
	}

	// Determine hometown room
	var destRoom int
	if ht, ok := cvict.(hometowner); ok {
		switch ht.GetHometown() {
		case 1:
			destRoom = KiroshiStartRoom
		case 3:
			destRoom = AlaozarStartRoom
		default:
			destRoom = MortalStartRoom
		}
	} else {
		destRoom = MortalStartRoom
	}

	// Transfer the victim
	sendToVictim(cvict, "You feel a brief tingling sensation...\r\n")
	if err := w.PlayerTransfer(cvict, destRoom); err != nil {
		sendToCaster(ch, fmt.Sprintf("Recall failed: %s\r\n", err))
		return
	}

	sendToVictim(cvict, "You have a strange dream about falling..\r\n")
}

// spell_teleport ports src/spells.c spell_teleport (lines 168–217).
// Random room teleport. Self-only for PCs. NPCs get saving throw.
// Can't use in peaceful rooms. Avoids PRIVATE rooms.
func castTeleport(level int, ch, cvict, world interface{}) {
	_ = level

	w, ok := world.(worldTransfer)
	if !ok {
		sendToCaster(ch, "Teleport failed: world interface not available.\r\n")
		return
	}

	if cvict == nil {
		sendToCaster(ch, "Who do you want this done to?\r\n")
		return
	}

	// Check peaceful room
	chRoom := ch.(roomGetter2).GetRoomVNum()
	chRoomData := w.GetRoomInWorld(chRoom)
	if chRoomData != nil && chRoomData.HasFlag(RoomPeaceful) {
		sendToCaster(ch, "The gods deny thy magick.\r\n")
		return
	}

	// PCs can only teleport self
	type npcChecker interface{ IsNPC() bool }
	if chCaster, ok := ch.(npcChecker); ok && !chCaster.IsNPC() {
		type namer interface{ GetName() string }
		chName := ch.(namer).GetName()
		victName := cvict.(namer).GetName()
		if chName != victName {
			sendToCaster(ch, "You can only will this power upon yourself!\r\n")
			return
		}
	}

	// NPCs get a saving throw
	if vNPC, ok := cvict.(npcChecker); ok && vNPC.IsNPC() {
		if magSavingThrow(cvict, int(SaveSpell)) {
			sendToCaster(ch, "The magic words fail to form properly.\r\n")
			return
		}
	}

	// Pick a random room, avoiding PRIVATE
	roomCount := w.GetRoomCount()
	for attempts := 0; attempts < 100; attempts++ {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		toRoom := rand.Intn(roomCount)
		roomData := w.GetRoomInWorld(toRoom)
		if roomData != nil && !roomData.HasFlag(RoomPrivate) {
			sendToCaster(ch, "The world around you turns black and you suddenly find yourself..\r\n")
			sendToVictim(cvict, "The world around you turns black and you suddenly find yourself..\r\n")

			// Transfer — use CharTransfer via appropriate path
			if vNPC, ok := cvict.(npcChecker); ok && vNPC.IsNPC() {
				if err := w.MobTransfer(cvict, toRoom); err != nil {
					slog.Error("MobTransfer failed", "error", err)
				}
			} else {
				if err := w.PlayerTransfer(cvict, toRoom); err != nil {
					slog.Error("PlayerTransfer failed", "error", err)
				}
			}
			return
		}
	}

	sendToCaster(ch, "The magic fails to find a destination.\r\n")
}

// castMeteorSwarm ports src/spells.c spell_meteor_swarm (lines 1088-1133).
// AoE damage to all non-grouped characters in room. Must be outdoors.
func castMeteorSwarm(level int, ch, world interface{}) {
	w, ok := world.(worldAoe)
	if !ok {
		sendToCaster(ch, "The magic fizzles.\r\n")
		return
	}

	chRG, _ := ch.(roomGetter2)
	if chRG == nil {
		return
	}
	roomVNum := chRG.GetRoomVNum()

	// Peaceful room check
	roomData := w.GetRoomInWorld(roomVNum)
	if roomData != nil && roomData.HasFlag(RoomPeaceful) {
		sendToCaster(ch, "This room just has such a peaceful, easy feeling..\r\n")
		return
	}

	// OUTSIDE check: not INDOORS flag OR sector != INSIDE
	if roomData != nil && roomData.HasFlag(3) && roomData.Sector == 0 {
		sendToCaster(ch, "You can only do this outdoors!\r\n")
		return
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	dam := level*6 + rand.Intn(level*3+11) - 10
	if dam < 1 {
		dam = 1
	}

	sendToCaster(ch, "Your incantation brings the heavens down upon your victims!\r\n")

	// Damage all non-grouped, non-immort, non-caster chars in room
	casterName := ""
	if cn, ok := ch.(interface{ GetName() string }); ok {
		casterName = cn.GetName()
	}
	casterLevel := 0
	if cl, ok := ch.(interface{ GetLevel() int }); ok {
		casterLevel = cl.GetLevel()
	}

	chars := w.GetAllCharsInRoom(roomVNum)
	for _, c := range chars {
		if c == ch {
			continue
		}
		cn, _ := c.(charInRoom)
		if cn == nil {
			continue
		}
		// Skip immortals (non-NPCs level >= 100)
		if nc, ok := c.(npcChecker2); ok && !nc.IsNPC() && cn.GetLevel() >= 100 {
			continue
		}
		// Skip grouped
		if areGrouped(ch, c) {
			continue
		}
		w.DoSpellDamage(ch, c, dam, "meteor swarm")
	}

	_ = casterName
	_ = casterLevel
}

// castHellfire ports src/spells.c spell_hellfire (lines 701-767).
// AoE fire damage. DEX check can knock targets to sitting.
// C bug: iterates character_list instead of room people (inefficient but functionally same).
func castHellfire(level int, ch, world interface{}) {
	w, ok := world.(worldAoe)
	if !ok {
		sendToCaster(ch, "The magic fizzles.\r\n")
		return
	}

	chRG, _ := ch.(roomGetter2)
	if chRG == nil {
		return
	}
	roomVNum := chRG.GetRoomVNum()

	// Peaceful room check
	roomData := w.GetRoomInWorld(roomVNum)
	if roomData != nil && roomData.HasFlag(RoomPeaceful) {
		sendToCaster(ch, "This room just has such a peaceful, easy feeling..\r\n")
		return
	}

	dam := dice(level/5+1, 12) + (2*level) - 10
	if dam < 1 {
		dam = 1
	}

	sendToCaster(ch, "The bowels of hell open beneath your feet!!\r\n")

	// Dice function
	// dice(num, sides) — need this
	chars := w.GetAllCharsInRoom(roomVNum)
	for _, c := range chars {
		if c == ch {
			continue
		}
		cn, ok := c.(charInRoom)
		if !ok {
			continue
		}
		// Skip immortals
		if nc, ok := c.(npcChecker2); ok && !nc.IsNPC() && cn.GetLevel() >= 100 {
			continue
		}
		// Skip grouped
		if areGrouped(ch, c) {
			continue
		}

		// C source: spells.c:729-756 — level <= 4 takes GET_MAX_HIT*12 (instant kill)
		if cn.GetLevel() <= 4 {
			if vn, ok := c.(interface{ SendMessage(string) }); ok {
				vn.SendMessage("The fires of hell overcome you!!\r\n")
			}
			w.DoSpellDamage(ch, c, cn.GetMaxHP()*12, "hellfire")
		} else {
			// Level > 4: normal damage
			if vn, ok := c.(interface{ SendMessage(string) }); ok {
				vn.SendMessage("The fires of hell bring blisters on your skin!\r\n")
			}
			w.DoSpellDamage(ch, c, dam, "hellfire")
		}
	}
}



// castCharm ports src/spells.c spell_charm (lines 407-476).
// Charms a mob to follow the caster. Checks: MOB_NOCHARM, level, circle follow,
// saving throw, max followers (CHA/2), PC-victims blocked.
func castCharm(level int, ch, cvict, world interface{}) {
	if cvict == nil || ch == nil {
		return
	}

	type namer interface{ GetName() string }
	type lever interface{ GetLevel() int }
	type intGetter interface{ GetInt() int }
	type npcCheck interface{ IsNPC() bool }

	chName := ""
	if n, ok := ch.(namer); ok {
		chName = n.GetName()
	}
	victName := ""
	if n, ok := cvict.(namer); ok {
		victName = n.GetName()
	}

	if chName == victName {
		sendToCaster(ch, "You like yourself even better!\r\n")
		return
	}

	victIsNPC := true
	if nc, ok := cvict.(npcCheck); ok {
		victIsNPC = nc.IsNPC()
	}

	// MOB_NOCHARM check on mob victim
	if victIsNPC {
		type mobFlagger interface{ HasMobFlag(flag uint64) bool }
		if mf, ok := cvict.(mobFlagger); ok && mf.HasMobFlag(1<<4) {
			sendToCaster(ch, "Your victim resists!\r\n")
			return
		}
	}

	// Level check
	if victL, ok := cvict.(lever); ok {
		if level < victL.GetLevel() {
			sendToCaster(ch, "You fail.\r\n")
			return
		}
	}

	// Circle follow check
	if fw, ok := world.(followerWorld); ok {
		if fw.CircleFollowByName(victName, chName) {
			sendToCaster(ch, "Sorry, following in circles can not be allowed.\r\n")
			return
		}
	}

	// Saving throw
	if victIsNPC && magSavingThrow(cvict, int(SaveParalysis)) {
		sendToCaster(ch, "Your victim resists!\r\n")
		return
	}

	// Max followers check (CHA/2)
	if fw, ok := world.(followerWorld); ok {
		chCha := 10
		if ci, ok := ch.(intGetter); ok {
			chCha = ci.GetInt()
		}
		if fw.NumFollowers(chName) >= chCha/2 {
			sendToCaster(ch, "You can't have any more followers!\r\n")
			return
		}
	}

	// No charming PCs
	if !victIsNPC {
		sendToCaster(ch, "You can't charm other players!\r\n")
		return
	}

	// Stop victim's current following, add as our follower
	if fw, ok := world.(followerWorld); ok {
		fw.StopFollowerByName(victName)
		fw.AddFollowerQuiet(cvict, ch)
	}

	// C source: spells.c:448-464 — apply AFF_CHARM affect with duration
	// Duration = 24 * 18 / GET_INT(victim) (zero-guard: 24 * 18)
	dur := 24 * 18
	if vi, ok := cvict.(intGetter); ok && vi.GetInt() > 0 {
		dur = dur / vi.GetInt()
	}
	applyAffect(cvict, engine.NewAffectDirect(SpellCharm, engine.ApplyNone, dur, 0, engine.AFFCharm, "charm"))

	// C source: spells.c:462-464 — remove MOB_AGGRESSIVE and MOB_SPEC from charmed mob
	// MobFlagAggressive = 5, MobFlagSpec = 0 (from pkg/game/mob_flags_bits.go)
	type mobFlagOps interface {
		HasMobFlag(int) bool
		ClearMobFlag(int)
	}
	if mf, ok := cvict.(mobFlagOps); ok {
		if mf.HasMobFlag(5) { // MOB_AGGRESSIVE
			mf.ClearMobFlag(5)
		}
		if mf.HasMobFlag(0) { // MOB_SPEC
			mf.ClearMobFlag(0)
		}
	}

	sendToCaster(ch, "They are now your loyal servant.\r\n")
}

// castSummon ports src/spells.c spell_summon (lines 220-355).
// Summons victim to caster's room. Complex: circle check, PRF flags, MOB_NOSUMMON,
// room exit check, saving throw with backfire, room transfer with mount handling.
func castSummon(level int, ch, cvict, world interface{}) {
	if ch == nil || cvict == nil {
		return
	}

	type namer interface{ GetName() string }
	type lever interface{ GetLevel() int }
	type roomGet interface{ GetRoomVNum() int }
	type npcCheck interface{ IsNPC() bool }
	type mobFlagger interface{ HasMobFlag(flag uint64) bool }

	chName := ""
	if n, ok := ch.(namer); ok {
		chName = n.GetName()
	}
	victName := ""
	if n, ok := cvict.(namer); ok {
		victName = n.GetName()
	}
	chRoom := 0
	if rg, ok := ch.(roomGet); ok {
		chRoom = rg.GetRoomVNum()
	}
	victRoom := 0
	if rg, ok := cvict.(roomGet); ok {
		victRoom = rg.GetRoomVNum()
	}

	chIsNPC := false
	if nc, ok := ch.(npcCheck); ok {
		chIsNPC = nc.IsNPC()
	}
	victIsNPC := false
	if nc, ok := cvict.(npcCheck); ok {
		victIsNPC = nc.IsNPC()
	}

	// Level check: can't summon someone > level+3
	if vl, ok := cvict.(lever); ok {
		victLevel := vl.GetLevel()
		if victLevel > level+3 {
			sendToCaster(ch, "You failed.\r\n")
			if vs, ok := cvict.(interface{ SendMessage(string) }); ok {
				vs.SendMessage(fmt.Sprintf("%s just tried to summon you but failed.\r\n", chName))
			}
			return
		}
	}

	// MOB_NOSUMMON check
	if victIsNPC {
		if mf, ok := cvict.(mobFlagger); ok && mf.HasMobFlag(1<<12) { // MOB_NOSUMMON
			sendToCaster(ch, "You failed.\r\n")
			return
		}
	}

	// MOB_NOCHARM check (C checks this in summon too)
	if victIsNPC {
		if mf, ok := cvict.(mobFlagger); ok && mf.HasMobFlag(1<<4) { // MOB_NOCHARM
			sendToCaster(ch, "You failed.\r\n")
			return
		}
	}

	// Peaceful room check for NPC victims
	type worldRoom interface{ GetRoomInWorld(vnum int) *parser.Room }
	if wr, ok := world.(worldRoom); ok {
		chRoomData := wr.GetRoomInWorld(chRoom)
		if chRoomData != nil && chRoomData.HasFlag(RoomPeaceful) && victIsNPC {
			// Fail silently for peaceful room + NPC victim
			sendToCaster(ch, "You failed.\r\n")
			return
		}
		// NOMAGIC check on victim's room
		victRoomData := wr.GetRoomInWorld(victRoom)
		if victRoomData != nil && victRoomData.HasFlag(RoomNoMagic) {
			sendToCaster(ch, "You failed.\r\n")
			return
		}
	}

	// Room exit check: caster's room must have at least one open exit
	type worldExits interface{ GetRoomInWorld(vnum int) *parser.Room }
	roomOK := false
	if we, ok := world.(worldExits); ok {
		roomData := we.GetRoomInWorld(chRoom)
		if roomData != nil && len(roomData.Exits) > 0 {
			roomOK = true
		}
	}
	if !roomOK {
		sendToCaster(ch, "You failed.\r\n")
		return
	}

	// Saving throw with backfire
	if victIsNPC && magSavingThrow(cvict, int(SaveSpell)) {
		// 10% backfire chance for PC casters
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if !chIsNPC && rand.Intn(10) == 0 {
			sendToCaster(ch, "Your spell backfires!\r\n")
			// Transfer caster to victim's room instead
			type transferWorld interface{ PlayerTransfer(ch interface{}, toRoomVNum int) error; MobTransfer(ch interface{}, toRoomVNum int) error }
			if tw, ok := world.(transferWorld); ok {
				if chIsNPC {
					if err := tw.MobTransfer(ch, victRoom); err != nil {
						slog.Error("MobTransfer failed", "error", err)
					}
				} else {
					if err := tw.PlayerTransfer(ch, victRoom); err != nil {
						slog.Error("PlayerTransfer failed", "error", err)
					}
				}
			}
			return
		}
		sendToCaster(ch, "You failed.\r\n")
		if vs, ok := cvict.(interface{ SendMessage(string) }); ok {
			vs.SendMessage(fmt.Sprintf("%s just tried to summon you but failed.\r\n", chName))
		}
		return
	}

	// Success — transfer victim to caster's room
	type transferWorld interface{ PlayerTransfer(ch interface{}, toRoomVNum int) error; MobTransfer(ch interface{}, toRoomVNum int) error }
	if tw, ok := world.(transferWorld); ok {
		if victIsNPC {
			if err := tw.MobTransfer(cvict, chRoom); err != nil {
				slog.Error("MobTransfer failed", "error", err)
			}
		} else {
			if err := tw.PlayerTransfer(cvict, chRoom); err != nil {
				slog.Error("PlayerTransfer failed", "error", err)
			}
		}
	}

	if vs, ok := cvict.(interface{ SendMessage(string) }); ok {
		vs.SendMessage(fmt.Sprintf("%s has summoned you!\r\n", chName))
	}
	sendToCaster(ch, fmt.Sprintf("%s has been summoned.\r\n", victName))
}

// castDivineInt ports src/spells.c spell_divine_int (lines 1170-1215).
// Spawns an angel mob based on caster's alignment. Good→85, Evil→86, Neutral fails.
// Saving throw determines success. Alignment extremes (±1000) spawn 2 angels.
func castDivineInt(level int, ch, world interface{}) {
	if ch == nil {
		return
	}

	type aligner interface{ GetAlignment() int }
	type roomGet interface{ GetRoomVNum() int }

	alignment := 0
	if a, ok := ch.(aligner); ok {
		alignment = a.GetAlignment()
	}

	// Neutral alignment fails
	if alignment == 0 {
		sendToCaster(ch, "Your request for intervention falls on deaf ears.\r\n")
		return
	}

	// Saving throw — caster must FAIL for it to work (C quirk: !mag_savingthrow = success)
	// Actually re-reading C: if (!mag_savingthrow(ch, SAVING_SPELL)) → caster failed save → nothing happens
	// So if caster saves, the spell works. If caster fails, nothing happens.
	// Wait, let me re-read: "if (!mag_savingthrow(ch, SAVING_SPELL)) { stc('Nothing seems to happen.'); return; }"
	// So if saving throw FAILS (returns false), nothing happens. If saving throw SUCCEEDS (returns true), proceed.
	// That's backwards from usual. This is a C quirk — divine intervention requires the caster to resist.
	if !magSavingThrow(ch, int(SaveSpell)) {
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}

	// Determine angel type
	mobVNum := 86 // default evil angel
	if alignment > 0 {
		mobVNum = 85 // good angel
	}

	roomVNum := 0
	if rg, ok := ch.(roomGet); ok {
		roomVNum = rg.GetRoomVNum()
	}
	if roomVNum <= 0 {
		sendToCaster(ch, "You can't do that here.\r\n")
		return
	}

	number := 1
	if alignment == -1000 || alignment == 1000 {
		number = 2
	}

	sendToCaster(ch, "You pray for the intervention of your deity.\r\n")
	sendToCaster(ch, "Suddenly, a portal of light appears out of nowhere!\r\n")

	type mobSpawner interface {
		SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error)
	}
	spawner, ok := world.(mobSpawner)
	if !ok {
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	type followerAdder interface{ AddFollowerQuiet(ch, leader interface{}) }
	fa, _ := world.(followerAdder)

	for i := 0; i < number; i++ {
		mobLevel := level / 2
		if mobLevel < 1 {
			mobLevel = 1
		}
		mob, err := spawner.SpawnMobWithLevelI(mobVNum, roomVNum, mobLevel)
		if err != nil {
			slog.Warn("spell_divine_int: failed to spawn angel", "error", err, "vnum", mobVNum)
			continue
		}
		if fa != nil {
			fa.AddFollowerQuiet(mob, ch)
		}
	}
}

// castConjureElemental ports src/spells.c spell_conjure_elemental (lines 1039-1086).
// Conjures an elemental by consuming a component object from the room.
// Components: earth(81→81), water(82→82), wind(83→83), fire(84→84).
// The elemental mob VNUM matches the component VNUM.
func castConjureElemental(level int, ch, world interface{}) {
	if ch == nil {
		return
	}

	type roomGet interface{ GetRoomVNum() int }
	roomVNum := 0
	if rg, ok := ch.(roomGet); ok {
		roomVNum = rg.GetRoomVNum()
	}
	if roomVNum <= 0 {
		sendToCaster(ch, "You can't do that here.\r\n")
		return
	}

	// Elemental components: [mobVNum, componentVNum]
	elemComponents := [][2]int{
		{81, 81}, // earth
		{82, 82}, // water
		{83, 83}, // wind
		{84, 84}, // fire
	}

	// Find a component in the room
	type itemChecker interface{ GetVNum() int; GetName() string }
	type roomItems interface{ GetItemsInRoomI(roomVNum int) []interface{} }

	ri, ok := world.(roomItems)
	if !ok {
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	var component interface{}
	var componentVNum int
	var mobVNum int

	for _, ec := range elemComponents {
		items := ri.GetItemsInRoomI(roomVNum)
		for _, item := range items {
			if ic, ok := item.(itemChecker); ok && ic.GetVNum() == ec[1] {
				component = item
				componentVNum = ec[1]
				mobVNum = ec[0]
				break
			}
		}
		if component != nil {
			break
		}
	}

	if component == nil {
		sendToCaster(ch, "You begin to chant, but nothing seems to happen.\r\n")
		return
	}

	compName := ""
	if ic, ok := component.(itemChecker); ok {
		compName = ic.GetName()
	}
	sendToCaster(ch, fmt.Sprintf("You begin to chant slowly, drawing power from %s.\r\n", compName))

	// Spawn the elemental
	mobLevel := level/2 + 3
	type mobSpawner interface {
		SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error)
	}
	spawner, ok := world.(mobSpawner)
	if !ok {
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	mob, err := spawner.SpawnMobWithLevelI(mobVNum, roomVNum, mobLevel)
	if err != nil {
		slog.Warn("spell_conjure_elemental: failed to spawn", "error", err, "vnum", mobVNum)
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	// Add as follower
	type followerAdder interface{ AddFollowerQuiet(ch, leader interface{}) }
	if fa, _ := world.(followerAdder); ok {
		fa.AddFollowerQuiet(mob, ch)
	}

	// Extract the component
	type extractor interface{ ExtractFromRoom(roomVNum int) }
	if ex, ok := component.(extractor); ok {
		ex.ExtractFromRoom(roomVNum)
	}

	sendToCaster(ch, "An elemental appears before you!\r\n")
	_ = componentVNum
}


// castMindsight ports src/spells.c spell_mindsight (lines 912-955).
func castMindsight(level int, ch, cvict, world interface{}) {
	_ = level
	if cvict == nil || ch == nil {
		return
	}

	type lever interface{ GetLevel() int }
	type roomGet interface{ GetRoomVNum() int }
	type npcCheck interface{ IsNPC() bool }

	chRoom := 0
	if rg, ok := ch.(roomGet); ok {
		chRoom = rg.GetRoomVNum()
	}

	victIsNPC := false
	if nc, ok := cvict.(npcCheck); ok {
		victIsNPC = nc.IsNPC()
	}

	// Level resist check
	victLevel := 0
	if vl, ok := cvict.(lever); ok {
		victLevel = vl.GetLevel()
	}
	casterLevel := 0
	if cl, ok := ch.(lever); ok {
		casterLevel = cl.GetLevel()
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if (victLevel > casterLevel+4 && rand.Intn(5) == 0) ||
		(!victIsNPC && victLevel >= 100 && casterLevel <= victLevel) {
		sendToCaster(ch, "With a searing pain, your psionic energy recoils!\r\n")
		return
	}

	victRoom := 0
	if rg, ok := cvict.(roomGet); ok {
		victRoom = rg.GetRoomVNum()
	}
	if victRoom <= 0 {
		return
	}

	// Transfer caster to victim's room, look, transfer back
	type transferWorld interface {
		PlayerTransfer(ch interface{}, toRoomVNum int) error
		LookAtRoomSimple(roomVNum int, sender interface{})
	}
	tw, ok := world.(transferWorld)
	if !ok {
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	if err := tw.PlayerTransfer(ch, victRoom); err != nil {
		slog.Error("PlayerTransfer failed", "error", err)
	}
	tw.LookAtRoomSimple(victRoom, ch)
	if err := tw.PlayerTransfer(ch, chRoom); err != nil {
		slog.Error("PlayerTransfer failed", "error", err)
	}
	sendToCaster(ch, "You have a strange dream about seeing...\r\n")
}

// castGate creates a red portal in one of the 8 legal rooms.
// If a portal already exists, the caster dies from cosmic energy.
// If the room is not legal, nothing happens.
func castGate(level int, ch, world interface{}) {
	_ = level
	if ch == nil {
		return
	}

	type roomGetter interface{ GetRoomVNum() int }
	rg, ok := ch.(roomGetter)
	if !ok {
		return
	}
	roomVNum := rg.GetRoomVNum()

	// Check if portal already exists in room
	type roomObjects interface{ GetObjectsInRoom(roomVNum int) []interface{} }
	if wo, ok := world.(roomObjects); ok {
		objs := wo.GetObjectsInRoom(roomVNum)
		for _, obj := range objs {
			type objVnum interface{ GetVNum() int }
			if ov, ok := obj.(objVnum); ok {
				// Red portal vnum 10, Blue portal vnum 11 (from gate.c)
				if ov.GetVNum() == 4002 || ov.GetVNum() == 4001 {
					sendToCaster(ch, "The magick flows through you, then out into the world, changing it....\r\n")
					sendToRoom("The fabric of time and space warps and stretches around you...\r\n", ch, nil, nil, "", "", world)
					sendToCaster(ch, "In your final moments, the only thing you can feel is a wave of cosmic energy coursing through you, tearing your soul to shreds.\r\n")
					// Kill the caster
					type killer interface{ Die() }
					if k, ok := ch.(killer); ok {
						k.Die()
					}
					return
				}
			}
		}
	}

	// Legal rooms for gate creation (from gate.c)
	legalRooms := map[int]bool{4001: true, 4002: true, 4003: true, 4004: true, 4005: true, 4006: true, 4007: true, 4008: true}
	if !legalRooms[roomVNum] {
		sendToCaster(ch, "The magic flows through you, but nothing else happens.\r\n")
		return
	}

	// Create red portal in room
	type objSpawner interface{ SpawnObject(vnum, roomVNum int) (interface{}, error) }
	if wo, ok := world.(objSpawner); ok {
		_, err := wo.SpawnObject(4002, roomVNum) // red_portal = 4002
		if err != nil {
			sendToCaster(ch, "The magic flows through you, but nothing else happens.\r\n")
			return
		}
		sendToCaster(ch, "The magick flows through you, then out into the world, changing it....\r\n")
		sendToRoom("A shimmering red portal fades into existence.\r\n", ch, nil, nil, "", "", world)
	}
}

// castLocateObject finds objects matching the arg name in the world.
func castLocateObject(level int, ch, ovict, world interface{}) {
	if ch == nil {
		return
	}

	// Use ovict (target object) to get the name to search for
	type objName interface{ GetName() string }
	if ovict == nil {
		sendToCaster(ch, "What object are you looking for?\r\n")
		return
	}
	name := ""
	if on, ok := ovict.(objName); ok {
		name = on.GetName()
	}
	if name == "" {
		sendToCaster(ch, "What object are you looking for?\r\n")
		return
	}

	type worldSearch interface{ FindObjectByName(name string) []interface{} }
	if ws, ok := world.(worldSearch); ok {
		objs := ws.FindObjectByName(name)
		count := level >> 1
		for _, obj := range objs {
			if count <= 0 {
				break
			}
			type objDesc interface{ GetShortDesc() string }
			if od, ok := obj.(objDesc); ok {
				sendToCaster(ch, od.GetShortDesc()+"\r\n")
			}
			count--
		}
	}
}

// castMirrorImage creates a clone of the caster.
func castMirrorImage(level int, ch, world interface{}) {
	if ch == nil {
		return
	}

	type roomGetter interface{ GetRoomVNum() int }
	rg, ok := ch.(roomGetter)
	if !ok {
		return
	}
	roomVNum := rg.GetRoomVNum()

	type mobSpawner interface {
		SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error)
	}
	if ms, ok := world.(mobSpawner); ok {
		// MOB_CLONE vnum — check C source
		mob, err := ms.SpawnMobWithLevelI(69, roomVNum, level) // MOB_CLONE = 69
		if err != nil {
			sendToCaster(ch, "You fail to divide yourself.\r\n")
			return
		}

		// Name the clone after the caster
		type named interface{ GetName() string }
		if n, ok := ch.(named); ok {
			type nameSetter interface{ SetName(string) }
			if ns, ok := mob.(nameSetter); ok {
				ns.SetName(n.GetName())
			}
		}

		sendToCaster(ch, "You divide yourself in two!\r\n")
		sendToRoom("$n divides $mself in two!\r\n", ch, nil, nil, "", "", world)
	}
}

