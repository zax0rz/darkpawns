package parser

import "strings"

// cFName is the parser package's port of C's fname() (src/handler.c:68-79):
// the leading run of ASCII letters, stopping at the first non-letter. C's
// runtime twin is game.CFName; this file keeps its own copy because pkg/game
// imports pkg/parser, so the leaf cannot reference the game helper. The two
// must stay in step (DP-1500).
func cFName(name string) string {
	end := 0
	for end < len(name) && isASCIIAlphaByte(name[end]) {
		end++
	}
	return name[:end]
}

func isASCIIAlphaByte(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// cFixShortDescArticle ports the load-time article fix C applies to both object
// short_descriptions (src/db.c:1388-1392) and mobile short_descrs
// (src/db.c:1294-1298). When fname(sd) — the leading alphabetic run — equals
// "a", "an" or "the" case-insensitively, C lowercases only the first byte
// (*tmpptr = LOWER(*tmpptr)); everything else is left untouched. A short desc
// whose leading byte is not a letter, or whose first run is a different word,
// is returned unchanged.
func cFixShortDescArticle(sd string) string {
	switch strings.ToLower(cFName(sd)) {
	case "a", "an", "the":
		if b := sd[0]; b >= 'A' && b <= 'Z' {
			return string(b+'a'-'A') + sd[1:]
		}
	}
	return sd
}
