package game

import (
	"fmt"
	"log/slog"
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
	if cmd == "" || !me.IsNPC() || me.GetHP() < 0 {
		return false
	}

	if !me.IsFighting() {
		// C source: spec_procs.c:1618-1630 — conductor wanders east/west when idle.
		// Must be standing (awake) to move. Cases 1-2: east, 9-10: west.
		if me.GetPosition() > combat.PosStanding {
			return false
		}
		walkRoll := randRange(1, 10)
		switch walkRoll {
		case 1, 2:
			// C: perform_move(ch, SCMD_EAST-1, 0) → direction index 1 (east)
			w.mobPerformMove(me, 1)
			return true
		case 9, 10:
			// C: perform_move(ch, SCMD_WEST-1, 0) → direction index 3 (west)
			w.mobPerformMove(me, 3)
			return true
		}
	}

	if me.IsFighting() {
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
				w.roomMessage(me.GetRoom(), msg)
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

	if me.GetRoom() == 5065 && cmd == "west" {
		w.roomMessage(me.GetRoom(), "The brass dragon humiliates $n, and blocks $s way.")
		sendToChar(ch, "The brass dragon humiliates you, and blocks your way.\r\n")
		return true
	}

	return false
}

func specOutOfJailGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil || me == nil || cmd == "" || !isMoveCmd(cmd) {
		return false
	}

	// C: spec_procs.c:1765-1767 — only mortal, non-hunting movers reach the
	// room-specific guard. Players have no hunting state in this port, so the
	// world query is the faithful false result for the player case.
	if ch.GetLevel() >= lvlImmort || w.IsHunting(ch.GetName(), false) {
		return false
	}

	// C: ch->in_room, not the special mob's room. CMD_IS("south") is already
	// canonicalized by command dispatch; the move-command gate above rejects
	// all non-direction commands before this exact comparison.
	if ch.GetRoomVNum() == 8117 && cmd == "south" {
		Act(w, false, ch, nil, nil, nil,
			"The guard grabs $n by the collar and blocks $s way.", "", ToRoom)
		sendToChar(ch, "The guard stops you from entering with one quick jerk of your collar.")
		return true
	}

	return false
}

func specJailGuard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || ch == nil || me == nil || cmd == "" || !isMoveCmd(cmd) {
		return false
	}

	// C: spec_procs.c:1783-1785 — only mortal, non-hunting movers reach the
	// room-specific guard. Players have no hunting state in this port, so the
	// world query is the faithful false result for the player case.
	if ch.GetLevel() >= lvlImmort || w.IsHunting(ch.GetName(), false) {
		return false
	}

	// C: ch->in_room, not the special mob's room. CMD_IS("north") is already
	// canonicalized by command dispatch; the move-command gate above rejects
	// all non-direction commands before this exact comparison.
	if ch.GetRoomVNum() == 8118 && cmd == "north" {
		Act(w, false, ch, nil, nil, nil,
			"The guard grabs $n with one hand and throws $m back in the room.", "", ToRoom)
		sendToChar(ch, "The guard stops you from leaving with one flabby hand.")
		return true
	}

	return false
}

func specDracula(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if w == nil || me == nil {
		return false
	}
	if cmd != "look" && cmd != "" {
		return false
	}

	if cmd == "" {
		if me.IsFighting() {
			return specMagicUser(w, ch, me, cmd, arg)
		}
		return false
	}

	if ch == nil || ch.GetFlags()&(1<<uint(PrfNohassle)) != 0 {
		return false
	}
	arg = strings.TrimSpace(arg)
	if !isnameWithAbbrevs(arg, charKeywords(me)) {
		return false
	}

	sendToChar(ch, "You feel mesmerized... your will weakens.")
	sendToChar(ch, fmt.Sprintf("%s sinks his fangs into your neck!", me.GetName()))
	Act(w, false, ch, nil, nil, nil, fmt.Sprintf("$n looks at %s.\r\n", me.GetName()), "", ToRoom)
	Act(w, false, ch, nil, nil, nil, fmt.Sprintf("%s gazes intently at $n.\r\n", me.GetName()), "", ToRoom)
	Act(w, false, ch, nil, nil, nil, fmt.Sprintf("%s sinks his fangs into $n!\r\n", me.GetName()), "", ToRoom)
	w.DoSay(ch, "Now I know... The blood is the life!")
	if ch.GetFlags()&(1<<uint(PlrVampire)) == 0 && ch.GetFlags()&(1<<uint(PlrWerewolf)) == 0 {
		ch.SetPlrFlag(PlrVampire, true)
		sendToChar(ch, "Your blood boils with a stinging fire...")
	}

	return true
}

func petPrice(pet *MobInstance) int {
	return pet.GetLevel() * 25
}

func specPetShops(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me == nil {
		return false
	}
	petRoom := me.GetRoom() + 1

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
		if ch.GetGold() < price {
			sendToChar(ch, "You don't have enough gold!\r\n")
			return true
		}
		ch.SetGold(ch.GetGold() - price)

		newPet, err := w.SpawnMob(pet.GetVNum(), me.GetRoom())
		if err != nil {
			sendToChar(ch, "Something went wrong.\r\n")
			return true
		}

		if petName != "" {
			_ = newPet // name would be set on prototype
		}

		w.roomMessage(me.GetRoom(), "$n buys $N as a pet.\r\n")
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
		w.roomMessage(me.GetRoom(), "$n enters the circle which suddenly starts glowing brightly, obscuring your view of $m!")
		ch.SetRoom(portalRoom)
		w.doLook(ch, nil, "look", "")
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
	w.roomMessage(me.GetRoom(), "The portal begins to rise, lifted by the air elemental summoned by $n!\r\n\r\n")

	players := w.GetPlayersInRoom(portalRoom)
	for _, p := range players {
		if p != nil {
			p.SetRoom(elevatorDest)
			w.doLook(p, nil, "look", "")
		}
	}

	mobs := w.GetMobsInRoom(portalRoom)
	for i, m := range mobs {
		if i >= 2 {
			break
		}
		m.SetRoom(elevatorDest)
	}

	return true
}

func specElementalRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "" {
		return false
	}

	mobs := w.GetMobsInRoom(me.GetRoom())
	for _, m := range mobs {
		if !m.IsNPC() {
			continue
		}
		room := w.GetRoomInWorld(m.GetRoom())
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

		m.SetHealth(m.GetHP() - 100)
		if m.GetHP() <= 0 {
			w.roomMessage(me.GetRoom(), "The forces of nature slowly rip $N to shreds.")
			w.HandleDeath(m, nil, -1)
		}
	}

	return false
}

func specPrayForItems(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "pray" {
		return false
	}

	what, _ := oneArgument(arg)

	if what == "immortality" {
		level := 0
		switch ch.GetName() {
		case "Serapis", "Orodreth":
			level = 40
		case "Frontline":
			level = 39
		case "this is not here":
			// C evaluates these independent if statements in order; the
			// later level-31 assignment is the final value for this name.
			level = 31
		case "neither is this":
			level = 36
		case "no entry here", "neither here":
			level = 31
		}
		if level > 0 {
			ch.SetLevel(level)
			sendToChar(ch, "Welcome back "+ch.GetName()+".")
			sendToChar(ch, "You feel the power pulse through your veins again!")
		}
		// C's immortality branch returns TRUE even when the player's name
		// matches none of its hard-coded resurrection entries.
		return true
	}

	key := "item_for_" + ch.GetName()
	gold := 0
	found := false
	for _, tmpObj := range w.GetItemsInRoom(ch.GetRoomVNum()) {
		for _, extra := range tmpObj.GetExtraDescs() {
			if !strings.EqualFold(key, extra.Keywords) {
				continue
			}
			if gold == 0 {
				gold = 1
				w.actMessage(
					ch.GetRoomVNum(), ch, nil,
					"", "", ch.GetName()+" kneels and at the altar and chants a prayer to Odin.",
				)
				sendToChar(ch, "You notice a faint light in Odin's eye.")
			}

			obj, err := w.SpawnObject(tmpObj.GetVNum(), -1)
			if err != nil {
				slog.Error("pray_for_items failed to read object", "obj_vnum", tmpObj.GetVNum(), "error", err)
				continue
			}
			if err := w.MoveObjectToRoom(obj, ch.GetRoomVNum()); err != nil {
				slog.Error("pray_for_items failed to place object", "obj_vnum", tmpObj.GetVNum(), "room", ch.GetRoomVNum(), "error", err)
				w.ExtractObject(obj, ch.GetRoomVNum())
				continue
			}
			w.actMessage(
				ch.GetRoomVNum(), ch, nil,
				"", "", obj.GetShortDesc()+" slowly fades into existence.",
			)
			sendToChar(ch, obj.GetShortDesc()+" slowly fades into existence.")
			gold += obj.GetCost()
			found = true
		}
	}

	if found {
		if remaining := ch.GetGold() - gold; remaining > 0 {
			ch.SetGold(remaining)
		} else {
			ch.SetGold(0)
		}
		return true
	}

	// The C special returns FALSE here, allowing interpreter.c to dispatch
	// the ordinary pray social. In particular, me is nil for room specials.
	return false
}

func specFearface(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd == "" && (!me.IsNPC() || !me.IsFighting()) {
		return false
	}
	return false
}

func specStartRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	_ = me
	_ = cmd
	_ = arg
	if ch == nil {
		return false
	}

	players := w.GetPlayersInRoom(ch.GetRoomVNum())
	for _, player := range players {
		if player.GetLevel() >= lvlImmort {
			return false
		}
	}

	for _, player := range players {
		player.SendMessage(startRoomBirthMessage(player.GetName()))
		player.SetRoom(NewbieHometownRoom(player.GetHometown()))
		// The C start_room path calls do_look with the new mortal's default
		// PRF_AUTOEXIT state, which is off in the oracle vehicle.
		player.SetAutoExit(false)
		w.lookAtRoom(player, false)
	}

	return true
}

func startRoomBirthMessage(name string) string {
	// The C body builds the first three lines, then passes the same buffer as
	// both sprintf source and destination. The oracle's libc result drops
	// those lines; preserve the player-facing bytes observed on that path.
	msg := fmt.Sprintf("   '%s, now is not your time to die,' speaks the figure.\r\n", name)
	msg += "   'Prove your worth and I may well grant you eternal life.'\r\n"
	msg += "   'Trust no one, for all here are but dark pawns above which you must\r\nstruggle to prove yourself.  All here strive to be a king... at any cost.'\r\n"
	msg += "   The figure glows a moment, then disappears, but his voice remains.\r\n"
	msg += "   'Your life begins now...' it says, then fades -- just as the world around\r\nyou does the same.\r\n\r\n"
	return msg
}

func specNewbieZoneEntrance(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "south" {
		return false
	}

	if ch.GetLevel() >= newbieLevel && ch.GetLevel() < lvlImmort {
		sendToChar(ch, "Nah, you're too much of a badass to go in there!")
		return true
	}

	return false
}

func specSuckIn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "look" {
		return false
	}

	what, _ := oneArgument(arg)
	if what != "painting" {
		return false
	}

	// C calls do_look with the first one_argument token before emitting the
	// transition bytes. Room specials receive no mob receiver, so use ch as
	// the act() actor and let TO_ROOM exclude the actor and substitute $n.
	w.doLook(ch, me, "look", what)
	ch.SendMessage("\r\n\r\n\r\n\r\nYou suddenly feel very dizzy...\r\n\r\n")
	Act(w, false, ch, nil, nil, nil, "$n suddenly vanishes!", "", ToRoom)
	ch.SetRoom(paintingRoom)
	w.lookAtRoom(ch, true)
	return true
}

func specOroQuartersRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if me.IsNPC() || cmd != "south" {
		return false
	}

	if ch.Name != "Orodreth" {
		w.roomMessage(me.GetRoom(), "A strong force jolts $n in $s attempt to leave south.")
		sendToChar(ch, "A strong force blocks your way and gives you a nasty jolt.\r\n")
		ch.mu.Lock()
		ch.SetHP(ch.GetHP() / 2)
		ch.mu.Unlock()
		return true
	}

	return false
}

func specOroStudyRoom(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Room specials receive no mob receiver; C's IS_MOB(ch) gate is already
	// represented by the player-only handler boundary. A non-nil receiver is
	// retained as the equivalent NPC guard for focused direct-call tests.
	if (me != nil && me.IsNPC()) || cmd != "north" {
		return false
	}

	if ch.Name != "Orodreth" {
		Act(w, false, ch, nil, nil, nil,
			"A strong force jolts $n in $s attempt to leave north.", "", ToRoom)
		sendToChar(ch, "A strong force blocks your way and gives you a nasty jolt.")
		ch.SetHP(ch.GetHP() / 2)
		return true
	}

	return false
}

func specBank(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Banking moves coins between carried gold and the bank account. Ported from
	// src/spec_procs.c SPECIAL(bank). The previous Go port ignored BankGold:
	// balance showed carried gold, deposit destroyed coins, and withdraw minted
	// coins from nothing with no balance check (an economy exploit).
	if cmd == "balance" {
		if ch.GetBankGold() > 0 {
			sendToChar(ch, fmt.Sprintf("Your current balance is %d coins.\r\n", ch.GetBankGold()))
		} else {
			sendToChar(ch, "You currently have no money deposited.\r\n")
		}
		return true
	}

	if cmd == "deposit" {
		amount := 0
		if _, err := fmt.Sscanf(arg, "%d", &amount); err != nil {
			slog.Warn("fmt.Sscanf failed in deposit", "arg", arg, "error", err)
		}
		if amount <= 0 {
			sendToChar(ch, "How much do you want to deposit?\r\n")
			return true
		}
		if ch.GetGold() < amount {
			sendToChar(ch, "You don't have that many coins!\r\n")
			return true
		}
		ch.SetGold(ch.GetGold() - amount)
		ch.SetBankGold(ch.GetBankGold() + amount)
		sendToChar(ch, fmt.Sprintf("You deposit %d coins.\r\n", amount))
		w.roomMessage(me.GetRoom(), "$n makes a bank transaction.")
		return true
	}

	if cmd == "withdraw" {
		amount := 0
		if _, err := fmt.Sscanf(arg, "%d", &amount); err != nil {
			slog.Warn("fmt.Sscanf failed in withdraw", "arg", arg, "error", err)
		}
		if amount <= 0 {
			sendToChar(ch, "How much do you want to withdraw?\r\n")
			return true
		}
		if ch.GetBankGold() < amount {
			sendToChar(ch, "You don't have that many coins deposited!\r\n")
			return true
		}
		ch.SetGold(ch.GetGold() + amount)
		ch.SetBankGold(ch.GetBankGold() - amount)
		sendToChar(ch, fmt.Sprintf("You withdraw %d coins.\r\n", amount))
		w.roomMessage(me.GetRoom(), "$n makes a bank transaction.")
		return true
	}

	return false
}

func specHorn(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if cmd != "use" {
		return false
	}

	arg = strings.TrimSpace(arg)
	if !strings.Contains(strings.ToLower(arg), "horn") {
		return false
	}

	sendToChar(ch, "You inhale deeply then blow hard!\r\n")
	sendToChar(ch, "A blaring note resounds through the air.\r\n")
	w.roomMessage(ch.GetRoomVNum(), ch.GetName()+" blows into a horn.")
	w.roomMessage(ch.GetRoomVNum(), "A horn lets out a blaring note...")
	return true
}
