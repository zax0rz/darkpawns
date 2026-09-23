package stringcensus

import (
	"strconv"
	"strings"
)

// clex.go holds a deliberately small C lexer. The census does not parse C; it
// needs three things only: string literals with their source positions and
// escaping resolved, the shape of call argument lists, and the enclosing
// function of each literal. Anything the lexer does not care about (operators,
// types, control flow) is emitted as a single punctuation or identifier token.

// cTokKind classifies a token.
type cTokKind int

const (
	cTokString cTokKind = iota
	cTokChar
	cTokIdent
	cTokNumber
	cTokPunct
)

// cTok is one C token. text holds raw source text, so a string token includes
// its quotes and escapes; callers unescape with unescapeC.
type cTok struct {
	kind cTokKind
	text string
	line int
}

// lexC tokenizes a C translation unit. Comments are dropped, preprocessor
// directives are skipped whole (a #define body is a macro definition, not a
// string any call site passes), and both string and character literals are
// scanned with escape awareness so an escaped quote does not end them.
func lexC(src string) []cTok {
	toks := make([]cTok, 0, len(src)/6)
	line := 1
	atLineStart := true
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == '\n':
			line++
			i++
			atLineStart = true
		case c == ' ' || c == '\t' || c == '\r' || c == '\v' || c == '\f':
			i++
		case c == '/' && strings.HasPrefix(src[i:], "//"):
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '/' && strings.HasPrefix(src[i:], "/*"):
			i += 2
			for i < len(src) && !strings.HasPrefix(src[i:], "*/") {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			i = min(i+2, len(src))
		case c == '#' && atLineStart:
			i = skipDirective(src, i, &line)
		case c == '"' || c == '\'':
			raw, n, newlines := scanCQuoted(src[i:], c)
			kind := cTokString
			if c == '\'' {
				kind = cTokChar
			}
			toks = append(toks, cTok{kind: kind, text: raw, line: line})
			line += newlines
			i += n
			atLineStart = false
		case isIdentStart(c):
			j := i + 1
			for j < len(src) && isIdentByte(src[j]) {
				j++
			}
			toks = append(toks, cTok{kind: cTokIdent, text: src[i:j], line: line})
			i = j
			atLineStart = false
		case c >= '0' && c <= '9':
			j := i + 1
			for j < len(src) && (isIdentByte(src[j]) || src[j] == '.') {
				j++
			}
			toks = append(toks, cTok{kind: cTokNumber, text: src[i:j], line: line})
			i = j
			atLineStart = false
		default:
			toks = append(toks, cTok{kind: cTokPunct, text: src[i : i+1], line: line})
			i++
			atLineStart = false
		}
	}
	return toks
}

// scanCQuoted consumes a "..." or '...' literal at the start of src and returns
// the raw text (quotes included), its length, and how many newlines it spans.
// An unterminated literal consumes the rest of the file rather than looping,
// so a malformed oracle cannot hang the census.
func scanCQuoted(src string, quote byte) (raw string, length, newlines int) {
	i := 1
	for i < len(src) {
		switch src[i] {
		case '\\':
			if i+1 < len(src) && src[i+1] == '\n' {
				newlines++
			}
			i += 2
			continue
		case '\n':
			newlines++
		case quote:
			return src[:i+1], i + 1, newlines
		}
		i++
	}
	return src, len(src), newlines
}

// skipDirective advances past a preprocessor line, honouring backslash line
// continuations so a multi-line #define body is not mistaken for code.
func skipDirective(src string, i int, line *int) int {
	for i < len(src) {
		switch src[i] {
		case '\n':
			return i
		case '\\':
			i++
			if i < len(src) && src[i] == '\r' {
				i++
			}
			if i < len(src) && src[i] == '\n' {
				*line++
				i++
			}
		default:
			i++
		}
	}
	return i
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentByte(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

// unescapeC resolves C string escapes. Unknown escapes keep the escaped byte,
// which is what C compilers do, and a malformed tail fails the literal so the
// caller can skip it rather than invent text.
func unescapeC(raw string) (string, bool) {
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return "", false
	}
	body := raw[1 : len(raw)-1]
	if !strings.ContainsRune(body, '\\') {
		return body, true
	}
	var b strings.Builder
	b.Grow(len(body))
	for i := 0; i < len(body); {
		if body[i] != '\\' {
			b.WriteByte(body[i])
			i++
			continue
		}
		i++
		if i >= len(body) {
			return "", false
		}
		switch c := body[i]; c {
		case 'n':
			b.WriteByte('\n')
			i++
		case 't':
			b.WriteByte('\t')
			i++
		case 'r':
			b.WriteByte('\r')
			i++
		case 'a':
			b.WriteByte(0x07)
			i++
		case 'b':
			b.WriteByte(0x08)
			i++
		case 'f':
			b.WriteByte(0x0c)
			i++
		case 'v':
			b.WriteByte(0x0b)
			i++
		case 'x':
			j := i + 1
			for j < len(body) && isHexByte(body[j]) {
				j++
			}
			if j == i+1 {
				return "", false
			}
			v, err := strconv.ParseUint(body[i+1:j], 16, 32)
			if err != nil {
				return "", false
			}
			b.WriteByte(byte(v))
			i = j
		case '0', '1', '2', '3', '4', '5', '6', '7':
			j := i
			for j < len(body) && j < i+3 && body[j] >= '0' && body[j] <= '7' {
				j++
			}
			v, err := strconv.ParseUint(body[i:j], 8, 16)
			if err != nil {
				return "", false
			}
			b.WriteByte(byte(v))
			i = j
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String(), true
}

func isHexByte(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
