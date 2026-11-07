package game

import (
	"fmt"
	"testing"
)

// TestSpecAssignmentsHaveLegacyLanesForThePlayerWalk proves the transport
// walk's player lane stayed byte-identical after the SpecRegistry dispatch
// (DP-1474 follow-up): the old GetRoomSpec/GetObjSpecForObject/GetMobSpec
// lookups returned SpecRegistry[name].Legacy (objects could also hit an
// ObjSpecRegistry adapter), so every assigned name must still resolve to a
// non-nil Legacy — otherwise a player silently loses that special. A name the
// old lookup returned nil for was skipped then and stays non-consuming now.
func TestSpecAssignmentsHaveLegacyLanesForThePlayerWalk(t *testing.T) {
	type assigned struct {
		kind string
		name string
	}
	seen := map[string]bool{}
	var order []assigned
	add := func(kind string, table map[int]string) {
		for _, name := range table {
			if !seen[kind+":"+name] {
				seen[kind+":"+name] = true
				order = append(order, assigned{kind: kind, name: name})
			}
		}
	}
	add("room", RoomSpecAssign)
	add("obj", ObjSpecAssign)
	add("mob", MobSpecAssign)

	var exceptions []string
	for _, a := range order {
		if SpecRegistry[a.name].Legacy != nil {
			continue
		}
		// No Legacy: valid only when the old lookup also returned nil. The
		// room/mob lookups were GetSpec(name) — nil then and now. The object
		// lookup could still find an ObjSpecRegistry adapter, which would have
		// run for players before this walk and must be listed.
		if a.kind == "obj" && ObjSpecRegistry[a.name] != nil {
			exceptions = append(exceptions, fmt.Sprintf("%s %q has only an ObjSpecRegistry adapter", a.kind, a.name))
		}
	}
	if len(exceptions) > 0 {
		t.Fatalf("assigned specials without a SpecRegistry Legacy that the old lookups found:\n  %s", exceptions)
	}
}
