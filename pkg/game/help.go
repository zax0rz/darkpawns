package game

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// HelpEntry represents a single help file entry.
type HelpEntry struct {
	Keyword string // Primary keyword (for display)
	Entry   string // Full entry text
}

// LoadHelpFiles loads all .hlp files from the given directory into a help table.
// Format (from C db.c load_help):
//   keyword1 [keyword2 ...]
//   entry text lines
//   #
//   ...
//   $
//
// Each keyword line creates a separate HelpEntry with the same text.
// The '$' line terminates the file.
func LoadHelpFiles(dir string) ([]HelpEntry, error) {
	// Read the index file to get the list of .hlp files
	indexPath := filepath.Join(dir, "index")
	indexFile, err := os.Open(indexPath)
	if err != nil {
		// Try index.mini as fallback
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
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "$" {
			continue
		}
		// line is a .hlp filename
		hlpEntries, err := loadHelpFile(filepath.Join(dir, line))
		if err != nil {
			continue // skip unreadable files
		}
		entries = append(entries, hlpEntries...)
	}
	return entries, nil
}

// loadHelpFile loads a single .hlp file and returns all entries in it.
func loadHelpFile(path string) ([]HelpEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []HelpEntry
	var currentKeywords []string
	var currentEntry strings.Builder

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		// '$' terminates the entire file
		if strings.TrimSpace(line) == "$" {
			break
		}

		// '#' terminates the current entry
		if strings.TrimSpace(line) == "#" {
			// Save the current entry under all its keywords
			entryText := currentEntry.String()
			for _, kw := range currentKeywords {
				entries = append(entries, HelpEntry{
					Keyword: kw,
					Entry:   entryText,
				})
			}
			currentKeywords = nil
			currentEntry.Reset()
			continue
		}

		// If we have no keywords yet, this line is the keyword line
		if currentKeywords == nil {
			currentKeywords = strings.Fields(line)
			continue
		}

		// Otherwise, it's entry text
		currentEntry.WriteString(line)
		currentEntry.WriteString("\r\n")
	}

	// Handle any trailing entry (if file doesn't end with #)
	if currentEntry.Len() > 0 && len(currentKeywords) > 0 {
		entryText := currentEntry.String()
		for _, kw := range currentKeywords {
			entries = append(entries, HelpEntry{
				Keyword: kw,
				Entry:   entryText,
			})
		}
	}

	return entries, nil
}

// SearchHelp searches the help table for a keyword (case-insensitive).
// Returns the matching entry or nil if not found.
func SearchHelp(table []HelpEntry, keyword string) *HelpEntry {
	keyword = strings.ToLower(keyword)
	for i := range table {
		if strings.ToLower(table[i].Keyword) == keyword {
			return &table[i]
		}
	}
	return nil
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
