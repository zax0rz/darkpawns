package session

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// C-faithful skill/spell catalog + practice command.
// Source: src/act.other.c do_practice (543), src/spec_procs.c list_skills (157)
// + how_good (108) + prac_params (class.c:261). Dark Pawns has exactly ONE
// skill command — `practice`; skills/spells/learn/forget are Go inventions.

// pracTypeByClass indexes prac_params[PRAC_TYPE][class] (class.c:261), giving
// the prac_types[] index for each class: 0=spell, 1=skill, 2=art (BOTH).
// Order: MAG CLE THE WAR MAGU AVA ASS PAL NIN PSI RAN MYS.
var pracTypeByClass = [12]int{0, 0, 1, 1, 0, 2, 1, 2, 2, 2, 1, 2}

// pracTypeNames is C prac_types[] (spec_procs.c:136).
var pracTypeNames = [3]string{"spell", "skill", "art"}

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

// splSkl returns SPLSKL(ch) — the "spell"/"skill"/"art" word for the class.
func splSkl(class int) string {
	if class < 0 || class >= len(pracTypeByClass) {
		return "skill"
	}
	return pracTypeNames[pracTypeByClass[class]]
}

// renderSkillList ports spec_procs.c list_skills(). It shows the practice-session
// count then every skill/spell the class can learn at its current level (from the
// faithful classSpells catalog = init_spell_levels), alpha-sorted by display name,
// with how_good(GET_SKILL). Mana rendering for spells is a follow-on (the spell
// mana table is keyed by a different numbering — see the Phase 1 brief §4/§5).
func renderSkillList(p *game.Player) string {
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
	fmt.Fprintf(&b, "You know of the following %ss:\r\n", splSkl(class))

	// Build the catalog entries the class qualifies for at its level.
	type entry struct {
		name string
		pct  int
	}
	var entries []entry
	if class >= 0 && class < len(classSpells) {
		for _, e := range classSpells[class] {
			if level < e.Level {
				continue
			}
			name := game.SkillCatalogName(e.Num)
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

// cmdPractice ports src/act.other.c do_practice(): with no argument it lists the
// skill catalog; with any argument (when not standing on a guildmaster, whose
// spec proc would intercept first) it directs the player to their guild.
func cmdPractice(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	if len(args) > 0 {
		s.Send("You can only practice skills in your guild.\r\n")
		return nil
	}
	s.Send(renderSkillList(s.player))
	return nil
}
