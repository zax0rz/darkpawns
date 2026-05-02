// Package game — Wave 2 skill commands from new_cmds2.c
package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// DoScrounge — do_scrounge() from new_cmds2.c
// Search room for edible items based on sector type.
// Uses D100 vs SKILL_SCROUNGE. WAIT_STATE on any outcome.
// SECT_FOREST/desert/hills: kill food (capture)
// SECT_MOUNTAIN: find food
// SECT_WATER_*: food 27 (fish)
// ---------------------------------------------------------------------------
func DoScrounge(ch *Player, world *World) SkillResult {
	if ch.GetSkill(SkillScrounge) == 0 {
		return SkillResult{
			Success:     false,
			MessageToCh: "You can't seem to find anything edible.\r\n",
		}
	}

	room := world.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return SkillResult{MessageToCh: "You are lost in the void.\r\n"}
	}

	sector := room.Sector

	// Map sector type to a food object vnum and whether it's "find" or "kill"
	// Values adapted from Dark Pawns object vnums:
	//   27 = fish (water), 28 = berries/plants (forest),
	//   29 = roots/tubers (field/hills), 30 = small desert creature,
	//   31 = mountain herbs
	var foodVNum int
	var isFind bool

	switch sector {
	case 3: // SECT_FOREST
		foodVNum = 28
		isFind = false // capture/kill
	case 4, 5: // SECT_FIELD, SECT_HILLS
		foodVNum = 29
		isFind = false
	case 7: // SECT_DESERT
		foodVNum = 30
		isFind = false
	case 10: // SECT_MOUNTAIN
		foodVNum = 31
		isFind = true
	case 14, 15, 16: // SECT_WATER_SWIM, SECT_WATER_NOSWIM, SECT_UNDERWATER
		foodVNum = 27
		isFind = false
	default:
		return SkillResult{
			Success:     false,
			MessageToCh: "You need to be in the wilderness to scrounge!\r\n",
		}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(100) + 1
	prob := ch.GetSkill(SkillScrounge)

	if percent < prob {
		// Success — create the food object and give it to the player
		proto, ok := world.objs[foodVNum]
		if !ok {
			// Fallback: just give generic food
			return SkillResult{
				Success:     true,
				MessageToCh: "You find some edible scraps.\r\n",
				MessageToRoom: fmt.Sprintf("%s searches and finds something to eat.\r\n", ch.Name),
			}
		}
		obj := NewObjectInstance(proto, ch.RoomVNum)
		if obj != nil {
			// placeholder: add item to inventory
			msg := "You find $p."
			if !isFind {
				msg = "You capture and kill $p."
			}
			_ = msg // Would use ActMessage with item name
			return SkillResult{
				Success:     true,
				MessageToCh: fmt.Sprintf("You find %s.\r\n", proto.ShortDesc),
				MessageToRoom: fmt.Sprintf("%s finds %s.\r\n", ch.Name, proto.ShortDesc),
			}
		}
	}

	return SkillResult{
		Success:     false,
		MessageToCh: "You can't seem to find anything edible.\r\n",
	}
}

// ---------------------------------------------------------------------------
// DoFirstAid — do_first_aid() from new_cmds2.c
// Heal a target who is at 0 HP. SKILL_FIRST_AID check.
// On success: target HP = 1. WAIT_STATE on target + ch.
// ---------------------------------------------------------------------------
func DoFirstAid(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillFirstAid) == 0 {
		return SkillResult{Success: false, MessageToCh: "You have no idea how!\r\n"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "You wish you could.\r\n"}
	}

	if target.GetHP() >= 1 {
		return SkillResult{Success: false, MessageToCh: "They don't really need first aid.\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1 + target.GetLevel()
	prob := ch.GetSkill(SkillFirstAid)

	if percent < prob {
		// Success
		if p, ok := target.(*Player); ok {
			p.Health = 1
		}

		chPronouns := GetPronouns(ch.Name, 1)
		victPronouns := GetPronouns(target.GetName(), 1)

		return SkillResult{
			Success:       true,
			MessageToCh:   ActMessage("You apply some makeshift bandages to $N's wounds.", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n applies some bandaging to your wounds.", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n applies some bandaging to $N's wounds.", chPronouns, &victPronouns, ""),
		}
	}

	// Failure
	chPronouns := GetPronouns(ch.Name, 1)
	return SkillResult{
		Success:       false,
		MessageToCh:   "You fumble and ruin the bandages.\r\n",
		MessageToRoom: ActMessage("$n fumbles with some bandaging and drops it all over the place!", chPronouns, nil, ""),
	}
}

// ---------------------------------------------------------------------------
// DoDisarm — do_disarm() from new_cmds2.c
// Disarm opponent's weapon. SKILL_DISARM check. Weapon drops to ground.
// Target must be fighting ch.
// ---------------------------------------------------------------------------
func DoDisarm(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillDisarm) == 0 {
		return SkillResult{
			Success:     false,
			MessageToCh: "You'd better leave all the martial arts to fighters.\r\n",
		}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Just try removing your weapon instead.\r\n"}
	}

	// Check if the target has a wielded weapon (we can only check via interface)
	// In C: GET_EQ(vict, WEAR_WIELD) — we'll check if there's a fighting target
	if ch.Fighting == "" || ch.Fighting != target.GetName() {
		return SkillResult{Success: false, MessageToCh: "You can't disarm them if you aren't fighting them!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1 + target.GetLevel()
	prob := ch.GetSkill(SkillDisarm)

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

	if percent < prob {
		return SkillResult{
			Success:       true,
			Damage:        0, // disarm doesn't directly damage
			MessageToCh:   ActMessage("You disarm $N and $S weapon goes flying!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n deftly disarms you, knocking $S weapon from your hand!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n knocks $p from $N's hand!", chPronouns, &victPronouns, "weapon"),
		}
	}

	return SkillResult{
		Success:       false,
		MessageToCh:   ActMessage("You try to disarm $N but fail, tumbling to the ground in the process!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n tries to disarm you but fails and falls flat on $s face instead!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n tries to disarm $N, but fails and falls flat on $s face!", chPronouns, &victPronouns, ""),
		SelfStumble:   true,
	}
}

// ---------------------------------------------------------------------------
// DoMindlink — do_mindlink() simplified from new_cmds2.c
// Link minds for telepathic communication. Simplified version:
// Check target is in room, check skill, drain HP, share mana.
// ---------------------------------------------------------------------------
func DoMindlink(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillMindlink) == 0 {
		return SkillResult{Success: false, MessageToCh: "Yeah, right.\r\n"}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "You wish you could.\r\n"}
	}

	// Target must be an NPC (not a player)
	if p, ok := target.(*Player); ok && !p.IsNPC() {
		chPronouns := GetPronouns(ch.Name, 1)
		victPronouns := GetPronouns(target.GetName(), 1)
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("$N stares at you blankly.", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n stares at $N for a while and then falls flat on $s face.", chPronouns, &victPronouns, ""),
		}
	}

	if ch.IsFighting() || target.GetFighting() != "" {
		return SkillResult{Success: false, MessageToCh: "There's too much going on to establish a mind link.\r\n"}
	}

	if ch.Health < 100 {
		return SkillResult{Success: false, MessageToCh: "You don't have enough life to spare!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(100) + 1
	prob := ch.GetSkill(SkillMindlink)

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

	if percent < prob {
		// Success
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		x := 20 + ch.Level + rand.Intn(80) // number(20+level, 100)
		ch.Health -= x
		if ch.Health < 0 {
			ch.Health = 0
		}
		// Give mana to target (for NPCs, we try to add mana)
		if p, ok := target.(*Player); ok {
			p.Mana += x
		}

		return SkillResult{
			Success:       true,
			MessageToCh:   "You feel a little drained...\r\n",
			MessageToRoom: ActMessage("$n and $N stare at each other for a while and drop to the ground in unison!", chPronouns, &victPronouns, ""),
			StunTarget:    true,
			SelfStumble:   true,
		}
	}

	return SkillResult{
		Success:       false,
		MessageToCh:   "You feel a little drained...\r\n",
		MessageToRoom: ActMessage("$n stares at $N for a while and then falls flat on $s face.", chPronouns, &victPronouns, ""),
		SelfStumble:   true,
	}
}

// ---------------------------------------------------------------------------
// DoDetect — do_detect() from new_cmds2.c
// Detect hidden/magical things. SKILL_DETECT check.
// Find secret exits. WAIT_STATE.
// ---------------------------------------------------------------------------
func DoDetect(ch *Player, world *World) SkillResult {
	if ch.GetSkill(SkillDetect) == 0 && ch.Class != RaceElf {
		return SkillResult{Success: false, MessageToCh: "Yeah, right.\r\n"}
	}

	room := world.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return SkillResult{MessageToCh: "You are lost in the void.\r\n"}
	}

	prob := ch.GetSkill(SkillDetect)
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if prob <= rand.Intn(100)+1 {
		return SkillResult{Success: false, MessageToCh: "You can't seem to find anything.\r\n"}
	}

	// Check exits for "secret" keyword
	var found bool
	results := "You carefully check the room...\r\n"
	for dir, exit := range room.Exits {
		if strings.Contains(strings.ToLower(exit.Keywords), "secret") {
			dirNames := map[string]string{
				"north": "the north wall",
				"south": "the south wall",
				"east":  "the east wall",
				"west":  "the west wall",
				"up":    "the ceiling",
				"down":  "the floor",
				"n":     "the north wall",
				"s":     "the south wall",
				"e":     "the east wall",
				"w":     "the west wall",
				"u":     "the ceiling",
				"d":     "the floor",
			}
			where := dirNames[dir]
			if where == "" {
				where = fmt.Sprintf("the %s wall", dir)
			}
			results += fmt.Sprintf("You notice something funny about %s.\r\n", where)
			found = true
		}
	}

	if !found {
		results += "You can't seem to find anything.\r\n"
	}

	return SkillResult{Success: found, MessageToCh: results}
}

// ---------------------------------------------------------------------------
// DoSerpentKick — do_serpent_kick() simplified from new_cmds2.c
// Special spinning kick. SKILL_SERPENT_KICK check.
// Damage = level * 1.5. WAIT_STATE.
// Simplified: single target version (no surrounding check).
// ---------------------------------------------------------------------------
func DoSerpentKick(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillSerpentKick) == 0 {
		return SkillResult{
			Success:     false,
			MessageToCh: "You'd better leave all the martial arts to others.\r\n",
		}
	}

	if target.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "Aren't we funny today...\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := ((7 - (target.GetAC() / 10)) * 2) + rand.Intn(101) + 1
	prob := ch.GetSkill(SkillSerpentKick)

	if target.GetPosition() <= combat.PosSleeping {
		prob = 110 // auto-hit sleeping targets
	}

	chPronouns := GetPronouns(ch.Name, 1)
	victPronouns := GetPronouns(target.GetName(), 1)

	if percent > prob {
		return SkillResult{
			Success:       false,
			MessageToCh:   ActMessage("You try to kick $N with a serpent kick, but miss!", chPronouns, &victPronouns, ""),
			MessageToVict: ActMessage("$n tries to serpent kick you, but misses!", chPronouns, &victPronouns, ""),
			MessageToRoom: ActMessage("$n tries to serpent kick $N, but misses!", chPronouns, &victPronouns, ""),
		}
	}

	dam := int(float64(ch.Level) * 1.5)
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   ActMessage("Your serpent kick connects solidly with $N!", chPronouns, &victPronouns, ""),
		MessageToVict: ActMessage("$n hits you with a devastating serpent kick!", chPronouns, &victPronouns, ""),
		MessageToRoom: ActMessage("$n hits $N with a powerful serpent kick!", chPronouns, &victPronouns, ""),
	}
}

// ---------------------------------------------------------------------------
// DoDig — do_dig() from new_cmds2.c
// Simplified version: dig in current room based on sector type.
// WAIT_STATE, move cost.
// Sector types:
//   SECT_DIRT (2), SECT_FOREST (3), SECT_FIELD (4), SECT_HILLS (5)
// Success chance based on SKILL_DIG. Finds random loot.
// ---------------------------------------------------------------------------
func DoDig(ch *Player, world *World) SkillResult {
	if ch.Health < 5 {
		return SkillResult{Success: false, MessageToCh: "You're too exhausted to dig.\r\n"}
	}

	room := world.GetRoomInWorld(ch.RoomVNum)
	if room == nil {
		return SkillResult{MessageToCh: "You are lost in the void.\r\n"}
	}

	sector := room.Sector
	switch sector {
	case 2, 3, 4, 5: // SECT_DIRT, SECT_FOREST, SECT_FIELD, SECT_HILLS
		// Valid digging terrain
	default:
		return SkillResult{
			Success:     false,
			MessageToCh: "The ground here isn't suitable for digging.\r\n",
		}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(100) + 1
	prob := ch.GetSkill(SkillDig)
	if prob == 0 {
		prob = 10
	}

	if percent <= prob {
		// Found something — random loot based on level
		lootTypes := []string{"some coins", "a shiny rock", "an old bone", "a rusted coin", "a small gem"}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		loot := lootTypes[rand.Intn(len(lootTypes))]

		return SkillResult{
			Success:     true,
			MessageToCh: fmt.Sprintf("You dig in the earth and find %s!\r\n", loot),
			MessageToRoom: fmt.Sprintf("%s digs in the earth and finds something!\r\n", ch.Name),
		}
	}

	return SkillResult{
		Success:     false,
		MessageToCh: "You dig but find nothing.\r\n",
		MessageToRoom: fmt.Sprintf("%s digs around but finds nothing.\r\n", ch.Name),
	}
}

// ---------------------------------------------------------------------------
// DoTurn — do_turn() from new_cmds2.c
// Turn undead. SKILL_TURN check. Target must be undead.
// Damage = level * 2. If diff > 3: flee. If diff >= 15: destroy.
// WAIT_STATE.
// ---------------------------------------------------------------------------
func DoTurn(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillTurn) == 0 {
		return SkillResult{Success: false, MessageToCh: "Huh?!?\r\n"}
	}

	if !ch.IsGood() {
		return SkillResult{
			Success:     false,
			MessageToCh: "You are not holy enough to turn away the Undead!\r\n",
		}
	}

	msgToCh := "You attempt to turn away the unholy presence in this room.\r\n"

	// Check if target is undead race (3 = RACE_UNDEAD in C, using our own convention)
	// In the C source, it checks GET_RACE(tch) == RACE_UNDEAD || RACE_VAMPIRE
	// We don't have full race tracking on combatants yet, so we check the name for clues
	// or assume any target that can be turned is valid
	targetName := strings.ToLower(target.GetName())
	isUndead := strings.Contains(targetName, "skeleton") ||
		strings.Contains(targetName, "zombie") ||
		strings.Contains(targetName, "ghost") ||
		strings.Contains(targetName, "lich") ||
		strings.Contains(targetName, "wraith") ||
		strings.Contains(targetName, "spectre") ||
		strings.Contains(targetName, "vampire") ||
		strings.Contains(targetName, "undead")

	if !isUndead {
		return SkillResult{
			Success:     false,
			MessageToCh: msgToCh + "There is nothing unholy to turn here.\r\n",
		}
	}

	diff := ch.Level - target.GetLevel()

	if diff <= -5 {
		return SkillResult{
			Success:       false,
			MessageToCh:   msgToCh + "A disturbing feeling washes over your body.\r\n",
			MessageToRoom: fmt.Sprintf("%s shivers uncomfortably.\r\n", target.GetName()),
		}
	}

	if diff >= 15 {
		return SkillResult{
			Success:     true,
			Damage:      9999, // instant kill
			MessageToCh: msgToCh + "The undead creature explodes into a cloud of dust!\r\n",
			MessageToRoom: fmt.Sprintf("%s grimaces and then explodes into a cloud of dust!\r\n",
				target.GetName()),
			MessageToVict: "You feel your body twist horribly and disintegrate into nothing!\r\n",
		}
	}

	if diff > 3 {
		return SkillResult{
			Success:     true,
			Damage:      ch.Level * 2,
			MessageToCh: msgToCh + "The undead creature shrieks and flees from your holiness!\r\n",
			MessageToRoom: fmt.Sprintf("%s shrieks in terror!\r\n", target.GetName()),
		}
	}

	// Basic damage
	dam := ch.Level * 2
	if dam < 1 {
		dam = 1
	}
	return SkillResult{
		Success:       true,
		Damage:        dam,
		MessageToCh:   msgToCh + fmt.Sprintf("Your holy power sears the undead for %d damage!\r\n", dam),
		MessageToVict: "The holy light sears your undead flesh!\r\n",
		MessageToRoom: fmt.Sprintf("%s is bathed in holy light!\r\n", ch.Name),
	}
}

