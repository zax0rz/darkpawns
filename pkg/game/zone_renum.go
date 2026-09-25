package game

import (
	"log/slog"
	"sort"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// renumZoneTable ports renum_zone_table (db.c:968-1024), which runs once at
// boot after the world, mobiles and objects are indexed. Two things happen to
// every zone reset command:
//
//   - The legacy R format "R <if> <room> <obj vnum> -1" becomes C's current
//     form, arg2 = 1 (object) and arg3 = the vnum. From then on R reads
//     arg2 as the target kind (1 object, 0 mobile) and arg3 as its vnum, as
//     reset_zone, zedit and the webOLC schema all do.
//   - A command naming a room, mobile or object that does not exist is
//     disabled for the life of the server: it becomes '*' and is logged once.
//     A disabled M therefore fails like any other, and C's if-flag chain skips
//     the commands that depend on it.
//
// The world's zone commands are rewritten in place, as C rewrites its zone
// table. zedit, like C's, never saves a '*' command.
func (w *World) renumZoneTable() {
	numbers := make([]int, 0, len(w.zones))
	for number := range w.zones {
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	for _, number := range numbers {
		zone := w.zones[number]
		for index := range zone.Commands {
			w.renumZoneCommand(zone, index)
		}
	}
}

func (w *World) renumZoneCommand(zone *parser.Zone, index int) {
	cmd := &zone.Commands[index]
	room := func(vnum int) bool { _, ok := w.rooms[vnum]; return ok }
	mob := func(vnum int) bool { _, ok := w.mobs[vnum]; return ok }
	obj := func(vnum int) bool { _, ok := w.objs[vnum]; return ok }

	var valid bool
	switch cmd.Command {
	case "M":
		valid = mob(cmd.Arg1) && room(cmd.Arg3)
	case "O":
		valid = obj(cmd.Arg1) && (cmd.Arg3 == -1 || room(cmd.Arg3)) // NOWHERE is allowed
	case "G", "E":
		valid = obj(cmd.Arg1)
	case "P":
		valid = obj(cmd.Arg1) && obj(cmd.Arg3)
	case "L", "D":
		valid = room(cmd.Arg1)
	case "R":
		if cmd.Arg3 == -1 { // legacy: the vnum was arg2, and it was always an object
			cmd.Arg3 = cmd.Arg2
			cmd.Arg2 = 1
		}
		if cmd.Arg2 != 0 {
			valid = room(cmd.Arg1) && obj(cmd.Arg3)
		} else {
			valid = room(cmd.Arg1) && mob(cmd.Arg3)
		}
	default:
		return
	}
	if !valid {
		// log_zone_error(zone, cmd_no, "Invalid vnum, cmd disabled")
		slog.Warn("zone command disabled: invalid vnum",
			"zone", zone.Number, "command", index, "cmd", cmd.Command,
			"arg1", cmd.Arg1, "arg2", cmd.Arg2, "arg3", cmd.Arg3)
		cmd.Command = "*"
	}
}

// destroyMobPossessions is reset_zone's R-mobile prelude (db.c:2233-2238):
// everything the mobile wears, slot by slot, then everything it carries, is
// extracted rather than dropped, so extract_char leaves nothing behind.
func (w *World) destroyMobPossessions(mob *MobInstance) {
	if mob == nil {
		return
	}
	mob.mu.RLock()
	var items []*ObjectInstance
	for slot := 0; slot < NumWears; slot++ {
		if obj := mob.Equipment[slot]; obj != nil {
			items = append(items, obj)
		}
	}
	items = append(items, mob.Inventory...)
	mob.mu.RUnlock()
	for _, obj := range items {
		w.ExtractObject(obj, -1)
	}
}
