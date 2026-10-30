package spells

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// MagAffects applies spell affects to a character.
// Functions named MagXxx to match C convention; constants are RoutineXxx.
// magAffectsSaveRoll performs the saving throw for the hostile mag_affects arms.
// It is a package var so tests can force the save result deterministically:
// CheckSavingThrow is RNG-backed (math/rand/v2's global generator has no Seed),
// so asserting a specific outcome through a raw call flakes — e.g. a level-30
// victim vs a level-30 caster saves often enough that a bounded retry loop can
// lose every attempt.
var magAffectsSaveRoll = magSavingThrow

// magSummonSaveRoll keeps summon save branches deterministic in focused tests.
var magSummonSaveRoll = magSavingThrow

// magSummonOutlawResistanceRoll and magSummonBackfireRoll preserve the C
// summon draws while letting focused tests select each branch deterministically.
var (
	magSummonOutlawResistanceRoll = func() int { return dprng.Number(1, 100) }
	magSummonBackfireRoll         = func() int { return dprng.Number(0, 9) }
)

// magCharmSaveRoll keeps charm's C saving-throw branch deterministic in focused tests.
var magCharmSaveRoll = magSavingThrow

// magDivineIntSaveRoll gives the manual intervention branch deterministic
// success/failure coverage without changing the production saving-throw path.
var magDivineIntSaveRoll = magSavingThrow

func MagAffects(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	magAffectsForCast(level, ch, victim, spellNum, savetype, CastSpell, world)
}

// magAffectsForCast carries the C object-cast type to the final affect handler.
func magAffectsForCast(level int, ch, victim interface{}, spellNum, savetype int, castType CastType, world interface{}) {
	if victim == nil || ch == nil {
		return
	}

	// Pre-compute reagent bonuses before applying affects so the deterministic
	// apply path can be tested independently.
	reag := 0
	switch spellNum {
	case SpellBlindness, SpellSmokescreen:
		if isClassMage(ch) && consumeSpellReagentVNum(ch, 1234) {
			reag = 1
			sendToCaster(ch, "You crush a small lens under your heel.\r\n")
			sendAffectActRoom(ch, nil, "$n crushes a small lens under $s heel.", true, world)
		}
	case SpellWaterwalk:
		reag = checkReagents(ch, SpellWaterwalk, getLevel(ch), "frog leg",
			"A frog leg turns into dust as you mouth the words of the spell...")
	case SpellMetalskin:
		reag = checkReagents(ch, SpellMetalskin, getLevel(ch), "chunk of iron",
			"A small chunk of iron melts in your palm as you cast the spell...", "flat:1")
		if reag > 0 {
			sendAffectActRoom(ch, nil, "A small chunk of iron melts in $n's palm as $e cast a spell...", true, world)
		}
	case SpellSleep:
		// C magic.c:1203-1210 consumes the exact vnum-1226 sand component
		// before the outlaw/level gates or save roll. Keep the two audience
		// messages in that order and retain the reagent for the duration.
		if consumeSpellReagentVNum(ch, 1226) {
			reag = 1
			sendToCaster(ch, "Pulling a bit of sand from a pocket, you cast it about the room...\r\n")
			sendAffectRoom(ch, nil, "$n pulls a bit of sand out of a pocket and casts it about the room.\r\n", true, world)
		} else {
			sendToCaster(ch, "You attempt the spell without the components...\r\n")
		}
	}

	// C src/magic.c:1212-1227 applies these gates after reagent narration
	// and before mag_savingthrow. They also apply to potion self-targets:
	// src/spell_parser.c:699-718 enters call_magic without cast_spell's
	// TAR_NOT_SELF check at src/spell_parser.c:886.
	if spellNum == SpellSleep && !isNPC(victim) {
		if !hasOutlawFlag(ch) {
			sendToCaster(ch, "Your spell fails to affect them because you are not an Outlaw!\r\n")
			casterName := "Someone"
			if named, ok := ch.(interface{ GetName() string }); ok {
				casterName = named.GetName()
			}
			sendToVictim(victim, fmt.Sprintf("%s tried to cast a spell on you but failed because %s is not an Outlaw!\r\n", casterName, casterName))
			return
		}
		if getLevel(ch) < lvlImmort && absInt(getLevel(victim)-getLevel(ch)) > 3 {
			sendAffectRoom(victim, nil, "$n shakes his head wearily, but then snaps out of it!\r\n", true, world)
			return
		}
	}

	// C only rolls a save inside the hostile cases that explicitly request it.
	// Benign affects such as infravision consume no gameplay RNG.
	saved := false
	switch spellNum {
	case SpellBlindness, SpellSmokescreen:
		// magic.c:938's first disjunct short-circuits the save roll: a
		// MOB_NOBLIND victim (action flag bit 17) refuses without drawing.
		if hasSummonMobFlag(victim, 17) {
			saved = true
		} else {
			saved = magAffectsSaveRoll(victim, savetype)
		}
	case SpellSleep:
		// magic.c:1227's first disjunct short-circuits the save roll: a
		// MOB_NOSLEEP victim (action flag bit 15) refuses without drawing.
		if hasSummonMobFlag(victim, 15) {
			saved = true
		} else {
			saved = magAffectsSaveRoll(victim, savetype)
		}
	case SpellChillTouch, SpellCurse, SpellFlameStrike, SpellPoison:
		saved = magAffectsSaveRoll(victim, savetype)
	}
	// C magic.c:967-982 checks CURSE's save before consuming its mage reagent.
	// Blindness consumes its lens before the save; curse only consumes a
	// feather on the successful affect path.
	if spellNum == SpellCurse && !saved && isClassMage(ch) && consumeSpellReagentVNum(ch, 1233) {
		reag = 1
		sendToCaster(ch, "You throw a raven's feather into the air...\r\n")
		sendAffectActRoom(ch, nil, "$n throws a feather into the air...", true, world)
	}
	magAffectsApply(level, ch, victim, spellNum, saved, reag, world, castType)
}

// magAffectsApply is the deterministic core of MagAffects. It applies the spell
// affects for a given spell number assuming the saving throw result and reagent
// bonus are already known. This makes the C affect table testable without RNG.
// Messages follow C magic.c: each case only selects its to_vict/to_room/to_self
// strings; the send block at the bottom reproduces the C dispatch order
// (magic.c:1414-1421): to_self fires only when ch != victim, then to_vict,
// then to_room.
func magAffectsApply(level int, ch, victim interface{}, spellNum int, saved bool, reag int, world interface{}, castTypes ...CastType) {
	var aff *engine.Affect
	var toVictim, toRoom, toSelf string
	var pending []*engine.Affect
	// C magic.c:1421's aggro tail: an applied hostile affect whose case set
	// aggro=TRUE (poison, flamestrike) makes an NPC victim attack the caster
	// after the message block.
	aggroSpell := false

	switch spellNum {
	case SpellChillTouch:
		dur := 4
		if saved {
			dur = 1
		}
		aff = engine.NewAffect(SpellChillTouch, engine.ApplyStr, dur, -1, "chill touch")
		toVictim = "You feel your strength wither!\r\n"
		toSelf = "Summoning the forces of magick, you press your icy hand against $M.\r\n"
	case SpellBless:
		pending = append(pending, engine.NewAffect(SpellBless, engine.ApplyHitroll, 6, 2, "bless"), engine.NewAffect(SpellBless, engine.ApplySavingSpell, 6, -2, "bless"))
		toVictim = "You feel righteous.\r\n"
		toSelf = "You bestow the blessing of your gods on $M.\r\n"
	case SpellArmor:
		aff = engine.NewAffect(SpellArmor, engine.ApplyAC, 24, -15, "armor")
		toVictim = "You feel someone protecting you.\r\n"
		toSelf = "The magick protects $M.\r\n"
	case SpellBlindness, SpellSmokescreen:
		if saved {
			if spellNum == SpellBlindness {
				sendToCaster(ch, "Your magic fades, then dies out totally.\r\n")
			}
			npcRetaliate(world, victim, ch)
			return
		}
		// C types both affects with spellnum and gives both AFF_BLIND:
		// hitroll -(4+reag) for 2, and AC +40 for 2+reag (src/magic.c:879,
		// 949-957). Only SPELL_BLINDNESS speaks; smokescreen's mag_masses
		// announces once and blinds silently (src/magic.c:959-964).
		pending = append(pending,
			engine.NewAffectDirect(spellNum, engine.ApplyHitroll, 2, -(4+reag), engine.AFFBlind, "blindness"),
			engine.NewAffectDirect(spellNum, engine.ApplyAC, 2+reag, 40, engine.AFFBlind, "blindness"))
		if spellNum == SpellBlindness {
			toVictim = "You have been blinded!\r\n"
			toRoom = "$n seems to be blinded!\r\n"
			toSelf = "A streak of blackness courses from your hand!\r\n"
		}
	case SpellCurse:
		if saved {
			sendToCaster(ch, "Nothing seems to happen.\r\n")
			npcRetaliate(world, victim, ch)
			return
		}
		curseDur := 1 + (getLevel(ch) >> 1)
		// C applies exactly two AFF_CURSE records, for hitroll then damroll
		// (src/magic.c:986-994). ActiveAffects is oldest-first while C's
		// ch->affected list is newest-first, so keep the C insertion order
		// here and reverse only when rendering that linked-list view.
		pending = append(pending,
			engine.NewAffectDirect(SpellCurse, engine.ApplyHitroll, curseDur, -3, engine.AFFCurse, "curse"),
			engine.NewAffectDirect(SpellCurse, engine.ApplyDamroll, curseDur, -3, engine.AFFCurse, "curse"))
		toVictim = "You feel very uncomfortable.\r\n"
		toRoom = "$n briefly glows red!\r\n"
		toSelf = "A streak of red light courses from your hand!\r\n"
		aggroSpell = true
	case SpellInvisible, SpellTransparency:
		name := "invisibility"
		if spellNum == SpellTransparency {
			name = "transparency"
		}
		aff = engine.NewAffectDirect(spellNum, engine.ApplyNone, 12+getLevel(ch)/4, 0, engine.AFFInvisible, name)
		if spellNum == SpellInvisible {
			toVictim = "You vanish.\r\n"
		} else {
			toVictim = "Your skin turns transparent.\r\n"
		}
		toRoom = "$n slowly fades out of existence.\r\n"
	case SpellSanctuary:
		aff = engine.NewAffectDirect(SpellSanctuary, engine.ApplyNone, 4, 0, engine.AFFSanctuary, "sanctuary")
		if isEvil(victim) {
			toVictim = "A black aura momentarily surrounds you.\r\n"
			toRoom = "$n is surrounded by a black aura.\r\n"
		} else {
			toVictim = "A white aura momentarily surrounds you.\r\n"
			toRoom = "$n is surrounded by a white aura.\r\n"
		}
	case SpellSleep:
		// C magic.c:1229 folds MOB_NOSLEEP into the save gate; both arms
		// produce the same room-only "shakes his head" line and no
		// caster/victim bytes.
		type npcSleepChecker interface {
			IsNPC() bool
			HasMobFlag(flag uint64) bool
		}
		nosleep := false
		if ns, ok := victim.(npcSleepChecker); ok && ns.IsNPC() && ns.HasMobFlag(1<<15) {
			nosleep = true
		}
		if nosleep || saved {
			sendAffectRoom(victim, nil, "$n shakes his head wearily, but then snaps out of it!\r\n", true, world)
			npcRetaliate(world, victim, ch)
			return
		}
		pending = append(pending, engine.NewAffectDirect(SpellSleep, engine.ApplyNone, 4+getLevel(ch)/4+reag, 0, engine.AFFSleep, "sleep"))
		// C magic.c:1241-1247 — the sleepy lines and the position drop only
		// happen to a victim still above POS_SLEEPING.
		if getPos(victim) > int(PosSleeping) {
			toVictim = "You feel very sleepy...  Zzzz......\r\n"
			toRoom = "$n goes to sleep.\r\n"
			type poserSleep interface{ SetPosition(int) }
			if p, ok := victim.(poserSleep); ok {
				p.SetPosition(int(PosSleeping)) // PosSleeping = 4
			}
		}
	case SpellFlameStrike:
		// C source: magic.c:1109-1129 — outdoor-only DOT with saving throw
		if saved {
			sendToCaster(ch, "Nothing seems to happen.\r\n")
			npcRetaliate(world, victim, ch)
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
		toVictim = "A bolt of flame shoots down from the heavens and engulfs you!\r\n"
		toRoom = "A bolt of flame shoots down from the heavens and engulfs $n!\r\n"
		toSelf = "You call down a bolt of flame on $M!\r\n"
		aggroSpell = true
	case SpellPoison:
		if saved {
			sendToCaster(ch, "Nothing seems to happen.\r\n")
			npcRetaliate(world, victim, ch)
			return
		}
		dur := (level / 2) - 2
		if dur < 1 {
			dur = 1
		}
		// C stores exactly two AFF_POISON nodes, each carrying a stat
		// modifier (src/magic.c:1095-1102).
		pending = append(pending,
			engine.NewAffectDirect(SpellPoison, engine.ApplyStr, dur, -2, engine.AFFPoison, "poison"),
			engine.NewAffectDirect(SpellPoison, engine.ApplyHitroll, dur, -2, engine.AFFPoison, "poison"))
		toVictim = "You feel very sick.\r\n"
		toRoom = "$n gets violently ill!\r\n"
		toSelf = "Your tainted magick pulses towards $M.\r\n"
		aggroSpell = true
	case SpellHaste:
		aff = engine.NewAffectDirect(SpellHaste, engine.ApplyNone, level, 0, engine.AFFHaste, "haste")
		// C quirk (magic.c:1046): "You feel your movement quicken!" is a
		// to_self line, so the caster sees it when ch != victim and the victim
		// never does. Kept verbatim.
		toSelf = "You feel your movement quicken!\r\n"
	case SpellSlow:
		// C sets bitvector AFF_HASTE for SPELL_SLOW (magic.c:1051 quirk), so a
		// spell-slowed combatant carries the haste bit — fight.c's attacks++
		// haste check then fires (and the AFF_SLOW attacks-- check never does,
		// since the spell never sets it). Kept verbatim: slow makes you swing
		// more in C.
		aff = engine.NewAffectDirect(SpellSlow, engine.ApplyNone, level, 0, engine.AFFHaste, "slow")
		toVictim = "You feel the world speed up around you.\r\n"
		toSelf = "You send the forces of time against $S!\r\n"
	case SpellFly, SpellLevitate:
		name := "fly"
		if spellNum == SpellLevitate {
			name = "levitate"
		}
		aff = engine.NewAffectDirect(spellNum, engine.ApplyNone, getLevel(ch), 0, engine.AFFFlying, name)
		toVictim = "Your feet rise off the ground!\r\n"
		toRoom = "$n's feet rise off the ground!\r\n"
		toSelf = "Like a falling feather, your magick floats toward $M.\r\n"
	case SpellDetectMagic:
		aff = engine.NewAffectDirect(SpellDetectMagic, engine.ApplyNone, 12+level, 0, engine.AFFDetectMagic, "detect magic")
		toVictim = "Your eyes tingle.\r\n"
		// Verbatim C typo: "A streak blue light" (magic.c:1029).
		toSelf = "A streak blue light courses from your fingertips, washing over $M!\r\n"
	case SpellDetectInvis:
		aff = engine.NewAffectDirect(SpellDetectInvis, engine.ApplyNone, 12+level, 0, engine.AFFDetectInvisible, "detect invis")
		toVictim = "Your eyes tingle.\r\n"
		toSelf = "A streak of yellow light courses from your hand, washing over $M!\r\n"
	case SpellInfravision:
		aff = engine.NewAffectDirect(SpellInfravision, engine.ApplyNone, 12+level, 0, engine.AFFInfrared, "infravision")
		joinAffect(victim, aff, true, false)
		aff = nil
		toVictim = "Your eyes glow red.\r\n"
		toRoom = "$n's eyes glow red.\r\n"
		toSelf = "With a light touch, you bestow the magick into $S eyes.\r\n"
	case SpellWaterBreathe:
		aff = engine.NewAffectDirect(SpellWaterBreathe, engine.ApplyNone, getLevel(ch), 0, engine.AFFWaterBreathing, "water breathe")
		toVictim = "You feel your breath become colder.\r\n"
	case SpellDetectAlign, SpellKnowAlign:
		// C initializes every affect record's type from spellnum (magic.c:877-879).
		// Know Align shares Detect Align's flag and duration, but keeps its own ID.
		aff = engine.NewAffectDirect(spellNum, engine.ApplyNone, 12+level, 0, engine.AFFDetectAlign, GetSpellName(spellNum))
		if spellNum == SpellDetectAlign {
			toVictim = "Your eyes tingle.\r\n"
		} else {
			toVictim = "Like a physical blow, emotions of others wash over you.\r\n"
		}
	case SpellDreamTravel:
		aff = engine.NewAffectDirect(SpellDreamTravel, engine.ApplyNone, 6, 0, engine.AFFDream, "dream travel")
		toVictim = "You feel the power of the Dream Lords surround you.\r\n"
	case SpellProtFromEvil:
		if isEvil(victim) {
			sendToCaster(ch, "You cannot protect yourself from the Evil inside you!\r\n")
			logProtectionKill(world, ch, "Evil")
			// C source: magic.c:1142-1148 — raw_kill(ch, TYPE_BLAST) on alignment violation
			if c, ok := ch.(combat.Combatant); ok {
				combat.RawKill(c, combat.TYPE_BLAST)
			}
			return
		}
		// Set flag only — no stat modifier
		pending = append(pending, engine.NewAffectDirect(SpellProtFromEvil, engine.ApplyNone, 24, 0, engine.AFFProtectionEvil, "prot from evil"))
		toVictim = "A stream of silver light surges from your fingertips, covering you!\r\n"
		toRoom = "A stream of silver light surges from $n's fingertips, covering $m!\r\n"
	case SpellProtFromGood:
		if isGood(victim) {
			sendToCaster(ch, "The forces of Light destroy you for your betrayal!\r\n")
			logProtectionKill(world, ch, "Good")
			// C source: magic.c:1162-1168 — raw_kill(ch, TYPE_BLAST) on alignment violation
			if c, ok := ch.(combat.Combatant); ok {
				combat.RawKill(c, combat.TYPE_BLAST)
			}
			return
		}
		pending = append(pending, engine.NewAffectDirect(SpellProtFromGood, engine.ApplyNone, 24, 0, engine.AFFProtectionGood, "prot from good"))
		toVictim = "A stream of silver light surges from your fingertips, covering you!\r\n"
		toRoom = "A stream of silver light surges from $n's fingertips, covering $m!\r\n"
	case SpellAdrenaline, SpellStrength:
		mag := 1 + boolToInt(level > 18)
		if ch == victim && spellNum == SpellAdrenaline {
			mag++
		}
		// C initializes every affect record's type from spellnum (magic.c:877-879).
		aff = engine.NewAffect(spellNum, engine.ApplyStr, (getLevel(ch)>>1)+4, mag, GetSpellName(spellNum))
		toVictim = "You feel stronger!\r\n"
		toSelf = "Grabbing $M, you feel a strong flow of magick course between you.\r\n"
	case SpellSenseLife:
		aff = engine.NewAffectDirect(SpellSenseLife, engine.ApplyNone, getLevel(ch), 0, engine.AFFSenseLife, "sense life")
		// Verbatim C typo: "Your feel your awareness improve." (magic.c:1267).
		toVictim = "Your feel your awareness improve.\r\n"
	case SpellWaterwalk:
		// C duration expression "4+reag?20:0" parses as (4+reag)?20:0 — always
		// 20 for reag >= 0 (magic.c:1279 quirk kept).
		aff = engine.NewAffectDirect(SpellWaterwalk, engine.ApplyNone, 20, 0, engine.AFFWaterwalk, "waterwalk")
		toVictim = "You feel webbing between your toes.\r\n"
		toSelf = "Your magic makes $M light footed.\r\n"
	case SpellChangeDensity:
		// Shares C's waterwalk arm, including the always-20 duration quirk.
		aff = engine.NewAffectDirect(SpellChangeDensity, engine.ApplyNone, 20, 0, engine.AFFWaterwalk, "change density")
		toVictim = "Your molecular density shifts.\r\n"
		toSelf = "You shift $S molecular density.\r\n"
	case SpellChameleon:
		aff = engine.NewAffectDirect(SpellChameleon, engine.ApplyNone, getLevel(ch), 0, engine.AFFHide, "chameleon")
		toVictim = "You blend into the surroundings.\r\n"
	case SpellMetalskin:
		pending = append(pending, engine.NewAffectDirect(SpellMetalskin, engine.ApplyNone, 5, -(15+getLevel(ch)/2+reag), engine.AFFMetalskin, "metalskin"), engine.NewAffect(SpellMetalskin, engine.ApplyAC, 5, -(15+getLevel(ch)/2+reag), "metalskin"))
		toVictim = "Your skin turns metallic!\r\n"
	case SpellInvulnerability:
		// C magic.c:1316-1328 applies AFF_INVULN to both records; the first
		// modifies armor class and the second modifies spell saving throws.
		pending = append(pending,
			engine.NewAffectDirect(SpellInvulnerability, engine.ApplyAC, 7, -100, engine.AFFInvuln, "invulnerability"),
			engine.NewAffectDirect(SpellInvulnerability, engine.ApplySavingSpell, 7, -7, engine.AFFInvuln, "invulnerability"))
		toVictim = "A globe of protection appears around you!\r\n"
	case SpellPsyshield:
		aff = engine.NewAffect(SpellPsyshield, engine.ApplyAC, getLevel(ch)/2, -15, "psyshield")
		toVictim = "You feel a shield of energy form around you.\r\n"
	case SpellGreatPercept:
		pending = append(pending, engine.NewAffectDirect(SpellGreatPercept, engine.ApplyNone, level/2+4, 0, engine.AFFDetectInvisible, "great percept"), engine.NewAffectDirect(SpellGreatPercept, engine.ApplyNone, level/2+4, 0, engine.AFFSenseLife, "great percept"))
		toVictim = "Your eyes glow briefly.\r\n"
		toRoom = "$n's eyes glow briefly.\r\n"
	case SpellLessPercept:
		pending = append(pending, engine.NewAffectDirect(SpellLessPercept, engine.ApplyNone, level/2+4, 0, engine.AFFDetectAlign, "lesser percept"), engine.NewAffectDirect(SpellLessPercept, engine.ApplyNone, level/2+4, 0, engine.AFFInfrared, "lesser percept"))
		toVictim = "Your eyes glow briefly.\r\n"
		toRoom = "$n's eyes glow briefly.\r\n"
	case SpellIntellect:
		aff = engine.NewAffect(SpellIntellect, engine.ApplyInt, 8, 1, "intellect")
		toVictim = "Your head clears and you realize some of the secrets of life!\r\n"
	case SpellMindBar:
		// C applies -18 INT with bitvector AFF_NOTHING and no status flag
		// (magic.c:1369-1377) — a pure stat affect, no invented bit.
		aff = engine.NewAffect(SpellMindBar, engine.ApplyInt, (level/2)-2, -18, "mind bar")
		toVictim = "Suddenly, your mind numbs and you feel somewhat impaired.\r\n"
		toSelf = "You place a mental bar across $S mind.\r\n"
	default:
		return
	}

	// C magic.c:1387-1404 — pre-apply gates. A mob that carries this affect's
	// flag innately (mob file, not the spell) refuses the spell, and a victim
	// already affected by a non-accumulating spell refuses it; both answer
	// NOEFFECT to the caster. Runs after the save-gated case arms (C evaluates
	// the switch first) but before anything is applied.
	accumDur, accumAff, bit0, bit1 := affectGateFlags(spellNum)
	if npcInnatelyAffected(victim, bit0, bit1, spellNum) ||
		(hasSpellAffect(victim, spellNum) && !accumDur && !accumAff) {
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}

	apply := func(a *engine.Affect) {
		// C magic.c:1410-1412 sends every populated affect record through
		// affect_join, even when both accumulation flags are false. Spells
		// with accum_duration or accum_affect therefore merge an existing
		// matching (spell, APPLY location) record instead of appending a
		// duplicate affect node.
		if accumDur || accumAff {
			joinAffect(victim, a, accumDur, accumAff)
			return
		}
		applyAffect(victim, a)
	}
	for _, a := range pending {
		apply(a)
	}
	if aff != nil {
		apply(aff)
	}

	// C magic.c:1414-1421 — send block. to_self goes to the caster only when
	// ch != victim, with $M/$S taken from the victim; to_vict always fires;
	// to_room uses act(..., TRUE, victim, 0, ch, TO_ROOM), so it excludes the
	// affected character while retaining the caster as a room recipient.
	if toSelf != "" && ch != victim {
		sendToCaster(ch, substituteToSelfPronouns(toSelf, victim))
	}
	if toVictim != "" {
		sendToVictim(victim, toVictim)
	}
	if toRoom != "" {
		// C magic.c:1421 executes act(to_room, TRUE, victim, 0, ch, TO_ROOM).
		// The wand caster is the substitution target, not an excluded recipient.
		sendAffectRoom(victim, ch, toRoom, true, world)
	}
	if aggroSpell {
		npcRetaliate(world, victim, ch)
	}
}

func consumeSpellReagentVNum(ch interface{}, vnum int) bool {
	if consumer, ok := ch.(interface{ ConsumeSpellReagentVNum(int) bool }); ok {
		return consumer.ConsumeSpellReagentVNum(vnum)
	}
	return false
}

func hasOutlawFlag(ch interface{}) bool {
	const plrOutlaw = 0 // structs.h: PLR_OUTLAW
	if flags, ok := ch.(interface{ GetFlags() uint64 }); ok {
		return flags.GetFlags()&(1<<plrOutlaw) != 0
	}
	return false
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// cAffBit constants are C's AFF_* bit positions (structs.h:310-348) — the
// layout the victim's innate affect mask uses (game's aff* constants). The
// engine's own AFF_* flags are a DIFFERENT numbering (engine/affect.go) and
// must not leak into the mob-affection gate: sanctuary is C bit 7 but engine
// bit 4, so testing engine positions against the C-position mask silently
// disabled the gate for every mismatched spell.
const (
	cAffBlind        = 0
	cAffInvisible    = 1
	cAffDetectAlign  = 2
	cAffDetectInvis  = 3
	cAffDetectMagic  = 4
	cAffSenseLife    = 5
	cAffWaterwalk    = 6
	cAffSanctuary    = 7
	cAffCurse        = 9
	cAffInfravision  = 10
	cAffPoison       = 11
	cAffProtectEvil  = 12
	cAffProtectGood  = 13
	cAffSleep        = 14
	cAffHide         = 19
	cAffFly          = 26
	cAffInvuln       = 30
	cAffFlaming      = 31
	cAffHaste        = 33
	cAffDream        = 35
	cAffWaterBreathe = 36
	cAffMetalskin    = 37
)

// affectGateFlags mirrors C mag_affects' per-spell accum_affect/accum_duration
// switches and the first two affects' bitvectors (magic.c:888-1380), which the
// magic.c:1387-1404 pre-apply gates test. bit0/bit1 are raw C AFF_* bit
// positions; an UNSET slot stays 0, and C's gate tests IS_AFFECTED(victim, 0)
// — bit 0 is AFF_BLIND — so an innately blind mob refuses even a single-affect
// spell. Do not "fix" that: it is C behavior.
func affectGateFlags(spellNum int) (accumDuration, accumAffect bool, bit0, bit1 int) {
	switch spellNum {
	case SpellChillTouch:
		return true, false, affNothingBit, 0
	case SpellArmor:
		return false, false, affNothingBit, 0
	case SpellBless:
		return true, false, affNothingBit, 0
	case SpellBlindness, SpellSmokescreen:
		return false, false, cAffBlind, cAffBlind
	case SpellCurse:
		return true, true, cAffCurse, cAffCurse
	case SpellKnowAlign, SpellDetectAlign:
		return false, false, cAffDetectAlign, 0
	case SpellDetectInvis:
		return true, false, cAffDetectInvis, 0
	case SpellDetectMagic:
		return true, false, cAffDetectMagic, 0
	case SpellInfravision:
		return true, false, cAffInfravision, 0
	case SpellHaste:
		return false, false, cAffHaste, 0
	case SpellSlow:
		// C sets bitvector AFF_HASTE for SPELL_SLOW (magic.c:1051 quirk),
		// mirrored by the apply path.
		return false, false, cAffHaste, 0
	case SpellDreamTravel:
		return false, false, cAffDream, 0
	case SpellWaterBreathe:
		return true, false, cAffWaterBreathe, 0
	case SpellTransparency, SpellInvisible:
		return false, false, cAffInvisible, 0
	case SpellPoison:
		return false, false, cAffPoison, cAffPoison
	case SpellFlameStrike:
		return false, false, cAffFlaming, 0
	case SpellFly, SpellLevitate:
		return true, false, cAffFly, 0
	case SpellProtFromEvil:
		return false, false, cAffProtectEvil, 0
	case SpellProtFromGood:
		return false, false, cAffProtectGood, 0
	case SpellSanctuary:
		return true, false, cAffSanctuary, 0
	case SpellSleep:
		return false, false, cAffSleep, 0
	case SpellAdrenaline:
		return true, false, affNothingBit, 0
	case SpellStrength:
		return false, true, affNothingBit, 0
	case SpellSenseLife:
		return true, false, cAffSenseLife, 0
	case SpellWaterwalk, SpellChangeDensity:
		return true, false, cAffWaterwalk, 0
	case SpellChameleon:
		return false, false, cAffHide, 0
	case SpellMetalskin:
		return true, false, cAffMetalskin, 0
	case SpellInvulnerability:
		return true, false, cAffInvuln, 0
	case SpellPsyshield:
		return true, false, affNothingBit, 0
	case SpellGreatPercept:
		return false, false, cAffSenseLife, cAffDetectInvis
	case SpellLessPercept:
		return false, false, cAffInfravision, cAffDetectAlign
	case SpellIntellect:
		return false, true, affNothingBit, 0
	case SpellMindBar:
		// C sets no bitvector for MIND_BAR beyond AFF_NOTHING (magic.c:1373).
		return false, false, affNothingBit, 0
	}
	return false, false, 0, 0
}

// affNothingBit is C's AFF_NOTHING (structs.h:342) — stat-only spells still
// test this real bit index in the mob-affection gate.
const affNothingBit = 32

// npcInnatelyAffected mirrors the C mob-affection gate (magic.c:1387-1394):
// an NPC that already carries the spell's affect flag — from its mob file,
// not from this spell — refuses the spell. bit0/bit1 are raw C positions and
// are BOTH tested unconditionally, exactly as C tests IS_AFFECTED(victim,
// af[0].bitvector) || IS_AFFECTED(victim, af[1].bitvector) — an unset slot is
// 0, i.e. the AFF_BLIND bit, so an innately blind mob refuses single-affect
// spells too (C quirk, kept).
func npcInnatelyAffected(victim interface{}, bit0, bit1, spellNum int) bool {
	type npcCheck interface{ IsNPC() bool }
	nc, ok := victim.(npcCheck)
	if !ok || !nc.IsNPC() {
		return false
	}
	type afflicted interface{ IsAffected(int) bool }
	af, ok := victim.(afflicted)
	if !ok {
		return false
	}
	if hasSpellAffect(victim, spellNum) {
		return false
	}
	return af.IsAffected(bit0) || af.IsAffected(bit1)
}

func hasSpellAffect(victim interface{}, spellNum int) bool {
	type speller interface{ HasSpellAffect(int) bool }
	if s, ok := victim.(speller); ok {
		return s.HasSpellAffect(spellNum)
	}
	return false
}

// getPos reads a character's position; unknown types report standing.

func getPos(ch interface{}) int {
	type poser interface{ GetPosition() int }
	if p, ok := ch.(poser); ok {
		return p.GetPosition()
	}
	return int(PosStanding)
}

// MagPoints handles HP/MV restoration spells.
func MagPoints(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil {
		return
	}
	_ = savetype
	_ = world

	isPsiMystic := isClassPsionicOrMystic(getClass(ch))
	formula, ok := magPointsFormula(level, spellNum, isPsiMystic)
	if !ok {
		return
	}

	// C mag_points draws MOVE before HIT (magic.c: SPELL_VITALITY does
	// move = dice(10,10); hit = dice(5,10);), and only calls dice() for the
	// component a spell actually uses — it never rolls a dice(0,0) for the
	// unused one. Match both: move first, then hit, each drawn only when it has
	// dice. (Reversing this, or drawing a phantom dice(0,0), desynced the shared
	// stream after any points spell — e.g. a quaffed vitality+poison potion.)
	move := formula.moveFlat
	if formula.moveNum > 0 {
		move += dice(formula.moveNum, formula.moveSides)
	}
	hit := formula.hitFlat
	if formula.hitNum > 0 {
		hit += dice(formula.hitNum, formula.hitSides)
	}

	switch spellNum {
	case SpellCureLight:
		sendToVictim(victim, "You feel better.\r\n")
		if ch != victim {
			mendRoomLine(ch, victim, world)
		}
	case SpellCureCritic:
		sendToVictim(victim, "You feel a lot better!\r\n")
		if ch != victim {
			mendRoomLine(ch, victim, world)
		}
	case SpellHeal:
		if isPsiMystic {
			sendToVictim(victim, "You focus your mind on healing your body..\r\n")
			sendToVictim(victim, "Your wounds mend themselves!\r\n")
		} else {
			sendToVictim(victim, "A warm feeling floods your body.\r\n")
			if ch != victim {
				mendRoomLine(ch, victim, world)
			}
		}
	case SpellMassHeal:
		sendToVictim(victim, "A warm feeling floods your body.\r\n")
		if ch != victim {
			mendRoomLine(ch, victim, world)
		}
	case SpellCellAdjustment:
		if isPsiMystic {
			sendToVictim(victim, "You focus your mind on healing your body..\r\n")
			sendToVictim(victim, "Your wounds mend themselves!\r\n")
		} else {
			sendToVictim(victim, "A warm feeling floods your body.\r\n")
			if ch != victim {
				mendRoomLine(ch, victim, world)
			}
		}
	case SpellVitality:
		sendToVictim(victim, "You feel vitalized!\r\n")
	case SpellInvigorate:
		sendToVictim(victim, "You feel invigorated!\r\n")
	case SpellLayHands:
		if ch == victim {
			sendToVictim(victim, "Your wounds mend beneath your hands!\r\n")
		} else {
			sendAct("Your wounds start to heal beneath $n's hands!\r\n", ch, nil, victim, world)
		}
	}

	if hit > 0 {
		healHP(victim, hit)
	}
	if move > 0 {
		type mover interface {
			GetMove() int
			GetMaxMove() int
			SetMove(int)
		}
		if m, ok := victim.(mover); ok {
			newMove := m.GetMove() + move
			if newMove > m.GetMaxMove() {
				newMove = m.GetMaxMove()
			}
			m.SetMove(newMove)
		}
	}
}

// pointsFormula describes the dice + flat components of a healing/movement spell.
// It mirrors src/magic.c:mag_points() without applying any RNG.
type pointsFormula struct {
	hitNum, hitSides, hitFlat    int
	moveNum, moveSides, moveFlat int
}

// magPointsFormula returns the base healing (hit) and movement restoration formula for a spell.
// It mirrors src/magic.c:mag_points() dice formulas exactly, with no RNG and no side effects.
// isPsionicOrMystic controls the heal/cell-adjustment branch (C: IS_PSIONIC || IS_MYSTIC).
// Returns ok=false for spell numbers that are not healing spells in C's mag_points().
func magPointsFormula(level, spellNum int, isPsionicOrMystic bool) (pointsFormula, bool) {
	switch spellNum {
	case SpellCureLight:
		return pointsFormula{hitNum: 2, hitSides: 8, hitFlat: 1 + (level >> 2)}, true
	case SpellCureCritic:
		return pointsFormula{hitNum: 5, hitSides: 8, hitFlat: 3 + (level >> 2)}, true
	case SpellHeal, SpellCellAdjustment:
		if isPsionicOrMystic {
			return pointsFormula{hitNum: 2, hitSides: 8, hitFlat: 90}, true
		}
		return pointsFormula{hitNum: 3, hitSides: 8, hitFlat: 100}, true
	case SpellMassHeal:
		return pointsFormula{hitFlat: 200}, true
	case SpellVitality:
		return pointsFormula{hitNum: 5, hitSides: 10, moveNum: 10, moveSides: 10}, true
	case SpellInvigorate:
		return pointsFormula{moveNum: 10, moveSides: 10}, true
	case SpellLayHands:
		return pointsFormula{hitNum: 3, hitSides: level}, true
	default:
		return pointsFormula{}, false
	}
}

// MagUnaffects is the faithful port of mag_unaffects (magic.c:1828-1876):
// the cure arms are guarded — a victim who does not carry the removable
// affect gets NOEFFECT to the CASTER (silent for heal/mass-heal, which ride
// along other routines) — and the cured pair for the vision arm is
// BLINDNESS *and* SMOKESCREEN, with the to_vict/to_room act lines.
func MagUnaffects(level int, ch, victim interface{}, spellNum int, world interface{}) {
	if victim == nil {
		return
	}
	_ = level

	var spell, spell2 int
	var toVictim, toRoom string
	switch spellNum {
	case SpellCureBlind, SpellHeal, SpellMassHeal:
		spell, spell2 = SpellBlindness, SpellSmokescreen
		toVictim = "Your vision returns!\r\n"
		toRoom = "There's a momentary gleam in $n's eyes.\r\n"
	case SpellRemovePoison:
		spell = SpellPoison
		toVictim = "A warm feeling runs through your body!\r\n"
		toRoom = "$n looks better.\r\n"
	case SpellRemoveCurse:
		spell = SpellCurse
		toVictim = "You don't feel so unlucky.\r\n"
	default:
		return
	}

	if !hasSpellAffect(victim, spell) && !hasSpellAffect(victim, spell2) {
		if spellNum != SpellHeal && spellNum != SpellMassHeal {
			sendToCaster(ch, "Nothing seems to happen.\r\n")
		}
		return
	}

	removeAffect(victim, spell)
	if spell2 != 0 {
		removeAffect(victim, spell2)
	}
	if toVictim != "" {
		sendToVictim(victim, toVictim)
	}
	if toRoom != "" {
		sendAffectRoom(victim, ch, toRoom, true, world) // C magic.c:1884 act(to_room, TRUE, victim, 0, ch, TO_ROOM)
	}
}

// MagGroups applies group versions of spells.
// MagGroups applies group spells to all grouped characters in the room.

func MagGroups(level int, ch interface{}, spellNum, savetype int, world interface{}) {
	if ch == nil {
		return
	}
	// C mag_groups (src/magic.c:1469-1474) returns before iterating anyone
	// unless the caster carries AFF_GROUP. Item routes do not have the spoken
	// cast gate, so this check is required here as well.
	type affectChecker interface{ IsAffected(int) bool }
	groupedCaster, ok := ch.(affectChecker)
	if !ok || !groupedCaster.IsAffected(combat.AFF_GROUP) {
		return
	}

	// Get all characters in the room
	type roomGetter interface{ GetRoomVNum() int }
	type worldChars interface {
		GetAllCharsInRoom(roomVNum int) []interface{}
	}

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
		if !areGrouped(ch, c, world) && c != ch {
			continue
		}
		switch spellNum {
		case SpellGroupHeal:
			MagPoints(level, ch, c, SpellHeal, savetype, world)
		case SpellHolyShield:
			MagAffects(level, ch, c, SpellArmor, savetype, world)
		case SpellGroupInvis:
			MagAffects(level, ch, c, SpellInvisible, savetype, world)
		case SpellGroupRecall:
			// C src/magic.c:1442-1443 routes each grouped member through
			// spell_recall, including its BFR/fighting gates and sleeping landing.
			ExecuteManualSpell(SpellWordOfRecall, level, ch, c, nil, "", world)
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
	type worldChars interface {
		GetAllCharsInRoom(roomVNum int) []interface{}
	}
	type npcChecker interface{ IsNPC() bool }
	type lever interface{ GetLevel() int }
	type charmSkipper interface{ IsCharmedI(interface{}) bool }

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
		sendAffectRoom(ch, nil, "Suddenly, a cloud of thick, acrid black smoke appears, obscuring your view of $n!\r\n", false, world)
	}

	// Apply to each non-grouped character in the room
	chars := w.GetAllCharsInRoom(roomVNum)
	for _, c := range chars {
		if c == ch {
			continue
		}
		// Skip immortals. C mag_masses: !IS_NPC(tch) && GET_LEVEL(tch) >= LVL_IMMORT
		// (src/magic.c:1537).
		if nc, ok := c.(npcChecker); ok && !nc.IsNPC() {
			if l, ok := c.(lever); ok && l.GetLevel() >= combat.LVL_IMMORT {
				continue
			}
		}
		// Skip charmed NPCs. The charm bit differs between engine and pkg/game,
		// so delegate the AFF_CHARM check to the game layer via the world (same
		// pattern as CanRaiseUndeadI; DP-1015). C: mag_areas skips
		// if (IS_NPC(tch) && IS_AFFECTED(tch, AFF_CHARM)).
		if nc, ok := c.(npcChecker); ok && nc.IsNPC() {
			if cs, ok := world.(charmSkipper); ok && cs.IsCharmedI(c) {
				continue
			}
		}
		// Skip grouped
		if areGrouped(ch, c, world) {
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
	type worldChars interface {
		GetAllCharsInRoom(roomVNum int) []interface{}
	}
	type npcChecker interface{ IsNPC() bool }
	type lever interface{ GetLevel() int }
	type charmSkipper interface{ IsCharmedI(interface{}) bool }

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
		sendAffectRoom(ch, nil, "$n gracefully gestures and the earth begins to shake violently!\r\n", false, world)
	case SpellAcidBlast:
		sendToCaster(ch, "A spray of acid flows from your fingertips!\r\n")
		sendAffectRoom(ch, nil, "$n raises a hand and acid sprays from $s fingers!\r\n", false, world)
	}

	chars := w.GetAllCharsInRoom(roomVNum)
	for _, c := range chars {
		if c == ch {
			continue
		}
		// Skip immortals
		if nc, ok := c.(npcChecker); ok && !nc.IsNPC() {
			if l, ok := c.(lever); ok && l.GetLevel() >= lvlImmort {
				continue
			}
		}
		// Skip charmed NPCs. The charm bit differs between engine and pkg/game,
		// so delegate the AFF_CHARM check to the game layer via the world (same
		// pattern as CanRaiseUndeadI; DP-1015). C: mag_areas skips
		// if (IS_NPC(tch) && IS_AFFECTED(tch, AFF_CHARM)).
		if nc, ok := c.(npcChecker); ok && nc.IsNPC() {
			if cs, ok := world.(charmSkipper); ok && cs.IsCharmedI(c) {
				continue
			}
		}
		// Skip grouped
		if areGrouped(ch, c, world) {
			continue
		}
		if spellNum == SpellMassDominate {
			// C mag_areas routes each eligible target through spell_charm with
			// the caster's level (magic.c:1617-1618), not mag_damage.
			ExecuteManualSpell(SpellCharm, getLevel(ch), ch, c, nil, "", world)
			continue
		}
		// C mag_areas deliberately passes SAVING_SPELL (literal 1) to
		// mag_damage, even when call_magic was entered as CAST_BREATH
		// (magic.c:1611). Preserve that call-path quirk.
		MagDamage(level, ch, c, spellNum, 1, world)
	}
}

// MagSummons summons NPCs into the world.
// MagSummons spawns NPCs based on spell type (e.g. Animate Dead creates zombies).

// animateDeadPfailRoll returns number(0,101) for the SPELL_ANIMATE_DEAD pfail
// check. It is a package var so tests can make the ~8% failure branch
// deterministic without replacing the process-wide production stream.
var animateDeadPfailRoll = func() int { return dprng.Number(0, 101) } // #nosec G404 — intentional MUD RNG

func MagSummons(level int, ch, obj interface{}, spellNum int, world interface{}) {
	if ch == nil {
		return
	}

	switch spellNum {
	case SpellAnimateDead:
		type roomGetter interface{ GetRoomVNum() int }
		rg, ok := ch.(roomGetter)
		if !ok {
			return
		}
		roomVNum := rg.GetRoomVNum()

		// C mag_summons requires the explicitly targeted object to be a
		// container whose value[3] is nonzero (src/magic.c:1689-1697).
		type objectTypeGetter interface{ GetObjType() int }
		type objectValueGetter interface{ GetValue(int) int }
		type itemRemover interface {
			RemoveItemFromRoomI(item interface{}, roomVNum int)
		}

		objectType, hasObjectType := obj.(objectTypeGetter)
		objectValue, hasObjectValue := obj.(objectValueGetter)
		if !hasObjectType || !hasObjectValue || objectType.GetObjType() != 15 || objectValue.GetValue(3) == 0 {
			// C sends mag_summon_fail_msgs[7] through act(..., TO_CHAR), which
			// appends its own CRLF after the message's (src/magic.c:1693-1695).
			sendToCaster(ch, "There is no corpse!\r\n\r\n")
			return
		}

		// C magic.c: giddy (AFF_CHARM) check and follower cap, both delegated to
		// pkg/game because the spell layer cannot import game (import cycle).
		type raiseChecker interface {
			CanRaiseUndeadI(interface{}) (bool, string)
		}
		if checker, ok2 := world.(raiseChecker); ok2 {
			if canRaise, msg := checker.CanRaiseUndeadI(ch); !canRaise {
				sendToCaster(ch, msg)
				return
			}
		}

		// C magic.c: pfail = 8; number(0, 101) < pfail sends
		// mag_summon_fail_msgs[fmsg] (src/magic.c:1719-1722), and fmsg stays
		// 0 for SPELL_ANIMATE_DEAD: a bare CRLF, not "You failed.".
		if animateDeadPfailRoll() < 8 {
			sendToCaster(ch, "\r\n")
			return
		}

		type mobSpawner interface {
			SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error)
		}
		w, ok := world.(mobSpawner)
		if !ok {
			sendToCaster(ch, "The corpse refuses to come alive!\r\n")
			return
		}

		mobLevel := getLevel(ch) / 2
		if mobLevel < 1 {
			mobLevel = 1
		}
		mob, err := w.SpawnMobWithLevelI(10, roomVNum, mobLevel) // MOB_ZOMBIE = 10
		if err != nil {
			sendToCaster(ch, "The corpse refuses to come alive!\r\n")
			return
		}

		// C magic.c: extract_obj(corpse) runs only after create_mobile succeeds.
		if remover, ok2 := world.(itemRemover); ok2 {
			remover.RemoveItemFromRoomI(obj, roomVNum)
		}

		// C magic.c: SET_BIT(AFF_FLAGS(mob), AFF_CHARM); add_follower_quiet(mob, ch).
		type charmer interface{ CharmAndFollowI(mob, leader interface{}) }
		if cf, ok2 := world.(charmer); ok2 {
			cf.CharmAndFollowI(mob, ch)
		}

		// C indexes mag_summon_msgs with fmsg, which remains zero for
		// SPELL_ANIMATE_DEAD: act("\r\n", FALSE, ch, 0, mob, TO_ROOM)
		// (src/magic.c:1729), and act() appends its own CRLF, so every
		// listener, blind or not, gets two.
		if a, ok := world.(interface {
			SpellActToRoom(ch, vict interface{}, format string, hideInvisible bool) bool
		}); !ok || !a.SpellActToRoom(ch, mob, "\r\n", false) {
			sendAffectRoomFallback(ch, mob, nil, "\r\n\r\n", world)
		}
		sendToCaster(ch, "The corpse starts to twitch, then stands with a life of it's own!\r\n")
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
		if _, ok := ch.(roomGetter); !ok {
			return
		}

		type objectCreator interface {
			SpawnObjectToPlayerInventoryI(vnum int, player interface{}) (interface{}, error)
		}
		w, ok := world.(objectCreator)
		if !ok {
			return
		}

		obj, err := w.SpawnObjectToPlayerInventoryI(8062, ch)
		if err != nil {
			sendToCaster(ch, "I seem to have goofed.\r\n")
			return
		}

		type shortDescriber interface{ GetShortDesc() string }
		desc, ok := obj.(shortDescriber)
		if !ok {
			return
		}
		if roomAct, ok := world.(interface {
			SpellCreatedObjectAct(ch, obj interface{}) bool
		}); !ok || !roomAct.SpellCreatedObjectAct(ch, obj) {
			sendCreatedObjectRoom(ch, obj, world)
		}
		sendToCaster(ch, "You create "+desc.GetShortDesc()+".\r\n")
	}
}

func sendCreatedObjectRoom(ch, obj, world interface{}) {
	type roomGetter interface{ GetRoomVNum() int }
	type named interface{ GetName() string }
	type shortDescriber interface{ GetShortDesc() string }
	type sender interface{ SendMessage(string) }

	actor, actorOK := ch.(named)
	room, roomOK := ch.(roomGetter)
	object, objectOK := obj.(shortDescriber)
	w, worldOK := world.(roomIterable)
	if !actorOK || !roomOK || !objectOK || !worldOK {
		return
	}
	msg := capitalizeActMessage(actor.GetName() + " creates " + object.GetShortDesc() + ".\r\n")
	w.ForEachPlayerInRoomInterface(room.GetRoomVNum(), func(p interface{}) {
		if selfTarget(ch, p) {
			return
		}
		if !canReceiveAct(p) {
			return
		}
		if s, ok := p.(sender); ok {
			s.SendMessage(msg)
		}
	})
}

// MagAlterObjs alters objects.
// MagAlterObjs modifies target object attributes based on spell type.
func MagAlterObjs(level int, ch, obj interface{}, spellNum int, world interface{}) {
	if obj == nil || ch == nil {
		return
	}
	_ = level

	// Object interfaces for flag manipulation
	// The object's own extra flags, word 0 (GET_OBJ_EXTRA): the base the
	// spell's SET_BIT/REMOVE_BIT starts from.
	type extraFlagGetter interface{ ExtraFlagWord(word int) int }
	type extraFlagSetter interface{ SetExtraFlags(int) }
	type objTypeGetter interface{ GetObjType() int }
	type objValGetter interface{ GetObjVal(idx int) int }
	type objValSetter interface{ SetObjVal(idx, val int) }
	type objWeightGetter interface{ GetWeight() int }

	getExtraFlags := func(o interface{}) int {
		if f, ok := o.(extraFlagGetter); ok {
			return f.ExtraFlagWord(0)
		}
		return 0
	}
	setExtraFlags := func(o interface{}, flags int) {
		if f, ok := o.(extraFlagSetter); ok {
			f.SetExtraFlags(flags)
		}
	}

	var effect string
	switch spellNum {
	case SpellBless:
		flags := getExtraFlags(obj)
		// src/structs.h: ITEM_MAGIC=6 and ITEM_BLESS=8.
		cl := getLevel(ch)
		if wg, ok := obj.(objWeightGetter); ok && flags&(1<<8) == 0 && flags&(1<<6) == 0 && wg.GetWeight() <= 5*cl {
			setExtraFlags(obj, flags|(1<<8)) // ITEM_BLESS
			effect = "$p glows briefly with a ethereal light."
		}

	case SpellCurse:
		flags := getExtraFlags(obj)
		if flags&(1<<7) == 0 { // ITEM_NODROP
			setExtraFlags(obj, flags|(1<<7))
			if ot, ok := obj.(objTypeGetter); ok && ot.GetObjType() == 3 { // ITEM_WEAPON
				if vs, ok := obj.(objValSetter); ok {
					if vg, ok := obj.(objValGetter); ok {
						vs.SetObjVal(2, vg.GetObjVal(2)-1)
					}
				}
			}
			effect = "$p briefly glows red."
		}

	case SpellInvisible:
		flags := getExtraFlags(obj)
		// src/structs.h: ITEM_NOINVIS=4 and ITEM_INVISIBLE=5.
		if flags&(1<<4) == 0 {
			setExtraFlags(obj, flags|(1<<5)) // ITEM_INVISIBLE
			effect = "$p vanishes."
		}

	case SpellPoison:
		if ot, ok := obj.(objTypeGetter); ok {
			objType := ot.GetObjType()
			// ITEM_DRINKCON=17, ITEM_FOUNTAIN=18, ITEM_FOOD=19
			if objType == 17 || objType == 18 || objType == 19 {
				if vs, ok := obj.(objValSetter); ok {
					if vg, ok := obj.(objValGetter); ok {
						if vg.GetObjVal(3) == 0 {
							vs.SetObjVal(3, 1) // poison val
							effect = "$p steams briefly."
						}
					}
				}
			}
		}

	case SpellRemovePoison:
		if ot, ok := obj.(objTypeGetter); ok {
			objType := ot.GetObjType()
			if objType == 17 || objType == 18 || objType == 19 { // ITEM_DRINKCON, ITEM_FOUNTAIN, ITEM_FOOD
				if vs, ok := obj.(objValSetter); ok {
					if vg, ok := obj.(objValGetter); ok && vg.GetObjVal(3) != 0 {
						vs.SetObjVal(3, 0)
						effect = "$p steams briefly."
					}
				}
			}
		}

	case SpellRemoveCurse:
		flags := getExtraFlags(obj)
		if flags&(1<<7) == 0 { // not NODROP
			break
		}
		setExtraFlags(obj, flags&^(1<<7))                              // remove NODROP
		if ot, ok := obj.(objTypeGetter); ok && ot.GetObjType() == 3 { // ITEM_WEAPON
			if vs, ok := obj.(objValSetter); ok {
				if vg, ok := obj.(objValGetter); ok {
					vs.SetObjVal(2, vg.GetObjVal(2)+1)
				}
			}
		}
		effect = "$p briefly glows blue."
	}

	if effect == "" {
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}
	if acter, ok := world.(interface {
		SpellObjectAct(caster, target interface{}, message string, toRoom bool)
	}); ok {
		acter.SpellObjectAct(ch, obj, effect, false)
		acter.SpellObjectAct(ch, obj, effect, true)
		return
	}
	name := "something"
	if named, ok := obj.(interface{ GetShortDesc() string }); ok {
		name = named.GetShortDesc()
	}
	sendToCaster(ch, strings.ReplaceAll(effect, "$p", name)+"\r\n")
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
		castZen(level, ch, cvict, world)
	case SpellDetectPoison:
		castDetectPoison(level, ch, cvict, ovict, world)
	case SpellCalliope:
		castCalliope(level, ch, cvict, world)
	case SpellLycanthropy:
		castLycanthropy(level, ch, cvict)
	case SpellVampirism:
		castVampirism(level, ch, cvict)
	case SpellControlWeather:
		castControlWeather(level, ch, arg, world)
	case SpellCoC:
		castCoC(level, ch, world)
	case SpellMentalLapse:
		castMentalLapse(level, ch, cvict, world)
	case SpellEnchantWeapon:
		castEnchantWeapon(level, ch, ovict, world)
	case SpellEnchantArmor:
		castEnchantArmor(level, ch, ovict, world)
	case SpellCreateWater:
		castCreateWater(level, ch, ovict)
	case SpellIdentify:
		castIdentify(level, ch, cvict, ovict, world)
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
	case SpellCharm, SpellDominate:
		// src/spell_parser.c:505-507: C routes SPELL_DOMINATE through to
		// MANUAL_SPELL(spell_charm) alongside SPELL_CHARM. spell_charm takes
		// no spellnum (src/spells.c:407, ASPELL) and applies SPELL_CHARM as
		// its affect type, so dominate carries the same effect (DP-1405).
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
		// C's manual switch has no default clause at all (src/spell_parser.c:
		// 502-536): every MAG_MANUAL spell in the spell table has a case, so
		// this arm is unreachable and C emits nothing here. The port used to
		// send "Spell not yet implemented.\r\n", a string that appears in no
		// C file, which a live dominate hit (R4, DP-1405).
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

func joinAffect(entity interface{}, aff *engine.Affect, addDuration, addMagnitude bool) {
	type joiner interface {
		JoinAffect(*engine.Affect, bool, bool)
	}
	if j, ok := entity.(joiner); ok {
		j.JoinAffect(aff, addDuration, addMagnitude)
		return
	}
	applyAffect(entity, aff)
}

// mendRoomLine is mag_points' heal line (src/magic.c:1769,1776,1793,1803):
// act("$N's wounds glow and mend themselves.", TRUE, ch, 0, victim,
// TO_NOTVICT). Only a listener who can see the caster hears it, and $N is
// PERS(victim) for that listener.
func mendRoomLine(ch, victim, world interface{}) {
	if a, ok := world.(interface {
		SpellActNotVict(ch, vict interface{}, format string, hideInvisible bool) bool
	}); ok && a.SpellActNotVict(ch, victim, "$N's wounds glow and mend themselves.", true) {
		return
	}
	sendAffectRoomFallback(ch, victim, victim, "$N's wounds glow and mend themselves.\r\n", world)
}

func sendAffectRoom(actor, target interface{}, format string, hideInvisible bool, world interface{}) {
	type roomActor interface {
		SpellActToRoom(actor, target interface{}, format string, hideInvisible bool) bool
	}
	actFormat := strings.TrimRight(format, "\r\n")
	if a, ok := world.(roomActor); ok && a.SpellActToRoom(actor, target, actFormat, hideInvisible) {
		return
	}
	sendAffectRoomFallback(actor, target, nil, format, world)
}

// sendAffectRoomFallback keeps small spell test adapters usable when they do
// not provide the live act() renderer. Production worlds take the path above.
func sendAffectRoomFallback(actor, target, extraExclude interface{}, format string, world interface{}) {
	type roomed interface{ GetRoomVNum() int }
	type named interface{ GetName() string }
	type sender interface{ SendMessage(string) }
	w, ok := world.(roomIterable)
	ar, roomOK := actor.(roomed)
	actorName, actorNameOK := actor.(named)
	if !ok || !roomOK || !actorNameOK {
		return
	}
	targetName := ""
	if namedTarget, ok := target.(named); ok {
		targetName = namedTarget.GetName()
	}
	msg := strings.ReplaceAll(format, "$n", actorName.GetName())
	msg = strings.ReplaceAll(msg, "$N", targetName)
	msg = strings.ReplaceAll(msg, "$s", possessivePronoun(actor))
	msg = strings.ReplaceAll(msg, "$S", possessivePronoun(target))
	msg = strings.ReplaceAll(msg, "$m", objectivePronoun(actor))
	msg = strings.ReplaceAll(msg, "$M", objectivePronoun(target))
	if !strings.HasSuffix(msg, "\r\n") {
		msg += "\r\n"
	}
	w.ForEachPlayerInRoomInterface(ar.GetRoomVNum(), func(p interface{}) {
		if selfTarget(actor, p) || (extraExclude != nil && selfTarget(extraExclude, p)) {
			return
		}
		if !canReceiveAct(p) {
			return
		}
		if s, ok := p.(sender); ok {
			s.SendMessage(capitalizeActMessage(msg))
		}
	})
}

func sendAffectActRoom(actor, target interface{}, format string, hideInvisible bool, world interface{}) {
	format = strings.ReplaceAll(format, "$e", subjectivePronoun(actor))
	sendAffectRoom(actor, target, format, hideInvisible, world)
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

// npcRetaliate is C's hit(victim, ch, TYPE_UNDEFINED) at every spell
// save-refusal, summon and aggro site (src/spells.c:186,192,317,348,432,590;
// src/magic.c:945,972,1091,1114,1234,1421): the victim's first swing consumes
// hit()'s to-hit and damage draws and damage()'s protection block decides,
// enrolling at the damage point (the pending-body lane for a flagged standing
// body). C gates the call on IS_NPC(victim); a PC victim never retaliates.
// The seam goes through the game world's ExecuteSpellCounterattack — the old
// StartCombatWith name-based assertion had no implementation anywhere and
// silently no-opped every caller.
func npcRetaliate(world, victim, ch interface{}) {
	type npcCheck interface{ IsNPC() bool }
	if nc, ok := victim.(npcCheck); !ok || !nc.IsNPC() {
		return
	}
	if w, ok := world.(interface {
		ExecuteSpellCounterattack(victim, caster interface{})
	}); !ok {
		// No silent no-op: a world that cannot run hit() is a broken seam,
		// and every caller of this class once vanished here quietly.
		panic("spells: npcRetaliate world lacks ExecuteSpellCounterattack")
	} else {
		w.ExecuteSpellCounterattack(victim, ch)
	}
}

// spellCounterattack is C's hit(victim, ch, TYPE_UNDEFINED) at the spell
// refusal sites (src/spells.c:186, 192): the victim's first swing consumes
// hit()'s draws and damage()'s gates, delivered through the game world's
// counterattack seam. npcRetaliate above documents the older name-based
// no-op lane its callers still carry.
func spellCounterattack(world, victim, ch interface{}) {
	if w, ok := world.(interface {
		ExecuteSpellCounterattack(victim, caster interface{})
	}); ok {
		w.ExecuteSpellCounterattack(victim, ch)
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
	type messageSender interface{ SendMessage(string) }

	var found bool
	if consumer, ok := ch.(interface{ ConsumeSpellReagent(string) bool }); ok {
		found = consumer.ConsumeSpellReagent(reagentName)
	} else if holder, ok := ch.(InventoryHolder); ok {
		if item, ok := holder.GetInventory().FindItem(reagentName); ok {
			// Honour the remove's result: FindItem is a snapshot lookup, so
			// two concurrent casts can both find the same item and only the
			// one whose atomic remove succeeds has consumed it.
			found = holder.GetInventory().RemoveItem(item)
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
	// Spell info table transcribed verbatim from src/spell_parser.c spello() calls.
	setupSpellInfo(SpellArmor, PosFighting, 15, 30, 3, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellTeleport, PosFighting, 50, 60, 3, RoutineManual, false, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellBless, PosStanding, 10, 36, 2, RoutineAffects|RoutineAlterObjs, false, TarCharRoom|TarObjInv)
	setupSpellInfo(SpellBlindness, PosFighting, 25, 35, 1, RoutineAffects, true, TarCharRoom|TarNotSelf|TarFightVict)
	setupSpellInfo(SpellBurningHands, PosFighting, 20, 45, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellCallLightning, PosFighting, 52, 68, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellCharm, PosFighting, 50, 75, 5, RoutineManual, true, TarCharRoom|TarNotSelf)
	setupSpellInfo(SpellChillTouch, PosFighting, 15, 35, 5, RoutineDamage|RoutineAffects, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellClone, PosStanding, 65, 80, 5, RoutineSummons, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellColorSpray, PosFighting, 38, 58, 4, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellControlWeather, PosStanding, 25, 75, 5, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellCreateFood, PosStanding, 10, 35, 5, RoutineCreations, false, TarIgnore)
	setupSpellInfo(SpellCreateWater, PosStanding, 10, 35, 5, RoutineManual, false, TarObjInv|TarObjEquip)
	setupSpellInfo(SpellCureBlind, PosStanding, 5, 35, 5, RoutineUnaffects, false, TarCharRoom)
	setupSpellInfo(SpellCureCritic, PosFighting, 40, 70, 5, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellCureLight, PosFighting, 10, 30, 2, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellCurse, PosFighting, 50, 80, 2, RoutineAffects|RoutineAlterObjs, true, TarCharRoom|TarObjInv|TarFightVict)
	setupSpellInfo(SpellDetectAlign, PosStanding, 10, 20, 2, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellDetectInvis, PosStanding, 10, 20, 2, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellDetectMagic, PosStanding, 10, 20, 2, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellDetectPoison, PosStanding, 10, 20, 2, RoutineManual, false, TarCharRoom|TarObjInv|TarObjRoom)
	setupSpellInfo(SpellDispelEvil, PosFighting, 65, 95, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDispelGood, PosFighting, 65, 95, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellEarthquake, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellDreamTravel, PosStanding, 45, 60, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellEnchantWeapon, PosStanding, 150, 200, 10, RoutineManual, false, TarObjInv|TarObjEquip)
	setupSpellInfo(SpellEnergyDrain, PosFighting, 45, 60, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellHolyShield, PosStanding, 65, 90, 5, RoutineGroups, false, TarIgnore)
	setupSpellInfo(SpellFireball, PosFighting, 50, 70, 2, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellGroupHeal, PosFighting, 150, 210, 5, RoutineGroups, false, TarIgnore)
	setupSpellInfo(SpellChameleon, PosStanding, 30, 50, 5, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellGroupRecall, PosStanding, 125, 155, 5, RoutineGroups, false, TarIgnore)
	setupSpellInfo(SpellHarm, PosFighting, 75, 105, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellHaste, PosStanding, 140, 140, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellHeal, PosFighting, 80, 90, 3, RoutinePoints|RoutineAffects|RoutineUnaffects, false, TarCharRoom)
	setupSpellInfo(SpellHellfire, PosFighting, 150, 200, 10, RoutineManual|RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellInfravision, PosStanding, 25, 25, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellGroupInvis, PosStanding, 135, 135, 1, RoutineGroups, false, TarIgnore)
	setupSpellInfo(SpellInvisible, PosStanding, 45, 45, 1, RoutineAffects|RoutineAlterObjs, false, TarCharRoom|TarObjInv|TarObjRoom)
	setupSpellInfo(SpellLevitate, PosStanding, 70, 90, 5, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellLightningBolt, PosFighting, 34, 54, 4, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellLocateObject, PosStanding, 20, 25, 1, RoutineManual, false, TarObjWorld)
	setupSpellInfo(SpellMagicMissile, PosFighting, 15, 30, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMindPoke, PosFighting, 15, 30, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMindBlast, PosFighting, 40, 70, 2, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellPoison, PosFighting, 40, 50, 2, RoutineAffects|RoutineAlterObjs, true, TarCharRoom|TarNotSelf|TarObjInv|TarFightVict)
	setupSpellInfo(SpellFlamestrike, PosStanding, 100, 105, 1, RoutineAffects, true, TarCharRoom|TarNotSelf)
	setupSpellInfo(SpellProtFromEvil, PosStanding, 50, 50, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellProtFromGood, PosStanding, 50, 50, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellRemoveCurse, PosStanding, 45, 45, 1, RoutineUnaffects|RoutineAlterObjs, false, TarCharRoom|TarObjInv)
	setupSpellInfo(SpellSanctuary, PosStanding, 85, 110, 2, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellShockingGrasp, PosFighting, 35, 55, 5, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellSleep, PosStanding, 35, 40, 1, RoutineAffects, true, TarCharRoom|TarNotSelf)
	setupSpellInfo(SpellSobriety, PosStanding, 20, 35, 5, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellStrength, PosStanding, 30, 35, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellSummon, PosStanding, 70, 90, 1, RoutineManual, false, TarCharWorld|TarNotSelf)
	setupSpellInfo(SpellCoC, PosStanding, 70, 90, 1, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellWordOfRecall, PosFighting, 50, 50, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellRemovePoison, PosStanding, 30, 40, 1, RoutineUnaffects|RoutineAlterObjs, false, TarCharRoom|TarObjInv|TarObjRoom)
	setupSpellInfo(SpellSenseLife, PosStanding, 20, 30, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellSlow, PosStanding, 50, 80, 2, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellMassHeal, PosFighting, 100, 130, 1, RoutinePoints|RoutineAffects|RoutineUnaffects, false, TarCharRoom)
	setupSpellInfo(SpellWaterwalk, PosStanding, 55, 80, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellFly, PosStanding, 80, 100, 5, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellLycanthropy, PosStanding, 1, 1, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellVampirism, PosStanding, 1, 1, 1, RoutineManual, false, TarCharRoom)
	setupSpellInfo(SpellEnchantArmor, PosStanding, 130, 150, 10, RoutineManual, false, TarObjInv|TarObjEquip)
	setupSpellInfo(SpellIdentify, PosStanding, 100, 125, 10, RoutineManual, false, TarCharRoom|TarObjInv|TarObjRoom)
	setupSpellInfo(SpellMetalskin, PosFighting, 60, 75, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellInvulnerability, PosFighting, 85, 85, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellVitality, PosFighting, 100, 110, 1, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellInvigorate, PosFighting, 95, 110, 1, RoutinePoints, false, TarCharRoom)
	setupSpellInfo(SpellPsyshield, PosFighting, 20, 30, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellAdrenaline, PosStanding, 30, 35, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellMindAttack, PosFighting, 25, 55, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellLessPercept, PosStanding, 30, 40, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellGreatPercept, PosStanding, 45, 65, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellChangeDensity, PosStanding, 55, 70, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellAcidBlast, PosFighting, 20, 35, 1, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellDominate, PosFighting, 50, 75, 5, RoutineManual, true, TarCharRoom|TarNotSelf)
	setupSpellInfo(SpellMassDominate, PosStanding, 150, 220, 10, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellCellAdjustment, PosFighting, 75, 85, 1, RoutinePoints, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellZen, PosFighting, 60, 70, 4, RoutineManual, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellMirrorImage, PosStanding, 130, 150, 5, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellSoulLeech, PosFighting, 55, 60, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMindsight, PosStanding, 60, 70, 1, RoutineManual, false, TarCharWorld)
	setupSpellInfo(SpellMindBar, PosStanding, 100, 115, 1, RoutineAffects, true, TarCharRoom)
	setupSpellInfo(SpellTransparency, PosStanding, 25, 35, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellKnowAlign, PosStanding, 20, 20, 1, RoutineAffects, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellGate, PosStanding, 95, 95, 1, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellIntellect, PosStanding, 60, 60, 1, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellLayHands, PosStanding, 90, 90, 1, RoutinePoints, false, TarCharRoom|TarSelfOnly)
	setupSpellInfo(SpellMentalLapse, PosStanding, 90, 100, 1, RoutineManual, false, TarCharWorld)
	setupSpellInfo(SpellSmokescreen, PosFighting, 100, 100, 1, RoutineMasses, true, TarIgnore)
	setupSpellInfo(SpellDisrupt, PosFighting, 165, 175, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellDivineInt, PosStanding, 290, 290, 1, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellDisintegrate, PosFighting, 120, 120, 1, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellAnimateDead, PosStanding, 100, 120, 10, RoutineSummons, false, TarObjRoom)
	setupSpellInfo(SpellCalliope, PosFighting, 50, 100, 10, RoutineManual, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellMeteorSwarm, PosStanding, 170, 180, 5, RoutineManual, true, TarIgnore)
	setupSpellInfo(SpellPsiblast, PosFighting, 150, 180, 10, RoutineDamage, true, TarCharRoom|TarFightVict)
	setupSpellInfo(SpellWaterBreathe, PosStanding, 58, 92, 6, RoutineAffects, false, TarCharRoom)
	setupSpellInfo(SpellConjureElemental, PosStanding, 145, 165, 1, RoutineManual, false, TarIgnore)
	setupSpellInfo(SpellFireBreath, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellFrostBreath, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellGasBreath, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellAcidBreath, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
	setupSpellInfo(SpellLightningBreath, PosFighting, 50, 70, 5, RoutineAreas, true, TarIgnore)
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
		GetHP() int
		GetPosition() int
		SetPosition(int)
	}
	c, ok := cvict.(conder)
	if !ok {
		return
	}

	c.SetCondition(0, 0) // DRUNK = 0
	// C src/spells.c:694 calls update_pos after clearing DRUNK.
	c.SetPosition(combat.GetPositionFromHP(c.GetHP(), c.GetPosition()))
	// C's adjacent send_to_char calls form one output burst with LFCR at the end.
	sendToVictim(cvict, "You are splashed in the face with HOT coffee, but feel much more sober.\n\r")
}

// castZen ports src/spells.c:871-883. The spell is self-only; C sends one
// caster line and a room act before setting position and restoring hit points.
func castZen(level int, ch, cvict, world interface{}) {
	sendToCaster(ch, "You begin to meditate deeply, focusing your thoughts only on healing.\r\n")
	if renderer, ok := world.(interface {
		SpellActToRoom(caster, victim interface{}, format string, hideInvisible bool) bool
	}); ok {
		renderer.SpellActToRoom(ch, nil, "$n sinks into a deep, meditative sleep.", false)
	}

	// TAR_SELF_ONLY makes victim the caster on the registered C command path.
	healHP(ch, 2*level)

	type poser interface{ SetPosition(int) }
	if p, ok := ch.(poser); ok {
		p.SetPosition(int(PosStunned))
	}

	_ = cvict
}

// castDetectPoison — checks for poison on victims and objects.
// C source: spells.c:794-836. AFF_POISON = bit 11.
func castDetectPoison(level int, ch, cvict, ovict, world interface{}) {
	_ = level

	// Check victim for poison affect
	if cvict != nil {
		type affecter interface{ IsAffected(int) bool }

		a, ok := cvict.(affecter)
		if !ok {
			return
		}

		if a.IsAffected(11) { // AFF_POISON
			if cvict == ch {
				sendToCaster(ch, "You can sense poison in your blood.\r\n")
			} else {
				sendSpellActToCaster("You sense that $E is poisoned.", ch, cvict, nil, world)
			}
		} else {
			if cvict == ch {
				sendToCaster(ch, "You feel healthy.\r\n")
			} else {
				sendSpellActToCaster("You sense that $E is healthy.", ch, cvict, nil, world)
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
				sendSpellActToCaster("You sense that $p has been contaminated.", ch, nil, ovict, world)
			} else {
				sendSpellActToCaster("You sense that $p is safe for consumption.", ch, nil, ovict, world)
			}
		default:
			sendToCaster(ch, "You sense that it should not be consumed.\r\n")
		}
	}
}

// castCalliope — fires multiple magic missiles at a target.
// C source: spells.c:983-997. missiles = MAX(4, number(level/6, level*2)).
func castCalliope(level int, ch, cvict, world interface{}) {
	if cvict == nil {
		return
	}

	missiles := max(4, dprng.Number(level/6, level*2))

	for i := 0; i < missiles; i++ {
		CallMagic(ch, cvict, nil, SpellMagicMissile, level, CastSpell, world)
	}
}

// castLycanthropy — sets PLR_WEREWOLF flag.
// C source: spells.c:662-700.
func castLycanthropy(level int, ch, cvict interface{}) {
	_ = level
	if cvict == nil {
		sendToCaster(ch, "Specify a target.\n\r")
		return
	}
	type npcTarget interface{ IsNPC() bool }
	if npc, ok := cvict.(npcTarget); ok && npc.IsNPC() {
		// C's PLR_FLAGS and MOB_FLAGS both alias saved.act, and PLR_FLAGGED is
		// false for NPCs. spell_lycanthropy still sends to the NPC (which has no
		// descriptor) and sets bit 16, also MOB_NOBASH (src/utils.h:225-234;
		// src/structs.h:238-266; src/spells.c:676-680).
		sendToVictim(cvict, "You feel a strange sensation in your bones...\n\r")
		type mobFlagSetter interface{ SetMobFlag(int) }
		if setter, ok := cvict.(mobFlagSetter); ok {
			setter.SetMobFlag(16)
		}
		return
	}

	// PLR_WEREWOLF=16, PLR_VAMPIRE=17 via HasPLRFlag/SetPLRFlag interface
	type flagChecker interface {
		HasPLRFlag(int) bool
		SetPLRFlag(int)
	}
	p, ok := cvict.(flagChecker)
	if !ok {
		return
	}

	if p.HasPLRFlag(16) { // PLR_WEREWOLF
		sendToCaster(ch, "Already a werewolf.\r\n")
		return
	}
	if p.HasPLRFlag(17) { // PLR_VAMPIRE
		sendToCaster(ch, "Already a creature of the night.\n\r")
		return
	}

	sendToVictim(cvict, "You feel a strange sensation in your bones...\n\r")
	p.SetPLRFlag(16)
}

// castVampirism — sets PLR_VAMPIRE flag. PC only.
// C source: spells.c:766-794.
func castVampirism(level int, ch, cvict interface{}) {
	_ = level
	if cvict == nil {
		sendToCaster(ch, "Specify a target.\n\r")
		return
	}
	type npcTarget interface{ IsNPC() bool }
	if npc, ok := cvict.(npcTarget); ok && npc.IsNPC() {
		return
	}

	type flagChecker interface {
		HasPLRFlag(int) bool
		SetPLRFlag(int)
	}

	p, ok := cvict.(flagChecker)
	if !ok {
		return
	}

	if p.HasPLRFlag(17) { // PLR_VAMPIRE
		sendToCaster(ch, "Already a vampire.\n\r")
		return
	}
	if p.HasPLRFlag(16) { // PLR_WEREWOLF
		sendToCaster(ch, "Already a creature of the night.\n\r")
		return
	}

	sendToVictim(cvict, "You feel a strange sensation in your blood!\n\r")
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

	switch firstManualSpellArgument(arg) {
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

// firstManualSpellArgument mirrors C one_argument's leading fill-word skip and
// lowercase token handling (src/interpreter.c:1251-1281).
func firstManualSpellArgument(argument string) string {
	for {
		argument = strings.TrimLeft(argument, " \t\n\v\f\r")
		if argument == "" {
			return ""
		}
		end := strings.IndexAny(argument, " \t\n\v\f\r")
		if end < 0 {
			end = len(argument)
		}
		word := lowerManualSpellASCII(argument[:end])
		switch word {
		case "in", "from", "with", "the", "on", "at", "to":
			argument = argument[end:]
		default:
			return word
		}
	}
}

func lowerManualSpellASCII(word string) string {
	bytes := []byte(word)
	for i, b := range bytes {
		if b >= 'A' && b <= 'Z' {
			bytes[i] += 'a' - 'A'
		}
	}
	return string(bytes)
}

// castCoC — circle of summoning. Creates a COC object in the caster's room.
// C source: spells.c:1012-1039. COC_VNUM=64, timer=level/2+rand(-2,1).
func castCoC(level int, ch interface{}, world interface{}) {
	type roomGetter interface{ GetRoomVNum() int }
	type spawner interface {
		SpawnObjectToRoomForSpell(int, int) (interface{}, error)
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

	obj, err := s.SpawnObjectToRoomForSpell(CocVnum, roomVNum)
	if err != nil {
		slog.Error("spell_coc: failed to spawn COC object", "error", err, "vnum", CocVnum)
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	// Set timer
	type timerSetter interface{ SetTimer(int) }
	if ts, ok := obj.(timerSetter); ok {
		// #nosec G404 — game RNG, not cryptographic
		// #nosec G404
		timer := level/2 + dprng.Number(-2, 1)
		if timer < 1 {
			timer = 1
		}
		ts.SetTimer(timer)
	}

	sendToCaster(ch, "You draw a magic circle on the ground.\r\n")
	type roomActer interface {
		SpellActToRoom(caster, victim interface{}, format string, hideInvisible bool) bool
	}
	if acter, ok := world.(roomActer); ok {
		acter.SpellActToRoom(ch, nil, "$n draws a magic circle on the ground.", false)
	}
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
func castMentalLapse(level int, ch, cvict, world interface{}) {
	_ = level
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

	// C tests NPC status and HUNTING separately before sending the optional
	// mental-voice preamble and the target's possessive-pronoun act output.
	type npcChecker interface{ IsNPC() bool }
	isNPC := false
	if nc, ok := cvict.(npcChecker); ok {
		isNPC = nc.IsNPC()
	}

	type hunter interface {
		GetHunting() string
		ClearHunting()
	}
	h, hasHunting := cvict.(hunter)
	hunting := ""
	if hasHunting {
		hunting = h.GetHunting()
	}

	// A target already hunting the caster proceeds directly. A different
	// active target requires level 30 and gets the mental-voice preamble.
	if !isNPC || hunting == "" || hunting != casterName {
		type lever interface{ GetLevel() int }
		casterLevel := 0
		if l, ok := ch.(lever); ok {
			casterLevel = l.GetLevel()
		}
		if hunting != "" && hunting != casterName && casterLevel >= 30 {
			sendToCaster(ch, "You send out your mental voice...\r\n")
		} else {
			sendToCaster(ch, "Your psionic energy recoils!\r\n")
			return
		}
	}

	sendSpellActToCaster("You sense $S intentions even from where you are, and change $S mind.", ch, cvict, nil, world)

	if hasHunting {
		h.ClearHunting()
	}
}

// --- Wave B manual spell implementations ---

// castEnchantWeapon — adds +hitroll/+damroll to a non-magic weapon.
// C source: spells.c:621-662. Sets ITEM_MAGIC, alignment-colored glow.
func castEnchantWeapon(level int, ch, ovict, world interface{}) {
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
	newAffects[0] = parser.ObjAffect{Location: 18, Modifier: hitroll} // APPLY_HITROLL
	newAffects[1] = parser.ObjAffect{Location: 19, Modifier: damroll} // APPLY_DAMROLL
	obj.SetAffectsOverride(newAffects)

	// Alignment glow
	type aligner interface{ GetAlignment() int }
	if a, ok := ch.(aligner); ok {
		switch {
		case a.GetAlignment() > 0:
			obj.SetExtraFlag(0, 10) // ITEM_ANTI_EVIL
			sendSpellActToCaster("$p glows blue.", ch, nil, ovict, world)
		case a.GetAlignment() < 0:
			obj.SetExtraFlag(0, 9) // ITEM_ANTI_GOOD
			sendSpellActToCaster("$p glows red.", ch, nil, ovict, world)
		default:
			sendSpellActToCaster("$p glows yellow.", ch, nil, ovict, world)
		}
	}
}

// castEnchantArmor — adds AC improvement to non-magic armor/worn items.
// C source: spells.c:836-871. Sets ITEM_MAGIC, alignment-colored glow.
func castEnchantArmor(level int, ch, ovict, world interface{}) {
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
			sendSpellActToCaster("$p glows blue.", ch, nil, ovict, world)
		case a.GetAlignment() < 0:
			obj.SetExtraFlag(0, 9) // ITEM_ANTI_GOOD
			sendSpellActToCaster("$p glows red.", ch, nil, ovict, world)
		default:
			sendSpellActToCaster("$p glows yellow.", ch, nil, ovict, world)
		}
	}
}

// castCreateWater — fills a drink container with water or changes its liquid to slime.
// C source: spells.c:87-120. Drink-name helpers mutate the targeting keywords only.
func castCreateWater(level int, ch, ovict interface{}) {
	_ = level
	if ch == nil || ovict == nil {
		return
	}

	type typeFlagger interface{ GetTypeFlag() int }
	tf, ok := ovict.(typeFlagger)
	if !ok || tf.GetTypeFlag() != 17 { // ITEM_DRINKCON
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
		// C: name_from_drinkcon(); set LIQ_SLIME; name_to_drinkcon() —
		// src/spells.c:99-103 and src/act.item.c:861-896. This arm is silent.
		clearDrinkconName(ovict)
		ds.SetValue(2, 9)               // LIQ_SLIME
		setDrinkconName(ovict, "juice") // drinknames[LIQ_SLIME], src/constants.c:955
		return
	}

	// Fill with water up to capacity
	maxCap := v.GetValue(0)
	water := maxCap - current
	if water > 0 {
		ds.SetValue(2, 0) // LIQ_WATER
		ds.SetValue(1, current+water)
		if weightSetter, ok := ovict.(interface {
			GetWeight() int
			SetWeight(int)
		}); ok {
			weightSetter.SetWeight(weightSetter.GetWeight() + water)
		}
		clearDrinkconName(ovict)
		setDrinkconName(ovict, "water") // drinknames[LIQ_WATER], src/constants.c:946
		if desc, ok := ovict.(interface{ GetShortDesc() string }); ok {
			// C act("$p is filled.", ..., TO_CHAR) uses OBJS(obj, ch),
			// so the caster sees the object's short description (src/comm.c:2439).
			sendToCaster(ch, fmt.Sprintf("%s is filled.\r\n", desc.GetShortDesc()))
		}
	} else {
		sendToCaster(ch, "You cannot create water in that!\r\n")
	}
}

// clearDrinkconName and setDrinkconName mirror the keyword-only mutations of
// C's name_from_drinkcon/name_to_drinkcon; neither changes the short description.
func clearDrinkconName(ovict interface{}) {
	type keywordEditor interface {
		GetKeywords() string
		SetLiveKeywords(string)
	}
	if obj, ok := ovict.(keywordEditor); ok {
		keywords := obj.GetKeywords()
		if space := strings.IndexByte(keywords, ' '); space >= 0 {
			obj.SetLiveKeywords(keywords[space+1:])
		}
	}
}

func setDrinkconName(ovict interface{}, liquidKeyword string) {
	type keywordEditor interface {
		GetKeywords() string
		SetLiveKeywords(string)
	}
	if obj, ok := ovict.(keywordEditor); ok {
		obj.SetLiveKeywords(liquidKeyword + " " + obj.GetKeywords())
	}
}

// castIdentify — reveals item stats or PC stats.
// C source: spells.c:476-621. Simplified Go implementation.
func castIdentify(level int, ch, cvict, ovict, world interface{}) {
	if ovict != nil {
		castIdentifyObject(level, ch, ovict)
	} else if cvict != nil {
		castIdentifyCharacter(level, ch, cvict, world)
	}
}

func castIdentifyObject(level int, ch, ovict interface{}) {
	_ = level
	type typeFlagger interface{ GetTypeFlag() int }
	type valuer interface{ GetValue(int) int }
	type affecter interface{ GetAffects() []parser.ObjAffect }
	type namer interface{ GetName() string }
	type shortDescer interface{ GetShortDesc() string }
	type weighter interface{ GetWeight() int }
	type coster interface{ GetCost() int }
	type extraFlagger interface{ GetExtraFlags() [4]int }

	sendToCaster(ch, "You feel informed:\r\n")

	objectName := ""
	if d, ok := ovict.(shortDescer); ok {
		objectName = d.GetShortDesc()
	}
	if objectName == "" {
		if n, ok := ovict.(namer); ok {
			objectName = n.GetName()
		}
	}
	itemType := -1
	if tf, ok := ovict.(typeFlagger); ok {
		itemType = tf.GetTypeFlag()
	}
	typeName := identifyItemTypeName(itemType)
	sendToCaster(ch, fmt.Sprintf("Object '%s', Item type: %s\r\n", objectName, typeName))

	// C's object bitvector is not exposed by the Go object interface yet; the
	// parsed object format has no corresponding field, so this is NOBITS for
	// current instances and preserves the live C output for ordinary objects.
	sendToCaster(ch, "Item will give you following abilities:  "+identifyBits([4]int{}, identifyAffectedBitNames)+"\r\n")
	extra := [4]int{}
	if ef, ok := ovict.(extraFlagger); ok {
		extra = ef.GetExtraFlags()
	}
	sendToCaster(ch, "Item is: "+identifyBits(extra, identifyExtraBitNames)+"\r\n")

	weight := 0
	if w, ok := ovict.(weighter); ok {
		weight = w.GetWeight()
	}
	cost := 0
	if c, ok := ovict.(coster); ok {
		cost = c.GetCost()
	}
	sendToCaster(ch, fmt.Sprintf("Encumbrance: %d, Value: %d\r\n", weight, cost))

	v, _ := ovict.(valuer)
	if v != nil {
		switch itemType {
		case 2, 10: // ITEM_SCROLL, ITEM_POTION
			var spells string
			for i := 1; i <= 3; i++ {
				if value := v.GetValue(i); value >= 1 {
					spells += " " + identifySpellName(value)
				}
			}
			sendToCaster(ch, fmt.Sprintf("This %s casts: %s\r\n", typeName, spells))
		case 3, 4: // ITEM_WAND, ITEM_STAFF
			sendToCaster(ch, fmt.Sprintf("This %s casts: %s\r\nIt has %d maximum charge%s and %d remaining.\r\n",
				typeName, identifySpellName(v.GetValue(3)), v.GetValue(1), pluralS(v.GetValue(1)), v.GetValue(2)))
		case 5: // ITEM_WEAPON
			sendToCaster(ch, fmt.Sprintf("Damage Dice is '%dD%d' for an average per-round damage of %.1f.\r\n",
				v.GetValue(1), v.GetValue(2), ((float64(v.GetValue(2)+1)/2.0)*float64(v.GetValue(1)))))
		case 9: // ITEM_ARMOR
			sendToCaster(ch, fmt.Sprintf("AC-apply is %d\r\n", v.GetValue(0)))
		}
	}

	if a, ok := ovict.(affecter); ok {
		found := false
		for _, aff := range a.GetAffects() {
			if aff.Location == 0 || aff.Modifier == 0 {
				continue
			}
			if !found {
				sendToCaster(ch, "Can affect you as :\r\n")
				found = true
			}
			switch aff.Location {
			case 29: // APPLY_SPELL
				sendToCaster(ch, fmt.Sprintf("   Permanent %s when equipped.\r\n", identifyAffectedBitName(aff.Modifier)))
			case 25: // APPLY_RACE_HATE
				sendToCaster(ch, fmt.Sprintf("   Extra damage to: %ss.\r\n", identifyMobRaceName(aff.Modifier)))
			default:
				sendToCaster(ch, fmt.Sprintf("   Affects: %s By %d\r\n", identifyApplyTypeName(aff.Location), aff.Modifier))
			}
		}
	}
}

var identifyItemTypes = []string{
	"UNDEFINED", "LIGHT", "SCROLL", "WAND", "STAFF", "WEAPON", "FIRE WEAPON", "MISSILE",
	"TREASURE", "ARMOR", "POTION", "WORN", "OTHER", "TRASH", "TRAP", "CONTAINER",
	"NOTE", "LIQ CONTAINER", "KEY", "FOOD", "MONEY", "PEN", "BOAT", "FOUNTAIN",
}

var identifyExtraBitNames = []string{
	"GLOW", "HUM", "!RENT", "!DONATE", "!INVIS", "INVIS", "MAGIC", "!DROP", "BLESS", "!GOOD",
	"!EVIL", "!NEU", "!MAGE", "!CLE", "!THI", "!WAR", "!SELL", "NAMED", "!PSI", "!NIN",
	"!PAL", "!MAGUS", "!ASS", "!AVA", "RARE", "!LOCATE", "!RAN", "!MYS", "TWOHANDS",
}

var identifyAffectedBitNames = []string{
	"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE", "WATERWALK", "SANCT",
	"GROUP", "CURSE", "INFRA", "POISON", "PROT-EVIL", "PROT-GOOD", "SLEEP", "!TRACK", "FLESH-ALTER",
	"DODGE", "SNEAK", "HIDE", "BERSERK", "CHARM", "FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY",
	"WEREWOLF", "VAMPIRE", "MOUNTED", "INVULN", "FLAMING", "NOTHING", "HASTE", "SLOW", "DREAM",
	"WATERBREATHE", "METALSKIN", "ROBBED",
}

var identifyApplyTypes = []string{
	"NONE", "STR", "DEX", "INT", "WIS", "CON", "CHA", "CLASS", "LEVEL", "AGE", "CHAR_WEIGHT",
	"CHAR_HEIGHT", "MAXMANA", "MAXHIT", "MAXMOVE", "GOLD", "EXP", "ARMOR", "HITROLL", "DAMROLL",
	"SAVING_PARA", "SAVING_ROD", "SAVING_PETRI", "SAVING_BREATH", "SAVING_SPELL", "RACE_HATE", "HIT_REGEN",
	"MANA_REGEN", "MOVE_REGEN", "PERM_SPELL",
}

var identifyMobRaces = []string{
	"Human", "Elf", "Dwarf", "Kender", "Centaur", "Rakshasa", "Troll", "Lycanthrope", "Vampire",
	"Undead", "Dragon", "Demon", "Horse", "Reptile", "Arachnid", "Rodent", "Other", "Vegetable",
	"Giant", "Demi-god", "Ogre", "Insect", "Mammal", "Fish", "Avian", "Magical Construct", "Amphibian",
	"Humanoid", "Faery", "Ssaur", "Minotaur",
}

func identifyBits(flags [4]int, names []string) string {
	result := ""
	for word, flag := range flags {
		bits := identifyFlagWord(flag)
		for bit := 0; bits != 0; bit++ {
			if bits&1 != 0 {
				index := word*32 + bit
				if index < len(names) && names[index] != "" {
					result += names[index] + " "
				} else {
					result += "UNDEFINED "
				}
			}
			bits >>= 1
		}
	}
	if result == "" {
		return "NOBITS "
	}
	return result
}

func identifyFlagWord(flag int) uint32 {
	if flag < 0 || uint64(flag) > uint64(^uint32(0)) {
		return 0
	}
	return uint32(flag)
}

func identifyItemTypeName(itemType int) string {
	if itemType >= 0 && itemType < len(identifyItemTypes) {
		return identifyItemTypes[itemType]
	}
	return "UNDEFINED"
}

func identifySpellName(spellNum int) string {
	name := SpellRawName(spellNum)
	if name == "" {
		return "UNDEFINED"
	}
	return name
}

func identifyAffectedBitName(bit int) string {
	if bit >= 0 && bit < len(identifyAffectedBitNames) {
		return identifyAffectedBitNames[bit]
	}
	return "UNDEFINED"
}

func identifyApplyTypeName(location int) string {
	if location >= 0 && location < len(identifyApplyTypes) {
		return identifyApplyTypes[location]
	}
	return "UNDEFINED"
}

func identifyMobRaceName(race int) string {
	if race >= 0 && race < len(identifyMobRaces) {
		return identifyMobRaces[race]
	}
	return "UNDEFINED"
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func castIdentifyCharacter(level int, ch, cvict, world interface{}) {
	_ = level
	type lever interface{ GetLevel() int }
	type npcChecker interface{ IsNPC() bool }
	type identifyReport interface {
		GetName() string
		GetAgeParts() (int, int, int, int)
		GetHeight() int
		GetWeight() int
		GetLevel() int
		GetHP() int
		GetMana() int
		GetAC() int
		GetHitroll() int
		GetDamroll() int
		GetStr() int
		GetStrAdd() int
		GetInt() int
		GetWis() int
		GetDex() int
		GetCon() int
		GetCha() int
	}

	// C: identify on NPCs fails and aggros
	if nc, ok := cvict.(npcChecker); ok && nc.IsNPC() {
		sendToCaster(ch, "The magicks fail horribly!\r\n")
		npcRetaliate(world, cvict, ch)
		return
	}

	// C: identify on PCs level <= 5 is blocked
	if l, ok := cvict.(lever); ok {
		if nc, ok := cvict.(npcChecker); ok && !nc.IsNPC() && l.GetLevel() <= 5 {
			sendToCaster(ch, "You cannot identify them yet.\r\n")
			return
		}
	}

	if report, ok := cvict.(identifyReport); ok {
		sendToCaster(ch, fmt.Sprintf("Name: %s\r\n", report.GetName()))
		years, months, days, hours := report.GetAgeParts()
		sendToCaster(ch, fmt.Sprintf("%s is %d years, %d months, %d days and %d hours old.\r\n", report.GetName(), years, months, days, hours))
		sendToCaster(ch, fmt.Sprintf("Height %d cm, Weight %d pounds\r\n", report.GetHeight(), report.GetWeight()))
		sendToCaster(ch, fmt.Sprintf("Level: %d, Hits: %d, Mana: %d\r\n", report.GetLevel(), report.GetHP(), report.GetMana()))
		sendToCaster(ch, fmt.Sprintf("AC: %d, Hitroll: %d, Damroll: %d\r\n", report.GetAC(), report.GetHitroll(), report.GetDamroll()))
		sendToCaster(ch, fmt.Sprintf("Str: %d/%d, Int: %d, Wis: %d, Dex: %d, Con: %d, Cha: %d\r\n", report.GetStr(), report.GetStrAdd(), report.GetInt(), report.GetWis(), report.GetDex(), report.GetCon(), report.GetCha()))
	} else {
		// Real C player characters expose every GET_* field above. Keep an
		// explicit fallback for external test or integration character values.
		if name, ok := cvict.(interface{ GetName() string }); ok {
			sendToCaster(ch, "You feel informed:\r\n")
			sendToCaster(ch, fmt.Sprintf("Name: %s\r\n", name.GetName()))
		}
	}
}

// --- Room transfer spells ---

// Interfaces for room transfer spells.
type (
	roomGetter2 interface{ GetRoomVNum() int }

	// grouper for are_grouped checks
	grouper interface {
		IsInGroup() bool
		GetFollowing() string
		GetName() string
	}

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
// C src/utils.c:655-674 checks AFF_GROUP and the root/immediate-follower edge.
// Live World supplies retained master bodies; standalone legacy adapters retain
// their name-backed relation below.
func areGrouped(ch, victim interface{}, worlds ...interface{}) bool {
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
	// Live World retains actual master bodies. C src/utils.c:655-674 checks
	// the caster's master (or caster) and that body's immediate followers.
	if len(worlds) != 0 {
		if w, ok := worlds[0].(interface {
			FollowingBody(combat.Combatant) combat.Combatant
		}); ok {
			a, aok := ch.(combat.Combatant)
			b, bok := victim.(combat.Combatant)
			if !aok || !bok {
				return false
			}
			leader := w.FollowingBody(a)
			if leader == nil {
				leader = a
			}
			return b == leader || w.FollowingBody(b) == leader
		}
	}
	// Legacy standalone spell adapters do not expose retained bodies.
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

	// C spell_recall (spells.c:124-165): NPC or missing victims return
	// silently; every gate byte, the hometown transfer, and the room-look
	// landing live in the world adapter (game.ExecuteWordOfRecall) — the
	// spells package cannot import game. No invented failure text here.
	type npcChecker interface{ IsNPC() bool }
	if cvict == nil {
		return
	}
	if v, ok := cvict.(npcChecker); ok && v.IsNPC() {
		return
	}
	if w, ok := world.(interface {
		ExecuteWordOfRecall(ch, victim interface{})
	}); ok {
		w.ExecuteWordOfRecall(ch, cvict)
	}
}

// spell_teleport ports src/spells.c spell_teleport (lines 168–217).
// Random room teleport. Self-only for PCs. Different NPC victims get a saving
// throw; C's self-NPC teleporter vehicle does not.
// Can't use in peaceful rooms. Avoids PRIVATE rooms.
func castTeleport(level int, ch, cvict, world interface{}) {
	_ = level

	type teleportWorld interface {
		GetRoomInWorld(vnum int) *parser.Room
		GetRoomCount() int
	}
	w, ok := world.(teleportWorld)
	if !ok {
		sendToCaster(ch, "Teleport failed: world interface not available.\r\n")
		return
	}
	type characterTransferer interface {
		CharTransfer(charName string, isMob bool, toRoomVNum int) error
	}
	transferer, nameTransferOK := world.(characterTransferer)
	bodyTransfer, bodyOK := world.(interface {
		TransferCombatant(combat.Combatant, int) error
	})
	if !nameTransferOK && !bodyOK {
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

	// NPC victims other than the caster get a saving throw; a made save
	// refuses the teleport and the victim counterattacks
	// (src/spells.c:183-187). C's self-teleporting special passes the same
	// NPC as caster and victim, so that path skips this branch.
	if vNPC, ok := cvict.(npcChecker); ok && vNPC.IsNPC() && !selfTarget(ch, cvict) {
		if magSavingThrow(cvict, int(SaveSpell)) {
			sendToCaster(ch, "The magic words fail to form properly.\r\n")
			spellCounterattack(world, cvict, ch)
			return
		}
		// A below-immortal caster whose NPC victim is within two levels of
		// it fails utterly (src/spells.c:189-194): the refusal line to the
		// caster and the same victim counterattack, never the teleport.
		if getLevel(ch) < combat.LVL_IMMORT && (getLevel(ch)-getLevel(cvict)) < 3 {
			sendToCaster(ch, "The magic fails utterly!\r\n")
			spellCounterattack(world, cvict, ch)
			return
		}
	}

	// Pick a random room, avoiding PRIVATE
	roomCount := w.GetRoomCount()
	if roomCount <= 0 {
		return
	}
	type roomIndexer interface {
		GetRoomVNumAtIndex(index int) (int, bool)
	}
	indexedWorld, hasRoomIndex := w.(roomIndexer)
	for {
		// #nosec G404 — game RNG, not cryptographic
		roomIndex := dprng.Number(0, roomCount-1)
		toRoom := roomIndex
		if hasRoomIndex {
			var found bool
			toRoom, found = indexedWorld.GetRoomVNumAtIndex(roomIndex)
			if !found {
				continue
			}
		}
		roomData := w.GetRoomInWorld(toRoom)
		if roomData != nil && !roomData.HasFlag(RoomPrivate) {
			sendToVictim(cvict, "The world around you turns black and you suddenly find yourself..\r\n")
			sendAffectRoom(cvict, nil, "$n slowly fades out of existence and is gone.", false, world)

			// Live worlds transfer the supplied concrete combat body. Legacy test
			// adapters retain their existing name-taking bridge.
			victName, ok := cvict.(interface{ GetName() string })
			if !ok {
				return
			}
			isMob := false
			if vNPC, ok := cvict.(npcChecker); ok {
				isMob = vNPC.IsNPC()
			}
			var transferErr error
			if body, valid := cvict.(combat.Combatant); valid && bodyOK {
				transferErr = bodyTransfer.TransferCombatant(body, toRoom)
			} else if nameTransferOK {
				transferErr = transferer.CharTransfer(victName.GetName(), isMob, toRoom)
			}
			if err := transferErr; err != nil {
				slog.Error("CharTransfer failed", "error", err)
			}
			sendAffectRoom(cvict, nil, "$n slowly fades into existence.", false, world)
			// C calls look_at_room(victim, 0) after the destination act. The
			// game adapter owns the player-facing room renderer because spells
			// cannot import pkg/game without creating a cycle.
			if looker, ok := world.(interface{ LookAtRoomForSpell(victim interface{}) }); ok {
				looker.LookAtRoomForSpell(cvict)
			}
			return
		}
	}
}

// castMeteorSwarm ports src/spells.c spell_meteor_swarm (lines 1088-1133).
// AoE damage to all non-grouped characters in room. Must be outdoors.
func castMeteorSwarm(level int, ch, world interface{}) {
	// C src/spells.c:1094 consumes this damage draw before the peaceful and
	// outdoor refusal gates.
	// #nosec G404 -- game RNG, not cryptographic
	dam := level*6 + dprng.Number(-10, level*3)

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

	sendToCaster(ch, "Your incantation brings the heavens down upon your victims!\r\n")
	const roomAct = "As $n finishes $s incantation, bright lights from the heavens streak downward towards you."
	type roomActer interface {
		SpellActToRoom(actor, victim interface{}, format string, hideInvisible bool) bool
	}
	if acter, ok := world.(roomActer); ok {
		// C src/spells.c:1110 uses act(..., TRUE, ..., TO_ROOM).
		acter.SpellActToRoom(ch, nil, roomAct, true)
	}

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
		// Skip immortals (non-NPCs, leveled at LVL_IMMORT or above).
		// C spell_meteor_swarm: !IS_NPC(tch) && GET_LEVEL(tch) >= LVL_IMMORT
		// (src/spells.c:1120).
		if nc, ok := c.(npcChecker2); ok && !nc.IsNPC() && cn.GetLevel() >= combat.LVL_IMMORT {
			continue
		}
		// Skip grouped
		if areGrouped(ch, c, world) {
			continue
		}
		w.DoSpellDamage(ch, c, dam, "meteor swarm")
	}

	_ = casterName
	_ = casterLevel
}

// castHellfire ports src/spells.c spell_hellfire (lines 701-767).
// AoE fire damage. DEX check can knock targets to sitting.
// Uses fixed dice (12d5) matching C source, not level-scaled dice.
// C iterates the global character_list, so preserve its ordering before the
// source's same-room predicate; room-entry order is not equivalent here.
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

	dam := dice(12, 5) + (2 * level) - 10
	casterDex := 0
	if dexGetter, ok := ch.(interface{ GetDex() int }); ok {
		casterDex = dexGetter.GetDex()
	}

	sendToCaster(ch, "The bowels of hell open beneath your feet!!\r\n")
	if zoneSender, ok := world.(interface{ SendToZone(int, string) }); ok {
		zoneSender.SendToZone(roomVNum, "The shrieks of demons accompany the sounds of fire..\r\n")
	}
	spellActToRoom(world, ch, "$n conjures the flames of hell upon $s enemies!!", true)

	chars := w.GetAllCharsInRoom(roomVNum)
	if ordered, ok := world.(interface{ GetAllCharsInCharacterListOrder() []interface{} }); ok {
		chars = ordered.GetAllCharsInCharacterListOrder()
	}
	for _, c := range chars {
		if c == ch {
			continue
		}
		cn, ok := c.(charInRoom)
		if !ok {
			continue
		}
		if cn.GetRoomVNum() != roomVNum {
			continue
		}
		// Skip grouped
		if areGrouped(ch, c, world) {
			continue
		}

		isNPC := false
		if nc, ok := c.(npcChecker2); ok {
			isNPC = nc.IsNPC()
		}
		if cn.GetLevel() >= combat.LVL_IMMORT && !isNPC {
			castHellfireActToVictim(world, ch, c, "The flames of hell spring up around you!!\n\r")
			continue
		}

		if cn.GetLevel() <= 4 {
			castHellfireActToVictim(world, ch, c, "The fires of hell overcome you!!")
			castHellfireActNotVictim(world, ch, c, "The soul of $N is charred by the flames of hell!", false)
			w.DoSpellDamage(ch, c, cn.GetMaxHP()*12, "hellfire")
		} else {
			castHellfireActToVictim(world, ch, c, "The fires of hell bring blisters on your skin!!\n\r")
			castHellfireActNotVictim(world, ch, c, "The fires of hell scorch the skin of $N!\r\n", false)
			// C source: spells.c:737-742 uses number(1, 20) before the damage call.
			// #nosec G404 — game RNG, not cryptographic
			if dprng.Number(1, 20) > casterDex {
				cn.SetPosition(5) // POS_SITTING
				castHellfireActToVictim(world, ch, c, "The fires of hell bring you to your knees!\r\n")
				castHellfireActNotVictim(world, ch, c, "The fires of hell bring $N to $S knees!\r\n", false)
			}
			w.DoSpellDamage(ch, c, dam, "hellfire")
		}
	}
}

func castHellfireActToVictim(world, ch, victim interface{}, format string) {
	type victimActer interface {
		SpellActToVictim(caster, target interface{}, format string) bool
	}
	if acter, ok := world.(victimActer); ok && acter.SpellActToVictim(ch, victim, format) {
		return
	}
	if sender, ok := victim.(interface{ SendMessage(string) }); ok {
		sender.SendMessage(format + "\r\n")
	}
}

func castHellfireActNotVictim(world, ch, victim interface{}, format string, hideInvisible bool) {
	type notVictimActer interface {
		SpellActNotVict(caster, target interface{}, format string, hideInvisible bool) bool
	}
	if acter, ok := world.(notVictimActer); ok {
		acter.SpellActNotVict(ch, victim, format, hideInvisible)
	}
}

// castCharm ports src/spells.c spell_charm (lines 407-471), including C's
// sanctuary, no-charm, existing-charm, shopkeeper, circle, save, follower-cap,
// outlaw, and successful act branches.
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

	// C compares character pointers here; equal display names do not make two
	// distinct bodies a self-target.
	if ch == cvict {
		sendToCaster(ch, "You like yourself even better!\r\n")
		return
	}

	victIsNPC := true
	if nc, ok := cvict.(npcCheck); ok {
		victIsNPC = nc.IsNPC()
	}
	type affectChecker interface{ IsAffected(bit int) bool }
	if affected, ok := cvict.(affectChecker); ok && affected.IsAffected(affSanctuaryBit) {
		sendToCaster(ch, "Your victim is protected by sanctuary!\r\n")
		return
	}

	// src/structs.h:260 defines MOB_NOCHARM at bit 13. Use the C bit-position
	// mask from GetMobFlags, as MOB_FLAGGED does in the oracle.
	if victIsNPC && hasSummonMobFlag(cvict, mobNoCharmBit) {
		sendToCaster(ch, "Your victim resists!\r\n")
		return
	}
	if affected, ok := ch.(affectChecker); ok && affected.IsAffected(affCharmBit) {
		sendToCaster(ch, "You can't have any followers of your own!\r\n")
		return
	}
	if affected, ok := cvict.(affectChecker); ok && affected.IsAffected(affCharmBit) {
		sendToCaster(ch, "You fail.\r\n")
		return
	}

	// Level check
	if victL, ok := cvict.(lever); ok {
		if level < victL.GetLevel() {
			sendToCaster(ch, "You fail.\r\n")
			return
		}
	}
	if victIsNPC {
		type shopKeeperChecker interface{ IsSpellCharmShopkeeper(victim interface{}) bool }
		if checker, ok := world.(shopKeeperChecker); ok && checker.IsSpellCharmShopkeeper(cvict) {
			sendToCaster(ch, "Ha ha... Don't think so.\r\n")
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

	// C spell_charm checks the victim's saving throw for both NPCs and PCs
	// (spells.c:430-444). A PC victim is only gated by the caster's outlaw flag;
	// an outlaw can establish the same master relation that do_hit later tests.
	if magCharmSaveRoll(cvict, int(SaveParalysis)) {
		sendToCaster(ch, "Your victim resists!\r\n")
		// C: spells.c:430-433 — a resisting NPC immediately attacks the caster.
		npcRetaliate(world, cvict, ch)
		return
	}

	// Max followers check (CHA/2)
	if fw, ok := world.(followerWorld); ok {
		chCha := 10
		if ci, ok := ch.(interface{ GetCha() int }); ok {
			chCha = ci.GetCha()
		}
		if fw.NumFollowers(chName) >= chCha/2 {
			sendToCaster(ch, "You can't have any more followers!\r\n")
			return
		}
	}

	if !victIsNPC && !hasOutlawFlag(ch) {
		// C's text contains the original typo: "in not an Outlaw".
		sendToCaster(ch, "Your power fails to effect them because you are not an Outlaw!\r\n")
		if caster, ok := ch.(namer); ok {
			sendToVictim(cvict, fmt.Sprintf("%s tried to control you but failed because %s in not an Outlaw!\r\n", caster.GetName(), caster.GetName()))
		}
		return
	}

	// Stop victim's current following, add as our follower
	if fw, ok := world.(followerWorld); ok {
		fw.StopFollowerByName(victName)
	}
	if adder, ok := world.(interface {
		AddFollowerForSpell(follower, leader interface{}) bool
	}); !ok || !adder.AddFollowerForSpell(cvict, ch) {
		if fw, ok := world.(followerWorld); ok {
			fw.AddFollowerQuiet(cvict, ch)
		}
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

	const successAct = "Isn't $n just such a nice fellow?"
	if renderer, ok := world.(interface {
		SpellActToVictim(caster, victim interface{}, format string) bool
	}); !ok || !renderer.SpellActToVictim(ch, cvict, successAct) {
		sendAct(successAct, ch, nil, cvict, world)
	}
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

	chName := ""
	if n, ok := ch.(namer); ok {
		chName = n.GetName()
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

	// src/utils.c:877-893: circle_check runs for every caster and consumes one
	// tick from the first summoning circle in the caster's room. Only PCs need
	// a circle to continue.
	if !consumeSummoningCircle(chRoom, world) && !chIsNPC {
		sendToCaster(ch, "You lack the proper glyphs to perform this magick.\r\n")
		return
	}
	if !chIsNPC && !victIsNPC && getLevel(ch) < combat.LVL_IMMORT && !isSummonable(ch) {
		sendToCaster(ch, "Sorry, but you need to be summonable to summon other players.\r\n")
		return
	}

	// C caps the victim at the lower of level+3 and LVL_IMMORT-1.
	if vl, ok := cvict.(lever); ok {
		victLevel := vl.GetLevel()
		maxVictimLevel := combat.LVL_IMMORT - 1
		if level+3 < maxVictimLevel {
			maxVictimLevel = level + 3
		}
		if victLevel > maxVictimLevel {
			sendToCaster(ch, "You failed.\r\n")
			if vs, ok := cvict.(interface{ SendMessage(string) }); ok {
				vs.SendMessage(summonAttemptNotice(ch, chName, chRoom, world))
			}
			return
		}
	}

	// MOB_NOSUMMON check
	if victIsNPC && hasSummonMobFlag(cvict, mobNoSummonBit) {
		sendToCaster(ch, "You failed.\r\n")
		return
	}
	// src/spells.c:261-266 refuses a mortal PC caster's attempt to summon a
	// non-summonable, non-outlaw PC before the later save and transfer paths.
	if !chIsNPC && !victIsNPC && !isSummonable(cvict) && getLevel(ch) < combat.LVL_IMMORT && !hasOutlawFlag(cvict) {
		sendToCaster(ch, "You failed.\r\n")
		return
	}
	// C src/spells.c:268-279 gives an unsummonable outlaw PC a 36% chance to
	// resist before the exit check. The roll is unconditional for this class,
	// independent of the caster's level or summonable preference.
	if !chIsNPC && !victIsNPC && hasOutlawFlag(cvict) && !isSummonable(cvict) && magSummonOutlawResistanceRoll() >= 65 {
		sendToCaster(ch, "You failed.\r\n")
		if vs, ok := cvict.(interface{ SendMessage(string) }); ok {
			vs.SendMessage(summonAttemptNotice(ch, chName, chRoom, world))
		}
		return
	}

	// C's later refusal expression checks the room states and the caster exit
	// after taking the victim save, so those conditions participate in the
	// same backfire branch instead of returning early.
	type worldRoom interface{ GetRoomInWorld(vnum int) *parser.Room }
	var chRoomData, victRoomData *parser.Room
	if wr, ok := world.(worldRoom); ok {
		chRoomData = wr.GetRoomInWorld(chRoom)
		victRoomData = wr.GetRoomInWorld(victRoom)
	}

	// C accepts any existing, non-NOWHERE, open exit from the caster's room.
	roomOK := false
	if chRoomData != nil {
		for _, exit := range chRoomData.Exits {
			if exit.ToRoom != -1 && exit.ExitInfo&parser.ExitClosed == 0 {
				roomOK = true
				break
			}
		}
	}

	// src/spells.c:290-297 evaluates these gates in order after the save.
	summonRefused := (magSummonSaveRoll(cvict, int(SaveSpell)) && !isSummonable(cvict)) ||
		(chRoomData != nil && chRoomData.HasFlag(RoomPeaceful) && victIsNPC) ||
		(victRoomData != nil && victRoomData.HasFlag(RoomNoMagic)) ||
		(victIsNPC && hasSummonMobFlag(cvict, mobNoCharmBit)) ||
		!roomOK
	if summonRefused {
		// 10% backfire chance for PC casters
		// #nosec G404 — game RNG, not cryptographic
		backfireRoll := magSummonBackfireRoll()
		if backfireRoll == 0 && !chIsNPC {
			sendToCaster(ch, "Your spell backfires!\r\n")
			sendAffectRoom(ch, nil, "$n disappears suddenly.\r\n", true, world)
			// Transfer caster to victim's room instead
			if err := transferSummonCharacter(world, ch, victRoom); err != nil {
				slog.Error("summon backfire transfer failed", "error", err)
			}
			sendAffectRoom(ch, nil, "$n arrives suddenly.\r\n", true, world)
			if looker, ok := world.(interface{ LookAtRoomForSpell(victim interface{}) }); ok {
				looker.LookAtRoomForSpell(ch)
			}
			if victIsNPC && getPos(cvict) > int(PosSleeping) {
				npcRetaliate(world, cvict, ch)
			}
			return
		}
		sendToCaster(ch, "You failed.\r\n")
		if vs, ok := cvict.(interface{ SendMessage(string) }); ok {
			vs.SendMessage(summonAttemptNotice(ch, chName, chRoom, world))
		}
		return
	}

	// Success — transfer victim to caster's room
	sendAffectRoom(cvict, nil, "$n disappears suddenly.\r\n", true, world)
	if err := transferSummonCharacter(world, cvict, chRoom); err != nil {
		slog.Error("summon transfer failed", "error", err)
	}
	sendAffectRoom(cvict, nil, "$n arrives suddenly.\r\n", true, world)

	if getPos(cvict) > int(PosSleeping) {
		sendToVictim(cvict, fmt.Sprintf("%s has summoned you!\r\n", chName))
		// C calls look_at_room(victim, 0) immediately after the awake victim
		// message; the game adapter owns the room renderer to avoid a package
		// cycle from spells to game.
		if looker, ok := world.(interface{ LookAtRoomForSpell(victim interface{}) }); ok {
			looker.LookAtRoomForSpell(cvict)
		}
		// spells.c:347-349: an awake NPC victim attacks the caster right
		// after its arrival look.
		npcRetaliate(world, cvict, ch)
	} else {
		sendToVictim(cvict, "You have a strange dream about falling...\r\n")
	}
}

// transferSummonCharacter uses the live world's interface{} adapter. The
// concrete PlayerTransfer and MobTransfer methods take pointer types, so a
// widened interface assertion silently skipped both transfers before DP-1450.
func transferSummonCharacter(world, actor interface{}, roomVNum int) error {
	type spellTransferWorld interface {
		CharacterTransferForSpell(actor interface{}, toRoomVNum int) error
	}
	transfer, ok := world.(spellTransferWorld)
	if !ok {
		return fmt.Errorf("spell world does not support summon transfer")
	}
	return transfer.CharacterTransferForSpell(actor, roomVNum)
}

// consumeSummoningCircle mirrors utils.c:circle_check. C decrements the first
// room circle's timer at spell use, even when a later summon gate fails.
func consumeSummoningCircle(roomVNum int, world interface{}) bool {
	type roomObjectLister interface{ GetItemsInRoomI(int) []interface{} }
	type vnumGetter interface{ GetVNum() int }
	type timer interface {
		GetTimer() int
		SetTimer(int)
	}

	lister, ok := world.(roomObjectLister)
	if !ok {
		return false
	}
	for _, object := range lister.GetItemsInRoomI(roomVNum) {
		vnum, ok := object.(vnumGetter)
		if !ok || vnum.GetVNum() != CocVnum {
			continue
		}
		if timed, ok := object.(timer); ok {
			timed.SetTimer(timed.GetTimer() - 1)
		}
		return true
	}
	return false
}

const (
	affSanctuaryBit  = 7  // src/structs.h:317, AFF_SANCTUARY
	affCharmBit      = 21 // src/structs.h:331, AFF_CHARM
	mobNoCharmBit    = 13 // src/structs.h:260, MOB_NOCHARM
	mobNoSummonBit   = 14 // src/structs.h:261, MOB_NOSUMMON
	prfSummonableBit = 42 // PRF_SUMMONABLE=10 at game.prfBase=32
)

func hasSummonMobFlag(ch interface{}, bit uint) bool {
	type mobFlagReader interface{ GetMobFlags() uint64 }
	mob, ok := ch.(mobFlagReader)
	return ok && mob.GetMobFlags()&(uint64(1)<<bit) != 0
}

func isSummonable(ch interface{}) bool {
	type flagReader interface{ GetFlags() uint64 }
	flags, ok := ch.(flagReader)
	return ok && flags.GetFlags()&(uint64(1)<<prfSummonableBit) != 0
}

func summonAttemptNotice(ch interface{}, chName string, roomVNum int, world interface{}) string {
	roomName := ""
	if roomWorld, ok := world.(interface{ GetRoomInWorld(int) *parser.Room }); ok {
		if room := roomWorld.GetRoomInWorld(roomVNum); room != nil {
			roomName = room.Name
		}
	}
	if chName != "" {
		chName = strings.ToUpper(chName[:1]) + chName[1:]
	}
	pronoun := "She"
	if sex, ok := ch.(interface{ GetSex() int }); ok && sex.GetSex() == 0 {
		pronoun = "He"
	}
	return fmt.Sprintf("%s just tried to summon you to: %s.\r\n%s failed.\r\n", chName, roomName, pronoun)
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

	good := alignment >= 350
	evil := alignment <= -350
	if !good && !evil {
		sendToCaster(ch, "Your request for intervention falls on deaf ears.\r\n")
		return
	}

	mobVNum := 86
	if good {
		mobVNum = 85
	}

	sendToCaster(ch, "You pray for the intervention of your deity.\r\n")
	spellActToRoom(world, ch, "$n grovels in the dirt and prays for a miracle.", true)

	// C emits the prayer before this save. The unusual branch succeeds only
	// when the caster saves: src/spells.c:1195-1200.
	if !magDivineIntSaveRoll(ch, int(SaveSpell)) {
		sendToCaster(ch, "Nothing seems to happen.\r\n")
		return
	}

	roomVNum := 0
	if rg, ok := ch.(roomGet); ok {
		roomVNum = rg.GetRoomVNum()
	}
	angelCount := 1
	if alignment == -1000 || alignment == 1000 {
		angelCount++
	}

	sendToCaster(ch, "Suddenly, a portal of light appears out of nowhere!\r\n")
	spellActToRoom(world, ch, "Suddenly, a portal of light appears out of nowhere!", false)

	type mobSpawner interface {
		SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error)
	}
	spawner, ok := world.(mobSpawner)
	if !ok {
		return
	}

	type followerAdder interface{ AddFollowerQuiet(ch, leader interface{}) }
	fa, _ := world.(followerAdder)

	for i := 0; i < angelCount; i++ {
		mobLevel := level / 2
		mob, err := spawner.SpawnMobWithLevelI(mobVNum, roomVNum, mobLevel)
		if err != nil {
			slog.Warn("spell_divine_int: failed to spawn angel", "error", err, "vnum", mobVNum)
			continue
		}
		spellActToRoom(world, mob, "$n steps through the portal!", true)
		applyAffect(mob, engine.NewAffectDirect(SpellDivineInt, engine.ApplyNone, level/2-2, 0, engine.AFFCharm, "divine intervention"))
		if fa != nil {
			fa.AddFollowerQuiet(mob, ch)
		}
	}
}

func spellActToRoom(world, actor interface{}, format string, hideInvisible bool) {
	type roomActer interface {
		SpellActToRoom(actor, victim interface{}, format string, hideInvisible bool) bool
	}
	if acter, ok := world.(roomActer); ok {
		acter.SpellActToRoom(actor, nil, format, hideInvisible)
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
	type itemChecker interface {
		GetVNum() int
		GetName() string
	}
	type roomItems interface {
		GetItemsInRoomI(roomVNum int) []interface{}
	}

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
	// C spell_mindsight: ... || (!IS_NPC(victim) && GET_LEVEL(victim) >= LEVEL_IMMORT
	// && GET_LEVEL(ch) <= GET_LEVEL(victim)) (src/spells.c:922-923).
	if (victLevel > casterLevel+4 && dprng.Number(0, 4) == 0) ||
		(!victIsNPC && victLevel >= combat.LVL_IMMORT && casterLevel <= victLevel) {
		sendToCaster(ch, "With a searing pain, your psionic energy recoils!\r\n")
		return
	}
	// C spell_mindsight refuses AFF_NOTRACK before reading the target room
	// (src/spells.c:930-933; AFF_NOTRACK is C affect bit 15).
	type affectChecker interface{ IsAffected(int) bool }
	if affected, ok := cvict.(affectChecker); ok && affected.IsAffected(15) {
		sendToCaster(ch, "Cannot find the target of your will!\r\n")
		return
	}

	victRoom := 0
	if rg, ok := cvict.(roomGet); ok {
		victRoom = rg.GetRoomVNum()
	}
	if victRoom <= 0 {
		return
	}

	// C spell_mindsight uses char_from_room/char_to_room, do_look, then
	// notifies other PCs in the caster's current room (src/spells.c:938-955).
	type mindsightWorld interface {
		PlayerTransferForSpell(actor interface{}, toRoomVNum int) error
		LookAtRoomForSpell(actor interface{})
		MindsightNotifyWatchers(actor interface{})
	}
	tw, ok := world.(mindsightWorld)
	if !ok {
		sendToCaster(ch, "The magic fails.\r\n")
		return
	}

	if err := tw.PlayerTransferForSpell(ch, victRoom); err != nil {
		slog.Error("PlayerTransfer failed", "error", err)
	}
	tw.LookAtRoomForSpell(ch)
	tw.MindsightNotifyWatchers(ch)
	if currentRoom, ok := ch.(roomGet); ok && currentRoom.GetRoomVNum() == victRoom {
		if err := tw.PlayerTransferForSpell(ch, chRoom); err != nil {
			slog.Error("PlayerTransfer failed", "error", err)
		}
	}
}

// castGate creates a red portal in one of the 8 legal rooms.
// If a portal already exists, the caster dies from cosmic energy.
// If the room is not legal, nothing happens.
func castGate(level int, ch, world interface{}) {
	_ = level
	if ch == nil {
		return
	}
	type gateCaster interface {
		CastGateSpell(caster interface{})
	}
	if gateWorld, ok := world.(gateCaster); ok {
		gateWorld.CastGateSpell(ch)
	}
}

// castLocateObject finds objects matching the arg name in the world.
func castLocateObject(level int, ch, ovict, world interface{}) {
	if ch == nil {
		return
	}

	if ovict == nil {
		return
	}
	// C reaches this manual spell only after get_obj_vis found a target;
	// spell_locate_object then fname()s that object's keywords.
	name := ""
	if on, ok := ovict.(interface{ GetKeywords() string }); ok {
		name = on.GetKeywords()
	} else if on, ok := ovict.(interface{ GetName() string }); ok {
		name = on.GetName()
	}
	if name == "" {
		return
	}

	type locateResults interface {
		LocateObjectResults(name string, viewer interface{}, limit int, unseenOwnerText string) ([]string, bool)
	}
	if w, ok := world.(locateResults); ok {
		// say_spell leaves the syllable buffer in C's global buf. If the
		// matched object's owner is invisible, spell_locate_object's skipped
		// sprintf sends that buffer unchanged (src/spells.c:377-391).
		unseenOwnerText := ObfuscateSpellName(GetSpellName(SpellLocateObject))
		messages, blocked := w.LocateObjectResults(name, ch, level>>1, unseenOwnerText)
		if blocked {
			sendToCaster(ch, "Your magick suddenly dissolves into a wash of brilliant colours, and then, nothing.\r\n")
			return
		}
		if len(messages) == 0 {
			sendToCaster(ch, "You sense nothing.\n\r")
			return
		}
		// C writes every matching line into one output buffer. In particular,
		// the hidden-owner CAN_SEE arm reuses say_spell's buffer without a line
		// ending, so separate Go text events would each acquire a transport
		// line ending and change the bytes between results.
		sendToCaster(ch, strings.Join(messages, ""))
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
		type roomActer interface {
			SpellActToRoom(actor, victim interface{}, format string, hideInvisible bool) bool
		}
		if acter, ok := world.(roomActer); ok {
			// C src/spells.c:1153 uses act(..., TRUE, ..., TO_ROOM), so retain
			// the canonical act substitutions and invisibility filtering.
			acter.SpellActToRoom(ch, nil, "$n divides $mself in two!", true)
		}
	}
}

// src/magic.c:1146-1148,1166-1168: name the caster, after its warning,
// before raw_kill; BRF/31/file TRUE. Live World supplies the existing consumer.
func logProtectionKill(world, ch interface{}, protection string) {
	logger, ok := world.(interface{ MudLog(string, int, int, bool) })
	caster, named := ch.(interface{ GetName() string })
	if ok && named {
		logger.MudLog(fmt.Sprintf("%s killed by Protection from %s.", caster.GetName(), protection), 1, lvlImmort, true)
	}
}
