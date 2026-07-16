package game

import (
	"fmt"
	"sort"
	"strings"
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

// RenderSkillList ports spec_procs.c list_skills(): the practice-session count
// then every skill/spell the class can learn at its level (from ClassSpells =
// init_spell_levels), alpha-sorted by display name, with how_good(GET_SKILL).
// Mana rendering for spells is a follow-on (DP-1166 spell-name/number work).
func RenderSkillList(p *Player) string {
	class := p.GetClass()
	level := p.GetLevel()

	var b strings.Builder
	practices := p.GetPractices()
	if practices == 0 {
		b.WriteString("You have no practice sessions remaining.\r\n")
	} else {
		plural := "s"
		if practices == 1 {
			plural = ""
		}
		fmt.Fprintf(&b, "You have %d practice session%s remaining.\r\n", practices, plural)
	}
	fmt.Fprintf(&b, "You know of the following %ss:\r\n", SplSkl(class))

	type entry struct {
		name string
		pct  int
	}
	var entries []entry
	if class >= 0 && class < len(ClassSpells) {
		for _, e := range ClassSpells[class] {
			if level < e.Level {
				continue
			}
			name := SkillCatalogName(e.Num)
			if name == "" {
				continue
			}
			entries = append(entries, entry{name: name, pct: p.GetSkill(strings.ToLower(name))})
		}
	}
	// C displays in spell_sort_info order = alphabetical by name.
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	for _, e := range entries {
		// C: sprintf("%-20s %s %s\r\n", spells[i], how_good(pct), mana?manastring:"")
		// Skills have no mana → the final %s is empty (leaving a trailing space).
		fmt.Fprintf(&b, "%-20s %s \r\n", e.name, howGood(e.pct))
	}

	return b.String()
}

// FindSkillNum resolves a skill/spell name to its number, mirroring C
// find_skill_num() (spell_parser.c): case-insensitive prefix match against the
// spells[] display-name table, returning the first (lowest-numbered) match, or
// -1 if none. Multiword names match on the whole string as a prefix.
func FindSkillNum(name string) int {
	q := strings.ToLower(strings.TrimSpace(name))
	if q == "" {
		return -1
	}
	for num := 1; num < skillCatalogSize(); num++ {
		cname := SkillCatalogName(num)
		if cname == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(cname), q) {
			return num
		}
	}
	return -1
}
