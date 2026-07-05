package spells

import "testing"

// spellInfoGolden is transcribed verbatim from src/spell_parser.c mag_assign_spells().
// C spello() order: maxMana, minMana, manaChange, minPosition, targets, violent, routines.
// Go setupSpellInfo() order: minPosition, manaMin, manaMax, manaChange, routines, violent, targets.
var spellInfoGolden = []struct {
	spellNum    int
	name        string
	manaMax     int
	manaMin     int
	manaChange  int
	minPosition Position
	targets     TargetFlags
	violent     bool
	routines    MagRoutine
}{
	{ SpellArmor, "SpellArmor", 30, 15, 3, PosFighting, TarCharRoom, false, RoutineAffects },
	{ SpellTeleport, "SpellTeleport", 60, 50, 3, PosFighting, TarCharRoom | TarFightVict, false, RoutineManual },
	{ SpellBless, "SpellBless", 36, 10, 2, PosStanding, TarCharRoom | TarObjInv, false, RoutineAffects | RoutineAlterObjs },
	{ SpellBlindness, "SpellBlindness", 35, 25, 1, PosFighting, TarCharRoom | TarNotSelf | TarFightVict, true, RoutineAffects },
	{ SpellBurningHands, "SpellBurningHands", 45, 20, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellCallLightning, "SpellCallLightning", 68, 52, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellCharm, "SpellCharm", 75, 50, 5, PosFighting, TarCharRoom | TarNotSelf, true, RoutineManual },
	{ SpellChillTouch, "SpellChillTouch", 35, 15, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage | RoutineAffects },
	{ SpellClone, "SpellClone", 80, 65, 5, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineSummons },
	{ SpellColorSpray, "SpellColorSpray", 58, 38, 4, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellControlWeather, "SpellControlWeather", 75, 25, 5, PosStanding, TarIgnore, false, RoutineManual },
	{ SpellCreateFood, "SpellCreateFood", 35, 10, 5, PosStanding, TarIgnore, false, RoutineCreations },
	{ SpellCreateWater, "SpellCreateWater", 35, 10, 5, PosStanding, TarObjInv | TarObjEquip, false, RoutineManual },
	{ SpellCureBlind, "SpellCureBlind", 35, 5, 5, PosStanding, TarCharRoom, false, RoutineUnaffects },
	{ SpellCureCritic, "SpellCureCritic", 70, 40, 5, PosFighting, TarCharRoom, false, RoutinePoints },
	{ SpellCureLight, "SpellCureLight", 30, 10, 2, PosFighting, TarCharRoom, false, RoutinePoints },
	{ SpellCurse, "SpellCurse", 80, 50, 2, PosFighting, TarCharRoom | TarObjInv | TarFightVict, true, RoutineAffects | RoutineAlterObjs },
	{ SpellDetectAlign, "SpellDetectAlign", 20, 10, 2, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellDetectInvis, "SpellDetectInvis", 20, 10, 2, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellDetectMagic, "SpellDetectMagic", 20, 10, 2, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellDetectPoison, "SpellDetectPoison", 20, 10, 2, PosStanding, TarCharRoom | TarObjInv | TarObjRoom, false, RoutineManual },
	{ SpellDispelEvil, "SpellDispelEvil", 95, 65, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellDispelGood, "SpellDispelGood", 95, 65, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellEarthquake, "SpellEarthquake", 70, 50, 5, PosFighting, TarIgnore, true, RoutineAreas },
	{ SpellDreamTravel, "SpellDreamTravel", 60, 45, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellEnchantWeapon, "SpellEnchantWeapon", 200, 150, 10, PosStanding, TarObjInv | TarObjEquip, false, RoutineManual },
	{ SpellEnergyDrain, "SpellEnergyDrain", 60, 45, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellHolyShield, "SpellHolyShield", 90, 65, 5, PosStanding, TarIgnore, false, RoutineGroups },
	{ SpellFireball, "SpellFireball", 70, 50, 2, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellGroupHeal, "SpellGroupHeal", 210, 150, 5, PosFighting, TarIgnore, false, RoutineGroups },
	{ SpellChameleon, "SpellChameleon", 50, 30, 5, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellGroupRecall, "SpellGroupRecall", 155, 125, 5, PosStanding, TarIgnore, false, RoutineGroups },
	{ SpellHarm, "SpellHarm", 105, 75, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellHaste, "SpellHaste", 140, 140, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellHeal, "SpellHeal", 90, 80, 3, PosFighting, TarCharRoom, false, RoutinePoints | RoutineAffects | RoutineUnaffects },
	{ SpellHellfire, "SpellHellfire", 200, 150, 10, PosFighting, TarIgnore, true, RoutineManual | RoutineAreas },
	{ SpellInfravision, "SpellInfravision", 25, 25, 1, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellGroupInvis, "SpellGroupInvis", 135, 135, 1, PosStanding, TarIgnore, false, RoutineGroups },
	{ SpellInvisible, "SpellInvisible", 45, 45, 1, PosStanding, TarCharRoom | TarObjInv | TarObjRoom, false, RoutineAffects | RoutineAlterObjs },
	{ SpellLevitate, "SpellLevitate", 90, 70, 5, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellLightningBolt, "SpellLightningBolt", 54, 34, 4, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellLocateObject, "SpellLocateObject", 25, 20, 1, PosStanding, TarObjWorld, false, RoutineManual },
	{ SpellMagicMissile, "SpellMagicMissile", 30, 15, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellMindPoke, "SpellMindPoke", 30, 15, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellMindBlast, "SpellMindBlast", 70, 40, 2, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellPoison, "SpellPoison", 50, 40, 2, PosFighting, TarCharRoom | TarNotSelf | TarObjInv | TarFightVict, true, RoutineAffects | RoutineAlterObjs },
	{ SpellFlamestrike, "SpellFlamestrike", 105, 100, 1, PosStanding, TarCharRoom | TarNotSelf, true, RoutineAffects },
	{ SpellProtFromEvil, "SpellProtFromEvil", 50, 50, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellProtFromGood, "SpellProtFromGood", 50, 50, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellRemoveCurse, "SpellRemoveCurse", 45, 45, 1, PosStanding, TarCharRoom | TarObjInv, false, RoutineUnaffects | RoutineAlterObjs },
	{ SpellSanctuary, "SpellSanctuary", 110, 85, 2, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellShockingGrasp, "SpellShockingGrasp", 55, 35, 5, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellSleep, "SpellSleep", 40, 35, 1, PosStanding, TarCharRoom | TarNotSelf, true, RoutineAffects },
	{ SpellSobriety, "SpellSobriety", 35, 20, 5, PosStanding, TarCharRoom, false, RoutineManual },
	{ SpellStrength, "SpellStrength", 35, 30, 1, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellSummon, "SpellSummon", 90, 70, 1, PosStanding, TarCharWorld | TarNotSelf, false, RoutineManual },
	{ SpellCoC, "SpellCoC", 90, 70, 1, PosStanding, TarIgnore, false, RoutineManual },
	{ SpellWordOfRecall, "SpellWordOfRecall", 50, 50, 1, PosFighting, TarCharRoom, false, RoutineManual },
	{ SpellRemovePoison, "SpellRemovePoison", 40, 30, 1, PosStanding, TarCharRoom | TarObjInv | TarObjRoom, false, RoutineUnaffects | RoutineAlterObjs },
	{ SpellSenseLife, "SpellSenseLife", 30, 20, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellSlow, "SpellSlow", 80, 50, 2, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellMassHeal, "SpellMassHeal", 130, 100, 1, PosFighting, TarCharRoom, false, RoutinePoints | RoutineAffects | RoutineUnaffects },
	{ SpellWaterwalk, "SpellWaterwalk", 80, 55, 1, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellFly, "SpellFly", 100, 80, 5, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellLycanthropy, "SpellLycanthropy", 1, 1, 1, PosStanding, TarCharRoom, false, RoutineManual },
	{ SpellVampirism, "SpellVampirism", 1, 1, 1, PosStanding, TarCharRoom, false, RoutineManual },
	{ SpellEnchantArmor, "SpellEnchantArmor", 150, 130, 10, PosStanding, TarObjInv | TarObjEquip, false, RoutineManual },
	{ SpellIdentify, "SpellIdentify", 125, 100, 10, PosStanding, TarCharRoom | TarObjInv | TarObjRoom, false, RoutineManual },
	{ SpellMetalskin, "SpellMetalskin", 75, 60, 1, PosFighting, TarCharRoom, false, RoutineAffects },
	{ SpellInvulnerability, "SpellInvulnerability", 85, 85, 1, PosFighting, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellVitality, "SpellVitality", 110, 100, 1, PosFighting, TarCharRoom, false, RoutinePoints },
	{ SpellInvigorate, "SpellInvigorate", 110, 95, 1, PosFighting, TarCharRoom, false, RoutinePoints },
	{ SpellPsyshield, "SpellPsyshield", 30, 20, 1, PosFighting, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellAdrenaline, "SpellAdrenaline", 35, 30, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellMindAttack, "SpellMindAttack", 55, 25, 1, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellLessPercept, "SpellLessPercept", 40, 30, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellGreatPercept, "SpellGreatPercept", 65, 45, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellChangeDensity, "SpellChangeDensity", 70, 55, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellAcidBlast, "SpellAcidBlast", 35, 20, 1, PosFighting, TarIgnore, true, RoutineAreas },
	{ SpellDominate, "SpellDominate", 75, 50, 5, PosFighting, TarCharRoom | TarNotSelf, true, RoutineManual },
	{ SpellMassDominate, "SpellMassDominate", 220, 150, 10, PosStanding, TarIgnore, true, RoutineAreas },
	{ SpellCellAdjustment, "SpellCellAdjustment", 85, 75, 1, PosFighting, TarCharRoom | TarSelfOnly, false, RoutinePoints },
	{ SpellZen, "SpellZen", 70, 60, 4, PosFighting, TarCharRoom | TarSelfOnly, false, RoutineManual },
	{ SpellMirrorImage, "SpellMirrorImage", 150, 130, 5, PosStanding, TarIgnore, false, RoutineManual },
	{ SpellSoulLeech, "SpellSoulLeech", 60, 55, 1, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellMindsight, "SpellMindsight", 70, 60, 1, PosStanding, TarCharWorld, false, RoutineManual },
	{ SpellMindBar, "SpellMindBar", 115, 100, 1, PosStanding, TarCharRoom, true, RoutineAffects },
	{ SpellTransparency, "SpellTransparency", 35, 25, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellKnowAlign, "SpellKnowAlign", 20, 20, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutineAffects },
	{ SpellGate, "SpellGate", 95, 95, 1, PosStanding, TarIgnore, false, RoutineManual },
	{ SpellIntellect, "SpellIntellect", 60, 60, 1, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellLayHands, "SpellLayHands", 90, 90, 1, PosStanding, TarCharRoom | TarSelfOnly, false, RoutinePoints },
	{ SpellMentalLapse, "SpellMentalLapse", 100, 90, 1, PosStanding, TarCharWorld, false, RoutineManual },
	{ SpellSmokescreen, "SpellSmokescreen", 100, 100, 1, PosFighting, TarIgnore, true, RoutineMasses },
	{ SpellDisrupt, "SpellDisrupt", 175, 165, 1, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellDivineInt, "SpellDivineInt", 290, 290, 1, PosStanding, TarIgnore, false, RoutineManual },
	{ SpellDisintegrate, "SpellDisintegrate", 120, 120, 1, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellAnimateDead, "SpellAnimateDead", 120, 100, 10, PosStanding, TarObjRoom, false, RoutineSummons },
	{ SpellCalliope, "SpellCalliope", 100, 50, 10, PosFighting, TarCharRoom | TarFightVict, true, RoutineManual },
	{ SpellMeteorSwarm, "SpellMeteorSwarm", 180, 170, 5, PosStanding, TarIgnore, true, RoutineManual },
	{ SpellPsiblast, "SpellPsiblast", 180, 150, 10, PosFighting, TarCharRoom | TarFightVict, true, RoutineDamage },
	{ SpellWaterBreathe, "SpellWaterBreathe", 92, 58, 6, PosStanding, TarCharRoom, false, RoutineAffects },
	{ SpellConjureElemental, "SpellConjureElemental", 165, 145, 1, PosStanding, TarIgnore, false, RoutineManual },
	{ SpellFireBreath, "SpellFireBreath", 70, 50, 5, PosFighting, TarIgnore, true, RoutineAreas },
	{ SpellFrostBreath, "SpellFrostBreath", 70, 50, 5, PosFighting, TarIgnore, true, RoutineAreas },
	{ SpellGasBreath, "SpellGasBreath", 70, 50, 5, PosFighting, TarIgnore, true, RoutineAreas },
	{ SpellAcidBreath, "SpellAcidBreath", 70, 50, 5, PosFighting, TarIgnore, true, RoutineAreas },
	{ SpellLightningBreath, "SpellLightningBreath", 70, 50, 5, PosFighting, TarIgnore, true, RoutineAreas },
}

func TestSpellInfo_GoldenAgainstCSource(t *testing.T) {
	for _, want := range spellInfoGolden {
		got := GetSpellInfo(want.spellNum)
		if got == nil {
			t.Errorf("GetSpellInfo(%s) = nil, want registered spell", want.name)
			continue
		}
		if got.ManaMax != want.manaMax {
			t.Errorf("%s ManaMax = %d, want %d", want.name, got.ManaMax, want.manaMax)
		}
		if got.ManaMin != want.manaMin {
			t.Errorf("%s ManaMin = %d, want %d", want.name, got.ManaMin, want.manaMin)
		}
		if got.ManaChange != want.manaChange {
			t.Errorf("%s ManaChange = %d, want %d", want.name, got.ManaChange, want.manaChange)
		}
		if got.MinPosition != want.minPosition {
			t.Errorf("%s MinPosition = %v, want %v", want.name, got.MinPosition, want.minPosition)
		}
		if got.Routines.Violent != want.violent {
			t.Errorf("%s Violent = %v, want %v", want.name, got.Routines.Violent, want.violent)
		}
		if got.Routines.Targets != want.targets {
			t.Errorf("%s Targets = %v, want %v", want.name, got.Routines.Targets, want.targets)
		}
		if got.Routines.Routines != want.routines {
			t.Errorf("%s Routines = %v, want %v", want.name, got.Routines.Routines, want.routines)
		}
	}
}
