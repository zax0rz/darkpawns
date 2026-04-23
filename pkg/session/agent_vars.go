package session

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// Variable name constants for the agent subscription system.
// Agents subscribe to these by name; the server flushes dirty ones after each command.
const (
	VarHealth    = "HEALTH"
	VarMaxHealth = "MAX_HEALTH"
	VarMana      = "MANA"
	VarMaxMana   = "MAX_MANA"
	VarLevel     = "LEVEL"
	VarExp       = "EXP"
	VarRoomVnum  = "ROOM_VNUM"
	VarRoomName  = "ROOM_NAME"
	VarRoomExits = "ROOM_EXITS"
	VarRoomMobs  = "ROOM_MOBS"
	VarRoomItems = "ROOM_ITEMS"
	VarFighting  = "FIGHTING"
	VarInventory = "INVENTORY"
	VarEquipment = "EQUIPMENT"
	VarEvents    = "EVENTS"
)

// AllVariables lists every subscribable variable name.
var AllVariables = []string{
	VarHealth, VarMaxHealth, VarMana, VarMaxMana, VarLevel, VarExp,
	VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems,
	VarFighting, VarInventory, VarEquipment, VarEvents,
}

// RoomMobVar describes a mob in the current room for agent targeting.
// TargetString is the exact string to pass to "hit" — disambiguated if
// multiple mobs share the same first keyword ("goblin", "2.goblin", ...).
type RoomMobVar struct {
	Name         string `json:"name"`
	InstanceID   string `json:"instance_id"`   // "mob_<vnum>_<idx>"
	TargetString string `json:"target_string"` // exact string to pass to "hit"
	VNum         int    `json:"vnum"`
	Fighting     bool   `json:"fighting"`
}

// RoomItemVar describes an item on the floor of the current room.
type RoomItemVar struct {
	Name         string `json:"name"`
	InstanceID   string `json:"instance_id"`   // "obj_<vnum>_<idx>"
	TargetString string `json:"target_string"` // exact string to pass to "get"
	VNum         int    `json:"vnum"`
}

// handleSubscribe processes a subscribe message from an agent.
// {"type":"subscribe","data":{"variables":["HEALTH","ROOM_VNUM",...]}}
func (s *Session) handleSubscribe(data json.RawMessage) error {
	if !s.isAgent {
		s.sendError("subscribe is only available to agents")
		return nil
	}
	var sub struct {
		Variables []string `json:"variables"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}
	for _, v := range sub.Variables {
		s.subscribedVars[v] = true
	}
	return nil
}

// markDirty marks vars as needing a flush if this session is an agent
// and the variable was subscribed.
func (s *Session) markDirty(vars ...string) {
	if !s.isAgent {
		return
	}
	for _, v := range vars {
		if s.subscribedVars[v] {
			s.dirtyVars[v] = true
		}
	}
}

// flushDirtyVars serializes all dirty variables and sends a single
// {"type":"vars","data":{...}} message to the agent, then clears the set.
func (s *Session) flushDirtyVars() {
	if !s.isAgent || len(s.dirtyVars) == 0 {
		return
	}
	data := make(map[string]interface{}, len(s.dirtyVars))
	for varName := range s.dirtyVars {
		data[varName] = s.buildVarValue(varName)
	}
	s.dirtyVars = make(map[string]bool)
	msg, err := json.Marshal(ServerMessage{Type: MsgVars, Data: data})
	if err != nil {
		log.Printf("json.Marshal error: %v", err)
		return
	}
	select {
	case s.send <- msg:
	default:
	}
}

// sendFullVarDump sends all agent variables in a single vars message.
// Called on agent login (replaces the stub in agent.go).
func (s *Session) sendFullVarDump() {
	data := make(map[string]interface{}, len(AllVariables))
	for _, varName := range AllVariables {
		data[varName] = s.buildVarValue(varName)
	}
	msg, err := json.Marshal(ServerMessage{Type: MsgVars, Data: data})
	if err != nil {
		log.Printf("json.Marshal error: %v", err)
		return
	}
	select {
	case s.send <- msg:
	default:
	}
}

// buildVarValue returns the current value for a named agent variable.
func (s *Session) buildVarValue(varName string) interface{} {
	switch varName {
	case VarHealth:
		return s.player.Health
	case VarMaxHealth:
		return s.player.MaxHealth
	case VarMana:
		return s.player.Mana
	case VarMaxMana:
		return s.player.MaxMana
	case VarLevel:
		return s.player.Level
	case VarExp:
		return s.player.Exp
	case VarRoomVnum:
		return s.player.GetRoom()
	case VarRoomName:
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			return ""
		}
		return room.Name
	case VarRoomExits:
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			return []string{}
		}
		return getExitNames(room.Exits)
	case VarRoomMobs:
		return s.buildRoomMobs()
	case VarRoomItems:
		return s.buildRoomItems()
	case VarFighting:
		return s.manager.combatEngine.IsFighting(s.player.Name)
	case VarInventory:
		return s.buildInventory()
	case VarEquipment:
		return s.buildEquipment()
	case VarEvents:
		events := s.pendingEvents
		s.pendingEvents = nil
		if events == nil {
			return []interface{}{}
		}
		return events
	default:
		return nil
	}
}

// firstMeaningfulKeyword returns the first non-article word from a
// space-separated keyword string (skips "a", "an", "the").
func firstMeaningfulKeyword(keywords string) string {
	skip := map[string]bool{"a": true, "an": true, "the": true}
	for _, p := range strings.Fields(keywords) {
		low := strings.ToLower(p)
		if !skip[low] {
			return low
		}
	}
	// Fall back to first word if all were articles
	fields := strings.Fields(keywords)
	if len(fields) > 0 {
		return strings.ToLower(fields[0])
	}
	return "unknown"
}

// buildRoomMobs returns a []RoomMobVar for every mob in the player's room,
// with TargetStrings disambiguated when multiple mobs share a keyword.
func (s *Session) buildRoomMobs() []RoomMobVar {
	mobs := s.manager.world.GetMobsInRoom(s.player.GetRoom())
	if len(mobs) == 0 {
		return []RoomMobVar{}
	}

	// First pass: collect first keyword per mob, count occurrences
	keywords := make([]string, len(mobs))
	keywordCount := make(map[string]int)
	for i, mob := range mobs {
		kw := ""
		if mob.Prototype != nil {
			kw = firstMeaningfulKeyword(mob.Prototype.Keywords)
		}
		if kw == "" || kw == "unknown" {
			kw = fmt.Sprintf("mob%d", mob.VNum)
		}
		keywords[i] = kw
		keywordCount[kw]++
	}

	// Second pass: assign TargetStrings — first occurrence uses bare keyword,
	// subsequent occurrences get a numeric prefix ("2.goblin", "3.goblin", ...).
	keywordSeen := make(map[string]int)
	result := make([]RoomMobVar, len(mobs))
	for i, mob := range mobs {
		kw := keywords[i]
		keywordSeen[kw]++
		n := keywordSeen[kw]

		var targetString string
		if keywordCount[kw] == 1 || n == 1 {
			targetString = kw
		} else {
			targetString = fmt.Sprintf("%d.%s", n, kw)
		}

		result[i] = RoomMobVar{
			Name:         mob.GetShortDesc(),
			InstanceID:   fmt.Sprintf("mob_%d_%d", mob.VNum, i),
			TargetString: targetString,
			VNum:         mob.VNum,
			Fighting:     mob.Fighting,
		}
	}
	return result
}

// buildRoomItems returns a []RoomItemVar for every item on the room floor,
// with TargetStrings disambiguated when multiple items share a keyword.
func (s *Session) buildRoomItems() []RoomItemVar {
	items := s.manager.world.GetItemsInRoom(s.player.GetRoom())
	if len(items) == 0 {
		return []RoomItemVar{}
	}

	keywords := make([]string, len(items))
	keywordCount := make(map[string]int)
	for i, item := range items {
		kw := ""
		if item.Prototype != nil {
			kw = firstMeaningfulKeyword(item.Prototype.Keywords)
		}
		if kw == "" || kw == "unknown" {
			kw = fmt.Sprintf("obj%d", item.VNum)
		}
		keywords[i] = kw
		keywordCount[kw]++
	}

	keywordSeen := make(map[string]int)
	result := make([]RoomItemVar, len(items))
	for i, item := range items {
		kw := keywords[i]
		keywordSeen[kw]++
		n := keywordSeen[kw]

		var targetString string
		if keywordCount[kw] == 1 || n == 1 {
			targetString = kw
		} else {
			targetString = fmt.Sprintf("%d.%s", n, kw)
		}

		result[i] = RoomItemVar{
			Name:         item.GetShortDesc(),
			InstanceID:   fmt.Sprintf("obj_%d_%d", item.VNum, i),
			TargetString: targetString,
			VNum:         item.VNum,
		}
	}
	return result
}

// buildInventory returns the player's carried items as a list of maps.
func (s *Session) buildInventory() []map[string]interface{} {
	items := s.player.Inventory.FindItems("")
	result := make([]map[string]interface{}, 0, len(items))
	for i, item := range items {
		result = append(result, map[string]interface{}{
			"name":        item.GetShortDesc(),
			"vnum":        item.VNum,
			"instance_id": fmt.Sprintf("obj_%d_%d", item.VNum, i),
		})
	}
	return result
}

// buildEquipment returns the player's equipped items as slot → {name, vnum}.
func (s *Session) buildEquipment() map[string]interface{} {
	equipped := s.player.Equipment.GetEquippedItems()
	result := make(map[string]interface{}, len(equipped))
	for slot, item := range equipped {
		result[slot.String()] = map[string]interface{}{
			"name": item.GetShortDesc(),
			"vnum": item.VNum,
		}
	}
	return result
}
