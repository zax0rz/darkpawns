package game

// charm.go holds the game-layer charm query that pkg/spells delegates to.
//
// pkg/spells only knows engine's AFF_* masks (e.g. engine.AFFCharm == 1<<10),
// but pkg/game stores affects under a *different* internal bit layout
// (affCharm == 21, pkg/game/affects_constants.go). A spell-layer
// IsAffected(int(engine.AFFCharm)) therefore checks the wrong bit and is
// effectively always false. Because pkg/spells cannot import pkg/game (import
// cycle), the check is delegated here — the same pattern as CanRaiseUndeadI
// (DP-1008).

// affectBitChecker is satisfied by both *Player and *MobInstance; its argument
// is an internal affect bit index (not an engine AFF mask).
type affectBitChecker interface {
	IsAffected(bit int) bool
}

// IsCharmedI reports whether ch is currently affected by AFF_CHARM, using the
// correct internal charm bit index. It lives on *World so pkg/spells can call
// it through a local interface without importing pkg/game.
//
// C source: src/magic.c mag_areas and the mass-effect loops skip charmed NPCs:
//
//	if (IS_NPC(tch) && IS_AFFECTED(tch, AFF_CHARM)) continue;
//
// The IS_NPC gate stays in the spell layer; this method only answers the
// AFF_CHARM question. Returns false for unrecognized character types.
func (w *World) IsCharmedI(ch interface{}) bool {
	c, ok := ch.(affectBitChecker)
	if !ok {
		return false
	}
	return c.IsAffected(affCharm)
}
