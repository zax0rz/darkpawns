package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestResolvePlayerWorldRefusesInvisible pins get_player_vis's CAN_SEE gate
// (src/handler.c:1263-1276) for the wizard target-resolution batch: an
// AFF_INVISIBLE player is not returned even by exact name, matching C's
// "No such player around." arms. Fails if the resolver skips the visibility
// check or callers fall back to an ungated session lookup (R5h).
func TestResolvePlayerWorldRefusesInvisible(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Vault"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	w.MessageSink = func(string, []byte) {}

	god := NewPlayer(1, "Seergod", 1001)
	god.Level = 34
	if err := w.AddPlayer(god); err != nil {
		t.Fatalf("AddPlayer god: %v", err)
	}
	peer := NewPlayer(2, "Hiddenone", 1001)
	peer.Level = 1
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	// Spell invisibility on the peer; the god has no detect invisibility.
	peer.AddAffect(engine.NewAffectDirect(0, engine.ApplyNone, 1, 0, engine.AFFInvisible, "test"))

	if !peer.IsAffected(AffInvisible) {
		t.Fatal("fixture: peer is not AFF_INVISIBLE")
	}
	if _, ok := w.ResolvePlayerWorld(god, "Hiddenone"); ok {
		t.Fatal("ResolvePlayerWorld returned an AFF_INVISIBLE player to a god without detect invisibility")
	}
	if _, ok := w.ResolveCharWorld(god, "Hiddenone"); ok {
		t.Fatal("ResolveCharWorld returned an AFF_INVISIBLE player to a god without detect invisibility")
	}

	// A visible candidate of the same name is still found.
	peer.RemoveAffectBySpell(0)
	if peer.IsAffected(AffInvisible) {
		t.Fatal("fixture: RemoveAffectBySpell did not clear AFF_INVISIBLE")
	}
	if _, ok := w.ResolvePlayerWorld(god, "Hiddenone"); !ok {
		t.Fatal("ResolvePlayerWorld refused a visible exact-name player")
	}
}
