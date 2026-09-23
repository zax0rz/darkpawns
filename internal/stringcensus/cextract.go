package stringcensus

import (
	"path/filepath"
)

// cSink is one C call whose string literals can reach a player, or a
// buffer-building call whose result is later sent. The table is the tool's
// single source of truth for the C side; counts quoted in the comments come
// from `grep -o <name> src/*.c | wc -l` on 2026-09-23.
type cSink struct {
	// Name is the identifier at the call site.
	Name string
	// Alias names the sink this call really is, when Name is a macro.
	Alias string
	// Buffered marks a call that fills a buffer ("a simple heuristic is fine":
	// every literal in a sprintf/strcpy/strcat is kept and tagged buffered).
	Buffered bool
	// Why explains the sink in the report.
	Why string
}

// cSinks is the C sink table (brief item 2). It is the named list from the
// brief plus the two macro aliases (stc, and the same-function bounded buffer
// builders), which the brief's heuristic covers by intent: a literal written
// through them is still a literal C sends.
var cSinks = []cSink{
	{Name: "send_to_char", Why: "C's player-output primitive (2006 call sites)"},
	{Name: "stc", Alias: "send_to_char", Why: "utils.h:379 macro; send_to_char under a short name (327 call sites)"},
	{Name: "act", Why: "act()-style $code messages (1007 call sites)"},
	{Name: "SEND_TO_Q", Why: "comm.h:66 macro: write_to_output(desc, \"%s\", messg) (118 call sites)"},
	{Name: "write_to_output", Why: "descriptor output queue (107 call sites)"},
	{Name: "page_string", Why: "paged output (42 call sites)"},
	{Name: "send_to_room", Why: "room broadcast (21 call sites)"},
	{Name: "send_to_zone", Why: "zone broadcast (15 call sites)"},
	{Name: "send_to_outdoor", Why: "outdoor broadcast (12 call sites)"},
	{Name: "send_to_all", Why: "world broadcast (5 call sites)"},
	{Name: "sprintf", Buffered: true, Why: "buffer append; a later send_to_char flushes it (1324 call sites)"},
	{Name: "snprintf", Buffered: true, Why: "buffer append, bounded (12 call sites)"},
	{Name: "strcpy", Buffered: true, Why: "buffer append (375 call sites)"},
	{Name: "strncpy", Buffered: true, Why: "buffer append, bounded (29 call sites)"},
	{Name: "strcat", Buffered: true, Why: "buffer append (425 call sites)"},
	{Name: "strncat", Buffered: true, Why: "buffer append, bounded"},
}

// cSinkByName indexes cSinks for call-site lookup. A macro alias resolves to
// the sink it expands to, so stc() is reported as send_to_char.
var cSinkByName = func() map[string]cSink {
	m := make(map[string]cSink, len(cSinks))
	for _, s := range cSinks {
		m[s.Name] = s
	}
	return m
}()

// cTableFiles are the C translation units whose string tables are player-facing
// data (brief item 2, last sentence). Every literal in them counts.
var cTableFiles = map[string]bool{
	"constants.c": true,
	"class.c":     true,
}

// cFuncMacros are declaration macros whose first parenthesized identifier is the
// function name (interpreter.h:32 ACMD, structs.h:38 SPECIAL).
var cFuncMacros = map[string]bool{
	"ACMD":    true,
	"SPECIAL": true,
}

// cDeclKeywords are identifiers that can precede a parenthesized expression at
// file scope without introducing a function definition.
var cDeclKeywords = map[string]bool{
	"if": true, "for": true, "while": true, "switch": true, "return": true,
	"sizeof": true, "else": true, "do": true, "case": true, "assert": true,
}

// extractCFiles walks the C translation units the census was given (src/*.c)
// and returns every player-facing literal it can attribute. Table files keep
// every literal; other files keep literals that sit in a sink call's argument
// list.
func extractCFiles(files []sourceFile) []candidate {
	var out []candidate
	for _, f := range files {
		if filepath.Ext(f.rel) != ".c" {
			continue
		}
		if cTableFiles[filepath.Base(f.rel)] {
			out = append(out, tableCandidates(f.rel, f.data)...)
			continue
		}
		out = append(out, extractCFile(f.rel, f.data)...)
	}
	return out
}

// tableCandidates returns every string literal in a C string-table file.
func tableCandidates(file, src string) []candidate {
	var out []candidate
	for _, g := range cLiteralGroups(lexC(src)) {
		out = append(out, candidate{file: file, line: g.line, sink: "table", raw: g.text})
	}
	return out
}

// extractCFile walks one C translation unit: it finds sink call sites, keeps
// the literals inside their argument lists, and attributes each to the
// enclosing function.
func extractCFile(file, src string) []candidate {
	toks := lexC(src)
	groups := cLiteralGroups(toks)
	var out []candidate
	fn := ""
	for i := 0; i < len(toks); i++ {
		fn = cFuncAt(toks, i, fn)
		if toks[i].kind != cTokIdent {
			continue
		}
		s, ok := cSinkByName[toks[i].text]
		if !ok {
			continue
		}
		end, ok := cCallEnd(toks, i)
		if !ok {
			continue
		}
		sink := s.Name
		if s.Alias != "" {
			sink = s.Alias
		}
		if s.Buffered {
			sink = "buffered:" + s.Name
		}
		for _, g := range groups {
			if g.index > i && g.index < end {
				out = append(out, candidate{file: file, line: g.line, sink: sink, fn: fn, raw: g.text})
			}
		}
	}
	return out
}
