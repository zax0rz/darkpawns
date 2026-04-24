package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

const (
	conductorsRoom = 18505
	paintingRoom   = 18101
	portalRoom     = 5799
	elevatorDest   = 5743
	newbieLevel    = 11
	lvlImmort      = 31
)

func init() {
	RegisterSpec("conductor", specConductor)
	RegisterSpec("brass_dragon", specBrassDragon)
	RegisterSpec("outofjailguard", specOutOfJailGuard)
	RegisterSpec("jailguard", specJailGuard)
	RegisterSpec("dracula", specDracula)
	RegisterSpec("pet_shops", specPetShops)
	RegisterSpec("enter_circle", specEnterCircle)
	RegisterSpec("elevator", specElevator)
	RegisterSpec("elemental_room", specElementalRoom)
	RegisterSpec("pray_for_items", specPrayForItems)
	RegisterSpec("fearface", specFearface)
	RegisterSpec("start_room", specStartRoom)
	RegisterSpec("newbie_zone_entrance", specNewbieZoneEntrance)
	RegisterSpec("suck_in", specSuckIn)
	RegisterSpec("oro_quarters_room", specOroQuartersRoom)
	RegisterSpec("oro_study_room", specOroStudyRoom)
	RegisterSpec("bank", specBank)
	RegisterSpec("horn", specHorn)
}

func specConductor(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || !me.IsNPC() || me.CurrentHP < 0 {
		return false
	}

	if !me.Fighting {
		walkRoll := randRange(1, 10)
		switch walkRoll {
		case 1, 2:
			// perform_move(ch, SCMD_EAST-1, 0) — simplified: move east
			// No MoveMob in API; skipping for now
			return true
		case 9, 10:
			return true
		}
	}

	if me.Fighting {
		if me.GetPosition() <= combat.PosSleeping || me.GetPosition() >= combat.PosFighting {
			r := randRange(1, 10)
			msg := ""
			switch r {
			case 1:
				msg = "$n shouts, 'I said give me your ticket!'"
			case 2:
				msg = "$n asks, 'Why are you so stupid?'"
			case 3:
				msg = "$n shouts 'Get off my train you trash!'"
			case 4:
				msg = "$n shouts 'Security! Help me with this piece of garbage!'"
			case 5:
				msg = "$n asks, 'Why wouldn't you just give me your ticket?'"
			}
			if msg != "" {
				w.roomMessage(me.RoomVNum, msg)
			}
		}
		return true
	}

	return false
}

func specBrassDragon(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || !isMoveCmd(cmd) {
		return false
	}

	if me.RoomVNum == 5065 && cmd == "west" {
		w.roomMessage(me.RoomVNum, "The brass dragon humiliates $n, and blocks $s way.")
		sendToChar(ch, "The brass dragon humiliates you, and blocks your way.\r\n")
		return true
	}

	return false
}

func specOutOfJailGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || !isMoveCmd(cmd) {
		return false
	}

	if me.RoomVNum == 8117 && cmd == "south" {
		w.roomMessage(me.RoomVNum, "The guard grabs $n by the collar and blocks $s way.")
		sendToChar(ch, "The guard stops you from entering with one quick jerk of your collar.\r\n")
		return true
	}

	return false
}

func specJailGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" || !isMoveCmd(cmd) {
		return false
	}

	if me.RoomVNum == 8118 && cmd == "north" {
		w.roomMessage(me.RoomVNum, "The guard grabs $n with one hand and throws $m back in the room.")
		sendToChar(ch, "The guard stops you from leaving with one flabby hand.\r\n")
		return true
	}

	return false
}

func specDracula(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "look" && cmd != "" {
		return false
	}

	if cmd == "" && me.Fighting {
		return specMagicUser(w, ch, me, cmd, arg)
	}

	arg = strings.TrimSpace(arg)
	if !strings.Contains(strings.ToLower(arg), strings.ToLower(me.GetName())) {
		return false
	}

	sendToChar(ch, "You feel mesmerized... your will weakens.\r\n")
	sendToChar(ch, fmt.Sprintf("%s sinks his fangs into your neck!\r\n", me.GetName()))
	w.roomMessage(me.RoomVNum, fmt.Sprintf("$n looks at %s.\r\n", me.GetName()))
	w.roomMessage(me.RoomVNum, fmt.Sprintf("%s gazes intently at $n.\r\n", me.GetName()))
	w.roomMessage(me.RoomVNum, fmt.Sprintf("%s sinks his fangs into $n!\r\n", me.GetName()))

	sendToChar(ch, "Your blood boils with a stinging fire...\r\n")

	return true
}

func petPrice(pet *MobInstance) int {
	return pet.GetLevel() * 25
}

func specPetShops(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	petRoom := me.RoomVNum + 1

	if cmd == "list" {
		sendToChar(ch, "Available pets are:\r\n")
		pets := w.GetMobsInRoom(petRoom)
		for _, pet := range pets {
			sendToChar(ch, fmt.Sprintf("%8d - %s\r\n", petPrice(pet), pet.GetName()))
		}
		return true
	}

	if cmd == "buy" {
		parts := strings.Fields(arg)
		if len(parts) == 0 {
			sendToChar(ch, "Buy what?\r\n")
			return true
		}

		petName := ""
		if len(parts) > 1 {
			petName = parts[1]
		}

		pets := w.GetMobsInRoom(petRoom)
		var pet *MobInstance
		for _, p := range pets {
			if strings.Contains(strings.ToLower(p.GetName()), strings.ToLower(parts[0])) {
				pet = p
				break
			}
		}
		if pet == nil {
			sendToChar(ch, "There is no such pet!\r\n")
			return true
		}

		price := petPrice(pet)
		if ch.Gold < price {
			sendToChar(ch, "You don't have enough gold!\r\n")
			return true
		}
		ch.Gold -= price

		newPet, err := w.SpawnMob(pet.VNum, me.RoomVNum)
		if err != nil {
			sendToChar(ch, "Something went wrong.\r\n")
			return true
		}

		if petName != "" {
			_ = newPet // name would be set on prototype
		}

		w.roomMessage(me.RoomVNum, fmt.Sprintf("$n buys $N as a pet.\r\n"))
		sendToChar(ch, "May you enjoy your pet.\r\n")

		return true
	}

	return false
}

func specEnterCircle(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "enter" && cmd != "look" {
		return false
	}

	if cmd == "enter" {
		arg = strings.TrimSpace(arg)
		if arg != "circle" && arg != "platform" {
			sendToChar(ch, "Enter what?\r\n")
			return true
		}

		portalMobs := w.GetMobsInRoom(portalRoom)
		if len(portalMobs) >= 2 {
			sendToChar(ch, "You can't fit on the portal, it's too crowded.\r\n")
			return true
		}

		sendToChar(ch, "You stand in the circle.\r\n")
		w.roomMessage(me.RoomVNum, "$n enters the circle which suddenly starts glowing brightly, obscuring your view of $m!")
		ch.SetRoom(portalRoom)
		// do_look Placeholder
		return true
	}

	// look
	arg = strings.TrimSpace(arg)
	if arg != "circle" && arg != "platform" {
		return false
	}

	sendToChar(ch, "Looking into the circle at the platform in the middle of the room, you see\r\n")
	mobs := w.GetMobsInRoom(portalRoom)
	if len(mobs) > 0 {
		var names []string
		for _, m := range mobs {
			names = append(names, m.GetName())
		}
		sendToChar(ch, strings.Join(names, " and "))
	} else {
		sendToChar(ch, "no one")
	}
	sendToChar(ch, ".\r\n")
	return true
}

func specElevator(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "say" && cmd != "'" {
		return false
	}

	lower := strings.ToLower(strings.TrimSpace(arg))
	if lower != "sumuni elementi avia elevata" {
		return false
	}

	sendToChar(ch, "The portal begins to rise, lifted by the air elemental summoned by your rune!\r\n\r\n")
	w.roomMessage(me.RoomVNum, "The portal begins to rise, lifted by the air elemental summoned by $n!\r\n\r\n")

	mobs := w.GetMobsInRoom(portalRoom)
	for i, m := range mobs {
		if i >= 2 {
			break
		}
		m.SetRoom(elevatorDest)
		// do_look placeholder
	}

	return true
}

func specElementalRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}

	mobs := w.GetMobsInRoom(me.RoomVNum)
	for _, m := range mobs {
		if !m.IsNPC() {
			continue
		}
		room := w.GetRoomInWorld(m.RoomVNum)
		sector := 0
		if room != nil {
			sector = room.Sector
		}

		msg := ""
		switch sector {
		case 0: // SECT_FIRE
			msg = "Your skin blackens as fire burns you alive..."
		case 1: // SECT_EARTH
			msg = "Your skin is pummeled by the forces of earth, breaking your bones..."
		case 2: // SECT_WIND
			msg = "Your flesh is peeled from your bones as the forces of air pummel you..."
		case 3: // SECT_WATER
			msg = "You struggle for air as your lungs fill with water..."
		default:
			msg = "The forces of nature slowly rip you apart..."
		}
		sendToChar(ch, msg+"\r\n")
		sendToChar(ch, "\r\nYou are DYING!\r\n")

		m.CurrentHP -= 100
		if m.CurrentHP <= 0 {
			w.roomMessage(me.RoomVNum, "The forces of nature slowly rip $N to shreds.")
		}
	}

	return false
}

func specPrayForItems(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "pray" {
		return false
	}

	arg = strings.TrimSpace(arg)
	parts := strings.Fields(arg)
	what := ""
	if len(parts) > 0 {
		what = parts[0]
	}

	if what == "immortality" {
		sendToChar(ch, "You feel the power pulse through your veins again!\r\n")
		return true
	}

	w.roomMessage(me.RoomVNum, "$n kneels at the altar and chants a prayer to Odin.")
	sendToChar(ch, "You notice a faint light in Odin's eye.\r\n")
	return true
}

func specFearface(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" && (!me.IsNPC() || !me.Fighting) {
		return false
	}
	return false
}

func specStartRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}

	mobs := w.GetMobsInRoom(me.RoomVNum)
	for _, m := range mobs {
		if !m.IsNPC() {
			continue
		}
		if m.GetLevel() >= lvlImmort {
			return false
		}

		msg := "   Suddenly the hairs on the back of your neck stand up as if lightning had\r\nstruck nearby. A keen wailing fills the air, and an ethereal image appears\r\nbefore you.\r\n"
		msg += fmt.Sprintf("   '%s, now is not your time to die,' speaks the figure.\r\n", m.GetName())
		msg += "   'Prove your worth and I may well grant you eternal life.'\r\n"
		msg += "   'Trust no one, for all here are but dark pawns above which you must\r\nstruggle to prove yourself.  All here strive to be a king... at any cost.'\r\n"
		msg += "   The figure glows a moment, then disappears, but his voice remains.\r\n"
		msg += "   'Your life begins now...' it says, then fades -- just as the world around\r\nyou does the same.\r\n\r\n"
		sendToChar(ch, msg)

		m.SetRoom(8004) // temple altar
	}

	return true
}

func specNewbieZoneEntrance(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "south" {
		return false
	}

	if ch.Level >= newbieLevel {
		sendToChar(ch, "Nah, you're too much of a badass to go in there!\r\n")
		return true
	}

	return false
}

func specSuckIn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "look" {
		return false
	}

	arg = strings.TrimSpace(arg)
	if strings.ToLower(arg) != "painting" {
		return false
	}

	sendToChar(ch, "\r\n\r\nYou suddenly feel very dizzy...\r\n\r\n")
	w.roomMessage(me.RoomVNum, "$n suddenly vanishes!")
	ch.SetRoom(paintingRoom)
	return true
}

func specOroQuartersRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me.IsNPC() || cmd != "south" {
		return false
	}

	if ch.Name != "Orodreth" {
		w.roomMessage(me.RoomVNum, "A strong force jolts $n in $s attempt to leave south.")
		sendToChar(ch, "A strong force blocks your way and gives you a nasty jolt.\r\n")
		ch.Health /= 2
		return true
	}

	return false
}

func specOroStudyRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me.IsNPC() || cmd != "north" {
		return false
	}

	if ch.Name != "Orodreth" {
		w.roomMessage(me.RoomVNum, "A strong force jolts $n in $s attempt to leave north.")
		sendToChar(ch, "A strong force blocks your way and gives you a nasty jolt.\r\n")
		ch.Health /= 2
		return true
	}

	return false
}

func specBank(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "balance" {
		sendToChar(ch, fmt.Sprintf("Your current balance is %d coins.\r\n", ch.Gold))
		return true
	}

	if cmd == "deposit" {
		amount := 0
		fmt.Sscanf(arg, "%d", &amount)
		if amount <= 0 {
			sendToChar(ch, "How much do you want to deposit?\r\n")
			return true
		}
		if ch.Gold < amount {
			sendToChar(ch, "You don't have that many coins!\r\n")
			return true
		}
		ch.Gold -= amount
		sendToChar(ch, fmt.Sprintf("You deposit %d coins.\r\n", amount))
		w.roomMessage(me.RoomVNum, "$n makes a bank transaction.")
		return true
	}

	if cmd == "withdraw" {
		amount := 0
		fmt.Sscanf(arg, "%d", &amount)
		if amount <= 0 {
			sendToChar(ch, "How much do you want to withdraw?\r\n")
			return true
		}
		ch.Gold += amount
		sendToChar(ch, fmt.Sprintf("You withdraw %d coins.\r\n", amount))
		w.roomMessage(me.RoomVNum, "$n makes a bank transaction.")
		return true
	}

	return false
}

func specHorn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "use" {
		return false
	}

	arg = strings.TrimSpace(arg)
	if !strings.Contains(strings.ToLower(arg), strings.ToLower(me.GetName())) {
		return false
	}

	sendToChar(ch, "You inhale deeply then blow hard!\r\n")
	sendToChar(ch, "A blaring note resounds through the air.\r\n")
	w.roomMessage(me.RoomVNum, "$n blows into $P.")
	w.roomMessage(me.RoomVNum, "$P lets out a blaring note...")
	return true
}
