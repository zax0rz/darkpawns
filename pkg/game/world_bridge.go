package game

import (
	"log/slog"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// This file is the game's side of the Lua bridge (pkg/scripting/bridge.go):
// every method answers a C table function or binding from the live world,
// through the port's own game functions.

var _ scripting.Bridge = (*WorldScriptableAdapter)(nil)

func (a *WorldScriptableAdapter) resolveChar(ref scripting.CharRef) (Actor, *Player, *MobInstance) {
	if ref.NPC {
		m, ok := a.world.GetMobByID(ref.ID)
		if !ok || m == nil {
			return nil, nil, nil
		}
		return m, nil, m
	}
	p := a.world.GetPlayerByID(ref.ID)
	if p == nil {
		return nil, nil, nil
	}
	return p, p, nil
}

func (a *WorldScriptableAdapter) resolveObj(ref scripting.ObjRef) *ObjectInstance {
	a.world.mu.RLock()
	defer a.world.mu.RUnlock()
	return a.world.objectInstances[ref.ID]
}

func objRefs(objs []*ObjectInstance) []scripting.ObjRef {
	refs := make([]scripting.ObjRef, 0, len(objs))
	for _, o := range objs {
		if o != nil {
			refs = append(refs, scripting.ObjRef{ID: o.ID})
		}
	}
	return refs
}

// charRefFor names a live Player or MobInstance for the bridge.
func charRefFor(c interface{}) *scripting.CharRef {
	switch v := c.(type) {
	case *Player:
		if v != nil {
			return &scripting.CharRef{ID: v.ID}
		}
	case *MobInstance:
		if v != nil {
			return &scripting.CharRef{NPC: true, ID: v.GetID()}
		}
	}
	return nil
}

// masterRef resolves ch->master by the follow name, nil for none or self.
func (a *WorldScriptableAdapter) masterRef(self Actor, following string) *scripting.CharRef {
	if following == "" || following == self.GetName() {
		return nil
	}
	if p, ok := a.world.GetPlayer(following); ok {
		return charRefFor(p)
	}
	if m := a.world.GetMobByName(following); m != nil {
		return charRefFor(m)
	}
	return nil
}

// CharFields is char_to_table's view of a character (scripts.c:1823-1891).
func (a *WorldScriptableAdapter) CharFields(ref scripting.CharRef) (scripting.CharFields, bool) {
	_, p, m := a.resolveChar(ref)
	switch {
	case p != nil:
		f := scripting.CharFields{
			Name:  p.GetName(),
			Alias: charKeywords(p),
			Align: p.GetAlignment(),
			Gold:  p.GetGold(),
			Level: p.GetLevel(),
			HP:    p.GetHP(),
			MaxHP: p.GetMaxHP(),
			Pos:   p.GetPosition(),
			Evil:  p.GetAlignment() <= -350,
			IDNum: p.ID,
			Exp:   p.GetExp(),
			Bank:  p.GetBankGold(),
			Mana:  p.GetMana(),
			Move:  p.GetMove(),
			Cha:   p.Stats.Cha,
		}
		p.mu.RLock()
		f.Jail = p.JailTimer
		f.Tattoo = p.Tattoo
		p.mu.RUnlock()
		if p.Inventory != nil {
			f.Carry = objRefs(p.Inventory.FindItems(""))
		}
		if p.Equipment != nil {
			for _, slot := range cWearToGoSlot {
				if item, ok := p.Equipment.GetItemInSlot(slot); ok && item != nil {
					f.Worn = append(f.Worn, scripting.ObjRef{ID: item.ID})
				}
			}
		}
		f.Master = a.masterRef(p, p.GetFollowing())
		f.Followers = a.world.NumFollowers(p.GetName())
		return f, true
	case m != nil:
		f := scripting.CharFields{
			Name:  m.GetName(),
			Alias: charKeywords(m),
			Align: m.GetAlignment(),
			Gold:  m.GetGold(),
			Level: m.GetLevel(),
			HP:    m.GetHP(),
			MaxHP: m.GetMaxHP(),
			Pos:   m.GetPosition(),
			Evil:  m.GetAlignment() <= -350,
			NPC:   true,
			VNum:  m.GetVNum(),
			Timer: m.GetScriptWait(),
		}
		m.mu.RLock()
		f.Carry = objRefs(m.Inventory)
		for where := 0; where < NumWears; where++ {
			if item := m.Equipment[where]; item != nil {
				f.Worn = append(f.Worn, scripting.ObjRef{ID: item.ID})
			}
		}
		m.mu.RUnlock()
		f.Master = a.masterRef(m, m.GetFollowing())
		return f, true
	}
	return scripting.CharFields{}, false
}

// ObjFields is obj_to_table's view of an object (scripts.c:1894-1925).
func (a *WorldScriptableAdapter) ObjFields(ref scripting.ObjRef) (scripting.ObjFields, bool) {
	o := a.resolveObj(ref)
	if o == nil {
		return scripting.ObjFields{}, false
	}
	f := scripting.ObjFields{
		Name:     o.GetShortDesc(),
		Alias:    o.GetKeywords(),
		VNum:     o.GetVNum(),
		Cost:     o.GetCost(),
		Type:     o.GetTypeFlag(),
		Weight:   o.GetWeight(),
		Timer:    o.GetTimer(),
		Contents: objRefs(o.Contains),
	}
	if o.Prototype != nil {
		f.PercLoad = int(o.Prototype.LoadPercent)
	}
	for i := 0; i < 4; i++ {
		f.Val[i] = o.GetValue(i)
	}
	return f, true
}

// RoomFields is room_to_table's view of a room (scripts.c:1928-1972). People
// are in C's people-list order: most recent arrival first.
func (a *WorldScriptableAdapter) RoomFields(vnum int) (scripting.RoomFields, bool) {
	room := a.world.GetRoomInWorld(vnum)
	if room == nil {
		return scripting.RoomFields{}, false
	}
	f := scripting.RoomFields{VNum: room.VNum, Sect: room.Sector}
	for dir := range f.Exits {
		f.Exits[dir] = -1
		if exit, ok := room.Exits[dirKeys[dir]]; ok {
			f.Exits[dir] = exit.ToRoom
		}
	}
	type occupant struct {
		ref  scripting.CharRef
		seq  uint64
		name string
	}
	var people []occupant
	for _, m := range a.world.GetMobsInRoom(vnum) {
		people = append(people, occupant{*charRefFor(m), m.GetRoomEntrySequence(), m.GetName()})
	}
	for _, p := range a.world.GetPlayersInRoom(vnum) {
		people = append(people, occupant{*charRefFor(p), p.GetRoomEntrySequence(), p.GetName()})
	}
	sort.SliceStable(people, func(i, j int) bool {
		if people[i].seq != people[j].seq {
			return people[i].seq > people[j].seq
		}
		return people[i].name < people[j].name
	})
	for _, o := range people {
		f.People = append(f.People, o.ref)
	}
	f.Objs = objRefs(a.world.GetItemsInRoom(vnum))
	return f, true
}

// ApplyChar is table_to_char (scripts.c:1975-2052).
func (a *WorldScriptableAdapter) ApplyChar(ref scripting.CharRef, w scripting.CharWrite) {
	_, p, m := a.resolveChar(ref)
	switch {
	case p != nil:
		// If the character died during the script, keep what death left.
		if p.GetFlags()&(1<<uint(plrExtractBit)) != 0 || p.IsDying() {
			return
		}
		p.SetAlignment(w.Align)
		p.SetGold(w.Gold)
		p.SetLevel(w.Level)
		p.SetHP(w.HP)
		p.SetPosition(w.Pos)
		p.SetBankGold(w.Bank)
		p.SetExp(w.Exp)
		p.SetMana(w.Mana)
		p.SetMove(w.Move)
		p.mu.Lock()
		p.JailTimer = w.Jail
		p.Tattoo = w.Tattoo
		p.mu.Unlock()
	case m != nil:
		if !m.IsAlive() {
			return
		}
		m.SetAlignment(w.Align)
		m.SetGold(w.Gold)
		m.SetLevel(w.Level)
		m.SetHealth(w.HP)
		m.SetPosition(w.Pos)
		m.SetScriptWait(w.Timer)
	}
}

// ApplyObj is table_to_obj (scripts.c:2054-2094).
func (a *WorldScriptableAdapter) ApplyObj(ref scripting.ObjRef, w scripting.ObjWrite) {
	o := a.resolveObj(ref)
	if o == nil {
		return
	}
	// C strcpy's into obj->short_description; the port keeps it per instance.
	o.Runtime.ShortDescOverride = w.Name
	cost, weight := w.Cost, w.Weight
	o.CostOverride = &cost
	o.WeightOverride = &weight
	values := w.Val
	o.ValuesOverride = &values
	o.SetTimer(w.Timer)
}

// CharRoom is ch->in_room as a vnum.
func (a *WorldScriptableAdapter) CharRoom(ref scripting.CharRef) (int, bool) {
	actor, _, _ := a.resolveChar(ref)
	if actor == nil {
		return 0, false
	}
	room := actor.GetRoom()
	return room, room >= 0
}

// InRoomAboveZero is run_script's "ch->in_room > 0": a room whose real
// number is above 0, the lowest-numbered room being real number 0.
func (a *WorldScriptableAdapter) InRoomAboveZero(ref scripting.CharRef) bool {
	room, ok := a.CharRoom(ref)
	if !ok {
		return false
	}
	return a.world.RoomRNum(room) > 0
}

func (a *WorldScriptableAdapter) actorFor(ref *scripting.CharRef) Actor {
	if ref == nil {
		return nil
	}
	actor, _, _ := a.resolveChar(*ref)
	return actor
}

// Act is act() with the script's actors (scripts.c:80-120).
func (a *WorldScriptableAdapter) Act(txt string, hideInvisible bool, me *scripting.CharRef, obj *scripting.ObjRef, vict *scripting.CharRef, where int) {
	var object *ObjectInstance
	if obj != nil {
		object = a.resolveObj(*obj)
	}
	Act(a.world, hideInvisible, a.actorFor(me), a.actorFor(vict), object, nil, txt, "", luaActType(where))
}

// luaActType maps the TO_* values scripts pass (globals.lua: TO_ROOM 1,
// TO_VICT 2, TO_NOTVICT 3, TO_CHAR 4, TO_SLEEP 128) to the port's act types.
func luaActType(where int) int {
	actType := 0
	switch where &^ 128 {
	case 1:
		actType = ToRoom
	case 2:
		actType = ToVict
	case 3:
		actType = ToNotVict
	case 4:
		actType = ToChar
	}
	if where&128 != 0 {
		actType |= ToSleep
	}
	return actType
}

func (a *WorldScriptableAdapter) mobFor(ref scripting.CharRef) *MobInstance {
	_, _, m := a.resolveChar(ref)
	return m
}

// Say is lua_say's do_say(me, text).
func (a *WorldScriptableAdapter) Say(me scripting.CharRef, text string) {
	if m := a.mobFor(me); m != nil {
		a.world.npcSay(m, text)
	}
}

// Emote is lua_emote's do_echo(me, text, SCMD_EMOTE).
func (a *WorldScriptableAdapter) Emote(me scripting.CharRef, text string) {
	if m := a.mobFor(me); m != nil {
		a.world.npcEmote(m, text)
	}
}

// Tell is lua_tell's do_tell(me, " name message").
func (a *WorldScriptableAdapter) Tell(me scripting.CharRef, argument string) {
	if m := a.mobFor(me); m != nil {
		a.world.npcTell(m, argument)
	}
}

// Command is lua_action's command_interpreter(ch, line).
func (a *WorldScriptableAdapter) Command(ref scripting.CharRef, line string) {
	_, p, m := a.resolveChar(ref)
	switch {
	case m != nil:
		a.world.NPCCommand(m, line)
	case p != nil:
		if a.world.CommandExecFunc != nil {
			a.world.CommandExecFunc(p, line)
		}
	}
}

// CharIsHunting is HUNTING(ch) != NULL.
func (a *WorldScriptableAdapter) CharIsHunting(ref scripting.CharRef) bool {
	if m := a.mobFor(ref); m != nil {
		return m.IsHunting()
	}
	return false
}

// ExtractObj is extract_obj.
func (a *WorldScriptableAdapter) ExtractObj(ref scripting.ObjRef) {
	if o := a.resolveObj(ref); o != nil {
		a.world.ExtractObject(o, -1)
	}
}

// LoadObjToChar is read_object + obj_to_char (prepends, no carry limits).
func (a *WorldScriptableAdapter) LoadObjToChar(vnum int, to scripting.CharRef) (scripting.ObjRef, bool) {
	_, p, m := a.resolveChar(to)
	if p == nil && m == nil {
		return scripting.ObjRef{}, false
	}
	obj, err := a.world.SpawnObject(vnum, -1)
	if err != nil {
		return scripting.ObjRef{}, false
	}
	if p != nil {
		err = a.world.PlaceWizardLoadedObjectInInventory(obj, p)
	} else {
		err = a.world.MoveObjectToMobInventoryFront(obj, m)
	}
	if err != nil {
		slog.Error("lua oload to char failed", "vnum", vnum, "error", err)
		a.world.ExtractObject(obj, -1)
		return scripting.ObjRef{}, false
	}
	return scripting.ObjRef{ID: obj.ID}, true
}

// LoadObjToRoom is read_object + obj_to_room.
func (a *WorldScriptableAdapter) LoadObjToRoom(vnum int, roomVNum int) (scripting.ObjRef, bool) {
	obj, err := a.world.SpawnObject(vnum, -1)
	if err != nil {
		return scripting.ObjRef{}, false
	}
	if err := a.world.MoveObjectToRoomFront(obj, roomVNum); err != nil {
		slog.Error("lua oload to room failed", "vnum", vnum, "room", roomVNum, "error", err)
		a.world.ExtractObject(obj, -1)
		return scripting.ObjRef{}, false
	}
	return scripting.ObjRef{ID: obj.ID}, true
}

// SetSkill is lua_set_skill's SET_SKILL. Its affect_total changes nothing
// in the port, which applies modifiers incrementally (see bridgeSaveChar).
func (a *WorldScriptableAdapter) SetSkill(ref scripting.CharRef, skill, level int) {
	if _, p, _ := a.resolveChar(ref); p != nil {
		if name := spells.GetSpellName(skill); name != "" {
			p.SetSkill(name, level)
		}
	}
}

// Teleport is lua_tport's char_from_room, char_to_room and look_at_room.
func (a *WorldScriptableAdapter) Teleport(ref scripting.CharRef, roomVNum int) bool {
	if a.world.GetRoomInWorld(roomVNum) == nil {
		return false
	}
	_, p, m := a.resolveChar(ref)
	switch {
	case p != nil:
		if err := a.world.PlayerTransfer(p, roomVNum); err != nil {
			return false
		}
		a.world.lookAtRoom(p, false) // look_at_room(vict, 0)
	case m != nil:
		if err := a.world.MobTransfer(m, roomVNum); err != nil {
			return false
		}
	default:
		return false
	}
	return true
}

// RawKill is lua_raw_kill (scripts.c:1225-1258).
func (a *WorldScriptableAdapter) RawKill(vict scripting.CharRef, killer *scripting.CharRef, attackType int) {
	victim, p, m := a.resolveChar(vict)
	if victim == nil {
		return
	}
	if p != nil {
		if killer != nil {
			if k := a.actorFor(killer); k != nil {
				slog.Info("lua raw_kill", "victim", p.GetName(), "killer", k.GetName(), "room", p.GetRoom())
			}
		} else {
			slog.Info("lua raw_kill", "victim", p.GetName(), "room", p.GetRoom())
		}
		a.world.RawKillCombatant(p, attackType)
		return
	}
	a.world.RawKillCombatant(m, attackType)
}

// Log is mudlog for a script.
func (a *WorldScriptableAdapter) Log(msg string) {
	MudLog(msg, mudlogBrief, lvlImmort, false)
}

// CanSee is CAN_SEE(me, vict).
func (a *WorldScriptableAdapter) CanSee(me, vict scripting.CharRef) bool {
	observer, subject := a.actorFor(&me), a.actorFor(&vict)
	if observer == nil || subject == nil {
		return false
	}
	return canSee(observer, subject)
}

// InWorldMob is lua_inworld("mob", vnum): C keeps the last match in
// character_list, and read_mobile pushes each new mobile on the front, so
// the oldest instance wins. Mobile IDs are allocated in spawn order.
func (a *WorldScriptableAdapter) InWorldMob(vnum int) (scripting.CharRef, bool) {
	var oldest *MobInstance
	for _, m := range a.world.GetAllMobs() {
		if m.GetVNum() == vnum && (oldest == nil || m.GetID() < oldest.GetID()) {
			oldest = m
		}
	}
	if oldest == nil {
		return scripting.CharRef{}, false
	}
	return *charRefFor(oldest), true
}

// InWorldChar is lua_inworld("char", name): strcmp on GET_NAME, a player's
// name or a mobile's short description. The oldest mobile wins among
// mobiles, as in InWorldMob; players have no creation order the port
// shares with mobiles, so an exact mobile match is preferred to a player,
// which only matters when a mobile's short description is a player's name.
func (a *WorldScriptableAdapter) InWorldChar(name string) (scripting.CharRef, bool) {
	var oldest *MobInstance
	for _, m := range a.world.GetAllMobs() {
		if m.GetName() == name && (oldest == nil || m.GetID() < oldest.GetID()) {
			oldest = m
		}
	}
	if oldest != nil {
		return *charRefFor(oldest), true
	}
	for _, p := range a.world.GetAllPlayers() {
		if p.GetName() == name {
			return *charRefFor(p), true
		}
	}
	return scripting.CharRef{}, false
}

// AffFlagged is AFF_FLAGGED(ch, bit).
func (a *WorldScriptableAdapter) AffFlagged(ref scripting.CharRef, bit int) bool {
	switch _, p, m := a.resolveChar(ref); {
	case p != nil:
		return p.IsAffected(bit)
	case m != nil:
		return bit >= 0 && bit < 64 && m.IsAffected(bit)
	}
	return false
}

// SetAffFlag is SET_BIT_AR / REMOVE_BIT_AR on AFF_FLAGS(ch).
func (a *WorldScriptableAdapter) SetAffFlag(ref scripting.CharRef, bit int, on bool) {
	if bit < 0 || bit >= 64 {
		return
	}
	switch _, p, m := a.resolveChar(ref); {
	case p != nil:
		p.SetAffect(bit, on)
	case m != nil && on:
		m.SetAffected(bit)
	case m != nil:
		m.RemoveAffected(bit)
	}
}

// ActFlagged reads char_specials.saved.act: MOB_FLAGS for a mobile and
// PLR_FLAGS for a player are the same field.
func (a *WorldScriptableAdapter) ActFlagged(ref scripting.CharRef, bit int) bool {
	if bit < 0 || bit >= 64 {
		return false
	}
	switch _, p, m := a.resolveChar(ref); {
	case p != nil:
		return p.GetFlags()&(1<<uint(bit)) != 0
	case m != nil:
		return m.HasMobFlag(bit)
	}
	return false
}

// SetActFlag sets or clears a bit of the shared act field.
func (a *WorldScriptableAdapter) SetActFlag(ref scripting.CharRef, bit int, on bool) {
	if bit < 0 || bit >= 64 {
		return
	}
	switch _, p, m := a.resolveChar(ref); {
	case p != nil:
		p.SetPlrFlag(bit, on)
	case m != nil && on:
		m.SetMobFlag(bit)
	case m != nil:
		m.ClearMobFlag(bit)
	}
}

// ObjFlagged is OBJ_FLAGGED(obj, bit): bit indexes the extra-flag array.
func (a *WorldScriptableAdapter) ObjFlagged(ref scripting.ObjRef, bit int) bool {
	obj := a.resolveObj(ref)
	return obj != nil && bit >= 0 && obj.HasExtraFlag(bit/32, bit%32)
}

// SetObjExtra is SET_BIT_AR / REMOVE_BIT_AR on GET_OBJ_EXTRA(obj).
func (a *WorldScriptableAdapter) SetObjExtra(ref scripting.ObjRef, bit int, on bool) {
	obj := a.resolveObj(ref)
	if obj == nil || bit < 0 {
		return
	}
	if on {
		obj.SetExtraFlag(bit/32, bit%32)
	} else {
		obj.RemoveExtraFlag(bit/32, bit%32)
	}
}

// RoomExitInfo is dir_option[dir]->exit_info.
func (a *WorldScriptableAdapter) RoomExitInfo(room scripting.RoomRef, dir int) (int, bool) {
	r := a.world.GetRoomInWorld(room.VNum)
	if r == nil || dir < 0 || dir >= len(dirKeys) {
		return 0, false
	}
	exit, ok := r.Exits[dirKeys[dir]]
	return exit.ExitInfo, ok
}

// SetRoomExitInfo replaces dir_option[dir]->exit_info.
func (a *WorldScriptableAdapter) SetRoomExitInfo(room scripting.RoomRef, dir int, info int) bool {
	if dir < 0 || dir >= len(dirKeys) {
		return false
	}
	return a.world.SetExitInfo(room.VNum, dirKeys[dir], info)
}

// SetRoomSector is table_to_room's sector_type write.
func (a *WorldScriptableAdapter) SetRoomSector(room scripting.RoomRef, sect int) {
	a.world.mutateRoom(room.VNum, func(r *parser.Room) bool {
		r.Sector = sect
		return true
	})
}

// cIsCorpse is IS_CORPSE (utils.h:490-491).
func cIsCorpse(o *ObjectInstance) bool {
	return o.GetTypeFlag() == ITEM_CONTAINER && o.GetValue(3) == 1
}

// objInListVis is get_obj_in_list_vis (handler.c): the number-th object in
// list, in list order, whose keywords the name abbreviates and that the
// viewer can see (a light always counts).
func objInListVis(viewer Actor, name string, list []*ObjectInstance) *ObjectInstance {
	arg := strings.TrimSpace(name)
	number := GetNumber(&arg)
	if number <= 0 || arg == "" {
		return nil
	}
	found := 0
	for _, obj := range list {
		if !isnameWithAbbrevs(arg, obj.GetKeywords()) {
			continue
		}
		if !canSeeObject(viewer, obj) && obj.GetTypeFlag() != ITEM_LIGHT {
			continue
		}
		found++
		if found == number {
			return obj
		}
	}
	return nil
}

// carriedList and wornBySlot are ch->carrying and GET_EQ(ch, 0..NUM_WEARS-1)
// in C's order.
func (a *WorldScriptableAdapter) carriedList(ref scripting.CharRef) []*ObjectInstance {
	var out []*ObjectInstance
	f, ok := a.CharFields(ref)
	if !ok {
		return nil
	}
	for _, r := range f.Carry {
		if o := a.resolveObj(r); o != nil {
			out = append(out, o)
		}
	}
	return out
}

func (a *WorldScriptableAdapter) wornBySlot(ref scripting.CharRef) []*ObjectInstance {
	var out []*ObjectInstance
	f, ok := a.CharFields(ref)
	if !ok {
		return nil
	}
	for _, r := range f.Worn {
		if o := a.resolveObj(r); o != nil {
			out = append(out, o)
		}
	}
	return out
}

// LoadMob is lua_mload's read_mobile + char_to_room.
func (a *WorldScriptableAdapter) LoadMob(vnum, roomVNum int) (scripting.CharRef, bool) {
	if _, ok := a.world.GetMobPrototype(vnum); !ok {
		return scripting.CharRef{}, false
	}
	if a.world.GetRoomInWorld(roomVNum) == nil {
		slog.Error("lua mload: no such room (C's char_to_room gets NOWHERE)", "vnum", vnum, "room", roomVNum)
		return scripting.CharRef{}, false
	}
	mob, err := a.world.SpawnMob(vnum, roomVNum)
	if err != nil || mob == nil {
		slog.Error("lua mload failed", "vnum", vnum, "room", roomVNum, "error", err)
		return scripting.CharRef{}, false
	}
	return *charRefFor(mob), true
}

// ExtractChar is lua_extchar's extract_char. C marks the character and
// removes it at the next heartbeat (extract_pending_chars, comm.c:812); the
// port extracts at once (lua.bind-extchar-deferred).
func (a *WorldScriptableAdapter) ExtractChar(ref scripting.CharRef) {
	switch _, p, m := a.resolveChar(ref); {
	case m != nil:
		a.world.ExtractMob(m)
	case p != nil:
		slog.Warn("lua extchar on a player is not ported; ignored", "player", p.GetName())
	}
}

// ObjList is lua_obj_list's search (scripts.c:1040-1126). vict is the
// character whose carrying list or equipment held the object, which the
// binding makes the ch global.
func (a *WorldScriptableAdapter) ObjList(me scripting.CharRef, arg, where string) (scripting.ObjRef, *scripting.CharRef, bool) {
	viewer := a.actorFor(&me)
	room, ok := a.CharRoom(me)
	if viewer == nil || !ok {
		return scripting.ObjRef{}, nil, false
	}
	var list []*ObjectInstance
	var vict *scripting.CharRef
	switch where {
	case "room":
		list = a.world.GetItemsInRoom(room)
	case "char":
		for _, obj := range a.wornBySlot(me) {
			if isnameWithAbbrevs(arg, obj.GetKeywords()) {
				return scripting.ObjRef{ID: obj.ID}, nil, true
			}
		}
		list = a.carriedList(me)
	case "vict":
		people, _ := a.RoomFields(room)
		for _, p := range people.People {
			p := p
			carried := a.carriedList(p)
			if objInListVis(viewer, arg, carried) != nil {
				list, vict = carried, &p
				break
			}
			for _, obj := range a.wornBySlot(p) {
				if isnameWithAbbrevs(arg, obj.GetKeywords()) {
					return scripting.ObjRef{ID: obj.ID}, &p, true
				}
			}
		}
	case "cont", "corpse":
		for _, holder := range a.world.GetItemsInRoom(room) {
			if len(holder.Contains) == 0 || (where == "corpse" && !cIsCorpse(holder)) {
				continue
			}
			if objInListVis(viewer, arg, holder.Contains) != nil {
				list = holder.Contains
				break
			}
		}
	}
	if obj := objInListVis(viewer, arg, list); obj != nil {
		return scripting.ObjRef{ID: obj.ID}, vict, true
	}
	return scripting.ObjRef{}, nil, false
}

// ObjFrom is obj_from_room / obj_from_char / obj_from_obj: the object leaves
// where it is, when it is there.
func (a *WorldScriptableAdapter) ObjFrom(ref scripting.ObjRef, from string) {
	obj := a.resolveObj(ref)
	if obj == nil {
		return
	}
	loc := obj.Location
	var there bool
	switch from {
	case "room":
		there = loc.Kind == ObjInRoom
	case "char":
		there = loc.Kind == ObjInInventory
	case "obj":
		there = loc.Kind == ObjInContainer
	default:
		return
	}
	if !there {
		slog.Error("lua objfrom: object is not there", "obj_vnum", obj.VNum, "from", from)
		return
	}
	if err := a.world.MoveObjectToNowhere(obj); err != nil {
		slog.Error("lua objfrom failed", "obj_vnum", obj.VNum, "error", err)
	}
}

// ObjToRoom is obj_to_room.
func (a *WorldScriptableAdapter) ObjToRoom(ref scripting.ObjRef, roomVNum int) bool {
	obj := a.resolveObj(ref)
	if a.world.GetRoomInWorld(roomVNum) == nil {
		return false
	}
	if obj != nil {
		if err := a.world.MoveObjectToRoomFront(obj, roomVNum); err != nil {
			slog.Error("lua objto room failed", "obj_vnum", obj.VNum, "error", err)
		}
	}
	return true
}

// ObjToChar is obj_to_char, which prepends to the carrying list.
func (a *WorldScriptableAdapter) ObjToChar(ref scripting.ObjRef, to scripting.CharRef) {
	obj := a.resolveObj(ref)
	if obj == nil {
		return
	}
	var err error
	switch _, p, m := a.resolveChar(to); {
	case p != nil:
		err = a.world.PlaceWizardLoadedObjectInInventory(obj, p)
	case m != nil:
		err = a.world.MoveObjectToMobInventoryFront(obj, m)
	default:
		return
	}
	if err != nil {
		slog.Error("lua objto char failed", "obj_vnum", obj.VNum, "error", err)
	}
}

// ObjToObj is obj_to_obj.
func (a *WorldScriptableAdapter) ObjToObj(ref, into scripting.ObjRef) {
	obj, container := a.resolveObj(ref), a.resolveObj(into)
	if obj == nil || container == nil {
		return
	}
	if err := a.world.MoveObjectToContainer(obj, container); err != nil {
		slog.Error("lua objto obj failed", "obj_vnum", obj.VNum, "error", err)
	}
}

// Steal is lua_steal: obj_from_char(obj), obj_to_char(obj, me).
func (a *WorldScriptableAdapter) Steal(me scripting.CharRef, ref scripting.ObjRef) {
	a.ObjFrom(ref, "char")
	a.ObjToChar(ref, me)
}

// EquipCharObj is lua_equip_char: obj_from_char(obj), then equip_char(ch, obj,
// find_eq_pos(ch, obj, NULL)) (handler.c:679-747).
func (a *WorldScriptableAdapter) EquipCharObj(ref scripting.CharRef, objRef scripting.ObjRef) {
	obj := a.resolveObj(objRef)
	actor, p, m := a.resolveChar(ref)
	if obj == nil || actor == nil {
		return
	}
	pos := findEqPos(obj, "")
	if pos < 0 {
		// equip_char asserts pos >= 0 (handler.c:685): C aborts (R1a).
		slog.Error("lua equip_char: no wear position (C asserts)", "obj_vnum", obj.VNum)
		return
	}
	a.ObjFrom(objRef, "char")
	occupied := false
	if p != nil {
		occupied = a.world.IsEquipped(p, pos)
	} else if m != nil {
		m.mu.RLock()
		occupied = m.Equipment[pos] != nil
		m.mu.RUnlock()
	}
	if occupied {
		// "SYSERR: Char is already equipped": the object, already taken
		// from the carrier, is in no list.
		slog.Error("lua equip_char: char is already equipped", "char", actor.GetName(), "obj_vnum", obj.VNum)
		return
	}
	align := 0
	if p != nil {
		align = p.GetAlignment()
	} else {
		align = m.GetAlignment()
	}
	flags := obj.GetExtraFlags()[0]
	if flags&FlagAntiEvil != 0 && align <= -350 || flags&FlagAntiGood != 0 && align >= 350 ||
		flags&FlagAntiNeutral != 0 && align > -350 && align < 350 {
		Act(nil, false, actor, nil, obj, nil, "You are zapped by $p and instantly let go of it.", "", ToChar)
		Act(a.world, false, actor, nil, obj, nil, "$n is zapped by $p and instantly lets go of it.", "", ToRoom)
		a.ObjToChar(objRef, ref)
		return
	}
	if p != nil && objInvalidClass(p, obj) {
		Act(nil, false, actor, nil, obj, nil, "You cannot use $p.", "", ToChar)
		a.ObjToChar(objRef, ref)
		return
	}
	switch {
	case p != nil:
		if err := a.world.EquipItem(p, obj, pos); err != nil {
			slog.Error("lua equip_char failed", "player", p.GetName(), "obj_vnum", obj.VNum, "error", err)
		}
	case m != nil:
		m.mu.Lock()
		if m.Equipment == nil {
			m.Equipment = make(map[int]*ObjectInstance)
		}
		m.mu.Unlock()
		m.EquipItem(obj, pos)
	}
}

// AppendExtraDescs is lua_extra (scripts.c:512-540): every extra
// description gets the text appended, and the rebuilt list is in reverse
// order (each new entry is pushed on the front).
func (a *WorldScriptableAdapter) AppendExtraDescs(ref scripting.ObjRef, text string) {
	obj := a.resolveObj(ref)
	if obj == nil {
		return
	}
	descs := obj.LiveExtraDescs()
	out := make([]parser.ExtraDesc, 0, len(descs))
	for i := len(descs) - 1; i >= 0; i-- {
		d := descs[i]
		d.Description += text
		out = append(out, d)
	}
	obj.SetLiveExtraDescs(out)
}
