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
