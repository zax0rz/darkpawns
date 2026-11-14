package session

import (
	"bytes"
	"encoding/json"
)

// cInputLine applies the character pass of C's process_input to one input
// line (src/comm.c:1965-1982). A backspace removes the previously kept
// character, and only isascii() && isprint() bytes (0x20-0x7E) are copied:
// control characters, ESC sequences, DEL, a decoded telnet IAC (0xFF) and
// every byte of a multi-byte UTF-8 rune are dropped before any command sees
// the line (R1). The `$` doubling C performs in the same loop is handled
// where its consumers expect it and is not repeated here.
func cInputLine(line string) string {
	for i := 0; i < len(line); i++ {
		if c := line[i]; c < 0x20 || c > 0x7e {
			return cInputLineSlow(line)
		}
	}
	return line
}

// maxCInputLine is C's process_input budget: MAX_INPUT_LENGTH-1 kept bytes
// per line (src/structs.h:644, src/comm.c:1971).
const maxCInputLine = 255

// cTruncateInputLine applies C's over-long line rule on top of the character
// pass (src/comm.c:1969-1999): each kept character costs 1 against a 255-byte
// budget and a kept '$' costs 2 (C doubles it into the command buffer); a
// backspace pops the last kept character, restoring its cost. When the
// budget runs out before the line ends, C keeps the processed prefix and
// FIRST sends the player "Line too long.  Truncated to:\r\n<prefix>\r\n" --
// with the '$' doubling visible, exactly as C's buffer holds it -- and then
// runs the prefix as the command. Returns the line to run (never over
// budget, '$' undoubled for Go's consumers) and the notice to send first
// ("" when the line fits, matching C's (space_left <= 0 && ptr < nl_pos)
// condition exactly: a line ending as the budget empties is not too long).
func cTruncateInputLine(line string) (run string, notice string) {
	kept := make([]byte, 0, len(line))
	budget := maxCInputLine
	tooLong := false
	for i := 0; i < len(line); i++ {
		if budget <= 0 {
			// C's loop stops copying but still consumes to the newline;
			// the notice fired when the copy stopped with input remaining.
			tooLong = true
			break
		}
		switch c := line[i]; {
		case c == '\b':
			if len(kept) > 0 {
				last := kept[len(kept)-1]
				kept = kept[:len(kept)-1]
				if last == '$' {
					budget += 2
				} else {
					budget++
				}
			}
		case c >= 0x20 && c <= 0x7e:
			kept = append(kept, c)
			if c == '$' {
				budget -= 2
			} else {
				budget--
			}
		}
	}
	run = string(kept)
	if !tooLong {
		return run, ""
	}
	var doubled bytes.Buffer
	for _, c := range kept {
		doubled.WriteByte(c)
		if c == '$' {
			doubled.WriteByte('$')
		}
	}
	return run, "Line too long.  Truncated to:\r\n" + doubled.String() + "\r\n"
}

func cInputLineSlow(line string) string {
	kept := make([]byte, 0, len(line))
	for i := 0; i < len(line); i++ {
		switch c := line[i]; {
		case c == '\b':
			if len(kept) > 0 {
				kept = kept[:len(kept)-1]
			}
		case c >= 0x20 && c <= 0x7e:
			kept = append(kept, c)
		}
	}
	return string(kept)
}

// cInputJSON applies cInputLine to every string in a structured client
// message's data, so WebSocket JSON clients cannot carry bytes a C
// descriptor could never deliver. Numbers and booleans are preserved.
func cInputJSON(data json.RawMessage) (json.RawMessage, error) {
	if len(data) == 0 {
		return data, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	out, err := json.Marshal(cInputValue(v))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func cInputValue(v any) any {
	switch t := v.(type) {
	case string:
		return cInputLine(t)
	case []any:
		for i := range t {
			t[i] = cInputValue(t[i])
		}
		return t
	case map[string]any:
		for k, e := range t {
			t[k] = cInputValue(e)
		}
		return t
	default:
		return v
	}
}
