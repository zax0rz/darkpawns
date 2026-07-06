package game

import "math"

// safeInt32 converts an int to int32 with overflow protection.
// Kept in pkg/game for mail.go; pkg/boards maintains its own copy to avoid
// a circular dependency during the boards leaf extraction.
func safeInt32(v int) int32 {
	if v < math.MinInt32 || v > math.MaxInt32 {
		return 0
	}
	return int32(v)
}
