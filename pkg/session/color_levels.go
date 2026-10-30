package session

import (
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/screen"
)

// The colour primitives every coloured player-facing surface in this package
// shares: score, stat, report and the group listing. C defines them once in
// src/screen.h and every command reaches for the same macros, so the thresholds
// live in pkg/screen and this file is the thin, package-local spelling of them —
// pkg/game reads the same definitions through its own wrappers.

// ANSI codes from src/screen.h:34-41.
const (
	ansiNormal = screen.KNRM
	ansiRed    = screen.KRED
	ansiGreen  = screen.KGRN
	ansiYellow = screen.KYEL
	ansiCyan   = screen.KCYN
)

// colorLevel is C's _clrlevel(ch).
func colorLevel(flags uint64) int {
	return screen.LevelFromFlags(flags, uint(game.PrfColor1), uint(game.PrfColor2))
}

// hasSparseColor is C's clr(ch, C_SPR): _clrlevel(ch) >= 1.
func hasSparseColor(flags uint64) bool { return screen.SparseOn(colorLevel(flags)) }

// hasNormalColor is C's clr(ch, C_NRM): _clrlevel(ch) >= 2, so PRF_COLOR_2 alone
// ("color normal") is enough.
func hasNormalColor(flags uint64) bool { return screen.NormalOn(colorLevel(flags)) }

// hasCompleteColor is C's clr(ch, C_CMP): _clrlevel(ch) >= 3, so both bits are
// required. It is the package's only C_CMP predicate — score's field pairs, the
// kills row in stat and print_group's whole listing all gate on this level.
func hasCompleteColor(flags uint64) bool { return screen.CompleteOn(colorLevel(flags)) }

// colorCMP returns code when the viewer sits at C_CMP, and the empty string
// otherwise. Callers concatenate the result unconditionally, because below the
// level every C macro expands to nothing.
func colorCMP(p *game.Player, code string) string {
	if p == nil || !hasCompleteColor(p.GetFlags()) {
		return ""
	}
	return code
}

// colorWrap applies C's colour macro pair at the caller's gate.
func colorWrap(value, ansi string, colored bool) string {
	return screen.Wrap(value, ansi, colored)
}
