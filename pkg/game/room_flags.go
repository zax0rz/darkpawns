package game

import "strconv"

// roomHasFlagBit checks if a room's hex flag array has a specific bit set.
func roomHasFlagBit(flags []string, flagBit int) bool {
	if len(flags) < 1 {
		return false
	}
	word := flagBit / 32
	bit := flagBit % 32
	if word >= len(flags) {
		return false
	}
	val, err := strconv.ParseUint(flags[word], 16, 32)
	if err != nil {
		return false
	}
	return val&(1<<uint(bit)) != 0
}

// hasWearFlag checks if a [4]int wear flags array has a specific bit set.
func hasWearFlag(wf [4]int, bit int) bool {
	word := bit / 32
	b := bit % 32
	if word >= 4 {
		return false
	}
	return wf[word]&(1<<uint(b)) != 0
}
