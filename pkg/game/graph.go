package game

import (
	"fmt"
	"log"
	"math/rand/v2"
	"strings"

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

// bfsQueueEntry is a BFS queue element: room vnum and the first-step direction.
type bfsQueueEntry struct {
	room int
	dir  int
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
	if !trackThroughDoors && (exit.DoorState != 0) { // EX_CLOSED = 1
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

// canGo checks if a character can physically move in a direction from their current room.
func (w *World) canGo(roomVNum int, dir int) bool {
	room, ok := w.GetRoom(roomVNum)
	if !ok {
		return false
	}
	exit, exists := room.Exits[dirKeys[dir]]
	if !exists || exit.ToRoom == -1 {
		return false
	}
	if (exit.DoorState) != 0 { // EX_CLOSED
		return false
	}
	return true
}

// doTrack implements the player 'track' skill.
// Only usable by warriors, paladins, and rangers.
func (w *World) doTrack(ch *Player, me *MobInstance, argument string) bool {
	if me != nil {
		// Mobs don't track via this command
		return true
	}

	class := ch.GetClass()
	if class != ClassWarrior && class != ClassPaladin && class != ClassRanger {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	skill := ch.GetSkill("track")
	if skill <= 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	if argument == "" {
		ch.SendMessage("Whom are you trying to track?\r\n")
		return true
	}

	vict := w.getCharVis(ch, argument)
	if vict == nil {
		ch.SendMessage("You can't sense a trail to them from here.\r\n")
		return true
	}

	// Check sentinel mob flag on NPC victims
	if vict.IsNPC() && w.mobHasFlag(vict, MobSentinel) {
		ch.SendMessage("You sense no trail.\r\n")
		return true
	}

	// Evasion check for player victims
	if !vict.IsNPC() {
		evasion := vict.GetSkill("evasion")
		if evasion > 0 && rand.IntN(151)+1 <= evasion {
			ch.SendMessage("You sense no trail.\r\n")
			return true
		}
	}

	dir := w.findFirstStep(ch.GetRoom(), vict.GetRoom())

	switch dir {
	case BFS_ERROR:
		ch.SendMessage("Hmm.. something seems to be wrong.\r\n")
	case BFS_ALREADY_THERE:
		ch.SendMessage("You're already in the same room!!\r\n")
	case BFS_NO_PATH:
		ch.SendMessage("You can't sense a trail to them from here.\r\n")
	default:
		num := rand.IntN(102) // 0-101, 101% is complete failure

		// Weather penalty — not yet implemented.
		// When the weather system is active, apply a skill penalty for
		// adverse conditions (rain, snow, wind, darkness) that reduce
		// tracking effectiveness. Expected logic:
		//   if weather.IsPrecipitating(room.Zone) || weather.IsDark(room.Zone) {
		//       num += weather.TrackingPenalty(room.Zone) // 0-20
		//   }
		// See src/weather.c:weather_change() and src/graph.c:find_first_step()
		// for the original C interaction.

		if num >= skill {
			// Skill failure — pick a random valid direction
			for tries := 10; tries > 0; tries-- {
				dir = rand.IntN(6)
				if w.canGo(ch.GetRoom(), dir) {
					break
				}
			}
		} else {
			improveSkill(ch, "track")
		}

		if w.canGo(ch.GetRoom(), dir) {
			ch.SendMessage(fmt.Sprintf("You sense a trail %s from here!\r\n", dirKeys[dir]))
		} else {
			ch.SendMessage("There doesn't seem to be any way out of here!\r\n")
		}
	}

	return true
}

// huntVictim moves a mob toward its hunting target.
// Handles door opening, evasion, safe-room checks, and trash-talk messages.
func (w *World) huntVictim(m *MobInstance) {
	if m == nil || m.Hunting == "" {
		return
	}

	// Find the hunting target among players
	target := w.findPlayerByName(m.Hunting)
	if target == nil || target.GetRoom() < 2 {
		if m.CanSpeak() {
			w.mobSayTo(m, "Damn!  My prey is gone!!")
		}
		m.SetHunting("")
		return
	}

	// Evasion check
	if evasion := target.GetSkill("evasion"); evasion > 0 && rand.IntN(151)+1 < evasion {
		r := rand.IntN(7)
		if m.CanSpeak() && r == 0 {
			w.mobSayTo(m, "Where the hell did my prey go?!")
		} else if m.CanSpeak() && r == 1 {
			w.mobSayTo(m, "Fuck this...")
		}
		return
	}

	// Don't hunt into peaceful/house rooms
	if w.roomHasFlag(target.GetRoom(), "peaceful") || w.roomHasFlag(target.GetRoom(), "house") {
		return
	}

	dir := w.findFirstStep(m.GetRoom(), target.GetRoom())

	if dir < 0 {
		if dir == BFS_ALREADY_THERE && m.GetRoom() == target.GetRoom() {
			w.mobAttackPlayer(m, target)
		}
		m.SetHunting("")
		return
	}

	// Open doors if intelligent mob
	mobRoom, ok := w.GetRoom(m.GetRoom())
	if ok {
		if exit, exists := mobRoom.Exits[dirKeys[dir]]; exists {
			if (exit.DoorState != 0) && mobIsIntelligent(m) && exit.Keywords != "" {
				w.mobOpenDoor(m, dir, exit.Keywords)
			}
		}
	}

	// Move mob
	w.mobPerformMove(m, dir)

	// Check if arrived
	if m.GetRoom() == target.GetRoom() {
		w.mobAttackPlayer(m, target)
	} else if m.CanSpeak() {
		w.huntTrashTalk(m, target.GetName())
	}
}

// huntTrashTalk delivers the classic mob trash-talk messages while hunting.
func (w *World) huntTrashTalk(m *MobInstance, victimName string) {
	switch rand.IntN(151) {
	case 0:
		w.mobTellPlayer(m, victimName, "Let's have an ass-kicking contest")
	case 1:
		w.mobAuction(m, fmt.Sprintf("Corpse of %s for sale in a minute.. %d coins.", victimName, rand.IntN(1001)+1000))
	case 2:
		w.mobTellPlayer(m, victimName, "Run to your momma, pansy!")
	case 3:
		w.mobTellPlayer(m, victimName, "I'm coming to kill you!")
	case 4:
		w.mobGossip(m, fmt.Sprintf("I hear %s thinks they're bad.", victimName))
	case 5:
		w.mobGossip(m, fmt.Sprintf("Your momma ain't gonna save you this time, %s.", victimName))
	case 6:
		if rand.IntN(21) == 0 {
			w.mobGossip(m, fmt.Sprintf("%s flees like a rabbit...", victimName))
		}
	case 7:
		w.mobTellPlayer(m, victimName, "Come out and fight!")
	case 8:
		w.mobTellPlayer(m, victimName, "Watch out! Here I come to get you!")
	case 9:
		w.mobAuction(m, fmt.Sprintf("How much will I get for the head of %s?", victimName))
	case 10:
		w.mobGossip(m, fmt.Sprintf("Where is that little wimp, %s?", victimName))
	case 11:
		w.mobGossip(m, fmt.Sprintf("%s, you jerk!", victimName))
	case 12:
		w.mobShout(m, "Damn it!")
	}
}

// --- Stub helpers for mob communication and actions ---
// These delegate to existing World methods. Remove as native mob comm is added.

// findPlayerByName finds a player by name (case-insensitive).
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

// mobHasFlag checks if a player-backed mob has a given mob flag.
func (w *World) mobHasFlag(p *Player, flag string) bool {
	// Mob flags are checked via the mob prototype when the Player represents a mob
	return false // stub until mob flag access is implemented
}

// mobSayTo makes a mob say something to their room.
func (w *World) mobSayTo(m *MobInstance, msg string) {
	m.SendMessage(msg)
}

// mobTellPlayer makes a mob tell a player something.
func (w *World) mobTellPlayer(m *MobInstance, targetName, msg string) {
	target := w.findPlayerByName(targetName)
	if target == nil {
		log.Printf("mobTellPlayer: player %q not found", targetName)
		return
	}
	target.SendMessage(fmt.Sprintf("\r\n%s tells you, '%s'\r\n", m.GetName(), msg))
	m.SendMessage(fmt.Sprintf("\r\nYou tell %s, '%s'\r\n", targetName, msg))
}

// mobGossip makes a mob gossip.
func (w *World) mobGossip(m *MobInstance, msg string) {
	text := fmt.Sprintf("%s gossips, '%s'\r\n", m.GetName(), msg)
	for _, p := range w.GetPlayersInRoom(m.GetRoom()) {
		p.SendMessage(text)
	}
}

// mobAuction makes a mob auction.
func (w *World) mobAuction(m *MobInstance, msg string) {
	text := fmt.Sprintf("%s auctions, '%s'\r\n", m.GetName(), msg)
	for _, p := range w.GetPlayersInRoom(m.GetRoom()) {
		p.SendMessage(text)
	}
}

// mobShout makes a mob shout.
func (w *World) mobShout(m *MobInstance, msg string) {
	text := fmt.Sprintf("%s shouts, '%s'\r\n", m.GetName(), msg)
	w.mu.RLock()
	for _, p := range w.players {
		p.SendMessage(text)
	}
	w.mu.RUnlock()
}

// mobOpenDoor has a mob open a door in a given direction.
func (w *World) mobOpenDoor(m *MobInstance, dir int, keyword string) {
	room := w.GetRoomInWorld(m.GetRoom())
	if room == nil {
		return
	}
	dirName := dirKeys[dir]
	ext, hasExit := room.Exits[dirName]
	if !hasExit {
		return
	}
	if ext.DoorState != doorClosed && ext.DoorState != doorLocked {
		return
	}

	w.mu.Lock()
	ext.DoorState = doorOpen
	room.Exits[dirName] = ext

	// Handle reverse exit
	otherRoomVNum := ext.ToRoom
	otherRoom := w.GetRoomInWorld(otherRoomVNum)
	if otherRoom != nil {
		backDir := revDir[dir]
		backExt, hasBack := otherRoom.Exits[dirs[backDir]]
		if hasBack && backExt.ToRoom == m.GetRoom() {
			backExt.DoorState = doorOpen
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
	m.SetFighting(target.GetName())
	target.SetFighting(m.GetName())
	m.Attack(target, w)
}

// mobIsIntelligent checks if a mob is intelligent enough to open doors.
func mobIsIntelligent(m *MobInstance) bool {
	// Check mob prototype flags for intelligence
	if m.Prototype != nil {
		for _, f := range m.Prototype.AffectFlags {
			if f == "intelligent" {
				return true
			}
		}
	}
	return false
}
