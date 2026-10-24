package game

import (
	"fmt"
	"log/slog"
	"strconv"
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

// trackNumber is a test seam for doTrack's CMWC draws, mirroring huntNumber.
// Production always uses the process-wide deterministic stream.
var trackNumber = dprng.Number

// doTrack ports ACMD(do_track) from src/graph.c:187-258. C registers it for
// every player at POS_STANDING with no level gate (src/interpreter.c:782);
// the warrior/paladin/ranger class gate is the command's own first check.
func (w *World) doTrack(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// C IS_WARRIOR/IS_PALADIN/IS_RANGER all carry !IS_NPC, so only a player
	// body can pass (src/utils.h:408-421,578-580).
	switch ch.GetClass() {
	case ClassWarrior, ClassPaladin, ClassRanger:
	default:
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	if ch.GetSkill("track") == 0 {
		ch.SendMessage("You have no idea how.\r\n")
		return true
	}

	name, _ := oneArgument(arg)
	if name == "" {
		ch.SendMessage("Whom are you trying to track?\r\n")
		return true
	}

	// get_char_vis: visible in the room first, then the whole world.
	target, found := w.ResolveCharWorld(ch, name)
	if !found {
		ch.SendMessage("You can't sense a trail to them from here.\r\n")
		return true
	}

	notrack := false
	switch body := target.Combatant.(type) {
	case *Player:
		notrack = body.IsAffected(affNoTrack)
	case *MobInstance:
		notrack = body.IsAffected(affNoTrack) || body.HasMobFlag(MobFlagSentinel)
	}
	if notrack {
		ch.SendMessage("You sense no trail.\r\n")
		return true
	}

	// Only a non-NPC victim can carry SKILL_EVASION (src/graph.c:216).
	if target.Player != nil {
		if evasion := target.Player.GetSkill("evasion"); evasion != 0 {
			// #nosec G404 — game RNG, not cryptographic
			if trackNumber(1, 151) <= evasion {
				ch.SendMessage("You sense no trail.\r\n")
				return true
			}
		}
	}

	dir := w.findFirstStep(ch.GetRoom(), target.Combatant.GetRoom())

	switch dir {
	case BFS_ERROR:
		ch.SendMessage("Hmm.. something seems to be wrong.\r\n")
	case BFS_ALREADY_THERE:
		ch.SendMessage("You're already in the same room!!\r\n")
	case BFS_NO_PATH:
		ch.SendMessage(fmt.Sprintf("You can't sense a trail to %s from here.\r\n", hmhr(asActor(target.Combatant))))
	default:
		num := trackNumber(0, 101) // 101% is a complete failure
		if w.charOutsideC(ch) && weatherInfo.Sky == SkyRaining {
			num += 35
		}
		if num >= ch.GetSkill("track") { // failure
			// C: do { dir = number(0, NUM_OF_DIRS-1); } while (!CAN_GO(ch,
			// dir) && --tries); — at most ten draws, and dir may still name
			// no exit afterwards (src/graph.c:243-245).
			tries := 10
			for {
				dir = trackNumber(0, len(dirKeys)-1)
				if w.canGoC(ch, dir) {
					break
				}
				tries--
				if tries == 0 {
					break
				}
			}
		} else {
			improveSkill(ch, "track")
		}

		if w.canGoC(ch, dir) {
			ch.SendMessage(fmt.Sprintf("You sense a trail %s from here!\r\n", dirKeys[dir]))
		} else {
			ch.SendMessage("There doesn't seem to be any way out of here!\r\n")
		}
	}
	return true
}

// ExecTrack is the session-bridge form of doTrack (the "track" command).
func (w *World) ExecTrack(ch *Player, arg string) { w.doTrack(ch, nil, "track", arg) }

// canGoC is CAN_GO(ch, door) from src/utils.h:565-567: the exit exists, does
// not lead to NOWHERE, and is not closed.
func (w *World) canGoC(ch *Player, dir int) bool {
	if dir < 0 || dir >= len(dirKeys) {
		return false
	}
	room, ok := w.GetRoom(ch.GetRoom())
	if !ok {
		return false
	}
	exit, exists := room.Exits[dirKeys[dir]]
	return exists && exit.ToRoom != -1 && exit.ExitInfo&parser.ExitClosed == 0
}

// charOutsideC is OUTSIDE(ch) from src/utils.h:581-582: the room is not
// ROOM_INDOORS-flagged or its sector is not SECT_INSIDE.
func (w *World) charOutsideC(ch *Player) bool {
	room, ok := w.GetRoom(ch.GetRoom())
	if !ok {
		return false
	}
	return !w.roomHasFlag(ch.GetRoom(), "indoors") || room.Sector != SECT_INSIDE
}

// huntVictim moves a mob toward its hunting target.
// Handles door opening, evasion, safe-room checks, and trash-talk messages.
func (w *World) huntVictim(m *MobInstance) {
	if m == nil || m.GetHunting() == "" {
		return
	}

	// C HUNTING prefers a retained mobile pointer, then resolves a player ID
	// through a playing descriptor (src/utils.c:715-733).
	m.mu.RLock()
	preyID, playerID, preyName := m.HuntingMobID, m.HuntingID, m.Hunting
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
	} else if playerID != "" {
		id, err := strconv.Atoi(playerID)
		if err != nil {
			return
		}
		player := w.GetPlayerByID(id)
		if player == nil || !w.playerHasPlayingDescriptor(player) {
			// get_char_by_id only traverses CON_PLAYING descriptors. HUNTING
			// is NULL while the player is linkless, so C returns untouched.
			return
		}
		target = player
		evasion = player.GetSkill("evasion")
	} else if player := w.findPlayerByName(preyName); player != nil {
		// Name-only targets are retained for legacy standalone fixtures. Every
		// production set_hunting caller stores the C player ID instead.
		if !w.playerHasPlayingDescriptor(player) {
			return
		}
		target = player
		evasion = player.GetSkill("evasion")
	} else if mob := w.GetMobByName(preyName); mob != nil {
		target = mob
	}
	if target == nil {
		if m.CanSpeak() {
			w.npcSay(m, "Damn!  My prey is gone!!")
		}
		m.SetHunting("")
		return
	}
	if target.GetRoom() < 2 {
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

	// Open doors if intelligent mob. C builds "<keyword> <reverse-dir>" and
	// calls the real do_gen_door(SCMD_OPEN) (src/graph.c:320-339), whose
	// NEED_UNLOCKED gate refuses locked doors.
	mobRoom, ok := w.GetRoom(m.GetRoom())
	if ok {
		if exit, exists := mobRoom.Exits[dirKeys[dir]]; exists {
			if exit.ExitInfo&parser.ExitClosed != 0 && mobIsIntelligent(m) && exit.Keywords != "" {
				w.mobGenDoor(m, firstWord(exit.Keywords), scmdOpen)
			}
		}
	}

	// C moves through perform_move(ch, dir, 1) (src/graph.c:340), which
	// refuses closed exits and drags followers; mobPerformMove's silent
	// closed-door crossing let hunters pass locked doors.
	w.performMoveMob(m, dir, true)

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

// intelligentRaces is C's intelligent_races[] (src/constants.c:353-373) as
// the set actually scanned: is_intelligent iterates i < NUM_INTEL_RACES = 18
// (src/utils.c:675-682), so the 19th listed entry, RACE_NPCMINOTAUR (30), is
// past the loop bound and not intelligent in C. Race numbers are the NPC
// RACE_* defines (src/structs.h:131-161).
var intelligentRaces = map[int]bool{
	0:  true, // RACE_NPCHUMAN
	1:  true, // RACE_NPCELF
	2:  true, // RACE_NPCDWARF
	3:  true, // RACE_NPCKENDER
	4:  true, // RACE_NPCCENTAUR
	5:  true, // RACE_NPCRAKSHASA
	6:  true, // RACE_TROLL
	7:  true, // RACE_LYCANTHROPE
	8:  true, // RACE_VAMPIRE
	9:  true, // RACE_UNDEAD
	10: true, // RACE_DRAGON
	11: true, // RACE_DEMON
	18: true, // RACE_GIANT
	19: true, // RACE_DEMIGOD
	20: true, // RACE_OGRE
	27: true, // RACE_HUMANOID
	28: true, // RACE_FAERY
	29: true, // RACE_NPCSSAUR
}

// mobIsIntelligent ports is_intelligent (src/utils.c:675-682): a race check
// against intelligent_races. It is not an affect flag.
func mobIsIntelligent(m *MobInstance) bool {
	if m.Proto() == nil {
		return false
	}
	return intelligentRaces[m.Proto().Race]
}

// attackHuntingPrey ports hunt_victim's arrival attack: C calls
// do_follow(ch, GET_NAME(HUNTING(ch)), 0, 0) and then
// hit(ch, HUNTING(ch), TYPE_UNDEFINED) (src/graph.c:312, 340-345) — the
// swing draws and runs damage()'s gates against the pre-enrolment state
// before anyone enrols. hunt_victim itself refuses to path into the prey's
// ROOM_PEACEFUL/ROOM_HOUSE room earlier (graph.c:300-303).
func (w *World) attackHuntingPrey(m *MobInstance, target combat.Combatant) {
	if err := w.mobHit(m, target); err != nil {
		slog.Warn("mobile prey attack failed", "mob", m.GetName(), "target", target.GetName(), "error", err)
	}
}
