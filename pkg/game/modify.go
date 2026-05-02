package game

import (
	"fmt"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// modify.go — ported from modify.c
// Currently ~40% complete.
// Missing: do_set, do_stat, do_gecho, do_social
// Original C: modify.c ~300 lines total
// ---------------------------------------------------------------------------

// TODO: implement do_set - full character field editor
// Expected: allows immortals to inspect and modify any character field by name
// Original: modify.c ~200 lines

// TODO: implement do_stat - character inspection command
// Expected: shows detailed stats for a target character
// Original: modify.c ~50 lines (do_stat + helper display functions)

// TODO: implement do_gecho - broadcast message to entire game
// Expected: sends a message to every player on the server
// Original: modify.c ~20 lines

// TODO: implement do_social - social/toggle command
// Expected: sets social struct fields on the player
// Original: modify.c ~30 lines

// ---------------------------------------------------------------------------
// modify.go — ported from modify.c (skillset, string editing)
// ---------------------------------------------------------------------------

// doSkillset — ACMD(do_skillset) — sets a skill level on another player.
//
// Syntax: skillset <target> '<skill_name>' <value>
//
// Wraps the target extraction, skill name extraction from single quotes,
// and value clamping (0-100).  Currently skips the spells[] listing that
// the C version prints when no arguments are given.
func (w *World) doSkillset(ch *Player, me *MobInstance, cmd string, arg string) bool {
	first, rest := splitArg(arg)
	if first == "" {
		ch.SendMessage("Syntax: skillset <name> '<skill>' <value>\r\n")
		return true
	}

	// Find target in the same room as ch.
	targetPl, targetMob := w.findCharInRoom(ch, ch.RoomVNum, first)
	if targetPl == nil && targetMob == nil {
		ch.SendMessage("No one here by that name.\r\n")
		return true
	}

	// Must be a Player; reject mobs.
	if targetPl == nil {
		ch.SendMessage("You can't set NPC skills.\r\n")
		return true
	}
	vict := targetPl

	skillArg := strings.TrimSpace(rest)
	if skillArg == "" {
		ch.SendMessage("Skill name expected.\r\n")
		return true
	}

	// Skill name must be enclosed in single quotes.
	if skillArg[0] != '\'' {
		ch.SendMessage("Skill must be enclosed in: ''\r\n")
		return true
	}

	// Find closing quote.
	qend := 1
	for qend < len(skillArg) && skillArg[qend] != '\'' {
		qend++
	}
	if qend >= len(skillArg) || skillArg[qend] != '\'' {
		ch.SendMessage("Skill must be enclosed in: ''\r\n")
		return true
	}

	skillName := strings.ToLower(skillArg[1:qend])
	if skillName == "" {
		ch.SendMessage("Skill name expected.\r\n")
		return true
	}

	// Grab remainder after closing quote.
	remainder := strings.TrimSpace(skillArg[qend+1:])
	if remainder == "" {
		ch.SendMessage("Learned value expected.\r\n")
		return true
	}

	value, err := strconv.Atoi(remainder)
	if err != nil {
		ch.SendMessage("Numeric value expected.\r\n")
		return true
	}
	if value < 0 {
		ch.SendMessage("Minimum value for learned is 0.\r\n")
		return true
	}
	if value > 100 {
		ch.SendMessage("Max value for learned is 100.\r\n")
		return true
	}

	// Apply the skill.
	vict.SetSkill(skillName, value)

	ch.SendMessage(fmt.Sprintf("You change %s's %s to %d.\r\n", vict.GetName(), skillName, value))
	// Also notify the target if they are the one being changed.
	if vict.GetName() != ch.GetName() {
		vict.SendMessage(fmt.Sprintf("%s changed your %s to %d.\r\n", ch.GetName(), skillName, value))
	}

	return true
}

// doString — ACMD(do_string) — edits string fields on objects.
//
// Syntax: string <object> <field> [value]
//
// Fields (1-5):
//
//	1 = name (Keywords)
//	2 = short description (ShortDesc)
//	3 = long description (LongDesc)
//	4 = extra/action description (ActionDesc / an extra description)
//	5 = title (Title — stored on the object prototype)
//
// If value is omitted the C version enters an interactive edit mode;
// this port prints a stub message asking the builder to supply the
// value inline instead.
func (w *World) doString(ch *Player, me *MobInstance, cmd string, arg string) bool {
	first, rest := splitArg(arg)
	if first == "" {
		ch.SendMessage("Usage: string <object> <field> [value]\r\n")
		return true
	}

	obj := w.findObjNear(ch, first)
	if obj == nil {
		ch.SendMessage("You don't see that here.\r\n")
		return true
	}

	fieldStr, valueArg := splitArg(rest)
	if fieldStr == "" {
		ch.SendMessage("Usage: string <object> <field> [value]\r\n")
		return true
	}

	field, err := strconv.Atoi(fieldStr)
	if err != nil || field < 1 || field > 5 {
		ch.SendMessage("Field must be 1-5 (1=name, 2=short, 3=long, 4=description/extra, 5=title).\r\n")
		return true
	}

	if valueArg == "" {
		ch.SendMessage("Enter string mode not yet supported — use 'string <obj> <field> <value>' directly.\r\n")
		return true
	}

	// Apply the string value to the selected field.
	value := strings.TrimSpace(valueArg)
	fieldNames := []string{"", "name", "short description", "long description", "description", "title"}
	var fieldLabel string
	if field >= 1 && field <= len(fieldNames)-1 {
		fieldLabel = fieldNames[field]
	} else {
		fieldLabel = fmt.Sprintf("field %d", field)
	}

	switch field {
	case 1:
		obj.Prototype.Keywords = value
	case 2:
		obj.Prototype.ShortDesc = value
	case 3:
		obj.Prototype.LongDesc = value
	case 4:
		// If an extra description with the object's keyword exists, edit that;
		// otherwise set ActionDesc as the direct editor does when no extra desc
		// key is given.
		found := false
		for i, ed := range obj.Prototype.ExtraDescs {
			if ed.Keywords == obj.Prototype.Keywords {
				obj.Prototype.ExtraDescs[i].Description = value
				found = true
				break
			}
		}
		if !found {
			obj.Prototype.ActionDesc = value
		}
	case 5:
		// Title is stored per-object-prototype in the C version.
		// Use ActionDesc as a reasonable proxy for lack of a dedicated Title field.
		obj.Prototype.ActionDesc = value
	}

	ch.SendMessage(fmt.Sprintf("You set %s's %s to '%s'.\r\n", obj.Prototype.ShortDesc, fieldLabel, value))
	return true
}
