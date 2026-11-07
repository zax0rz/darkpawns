package common

import "strings"

// SprintbitArray mirrors C's sprintbitarray and its signed-int-to-64-bit cast.
func SprintbitArray(bitvector []uint32, names []string, maxar int) string {
	var b strings.Builder
	for i := 0; i < maxar; i++ {
		tmp := sprintnbit(uint64(int64(int32(bitvector[i]))), names, i*32) // #nosec G115 -- src/utils.c:835 sign-extends each int word.
		if tmp != "NOBITS " {
			b.WriteString(tmp)
		}
	}
	if b.Len() == 0 {
		return "NOBITS "
	}
	return b.String()
}

func sprintnbit(bitvector uint64, names []string, bitOffset int) string {
	var b strings.Builder
	nr := bitOffset
	for bv := bitvector; bv != 0; bv >>= 1 {
		if bv&1 != 0 {
			if nr < len(names) && names[nr] != "" {
				b.WriteString(names[nr])
				b.WriteByte(' ')
			} else {
				b.WriteString("UNDEFINED ")
			}
		}
		if nr < len(names) && names[nr] != "" {
			nr++
		}
	}
	if b.Len() == 0 {
		return "NOBITS "
	}
	return b.String()
}
