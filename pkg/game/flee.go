package game

import (
	"fmt"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func sendBodyMessage(w *World, body combat.Combatant, message string) {
	switch body := body.(type) {
	case *Player:
		body.SendMessage(message)
	case *MobInstance:
		if w != nil && w.MobileMessageSink != nil {
			w.MobileMessageSink(body, []byte(message))
		}
	}
}

func actForPrivateBody(world *World, hideInvisible bool, ch, vict Actor, obj, victObj *ObjectInstance, format, arg2 string, actType int, body combat.Combatant) {
	mob, ok := body.(*MobInstance)
	if !ok || world == nil || world.MobileMessageSink == nil {
		Act(world, hideInvisible, ch, vict, obj, victObj, format, arg2, actType)
		return
	}
	ActWithPrivateSink(world, hideInvisible, ch, vict, obj, victObj, format, arg2, actType, mob, func(msg string) {
		world.MobileMessageSink(mob, []byte(msg))
	})
}

// DoFlee is the direct do_flee() call used by skills and combat callbacks. The
// command parser is not involved, so this path does not consume command-entry
// draws or inspect an argument.
func (w *World) DoFlee(ch combat.Combatant) {
	if ch == nil {
		return
	}
	var player *Player
	var mob *MobInstance
	switch body := ch.(type) {
	case *Player:
		player = body
	case *MobInstance:
		mob = body
	default:
		return
	}

	var loss int
	if opponent := ch.GetFightingBody(); opponent != nil {
		loss = (opponent.GetMaxHP() - opponent.GetHP()) * opponent.GetLevel()
	}
	if ch.GetPosition() < combat.PosFighting {
		sendBodyMessage(w, ch, "Get on your feet first!\r\n")
		return
	}
	if mob != nil && mob.GetWaitState() != 0 {
		sendBodyMessage(w, mob, "You can't flee yet!\r\n")
		return
	}
	if player != nil && (player.GetClass() == ClassThief || player.GetClass() == ClassAssassin) && player.GetWaitState() > 1 {
		sendBodyMessage(w, player, "You attempt to flee but cannot!\r\n")
		return
	}

	for range len(dirs) {
		// C uses six independent draws and permits repeated directions.
		// #nosec G404 — game RNG, not cryptographic
		dir := dprng.Number(0, len(dirs)-1)
		room := w.GetRoomInWorld(ch.GetRoom())
		if room == nil {
			return
		}
		exit, ok := room.Exits[dirs[dir]]
		if !ok || exit.ToRoom == -1 || exit.ExitInfo&parser.ExitClosed != 0 {
			continue
		}
		destination := w.GetRoomInWorld(exit.ToRoom)
		if destination == nil || movementRoomHasFlag(destination, roomFlagDeath, "death") {
			continue
		}

		Act(w, true, ch.(Actor), nil, nil, nil, "$n panics, and attempts to flee!", "", ToRoom)
		moved := false
		if player != nil {
			moved = w.DoFleeMove(player, dirs[dir])
		} else {
			moved = w.doFleeMoveMob(mob, dir)
		}
		if !moved {
			Act(w, true, ch.(Actor), nil, nil, nil, "$n tries to flee, but can't!", "", ToRoom)
			return
		}
		sendBodyMessage(w, ch, "You flee head over heels.\r\n")
		if player != nil {
			if player.GetLevel() > 10 {
				loss += int(500 * (float64(player.GetLevel()) / 2.6))
			}
			if loss > 0 {
				player.LoseExp(loss)
			}
		}
		return
	}
	sendBodyMessage(w, ch, "PANIC!  You couldn't escape!\r\n")
}

// doFleeMoveMob executes the do_simple_move leg for an NPC that flees through
// a direct C call. It keeps the movement text and state changes on the normal
// movement path; the NPC does not use the player command dispatcher.
func (w *World) doFleeMoveMob(ch *MobInstance, dir int) bool {
	if ch == nil || dir < 0 || dir >= len(dirs) {
		return false
	}
	direction := dirs[dir]
	if w.movementSpecialBlocksMob(ch, direction) {
		return false
	}
	if ch.IsAffected(affCharm) {
		if master := w.combatFollowingBody(ch); master != nil && master.GetRoom() == ch.GetRoom() {
			sendBodyMessage(w, ch, "The thought of leaving your master makes you weep.\r\n")
			return false
		}
	}

	fromRoom := w.GetRoomInWorld(ch.GetRoom())
	if fromRoom == nil {
		return false
	}
	exit, ok := fromRoom.Exits[direction]
	if !ok || exit.ToRoom == -1 || exit.ExitInfo&parser.ExitClosed != 0 {
		return false
	}
	toRoom := w.GetRoomInWorld(exit.ToRoom)
	if toRoom == nil {
		return false
	}
	if (fromRoom.Sector == SECT_WATER_NOSWIM || toRoom.Sector == SECT_WATER_NOSWIM) && !mobHasBoat(ch) {
		sendBodyMessage(w, ch, "You need a boat to go there.\r\n")
		return false
	}
	if fromRoom.Sector < 0 || fromRoom.Sector >= len(movementLoss) || toRoom.Sector < 0 || toRoom.Sector >= len(movementLoss) {
		return false
	}
	needMovement := (movementLoss[fromRoom.Sector] + movementLoss[toRoom.Sector]) >> 1
	if ch.IsAffected(affMounted) {
		rider := w.GetRider(ch)
		if rider == nil {
			ch.RemoveAffected(affMounted)
			return false
		}
		if ch.GetPosition() < combat.PosStanding {
			sendBodyMessage(w, ch, "You're in no position to go ANYWHERE!\r\n")
			return false
		}
		if ch.GetMove() < needMovement {
			sendBodyMessage(w, ch, "You are too exhausted to carry your rider further.\r\n")
			rider.SendMessage("Your mount is too exhausted to carry you further.\r\n")
			return false
		}
	}
	if movementRoomHasFlag(toRoom, roomFlagTunnel, "tunnel") && len(w.GetPlayersInRoom(toRoom.VNum)) >= 1 {
		sendBodyMessage(w, ch, "There isn't enough room there!\r\n")
		return false
	}
	if ch.IsAffected(affMounted) && movementRoomHasFlag(toRoom, roomFlagIndoors, "indoors") {
		sendBodyMessage(w, ch, "You can't ride in there! Dismount first!\r\n")
		return false
	}

	wasIn := ch.GetRoom()
	if !ch.IsAffected(affSneak) {
		Act(w, true, ch, nil, nil, nil, fmt.Sprintf("$n leaves %s.", direction), "", ToRoom)
	}
	if err := w.MobTransfer(ch, exit.ToRoom); err != nil {
		slog.Error("mobile flee transfer failed", "mob", ch.GetName(), "from", wasIn, "to", exit.ToRoom, "error", err)
		return false
	}

	if !ch.IsAffected(affSneak) {
		switch dir {
		case 0, 1, 2, 3:
			arrival := fmt.Sprintf("$n arrives from the %s.", dirs[revDir[dir]])
			if ch.IsAffected(affFly) {
				arrival = fmt.Sprintf("$n flies in from the %s.", dirs[revDir[dir]])
			} else if toRoom.Sector == SECT_UNDERWATER {
				arrival = fmt.Sprintf("$n swims in from the %s.", dirs[revDir[dir]])
			}
			Act(w, true, ch, nil, nil, nil, arrival, "", ToRoom)
		case 4:
			arrival := "$n climbs in from below."
			if ch.IsAffected(affFly) {
				arrival = "$n flies in from below."
			} else if toRoom.Sector == SECT_UNDERWATER {
				arrival = "$n swims in from below."
			}
			Act(w, true, ch, nil, nil, nil, arrival, "", ToRoom)
		case 5:
			arrival := "$n climbs in from above."
			if ch.IsAffected(affFly) {
				arrival = "$n flies in from above."
			} else if toRoom.Sector == SECT_UNDERWATER {
				arrival = "$n swims in from above."
			}
			Act(w, true, ch, nil, nil, nil, arrival, "", ToRoom)
		}
	}

	w.EntryProg(ch, ch.GetRoom())
	if ch.GetRoom() < 0 {
		return false
	}
	w.MpGreetActor(ch, ch.GetRoom())
	if ch.GetRoom() < 0 {
		return false
	}
	w.runMobGreetScripts(ch, ch.GetRoom())
	if ch.GetRoom() < 0 {
		return false
	}
	if room := w.GetRoomInWorld(ch.GetRoom()); movementRoomHasFlag(room, roomFlagDeath, "death") {
		if room != nil {
			LogDeathTrap(ch.GetName(), room.VNum, room.Name)
		}
		combat.DeathCry(ch)
		w.ExtractMob(ch)
		return false
	}
	w.RunRoomEnterScriptForActor(ch)
	return true
}

func mobHasBoat(ch *MobInstance) bool {
	if ch.GetLevel() >= lvlImmort || ch.IsAffected(affWaterWalk) || ch.IsAffected(affFly) {
		return true
	}
	for _, obj := range ch.InventorySnapshot() {
		if obj != nil && obj.GetTypeFlag() == ITEM_BOAT {
			return true
		}
	}
	for _, obj := range ch.EquipmentSnapshot() {
		if obj != nil && obj.GetTypeFlag() == ITEM_BOAT {
			return true
		}
	}
	return false
}

// movementSpecialBlocksMob invokes the actor-aware native and object special
// lanes used by special(ch, dir+1, ""). An unsupported registered handler
// blocks this move so the port does not silently claim parity for an unported
// legacy-only NPC special lane.
func (w *World) movementSpecialBlocksMob(ch *MobInstance, direction string) bool {
	ctx := CharacterSpecialContext{Actor: ch}
	room := ch.GetRoom()
	if name, ok := GetRoomSpecName(room); ok {
		if DispatchCharacterSpecial(name, w, ctx, nil, direction, "") != SpecialDeclined {
			return true
		}
	}
	for _, obj := range COrderedMobWorn(ch) {
		if obj == nil {
			continue
		}
		if name, ok := GetObjSpecName(obj.VNum); ok {
			if DispatchCharacterObjectSpecial(name, w, ctx, obj, direction, "") != SpecialDeclined {
				return true
			}
		}
	}
	for _, obj := range ch.InventorySnapshot() {
		if obj == nil {
			continue
		}
		if name, ok := GetObjSpecName(obj.VNum); ok {
			if DispatchCharacterObjectSpecial(name, w, ctx, obj, direction, "") != SpecialDeclined {
				return true
			}
		}
	}
	for _, mob := range w.COrderedRoomMobs(room) {
		if mob == nil || mob.HasMobFlag(MobFlagExtract) {
			continue
		}
		if name, ok := GetMobSpecName(mob.VNum); ok {
			if DispatchCharacterSpecial(name, w, ctx, mob, direction, "") != SpecialDeclined {
				return true
			}
		}
	}
	for _, obj := range w.GetItemsInRoom(room) {
		if obj == nil {
			continue
		}
		if name, ok := GetObjSpecName(obj.VNum); ok {
			if DispatchCharacterObjectSpecial(name, w, ctx, obj, direction, "") != SpecialDeclined {
				return true
			}
		}
		ref := &scripting.CharRef{NPC: true, ID: ch.GetID()}
		if w.RunObjOnCmdScript(ref, nil, obj, room, direction) {
			return true
		}
	}
	return false
}

func (w *World) runMobGreetScripts(who combat.Combatant, room int) {
	if ScriptEngine == nil {
		return
	}
	for _, mob := range w.COrderedRoomMobs(room) {
		if mob == nil || mob == who || !mob.HasScript("greet") {
			continue
		}
		ctx := mob.CreateScriptContext(nil, nil, "")
		ctx.ChRef = combatantScriptRef(who)
		ctx.World = NewWorldScriptableAdapter(w)
		ctx.RoomVNum = room
		if _, err := mob.RunScript("greet", ctx); err != nil {
			slog.Warn("greet script error", "mob_vnum", mob.GetVNum(), "error", err)
		}
	}
}

func combatantScriptRef(ch combat.Combatant) *scripting.CharRef {
	switch body := ch.(type) {
	case *Player:
		return &scripting.CharRef{ID: body.GetID()}
	case *MobInstance:
		return &scripting.CharRef{NPC: true, ID: body.GetID()}
	default:
		return nil
	}
}
