package game

import "testing"

// TestSanaUsesRuntimeKeywords verifies sana() consults GetKeywords() so
// synthetic objects (corpses, money) without a Prototype get the right
// article instead of always defaulting to "a".
func TestSanaUsesRuntimeKeywords(t *testing.T) {
	if got := sana(nil); got != "a" {
		t.Errorf("nil object: got %q, want %q", got, "a")
	}

	synthetic := &ObjectInstance{Runtime: ObjectRuntimeState{Keywords: "apple"}}
	if got := sana(synthetic); got != "an" {
		t.Errorf("synthetic object starting with a vowel: got %q, want %q", got, "an")
	}

	// GetKeywords() falls back to "object generic" for a totally bare object
	// (no Runtime.Keywords, no Prototype) — "object" starts with a vowel.
	bare := &ObjectInstance{}
	if got := sana(bare); got != "an" {
		t.Errorf("object with no keywords at all: got %q, want %q", got, "an")
	}
}

// TestObjNameUsesRuntimeKeywords verifies objName() falls back to
// GetKeywords() for synthetic, Prototype-less objects instead of always
// returning "something".
func TestObjNameUsesRuntimeKeywords(t *testing.T) {
	observer := NewPlayer(1, "Observer", 1001)

	synthetic := &ObjectInstance{Runtime: ObjectRuntimeState{Keywords: "corpse goblin"}}
	if got := objName(synthetic, observer); got != "corpse" {
		t.Errorf("got %q, want first keyword %q", got, "corpse")
	}

	if got := objName(nil, observer); got != "something" {
		t.Errorf("nil object: got %q, want %q", got, "something")
	}
}
