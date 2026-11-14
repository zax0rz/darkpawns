package game

import "github.com/zax0rz/darkpawns/pkg/parser"

// Live string surfaces for do_string (src/modify.c:594-772).
//
// C's do_string writes straight into the live char_data / obj_data string
// fields (mob->player.name, obj->short_description, ed->description, …), so the
// edit is visible to every reader immediately and dies with the instance: a mob
// reverts on its next zone reset and on reboot, while an object carries the new
// text with it for the rest of its life. Nothing is written to the world files.
//
// Go keeps mob and object prototypes immutable and shared between instances, so
// each helper below lands the edit on exactly one instance using the mechanism
// the port already uses for instance-local strings:
//
//   - mobs clone their prototype snapshot and swap it in atomically
//     (MobInstance.SetProto), the same shape RefreshLiveMobStrings uses;
//   - objects use the typed Runtime string overrides that corpses, money piles,
//     drink containers and worn name-takers already write;
//   - object extra descriptions take a per-instance snapshot
//     (Runtime.ExtraDescs / ExtraDescsLive) because C's delete path removes a
//     node from one object's list.

// setLiveProtoString clones the instance's prototype snapshot, applies mutate to
// the clone, and swaps the clone in. The shared prototype is never mutated.
func (m *MobInstance) setLiveProtoString(mutate func(*parser.Mob)) {
	if proto := m.Proto(); proto != nil {
		clone := *proto
		mutate(&clone)
		m.SetProto(&clone)
	}
}

// SetLiveKeywords applies do_string field 1 to one mob instance
// (C: &mob->player.name).
func (m *MobInstance) SetLiveKeywords(value string) {
	m.setLiveProtoString(func(clone *parser.Mob) { clone.Keywords = value })
}

// SetLiveShortDesc applies do_string field 2 to one mob instance
// (C: &mob->player.short_descr).
func (m *MobInstance) SetLiveShortDesc(value string) {
	m.setLiveProtoString(func(clone *parser.Mob) { clone.ShortDesc = value })
}

// SetLiveLongDesc applies do_string field 3 to one mob instance
// (C: &mob->player.long_descr).
func (m *MobInstance) SetLiveLongDesc(value string) {
	m.setLiveProtoString(func(clone *parser.Mob) { clone.LongDesc = value })
}

// SetLiveDetailedDesc applies do_string field 4 to one mob instance
// (C: &mob->player.description).
func (m *MobInstance) SetLiveDetailedDesc(value string) {
	m.setLiveProtoString(func(clone *parser.Mob) { clone.DetailedDesc = value })
}

// SetLiveKeywords applies do_string field 1 to one object
// (C: &obj->name).
func (o *ObjectInstance) SetLiveKeywords(value string) {
	o.Runtime.Keywords = value
}

// SetLiveShortDesc applies do_string field 2 to one object
// (C: &obj->short_description). The worn name-taker override shares C's single
// short_description field, so an explicit write clears it: C's last writer wins.
func (o *ObjectInstance) SetLiveShortDesc(value string) {
	o.Runtime.ShortDesc = value
	o.Runtime.ShortDescOverride = ""
}

// SetLiveLongDesc applies do_string field 3 to one object
// (C: &obj->long_description).
func (o *ObjectInstance) SetLiveLongDesc(value string) {
	o.Runtime.LongDesc = value
}

// LiveExtraDescs returns the instance's current extra-description list: the
// published snapshot once do_string or a prototype write-through has touched it,
// otherwise the effective list (prototype entries plus Lua-added runtime
// entries). The returned slice is always a copy, so callers mutate and write it
// back with SetLiveExtraDescs.
func (o *ObjectInstance) LiveExtraDescs() []parser.ExtraDesc {
	return append([]parser.ExtraDesc(nil), o.GetExtraDescs()...)
}

// SetLiveExtraDescs replaces the instance's extra-description list. The first
// call turns the published snapshot on, after which GetExtraDescs serves this
// list instead of the prototype's.
func (o *ObjectInstance) SetLiveExtraDescs(descs []parser.ExtraDesc) {
	o.publishExtraDescs(descs)
}

// publishExtraDescs stores a fresh copy as the instance's published list and
// keeps the Runtime fields in step with it. The atomic publication is what
// readers load, so a writer in one player's goroutine never races a look in
// another's.
func (o *ObjectInstance) publishExtraDescs(descs []parser.ExtraDesc) {
	stored := append([]parser.ExtraDesc(nil), descs...)
	o.Runtime.ExtraDescs = stored
	o.Runtime.ExtraDescsLive = true
	o.liveExtraDescs.Store(&stored)
}

// clearLiveExtraDescs drops the published list so the instance reads its
// prototype again. C's oedit_save_internally replaces each live instance with a
// whole-struct copy of the prototype (*obj = obj_proto[nr]), which discards any
// instance-local ex_description edits.
func (o *ObjectInstance) clearLiveExtraDescs() {
	o.Runtime.ExtraDescs = nil
	o.Runtime.ExtraDescsLive = false
	o.liveExtraDescs.Store(nil)
}

// FindExtraDescIndex mirrors find_exdesc's scan (src/act.informative.c:987-996):
// the first list entry whose keyword namelist matches word via
// isname_with_abbrevs. It returns -1 when nothing matches. C returns the entry's
// description pointer, so callers that care about C's `!find_exdesc` test must
// additionally treat a NULL description (NullDescription) as "not found".
func FindExtraDescIndex(descs []parser.ExtraDesc, word string) int {
	for i := range descs {
		if isnameWithAbbrevs(word, descs[i].Keywords) {
			return i
		}
	}
	return -1
}

// IsNameWithAbbrevs exposes C's isname_with_abbrevs (src/handler.c:115-134) to
// command packages. do_string's delete-description path uses it twice: once
// through find_exdesc, and once directly against the list head.
func IsNameWithAbbrevs(str, namelist string) bool {
	return isnameWithAbbrevs(str, namelist)
}
