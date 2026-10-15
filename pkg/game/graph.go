package game

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// BFS return constants — from graph.c
const (
	BFS_ERROR         = -1
	BFS_ALREADY_THERE = -2
	BFS_NO_PATH       = -3
)

// dirKeys maps direction indices to exit map keys in parser.Room.Exits.
var dirKeys = []string{"north", "east", "south", "west", "up", "down"}

// trackThroughDoors enables tracking through closed/hidden doors.
const trackThroughDoors = true

// huntNumber is a test seam for verifying hunt_victim's CMWC draw order.
// Production always uses the process-wide deterministic stream.
var huntNumber = dprng.Number

// bfsQueueEntry is a BFS queue element: room vnum and the first-step direction.
type bfsQueueEntry struct {
	room int
	dir  int
}

// FindFirstStep returns the first direction on the shortest path from src to target.
// Returns direction (0-5) or BFS_ERROR (-1) / BFS_ALREADY_THERE (-2) / BFS_NO_PATH (-3).
// Public wrapper around the private findFirstStep, exposed for the scripting engine.
func (w *World) FindFirstStep(src int, target int) int {
	return w.findFirstStep(src, target)
}

// findFirstStep returns the first direction on the shortest path from src to target.
// Returns direction (0-5) or BFS_ERROR/BFS_ALREADY_THERE/BFS_NO_PATH.
func (w *World) findFirstStep(src int, target int) int {
	if src == target {
		return BFS_ALREADY_THERE
	}

	marks := make(map[int]bool)
	queue := make([]bfsQueueEntry, 0, 64)
	queueHead := 0

	marks[src] = true

	// Enqueue first steps from src room
	srcRoom, ok := w.GetRoom(src)
	if !ok {
		return BFS_ERROR
	}
	for dir := 0; dir < 6; dir++ {
		exit, exists := srcRoom.Exits[dirKeys[dir]]
		if exists && w.validEdge(exit, marks) {
			marks[exit.ToRoom] = true
			queue = append(queue, bfsQueueEntry{room: exit.ToRoom, dir: dir})
		}
	}

	// Classic BFS
	for queueHead < len(queue) {
		curr := queue[queueHead]
		queueHead++

		if curr.room == target {
			return curr.dir
		}

		currRoom, ok := w.GetRoom(curr.room)
		if !ok {
			continue
		}
		for dir := 0; dir < 6; dir++ {
			exit, exists := currRoom.Exits[dirKeys[dir]]
			if exists && w.validEdge(exit, marks) {
				marks[exit.ToRoom] = true
				queue = append(queue, bfsQueueEntry{room: exit.ToRoom, dir: curr.dir})
			}
		}
	}

	return BFS_NO_PATH
}

// validEdge checks if an exit is a valid BFS edge.
// Does NOT modify room flags — uses a separate marks map.
func (w *World) validEdge(exit parser.Exit, marks map[int]bool) bool {
	if exit.ToRoom == -1 { // NOWHERE
		return false
	}
	if marks[exit.ToRoom] {
		return false
	}
	if w.roomHasFlag(exit.ToRoom, "notrack") {
		return false
	}
	if !trackThroughDoors && exit.ExitInfo&parser.ExitClosed != 0 {
		return false
	}
	// Water sector checks
	if destRoom, ok := w.GetRoom(exit.ToRoom); ok {
		if destRoom.Sector == SECT_WATER_SWIM || destRoom.Sector == SECT_WATER_NOSWIM {
			return false
		}
	}
	return true
}

// huntVictim moves a mob toward its hunting target.
// Handles door opening, evasion, safe-room checks, and trash-talk messages.
func (w *World) huntVictim(m *MobInstance) {
	if m == nil || m.GetHunting() == "" {
		return
	}

	// Ordinary name-based hunting retains its player lookup. Equipment's
	// zero-CHA check stores C's exact mobile prey identity (handler.c:669).
	m.mu.RLock()
	preyID := m.HuntingMobID
	m.mu.RUnlock()
	var target combat.Combatant
	var evasion int
	if preyID != 0 {
		w.mu.RLock()
		prey := w.activeMobs[preyID]
		w.mu.RUnlock()
		if prey != nil {
			target = prey
		}
	} else if player := w.findPlayerByName(m.GetHunting()); player != nil {
		// C's HUNTING(ch) resolves a player ID through get_char_by_id(),
		// which walks only CON_PLAYING descriptors (utils.c:691-704).
		// A retained linkless body therefore leaves HUNTING(ch) NULL and
		// hunt_victim returns before changing the hunt or drawing RNG.
		if !w.playerHasPlayingDescriptor(player) {
			return
		}
		target = player
		evasion = player.GetSkill("evasion")
	}
	if target == nil || target.GetRoom() < 2 {
		if m.CanSpeak() {
			w.npcSay(m, "Damn!  My prey is gone!!")
		}
		m.SetHunting("")
		return
	}

	dir := w.findFirstStep(m.GetRoom(), target.GetRoom())

	// C finds the path first and skips evasion only when hunter and target are
	// already together. Its two speech checks use independent number(0,6) draws.
	// #nosec G404 — game RNG, not cryptographic
	if evasion > 0 && dir != BFS_ALREADY_THERE && huntNumber(1, 151) < evasion {
		if m.CanSpeak() && huntNumber(0, 6) == 0 {
			w.npcSay(m, "Where the hell did my prey go?!")
		} else if m.CanSpeak() && huntNumber(0, 6) == 0 {
			w.npcSay(m, "Fuck this...")
		}
		return
	}

	// Don't hunt into peaceful/house rooms
	if w.roomHasFlag(target.GetRoom(), "peaceful") || w.roomHasFlag(target.GetRoom(), "house") {
		return
	}

	if dir < 0 {
		if dir == BFS_ALREADY_THERE && m.GetRoom() == target.GetRoom() {
			w.attackHuntingPrey(m, target)
		}
		m.SetHunting("")
		return
	}

	// Open doors if intelligent mob
	mobRoom, ok := w.GetRoom(m.GetRoom())
	if ok {
		if exit, exists := mobRoom.Exits[dirKeys[dir]]; exists {
			if exit.ExitInfo&parser.ExitClosed != 0 && mobIsIntelligent(m) && exit.Keywords != "" {
				w.mobOpenDoor(m, dir, exit.Keywords)
			}
		}
	}

	// Move mob
	w.mobPerformMove(m, dir)

	// Check if arrived
	if m.GetRoom() == target.GetRoom() {
		w.attackHuntingPrey(m, target)
	} else if m.CanSpeak() {
		if victim := asActor(target); victim != nil {
			w.huntTrashTalk(m, victim)
		}
	}
}

// huntTrashTalk delivers C's hunt_victim taunts (src/graph.c:346-402). Every
// sink is the real command: do_tell for the tells, do_gen_comm for the gossip,
// auction and shout. C prepends the victim's name to each tell's text, and that
// name is the tell's target argument.
func (w *World) huntTrashTalk(m *MobInstance, victim Actor) {
	victimName := victim.GetName()
	// #nosec G404 — game RNG, not cryptographic
	switch huntNumber(0, 150) {
	case 0:
		w.npcTell(m, victimName+" Let's have an ass-kicking contest")
	case 1:
		// #nosec G404 — game RNG, not cryptographic
		w.mobGenComm(m, fmt.Sprintf("Corpse of %s for sale in a minute.. %d coins.", victimName, huntNumber(1000, 2000)), "auction")
	case 2:
		w.npcTell(m, victimName+" Run to your momma, pansy!")
	case 3:
		w.npcTell(m, victimName+" I'm coming to kill you!")
	case 4:
		// C: sprintf(msg, "I hear %s thinks %s's bad.", GET_NAME(HUNTING(ch)),
		// HSSH(HUNTING(ch))) (src/graph.c:367-368).
		w.mobGenComm(m, fmt.Sprintf("I hear %s thinks %s's bad.", victimName, hssh(victim)), "gossip")
	case 5:
		w.mobGenComm(m, fmt.Sprintf("Your momma ain't gonna save you this time, %s.", victimName), "gossip")
	case 6:
		// #nosec G404 — game RNG, not cryptographic
		if huntNumber(0, 20) == 0 {
			w.mobGenComm(m, fmt.Sprintf("%s flees like a rabbit...", victimName), "gossip")
		}
	case 7:
		w.npcTell(m, victimName+" Come out and fight!")
	case 8:
		w.npcTell(m, victimName+" Watch out! Here I come to get you!")
	case 9:
		w.mobGenComm(m, fmt.Sprintf("How much will I get for the head of %s?", victimName), "auction")
	case 10:
		w.mobGenComm(m, fmt.Sprintf("Where is that little wimp, %s?", victimName), "gossip")
	case 11:
		w.mobGenComm(m, fmt.Sprintf("%s, you jerk!", victimName), "gossip")
	case 12:
		w.mobGenComm(m, "Damn it!", "shout")
	}
}

// findPlayerByName finds a player by name (case-insensitive). It is the
// name-keyed lookup hunt_victim and the mobile command helpers use; it is not
// a communication sink.
func (w *World) findPlayerByName(name string) *Player {
	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, p := range w.players {
		if strings.EqualFold(p.Name, name) {
			return p
		}
	}
	return nil
}

// mobOpenDoor has a mob open a door in a given direction.
func (w *World) mobOpenDoor(m *MobInstance, dir int, keyword string) {
	w.mu.Lock()
	room := w.rooms[m.GetRoom()]
	if room == nil {
		w.mu.Unlock()
		return
	}
	dirName := dirKeys[dir]
	ext, hasExit := room.Exits[dirName]
	if !hasExit {
		w.mu.Unlock()
		return
	}
	if ext.ExitInfo&parser.ExitClosed == 0 {
		w.mu.Unlock()
		return
	}

	ext.ExitInfo &^= parser.ExitClosed | parser.ExitLocked
	room.Exits[dirName] = ext

	// Handle reverse exit
	otherRoomVNum := ext.ToRoom
	otherRoom := w.rooms[otherRoomVNum]
	if otherRoom != nil {
		backDir := revDir[dir]
		backExt, hasBack := otherRoom.Exits[dirs[backDir]]
		if hasBack && backExt.ToRoom == m.GetRoom() {
			backExt.ExitInfo &^= parser.ExitClosed | parser.ExitLocked
			otherRoom.Exits[dirs[backDir]] = backExt
		}
	}
	w.mu.Unlock()

	m.SendMessage("You open the door.\r\n")
	actToRoom(w, m.GetRoom(), fmt.Sprintf("%s opens the door.", m.GetName()), m.GetName())
}

// mobPerformMove moves a mob in a given direction.
func (w *World) mobPerformMove(m *MobInstance, dir int) {
	oldRoom := m.GetRoom()
	toRoomVNum := -1
	if room, ok := w.GetRoom(oldRoom); ok {
		if exit, exists := room.Exits[dirKeys[dir]]; exists {
			toRoomVNum = exit.ToRoom
		}
	}
	if toRoomVNum == -1 {
		return
	}

	// Notify old room
	for _, p := range w.GetPlayersInRoom(oldRoom) {
		p.SendMessage(fmt.Sprintf("%s leaves %s.\r\n", m.GetName(), dirKeys[dir]))
	}

	m.SetRoom(toRoomVNum)

	// Notify new room
	for _, p := range w.GetPlayersInRoom(toRoomVNum) {
		p.SendMessage(fmt.Sprintf("%s has arrived.\r\n", m.GetName()))
	}
}

// mobAttackPlayer makes a mob attack a player.
func (w *World) mobAttackPlayer(m *MobInstance, target *Player) {
	m.SetFightingBody(target)
	target.SetFightingBody(m)
	if err := m.Attack(target, w); err != nil {
		slog.Warn("Attack failed in mobAttackPlayer", "mob", m.GetName(), "target", target.Name, "error", err)
	}
}

// mobIsIntelligent checks if a mob is intelligent enough to open doors.
func mobIsIntelligent(m *MobInstance) bool {
	// Check mob prototype flags for intelligence
	if m.Proto() != nil {
		for _, f := range m.Proto().AffectFlags {
			if f == "intelligent" {
				return true
			}
		}
	}
	return false
}

// Mobile prey is selected only by check_for_bad_stats; existing player hunting
// keeps its original attack path. C hunt_victim calls hit(ch, HUNTING(ch), -1).
func (w *World) attackHuntingPrey(m *MobInstance, target combat.Combatant) {
	if player, ok := target.(*Player); ok {
		w.mobAttackPlayer(m, player)
		return
	}
	if err := w.mobHit(m, target); err != nil {
		slog.Warn("mobile prey attack failed", "mob", m.GetName(), "target", target.GetName(), "error", err)
	}
}
