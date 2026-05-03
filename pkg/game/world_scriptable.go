package game

import (
	"context"
	"fmt"
	"log/slog"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

func (w *World) GetPlayersInRoomScriptable(roomVNum int) []scripting.ScriptablePlayer {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var players []scripting.ScriptablePlayer
	for _, p := range w.players {
		if p.RoomVNum == roomVNum {
			players = append(players, p)
		}
	}
	return players
}

// GetObjPrototypeScriptable returns an object prototype by vnum as ScriptableObject.
func (w *World) GetObjPrototypeScriptable(vnum int) scripting.ScriptableObject {
	w.mu.RLock()
	defer w.mu.RUnlock()

	obj, ok := w.objs[vnum]
	if !ok {
		return nil
	}
	// Create a wrapper that implements ScriptableObject
	return &scriptableObjWrapper{obj: obj}
}

// AddItemToRoomScriptable adds an item to a room.
// If obj is a scriptableObjInstanceWrapper the live ObjectInstance is used directly;
// otherwise a new instance is created from the object prototype.
func (w *World) AddItemToRoomScriptable(obj scripting.ScriptableObject, roomVNum int) error {
	item := objectInstanceFromScriptable(obj)
	if item == nil {
		// Fallback: create a new instance from prototype
		proto, ok := w.GetObjPrototype(obj.GetVNum())
		if !ok {
			return fmt.Errorf("AddItemToRoom: prototype vnum %d not found", obj.GetVNum())
		}
		item = NewObjectInstance(proto, roomVNum)
	}
	return w.MoveObjectToRoom(item, roomVNum)
}

// HandleNonCombatDeathScriptable handles player death from non-combat damage.
// Resolves the ScriptablePlayer to a *Player and delegates to the full death
// handler (HP reset, corpse creation, equipment drop, room echo, respawn).
func (w *World) HandleNonCombatDeathScriptable(player scripting.ScriptablePlayer) {
	slog.Debug("player died from non-combat damage", "player", player.GetName())
	if p, ok := w.GetPlayer(player.GetName()); ok {
		w.HandleNonCombatDeath(p)
	} else {
		slog.Warn("HandleNonCombatDeathScriptable: player not found in world",
			"player", player.GetName())
	}
}

// HandleSpellDeathScriptable handles death caused by a spell.
func (w *World) HandleSpellDeathScriptable(victimName string, spellNum int, roomVNum int) {
	slog.Info("spell death", "victim", victimName, "spell_num", spellNum, "room_vnum", roomVNum)

	// First, check if it's a player
	if player, ok := w.GetPlayer(victimName); ok {
		// Player died to spell
		// We need to handle this as a combat death with spell attack type
		// For now, we'll call handlePlayerDeath directly
		// The caster name is not tracked through the spell pipeline currently.
		// When spell casting tracks the caster entity through CallMagic, this
		// should resolve the caster name for death messages and PK logging.
		w.handlePlayerDeath(player, true, spellNum, "")
		return
	}

	// Check if it's a mob
	w.mu.RLock()
	for _, mob := range w.activeMobs {
		if mob.GetShortDesc() == victimName && mob.GetRoom() == roomVNum {
			w.mu.RUnlock()
			w.handleMobDeath(mob, nil, spellNum)
			return
		}
	}
	w.mu.RUnlock()
}

// ScriptableWorld interface implementation via adapter

// WorldScriptableAdapter adapts World to ScriptableWorld
type WorldScriptableAdapter struct {
	world *World
}

// NewWorldScriptableAdapter creates a WorldScriptableAdapter.
func NewWorldScriptableAdapter(w *World) *WorldScriptableAdapter {
	return &WorldScriptableAdapter{world: w}
}

func (a *WorldScriptableAdapter) GetRoomInWorld(vnum int) *parser.Room {
	return a.world.GetRoomInWorld(vnum)
}

func (a *WorldScriptableAdapter) GetPlayersInRoom(roomVNum int) []scripting.ScriptablePlayer {
	return a.world.GetPlayersInRoomScriptable(roomVNum)
}

func (a *WorldScriptableAdapter) GetMobsInRoom(roomVNum int) []scripting.ScriptableMob {
	return a.world.GetMobsInRoomScriptable(roomVNum)
}

func (a *WorldScriptableAdapter) GetMobByVNumAndRoom(vnum int, roomVNum int) scripting.ScriptableMob {
	return a.world.GetMobByVNumAndRoomScriptable(vnum, roomVNum)
}

func (a *WorldScriptableAdapter) GetObjPrototype(vnum int) scripting.ScriptableObject {
	return a.world.GetObjPrototypeScriptable(vnum)
}

func (a *WorldScriptableAdapter) AddItemToRoom(obj scripting.ScriptableObject, roomVNum int) error {
	return a.world.AddItemToRoomScriptable(obj, roomVNum)
}

func (a *WorldScriptableAdapter) HandleNonCombatDeath(player scripting.ScriptablePlayer) {
	// ScriptablePlayer is an interface — find the actual Player by name
	if p, ok := a.world.GetPlayer(player.GetName()); ok {
		a.world.HandleNonCombatDeath(p)
	}
}

func (a *WorldScriptableAdapter) HandleSpellDeath(victimName string, spellNum int, roomVNum int) {
	a.world.HandleSpellDeathScriptable(victimName, spellNum, roomVNum)
}

// SendTell delivers a private tell message to a named online player.
// Source: act.comm.c do_tell().
func (a *WorldScriptableAdapter) SendTell(targetName, message string) {
	if p, ok := a.world.GetPlayer(targetName); ok {
		p.SendMessage(message)
	}
}

func (a *WorldScriptableAdapter) GetItemsInRoom(roomVNum int) []scripting.ScriptableObject {
	return a.world.GetItemsInRoomScriptable(roomVNum)
}

func (a *WorldScriptableAdapter) HasItemByVNum(charName string, vnum int) bool {
	return a.world.HasItemByVNumScriptable(charName, vnum)
}

func (a *WorldScriptableAdapter) RemoveItemFromRoom(vnum int, roomVNum int) scripting.ScriptableObject {
	return a.world.RemoveItemFromRoomByVNum(vnum, roomVNum)
}

func (a *WorldScriptableAdapter) RemoveItemFromChar(charName string, vnum int) scripting.ScriptableObject {
	return a.world.RemoveItemFromCharByVNum(charName, vnum)
}

func (a *WorldScriptableAdapter) GiveItemToChar(charName string, obj scripting.ScriptableObject) error {
	return a.world.GiveItemToCharScriptable(charName, obj)
}

// SendToZone sends a message to all players in the same zone as the given room.
// Source: comm.c send_to_zone().
func (a *WorldScriptableAdapter) SendToZone(roomVNum int, msg string) {
	a.world.SendToZone(roomVNum, msg)
}

// SendToAll sends a message to all online players.
// Source: comm.c send_to_all().
func (a *WorldScriptableAdapter) SendToAll(msg string) {
	a.world.SendToAll(msg)
}

// ExecuteMobCommand makes a mob execute a game command.
// Source: scripts.c lua_action() → command_interpreter().
func (a *WorldScriptableAdapter) ExecuteMobCommand(mobVNum int, cmdStr string) {
	a.world.executeMobCommand(mobVNum, cmdStr)
}

// FindFirstStep returns direction (0-5) from src room to target room.
// Delegates to World.FindFirstStep (public wrapper added to graph.go).
func (a *WorldScriptableAdapter) FindFirstStep(src, target int) int {
	return a.world.FindFirstStep(src, target)
}

// IsRoomDark returns true if the given room VNum is dark.
// Based on utils.h IS_DARK() macro — checks ROOM_DARK flag.
func (a *WorldScriptableAdapter) IsRoomDark(roomVNum int) bool {
	return a.world.IsRoomDark(roomVNum)
}

// GetRoomZone returns the zone number for a given room VNum.
func (a *WorldScriptableAdapter) GetRoomZone(roomVNum int) int {
	return a.world.GetRoomZone(roomVNum)
}

// CreateEvent schedules a timed event on the world's event queue.
// Source: scripts.c lua_create_event() — create_event(source, target, obj, argument, trigger, delay, type)
func (a *WorldScriptableAdapter) CreateEvent(delay int, source, target, obj, argument int, trigger string, eventType int) uint64 {
	if a.world.EventQueue == nil {
		slog.Error("cannot create event: EventQueue is nil")
		return 0
	}

	// In the original C code, delay is in PULSE_VIOLENCE units (2 seconds = 20 pulses).
	// The Lua scripts pass small integers like 1, 6, 10 meaning "N * PULSE_VIOLENCE".
	// We convert to pulses: 1 delay unit = 20 pulses = 2 seconds.
	// Source: scripts.c line 306: event->count = PULSE_VIOLENCE * time
	// Source: structs.h: PULSE_VIOLENCE = (2 RL_SEC) = 20 pulses
	pulseDelay := int64(delay) * 20

	return a.world.EventQueue.Create(pulseDelay, source, target, obj, argument, trigger, eventType,
		func(ctx context.Context, src, tgt, o, arg int, trig string, et int) int64 {
			// When the event fires, dispatch the Lua trigger on the mob.
			// Source: events.c event_process() — calls the_event->func(event_obj)
			// The original lua_create_event stored a script_event struct with
			// me, obj, room, fname, type and called run_script() when fired.
			a.world.dispatchScriptEvent(src, tgt, o, arg, trig, et)
			return 0
		})
}

// ---------------------------------------------------------------------------
// ScriptableWorld adapter — STATE MUTATIONS & LOOKUPS
// (lua_batch2_mutations.go)
// ---------------------------------------------------------------------------

func (a *WorldScriptableAdapter) EquipChar(charName string, isMob bool, objVNum int) bool {
	return a.world.EquipChar(charName, isMob, objVNum)
}

func (a *WorldScriptableAdapter) SetFollower(followerName, leaderName string, followerIsMob bool) error {
	return a.world.SetFollower(followerName, leaderName, followerIsMob)
}

func (a *WorldScriptableAdapter) MountPlayer(playerName, mountName string) error {
	return a.world.MountPlayer(playerName, mountName)
}

func (a *WorldScriptableAdapter) DismountPlayer(playerName string) error {
	return a.world.DismountPlayer(playerName)
}

func (a *WorldScriptableAdapter) ClearAffects(charName string, isMob bool) {
	a.world.ClearAffects(charName, isMob)
}

func (a *WorldScriptableAdapter) EquipMob(mobVNum, roomVNum, objVNum int) {
	a.world.EquipMobByVNum(mobVNum, roomVNum, objVNum)
}

func (a *WorldScriptableAdapter) CanCarryObject(charName string, objVNum int) bool {
	return a.world.CanCarryObject(charName, objVNum)
}

func (a *WorldScriptableAdapter) IsCorpseObj(objVNum int) bool {
	return a.world.IsCorpseObj(objVNum)
}

func (a *WorldScriptableAdapter) SetHunting(hunterName, preyName string, hunterIsMob bool) {
	a.world.SetHunting(hunterName, preyName, hunterIsMob)
}

func (a *WorldScriptableAdapter) IsHunting(charName string, isMob bool) bool {
	return a.world.IsHunting(charName, isMob)
}

func (a *WorldScriptableAdapter) GetPlayerByID(id int) scripting.ScriptablePlayer {
	return a.world.GetPlayerByID(id)
}

func (a *WorldScriptableAdapter) SetObjectExtraDesc(vnum int, keyword string, description string) bool {
	return a.world.SetObjectExtraDesc(vnum, keyword, description)
}

func (a *WorldScriptableAdapter) SetObjectExtraFlag(vnum int, flag int, set bool) bool {
	return a.world.SetObjectExtraFlag(vnum, flag, set)
}

func (a *WorldScriptableAdapter) SetExitDoorState(roomVNum int, direction string, state int) bool {
	return a.world.SetExitDoorState(roomVNum, direction, state)
}

// scriptableObjWrapper wraps parser.Obj to implement ScriptableObject
type scriptableObjWrapper struct {
	obj *parser.Obj
}

func (w *scriptableObjWrapper) GetVNum() int {
	return w.obj.VNum
}

func (w *scriptableObjWrapper) GetKeywords() string {
	return w.obj.Keywords
}

func (w *scriptableObjWrapper) GetShortDesc() string {
	return w.obj.ShortDesc
}

func (w *scriptableObjWrapper) GetCost() int {
	return w.obj.Cost
}

func (w *scriptableObjWrapper) GetTimer() int {
	return 0 // parser.Obj has no Timer field — runtime timer tracked on ObjectInstance
}

func (w *scriptableObjWrapper) SetTimer(timer int) {
	// No-op: parser.Obj is a prototype — timer tracked on ObjectInstance at runtime
}

func (w *scriptableObjWrapper) GetTypeFlag() int {
	return w.obj.TypeFlag
}

// scriptableObjInstanceWrapper wraps ObjectInstance to implement ScriptableObject.
// Used by item-transfer Lua functions (objfrom/objto) to carry live instances.
type scriptableObjInstanceWrapper struct {
	item *ObjectInstance
}

func (w *scriptableObjInstanceWrapper) GetVNum() int         { return w.item.VNum }
func (w *scriptableObjInstanceWrapper) GetKeywords() string  { return w.item.GetKeywords() }
func (w *scriptableObjInstanceWrapper) GetShortDesc() string { return w.item.GetShortDesc() }
func (w *scriptableObjInstanceWrapper) GetCost() int         { return w.item.GetCost() }
func (w *scriptableObjInstanceWrapper) GetTimer() int        { return w.item.GetTimer() }
func (w *scriptableObjInstanceWrapper) SetTimer(t int)       { w.item.SetTimer(t) }
func (w *scriptableObjInstanceWrapper) GetTypeFlag() int     { return w.item.GetTypeFlag() }

// objectInstanceFromScriptable extracts the underlying ObjectInstance from a
// scriptableObjInstanceWrapper, returning nil for other ScriptableObject types.
func objectInstanceFromScriptable(obj scripting.ScriptableObject) *ObjectInstance {
	if w, ok := obj.(*scriptableObjInstanceWrapper); ok {
		return w.item
	}
	return nil
}

// GetItemsInRoomScriptable returns all items in a room as []ScriptableObject.
func (w *World) GetItemsInRoomScriptable(roomVNum int) []scripting.ScriptableObject {
	w.mu.RLock()
	defer w.mu.RUnlock()
	items := w.roomItems[roomVNum]
	result := make([]scripting.ScriptableObject, 0, len(items))
	for _, item := range items {
		result = append(result, &scriptableObjInstanceWrapper{item: item})
	}
	return result
}

// HasItemByVNumScriptable returns true if the named player has an item with the given vnum.
func (w *World) HasItemByVNumScriptable(charName string, vnum int) bool {
	w.mu.RLock()
	p, ok := w.players[charName]
	w.mu.RUnlock()
	if !ok {
		return false
	}
	p.Inventory.mu.RLock()
	defer p.Inventory.mu.RUnlock()
	for _, item := range p.Inventory.Items {
		if item.VNum == vnum {
			return true
		}
	}
	return false
}

// RemoveItemFromRoomByVNum removes the first item with the given vnum from the room.
// Returns the removed item as ScriptableObject, or nil if not found.
func (w *World) RemoveItemFromRoomByVNum(vnum int, roomVNum int) scripting.ScriptableObject {
	w.mu.Lock()
	defer w.mu.Unlock()
	items := w.roomItems[roomVNum]
	for i, item := range items {
		if item.VNum == vnum {
			w.roomItems[roomVNum] = append(items[:i], items[i+1:]...)
			item.RoomVNum = -1
			return &scriptableObjInstanceWrapper{item: item}
		}
	}
	return nil
}

// RemoveItemFromCharByVNum removes the first item with the given vnum from the named player.
// Returns the removed item as ScriptableObject, or nil if not found.
func (w *World) RemoveItemFromCharByVNum(charName string, vnum int) scripting.ScriptableObject {
	w.mu.RLock()
	p, ok := w.players[charName]
	w.mu.RUnlock()
	if !ok {
		return nil
	}
	item, found := p.Inventory.removeItemByVNum(vnum)
	if !found {
		return nil
	}
	return &scriptableObjInstanceWrapper{item: item}
}

// GiveItemToCharScriptable adds a ScriptableObject to the named player's inventory.
func (w *World) GiveItemToCharScriptable(charName string, obj scripting.ScriptableObject) error {
	w.mu.RLock()
	p, ok := w.players[charName]
	w.mu.RUnlock()
	if !ok {
		return fmt.Errorf("player %q not found", charName)
	}
	item := objectInstanceFromScriptable(obj)
	if item == nil {
		return fmt.Errorf("GiveItemToChar: object is not an ObjectInstance")
	}
	return p.Inventory.addItem(item)
}

// FireMobFightScript fires the "fight" trigger on a mob after a combat round.
// Called by the combat engine's ScriptFightFunc after each round.
// Source: mobact.c — mob_activity() calls Lua fight trigger during violence.
func (w *World) FireMobFightScript(mobName string, targetName string, roomVNum int) {
	if ScriptEngine == nil {
		return
	}

	// Find the mob by name in the room
	w.mu.RLock()
	var mob *MobInstance
	for _, m := range w.activeMobs {
		if m.GetRoom() == roomVNum && m.GetName() == mobName && m.HasScript("fight") {
			mob = m
			break
		}
	}
	// Find the target player
	var target scripting.ScriptablePlayer
	for _, p := range w.players {
		if p.GetName() == targetName {
			target = p
			break
		}
	}
	w.mu.RUnlock()

	if mob == nil {
		return
	}

	ctx := mob.CreateScriptContext(nil, nil, "")
	if target != nil {
		if p, ok := target.(*Player); ok {
			ctx.Ch = p
		}
	}
	ctx.World = NewWorldScriptableAdapter(w)
	ctx.RoomVNum = roomVNum

	if _, err := mob.RunScript("fight", ctx); err != nil {
		slog.Warn("fight script error", "mob_vnum", mob.GetVNum(), "mob_name", mob.GetName(), "error", err)
	}
}

// dispatchScriptEvent dispatches a Lua trigger when a scheduled event fires.
// This is the callback registered with EventQueue.Create() for script events.
// Based on the original lua_create_event() in scripts.c lines 247-316.
//
// The original stored: me, ch, obj, room, fname (trigger), type
// and called run_script() when the event fired.
func (w *World) dispatchScriptEvent(source, target, objVNum, argument int, trigger string, eventType int) {
	if ScriptEngine == nil {
		return
	}

	// Find the mob by instance ID (source)
	w.mu.RLock()
	var mob *MobInstance
	if source > 0 {
		mob = w.activeMobs[source]
	}
	w.mu.RUnlock()

	if mob == nil {
		// Mob may have died or been extracted — event is a no-op
		// This matches original behavior where extract_char cleans up events
		return
	}

	// Build script context
	ctx := mob.CreateScriptContext(nil, nil, "")
	ctx.World = NewWorldScriptableAdapter(w)
	ctx.RoomVNum = mob.GetRoom()

	// Target resolution for cross-entity script triggers.
	// In the original C code, target was a char_data pointer directly.
	// Our Go version stores target as a numeric int that may be a player ID,
	// mob instance ID, or 0 (no target). For now we only support events
	// where the mob is its own target (target == source). When an entity
	// lookup by ID is added (e.g., GetPlayerByID, GetMobByID), resolve
	// target here and pass the resulting entity name into ctx as needed.

	// Run the trigger function in the mob's script
	// The trigger name is the Lua function to call (e.g., "port", "jail", "bane_one")
	if mob.HasScript(trigger) {
		if _, err := mob.RunScript(trigger, ctx); err != nil {
			slog.Error("script error", "mob_vnum", mob.GetVNum(), "trigger", trigger, "error", err)
		}
	} else {
		// The mob's prototype may not have the trigger bit set, but the
		// function might still exist in the Lua file — try anyway
		if _, err := ScriptEngine.RunScript(ctx, mob.Prototype.ScriptName, trigger); err != nil {
			slog.Error("script error", "mob_vnum", mob.GetVNum(), "trigger", trigger, "error", err)
		}
	}
}
