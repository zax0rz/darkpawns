// spec_procs.go — Special procedure implementations for mobiles/objects/rooms.
//
// Ported from Dark Pawns MUD C source (spec_procs.c).
// Each handler is registered via RegisterSpec in an init() function.
//
// Signature: func(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool
//   - cmd is the command string typed by the player (e.g. "practice", "drop", "north")
//   - cmd == "" means the mob triggers on its own pulse/tick
//   - arg is the remainder of the command line after the command
//   - return true if the spec handled the interaction (blocking further processing)
//   - return false if the spec did not handle it

package game

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// ================================================================
// Helpers
// ================================================================

func randRange(min, max int) int {
	if min > max {
		return min
	}
	return rand.Intn(max-min+1) + min
}

func randN(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}

func (w *World) roomMessage(roomVNum int, msg string) {
	players := w.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		p.SendMessage(msg + "\r\n")
	}
}

func sendToChar(ch *Player, msg string) {
	ch.SendMessage(msg + "\r\n")
}

func isMoveCmd(cmd string) bool {
	switch cmd {
	case "north", "south", "east", "west", "up", "down",
		"n", "s", "e", "w", "u", "d":
		return true
	}
	return false
}

func (w *World) roomCleanup(roomVNum int) int {
	items := w.GetItemsInRoom(roomVNum)
	totalVal := 0
	for _, obj := range items {
		w.roomMessage(roomVNum, obj.GetShortDesc()+" vanishes in a puff of smoke!")
		w.RemoveItemFromRoom(obj, roomVNum)
		cost := obj.GetCost()
		if cost < 1 {
			cost = 1
		}
		v := cost / 10
		if v < 1 {
			v = 1
		}
		if v > 10 {
			v = 10
		}
		totalVal += v
	}
	return totalVal
}

// mobMeleeTarget returns the mob's current melee opponent as a MobInstance.
func mobMeleeTarget(me *MobInstance) *MobInstance {
	if me.Target != nil {
		return me.Target
	}
	return nil
}

// ================================================================
// MOB SPECIALS
// ================================================================

// guild — practice skills with a guildmaster mob
func specGuild(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() || cmd != "practice" {
		return false
	}

	if ch.SkillManager == nil {
		sendToChar(ch, "You do not seem to be able to practice now.")
		return true
	}

	if arg == "" {
		sendToChar(ch, "Practise what?  You know of the following skills:")
		for _, skill := range ch.SkillManager.GetLearnedSkills() {
			sendToChar(ch, fmt.Sprintf("  %s (%d%%)", skill.DisplayName, skill.Level))
		}
		return true
	}

	skillName := strings.ToLower(strings.TrimSpace(arg))
	skill := ch.SkillManager.GetSkill(skillName)
	if skill == nil {
		sendToChar(ch, "You do not know of that skill.")
		return true
	}
	if !skill.Learned {
		sendToChar(ch, "You do not know of that skill.")
		return true
	}
	if skill.Difficulty > ch.GetLevel() {
		sendToChar(ch, "You do not know of that skill.")
		return true
	}
	if skill.Level >= skill.MaxLevel {
		sendToChar(ch, "You are already learned in that area.")
		return true
	}
	if skill.Practice <= 0 {
		sendToChar(ch, "You do not seem to be able to practice now.")
		return true
	}

	sendToChar(ch, "You practice for a while...")

	intScore := 10 // fallback if Stats not populated
	ch.SkillManager.PracticeSkill(skillName, ch.GetLevel(), intScore)

	return true
}

// dump — room spec: trash items vanish, player gets XP/gold
func specDump(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	roomVNum := ch.GetRoomVNum()
	_ = w.roomCleanup(roomVNum)

	if cmd != "drop" {
		return false
	}

	value := w.roomCleanup(roomVNum)
	if value > 0 {
		sendToChar(ch, "You are awarded for outstanding performance.")
		w.roomMessage(roomVNum, ch.GetName()+" has been awarded by the gods!")
		if ch.GetLevel() < 3 {
			ch.Exp += value
		} else {
			ch.Gold += value
		}
	}
	return true
}

// snake — mob spec: poison bite in combat
func specSnake(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	if randN(32-me.GetLevel()) != 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil {
		return false
	}
	w.roomMessage(me.RoomVNum, me.GetName()+" bites "+melee.GetName()+"!")
	spells.Cast(me, melee, spells.SpellPoison, me.GetLevel(), nil)
	return true
}

// summoner — mob spec: summons player to it
func specSummoner(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosStanding {
		return false
	}
	var vict *Player
	for _, memName := range me.Memory {
		if p, ok := w.players[memName]; ok {
			vict = p
			break
		}
	}
	if vict == nil {
		for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
			if !p.IsNPC() {
				for _, memName := range me.Memory {
					if memName == p.Name {
						vict = p
						break
					}
				}
				if vict != nil {
					break
				}
			}
		}
	}
	if vict != nil && randN(4) == 0 {
		spells.Cast(me, vict, spells.SpellTeleport, me.GetLevel(), nil)
		if me.RoomVNum == vict.GetRoomVNum() {
			me.Fighting = true
			me.FightingTarget = vict.Name
		}
		return true
	}
	return false
}

// thief — mob spec: steals gold
func specThief(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosStanding {
		return false
	}
	for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
		if !p.IsNPC() && p.GetLevel() < 50 && randN(5) == 0 {
			npcSteal(w, me, p)
			return true
		}
	}
	return false
}

func npcSteal(w *World, me *MobInstance, victim *Player) {
	if victim.IsNPC() || victim.GetLevel() >= 50 {
		return
	}
	if victim.GetPosition() > combat.PosSleeping && randN(me.GetLevel()) == 0 {
		w.roomMessage(me.RoomVNum, me.GetName()+" tries to steal gold from "+victim.GetName()+".")
		sendToChar(victim, "You discover that "+me.GetName()+" has its hands in your wallet.")
	} else {
		gold := (victim.Gold * randRange(1, 10)) / 100
		if gold > 0 {
			victim.Gold -= gold
		}
	}
}

// magic_user — mob spec: casts combat spells while fighting
func specMagicUser(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}

	var vict *Player
	for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
		if p.IsFighting() && p.GetName() == me.GetName() && randN(5) == 0 {
			vict = p
			break
		}
	}
	if vict == nil {
		if tName := me.FightingTarget; tName != "" {
			if p, ok := w.players[tName]; ok {
				vict = p
			}
		}
	}
	if vict == nil {
		return false
	}

	spellRoll := randN(me.GetLevel()/2+1) + me.GetLevel()/2
	switch {
	case spellRoll <= 5:
		spells.Cast(me, vict, spells.SpellMagicMissile, me.GetLevel(), nil)
	case spellRoll <= 7:
		spells.Cast(me, vict, spells.SpellChillTouch, me.GetLevel(), nil)
	case spellRoll <= 9:
		spells.Cast(me, vict, spells.SpellBurningHands, me.GetLevel(), nil)
	case spellRoll <= 11:
		spells.Cast(me, vict, spells.SpellShockingGrasp, me.GetLevel(), nil)
	case spellRoll == 12:
		spells.Cast(me, vict, spells.SpellDispelGood, me.GetLevel(), nil)
	case spellRoll == 13:
		spells.Cast(me, vict, spells.SpellLightningBolt, me.GetLevel(), nil)
	case spellRoll == 14:
		if randN(11) == 0 {
			spells.Cast(me, vict, spells.SpellTeleport, me.GetLevel(), nil)
		}
	case spellRoll >= 15 && spellRoll <= 17:
		spells.Cast(me, vict, spells.SpellColorSpray, me.GetLevel(), nil)
	case spellRoll == 20:
		spells.Cast(me, vict, spells.SpellHellfire, me.GetLevel(), nil)
	case spellRoll == 25:
		spells.Cast(me, vict, spells.SpellFlamestrike, me.GetLevel(), nil)
	case spellRoll == 30:
		spells.Cast(me, vict, spells.SpellDisintegrate, me.GetLevel(), nil)
	case spellRoll >= 31 && spellRoll <= 33:
		spells.Cast(me, vict, spells.SpellDisrupt, me.GetLevel(), nil)
	case spellRoll == 34:
		spells.Cast(me, vict, spells.SpellInvulnerability, me.GetLevel(), nil)
	case spellRoll >= 35 && spellRoll <= 36:
		spells.Cast(me, vict, spells.SpellFlamestrike, me.GetLevel(), nil)
	case spellRoll == 37:
		spells.Cast(me, vict, spells.SpellMeteorSwarm, me.GetLevel(), nil)
	case spellRoll == 38:
		spells.Cast(me, vict, spells.SpellDisrupt, me.GetLevel(), nil)
	default:
		spells.Cast(me, vict, spells.SpellFireball, me.GetLevel(), nil)
	}
	return true
}

// fighter — mob spec: uses martial skills in combat
func specFighter(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil {
		return false
	}
	switch randN(11) {
	case 1:
		w.roomMessage(me.RoomVNum, me.GetName()+" headbutts "+melee.GetName()+"!")
	case 2:
		w.roomMessage(me.RoomVNum, me.GetName()+" parries an attack!")
	case 3:
		w.roomMessage(me.RoomVNum, me.GetName()+" bashes "+melee.GetName()+"!")
	case 4:
		w.roomMessage(me.RoomVNum, me.GetName()+" goes berserk!")
	default:
		return false
	}
	return true
}

// paladin — mob spec: paladin combat
func specPaladin(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil {
		return false
	}
	switch randN(9) {
	case 0:
		w.roomMessage(me.RoomVNum, me.GetName()+" parries an attack!")
	case 1:
		w.roomMessage(me.RoomVNum, me.GetName()+" bashes "+melee.GetName()+"!")
	case 2:
		w.roomMessage(me.RoomVNum, me.GetName()+" charges "+melee.GetName()+"!")
	case 3:
		spells.Cast(me, melee, spells.SpellDispelEvil, me.GetLevel(), nil)
	case 5:
		w.roomMessage(me.RoomVNum, me.GetName()+" disarms "+melee.GetName()+"!")
	}
	return true
}

// guild_guard — mob spec: blocks unauthorized entry
func specGuildGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "flee" || cmd == "escape" || cmd == "retreat" {
		sendToChar(ch, "You try to flee inside the guild but the guard stops you!")
		w.roomMessage(ch.GetRoomVNum(), ch.GetName()+" tries to flee inside the guild but the guard blocks the way!")
		return true
	}
	if !isMoveCmd(cmd) {
		if me.Fighting {
			return specFighter(w, ch, me, cmd, arg)
		}
		return false
	}
	if ch.GetLevel() >= 50 || ch.IsNPC() {
		return false
	}

	type guildEntry struct {
		class     int
		room      int
		direction string
	}
	entries := []guildEntry{
		{ClassThief, 4813, "south"},
		{ClassMagus, 4821, "south"},
		{ClassMystic, 4825, "south"},
		{ClassNinja, 8012, "south"},
		{ClassAssassin, 8013, "south"},
		{ClassPaladin, 8015, "south"},
		{ClassMageUser, 21214, "south"},
		{ClassCleric, 21215, "south"},
		{ClassWarrior, 21216, "south"},
		{ClassRanger, 21217, "south"},
		{ClassPsionic, 8024, "south"},
		{ClassMagus, 8026, "south"},
	}
	roomVNum := ch.GetRoomVNum()
	for _, e := range entries {
		if ch.GetClass() != e.class && roomVNum == e.room && cmd == e.direction {
			sendToChar(ch, "The guard humiliates you, and blocks your way.")
			w.roomMessage(roomVNum, "The guard humiliates "+ch.GetName()+", and blocks their way.")
			return true
		}
	}
	return false
}

// puff — mob spec: random says on pulse
func specPuff(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}
	if me.GetHP() < 0 {
		w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Shit, I'm dead.'")
		return true
	}
	puffSayings := []string{
		"My god!  It's full of stars!",
		"How'd all those fish get up here?",
		"I'm a very female dragon.",
		"Boo!  Hiss!  I say!",
		"'The voices!  The voices!'",
		"Why are there so many songs about rainbows?",
		"'Help!  I'm being repressed!'",
		"What is the capital of Assyria?",
		"Our Lady of Blessed Acceleration, don't fail me now.",
		"Hi, do you have any Grey Poupon?",
		"Are we there yet?",
		"'We're not worthy!  We're not worthy!'",
		"What's the color of the wind?",
		"I see dead people.",
		"Is that a flame thrower in your pocket...",
		"'I have a nice, heavy club for you.'",
		"'She turned me into a newt!'",
		"'...I got better...'",
		"There is no magic, only rearranged physics.",
		"Filthy, Precious!  It stole us, Precious!",
		"I have no legs, and I must scream.",
		"Reach out and touch faith.",
		"Life?  Don't talk to me about life.",
		"He's not the Messiah, he's a very naughty boy!",
		"Nobody expects the Spanish Inquisition!",
		"Help, I'm a bug!",
		"I'm melting!  What a world!  What a world!",
		"Follow the yellow brick road.",
		"Praise Helix!",
		"If I only had a brain.",
		"More tea, vicar?",
		"I'll get you, my pretty!",
		"I'll be back.",
		"Negative.  I am a meat popsicle.",
	}
	if randN(91) == 0 {
		saying := puffSayings[randN(len(puffSayings))]
		w.roomMessage(me.RoomVNum, me.GetName()+" says, '"+saying+"'")
		return true
	}
	return false
}

// fido — mob spec: dog scavenges corpses
func specFido(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || randN(3) != 0 {
		return false
	}
	items := w.GetItemsInRoom(me.RoomVNum)
	for _, obj := range items {
		if strings.Contains(obj.GetKeywords(), "corpse") {
			w.roomMessage(me.RoomVNum, me.GetName()+" savagely devours "+obj.GetShortDesc()+".")
			w.RemoveItemFromRoom(obj, me.RoomVNum)
			return true
		}
	}
	return false
}

// janitor — mob spec: cleans up items
func specJanitor(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || randN(5) != 0 {
		return false
	}
	items := w.GetItemsInRoom(me.RoomVNum)
	for _, obj := range items {
		if !strings.Contains(obj.GetKeywords(), "corpse") && randN(2) == 0 {
			w.roomMessage(me.RoomVNum, me.GetName()+" picks up "+obj.GetShortDesc()+".")
			w.RemoveItemFromRoom(obj, me.RoomVNum)
			return true
		}
	}
	return false
}

// cityguard — mob spec: guards arrest outlaws
func specCityguard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isMoveCmd(cmd) && ch.GetRoomVNum() == me.RoomVNum {
		flags := ch.GetFlags()
		if flags&1 != 0 { // PLR_OUTLAW
			sendToChar(ch, me.GetName()+" says, 'HALT!  You are under arrest!'")
			w.roomMessage(me.RoomVNum, me.GetName()+" bars "+ch.GetName()+"'s way!")
			return true
		}
	}
	if me.Fighting {
		return true
	}
	return false
}

// mayor — mob spec: walks around greeting people
func specMayor(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || randN(4) != 0 {
		return false
	}
	mayorSayings := []string{
		"Hello, mate!",
		"Nice to see you!",
		"How d'you do?",
		"Another fine day!",
		"Welcome to New Thalos!",
		"Good day!",
		"Lovely to meet you!",
	}
	saying := mayorSayings[randN(len(mayorSayings))]
	w.roomMessage(me.RoomVNum, me.GetName()+" says, '"+saying+"'")
	return true
}

// dragon_breath — mob spec: breath weapon in combat
func specDragonBreath(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosFighting || me.GetHP() < 0 {
		return false
	}
	melee := mobMeleeTarget(me)
	if melee == nil || randN(4) != 0 {
		return false
	}
	breathSpells := []int{
		spells.SpellFireBreath,
		spells.SpellGasBreath,
		spells.SpellFrostBreath,
		spells.SpellAcidBreath,
		spells.SpellLightningBreath,
	}
	breathNames := []string{"fire", "gas", "frost", "acid", "lightning"}
	n := randN(len(breathSpells))
	w.roomMessage(me.RoomVNum, me.GetName()+" breathes "+breathNames[n]+" at "+melee.GetName()+"!")
	spells.Cast(me, melee, breathSpells[n], me.GetLevel(), nil)
	return true
}

// citizen — mob spec: random greetings
func specCitizen(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || randN(8) != 0 {
		return false
	}
	citizenSayings := []string{
		"Don't speak to me.",
		"Piss off.",
		"Get out of my face.",
		"Nice day.",
		"Good weather we're having.",
		"Huh?  What?  I'm busy.",
		"I've got an axe to grind.",
		"Who are you?",
		"Get away from me!",
	}
	saying := citizenSayings[randN(len(citizenSayings))]
	w.roomMessage(me.RoomVNum, me.GetName()+" says, '"+saying+"'")
	return true
}

// cuchi — mob spec: random speech
func specCuchi(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || randN(4) != 0 {
		return false
	}
	cuchiSayings := []string{
		"I am not amused.",
		"You're all just jealous.",
		"*sigh* Nobody understands me.",
		"Minions of the universe, unite!",
		"I am the master of all I survey.",
		"Bow before me, mortals!",
		"Your insolence will be your undoing.",
	}
	saying := cuchiSayings[randN(len(cuchiSayings))]
	w.roomMessage(me.RoomVNum, me.GetName()+" says, '"+saying+"'")
	return true
}

// mini_thief — mob spec: steals small items
func specMiniThief(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetPosition() != combat.PosStanding || randN(3) != 0 {
		return false
	}
	for _, p := range w.GetPlayersInRoom(me.RoomVNum) {
		if !p.IsNPC() && randN(2) == 0 {
			stealAmt := randRange(1, 20)
			if p.Gold >= stealAmt {
				p.Gold -= stealAmt
				w.roomMessage(me.RoomVNum, me.GetName()+" snatches some coins and giggles!")
				sendToChar(p, "You notice your coin purse feels lighter...")
			}
			return true
		}
	}
	return false
}

// black_undead_knight — mob spec: taunts + hates red undead
func specBlackUndeadKnight(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.Fighting {
		switch randRange(1, 20) {
		case 1:
			w.roomMessage(me.RoomVNum, me.GetName()+" screams, 'Protect the kingdom!'")
		case 2:
			w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'If I'm going to hell, you're going with me!'")
		case 3:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'You dirty rotten scoundrel. I'm gonna make you very sorry.'")
		case 4:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'I know what you're thinking...'")
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Did he fire five shots, or did he fire six.'")
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Well let me ask you...'")
			w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'Do you feel lucky PUNK?  Well... DO YOU?'")
		case 5:
			w.roomMessage(me.RoomVNum, me.GetName()+" claims, 'I am the greatest!'")
		}
		return true
	}
	mobs := w.GetMobsInRoom(me.RoomVNum)
	for _, m := range mobs {
		if m.VNum == 11471 && m != me && randN(3) == 0 {
			w.roomMessage(me.RoomVNum, me.GetName()+" sees "+m.GetName()+" and gives a battle cry!")
			me.Fighting = true
			me.Target = m
			me.FightingTarget = m.GetName()
			return true
		}
	}
	return false
}

// red_undead_knight — mob spec: taunts + hates black undead
func specRedUndeadKnight(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.Fighting {
		switch randRange(1, 20) {
		case 1:
			w.roomMessage(me.RoomVNum, me.GetName()+" screams, 'Protect the homeland!'")
		case 2:
			w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'If you think you have had a bad day before, watch this!'")
		case 3:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'Don't ever argue with the big dog,'")
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'cause the big dog is always right.'")
		case 4:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'There's more than one way to skin a cat:'")
			w.roomMessage(me.RoomVNum, me.GetName()+" continues: 'Way number 15 -- Krazy Glue and a toothbrush.'")
		case 5:
			w.roomMessage(me.RoomVNum, me.GetName()+" says, 'A friend with weed is a friend indeed.'")
		}
		return true
	}
	mobs := w.GetMobsInRoom(me.RoomVNum)
	for _, m := range mobs {
		if m.VNum == 11470 && m != me && randN(3) == 0 {
			w.roomMessage(me.RoomVNum, me.GetName()+" sees "+m.GetName()+" and gives a battle cry!")
			me.Fighting = true
			me.Target = m
			me.FightingTarget = m.GetName()
			return true
		}
	}
	return false
}

// mickey — mob spec: harasses and attacks (from Natural Born Killers)
func specMickey(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.Fighting {
		if me.GetPosition() > combat.PosSleeping && me.GetPosition() < combat.PosFighting {
			w.roomMessage(me.RoomVNum, me.GetName()+" stands up!")
		} else {
			switch randRange(1, 10) {
			case 1:
				w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'I'll always love you Mal, no matter what!'")
			case 2:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'Do you believe in fate?'")
			case 3:
				w.roomMessage(me.RoomVNum, me.GetName()+" says, 'You're not centered.'")
			case 4:
				w.roomMessage(me.RoomVNum, me.GetName()+" shouts, 'When they come and ask you who did this, tell them it was Mickey and Mallory Knox!'")
			case 5:
				w.roomMessage(me.RoomVNum, me.GetName()+" states, 'It's not nice to point.'")
			}
		}
		return true
	}
	return false
}

// mallory — mob spec: barks + calls mickey for revenge
func specMallory(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" || me.GetHP() < 0 {
		return false
	}
	if me.Fighting {
		if me.GetPosition() > combat.PosSleeping && me.GetPosition() < combat.PosFighting {
			w.roomMessage(me.RoomVNum, me.GetName()+" stands up!")
		} else {
			switch randRange(1, 10) {
			case 1:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'How do you like me now?'")
			case 2:
				w.roomMessage(me.RoomVNum, me.GetName()+" says, 'That was the worst head I've ever got.'")
			case 3:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'How sexy am I now, fucker?'")
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'How sexy am I NOW?'")
			case 5:
				w.roomMessage(me.RoomVNum, me.GetName()+" asks, 'Did you bring enough for everybody?  Here... try one...'")
			}
		}
		return true
	}
	return false
}

// ================================================================
// Initialization — registers all spec procs into SpecRegistry
// ================================================================

// WAVE 4a: remaining functions from spec_procs.c will be added in Wave 5

func init() {
	RegisterSpec("guild", specGuild)
	RegisterSpec("dump", specDump)
	RegisterSpec("snake", specSnake)
	RegisterSpec("summoner", specSummoner)
	RegisterSpec("thief", specThief)
	RegisterSpec("magic_user", specMagicUser)
	RegisterSpec("fighter", specFighter)
	RegisterSpec("paladin", specPaladin)
	RegisterSpec("guild_guard", specGuildGuard)
	RegisterSpec("puff", specPuff)
	RegisterSpec("fido", specFido)
	RegisterSpec("janitor", specJanitor)
	RegisterSpec("cityguard", specCityguard)
	RegisterSpec("mayor", specMayor)
	RegisterSpec("dragon_breath", specDragonBreath)
	RegisterSpec("citizen", specCitizen)
	RegisterSpec("cuchi", specCuchi)
	RegisterSpec("mini_thief", specMiniThief)
	RegisterSpec("black_undead_knight", specBlackUndeadKnight)
	RegisterSpec("red_undead_knight", specRedUndeadKnight)
	RegisterSpec("mickey", specMickey)
	RegisterSpec("mallory", specMallory)
}
