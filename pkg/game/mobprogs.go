// mobprogs.go — Ported from src/mobprog.c
//
// Mobile program triggers: greeting, bribery, sound, rescue, and town-citizen
// helpers for NPC behavior.

package game

import (
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// IS_* checks from mobprog.h
// ---------------------------------------------------------------------------

func isDog(mob *MobInstance) bool        { v := mob.GetVNum(); return v == 8063 || v == 8065 }
func isJanitor(mob *MobInstance) bool    { return mob.GetVNum() == 8061 }
func isDemon(mob *MobInstance) bool      { return mob.GetVNum() == 14401 }
func isMercenary(mob *MobInstance) bool  { return mob.GetVNum() == 3063 }

func isWhore(mob *MobInstance) bool {
	specName, ok := MobSpecAssign[mob.GetVNum()]
	return ok && specName == "prostitute"
}

// ---------------------------------------------------------------------------
// MpGreet — port of mp_greet()
// ---------------------------------------------------------------------------

func (w *World) MpGreet(who *Player, room int) {
	if room < 0 {
		return
	}
	mob := w.GetFirstMobInRoomByVNum(room, 8014)
	if mob == nil {
		return
	}
	if who.GetLevel() < 10 || !hasPlrFlag(who, "calibrate") {
		return
	}
	clanID := who.ClanID
	hasAccess := clanID == 1 || clanID == 2 || clanID == 3
	if !hasAccess {
		w.roomMessage(room, "$n says, 'You are not of the Inner Circle.  Be gone!'")
		w.MovePlayerToRoom(who, 8117)
		return
	}
	w.roomMessage(room, "$n says, 'You may pass, friend.'")
}

// ---------------------------------------------------------------------------
// MpRideGreet — port of mp_ride_greet()
// ---------------------------------------------------------------------------

func (w *World) MpRideGreet(who *Player, room int) {
	if room < 0 {
		return
	}
	mob := w.GetFirstMobInRoomByVNum(room, 8014)
	if mob == nil {
		return
	}
	if who.GetLevel() < 10 || !hasPlrFlag(who, "calibrate") {
		return
	}
	clanID := who.ClanID
	hasAccess := clanID == 1 || clanID == 2 || clanID == 3
	if hasAccess {
		w.roomMessage(room, "$n says, 'Welcome to the Emporium.'")
	} else {
		w.roomMessage(room, "$n says, 'Now get outta here!'")
		w.MovePlayerToRoom(who, 8117)
		if mount := w.GetMount(who); mount != nil {
			w.MovePlayerToRoom(mount, 8117)
		}
	}
}

// ---------------------------------------------------------------------------
// MpGive / MpBribe — port of mp_give()
// ---------------------------------------------------------------------------

func (w *World) MpGive(mob *MobInstance, ch *Player, amount int) {
	if mob == nil {
		return
	}
	vnum := mob.GetVNum()

	switch {
	case vnum == 8014:
		w.roomMessage(mob.GetRoom(), "$n says, 'Now get outta here!'")
		ch.SendMessage("$N throws you out of the cell!\r\n")
		ch.Gold -= amount
		if ch.Gold < 0 {
			ch.Gold = 0
		}
		w.MovePlayerToRoom(ch, 8117)
		if mount := w.GetMount(ch); mount != nil {
			w.MovePlayerToRoom(mount, 8117)
		}

	case w.isCityguard(mob):
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(3) != 0 || amount < 200 {
			w.roomMessage(mob.GetRoom(), "$n says, 'Are you trying to bribe me?  That's against the law you know...'")
			w.StartRoomCombat(mob, ch)
		} else {
			w.roomMessage(mob.GetRoom(), "$n glances around warily and says, 'I am off duty now...'")
			w.roomMessage(mob.GetRoom(), "$n lays down and falls asleep on the job!")
			mob.SetStatus("sleeping")
			ch.Gold -= amount
			if ch.Gold < 0 {
				ch.Gold = 0
			}
		}

	case w.isCitizen(mob):
		w.roomMessage(mob.GetRoom(), "$n says, 'Thanks for the investment, I appreciate it.'")
		w.roomMessage(mob.GetRoom(), "$n bows graciously.")

	case isWhore(mob):
		ch.SendMessage("$n pulls you into the shadows, and gives you a lot more than you\r\nexpected for your money.\r\n")
		ch.SendMessage("A few coins lighter and quite a bit happier, you continue on your way.\r\n")

	case vnum == 13108:
		if amount >= 1000 {
			w.roomMessage(mob.GetRoom(), "$n leads $N through a hidden door.")
			w.MovePlayerToRoom(ch, 13154)
			if mount := w.GetMount(ch); mount != nil {
				w.MovePlayerToRoom(mount, 13154)
			}
			ch.SendMessage("  You follow the gremlin through a series of tunnels, full of twists, turns,\r\nand circles. Slowly you become aware of a tiny crack of light coming through\r\nthe top of the cavern. Suddenly the gremlin turns around and hurries away\r\nbefore you can even turn around.\r\n\r\n")
			w.LookAtRoom(ch, false)
		} else {
			w.roomMessage(mob.GetRoom(), "$n exclaims, 'Cheap bastard, come back with some money!'")
			w.roomMessage(mob.GetRoom(), "$n kicks $N out on $S ass!")
			w.MovePlayerToRoom(ch, 13193)
			w.LookAtRoom(ch, false)
		}
	}
}

// MpBribe is an alias for MpGive.
func (w *World) MpBribe(mob *MobInstance, ch *Player, amount int) {
	w.MpGive(mob, ch, amount)
}

// ---------------------------------------------------------------------------
// EntryProg — port of entry_prog()
// ---------------------------------------------------------------------------

func (w *World) EntryProg(mob *MobInstance, room int) {
	if room < 0 || mob.GetVNum() != 8059 {
		return
	}
	for _, tch := range w.GetMobsInRoom(room) {
		if tch == mob {
			continue
		}
		switch {
		case w.isCityguard(tch):
			if tch.GetPosition() < PosStanding {
				w.roomMessage(room, "$n barks 'On your feet, slacker!'")
				w.roomMessage(room, "$n wakes up and quickly snaps to attention!")
				w.roomMessage(room, "$n growls, 'Report to my office at 0500 tomorrow morning.'")
				tch.SetStatus("standing")
			} else {
				w.roomMessage(room, "$n snaps to attention and salutes!")
				w.roomMessage(room, "$n growls, 'At ease, soldier.'")
			}
			return
		case w.isCitizen(tch):
			w.roomMessage(room, "$n frowns at $N.")
			w.roomMessage(room, "$n says, 'Hail unto the True One, Captain.'")
			return
		}
	}
}

// ---------------------------------------------------------------------------
// isCitizen / isCityguard — port of is_citizen(), is_cityguard()
// ---------------------------------------------------------------------------

func (w *World) isCitizen(ch *MobInstance) bool {
	if ch == nil || !ch.IsNPC() {
		return false
	}
	switch ch.GetVNum() {
	case 2749, 2750, 8062, 18201, 18202, 21243:
		return true
	}
	return false
}

func (w *World) isCityguard(ch *MobInstance) bool {
	if ch == nil || !ch.IsNPC() {
		return false
	}
	specName, ok := MobSpecAssign[ch.GetVNum()]
	if ok && specName == "cityguard" {
		return true
	}
	switch ch.GetVNum() {
	case 2747, 8001, 8002, 8020, 8027, 8059, 8060, 12111, 21200, 21201, 21203, 21227, 21228:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// getBadGuy / killBadGuy — port of get_bad_guy() and kill_bad_guy()
// ---------------------------------------------------------------------------

func (w *World) getBadGuy(ch *MobInstance) *MobInstance {
	candidates := w.GetMobsInRoom(ch.GetRoom())
	var badGuys []*MobInstance
	for _, m := range candidates {
		targetName := m.GetFighting()
		if targetName == "" {
			continue
		}
		for _, roomMob := range candidates {
			if roomMob.GetName() == targetName && (w.isCitizen(roomMob) || w.isCityguard(roomMob)) {
				badGuys = append(badGuys, m)
				break
			}
		}
	}
	if len(badGuys) == 0 {
		return nil
	}
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	iVictim := rand.Intn(len(badGuys) + 1)
	if iVictim == 0 {
		return nil
	}
	return badGuys[iVictim-1]
}

func (w *World) killBadGuy(ch *MobInstance) bool {
	if ch.GetPosition() < 6 || ch.GetFighting() != "" {
		return false
	}
	opponent := w.getBadGuy(ch)
	if opponent == nil {
		return false
	}
	w.roomMessage(ch.GetRoom(), "$n roars: 'Protect the innocent!  BANZAIIII!  CHARGE!'")
	w.StartMobCombat(ch, opponent)
	return true
}

// ---------------------------------------------------------------------------
// NpcRescue — port of npc_rescue()
// ---------------------------------------------------------------------------

func (w *World) NpcRescue(chHero *MobInstance, chVictim *Player) bool {
	roomMobs := w.GetMobsInRoom(chHero.GetRoom())
	var chBadGuy *MobInstance
	for _, m := range roomMobs {
		if m.GetFighting() == chVictim.GetName() {
			chBadGuy = m
			break
		}
	}
	if chBadGuy == nil || chBadGuy == chHero {
		return false
	}

	chVictim.SendMessage("You are rescued by $N, your loyal friend!\r\n")
	w.roomMessage(chHero.GetRoom(), "$n heroically rescues $N.")

	if chBadGuy.GetFighting() != "" {
		chBadGuy.StopFighting()
	}
	if chHero.GetFighting() != "" {
		chHero.StopFighting()
	}
	chHero.SetFighting(chBadGuy.GetName())
	chBadGuy.SetFighting(chHero.GetName())
	return true
}

// ---------------------------------------------------------------------------
// isJunk / isShopkeeper — port of is_junk() and is_shopkeeper()
// ---------------------------------------------------------------------------

func isJunk(obj *ObjectInstance) bool {
	if obj == nil || obj.Prototype == nil {
		return false
	}
	return obj.IsTakeable() && (obj.IsDrinkContainer() || obj.GetCost() <= 10)
}

func (w *World) isShopkeeper(ch *MobInstance) bool {
	if ch == nil || !ch.IsNPC() {
		return false
	}
	specName, ok := MobSpecAssign[ch.GetVNum()]
	if ok {
		switch specName {
		case "shop_keeper", "guild", "guild_guard", "butler", "clerk":
			return true
		}
	}
	switch ch.GetVNum() {
	case 8003, 8004, 8005, 8006, 8007, 8008, 8009, 8010, 8011, 8078:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// MpSound — port of mp_sound()
// ---------------------------------------------------------------------------

func (w *World) MpSound(mob *MobInstance) {
	sound := ""
	useSay := false

	switch mob.GetVNum() {
	case 8066:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(2) == 0 {
			sound = "Sign this, please! There's too much violence!"
		} else {
			sound = "You look like a kind person.. sign this petition?"
		}
		useSay = true
	case 8067:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(2) == 0 {
			sound = "adjusts his tool belt."
		} else {
			sound = "wipes the sweat of labor from his brow."
		}
		useSay = false
	case 8068:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(2) == 0 {
			sound = "Arch Bishop Dinive to arrive on the Day of Winter Dawning!"
		} else {
			sound = "By mandate of the church, no violence in town! The penalty is jail time!"
		}
		useSay = true
	case 8069:
		sound = "Repent sinners! The end time is near!"
		useSay = true
	case 8071:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(2) == 0 {
			sound = "Spare a coin, buddy?"
			useSay = true
		} else {
			sound = "jingles his cup."
			useSay = false
		}
	case 8072:
		sound = "sings an old war ditty... badly off-key."
		useSay = false
	case 8074:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(2) == 0 {
			sound = "plays a lilting tune about your mother's beauty."
		} else {
			sound = "sings a melody about your conquests in battle."
		}
		useSay = false
	case 8079:
		sound = "tries to escape from an invisible box only he can see."
		useSay = false
	case 14202:
		sound = "tokes up on some kind bud."
		useSay = false
	case 8059:
		w.roomMessage(mob.GetRoom(), "$n looks at you.")
		sound = "Carry on, citizen."
		useSay = true
	case 8023:
		sound = "jiggles in your direction."
		useSay = false
	case 16300:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		if rand.Intn(2) == 0 {
			sound = "smiles at you."
		} else {
			sound = "shuffles some papers around on his desk."
		}
		useSay = false
	default:
		return
	}

	if useSay {
		w.roomMessage(mob.GetRoom(), fmt.Sprintf("$n says, '%s'", sound))
	} else {
		w.roomMessage(mob.GetRoom(), fmt.Sprintf("$n %s", sound))
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if isDemon(mob) && rand.Intn(3) != 0 {
		w.roomMessage(mob.GetRoom(), "$n says, 'I seek the dull blackened stones in which the souls of mortals have been trapped!'")
		w.roomMessage(mob.GetRoom(), "$n says, 'I shall open a portal to the Grey Fortress in exchange for a soul stone.'")
	}

	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if isDog(mob) && rand.Intn(26) == 0 {
		w.roomMessage(mob.GetRoom(), "$n relieves itself, nearly hitting your foot.")
		puddle := w.CreateObject(20, mob.GetRoom())
		if puddle != nil {
			puddle.SetTimer(2)
		}
	}
}

// ---------------------------------------------------------------------------
// Helper methods needed by mobprogs
// ---------------------------------------------------------------------------

// GetFirstMobInRoomByVNum returns the first mob in a room with the given VNum.
func (w *World) GetFirstMobInRoomByVNum(roomVNum int, vnum int) *MobInstance {
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if m.GetVNum() == vnum {
			return m
		}
	}
	return nil
}

// MovePlayerToRoom moves a player to a room by VNum.
func (w *World) MovePlayerToRoom(p *Player, vnum int) {
	p.SetRoom(vnum)
}

// GetMount returns the player's mount if they have one (stub).
func (w *World) GetMount(p *Player) *Player { return nil }

// LookAtRoom sends the room description to the player (stub).
func (w *World) LookAtRoom(p *Player, brief bool) {}

// CreateObject creates an object instance from a prototype and spawns it in a room.
func (w *World) CreateObject(vnum int, roomVNum int) *ObjectInstance {
	proto, ok := w.GetObjPrototype(vnum)
	if !ok {
		return nil
	}
	obj := NewObjectInstance(proto, roomVNum)
	if err := w.MoveObjectToRoom(obj, roomVNum); err != nil {
		slog.Warn("MoveObjectToRoom failed in CreateObject", "obj_vnum", obj.GetVNum(), "room", roomVNum, "error", err)
	}
	return obj
}

// StartRoomCombat initiates combat between a mob and a player.
// StartRoomCombat initiates combat between attacker and defender.
func (w *World) StartRoomCombat(attacker, defender combat.Combatant) {
	if aiCombatEngine != nil {
		if err := aiCombatEngine.StartCombat(attacker, defender); err != nil {
			slog.Warn("StartCombat failed in StartRoomCombat", "error", err)
		}
	}
}

// StartMobCombat initiates combat between two mobs.
func (w *World) StartMobCombat(attacker, defender *MobInstance) {
	if aiCombatEngine != nil {
		if err := aiCombatEngine.StartCombat(attacker, defender); err != nil {
			slog.Warn("StartCombat failed in StartMobCombat", "error", err)
		}
	}
}

// hasPlrFlag checks if a player has a named flag (stub).
func hasPlrFlag(p *Player, flag string) bool { return false }

// IsTakeable returns true if the object has ITEM_WEAR_TAKE flag.
func (o *ObjectInstance) IsTakeable() bool {
	if o.Prototype == nil || len(o.Prototype.WearFlags) == 0 {
		return false
	}
	// WearFlag 1 = ITEM_TAKE (from CircleMUD item wear bits)
	return o.Prototype.WearFlags[0]&1 != 0
}

// IsDrinkContainer returns true if the object is a drink container.
func (o *ObjectInstance) IsDrinkContainer() bool {
	if o.Prototype == nil {
		return false
	}
	return o.Prototype.TypeFlag == 9 // ITEM_DRINKCON = 9
}

