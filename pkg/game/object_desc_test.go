package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestGetShortDescRuntimeFirst verifies GetShortDesc checks Runtime fields
// before falling back to the prototype, so synthetic objects (corpses, money
// piles) with a nil Prototype render their real description instead of the
// generic placeholder.
func TestGetShortDescRuntimeFirst(t *testing.T) {
	synthetic := &ObjectInstance{Runtime: ObjectRuntimeState{ShortDesc: "the corpse of a rat"}}
	if got := synthetic.GetShortDesc(); got != "the corpse of a rat" {
		t.Errorf("got %q, want Runtime.ShortDesc", got)
	}

	bare := &ObjectInstance{}
	if got := bare.GetShortDesc(); got != "a generic object" {
		t.Errorf("got %q, want generic fallback", got)
	}

	withProto := NewObjectInstance(&parser.Obj{ShortDesc: "a steel sword"}, -1)
	if got := withProto.GetShortDesc(); got != "a steel sword" {
		t.Errorf("got %q, want prototype short desc", got)
	}

	withProto.Runtime.ShortDesc = "a renamed sword"
	if got := withProto.GetShortDesc(); got != "a renamed sword" {
		t.Errorf("got %q, want Runtime.ShortDesc over prototype", got)
	}

	// ShortDescOverride wins even over Runtime.ShortDesc and the prototype.
	withProto.Runtime.ShortDescOverride = "a slab of carved meat"
	if got := withProto.GetShortDesc(); got != "a slab of carved meat" {
		t.Errorf("got %q, want ShortDescOverride", got)
	}
}

// TestGetLongDescRuntimeFirst verifies GetLongDesc checks Runtime.LongDesc
// before falling back to the prototype.
func TestGetLongDescRuntimeFirst(t *testing.T) {
	synthetic := &ObjectInstance{Runtime: ObjectRuntimeState{LongDesc: "A bloated corpse lies here."}}
	if got := synthetic.GetLongDesc(); got != "A bloated corpse lies here." {
		t.Errorf("got %q, want Runtime.LongDesc", got)
	}

	bare := &ObjectInstance{}
	if got := bare.GetLongDesc(); got != "A generic object lies here." {
		t.Errorf("got %q, want generic fallback", got)
	}

	withProto := NewObjectInstance(&parser.Obj{LongDesc: "A sword lies here."}, -1)
	if got := withProto.GetLongDesc(); got != "A sword lies here." {
		t.Errorf("got %q, want prototype long desc", got)
	}

	withProto.Runtime.LongDesc = "A renamed sword lies here."
	if got := withProto.GetLongDesc(); got != "A renamed sword lies here." {
		t.Errorf("got %q, want Runtime.LongDesc over prototype", got)
	}
}
