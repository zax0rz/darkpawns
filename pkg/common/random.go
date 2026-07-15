package common

import "github.com/zax0rz/darkpawns/pkg/dprng"

// Number returns a random integer in [min, max] inclusive.
// Equivalent to C's number(min, max) from utils.c.
func Number(min, max int) int {
	return dprng.Number(min, max)
}
