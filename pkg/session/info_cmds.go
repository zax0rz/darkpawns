package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// className returns the class name for a class number.
func className(class int) string {
	names := map[int]string{
		0: "Mage",
		1: "Cleric",
		2: "Warrior",
		3: "Rogue",
		4: "Monk",
		5: "Paladin",
		6: "Ranger",
		7: "Thief",
		8: "Bard",
		9: "Warlock",
	}
	if name, ok := names[class]; ok {
		return name
	}
	return fmt.Sprintf("Class %d", class)
}

// positionName returns a human-readable position string.
func positionName(pos int) string {
	switch pos {
	case combat.PosDead:
		return "Dead"
	case combat.PosMortally:
		return "Mortally Wounded"
	case combat.PosIncap:
		return "Incapacitated"
	case combat.PosStunned:
		return "Stunned"
	case combat.PosSleeping:
		return "Sleeping"
	case combat.PosResting:
		return "Resting"
	case combat.PosSitting:
		return "Sitting"
	case combat.PosFighting:
		return "Fighting"
	case combat.PosStanding:
		return "Standing"
	default:
		return fmt.Sprintf("Pos %d", pos)
	}
}

// conditionLabel returns a label for condition-related output.
func conditionLabel(p *game.Player) string {
	var parts []string
	if p.Hunger <= 8 {
		parts = append(parts, "Hungry")
	} else if p.Hunger >= 20 {
		parts = append(parts, "Full")
	}
	if p.Thirst <= 8 {
		parts = append(parts, "Thirsty")
	} else if p.Thirst >= 20 {
		parts = append(parts, "Hydrated")
	}
	if p.Drunk > 0 {
		parts = append(parts, "Drunk")
	}
	if len(parts) == 0 {
		return "Normal"
	}
	return strings.Join(parts, ", ")
}

// ---------------------------------------------------------------------------
// info — fancy ASCII boxed character overview
// ---------------------------------------------------------------------------
func cmdInfo(s *Session, args []string) error {
	p := s.player
	if p == nil {
		return nil
	}

	levelStr := fmt.Sprintf("Level %d %s", p.Level, className(p.Class))
	nameLine := fmt.Sprintf("  %s  (%s)", p.Name, levelStr)
	if len(nameLine) > 38 {
		nameLine = nameLine[:38]
	}
	padding := 40 - len(nameLine)
	nameLine = nameLine + strings.Repeat(" ", padding)

	var buf strings.Builder
	buf.WriteString("╔" + strings.Repeat("═", 40) + "╗\n")
	buf.WriteString("║" + nameLine + "║\n")
	buf.WriteString("╠" + strings.Repeat("═", 40) + "╣\n")
	fmt.Fprintf(&buf, "║  %-18s│ %7d/%-3d║\n", "HIT POINTS", p.Health, p.MaxHealth)
	fmt.Fprintf(&buf, "║  %-18s│ %7d/%-3d║\n", "MANA", p.Mana, p.MaxMana)
	fmt.Fprintf(&buf, "║  %-18s│ %7d/%-3d║\n", "MOVE", p.Move, p.MaxMove)
	buf.WriteString("╠" + strings.Repeat("═", 19) + "╬" + strings.Repeat("═", 20) + "╣\n")

	strStr := fmt.Sprintf("STR: %d/%d", p.Stats.Str, 18)
	wisStr := fmt.Sprintf("WIS: %d/%d", p.Stats.Wis, 12)
	fmt.Fprintf(&buf, "║  %-19s│ %-19s║\n", strStr, wisStr)
	intStr := fmt.Sprintf("INT: %d/%d", p.Stats.Int, 13)
	chaStr := fmt.Sprintf("CHA: %d/%d", p.Stats.Cha, 15)
	fmt.Fprintf(&buf, "║  %-19s│ %-19s║\n", intStr, chaStr)
	dexStr := fmt.Sprintf("DEX: %d/%d", p.Stats.Dex, 16)
	conStr := fmt.Sprintf("CON: %d/%d", p.Stats.Con, 14)
	fmt.Fprintf(&buf, "║  %-19s│ %-19s║\n", dexStr, conStr)

	buf.WriteString("╠" + strings.Repeat("═", 19) + "╬" + strings.Repeat("═", 20) + "╣\n")

	acStr := fmt.Sprintf("AC: %d", p.AC)
	hitStr := fmt.Sprintf("HITROLL: %+d", p.Hitroll)
	fmt.Fprintf(&buf, "║  %-19s│ %-19s║\n", acStr, hitStr)
	damStr := fmt.Sprintf("DAMROLL: %+d", p.Damroll)
	alignStr := fmt.Sprintf("ALIGN: %d", p.Alignment)
	fmt.Fprintf(&buf, "║  %-19s│ %-19s║\n", damStr, alignStr)

	buf.WriteString("╠" + strings.Repeat("═", 19) + "╩" + strings.Repeat("═", 20) + "╣\n")
	fmt.Fprintf(&buf, "║  %-38s║\n", "Conditions: "+conditionLabel(p))
	fmt.Fprintf(&buf, "║  %-38s║\n", "Position: "+positionName(p.Position))
	fmt.Fprintf(&buf, "║  %-38s║\n", "Gold: "+fmt.Sprintf("%d", p.Gold))
	buf.WriteString("╚" + strings.Repeat("═", 40) + "╝\n")

	s.Send(buf.String())
	return nil
}
