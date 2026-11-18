package game

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/spells"
)

// C-faithful skill/spell catalog + practice math.
// Sources: src/spec_procs.c list_skills (157) / how_good (108) / guild (201);
// src/class.c prac_params (261). Rendering + the guild practice proc share this.

// pracTypeByClass indexes prac_params[PRAC_TYPE][class] (class.c:261): the
// prac_types[] index per class — 0=spell, 1=skill, 2=art (BOTH).
// Order: MAG CLE THE WAR MAGU AVA ASS PAL NIN PSI RAN MYS.
var pracTypeByClass = [12]int{0, 0, 1, 1, 0, 2, 1, 2, 2, 2, 1, 2}

// pracTypeNames is C prac_types[] (spec_procs.c:136).
var pracTypeNames = [3]string{"spell", "skill", "art"}

// prac_params rows (class.c:261), indexed by class.
// learned level (% considered "learned"), max/min gain per practice.
var (
	pracLearnedLevel = [12]int{95, 95, 85, 80, 95, 95, 85, 80, 85, 95, 80, 95}
	pracMaxGain      = [12]int{100, 100, 25, 25, 100, 100, 25, 25, 25, 100, 25, 100}
	pracMinGain      = [12]int{25, 25, 0, 0, 25, 25, 0, 0, 0, 25, 0, 25}
)

const cSkillListMaxStringLength = 8192 // src/structs.h:643

func pracLearned(class int) int {
	if class < 0 || class >= 12 {
		return 100
	}
	return pracLearnedLevel[class]
}

func pracMax(class int) int {
	if class < 0 || class >= 12 {
		return 0
	}
	return pracMaxGain[class]
}

func pracMin(class int) int {
	if class < 0 || class >= 12 {
		return 0
	}
	return pracMinGain[class]
}

// SplSkl returns SPLSKL(ch) — the "spell"/"skill"/"art" word for the class.
func SplSkl(class int) string {
	if class < 0 || class >= len(pracTypeByClass) {
		return "skill"
	}
	return pracTypeNames[pracTypeByClass[class]]
}

// howGood mirrors spec_procs.c:108 how_good(). Every string has a LEADING space.
func howGood(percent int) string {
	switch {
	case percent == 0:
		return " (not learned)"
	case percent <= 10:
		return " (awful)"
	case percent <= 20:
		return " (bad)"
	case percent <= 40:
		return " (poor)"
	case percent <= 55:
		return " (average)"
	case percent <= 70:
		return " (fair)"
	case percent <= 80:
		return " (good)"
	case percent <= 85:
		return " (very good)"
	case percent <= 98:
		return " (superb)"
	default:
		return " (MASTER)"
	}
}

// RenderSkillList ports the player-visible list_skills() result from
// spec_procs.c: every skill/spell whose spell_info min_level is reached,
// alpha-sorted by the raw spells[] name, with
// how_good(GET_SKILL). The practice-count prelude follows
// spec_procs.c:164-168 exactly: no sessions, one session, or a plural count.
// list_skills was not patched in the oracle; the reference oracle is simply
// built at -O0, where the prelude survives its own appends. Every entry with
// a non-zero mag_manacost carries C's cost column (spec_procs.c:185-192),
// through the same ManaCost formula cast uses.
// RenderActorSkillList renders the skill/spell list for either a player or a
// switched mobile (src/spec_procs.c:157-192).
func RenderActorSkillList(actor Actor) string {
	if p, ok := actor.(*Player); ok {
		return RenderSkillList(p)
	}
	if mob, ok := actor.(*MobInstance); ok {
		return fmt.Sprintf("You have no practice sessions remaining.\r\nYou know of the following %ss:\r\n", SplSkl(mob.GetClass()))
	}
	return ""
}

func RenderSkillList(p *Player) string {
	class := p.GetClass()
	level := p.GetLevel()

	var b strings.Builder
	practices := p.GetPractices()
	if practices == 0 {
		b.WriteString("You have no practice sessions remaining.\r\n")
	} else {
		suffix := "s"
		if practices == 1 {
			suffix = ""
		}
		fmt.Fprintf(&b, "You have %d practice session%s remaining.\r\n", practices, suffix)
	}
	fmt.Fprintf(&b, "You know of the following %ss:\r\n", SplSkl(class))

	type entry struct {
		num     int
		rawName string
	}
	entries := make([]entry, 0, skillCatalogSize()-1)
	for num := 1; num < skillCatalogSize(); num++ {
		entries = append(entries, entry{num: num, rawName: spells.SpellRawName(num)})
	}
	// C sort_spells sorts every catalog slot by raw spells[] name, including
	// unused entries that list_skills later skips by min_level.
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].rawName < entries[j].rawName })

	for _, e := range entries {
		// list_skills checks the output bound before its min_level gate on every
		// sort position (src/spec_procs.c:175-184).
		if b.Len() >= cSkillListMaxStringLength-32 {
			b.WriteString("**OVERFLOW**\r\n")
			break
		}
		if level < ClassSkillMinLevel(class, e.num) {
			continue
		}
		name := SkillCatalogName(e.num)
		if name == "" {
			continue
		}
		// C: sprintf("%-20s %s %s\r\n", spells[i], how_good(pct), mana?manastring:"")
		// spec_procs.c:185-192: the cost comes from mag_manacost over
		// find_skill_num(spells[i]) — the same formula cast uses — and prints
		// "( CCRED %d mana|psi pts CCNRM )" only when non-zero. Skills have
		// no mana → the final %s is empty (leaving a trailing space).
		manastring := ""
		if mana := ManaCost(FindSkillNum(name), class, level); mana != 0 {
			red, nrm := "", ""
			if colorLevel(p) >= colorComplete {
				red, nrm = ansiRed, ansiReset // CCRED/CCNRM at C_CMP (screen.h:50-51)
			}
			unit := "mana"
			if class == ClassPsionic || class == ClassMystic {
				unit = "psi pts" // IS_PSIONIC || IS_MYSTIC (utils.h:366,418)
			}
			manastring = fmt.Sprintf("( %s%d %s%s )", red, mana, unit, nrm)
		}
		fmt.Fprintf(&b, "%-20s %s %s\r\n", name, howGood(p.GetSkill(SkillStorageName(e.num))), manastring)
	}

	return b.String()
}

// FindSkillNum resolves a skill/spell name to its number, mirroring C
// find_skill_num() (spell_parser.c): case-insensitive prefix match against the
// spells[] display-name table, returning the first (lowest-numbered) match, or
// -1 if none. C also accepts an abbreviation for each token of a multiword
// spell name (for example, "c li" resolves to "cure light").
func FindSkillNum(name string) int {
	q := strings.ToLower(strings.TrimSpace(name))
	if q == "" {
		return -1
	}
	queryWords := strings.Fields(q)
	for num := 1; num < skillCatalogSize(); num++ {
		cname := SkillCatalogName(num)
		if cname == "" {
			continue
		}
		canonical := strings.ToLower(cname)
		if strings.HasPrefix(canonical, q) {
			return num
		}
		canonicalWords := strings.Fields(canonical)
		if len(queryWords) > len(canonicalWords) {
			continue
		}
		match := true
		for i, word := range queryWords {
			if !strings.HasPrefix(canonicalWords[i], word) {
				match = false
				break
			}
		}
		if match {
			return num
		}
	}
	return -1
}
