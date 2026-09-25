package game

import (
	"log/slog"
	"sort"

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

// ObjFields is obj_to_table's view of an object (scripts.c:1893-1926).
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

// RoomFields is room_to_table's view of a room (scripts.c:1928-1970). People
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
	slog.Info("lua", "message", msg)
}
