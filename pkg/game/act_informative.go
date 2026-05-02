//nolint:unused // linter false positives — all exported symbols are used across pkg/game
package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// act_informative.go — ported from act.informative.c
// Player-level commands of an informative nature: look, exa, who, score, etc.
// ---------------------------------------------------------------------------

// Affect bit positions (from structs.h AFF_*)
const (
	affBlind      = 0  // AFF_BLIND
	affSenseLife  = 5  // AFF_SENSE_LIFE  Char can sense hidden life
	affInfravision = 10 // AFF_INFRAVISION Char can see in dark
)

// dirList is the canonical direction order.
var dirList = []string{"north", "east", "south", "west", "up", "down"}

// ---------------------------------------------------------------------------
// doLook — ACMD(do_look) — room, target, direction, or "read"
// ---------------------------------------------------------------------------

func splitArg(arg string) (string, string) {
	arg = strings.TrimSpace(arg)
	parts := strings.SplitN(arg, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.TrimSpace(parts[1])
}

func indexOf(list []string, item string) int {
	for i, s := range list {
		if s == item {
			return i
		}
	}
	return -1
}

func chCanSee(ch *Player, target interface{}) bool {
	return !ch.IsAffected(affBlind)
}

// mobCanSee checks whether a mob can see. Uses the mob's AffectFlags for blindness.
func mobCanSee(m *MobInstance) bool {
	if m.Prototype != nil {
		for _, aff := range m.Prototype.AffectFlags {
			if strings.EqualFold(aff, "BLIND") {
				return false
			}
		}
	}
	return true
}

func chCanSeeObj(ch *Player, obj *ObjectInstance) bool {
	ef := obj.Prototype.ExtraFlags
	if len(ef) > 0 && ef[0]&1 != 0 {
		// ITEM_INVISIBLE - immortals can see invisible items
		if ch.Level >= 31 {
			return true
		}
		return false
	}
	return chCanSee(ch, nil)
}

func chCanSeeInDark(ch *Player) bool {
	return ch.IsAffected(affInfravision) || ch.Level >= 31
}

func (w *World) isRoomDark(vnum int) bool {
	room := w.GetRoomInWorld(vnum)
	if room == nil {
		return false
	}
	for _, f := range room.Flags {
		if f == "dark" {
			return true
		}
	}
	return false
}

func (w *World) roomIsDeath(room *parser.Room) bool {
	for _, f := range room.Flags {
		if f == "death" {
			return true
		}
	}
	return false
}

// findCharInRoom finds a character by name in the same room.
// Returns the player and mob — exactly one will be non-nil.
func (w *World) findCharInRoom(ch *Player, roomVNum int, name string) (*Player, *MobInstance) {
	argLower := strings.ToLower(name)
	// Check players first
	for _, p := range w.GetPlayersInRoom(roomVNum) {
		if strings.Contains(strings.ToLower(p.GetName()), argLower) {
			return p, nil
		}
	}
	// Check mobs
	for _, m := range w.GetMobsInRoom(roomVNum) {
		if strings.Contains(strings.ToLower(m.Prototype.ShortDesc), argLower) {
			return nil, m
		}
	}
	return nil, nil
}

// findObjNear finds an object near the player (inventory, equipment, room).
func (w *World) findObjNear(ch *Player, name string) *ObjectInstance {
	argLower := strings.ToLower(name)
	// Check inventory
	for _, item := range ch.Inventory.Items {
		if item != nil && strings.Contains(strings.ToLower(item.Prototype.ShortDesc), argLower) {
			return item
		}
	}
	// Check equipment
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if ok && strings.Contains(strings.ToLower(item.Prototype.ShortDesc), argLower) {
			return item
		}
	}
	// Check room items
	for _, item := range w.roomItems[ch.RoomVNum] {
		if strings.Contains(strings.ToLower(item.Prototype.ShortDesc), argLower) {
			return item
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Under-ported helpers (from act.informative.c / act.wizard.c / spec_procs2.c)
// ---------------------------------------------------------------------------

// FindTargetRoom resolves a target room string to a VNum (from act.wizard.c:184).
func (w *World) FindTargetRoom(ch *Player, raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return -1
	}
	vnum := 0
	if _, err := fmt.Sscanf(raw, "%d", &vnum); err == nil && vnum > 0 {
		if _, ok := w.rooms[vnum]; ok {
			return vnum
		}
		return -1
	}
	lower := strings.ToLower(raw)
	for vnum, room := range w.rooms {
		if room == nil {
			continue
		}
		if strings.Contains(strings.ToLower(room.Name), lower) {
			return vnum
		}
	}
	return -1
}

// PrintObjectLocation formats where an object is (in room, carried, worn, inside another).
func (w *World) PrintObjectLocation(num int, obj *ObjectInstance, ch *Player, recur bool) string {
	var b strings.Builder
	if num > 0 {
		fmt.Fprintf(&b, "O%3d. %-25s - ", num, obj.Prototype.ShortDesc)
	} else {
		fmt.Fprintf(&b, "%33s", " - ")
	}
	switch {
	case obj.RoomVNum > 0:
		if room, ok := w.rooms[obj.RoomVNum]; ok && room != nil {
			fmt.Fprintf(&b, "[%5d] %s\r\n", obj.RoomVNum, room.Name)
		} else {
			fmt.Fprintf(&b, "[%5d] (unknown room)\r\n", obj.RoomVNum)
		}
	case obj.Location.Kind == ObjInInventory || obj.Location.Kind == ObjEquipped:
		name := "someone"
		switch obj.Location.OwnerKind {
		case OwnerPlayer:
			if p, ok := w.players[obj.Location.PlayerName]; ok {
				name = p.GetName()
			}
		case OwnerMob:
			if m, ok := w.activeMobs[obj.Location.MobID]; ok {
				name = m.GetName()
			}
		}
		if obj.Location.Kind == ObjEquipped {
			fmt.Fprintf(&b, "worn by %s\r\n", name)
		} else {
			fmt.Fprintf(&b, "carried by %s\r\n", name)
		}
	case obj.Location.Kind == ObjInContainer:
		if container, ok := w.objectInstances[obj.Location.ContainerObjID]; ok {
			fmt.Fprintf(&b, "inside %s\r\n", container.Prototype.ShortDesc)
		} else {
			b.WriteString("in an unknown container\r\n")
		}
	default:
		b.WriteString("in an unknown location\r\n")
	}
	return b.String()
}

// KenderSteal attempts to pilfer a random item from a mob (from spec_procs2.c:594).
func (w *World) KenderSteal(ch *Player, mob *MobInstance) {
	if mob == nil || len(mob.Inventory) == 0 {
		return
	}
	for _, obj := range mob.Inventory {
		if obj == nil || obj.Prototype == nil || !chCanSeeObj(ch, obj) {
			continue
		}
		if int(w.randPct()%601) >= ch.GetLevel() {
			continue
		}
		percent := int(w.randPct()%100) + 1
		if mob.GetPosition() < posSleeping {
			percent = -1
		}
		if ch.GetLevel() >= lvlImmort {
			percent = 101
		}
		if ch.GetLevel() <= 10 || mob.GetLevel() <= 10 {
			return
		}
		if percent < 0 {
			mob.RemoveFromInventory(obj)
			if err := ch.Inventory.AddItem(obj); err != nil {
				mob.AddToInventory(obj) // restore on failure
				return
			}
			ch.SendMessage("You stealthily filch an item.\r\n")
			return
		}
	}
}

// FindClassBitvector maps a class letter to a bitvector bit for skill/spell filtering.
func FindClassBitvector(arg byte) int64 {
	switch arg {
	case 'm':
		return 1 << 0 // mage
	case 'c':
		return 1 << 1 // cleric
	case 't':
		return 1 << 2 // thief
	case 'w':
		return 1 << 3 // warrior
	case 'a':
		return 1 << 4 // magus
	case 'v':
		return 1 << 5 // avatar
	case 's':
		return 1 << 6 // assassin
	case 'p':
		return 1 << 7 // paladin
	case 'n':
		return 1 << 8 // ninja
	case 'i':
		return 1 << 9 // psionic
	default:
		return 0
	}
}

// randPct returns a simple pseudo-random uint64 for game RNG needs.
func (w *World) randPct() uint64 {
	return uint64(time.Now().UnixNano()) * 6364136223846793005 % (1 << 32)
}
