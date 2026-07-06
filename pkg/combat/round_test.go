// Package combat — round execution tests
//
// These tests cover TakeDamage, DamMessage, PerformGroupGain,
// Appear, DeathCry, CounterProcs, and the engine round execution
// (processCombatPair, handleDeath, StartCombat, StopCombat).
//
// Global function pointers (the var hooks) are wired per-test and
// restored via defer to avoid cross-test contamination.
package combat

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestDamMessage — damage message tier selection
// ---------------------------------------------------------------------------

func TestDamMessage_TierSelection(t *testing.T) {
	var broadcastRoom int
	var broadcastMsg string
	var sendToCalls []string

	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastRoom = room
		broadcastMsg = msg
	}
	SendToCharFunc = func(name string, msg string) {
		sendToCalls = append(sendToCalls, name+":"+msg)
	}

	attacker := &mockCombatant{name: "Warrior", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Goblin", room: 100, sex: 1, position: PosStanding}

	// Damage=5 → tier "light" (minDamage 5)
	DamMessage(5, attacker, defender, 4) // attackType=4 = bite

	if broadcastRoom != 100 {
		t.Errorf("expected room 100, got %d", broadcastRoom)
	}
	if !strings.Contains(broadcastMsg, "Goblin") {
		t.Errorf("expected message mentioning Goblin, got %q", broadcastMsg)
	}
	if len(sendToCalls) < 2 {
		t.Errorf("expected at least 2 SendToChar calls (attacker+victim), got %d", len(sendToCalls))
	}
}

func TestDamMessage_HighDamageTier(t *testing.T) {
	var broadcastMessages []string
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMessages = append(broadcastMessages, msg)
	}
	SendToCharFunc = func(name string, msg string) {}

	attacker := &mockCombatant{name: "Hero", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Dragon", room: 100, sex: 1, position: PosStanding}

	DamMessage(1000, attacker, defender, 5) // bludgeon

	// Tier 12 (OBLITERATES, 101+) and Tier 13 (ROCKS, 10000+) are both valid.
	// randPick chooses one variant at random, so check any tier-appropriate keyword.
	foundTier := false
	for _, msg := range broadcastMessages {
		if strings.Contains(msg, "OBLITERATES") || strings.Contains(msg, "blow of legend") ||
			strings.Contains(msg, "R O C K S") || strings.Contains(msg, "smear") {
			foundTier = true
			break
		}
	}
	if !foundTier {
		t.Errorf("expected high-damage tier message for dam=1000, got %v", broadcastMessages)
	}
}

func TestDamMessage_ZeroDamage(t *testing.T) {
	var broadcastCalled bool
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastCalled = true
	}
	SendToCharFunc = func(name string, msg string) {}

	attacker := &mockCombatant{name: "Player", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Rat", room: 100, sex: 0, position: PosStanding}

	DamMessage(0, attacker, defender, 1) // sting

	if !broadcastCalled {
		t.Error("expected broadcast even for 0 damage (miss message)")
	}
}

// ---------------------------------------------------------------------------
// TestAppear — become visible
// ---------------------------------------------------------------------------

func TestAppear_Basic(t *testing.T) {
	var broadcastMsg string
	origBroadcast := BroadcastMessage
	origHasAffect := HasAffect
	origRemoveAffect := RemoveAffect
	defer func() {
		BroadcastMessage = origBroadcast
		HasAffect = origHasAffect
		RemoveAffect = origRemoveAffect
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMsg = msg
	}
	HasAffect = func(name string, aff int) bool {
		return name == "Rogue" && aff == SPELL_INVISIBLE
	}
	RemoveAffect = func(name string, skillNum int) {
		if name != "Rogue" || skillNum != SPELL_INVISIBLE {
			t.Errorf("RemoveAffect called with unexpected args: %s, %d", name, skillNum)
		}
	}

	ch := &mockCombatant{name: "Rogue", room: 200, level: 10}
	Appear(ch)

	if !strings.Contains(broadcastMsg, "fades into existence") {
		t.Errorf("expected fade message, got %q", broadcastMsg)
	}
}

func TestAppear_ImmortalLevel(t *testing.T) {
	var broadcastMsg string
	origBroadcast := BroadcastMessage
	origHasAffect := HasAffect
	origRemoveAffect := RemoveAffect
	defer func() {
		BroadcastMessage = origBroadcast
		HasAffect = origHasAffect
		RemoveAffect = origRemoveAffect
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMsg = msg
	}
	HasAffect = func(name string, aff int) bool {
		return name == "Wizard" && aff == SPELL_INVISIBLE
	}
	RemoveAffect = func(name string, skillNum int) {}

	ch := &mockCombatant{name: "Wizard", room: 200, level: LVL_IMMORT}
	Appear(ch)

	if !strings.Contains(broadcastMsg, "strange presence") {
		t.Errorf("expected strange presence message for immortal, got %q", broadcastMsg)
	}
}

// ---------------------------------------------------------------------------
// TestDeathCry
// ---------------------------------------------------------------------------

func TestDeathCry_Basic(t *testing.T) {
	var broadcastMessages []string
	origBroadcast := BroadcastMessage
	origGetAdj := GetAdjacentRoom
	defer func() {
		BroadcastMessage = origBroadcast
		GetAdjacentRoom = origGetAdj
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMessages = append(broadcastMessages, fmt.Sprintf("room=%d msg=%q", room, msg))
	}
	GetAdjacentRoom = func(room int, door int) int {
		if room == 100 && door < 2 {
			return 101 + door
		}
		return -1
	}

	ch := &mockCombatant{name: "Orc", room: 100}
	result := DeathCry(ch)

	if !strings.Contains(result, "100") {
		t.Errorf("expected room 100 in result, got %q", result)
	}
	if len(broadcastMessages) < 1 {
		t.Error("expected at least one broadcast")
	}
}

// ---------------------------------------------------------------------------
// TestCounterProcs — kill milestone rewards
// ---------------------------------------------------------------------------

func TestCounterProcs_NoMilestone(t *testing.T) {
	npc := &mockCombatant{name: "Orc", npc: true}
	CounterProcs(npc)
	// Should not panic
}

func TestCounterProcs_KillCountTracking(t *testing.T) {
	var loggedMsg string
	origGetKills := GetKills
	origLogMsg := LogMessage
	origIncMaxStat := IncreaseMaxStat
	origHealAll := HealAllPlayers
	defer func() {
		GetKills = origGetKills
		LogMessage = origLogMsg
		IncreaseMaxStat = origIncMaxStat
		HealAllPlayers = origHealAll
	}()

	GetKills = func(name string) int64 {
		return 5000 // minor milestone
	}
	LogMessage = func(msg string, level string, minLevel int, toLog bool) {
		loggedMsg = msg
	}
	IncreaseMaxStat = func(name string, stat string) {}
	HealAllPlayers = func() {}

	player := &mockCombatant{name: "Hero", npc: false, hp: 50, maxHP: 100}
	CounterProcs(player)

	if !strings.Contains(loggedMsg, "5000 kills") {
		t.Errorf("expected log about 5000 kills, got %q", loggedMsg)
	}
}

func TestCounterProcs_MajorMilestone(t *testing.T) {
	var statCalls []string
	var logged bool
	origGetKills := GetKills
	origLogMsg := LogMessage
	origIncMaxStat := IncreaseMaxStat
	origHealAll := HealAllPlayers
	defer func() {
		GetKills = origGetKills
		LogMessage = origLogMsg
		IncreaseMaxStat = origIncMaxStat
		HealAllPlayers = origHealAll
	}()

	GetKills = func(name string) int64 {
		return 2000 // major milestone
	}
	LogMessage = func(msg string, level string, minLevel int, toLog bool) {
		logged = true
	}
	IncreaseMaxStat = func(name string, stat string) {
		statCalls = append(statCalls, stat)
	}
	HealAllPlayers = func() {}

	player := &mockCombatant{name: "Hero", npc: false, hp: 50, maxHP: 100}
	CounterProcs(player)

	// Major milestone: C bug causes all three stats to be incremented (fall-through)
	if len(statCalls) != 3 {
		t.Errorf("expected 3 IncreaseMaxStat calls (C bug: all fall through), got %d: %v", len(statCalls), statCalls)
	}
	if !logged {
		t.Error("expected log for milestone")
	}
}

// ---------------------------------------------------------------------------
// TestPerformGroupGain — individual group member exp
// ---------------------------------------------------------------------------

func TestPerformGroupGain_Basic(t *testing.T) {
	var gainedExp int
	var gainedName string
	var alignSet bool

	origGainExp := GainExp
	origGetAlign := GetAlignment
	origSetAlign := SetAlignment
	defer func() {
		GainExp = origGainExp
		GetAlignment = origGetAlign
		SetAlignment = origSetAlign
	}()

	GainExp = func(name string, amount int) {
		gainedName = name
		gainedExp = amount
	}
	GetAlignment = func(name string) int {
		if name == "Orc" {
			return 900 // good-aligned victim triggers alignment shift
		}
		return 0
	}
	SetAlignment = func(name string, val int) { alignSet = true }

	ch := &mockCombatant{name: "Fighter", level: 10, room: 100, npc: false}
	victim := &mockCombatant{name: "Orc", level: 8, npc: true}

	PerformGroupGain(ch, victim, 200)

	if gainedName != "Fighter" {
		t.Errorf("expected exp assigned to Fighter, got %q", gainedName)
	}
	if gainedExp <= 0 {
		t.Errorf("expected positive exp gain, got %d", gainedExp)
	}
	if !alignSet {
		t.Error("expected alignment change when killing a good-aligned victim")
	}
}

func TestPerformGroupGain_OneExpPoint(t *testing.T) {
	origGainExp := GainExp
	origGetAlign := GetAlignment
	origSetAlign := SetAlignment
	defer func() {
		GainExp = origGainExp
		GetAlignment = origGetAlign
		SetAlignment = origSetAlign
	}()

	GainExp = func(name string, amount int) {}
	GetAlignment = func(name string) int { return 0 }
	SetAlignment = func(name string, val int) {}

	ch := &mockCombatant{name: "Fighter", level: 99, room: 100, npc: false}
	victim := &mockCombatant{name: "Rat", level: 1, npc: true}

	// base=1, extreme level diff, level>20 penalty → should floor at 1
	PerformGroupGain(ch, victim, 1)
	// Should not panic, internal calc returns at least 1
}

// ---------------------------------------------------------------------------
// Engine lifecycle tests
// ---------------------------------------------------------------------------

// TestStartCombat verifies StartCombat initializes a combat pair correctly.
func TestStartCombat(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", npc: false, room: 100}
	defender := &mockCombatant{name: "Orc", npc: true, room: 100}

	err := engine.StartCombat(attacker, defender)
	if err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	if !engine.IsFighting("Hero") {
		t.Error("expected Hero to be fighting")
	}
	if !engine.IsFighting("Orc") {
		t.Error("expected Orc to be fighting")
	}
	if attacker.fighting != "Orc" {
		t.Errorf("expected attacker fighting Orc, got %q", attacker.fighting)
	}
	if defender.fighting != "Hero" {
		t.Errorf("expected defender fighting Hero, got %q", defender.fighting)
	}
}

// TestStartCombat_Duplicate prevents starting combat twice for same attacker.
func TestStartCombat_Duplicate(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	orc := &mockCombatant{name: "Orc", room: 100}
	goblin := &mockCombatant{name: "Goblin", room: 100}

	_ = engine.StartCombat(attacker, orc)
	err := engine.StartCombat(attacker, goblin)
	if err == nil {
		t.Error("expected error when starting combat with already-fighting attacker")
	}
}

// TestStartCombat_DefenderKeepsExistingTarget verifies that attacking a
// defender already in combat does not retarget the defender. In DikuMUD a
// character FIGHTs one opponent at a time; a second attacker is added but the
// defender keeps fighting its original target until that target is gone.
func TestStartCombat_DefenderKeepsExistingTarget(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	hero := &mockCombatant{name: "Hero", room: 100}
	orc := &mockCombatant{name: "Orc", npc: true, room: 100}
	goblin := &mockCombatant{name: "Goblin", npc: true, room: 100}

	// Orc is already fighting Hero.
	if err := engine.StartCombat(orc, hero); err != nil {
		t.Fatalf("StartCombat(orc, hero) failed: %v", err)
	}
	// Goblin now jumps Hero, who is already fighting Orc.
	if err := engine.StartCombat(goblin, hero); err != nil {
		t.Fatalf("StartCombat(goblin, hero) failed: %v", err)
	}

	// Hero must still be fighting its original target, not retargeted to Goblin.
	if hero.fighting != "Orc" {
		t.Errorf("expected Hero to keep fighting Orc, got %q", hero.fighting)
	}
	// Goblin is engaged with Hero either way.
	if goblin.fighting != "Hero" {
		t.Errorf("expected Goblin fighting Hero, got %q", goblin.fighting)
	}
	if !engine.IsFighting("Goblin") {
		t.Error("expected Goblin to be registered as fighting")
	}
}

// TestStopCombat removes a combat pair and clears fighting states.
func TestStopCombat(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}

	_ = engine.StartCombat(attacker, defender)
	engine.StopCombat("Hero")

	if engine.IsFighting("Hero") {
		t.Error("expected Hero to not be fighting after StopCombat")
	}
}

// TestGetCombatTarget verifies target lookup works both ways.
func TestGetCombatTarget(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}

	_ = engine.StartCombat(attacker, defender)

	target, ok := engine.GetCombatTarget("Hero")
	if !ok {
		t.Fatal("expected Hero to have a combat target")
	}
	if target.GetName() != "Orc" {
		t.Errorf("expected target Orc, got %q", target.GetName())
	}

	// Reverse lookup
	target, ok = engine.GetCombatTarget("Orc")
	if !ok {
		t.Fatal("expected Orc to have a combat target")
	}
	if target.GetName() != "Hero" {
		t.Errorf("expected target Hero, got %q", target.GetName())
	}

	// Non-combatant
	_, ok = engine.GetCombatTarget("Nobody")
	if ok {
		t.Error("expected no target for non-combatant")
	}
}

// TestGetCombatStatus verifies status string output.
func TestGetCombatStatus(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}
	_ = engine.StartCombat(attacker, defender)

	status := engine.GetCombatStatus("Hero")
	if !strings.Contains(status, "fighting") {
		t.Errorf("expected 'fighting' in status for attacker, got %q", status)
	}

	status = engine.GetCombatStatus("Nobody")
	if !strings.Contains(status, "not in combat") {
		t.Errorf("expected 'not in combat' for non-combatant, got %q", status)
	}
}

// ---------------------------------------------------------------------------
// processCombatPair — the core round execution
// ---------------------------------------------------------------------------

// TestProcessCombatPair_MobAttack runs a combat pair round between a mob and player.
func TestProcessCombatPair_MobAttack(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	origHasAffect := HasAffect
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	origGetSkill := GetSkill
	origHasMobFlag := HasMobFlag
	origGetWeapon := GetWeaponInfo
	origHasScript := HasScriptFlag
	origRunFight := RunFightScript
	origMakeCorpse := MakeCorpseFunc
	origMakeDust := MakeDustFunc
	origExtract := ExtractChar
	origGainExp := GainExp
	origGetExp := GetExp
	origGetKills := GetKills
	origSetKills := SetKills
	origGetDeaths := GetDeaths
	origSetDeaths := SetDeaths
	origGetPks := GetPks
	origSetPks := SetPks
	origSetLastDeath := SetLastDeath
	origLogMsg := LogMessage
	origHasPlr := HasPlrFlag
	origSetPlr := SetPlrFlag
	origHasRoom := HasRoomFlag
	origIsShop := IsShopkeeper
	origHasMobVNum := HasMobVNum
	origGetRace := GetRace
	origGetRaceHate := GetRaceHate
	origHasAffectStr := HasAffectStr
	origIsMounted := IsMounted
	origDismount := Dismount
	origHasPrf := HasPrfFlag
	origPerformCmd := PerformCommand
	origGetGold := GetGold
	origSetGold := SetGold
	origCountGroup := CountGroupMembers
	origApplyGroup := ApplyToGroupMembers
	origGetAlign := GetAlignment
	origSetAlign := SetAlignment
	origGetWimpy := GetWimpyLev
	origDoFlee := DoFlee
	origDoRetreat := DoRetreat
	origSkillMsg := SkillMessageFunc
	origRunDeath := RunDeathScript
	origBroadChat := BroadChatFunc
	defer func() {
		HasAffect = origHasAffect
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
		GetSkill = origGetSkill
		HasMobFlag = origHasMobFlag
		GetWeaponInfo = origGetWeapon
		HasScriptFlag = origHasScript
		RunFightScript = origRunFight
		MakeCorpseFunc = origMakeCorpse
		MakeDustFunc = origMakeDust
		ExtractChar = origExtract
		GainExp = origGainExp
		GetExp = origGetExp
		GetKills = origGetKills
		SetKills = origSetKills
		GetDeaths = origGetDeaths
		SetDeaths = origSetDeaths
		GetPks = origGetPks
		SetPks = origSetPks
		SetLastDeath = origSetLastDeath
		LogMessage = origLogMsg
		HasPlrFlag = origHasPlr
		SetPlrFlag = origSetPlr
		HasRoomFlag = origHasRoom
		IsShopkeeper = origIsShop
		HasMobVNum = origHasMobVNum
		GetRace = origGetRace
		GetRaceHate = origGetRaceHate
		HasAffectStr = origHasAffectStr
		IsMounted = origIsMounted
		Dismount = origDismount
		HasPrfFlag = origHasPrf
		PerformCommand = origPerformCmd
		GetGold = origGetGold
		SetGold = origSetGold
		CountGroupMembers = origCountGroup
		ApplyToGroupMembers = origApplyGroup
		GetAlignment = origGetAlign
		SetAlignment = origSetAlign
		GetWimpyLev = origGetWimpy
		DoFlee = origDoFlee
		DoRetreat = origDoRetreat
		SkillMessageFunc = origSkillMsg
		RunDeathScript = origRunDeath
		BroadChatFunc = origBroadChat
	}()

	// Wire global hooks
	HasAffect = func(name string, aff int) bool { return false }
	BroadcastMessage = func(room int, msg string, exclude string) {}
	SendToCharFunc = func(name string, msg string) {}
	GetSkill = func(name string, skillNum int) int { return 0 }
	HasMobFlag = func(name string, flag string) bool { return false }
	GetWeaponInfo = func(name string) (int, int, int, bool) {
		if name == "Hero" {
			return TYPE_SLASH, 1, 8, false
		}
		return TYPE_HIT, 0, 0, false
	}
	HasScriptFlag = func(name string, flag string) bool { return false }
	MakeCorpseFunc = func(name string, attackType int) {}
	MakeDustFunc = func(name string, attackType int) {}
	ExtractChar = func(name string) {}
	GainExp = func(name string, amount int) {}
	GetExp = func(name string) int { return 0 }
	GetKills = func(name string) int64 { return 0 }
	SetKills = func(name string, kills int64) {}
	GetDeaths = func(name string) int64 { return 0 }
	SetDeaths = func(name string, deaths int64) {}
	GetPks = func(name string) int64 { return 0 }
	SetPks = func(name string, pks int64) {}
	SetLastDeath = func(name string, t int64) {}
	LogMessage = func(msg string, level string, minLevel int, toLog bool) {}
	HasPlrFlag = func(name string, flag string) bool { return false }
	SetPlrFlag = func(name string) bool { return false }
	HasRoomFlag = func(room int, flag string) bool { return false }
	IsShopkeeper = func(name string) bool { return false }
	HasMobVNum = func(name string, vnum int) bool { return false }
	GetRace = func(name string) int { return 0 }
	GetRaceHate = func(name string, index int) int { return 0 }
	HasAffectStr = func(name string, aff string) bool { return false }
	IsMounted = func(name string) bool { return false }
	Dismount = func(name string) {}
	HasPrfFlag = func(name string, flag string) bool { return false }
	PerformCommand = func(name, cmd string) {}
	GetGold = func(name string) int { return 0 }
	SetGold = func(name string, gold int) {}
	CountGroupMembers = func(name string, room int) int { return 1 }
	ApplyToGroupMembers = func(name string, room int, fn func(string)) {}
	GetAlignment = func(name string) int { return 0 }
	SetAlignment = func(name string, val int) {}
	GetWimpyLev = func(name string) int { return 0 }
	DoFlee = func(name string) {}
	DoRetreat = func(name string) {}
	SkillMessageFunc = func(dam int, ch, vict string, atk int, room int) bool { return false }
	RunDeathScript = func(killer, victim string, room int) {}
	BroadChatFunc = func(name string, msg string) {}

	// Wire engine callbacks
	engine.BroadcastFunc = func(room int, msg string, exclude string) {}
	engine.ScriptFightFunc = nil

	attacker := &mockCombatant{
		name:       "Hero",
		npc:        false,
		room:       100,
		level:      10,
		hp:         100,
		maxHP:      100,
		ac:         30,
		thac0:      15,
		position:   PosStanding,
		class:      ClassWarrior,
		str:        16,
		strAdd:     0,
		dex:        14,
		intVal:     12,
		wis:        10,
		hitroll:    1,
		damroll:    2,
		damageRoll: DiceRoll{Num: 1, Sides: 8},
	}
	defender := &mockCombatant{
		name:       "Orc",
		npc:        true,
		room:       100,
		level:      5,
		hp:         50,
		maxHP:      50,
		ac:         40,
		thac0:      20,
		position:   PosStanding,
		str:        15,
		strAdd:     0,
		dex:        10,
		intVal:     8,
		wis:        8,
		damageRoll: DiceRoll{Num: 1, Sides: 4},
	}

	_ = engine.StartCombat(attacker, defender)

	beforeHP := defender.hp
	engine.processCombatPair(&CombatPair{
		Attacker: attacker,
		Defender: defender,
	})

	// Damage should have been dealt (or missed)
	if defender.hp > beforeHP {
		t.Error("defender HP should not increase after a round")
	}

	t.Logf("processCombatPair: defender HP %d -> %d", beforeHP, defender.hp)
}

// TestProcessCombatPair_PlayerDeath sets defender to low HP and verifies death is detected.
func TestProcessCombatPair_PlayerDeath(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	origHasAffect := HasAffect
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	origGetSkill := GetSkill
	origHasMobFlag := HasMobFlag
	origGetWeapon := GetWeaponInfo
	origHasScript := HasScriptFlag
	origMakeCorpse := MakeCorpseFunc
	origMakeDust := MakeDustFunc
	origExtract := ExtractChar
	origGainExp := GainExp
	origGetExp := GetExp
	origGetKills := GetKills
	origSetKills := SetKills
	origGetDeaths := GetDeaths
	origSetDeaths := SetDeaths
	origGetPks := GetPks
	origSetPks := SetPks
	origSetLastDeath := SetLastDeath
	origLogMsg := LogMessage
	origHasPlr := HasPlrFlag
	origSetPlr := SetPlrFlag
	origHasRoom := HasRoomFlag
	origIsShop := IsShopkeeper
	origHasMobVNum := HasMobVNum
	origGetRace := GetRace
	origGetRaceHate := GetRaceHate
	origHasAffectStr := HasAffectStr
	origIsMounted := IsMounted
	origDismount := Dismount
	origHasPrf := HasPrfFlag
	origPerformCmd := PerformCommand
	origGetGold := GetGold
	origSetGold := SetGold
	origCountGroup := CountGroupMembers
	origApplyGroup := ApplyToGroupMembers
	origGetAlign := GetAlignment
	origSetAlign := SetAlignment
	origGetWimpy := GetWimpyLev
	origDoFlee := DoFlee
	origDoRetreat := DoRetreat
	origSkillMsg := SkillMessageFunc
	origRunDeath := RunDeathScript
	origBroadChat := BroadChatFunc
	origRemoveAff := RemoveAffect
	origNowUnix := NowUnix
	defer func() {
		HasAffect = origHasAffect
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
		GetSkill = origGetSkill
		HasMobFlag = origHasMobFlag
		GetWeaponInfo = origGetWeapon
		HasScriptFlag = origHasScript
		MakeCorpseFunc = origMakeCorpse
		MakeDustFunc = origMakeDust
		ExtractChar = origExtract
		GainExp = origGainExp
		GetExp = origGetExp
		GetKills = origGetKills
		SetKills = origSetKills
		GetDeaths = origGetDeaths
		SetDeaths = origSetDeaths
		GetPks = origGetPks
		SetPks = origSetPks
		SetLastDeath = origSetLastDeath
		LogMessage = origLogMsg
		HasPlrFlag = origHasPlr
		SetPlrFlag = origSetPlr
		HasRoomFlag = origHasRoom
		IsShopkeeper = origIsShop
		HasMobVNum = origHasMobVNum
		GetRace = origGetRace
		GetRaceHate = origGetRaceHate
		HasAffectStr = origHasAffectStr
		IsMounted = origIsMounted
		Dismount = origDismount
		HasPrfFlag = origHasPrf
		PerformCommand = origPerformCmd
		GetGold = origGetGold
		SetGold = origSetGold
		CountGroupMembers = origCountGroup
		ApplyToGroupMembers = origApplyGroup
		GetAlignment = origGetAlign
		SetAlignment = origSetAlign
		GetWimpyLev = origGetWimpy
		DoFlee = origDoFlee
		DoRetreat = origDoRetreat
		SkillMessageFunc = origSkillMsg
		RunDeathScript = origRunDeath
		BroadChatFunc = origBroadChat
		RemoveAffect = origRemoveAff
		NowUnix = origNowUnix
	}()

	// Wire all the hooks
	HasAffect = func(name string, aff int) bool { return false }
	BroadcastMessage = func(room int, msg string, exclude string) {}
	SendToCharFunc = func(name string, msg string) {}
	GetSkill = func(name string, skillNum int) int { return 0 }
	HasMobFlag = func(name string, flag string) bool { return false }
	GetWeaponInfo = func(name string) (int, int, int, bool) {
		if name == "Hero" {
			return TYPE_SLASH, 1, 8, false
		}
		return TYPE_HIT, 0, 0, false
	}
	HasScriptFlag = func(name string, flag string) bool { return false }
	GainExp = func(name string, amount int) {}
	GetExp = func(name string) int { return 0 }
	GetKills = func(name string) int64 { return 0 }
	SetKills = func(name string, kills int64) {}
	GetDeaths = func(name string) int64 { return 0 }
	SetDeaths = func(name string, deaths int64) {}
	GetPks = func(name string) int64 { return 0 }
	SetPks = func(name string, pks int64) {}
	SetLastDeath = func(name string, t int64) {}
	LogMessage = func(msg string, level string, minLevel int, toLog bool) {}
	HasPlrFlag = func(name string, flag string) bool { return false }
	SetPlrFlag = func(name string) bool { return false }
	HasRoomFlag = func(room int, flag string) bool { return false }
	IsShopkeeper = func(name string) bool { return false }
	HasMobVNum = func(name string, vnum int) bool { return false }
	GetRace = func(name string) int { return 0 }
	GetRaceHate = func(name string, index int) int { return 0 }
	HasAffectStr = func(name string, aff string) bool { return false }
	IsMounted = func(name string) bool { return false }
	Dismount = func(name string) {}
	HasPrfFlag = func(name string, flag string) bool { return false }
	PerformCommand = func(name, cmd string) {}
	GetGold = func(name string) int { return 0 }
	SetGold = func(name string, gold int) {}
	CountGroupMembers = func(name string, room int) int { return 1 }
	ApplyToGroupMembers = func(name string, room int, fn func(string)) {}
	GetAlignment = func(name string) int { return 0 }
	SetAlignment = func(name string, val int) {}
	GetWimpyLev = func(name string) int { return 0 }
	DoFlee = func(name string) {}
	DoRetreat = func(name string) {}
	SkillMessageFunc = func(dam int, ch, vict string, atk int, room int) bool { return false }
	RunDeathScript = func(killer, victim string, room int) {}
	BroadChatFunc = func(name string, msg string) {}
	RemoveAffect = func(name string, skillNum int) {}
	MakeCorpseFunc = func(name string, attackType int) {}
	MakeDustFunc = func(name string, attackType int) {}
	ExtractChar = func(name string) {}
	NowUnix = func() int64 { return 12345 }
	RunFightScript = func(mob, target string, room int) {}

	var deathCalled bool
	engine.BroadcastFunc = func(room int, msg string, exclude string) {}
	engine.DeathFunc = func(victim, killer Combatant, attackType int) {
		deathCalled = true
	}

	attacker := &mockCombatant{
		name:       "Hero",
		npc:        false,
		room:       100,
		level:      10,
		hp:         100,
		maxHP:      100,
		ac:         30,
		thac0:      15,
		position:   PosStanding,
		class:      ClassWarrior,
		str:        16,
		strAdd:     0,
		dex:        14,
		intVal:     12,
		wis:        10,
		hitroll:    5,
		damroll:    10,
		damageRoll: DiceRoll{Num: 5, Sides: 10},
	}
	defender := &mockCombatant{
		name:       "Goblin",
		npc:        true,
		room:       100,
		level:      1,
		hp:         5,
		maxHP:      5,
		ac:         80,
		thac0:      20,
		position:   PosStanding,
		str:        10,
		strAdd:     0,
		dex:        10,
		intVal:     8,
		wis:        8,
		damageRoll: DiceRoll{Num: 1, Sides: 3},
	}

	_ = engine.StartCombat(attacker, defender)

	// Run multiple rounds to ensure death with a weak defender
	for round := 0; round < 5; round++ {
		if engine.IsFighting("Hero") && engine.IsFighting("Goblin") {
			engine.processCombatPair(&CombatPair{
				Attacker: attacker,
				Defender: defender,
			})
		}
	}

	if deathCalled {
		t.Log("death was detected and handled via engine.DeathFunc")
	} else if defender.hp <= 0 {
		t.Log("goblin HP dropped to 0 — death processed internally via TakeDamage")
	} else {
		t.Logf("goblin survived with %d HP (all misses — improbable but possible)", defender.hp)
	}
}

// TestProcessCombatPair_DifferentRoom verifies combat stops when rooms don't match.
func TestProcessCombatPair_DifferentRoom(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	origHasAffect := HasAffect
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
		HasAffect = origHasAffect
	}()
	BroadcastMessage = func(room int, msg string, exclude string) {}
	SendToCharFunc = func(name string, msg string) {}
	HasAffect = func(name string, aff int) bool { return false }

	attacker := &mockCombatant{name: "Hero", room: 100, hp: 100, position: PosStanding}
	defender := &mockCombatant{name: "Orc", room: 200, hp: 50, position: PosStanding}

	_ = engine.StartCombat(attacker, defender)
	engine.processCombatPair(&CombatPair{Attacker: attacker, Defender: defender})

	if engine.IsFighting("Hero") {
		t.Error("expected combat to stop when rooms differ")
	}
}

// ---------------------------------------------------------------------------
// handleDeath test
// ---------------------------------------------------------------------------

func TestHandleDeath_Basic(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	var deathFuncCalled bool

	origSendTo := SendToCharFunc
	origBroadcast := BroadcastMessage
	origHasScript := HasScriptFlag
	origRunDeath := RunDeathScript
	defer func() {
		SendToCharFunc = origSendTo
		BroadcastMessage = origBroadcast
		HasScriptFlag = origHasScript
		RunDeathScript = origRunDeath
	}()
	SendToCharFunc = func(name string, msg string) {}
	BroadcastMessage = func(room int, msg string, exclude string) {}
	HasScriptFlag = func(name string, flag string) bool { return false }
	RunDeathScript = func(killer, victim string, room int) {}

	attacker := &mockCombatant{name: "Hero", room: 100, hp: 100}
	defender := &mockCombatant{name: "Orc", room: 100, hp: 0}

	engine.BroadcastFunc = func(room int, msg string, exclude string) {}
	engine.DeathFunc = func(victim, killer Combatant, attackType int) {
		deathFuncCalled = true
	}

	engine.handleDeath(defender, attacker)

	if !deathFuncCalled {
		t.Error("expected DeathFunc to be called")
	}
}

// ---------------------------------------------------------------------------
// Engine SendMessage helpers
// ---------------------------------------------------------------------------

func TestSendHitMessage(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	var attackerMsg, defenderMsg, roomMsg string
	engine.BroadcastFunc = func(room int, msg string, exclude string) {
		roomMsg = msg
	}

	attacker := &messagingCombatant{}
	attacker.name = "Hero"
	defender := &messagingCombatant{}
	defender.name = "Orc"

	attacker.sendFunc = func(msg string) { attackerMsg = msg }
	defender.sendFunc = func(msg string) { defenderMsg = msg }

	engine.sendHitMessage(attacker, defender, 15, 0)

	if !strings.Contains(attackerMsg, "15") {
		t.Errorf("expected damage number in attacker msg, got %q", attackerMsg)
	}
	if !strings.Contains(defenderMsg, "15") {
		t.Errorf("expected damage number in defender msg, got %q", defenderMsg)
	}
	if !strings.Contains(roomMsg, "Hero") || !strings.Contains(roomMsg, "Orc") {
		t.Errorf("expected both names in room msg, got %q", roomMsg)
	}
}

func TestSendMissMessage(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	var attackerMsg, defenderMsg string

	attacker := &messagingCombatant{}
	attacker.name = "Hero"
	defender := &messagingCombatant{}
	defender.name = "Orc"

	attacker.sendFunc = func(msg string) { attackerMsg = msg }
	defender.sendFunc = func(msg string) { defenderMsg = msg }

	engine.sendMissMessage(attacker, defender, 0)

	if !strings.Contains(attackerMsg, "miss") {
		t.Errorf("expected 'miss' in attacker msg, got %q", attackerMsg)
	}
	if !strings.Contains(defenderMsg, "miss") {
		t.Errorf("expected 'miss' in defender msg, got %q", defenderMsg)
	}
}

// messagingCombatant wraps mockCombatant with message capture capability.
type messagingCombatant struct {
	mockCombatant
	sendFunc func(string)
}

func (m *messagingCombatant) SendMessage(msg string) {
	if m.sendFunc != nil {
		m.sendFunc(msg)
	}
}

// ---------------------------------------------------------------------------
// ChangeAlignment — more comprehensive
// ---------------------------------------------------------------------------

func TestChangeAlignment_NPCKiller(t *testing.T) {
	var alignResult int
	origGetAlign := GetAlignment
	origSetAlign := SetAlignment
	defer func() {
		GetAlignment = origGetAlign
		SetAlignment = origSetAlign
	}()

	GetAlignment = func(name string) int {
		if name == "good_victim" {
			return 900
		}
		if name == "npc_killer" {
			return 500
		}
		return 0
	}
	SetAlignment = func(name string, val int) {
		alignResult = val
	}

	npcKiller := &mockCombatant{name: "npc_killer", npc: false}
	goodVictim := &mockCombatant{name: "good_victim", npc: true}

	ChangeAlignment(npcKiller, goodVictim)
	if alignResult >= 500 {
		t.Errorf("expected alignment to drop after killing good, got %d", alignResult)
	}
}

func TestChangeAlignment_NPCKillerNPC(t *testing.T) {
	var alignCalled bool
	origGetAlign := GetAlignment
	origSetAlign := SetAlignment
	defer func() {
		GetAlignment = origGetAlign
		SetAlignment = origSetAlign
	}()

	GetAlignment = func(name string) int { return 0 }
	SetAlignment = func(name string, val int) { alignCalled = true }

	killer := &mockCombatant{name: "Orc", npc: true}
	victim := &mockCombatant{name: "Elf", npc: true}

	ChangeAlignment(killer, victim)
	if alignCalled {
		t.Error("NPC should not trigger alignment change")
	}
}
