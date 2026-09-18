package game

import (
	"sort"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// CloneMob returns an independent mob value suitable for an OLC working copy.
// Mob prototypes are edited off the live world and swapped in atomically only
// when the C MEDIT confirmation accepts them.
func CloneMob(mob parser.Mob) parser.Mob {
	copyMob := mob
	copyMob.ActionFlags = append([]string(nil), mob.ActionFlags...)
	copyMob.AffectFlags = append([]string(nil), mob.AffectFlags...)
	return copyMob
}

// SnapshotMob returns a deep copy of one mob prototype while holding the
// world lock. It is the safe setup boundary for descriptor-owned OLC state.
func (w *World) SnapshotMob(vnum int) (parser.Mob, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	mob, ok := w.mobs[vnum]
	if !ok || mob == nil {
		return parser.Mob{}, false
	}
	return CloneMob(*mob), true
}

// SnapshotMobs returns deep mob-prototype copies from one locked point in
// time. The result is ordered by VNUM because C's mob table and the medit
// disk writer both walk the ascending mob_index[] array.
func (w *World) SnapshotMobs() []parser.Mob {
	w.mu.RLock()
	defer w.mu.RUnlock()

	mobs := make([]parser.Mob, 0, len(w.mobs))
	for _, mob := range w.mobs {
		if mob == nil {
			continue
		}
		mobs = append(mobs, CloneMob(*mob))
	}
	sort.Slice(mobs, func(i, j int) bool { return mobs[i].VNum < mobs[j].VNum })
	return mobs
}

// CommitEditedMob atomically replaces or inserts a mob prototype after
// MEDIT's "save internally" confirmation. The parsed-world copy is updated so
// a restart or a subsequent world export sees the accepted in-memory
// definition.
//
// C's medit_save_internally also shifts mob rnums and rewrites zone M-command
// and shop-keeper references when inserting a new mob. The Go world keys zone
// commands and shop keepers by VNUM directly (parser.ZoneCommand.Arg1,
// shopKeepers), so there is no rnum indirection to repair: inserting under the
// VNUM key is the complete observable effect.
func (w *World) CommitEditedMob(mob parser.Mob) bool {
	mob = CloneMob(mob)
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.mobs == nil {
		w.mobs = make(map[int]*parser.Mob)
	}
	stored := CloneMob(mob)
	w.mobs[mob.VNum] = &stored

	if w.parsedData != nil {
		parsedIndex := -1
		for i := range w.parsedData.Mobs {
			if w.parsedData.Mobs[i].VNum == mob.VNum {
				parsedIndex = i
				break
			}
		}
		if parsedIndex < 0 {
			w.parsedData.Mobs = append(w.parsedData.Mobs, CloneMob(mob))
		} else {
			w.parsedData.Mobs[parsedIndex] = CloneMob(mob)
		}
		// Repoint the live map at the parsed backing array so old snapshots
		// and escaped read pointers stay immutable.
		for i := range w.parsedData.Mobs {
			w.mobs[w.parsedData.Mobs[i].VNum] = &w.parsedData.Mobs[i]
		}
	}
	return true
}

// SetMobScript replaces the live script fields used by MEDIT's script menu.
// C shallow-copies the mob script pointer into OLC (medit.c copy_mobile), so
// script name/flag edits intentionally bypass the working mob and become
// visible before a mob save/quit.
func (w *World) SetMobScript(vnum int, name string, flags int) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	mob, ok := w.mobs[vnum]
	if !ok || mob == nil {
		return false
	}
	updated := CloneMob(*mob)
	updated.ScriptName = name
	updated.LuaFunctions = flags
	stored := updated
	w.mobs[vnum] = &stored
	if w.parsedData != nil {
		for i := range w.parsedData.Mobs {
			if w.parsedData.Mobs[i].VNum == vnum {
				w.parsedData.Mobs[i].ScriptName = name
				w.parsedData.Mobs[i].LuaFunctions = flags
				w.mobs[vnum] = &w.parsedData.Mobs[i]
				break
			}
		}
	}
	return true
}
