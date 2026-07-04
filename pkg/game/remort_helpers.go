package game

// number is an alias for randRange for C compatibility.
func number(from, to int) int {
	return randRange(from, to)
}
