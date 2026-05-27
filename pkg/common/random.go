package common

import "math/rand/v2"

// Number returns a random integer in [min, max] inclusive.
// Equivalent to C's number(min, max) from utils.c.
func Number(min, max int) int {
	if max <= min {
		return min
	}
	return min + rand.IntN(max-min+1)
}
