// Package screen holds C's screen.h colour law in one place: the K* ANSI codes
// and the COLOR_LEV thresholds every CC* macro gates on (src/screen.h:32-56).
//
// C defines these once and every command reaches for them. The port had
// re-derived the arithmetic per package, which is how two gate bugs survived
// review — do_stat_character's kills row and print_group's whole listing both
// used the wrong level (docs/fidelity/audit-reports/ansi-color-audit.md §11, §12).
// pkg/game and pkg/session now both take their colour decisions from here.
package screen

// C's raw colour bytes (src/screen.h:34-41).
const (
	KNRM = "\x1b[0m"
	KRED = "\x1b[31m"
	KGRN = "\x1b[32m"
	KYEL = "\x1b[33m"
	KBLU = "\x1b[34m"
	KMAG = "\x1b[35m"
	KCYN = "\x1b[36m"
	KWHT = "\x1b[37m"
)

// C's colour levels (src/screen.h:41-44).
const (
	Off      = 0 // C_OFF
	Sparse   = 1 // C_SPR
	Normal   = 2 // C_NRM
	Complete = 3 // C_CMP
)

// Level is C's _clrlevel(ch): PRF_COLOR_1 counts 1 and PRF_COLOR_2 counts 2
// (src/screen.h:45-47), giving the four levels C's `color` command names off,
// sparse, normal and complete.
func Level(color1, color2 bool) int {
	level := Off
	if color1 {
		level++
	}
	if color2 {
		level += 2
	}
	return level
}

// LevelFromFlags reads the two PRF_COLOR bits out of a preference word and
// reports the level. Callers pass their own bit indices because the flag
// constants live in the package that owns the player model.
func LevelFromFlags(flags uint64, color1Bit, color2Bit uint) int {
	return Level(flags&(1<<color1Bit) != 0, flags&(1<<color2Bit) != 0)
}

// Sparse reports C's clr(ch, C_SPR): _clrlevel(ch) >= 1, so either bit is
// enough. It is the lowest gate C uses, and the one do_who's CCYEL sites read.
func SparseOn(level int) bool { return level >= Sparse }

// NormalOn reports C's clr(ch, C_NRM): _clrlevel(ch) >= 2, so PRF_COLOR_2 alone
// ("color normal") is enough.
func NormalOn(level int) bool { return level >= Normal }

// CompleteOn reports C's clr(ch, C_CMP): _clrlevel(ch) >= 3, so both bits are
// required. score's field pairs, the kills row in stat and print_group's whole
// listing all gate on this level.
func CompleteOn(level int) bool { return level >= Complete }

// Wrap applies C's colour macro pair — CCCYN(ch, C_NRM) value CCNRM(ch, C_NRM) —
// or returns the value untouched when the caller's gate is closed
// (src/screen.h:45-56). Callers format padding before the call so the field
// width lands inside the colour, exactly as C's sprintf does.
func Wrap(value, code string, on bool) string {
	if !on {
		return value
	}
	return code + value + KNRM
}
