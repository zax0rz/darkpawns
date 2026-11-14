package game

import (
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// CloneObj returns an independent object value suitable for an OLC working
// copy. Object prototypes are edited off the live world and swapped in
// atomically only when the C OEDIT confirmation accepts them.
func CloneObj(obj parser.Obj) parser.Obj {
	clone := obj
	clone.Affects = append([]parser.ObjAffect(nil), obj.Affects...)
	clone.ExtraDescs = append([]parser.ExtraDesc(nil), obj.ExtraDescs...)
	return clone
}

// SnapshotObj returns a deep copy of one object prototype while holding the
// world lock. It is the safe setup boundary for descriptor-owned OLC state.
func (w *World) SnapshotObj(vnum int) (parser.Obj, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	obj, ok := w.objs[vnum]
	if !ok || obj == nil {
		return parser.Obj{}, false
	}
	return CloneObj(*obj), true
}

// SnapshotObjs returns deep object-prototype copies from one locked point in
// time. The result is ordered by VNUM because C's object table and the oedit
// disk writer both walk the ascending obj_index[] array.
func (w *World) SnapshotObjs() []parser.Obj {
	w.mu.RLock()
	defer w.mu.RUnlock()

	objs := make([]parser.Obj, 0, len(w.objs))
	for _, obj := range w.objs {
		if obj == nil {
			continue
		}
		objs = append(objs, CloneObj(*obj))
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].VNum < objs[j].VNum })
	return objs
}

// CommitEditedObj atomically replaces or inserts an object prototype after
// OEDIT's "save internally" confirmation. The parsed-world copy is updated so
// a restart or a subsequent world export sees the accepted in-memory
// definition.
//
// C's oedit_save_internally also shifts object rnums and rewrites zone
// P/O/G/E/R commands, notice boards, and shop products when inserting a new
// object. The Go world keys zone commands and shop products by VNUM directly
// (parser.ZoneCommand.Arg1/Arg3, shop product lists), and the board subsystem
// stores no object prototype reference, so there is no rnum indirection to
// repair: inserting under the VNUM key is the complete observable effect.
// (See docs/fidelity/depth/oedit.tsv for the audited consumers.)
func (w *World) CommitEditedObj(obj parser.Obj) bool {
	obj = CloneObj(obj)
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.objs == nil {
		w.objs = make(map[int]*parser.Obj)
	}
	stored := CloneObj(obj)
	w.objs[obj.VNum] = &stored

	if w.parsedData != nil {
		if existing, ok := w.parsedData.ObjByVnum(obj.VNum); ok {
			// Same element the first-match scan found; writing through the
			// index pointer is the slice assignment it performed.
			*existing = CloneObj(obj)
		} else {
			w.parsedData.Objs = append(w.parsedData.Objs, CloneObj(obj))
			// The append may have reallocated the backing array; every index
			// pointer into it is stale.
			w.parsedData.RebuildObjIndex()
		}
		// Repoint the live map at the parsed backing array so old snapshots
		// and escaped read pointers stay immutable.
		for i := range w.parsedData.Objs {
			w.objs[w.parsedData.Objs[i].VNum] = &w.parsedData.Objs[i]
		}
	}
	return true
}

// SetLiveObjExtraDesc applies do_string's field-4 write to an extra description
// the prototype already carries (src/modify.c:701-707).
//
// C's read_object copies the obj_data struct shallowly (src/db.c:1947-1951), so
// the instance's ex_description pointer IS the prototype's list: freeing the
// node's description and pointing d->str at it edits the prototype, and every
// instance sees the new text until the process restarts (the player object file
// stores no extra descriptions; src/structs.h:770-786). The port keeps
// prototypes immutable, so this publishes a clone and patches the live
// instances' atomically published lists instead of writing the shared node.
//
// It returns false when the prototype has no such keyword, and the caller then
// keeps C's instance-local new-node path (src/modify.c:690-700).
func (w *World) SetLiveObjExtraDesc(vnum int, keyword, text string) bool {
	return w.mutateLiveObjExtraDesc(vnum, keyword, func(desc *parser.ExtraDesc) {
		desc.Description = text
		desc.NullDescription = false
	})
}

// ClearLiveObjExtraDesc is the editor-entry half of the same write: C frees the
// node's description and leaves the pointer NULL (src/modify.c:702-704), so a
// look during the edit misses the keyword, exactly as find_exdesc's NULL return
// does (src/act.informative.c:987-996).
func (w *World) ClearLiveObjExtraDesc(vnum int, keyword string) bool {
	return w.mutateLiveObjExtraDesc(vnum, keyword, func(desc *parser.ExtraDesc) {
		desc.Description = ""
		desc.NullDescription = true
	})
}

// mutateLiveObjExtraDesc publishes mutate applied to the prototype entry whose
// keyword matches, then to every live instance of that prototype.
//
// Two things it deliberately does not do: rewrite inst.Prototype (readers hold
// that pointer without a lock, so the instance's own published list carries the
// change instead), and touch parsedData (C never writes the zone file here; a
// later oedit save of the zone serialises World.objs and so keeps the edit as a
// side effect, as C's oedit_save_to_disk does).
func (w *World) mutateLiveObjExtraDesc(vnum int, keyword string, mutate func(*parser.ExtraDesc)) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	proto, ok := w.objs[vnum]
	if !ok || proto == nil {
		return false
	}
	// C compares the whole keyword string with str_cmp (src/utils.c:107), not
	// token-wise like isname.
	index := -1
	for i := range proto.ExtraDescs {
		if strings.EqualFold(proto.ExtraDescs[i].Keywords, keyword) {
			index = i
			break
		}
	}
	if index < 0 {
		return false
	}

	updated := CloneObj(*proto)
	mutate(&updated.ExtraDescs[index])
	stored := updated
	w.objs[vnum] = &stored

	for _, inst := range w.objectInstances {
		if inst == nil || inst.VNum != vnum {
			continue
		}
		// A copy: the published slice is what concurrent readers load.
		descs := inst.LiveExtraDescs()
		patched := false
		for i := range descs {
			if strings.EqualFold(descs[i].Keywords, keyword) {
				mutate(&descs[i])
				patched = true
				break
			}
		}
		if patched {
			inst.publishExtraDescs(descs)
		}
	}
	return true
}

// RefreshLiveObjInstances swaps every live instance of vnum onto the edited
// prototype, mirroring oedit_save_internally's object_list sweep (src/oedit.c):
// the full obj_data struct is replaced while the runtime placement fields
// (in_room, carried_by, worn_on, contains, next) are preserved.
//
// In the Go model those placement fields live on ObjectInstance (Location,
// RoomVNum, Contains, Runtime), not on the prototype, so repointing Prototype
// preserves them for free. The instance-level override fields, however, stand
// in for C struct fields that a whole-struct assignment WOULD overwrite
// (obj_flags/affected, e.g. spell enchantment or a drink container's liquid
// weight), so they are cleared here to match C.
//
// Two Go-only runtime fields have no C analogue and are intentionally left
// alone: ObjectInstance.Timer (parser.Obj has no timer field, so C's
// obj_flags.timer cannot be carried — see the oedit depth manifest) and
// Runtime.* (synthetic corpse/money identity that C has no struct slot for).
func (w *World) RefreshLiveObjInstances(vnum int, obj parser.Obj) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, inst := range w.objectInstances {
		if inst == nil || inst.VNum != vnum {
			continue
		}
		clone := CloneObj(obj)
		inst.Prototype = &clone
		// C's *obj = obj_proto[nr] discards the instance's own ex_description
		// edits (the struct assignment overwrites the pointer), so the
		// published list goes with the other instance-level overrides below.
		inst.clearLiveExtraDescs()
		inst.ResetExtraFlags()
		inst.AffectsOverride = nil
		inst.ValuesOverride = nil
		inst.TypeFlagOverride = nil
		inst.WeightOverride = nil
	}
}

// SetObjScript replaces the live script fields used by OEDIT's script menu.
// C resolves GET_OBJ_SCRIPT/OBJ_SCRIPT_FLAGS through obj_index[rnum], so
// script name/flag edits bypass the working prototype and become visible
// before an object save/quit.
func (w *World) SetObjScript(vnum int, name string, flags int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	obj, ok := w.objs[vnum]
	if !ok || obj == nil {
		return false
	}
	updated := CloneObj(*obj)
	updated.ScriptName = name
	updated.LuaFunctions = flags
	stored := updated
	w.objs[vnum] = &stored
	if w.parsedData != nil {
		if existing, ok := w.parsedData.ObjByVnum(vnum); ok {
			existing.ScriptName = name
			existing.LuaFunctions = flags
			w.objs[vnum] = existing
		}
	}
	return true
}
