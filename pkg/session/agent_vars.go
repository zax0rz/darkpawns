package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Variable name constants for the agent subscription system.
// Agents subscribe to these by name; the server flushes dirty ones after each command.
const (
	VarHealth    = "HEALTH"
	VarMaxHealth = "MAX_HEALTH"
	VarMana      = "MANA"
	VarMaxMana   = "MAX_MANA"
	VarMove      = "MOVE"
	VarMaxMove   = "MAX_MOVE"
	VarGold      = "GOLD"
	VarPosition  = "POSITION"
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
	VarHealth, VarMaxHealth, VarMana, VarMaxMana, VarMove, VarMaxMove,
	VarGold, VarPosition, VarLevel, VarExp,
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
	s.agentMu.Lock()
	allowed := s.isAgent || s.wantsStructuredData
	s.agentMu.Unlock()
	if !allowed {
		s.sendError("subscribe is only available to agents or structured clients")
		return nil
	}
	var sub struct {
		Variables []string `json:"variables"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}
	s.agentMu.Lock()
	for _, v := range sub.Variables {
		s.subscribedVars[v] = true
	}
	s.agentMu.Unlock()
	return nil
}

// markDirty marks vars as needing a flush if this session is an agent
// and the variable was subscribed.
// Safe to call from any goroutine (readPump or combat ticker).
func (s *Session) markDirty(vars ...string) {
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	if !s.isAgent && !s.wantsStructuredData {
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
	s.agentMu.Lock()
	if (!s.isAgent && !s.wantsStructuredData) || len(s.dirtyVars) == 0 {
		s.agentMu.Unlock()
		return
	}
	// Copy dirty keys and clear under lock, then build values without lock
	// (buildVarValue reads Player which has its own mutex).
	dirty := make(map[string]bool, len(s.dirtyVars))
	for k, v := range s.dirtyVars {
		dirty[k] = v
	}
	s.dirtyVars = make(map[string]bool)
	s.agentMu.Unlock()

	data := make(map[string]interface{}, len(dirty))
	for varName := range dirty {
		data[varName] = s.buildVarValue(varName)
	}
	msg, err := json.Marshal(ServerMessage{Type: MsgVars, Data: data})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("flushDirtyVars channel full — dropping vars", "player", s.playerName, "dirty_count", len(dirty))
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
		slog.Error("json.Marshal error", "error", err)
		return
	}
	select {
	case s.send <- msg:
	default:
		slog.Warn("sendFullVarDump channel full — dropping vars", "player", s.playerName)
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
	case VarMove:
		return s.player.Move
	case VarMaxMove:
		return s.player.MaxMove
	case VarGold:
		return s.player.Gold
	case VarPosition:
		pos := s.player.Position
		if pos >= 0 && pos < len(game.PositionNames) {
			return game.PositionNames[pos]
		}
		return fmt.Sprintf("unknown (%d)", pos)
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
		target, fighting := s.manager.combatEngine.GetCombatTarget(s.player.Name)
		if !fighting {
			return false
		}
		s.agentMu.Lock()
		isAgent := s.isAgent
		s.agentMu.Unlock()
		if isAgent {
			return true
		}
		return map[string]interface{}{
			"fighting": true,
			"target":   target.GetName(),
			"hp":       target.GetHP(),
			"max_hp":   target.GetMaxHP(),
		}
	case VarInventory:
		return s.buildInventory()
	case VarEquipment:
		return s.buildEquipment()
	case VarEvents:
		s.agentMu.Lock()
		isAgent := s.isAgent
		events := s.pendingEvents
		s.pendingEvents = nil
		s.agentMu.Unlock()
		if !isAgent || events == nil {
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

func disambiguatedTargetStrings(keywords []string) []string {
	keywordCount := make(map[string]int, len(keywords))
	for _, keyword := range keywords {
		keywordCount[keyword]++
	}

	keywordSeen := make(map[string]int, len(keywordCount))
	result := make([]string, len(keywords))
	for i, keyword := range keywords {
		keywordSeen[keyword]++
		n := keywordSeen[keyword]
		if keywordCount[keyword] == 1 || n == 1 {
			result[i] = keyword
		} else {
			result[i] = fmt.Sprintf("%d.%s", n, keyword)
		}
	}
	return result
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
	for i, mob := range mobs {
		kw := ""
		if mob.Prototype != nil {
			kw = firstMeaningfulKeyword(mob.Prototype.Keywords)
		}
		if kw == "" || kw == "unknown" {
			kw = fmt.Sprintf("mob%d", mob.VNum)
		}
		keywords[i] = kw
	}

	targetStrings := disambiguatedTargetStrings(keywords)
	result := make([]RoomMobVar, len(mobs))
	for i, mob := range mobs {
		result[i] = RoomMobVar{
			Name:         mob.GetShortDesc(),
			InstanceID:   fmt.Sprintf("mob_%d_%d", mob.VNum, i),
			TargetString: targetStrings[i],
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
	for i, item := range items {
		kw := ""
		if item.Prototype != nil {
			kw = firstMeaningfulKeyword(item.Prototype.Keywords)
		}
		if kw == "" || kw == "unknown" {
			kw = fmt.Sprintf("obj%d", item.VNum)
		}
		keywords[i] = kw
	}

	targetStrings := disambiguatedTargetStrings(keywords)
	result := make([]RoomItemVar, len(items))
	for i, item := range items {
		result[i] = RoomItemVar{
			Name:         item.GetShortDesc(),
			InstanceID:   fmt.Sprintf("obj_%d_%d", item.VNum, i),
			TargetString: targetStrings[i],
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
