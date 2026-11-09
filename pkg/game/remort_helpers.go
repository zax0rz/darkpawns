package game

// number returns a uniform random integer in [from, to] inclusive, matching
// C's number(from, to). Use this for C-ported probability gates; array
// indexing must subtract one to an exclusive bound, as C's number does.
func number(from, to int) int {
	return randRange(from, to)
}
