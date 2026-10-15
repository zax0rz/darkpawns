package session

import "strings"

// cMaxInputLength is MAX_INPUT_LENGTH (src/structs.h:644).
const cMaxInputLength = 256

// applyInputHistory is process_input's history step (src/comm.c:1999-2005),
// run on every line after the character pass and before any consumer. A line
// starting with '!' is replaced whole by last_input; a line starting with '^'
// is perform_subst of last_input and, when it succeeds, becomes last_input;
// any other line, a bare RETURN included, becomes last_input. A failed
// substitution queues nothing: it reports false after C's "Invalid
// substitution." The DP_CLOCK seam consumes its control line before this
// step, so a control never becomes last_input.
func (s *Session) applyInputHistory(line string) (string, bool) {
	switch {
	case strings.HasPrefix(line, "!"):
		return s.lastInput, true
	case strings.HasPrefix(line, "^"):
		out, ok := performSubst(s.lastInput, line)
		if !ok {
			s.Send("Invalid substitution.\r\n")
			return "", false
		}
		s.lastInput = out
		return out, true
	default:
		s.lastInput = line
		return line, true
	}
}

// performSubst is perform_subst (src/comm.c:2037-2080) for "^old^new":
// replace the first occurrence of old in orig with new. A missing second '^'
// or an old that orig does not contain is invalid. strstr finds an empty old
// at position 0, and the result is bounded by C's MAX_INPUT_LENGTH buffer.
func performSubst(orig, subst string) (string, bool) {
	first := subst[1:]
	sep := strings.IndexByte(first, '^')
	if sep < 0 {
		return "", false
	}
	old, repl := first[:sep], first[sep+1:]
	pos := strings.Index(orig, old)
	if pos < 0 {
		return "", false
	}
	out := orig[:pos] + repl + orig[pos+len(old):]
	if len(out) > cMaxInputLength-1 {
		out = out[:cMaxInputLength-1]
	}
	return out, true
}
