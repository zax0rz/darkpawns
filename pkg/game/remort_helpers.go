package game

// number returns a uniform random integer in [from, to] inclusive, matching
// C's number(from, to). Use this for C-ported probability gates; use randN(n)
// for array indexing where an exclusive upper bound is required.
func number(from, to int) int {
	return randRange(from, to)
}
