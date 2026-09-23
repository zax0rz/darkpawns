package stringcensus

import (
	"strings"
)

// cLitGroup is one literal after C's adjacent-literal concatenation: "a" "b"
// and "a" /* comment */ "b" are one string.
type cLitGroup struct {
	index int
	line  int
	text  string
}

// cLiteralGroups unescapes every string literal and merges runs of adjacent
// literals, which is how C splits a long message across source lines.
func cLiteralGroups(toks []cTok) []cLitGroup {
	var out []cLitGroup
	for i := 0; i < len(toks); i++ {
		if toks[i].kind != cTokString {
			continue
		}
		start := i
		var b strings.Builder
		for i < len(toks) && toks[i].kind == cTokString {
			s, ok := unescapeC(toks[i].text)
			if !ok {
				s = ""
			}
			b.WriteString(s)
			i++
		}
		i--
		out = append(out, cLitGroup{index: start, line: toks[start].line, text: b.String()})
	}
	return out
}

// cCallEnd returns the index of the closing ')' of the call whose '(' follows
// toks[at], or false when this identifier is not a call at all.
func cCallEnd(toks []cTok, at int) (int, bool) {
	if at+1 >= len(toks) || toks[at+1].kind != cTokPunct || toks[at+1].text != "(" {
		return 0, false
	}
	depth := 0
	for i := at + 1; i < len(toks); i++ {
		if toks[i].kind != cTokPunct {
			continue
		}
		switch toks[i].text {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return i, true
			}
		case "{":
			return 0, false
		}
	}
	return 0, false
}

// cFuncAt keeps the "current function" tracker up to date as the caller walks
// the token stream in order. When a '{' opens a definition the tracker moves to
// that function; a ';' at file scope clears a stale candidate. It is a
// heuristic, which is all the report needs: the column exists so a reviewer can
// jump to the right function, not to prove C structure.
func cFuncAt(toks []cTok, i int, current string) string {
	t := toks[i]
	if t.kind != cTokPunct || t.text != "{" {
		return current
	}
	name, ok := cDefinitionName(toks, i-1)
	if !ok {
		return current
	}
	return name
}

// cDefinitionName looks backwards from the ')' that precedes the '{' at
// toks[close+1] for the name of the function being defined. It returns false
// when the brace opens anything else (an initializer, a struct body, a block).
func cDefinitionName(toks []cTok, close int) (string, bool) {
	if close < 1 || toks[close].kind != cTokPunct || toks[close].text != ")" {
		return "", false
	}
	depth := 0
	open := -1
scan:
	for i := close; i >= 0; i-- {
		if toks[i].kind != cTokPunct {
			continue
		}
		switch toks[i].text {
		case ")":
			depth++
		case "(":
			depth--
			if depth == 0 {
				open = i
				break scan
			}
		}
	}
	if open < 1 || toks[open-1].kind != cTokIdent {
		return "", false
	}
	name := toks[open-1].text
	if !cFuncMacros[name] {
		if cDeclKeywords[name] {
			return "", false
		}
		return name, true
	}
	// ACMD(do_quit): the function name is the first identifier inside.
	for i := open + 1; i < close; i++ {
		if toks[i].kind == cTokIdent {
			return toks[i].text, true
		}
	}
	return "", false
}
