package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
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
	InstanceID   string `json:"instance_id"`   // "mob_<runtimeID>"
	TargetString string `json:"target_string"` // exact string to pass to "hit"
	Fighting     bool   `json:"fighting"`
}

// RoomItemVar describes an item on the floor of the current room.
type RoomItemVar struct {
	Name         string `json:"name"`
	InstanceID   string `json:"instance_id"`   // "obj_<runtimeID>"
	TargetString string `json:"target_string"` // exact string to pass to "get"
}

// handleSubscribe processes a subscribe message from an agent.
// {"type":"subscribe","data":{"variables":["HEALTH","ROOM_VNUM",...]}}
func (s *Session) handleSubscribe(data json.RawMessage) error {
	s.agentMu.Lock()
	allowed := s.wantsStructuredData
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
	if !s.wantsStructuredData {
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
	if !s.wantsStructuredData || len(s.dirtyVars) == 0 {
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
	if s.stageHeartbeat(msg, "", false) {
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
	if s.stageHeartbeat(msg, "", false) {
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
		return s.player.GetHP()
	case VarMaxHealth:
		return s.player.GetMaxHealth()
	case VarMana:
		return s.player.GetMana()
	case VarMaxMana:
		return s.player.VitalsSnapshot().MaxMana
	case VarMove:
		return s.player.GetMove()
	case VarMaxMove:
		return s.player.VitalsSnapshot().MaxMove
	case VarGold:
		return s.player.GetGold()
	case VarPosition:
		pos := s.player.GetPosition()
		if pos >= 0 && pos < len(game.PositionNames) {
			return game.PositionNames[pos]
		}
		return fmt.Sprintf("unknown (%d)", pos)
	case VarLevel:
		return s.player.GetLevel()
	case VarExp:
		return s.player.GetExp()
	case VarRoomVnum:
		return s.player.GetRoom()
	case VarRoomName:
		if !s.agentCanSeeRoom() {
			return ""
		}
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			return ""
		}
		return room.Name
	case VarRoomExits:
		if !s.agentCanSeeRoom() {
			return []string{}
		}
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if !ok {
			return []string{}
		}
		return s.visibleExitNames(room)
	case VarRoomMobs:
		return s.buildRoomMobs()
	case VarRoomItems:
		return s.buildRoomItems()
	case VarFighting:
		target, fighting := s.manager.combatEngine.GetCombatTarget(s.player)
		if !fighting {
			return false
		}
		// R4: a mortal never sees integer enemy HP — do_diagnose renders
		// eight qualitative bands (diag_char_to_char, act.informative.c:363-
		// 382). Emit exactly those bytes plus the bucket index; nothing finer.
		return map[string]interface{}{
			"fighting":      true,
			"target":        target.GetName(),
			"condition":     game.DiagCondition(target.GetHP(), target.GetMaxHP()),
			"health_bucket": diagBucket(target.GetHP(), target.GetMaxHP()),
		}
	case VarInventory:
		return s.buildInventory()
	case VarEquipment:
		return s.buildEquipment()
	case VarEvents:
		return []interface{}{}
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

// objectFeedID returns a stable feed identifier for an object. World-
// registered objects use their registry ID; raw-constructed objects (rent
// restore, houses, shops — anything not yet seen by the world's counter)
// get a per-session sequence number assigned on first sight, stable for the
// object's life in this session. Never the raw obj.ID when it is 0: every
// unregistered object shares 0, which would collide in the feed.
func (s *Session) objectFeedID(item *game.ObjectInstance) string {
	if id := item.GetInstanceID(); id != 0 {
		return fmt.Sprintf("obj_%d", id)
	}
	s.agentMu.Lock()
	defer s.agentMu.Unlock()
	if s.seenObjectIDs == nil {
		s.seenObjectIDs = make(map[*game.ObjectInstance]int)
	}
	if n, ok := s.seenObjectIDs[item]; ok {
		return fmt.Sprintf("obj_local_%d", n)
	}
	s.localObjectSeq++
	s.seenObjectIDs[item] = s.localObjectSeq
	return fmt.Sprintf("obj_local_%d", s.localObjectSeq)
}

// agentCanSeeRoom is the mortal's own sight gate (look_at_room's darkness
// branch, look.go:216-227): blind, or a dark room without infravision /
// holy-light, suppresses the room exactly as "Darkness" does on screen
// (R4; docs/gmcp.md 17-25 states the same rule for Room.Info).
func (s *Session) agentCanSeeRoom() bool {
	if s.player == nil {
		return false
	}
	if s.player.IsAffected(game.AffBlind) {
		return false
	}
	if s.manager.world.IsRoomDark(s.player.GetRoom()) && !game.ChCanSeeInDark(s.player) {
		return false
	}
	return true
}

// visibleExitNames mirrors do_auto_exits (act.informative.c): closed doors
// are omitted for mortals; immortals see them marked (R4; gmcpVisibleExits
// applies the identical rule to GMCP Room.Info).
func (s *Session) visibleExitNames(room *parser.Room) []string {
	immortal := s.player.GetLevel() >= game.LVL_IMMORT
	names := make([]string, 0, len(room.Exits))
	for _, direction := range game.DirList() {
		exit, ok := room.Exits[direction]
		if !ok || exit.ToRoom <= 0 {
			continue
		}
		if exit.ExitInfo&parser.ExitClosed != 0 {
			if immortal {
				names = append(names, "("+direction+")")
			}
			continue
		}
		names = append(names, direction)
	}
	return names
}

// diagBucket maps HP to diag_char_to_char's eight bands
// (act.informative.c:363-382): 0 excellent, 1 few scratches, 2 small wounds,
// 3 quite a few wounds, 4 big nasty wounds, 5 pretty hurt, 6 awful, 7 the
// degenerate max<=0 "bleeding awfully" arm. Index only — never a finer value
// (R4).
func diagBucket(hp, maxHP int) int {
	percent := -1
	if maxHP > 0 {
		percent = (100 * hp) / maxHP
	}
	switch {
	case percent >= 100:
		return 0
	case percent >= 90:
		return 1
	case percent >= 75:
		return 2
	case percent >= 50:
		return 3
	case percent >= 30:
		return 4
	case percent >= 15:
		return 5
	case percent >= 0:
		return 6
	default:
		return 7
	}
}

// buildRoomMobs returns a []RoomMobVar for every mob in the player's room,
// with TargetStrings disambiguated when multiple mobs share a keyword.
func (s *Session) buildRoomMobs() []RoomMobVar {
	if !s.agentCanSeeRoom() {
		return []RoomMobVar{}
	}
	mobs := s.manager.world.GetMobsInRoom(s.player.GetRoom())
	// R4: the agent sees the occupants a mortal sees — CAN_SEE-filtered
	// (invisible mobs hidden unless the viewer sees invisible; look.go:418).
	visible := mobs[:0:0]
	for _, mob := range mobs {
		if game.ChCanSee(s.player, mob) {
			visible = append(visible, mob)
		}
	}
	mobs = visible
	if len(mobs) == 0 {
		return []RoomMobVar{}
	}

	// First pass: collect first keyword per mob, count occurrences
	keywords := make([]string, len(mobs))
	for i, mob := range mobs {
		kw := ""
		if mob.Proto() != nil {
			kw = firstMeaningfulKeyword(mob.Proto().Keywords)
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
			InstanceID:   fmt.Sprintf("mob_%d", mob.GetID()),
			TargetString: targetStrings[i],
			Fighting:     mob.IsFighting(),
		}
	}
	return result
}

// buildRoomItems returns a []RoomItemVar for every item on the room floor,
// with TargetStrings disambiguated when multiple items share a keyword.
func (s *Session) buildRoomItems() []RoomItemVar {
	if !s.agentCanSeeRoom() {
		return []RoomItemVar{}
	}
	items := s.manager.world.GetItemsInRoom(s.player.GetRoom())
	// R4: CAN_SEE_OBJ-filtered — invisible items hidden from mortal viewers
	// (act_informative.c chCanSeeObj).
	visible := items[:0:0]
	for _, item := range items {
		if game.ChCanSeeObj(s.player, item) {
			visible = append(visible, item)
		}
	}
	items = visible
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
			InstanceID:   s.objectFeedID(item),
			TargetString: targetStrings[i],
		}
	}
	return result
}

// buildInventory returns the player's carried items as a list of maps.
func (s *Session) buildInventory() []map[string]interface{} {
	items := s.player.Inventory.FindItems("")
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]interface{}{
			"name":        item.GetShortDesc(),
			"instance_id": s.objectFeedID(item),
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
