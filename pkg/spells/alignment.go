package spells

// C's alignment classes (src/utils.h:454-455): IS_GOOD is GET_ALIGNMENT >= 350,
// IS_EVIL is GET_ALIGNMENT <= -350, and everything between is neutral. Every
// alignment gate in the spell code and its Lua/scripting mirrors must use these
// thresholds, not the sign of the value.
const (
	goodAlignThreshold = 350
	evilAlignThreshold = -350
)

// IsGoodAlign reports C's IS_GOOD for a raw alignment value.
func IsGoodAlign(align int) bool {
	return align >= goodAlignThreshold
}

// IsEvilAlign reports C's IS_EVIL for a raw alignment value.
func IsEvilAlign(align int) bool {
	return align <= evilAlignThreshold
}
