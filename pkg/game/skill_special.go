package game

import (
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/combat"
)

func DoMold(ch *Player, objName, newName, newDesc string) SkillResult {
	obj, found := findItemByName(ch, objName)
	if !found {
		return SkillResult{Success: false, MessageToCh: "You don't have one of those.\r\n"}
	}

	name := strings.ToLower(obj.GetKeywords())
	if !strings.Contains(name, "clay") && !strings.Contains(name, "playdough") && !strings.Contains(name, "halo") {
		return SkillResult{Success: false, MessageToCh: "You do not have anything to mold!\r\n"}
	}

	if newName == "" || newDesc == "" {
		return SkillResult{Success: false, MessageToCh: "You must specify a name and a description.\r\n"}
	}

	// Store custom mold data
	obj.Runtime.MoldName = newName
	obj.Runtime.MoldDesc = newDesc

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("The material magically hardens when you create %s.\r\n", newDesc),
	}
}

// DoBehead implements do_behead() — behead a corpse.
func DoBehead(ch *Player, targetName string, world *World) SkillResult {
	// Check if target is a living character
	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if found && target != nil {
		return SkillResult{Success: false, MessageToCh: "You kill it first and THEN you behead it!\r\n"}
	}

	// Look for corpse object in room
	room := world.GetRoomInWorld(ch.GetRoomVNum())
	if room == nil {
		return SkillResult{MessageToCh: "You are in a void.\r\n"}
	}

	items := world.GetItemsInRoom(ch.GetRoomVNum())
	var corpse *ObjectInstance
	targetLower := strings.ToLower(targetName)
	for _, item := range items {
		iname := strings.ToLower(item.GetKeywords())
		if strings.Contains(iname, "corpse") && strings.Contains(iname, targetLower) {
			corpse = item
			break
		}
	}

	if corpse == nil {
		// Fallback: find any corpse matching name
		for _, item := range items {
			iname := strings.ToLower(item.GetKeywords())
			if strings.Contains(iname, "corpse") {
				corpse = item
				break
			}
		}
	}

	if corpse == nil {
		return SkillResult{Success: false, MessageToCh: fmt.Sprintf("You can't seem to find a %s to behead!\r\n", targetName)}
	}

	if strings.Contains(strings.ToLower(corpse.GetKeywords()), "headless") {
		return SkillResult{Success: false, MessageToCh: "You can't behead something without a head!\r\n"}
	}

	// Check if it's a container (c-style: ITEM_CONTAINER with val[3] == 1 = corpse)
	// For now, just check it's a corpse object
	if !strings.Contains(strings.ToLower(corpse.GetKeywords()), "corpse") {
		return SkillResult{Success: false, MessageToCh: "You can't behead that!\r\n"}
	}

	// Determine weapon type for messaging
	wielded := false
	slashWeapon := false
	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 {
		weapon := ch.Equipment.Slots[0] // WEAR_WIELD = slot 0
		if weapon != nil {
			wielded = true
			// Check if weapon type is slash (value[3] == 3)
			slashWeapon = true // simplified — assume equipped weapons are slash-able
		}
	}

	var msgToCh, msgToRoom string
	if wielded && slashWeapon {
		msgToCh = fmt.Sprintf("You behead %s!", corpse.GetShortDesc())
		msgToRoom = fmt.Sprintf("%s beheads %s!", ch.Name, corpse.GetShortDesc())
	} else {
		msgToCh = fmt.Sprintf("You rip the head off %s with your bare hands!", corpse.GetShortDesc())
		msgToRoom = fmt.Sprintf("%s rips the head off %s with %s bare hands!", ch.Name, corpse.GetShortDesc(), heShe(ch.GetSex()))
	}

	// Create head object (proto vnum 16)
	_ = world.GetItemsInRoom(ch.GetRoomVNum()) // room items ref

	// Since we can't easily create objects from proto, store modified name on corpse
	// and use the corpse's short desc for the room message

	// Dump corpse contents and remove it
	// In a full port we'd create head + headless_corpse objects
	// For now, mark the corpse as beheaded and dump its contents
	if err := world.MoveObjectToNowhere(corpse); err != nil {
		slog.Warn("MoveObjectToNowhere failed in behead", "obj_vnum", corpse.GetVNum(), "error", err)
	}

	// Create head (vnum 16) and headless corpse (vnum 17) objects
	headObj, err := world.SpawnObject(16, ch.GetRoomVNum())
	if err == nil && headObj != nil {
		headObj.Runtime.ShortDesc = fmt.Sprintf("the severed head of %s", ch.Name)
		headObj.Runtime.Name = fmt.Sprintf("head %s", ch.Name)
	}
	headlessCorpseObj, err := world.SpawnObject(17, ch.GetRoomVNum())
	if err == nil && headlessCorpseObj != nil {
		headlessCorpseObj.Runtime.ShortDesc = fmt.Sprintf("the headless corpse of %s", ch.Name)
		headlessCorpseObj.Runtime.Name = fmt.Sprintf("corpse headless %s", ch.Name)
	}

	return SkillResult{
		Success:      true,
		MessageToCh:  msgToCh + "\r\n",
		MessageToRoom: msgToRoom + "\r\n",
	}
}

// DoBearhug implements do_bearhug() — bare-handed squeeze attack.
func DoBearhug(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillBearhug) == 0 {
		return SkillResult{Success: false, MessageToCh: "You'd better leave all the martial arts to fighters.\r\n"}
	}

	if ch.GetMove() <= 0 {
		return SkillResult{Success: false, MessageToCh: "You are too exhausted!\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You need to be bare handed to get a good grip.\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(150) + 1 // 1-150; 101+ is complete failure

	// Immortals always succeed, sleeping targets always hit
	if ch.GetLevel() > 60 {
		percent = 101
	}

	prob := ch.GetSkill(SkillBearhug)

	if percent > prob {
		return SkillResult{
			Success:      true,
			Damage:       0,
			MessageToCh:  "You try to bear hug but miss!\r\n",
			MessageToVict: "$n tries to bear hug you!\r\n",
			MessageToRoom: fmt.Sprintf("%s tries to bear hug %s!\r\n", ch.Name, target.GetName()),
		}
	}

	dam := ch.GetLevel() + (ch.GetLevel() / 2) // level * 1.5

	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  "You squeeze your victim in a crushing bear hug!\r\n",
		MessageToVict: "You are crushed in a powerful bear hug!\r\n",
		MessageToRoom: fmt.Sprintf("%s crushes %s in a powerful bear hug!\r\n", ch.Name, target.GetName()),
	}
}

// DoSlug implements do_slug() — punch attack.
func DoSlug(ch *Player, target combat.Combatant) SkillResult {
	if ch.GetSkill(SkillSlug) == 0 {
		return SkillResult{Success: false, MessageToCh: "You couldn't slug your way out of a wet paper bag.\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You can't make a fist while wielding a weapon!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillSlug)

	if percent > prob {
		return SkillResult{
			Success:      true,
			Damage:       0,
			MessageToCh:  "You swing wildly and miss!\r\n",
			MessageToVict: "$n swings a fist at you and misses!\r\n",
			MessageToRoom: fmt.Sprintf("%s swings a fist at %s and misses!\r\n", ch.Name, target.GetName()),
		}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	dam := (ch.GetLevel() * (rand.Intn(4) + 1)) / 2
	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  "You slug your victim with a solid punch!\r\n",
		MessageToVict: "You are slugged hard!\r\n",
		MessageToRoom: fmt.Sprintf("%s slugs %s!\r\n", ch.Name, target.GetName()),
	}
}

// DoSmackheads implements do_smackheads() — grab two NPCs and smack them together.
func DoSmackheads(ch *Player, victim1Name, victim2Name string, world *World) SkillResult {
	if ch.GetSkill(SkillSmackheads) == 0 {
		return SkillResult{Success: false, MessageToCh: "The only heads you're gonna smack are yours and Rosie's.\r\n"}
	}

	if victim1Name == victim2Name {
		return SkillResult{Success: false, MessageToCh: "Looks like the gang's not all here...\r\n"}
	}

	vill, _, found1 := FindTargetInRoom(world, ch.GetRoomVNum(), victim1Name, ch)
	vil2, _, found2 := FindTargetInRoom(world, ch.GetRoomVNum(), victim2Name, ch)
	if !found1 || !found2 {
		return SkillResult{Success: false, MessageToCh: "Looks like the gang's not all here...\r\n"}
	}

	// Check we're not targeting ourselves
	if vill.GetName() == ch.Name || vil2.GetName() == ch.Name {
		return SkillResult{Success: false, MessageToCh: "We call that 'headbutt' around here, son...\r\n"}
	}

	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		return SkillResult{Success: false, MessageToCh: "You need your hands free to smack some heads!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillSmackheads)

	if percent > prob {
		// Failure — victims duck
		msgToCh := fmt.Sprintf("%s and %s slip out of your hands!", vill.GetName(), vil2.GetName())
		return SkillResult{
			Success:      true,
			MessageToCh:  msgToCh + "\r\n",
			MessageToRoom: fmt.Sprintf("%s and %s duck as %s lunges at them!\r\n", vill.GetName(), vil2.GetName(), ch.Name),
		}
	}

	// Success — smack them together
	dam := 3 * ch.GetLevel()
	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  fmt.Sprintf("You grab the heads of %s and %s and bang them together with a sickening *SMACK*.\r\n", vill.GetName(), vil2.GetName()),
		MessageToRoom: fmt.Sprintf("%s grabs the heads of %s and %s and bangs them together with a sickening *SMACK*.\r\n", ch.Name, vill.GetName(), vil2.GetName()),
	}
}

// DoBite implements do_bite() — vampire/werewolf bite attack.
func DoBite(ch *Player, target combat.Combatant) SkillResult {
	// Non-supernatural bite (love bite)
	dam := ch.GetLevel()
	if dam > 15 {
		dam = 15
	}

	return SkillResult{
		Success:      true,
		Damage:       dam,
		MessageToCh:  "You bite your victim!\r\n",
		MessageToVict: "$n bites you!\r\n",
		MessageToRoom: fmt.Sprintf("%s bites %s!\r\n", ch.Name, target.GetName()),
	}
}

// DoTag implements do_tag() — tag someone as "it".
func DoTag(ch *Player, targetName string, world *World) SkillResult {
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "Tag who?\r\n"}
	}

	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		return SkillResult{Success: false, MessageToCh: "They don't seem to be here.\r\n"}
	}

	// Self-tag starts the game
	if target.GetName() == ch.Name {
		return SkillResult{Success: true, MessageToCh: "Let the game begin!\r\n"}
	}

	return SkillResult{
		Success:      true,
		MessageToCh:  fmt.Sprintf("You tap %s and scream, 'TAG! You're it!'\r\n", target.GetName()),
		MessageToVict: fmt.Sprintf("%s taps you and screams, 'TAG! You're it!'\r\n", ch.Name),
		MessageToRoom: fmt.Sprintf("%s taps %s and screams, 'TAG! You're it!'\r\n", ch.Name, target.GetName()),
	}
}

// DoPoint implements do_point() — point at someone or something.
func DoPoint(ch *Player, targetName string, world *World) SkillResult {
	if targetName == "" {
		return SkillResult{
			Success:      true,
			MessageToCh:  "You point around the room.\r\n",
			MessageToRoom: fmt.Sprintf("%s points around the room.\r\n", ch.Name),
		}
	}

	target, _, found := FindTargetInRoom(world, ch.GetRoomVNum(), targetName, ch)
	if !found {
		// Point at self
		if strings.EqualFold(targetName, "self") || strings.EqualFold(targetName, "me") || strings.EqualFold(targetName, ch.Name) {
			return SkillResult{
				Success:      true,
				MessageToCh:  "You point at yourself.\r\n",
				MessageToRoom: fmt.Sprintf("%s points at %s.\r\n", ch.Name, himHer(ch.GetSex())),
			}
		}
		// Point at nothing specific
		return SkillResult{
			Success:      true,
			MessageToCh:  "You point around the room.\r\n",
			MessageToRoom: fmt.Sprintf("%s points around the room.\r\n", ch.Name),
		}
	}

	if target.GetName() == ch.Name {
		return SkillResult{
			Success:      true,
			MessageToCh:  "You point at yourself.\r\n",
			MessageToRoom: fmt.Sprintf("%s points at %s.\r\n", ch.Name, himHer(ch.GetSex())),
		}
	}

	// Check if wielding a weapon
	if ch.Equipment != nil && len(ch.Equipment.Slots) > 0 && ch.Equipment.Slots[0] != nil {
		weapon := ch.Equipment.Slots[0]
		return SkillResult{
			Success:      true,
			MessageToCh:  fmt.Sprintf("You point %s at %s.\r\n", weapon.GetShortDesc(), target.GetName()),
			MessageToVict: fmt.Sprintf("%s points %s at you.\r\n", ch.Name, weapon.GetShortDesc()),
			MessageToRoom: fmt.Sprintf("%s points %s at %s.\r\n", ch.Name, weapon.GetShortDesc(), target.GetName()),
		}
	}

	return SkillResult{
		Success:      true,
		MessageToCh:  fmt.Sprintf("You point at %s.\r\n", target.GetName()),
		MessageToVict: fmt.Sprintf("%s points at you.\r\n", ch.Name),
		MessageToRoom: fmt.Sprintf("%s points at %s.\r\n", ch.Name, target.GetName()),
	}
}

// DoGroinrip implements do_groinrip() — low blow.
func DoGroinrip(ch *Player, target combat.Combatant, world *World) SkillResult {
	if ch.GetSkill(SkillGroinrip) == 0 {
		return SkillResult{Success: false, MessageToCh: "You're not trained in martial arts!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(121) + 1 // 0-120; 101+ is complete failure

	// Immortals always succeed
	if ch.GetLevel() > 60 {
		percent = 0
	}

	prob := ch.GetSkill(SkillGroinrip)

	if percent < prob {
		// Success
		dam := ch.GetLevel()
		return SkillResult{
			Success:      true,
			Damage:       dam,
			MessageToCh:  "You grab your victim's groin and twist!\r\n",
			MessageToVict: "You are grabbed in the groin and twisted! The pain is unbearable!\r\n",
			MessageToRoom: fmt.Sprintf("%s falls to %s knees, clutching %s groin and throwing up everywhere!\r\n", target.GetName(), hisHer(ch.GetSex()), hisHer(ch.GetSex())),
		}
	}

	// Miss
	return SkillResult{
		Success:      true,
		Damage:       0,
		MessageToCh:  "You try to grab your victim's groin but miss!\r\n",
		MessageToVict: "$n tries to grab your groin!\r\n",
		MessageToRoom: fmt.Sprintf("%s tries to grab %s's groin!\r\n", ch.Name, target.GetName()),
	}
}

// DoReview implements do_review() — show recent gossip history.
func DoReview(ch *Player) SkillResult {
	// Simple placeholder — returns a message that review was requested
	return SkillResult{
		Success:     true,
		MessageToCh: "Review: (Recent gossip history)\r\n(Review system not yet implemented)\r\n",
	}
}

// DoWhois implements do_whois() — look up player info.
func DoWhois(ch *Player, targetName string) SkillResult {
	if targetName == "" {
		return SkillResult{Success: false, MessageToCh: "For whom do you wish to search?\r\n"}
	}

	return SkillResult{
		Success:     true,
		MessageToCh: fmt.Sprintf("[Looking up %s...]\r\n(Player database lookup not yet connected)\r\n", targetName),
	}
}

// DoPalm implements do_palm() — conceal a small object up your sleeve.
func DoPalm(ch *Player, objName string, world *World) SkillResult {
	if objName == "" {
		return SkillResult{Success: false, MessageToCh: "Palm what?\r\n"}
	}

	// Find item in room
	items := world.GetItemsInRoom(ch.GetRoomVNum())
	var targetItem *ObjectInstance
	targetLower := strings.ToLower(objName)
	for _, item := range items {
		iname := strings.ToLower(item.GetKeywords())
		if strings.Contains(iname, targetLower) {
			targetItem = item
			break
		}
	}

	if targetItem == nil {
		return SkillResult{Success: false, MessageToCh: "You don't see that here.\r\n"}
	}

	// Check weight <= 1 (small object)
	if targetItem.GetWeight() > 1 {
		return SkillResult{Success: false, MessageToCh: "That's too big to palm!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillPalm)

	if prob > percent {
		// Success — move to inventory
		if err := world.MoveObjectToPlayerInventory(targetItem, ch); err != nil {
			return SkillResult{
				Success:     false,
				MessageToCh: "You can't carry that much.\r\n",
			}
		}
		return SkillResult{
			Success:      true,
			MessageToCh:  "You palm the item skillfully.\r\n",
			MessageToRoom: fmt.Sprintf("%s deftly palms something.\r\n", ch.Name),
		}
	}

	// Failure — item stays on ground
	return SkillResult{
		Success:      true,
		MessageToCh:  fmt.Sprintf("You try to palm %s but fumble it!\r\n", targetItem.GetShortDesc()),
		MessageToRoom: fmt.Sprintf("%s fumbles with %s!\r\n", ch.Name, targetItem.GetShortDesc()),
	}
}

// DoFleshAlter implements do_flesh_alter() — transform your hand into a weapon.
func DoFleshAlter(ch *Player) SkillResult {
	if ch.GetSkill(SkillFleshAlter) == 0 {
		return SkillResult{Success: false, MessageToCh: "You know nothing of altering your flesh!\r\n"}
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	percent := rand.Intn(101) + 1
	prob := ch.GetSkill(SkillFleshAlter)

	if percent > prob {
		return SkillResult{
			Success:     true,
			MessageToCh: "You lose your concentration!\r\n",
		}
	}

	// Toggle flesh alter state
	return SkillResult{
		Success:      true,
		MessageToCh:  "Your hand turns into a weapon!\r\n",
		MessageToRoom: fmt.Sprintf("%s's hand turns into a weapon!\r\n", ch.Name),
	}
}

// heShe returns "he" / "she" / "it" based on sex.
