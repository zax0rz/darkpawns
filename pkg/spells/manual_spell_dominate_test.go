package spells

import (
	"strings"
	"testing"
)

// manualProbe is a caster or victim stub for the manual-spell dispatcher. It
// implements exactly the duck-typed interfaces castCharm and sendToCaster use,
// and records every line it is sent.
type manualProbe struct {
	name   string
	level  int
	intVal int
	msgs   []string
	npc    bool
}

func (p *manualProbe) GetName() string      { return p.name }
func (p *manualProbe) GetLevel() int        { return p.level }
func (p *manualProbe) GetInt() int          { return p.intVal }
func (p *manualProbe) GetPosition() int     { return int(PosFighting) }
func (p *manualProbe) IsNPC() bool          { return p.npc }
func (p *manualProbe) SendMessage(m string) { p.msgs = append(p.msgs, m) }

// manualWorld satisfies followerWorld, the charm path's world seam.
type manualWorld struct {
	follows []string
}

func (w *manualWorld) AddFollowerQuiet(ch, leader interface{}) {
	w.follows = append(w.follows, "add")
}
func (w *manualWorld) StopFollowerByName(string)              {}
func (w *manualWorld) CircleFollowByName(string, string) bool { return false }
func (w *manualWorld) NumFollowers(string) int                { return 0 }

const inventedManualText = "Spell not yet implemented."

// TestManualSpellsNeverPrintInventedText is the R5c class audit for DP-1405:
// every spell whose live table entry carries RoutineManual is dispatched with a
// real call and must never emit the port's invented line, which appears in no C
// source. Before the dominate fix this fails on SpellDominate.
func TestManualSpellsNeverPrintInventedText(t *testing.T) {
	seen := 0
	for spell := 1; spell < 400; spell++ {
		si := GetSpellInfo(spell)
		if si == nil || !si.HasRoutine(RoutineManual) {
			continue
		}
		seen++
		caster := &manualProbe{name: "Caster", level: 30, intVal: 10}
		victim := &manualProbe{name: "Victim", level: 5, intVal: 4, npc: true}
		ExecuteManualSpell(spell, 30, caster, victim, nil, "", &manualWorld{})
		for _, line := range append(append([]string{}, caster.msgs...), victim.msgs...) {
			if strings.Contains(line, inventedManualText) {
				t.Fatalf("spell %d printed the invented line %q", spell, line)
			}
		}
	}
	if seen != 26 {
		t.Fatalf("audited %d manual spells, want the table's 26", seen)
	}
}

// TestDominateRoutesToCharm proves DP-1405's routing: dominate produces exactly
// what charm produces, through the same dispatcher. Without the SpellDominate
// case dominate sent the invented line and nothing else, so the two probes
// differ and the invented line appears.
func TestDominateRoutesToCharm(t *testing.T) {
	run := func(spell int) (caster, victim *manualProbe, world *manualWorld) {
		caster = &manualProbe{name: "Caster", level: 30, intVal: 10}
		victim = &manualProbe{name: "Victim", level: 5, intVal: 4, npc: true}
		world = &manualWorld{}
		ExecuteManualSpell(spell, 30, caster, victim, nil, "", world)
		return caster, victim, world
	}
	domCaster, domVictim, domWorld := run(SpellDominate)
	chaCaster, chaVictim, chaWorld := run(SpellCharm)

	if len(domCaster.msgs) == 0 && len(domVictim.msgs) == 0 && len(domWorld.follows) == 0 {
		t.Fatal("a dominate cast must reach the charm path and produce its effects")
	}
	if strings.Join(domCaster.msgs, "|") != strings.Join(chaCaster.msgs, "|") ||
		strings.Join(domVictim.msgs, "|") != strings.Join(chaVictim.msgs, "|") ||
		len(domWorld.follows) != len(chaWorld.follows) {
		t.Fatalf("dominate=%v/%v/%d charm=%v/%v/%d",
			domCaster.msgs, domVictim.msgs, len(domWorld.follows),
			chaCaster.msgs, chaVictim.msgs, len(chaWorld.follows))
	}
}
