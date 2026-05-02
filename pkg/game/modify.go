//nolint:unused // Game logic port — not yet wired to command registry.
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

// do_set — full character field editor (immortal command)
func (w *World) doSet(ch *Player, me *MobInstance, cmd string, arg string) bool {
	name, rest := splitArg(arg)
	field, value := splitArg(rest)
	if name == "" || field == "" {
		ch.SendMessage("Usage: set <victim> <field> <value>\r\n")
		return true
	}

	// Find target
	targetPl, targetMob := w.findCharInRoom(ch, ch.RoomVNum, name)
	var target *Player
	if targetPl != nil {
		target = targetPl
	} else if targetMob != nil {
		ch.SendMessage("You can't set fields on mobs yet.\r\n")
		return true
	} else {
		ch.SendMessage("There is no such creature.\r\n")
		return true
	}

	// Level check
	if target.Level >= ch.Level && target != ch {
		ch.SendMessage("Maybe that's not such a great idea...\r\n")
		return true
	}

	switch strings.ToLower(field) {
	case "level":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's level: %d\r\n", target.GetName(), target.Level))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 || v > 110 {
			ch.SendMessage("Value must be 0-110.\r\n")
			return true
		}
		target.Level = v
		ch.SendMessage(fmt.Sprintf("%s's level set to %d.\r\n", target.GetName(), v))
	case "gold":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's gold: %d\r\n", target.GetName(), target.Gold))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.Gold = v
		ch.SendMessage(fmt.Sprintf("%s's gold set to %d.\r\n", target.GetName(), v))
	case "hit", "hp", "health":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's hit: %d/%d\r\n", target.GetName(), target.Health, target.MaxHealth))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.Health = v
		ch.SendMessage(fmt.Sprintf("%s's hit set to %d.\r\n", target.GetName(), v))
	case "mana":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's mana: %d/%d\r\n", target.GetName(), target.Mana, target.MaxMana))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.Mana = v
		ch.SendMessage(fmt.Sprintf("%s's mana set to %d.\r\n", target.GetName(), v))
	case "maxhit":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's maxhit: %d\r\n", target.GetName(), target.MaxHealth))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.MaxHealth = v
		ch.SendMessage(fmt.Sprintf("%s's maxhit set to %d.\r\n", target.GetName(), v))
	case "maxmana":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's maxmana: %d\r\n", target.GetName(), target.MaxMana))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.MaxMana = v
		ch.SendMessage(fmt.Sprintf("%s's maxmana set to %d.\r\n", target.GetName(), v))
	case "maxmove":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's maxmove: %d\r\n", target.GetName(), target.MaxMove))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.MaxMove = v
		ch.SendMessage(fmt.Sprintf("%s's maxmove set to %d.\r\n", target.GetName(), v))
	case "move":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's move: %d/%d\r\n", target.GetName(), target.Move, target.MaxMove))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.Move = v
		ch.SendMessage(fmt.Sprintf("%s's move set to %d.\r\n", target.GetName(), v))
	case "align", "alignment":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's alignment: %d\r\n", target.GetName(), target.Alignment))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil || v < -1000 || v > 1000 {
			ch.SendMessage("Value must be -1000 to 1000.\r\n")
			return true
		}
		target.Alignment = v
		ch.SendMessage(fmt.Sprintf("%s's alignment set to %d.\r\n", target.GetName(), v))
	case "ac":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's AC: %d\r\n", target.GetName(), target.AC))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.AC = v
		ch.SendMessage(fmt.Sprintf("%s's AC set to %d.\r\n", target.GetName(), v))
	case "hitroll":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's hitroll: %d\r\n", target.GetName(), target.Hitroll))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.Hitroll = v
		ch.SendMessage(fmt.Sprintf("%s's hitroll set to %d.\r\n", target.GetName(), v))
	case "damroll":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's damroll: %d\r\n", target.GetName(), target.Damroll))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be numeric.\r\n")
			return true
		}
		target.Damroll = v
		ch.SendMessage(fmt.Sprintf("%s's damroll set to %d.\r\n", target.GetName(), v))
	case "exp", "experience":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's exp: %d\r\n", target.GetName(), target.Exp))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 {
			ch.SendMessage("Value must be >= 0.\r\n")
			return true
		}
		target.Exp = v
		ch.SendMessage(fmt.Sprintf("%s's exp set to %d.\r\n", target.GetName(), v))
	case "practices":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's practices: %d\r\n", target.GetName(), target.Practices))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 {
			ch.SendMessage("Value must be >= 0.\r\n")
			return true
		}
		target.Practices = v
		ch.SendMessage(fmt.Sprintf("%s's practices set to %d.\r\n", target.GetName(), v))
	case "roomflag":
		target.RoomFlags = !target.RoomFlags
		if target.RoomFlags {
			ch.SendMessage("Room flags on.\r\n")
		} else {
			ch.SendMessage("Room flags off.\r\n")
		}
	case "name":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's name: %s\r\n", target.GetName(), target.Name))
			return true
		}
		target.Name = value
		ch.SendMessage(fmt.Sprintf("Name set to %s.\r\n", value))
	case "sex":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's sex: %d\r\n", target.GetName(), target.Sex))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil || v < 0 || v > 2 {
			ch.SendMessage("Value must be 0-2 (male/female/neutral).\r\n")
			return true
		}
		target.Sex = v
		ch.SendMessage(fmt.Sprintf("%s's sex set to %d.\r\n", target.GetName(), v))
	case "room":
		if value == "" {
			ch.SendMessage(fmt.Sprintf("%s's room: %d\r\n", target.GetName(), target.RoomVNum))
			return true
		}
		v, err := strconv.Atoi(value)
		if err != nil {
			ch.SendMessage("Value must be a room vnum.\r\n")
			return true
		}
		target.RoomVNum = v
		ch.SendMessage(fmt.Sprintf("%s's room set to %d.\r\n", target.GetName(), v))
	default:
		ch.SendMessage(fmt.Sprintf("Unknown field: %s\r\n", field))
	}
	return true
}

// do_stat — character inspection command
func (w *World) doStat(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Stats on who or what?\r\n")
		return true
	}

	// Try room first
	if strings.HasPrefix(strings.ToLower(arg), "room") {
		room := w.GetRoomInWorld(ch.RoomVNum)
		if room != nil {
			ch.SendMessage(fmt.Sprintf("Room: [%d] %s\r\nSector: %d  Flags: %v\r\n", room.VNum, room.Name, room.Sector, room.Flags))
		}
		return true
	}

	// Try player
	targetPl, _ := w.findCharInRoom(ch, ch.RoomVNum, arg)
	if targetPl != nil {
		t := targetPl
		ch.SendMessage(fmt.Sprintf("Name: %s  Level: %d  Class: %d  Race: %d\r\n", t.Name, t.Level, t.Class, t.Race))
		ch.SendMessage(fmt.Sprintf("Hit: %d/%d  Mana: %d/%d  Move: %d/%d\r\n", t.Health, t.MaxHealth, t.Mana, t.MaxMana, t.Move, t.MaxMove))
		ch.SendMessage(fmt.Sprintf("AC: %d  Hitroll: %d  Damroll: %d  Align: %d\r\n", t.AC, t.Hitroll, t.Damroll, t.Alignment))
		ch.SendMessage(fmt.Sprintf("Gold: %d  Exp: %d  Practices: %d  Room: %d\r\n", t.Gold, t.Exp, t.Practices, t.RoomVNum))
		ch.SendMessage(fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d  Cha: %d\r\n",
			t.Stats.Str, t.Stats.Int, t.Stats.Wis, t.Stats.Dex, t.Stats.Con, t.Stats.Cha))
		return true
	}

	// Try object
	obj := w.findObjNear(ch, arg)
	if obj != nil {
		p := obj.Prototype
		ch.SendMessage(fmt.Sprintf("Object: %s  VNum: %d  Type: %d\r\n", p.ShortDesc, p.VNum, p.TypeFlag))
		ch.SendMessage(fmt.Sprintf("Keywords: %s  Weight: %d  Cost: %d\r\n", p.Keywords, p.Weight, p.Cost))
		return true
	}

	ch.SendMessage("Nothing around by that name.\r\n")
	return true
}

// do_gecho — broadcast message to entire game
func (w *World) doGecho(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("That must be a mistake...\r\n")
		return true
	}
	msg := arg + "\r\n"
	w.mu.RLock()
	for _, p := range w.players {
		p.SendMessage(msg)
	}
	w.mu.RUnlock()
	return true
}

// do_social — social action command
func (w *World) doSocial(ch *Player, me *MobInstance, cmd string, arg string) bool {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		ch.SendMessage("What social action?\r\n")
		return true
	}

	// Broadcast social message to the room
	parts := strings.SplitN(arg, " ", 2)
	socialName := strings.ToLower(parts[0])
	targetArg := ""
	if len(parts) > 1 {
		targetArg = parts[1]
	}

	// Simple social format: player does <social> [at <target>]
	if targetArg != "" {
		w.SendToRoom(ch.RoomVNum, fmt.Sprintf("%s %s %s.\r\n", ch.GetName(), socialName, targetArg))
	} else {
		w.SendToRoom(ch.RoomVNum, fmt.Sprintf("%s %s.\r\n", ch.GetName(), socialName))
	}
	return true
}

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
