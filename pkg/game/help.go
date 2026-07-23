package game

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// HelpEntry represents a single help file entry.
type HelpEntry struct {
	Keyword string // Primary keyword (lowercased, one per word on the keyword line)
	// Entry is the entry text INCLUDING the keyword line as its first line
	// (C load_help stores key\r\n + body; do_help skips past the first \n at
	// display). Keeping the keyword line lets the display path mirror C exactly.
	Entry string
}

// LoadHelpFiles loads all .hlp files listed in dir/index (falling back to
// dir/index.mini, as C does only in mini_mud mode) into a keyword-sorted help
// table — a faithful port of C's index_boot(DB_BOOT_HLP) + load_help + qsort
// (db.c:644-685, 1618-1661).
//
// Record format (lib/text/help/*.hlp):
//
//	keyword1 [keyword2 ...]      ← keyword line; one_word splits it (lowercase,
//	                              "QUOTED PHRASE" → one keyword)
//	body line
//	…
//	#                            ← terminator (a line whose FIRST char is '#')
//	$
//
// Each keyword on the keyword line yields a separate HelpEntry sharing the same
// Entry text. The table is sorted by case-insensitive keyword (C's hsort), the
// order do_help's binary search depends on.
func LoadHelpFiles(dir string) ([]HelpEntry, error) {
	// C index_boot picks index.mini only in mini_mud mode; the normal path uses
	// index. We try index first and fall back to index.mini if it is absent
	// (keeps the loader usable in reduced-data checkouts without a mini flag).
	indexPath := filepath.Join(dir, "index")
	indexFile, err := os.Open(indexPath)
	if err != nil {
		indexPath = filepath.Join(dir, "index.mini")
		indexFile, err = os.Open(indexPath)
		if err != nil {
			return nil, err
		}
	}
	defer indexFile.Close()

	var entries []HelpEntry
	scanner := bufio.NewScanner(indexFile)
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		// C: while (*buf1 != '$') — '$' terminates the index. Blank lines are
		// not expected but skipped defensively.
		if name == "" || name == "$" {
			continue
		}
		hlpEntries, err := loadHelpFile(filepath.Join(dir, name))
		if err != nil {
			continue // C perror+exit; we skip unreadable files (boot resilience)
		}
		entries = append(entries, hlpEntries...)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// C qsort(help_table, …, hsort) — str_cmp (case-insensitive) on keyword.
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Keyword) < strings.ToLower(entries[j].Keyword)
	})
	return entries, nil
}

// loadHelpFile loads a single .hlp file — a faithful port of C load_help
// (db.c:1618-1649). The entry text is built as `keywordline\r\n` + each body
// line + `\r\n` (C: strcpy(entry, strcat(key, "\r\n")) then strcat of each
// line+"\r\n"), so the keyword line is retained as the first line. A body line
// whose first char is '#' terminates the entry (C: while (*line != '#')).
func loadHelpFile(path string) ([]HelpEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read raw lines preserving the trailing '\r' that get_one_line leaves in
	// place (it strips only the '\n'). bufio.ReadString('\n') keeps the '\n';
	// we trim exactly one trailing '\n' (and a following '\r' is retained in the
	// line content, matching C, then re-appended below — see getOneLine).
	var raw []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw = append(raw, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// getOneLine indexes: C reads sequentially; we mirror with a cursor.
	lines := make([]string, len(raw))
	copy(lines, raw)

	var entries []HelpEntry
	pos := 0
	// get the first keyword line (C: get_one_line(fl, key); while (*key != '$'))
	for pos < len(lines) {
		key := getOneLine(lines, &pos)
		if key == "" || key[0] == '$' {
			break
		}

		// Build entry = key + "\r\n" + each body line + "\r\n", until a line
		// whose first char is '#'.
		var entry strings.Builder
		entry.WriteString(key)
		entry.WriteString("\r\n")

		for pos < len(lines) {
			line := getOneLine(lines, &pos)
			if line != "" && line[0] == '#' {
				break
			}
			entry.WriteString(line)
			entry.WriteString("\r\n")
		}
		entryText := entry.String()

		// Add an entry under each keyword on the keyword line (C: one_word loop).
		for _, kw := range oneWord(key) {
			entries = append(entries, HelpEntry{Keyword: kw, Entry: entryText})
		}
	}
	return entries, nil
}

// getOneLine mirrors C get_one_line (db.c:1607): return the next line with
// exactly one trailing '\n' removed (the '\r', if present, is KEPT). Advances
// *pos past the consumed line. Returns "" at end of input.
func getOneLine(lines []string, pos *int) string {
	if *pos >= len(lines) {
		return ""
	}
	line := lines[*pos]
	*pos++
	// C: buf[strlen(buf)-1] = '\0' strips the trailing \n only. bufio.Scanner
	// already dropped the \n, so the line is as C would have it (with any \r).
	return line
}

// oneWord splits a help keyword line into individual lowercased keywords — a
// faithful port of C one_word (interpreter.c:1291-1320): a double-quoted span
// becomes a single keyword (e.g. `"FIRST AID"` → `first aid`); otherwise a run
// of non-space characters is one keyword. All bytes are lowercased.
func oneWord(s string) []string {
	var words []string
	i := 0
	for i < len(s) {
		// skip_spaces
		for i < len(s) && isASCIISpace(s[i]) {
			i++
		}
		if i >= len(s) {
			break
		}
		var w strings.Builder
		if s[i] == '"' {
			// quoted span: take until the closing quote
			i++
			for i < len(s) && s[i] != '"' {
				w.WriteByte(lowerByte(s[i]))
				i++
			}
			if i < len(s) {
				i++ // skip closing quote
			}
		} else {
			for i < len(s) && !isASCIISpace(s[i]) {
				w.WriteByte(lowerByte(s[i]))
				i++
			}
		}
		if w.Len() > 0 {
			words = append(words, w.String())
		}
	}
	return words
}

func isASCIISpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n' || b == '\v' || b == '\f'
}

func lowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// strnCmp is a faithful port of C strn_cmp (utils.c:125-138): a case-insensitive
// comparison of the first n bytes of a and b. It scans while EITHER string still
// has bytes, so:
//   - returns 0 if a is a prefix of (or equal to) b's first n bytes (a match);
//   - returns ±1 at the first differing byte;
//   - returns ±1 if a is longer than b but agrees on b's full length (so a
//     longer-than-keyword argument does NOT match — prefix, one direction).
func strnCmp(a, b string, n int) int {
	i := 0
	for (i < len(a) || i < len(b)) && n > 0 {
		var ca, cb byte
		if i < len(a) {
			ca = lowerByte(a[i])
		}
		if i < len(b) {
			cb = lowerByte(b[i])
		}
		if ca != cb {
			if ca < cb {
				return -1
			}
			return 1
		}
		i++
		n--
	}
	return 0
}

// SearchHelp searches a keyword-sorted help table for argument — a faithful port
// of C do_help's lookup (act.informative.c:1594-1630): a binary search on
// strnCmp(argument, keyword, len(argument)) (a PREFIX match), then a backtrack
// to the FIRST matching entry ("trace backwards… Thanks Jeff Fink!"). Returns
// the first-match entry, or nil if no entry's keyword has argument as a prefix.
//
// The table MUST be sorted by case-insensitive keyword (LoadHelpFiles does this).
func SearchHelp(table []HelpEntry, argument string) *HelpEntry {
	if len(table) == 0 || argument == "" {
		return nil
	}
	minlen := len(argument)
	bot, top := 0, len(table)-1
	for bot <= top {
		mid := (bot + top) / 2
		chk := strnCmp(argument, table[mid].Keyword, minlen)
		switch {
		case chk == 0:
			// Match — trace backwards to the first matching entry (Jeff Fink loop).
			for mid > 0 && strnCmp(argument, table[mid-1].Keyword, minlen) == 0 {
				mid--
			}
			return &table[mid]
		case chk > 0:
			bot = mid + 1
		default:
			top = mid - 1
		}
	}
	return nil
}

// sortHelpTable sorts a help table by case-insensitive keyword in place — C's
// qsort/hsort order (db.c:684). do_help's binary search requires it.
func sortHelpTable(table []HelpEntry) {
	sort.Slice(table, func(i, j int) bool {
		return strings.ToLower(table[i].Keyword) < strings.ToLower(table[j].Keyword)
	})
}

// LoadHelpScreen reads the no-argument help screen (lib/text/help/screen) into a
// string — the Go analog of C file_to_string_alloc(HELP_PAGE_FILE, &help)
// (db.c:193), which do_help page_strings on a bare `help` (act.informative.c:1584).
func LoadHelpScreen(dir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, "screen"))
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// RaceHelpEntries returns hardcoded race help text from src/constants.c:205-350.
// These are appended to the help table at boot so `help <race>` works.
func RaceHelpEntries() []HelpEntry {
	return []HelpEntry{
		{
			Keyword: "race help",
			Entry: "\r\n" +
				"Your race is pretty much class independant; it affects innate abilities such\r\n" +
				"as:\r\n" +
				"The type of terrain you see best in: \r\n" +
				"       RAKSHASA: desert              SSAUR: swamplands\r\n" +
				"       MINOTAUR & ELF: forest        DWARF: mountains\r\n" +
				"       KENDER & HUMAN: fairly good everywhere.\r\n" +
				"Magick resistance.: Elves and dwarves are a bit more hearty in this area.\r\n" +
				"Attitudes: Humans abound, so they are often suspicious of other races and\r\n" +
				"       give preferential treatment to their own kind.\r\n" +
				"Kender tend to 'acquire' other's objects unknowingly, and make excellent\r\n" +
				"       thieves. Only humans can belong to the ninja class.\r\n" +
				"Each race has its own language.\r\n",
		},
		{
			Keyword: "help human",
			Entry: "\r\n" +
				"Humans are the most common race on this world, and come in all sorts of shapes\r\n" +
				"and sizes. The appearance of humans are not the only thing that varies about\r\n" +
				"them, though, some are evil as sin, while others are good as good can be, but\r\n" +
				"most you shall find on your adventures are neutral, and will just mind their\r\n" +
				"own business and pay no attention to the affairs of adventurers. Also, humans\r\n" +
				"are the only race that can become ninjas, the dangerous oriental mercenaries.\r\n" +
				"They adapt easily to most climes, allowing them to build cities in almost any\r\n" +
				"location.\r\n",
		},
		{
			Keyword: "help dwarf",
			Entry: "\r\n" +
				"Dwarves are a noble race of demihumans who dwell under the earth, forging\r\n" +
				"great cities and waging massive wars against the forces of chaos and evil.\r\n" +
				"Dwarves also have much in common with the rocks and gems they love to work,\r\n" +
				"for they are both hard and unyielding. It's often been said that it's easier\r\n" +
				"to make a stone weep than it is to change a dwarf's mind. Standing an average\r\n" +
				"of four-and-a-half feet tall, dwarves tend to be stocky and muscular. They\r\n" +
				"have ruddy cheeks and bright eyes. Their skin is typically deep tan or light\r\n" +
				"brown. Their hair is usually black, grey, or brown, and worn long, though not\r\n" +
				"long enough to impair vision in any way. They favor long beards and moustaches\r\n" +
				"as well.\r\n",
		},
		{
			Keyword: "help elf",
			Entry: "\r\n" +
				"Though their lives span several human generations, elves appear at first\r\n" +
				"glance to be frail when compared to man, due to their delicate and finely\r\n" +
				"chiseled features. Elves have very pale complextions, which is odd because\r\n" +
				"they spend a great deal of time outdoors. They tend to be slim, almost \r\n" +
				"fragile. Though they are not as sturdy as humans, elves are much more agile.\r\n" +
				"Elves have learned that it is very important to understand the creatures, both\r\n" +
				"good and evil, that share their forest homes.\r\n",
		},
		{
			Keyword: "help kender",
			Entry: "\r\n" +
				"Kender are small, kind, but somewhat annoying, elf-like beings that have\r\n" +
				"recently spread across the globe. They do not seem to have any sort of kingdom\r\n" +
				"and most are found just wandering throughout the lands, exploring. Although\r\n" +
				"some are trained thieves, the whole of the kender race seems to have a knack\r\n" +
				"for stealing, and occasionally, without even noticing it sometimes, they have\r\n" +
				"been known to steal from friends and enemies alike. They act much like humans,\r\n" +
				"but four things make a kender's personality drastically different from that of\r\n" +
				"a typical human. Kender are utterly fearless, insatiably curious, unstoppably\r\n" +
				"mobile and independant, and will pick up anything that is not nailed down.\r\n",
		},
		{
			Keyword: "help minotaur",
			Entry: "\r\n" +
				"Minotaurs are either cursed humans or the offspring of minotaurs and humans.\r\n" +
				"They are usually found dwelling in underground labyrinths, for they seem to\r\n" +
				"have an innate ability to manuver in these places, and do not often lose their\r\n" +
				"sense of direction. Minotaurs are huge, well over seven feet tall, and their\r\n" +
				"broad bodies ripple with muscles. They have the head of a bull but the body of\r\n" +
				"a human male, there have been accounts of female minotaurs, but they are rare.\r\n" +
				"The color of their fur ranges from brown to black, while their body coloring\r\n" +
				"varies, as would a normal human's. Although they usually dwell in mazes\r\n" +
				"beneath the earth, it is noted that they also see very well in forests.\r\n",
		},
		{
			Keyword: "help rakshasa",
			Entry: "\r\n" +
				"Rakshasas are a race of malevolent spirits encased in flesh that hunt and\r\n" +
				"torment humanity. No one knows where these creatures originate, some say they\r\n" +
				"are the embodiment of nightmares. The only way to describe their form is that\r\n" +
				"they are humanoid tigers, with hands whose palms curve backward, away from the\r\n" +
				"body. Most of the worlds rakshasa are evil, but recently many have decided to\r\n" +
				"stop their tyrannical living and become adventurers, although they still\r\n" +
				"retain their fondness towards the great sandy wastes of their homeland.\r\n",
		},
		{
			Keyword: "help ssaur",
			Entry: "\r\n" +
				"Ssaurs are a relatively new race in the world. They are a more evolved type of\r\n" +
				"lizardman, and most are more intelligent than their aggressive ancestors, and\r\n" +
				"for that are shunned from the lizardman tribes, and the few that are born into\r\n" +
				"those tribes are cast out almost as soon as they are hatched. Other than the \r\n" +
				"intelligence, they appear to be the same as lizardman, although less evil-\r\n" +
				"looking. Ssaurs spend most of their lives in swamps and marshes, but some have\r\n" +
				"been known to adventure far away from their homes.\r\n",
		},
	}
}
