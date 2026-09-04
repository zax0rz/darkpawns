package session

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func cmdStat(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) == 0 {
		s.Send("Stats on who or what?\r\n")
		return nil
	}
	target := strings.Join(args, " ")
	if strings.ToLower(args[0]) == "room" {
		if s.manager == nil || s.manager.world == nil {
			s.Send("World not available.")
			return nil
		}
		room := s.manager.world.GetRoomInWorld(s.player.GetRoom())
		if room == nil {
			s.Send("Room data not found.")
			return nil
		}
		s.Send(fmt.Sprintf("Room: %s  VNum: [%d]  Zone: [%d]  Sector: [%d]", room.Name, room.VNum, room.Zone, room.Sector))
		if room.Description != "" {
			s.Send(fmt.Sprintf("Desc: %s", room.Description))
		}
		return nil
	}
	selector := strings.ToLower(args[0])
	switch selector {
	case "mob":
		if len(args) == 1 {
			s.Send("Stats on which mobile?\r\n")
			return nil
		}
		if target, ok := s.manager.world.ResolveCharWorld(s.player, args[1]); ok && target.Combatant != nil && target.Combatant.IsNPC() {
			s.sendStatMob(target.Combatant.(*game.MobInstance))
			return nil
		}
		s.Send("No such mobile around.\r\n")
		return nil
	case "player":
		if len(args) == 1 {
			s.Send("Stats on which player?\r\n")
			return nil
		}
		if sess := findSessionByName(s.manager, args[1]); sess != nil && sess.player != nil {
			s.sendStatPlayer(sess.player)
			return nil
		}
		s.Send("No such player around.\r\n")
		return nil
	case "file":
		if len(args) == 1 {
			s.Send("Stats on which player?\r\n")
			return nil
		}
		if !game.PlayerSaveExists(args[1]) {
			s.Send("There is no such player.\r\n")
			return nil
		}
	case "obj", "object":
		if len(args) == 1 {
			s.Send("Stats on which object?\r\n")
			return nil
		}
		s.sendStatObject(args[1])
		return nil
	}
	if sess := findSessionByName(s.manager, target); sess != nil && sess.player != nil {
		s.sendStatPlayer(sess.player)
		return nil
	}
	s.Send("Nothing around by that name.\r\n")
	return nil
}

func (s *Session) sendStatMob(mob *game.MobInstance) {
	if mob == nil || mob.Prototype == nil {
		return
	}
	s.Send(fmt.Sprintf("Name: %s  Level: %d", mob.GetName(), mob.GetLevel()))
}

func (s *Session) sendStatPlayer(p *game.Player) {
	if p == nil {
		return
	}
	s.Send(fmt.Sprintf("Name: %s  Level: %d  Class: %d  Race: %d  Alignment: %d",
		p.Name, p.Level, p.Class, p.Race, p.Alignment))
	s.Send(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d",
		p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove))
	s.Send(fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d  Cha: %d",
		p.Stats.Str, p.Stats.Int, p.Stats.Wis, p.Stats.Dex, p.Stats.Con, p.Stats.Cha))
	s.Send(fmt.Sprintf("Gold: %d  Exp: %d  Hitroll: %+d  Damroll: %+d  AC: %d  THAC0: %d",
		p.Gold, p.Exp, p.Hitroll, p.Damroll, p.AC, p.THAC0))

	posNames := map[int]string{
		0: "Dead", 1: "Mortally Wounded", 2: "Incapacitated",
		3: "Stunned", 4: "Sleeping", 5: "Resting", 6: "Sitting", 7: "Standing",
	}
	pos := p.Position
	if name, ok := posNames[int(pos)]; ok {
		s.Send(fmt.Sprintf("Position: %s", name))
	} else {
		s.Send(fmt.Sprintf("Position: %d", pos))
	}
	s.Send(fmt.Sprintf("Thirst: %d  Hunger: %d  Drunk: %d", p.Thirst, p.Hunger, p.Drunk))
	if len(p.Conditions) == 3 {
		s.Send(fmt.Sprintf("Conditions: Drunk=%d Full=%d Thirst=%d",
			p.Conditions[0], p.Conditions[1], p.Conditions[2]))
	}
	if p.Flags != 0 {
		s.Send(fmt.Sprintf("Flags: %d", p.Flags))
	}
}

func (s *Session) sendStatObject(name string) {
	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return
	}
	// Try as vnum first
	vnum, err := strconv.Atoi(name)
	if err == nil {
		if proto, ok := s.manager.world.GetObjPrototype(vnum); ok {
			s.sendObjProto(proto)
			return
		}
		s.Send("No object with that VNum.\r\n")
		return
	}
	// Search by keyword
	pw := s.manager.world.GetParsedWorld()
	if pw == nil {
		s.Send("World data not loaded.")
		return
	}
	nameLower := strings.ToLower(name)
	for i := range pw.Objs {
		if strings.Contains(strings.ToLower(pw.Objs[i].ShortDesc), nameLower) ||
			strings.Contains(strings.ToLower(pw.Objs[i].Keywords), nameLower) {
			s.sendObjProto(&pw.Objs[i])
			return
		}
	}
	s.Send("No such object around.\r\n")
}

func (s *Session) sendObjProto(o *parser.Obj) {
	s.Send(fmt.Sprintf("Object: [%d] %s\r\n", o.VNum, o.ShortDesc))
	s.Send(fmt.Sprintf("Keywords: %s\r\n", o.Keywords))
	s.Send(fmt.Sprintf("Type: %d  Weight: %d  Cost: %d\r\n", o.TypeFlag, o.Weight, o.Cost))
	s.Send(fmt.Sprintf("ExtraFlags: %v  WearFlags: %v\r\n", o.ExtraFlags, o.WearFlags))
	s.Send(fmt.Sprintf("Values: [%d] [%d] [%d] [%d]\r\n", o.Values[0], o.Values[1], o.Values[2], o.Values[3]))
	if len(o.Affects) > 0 {
		s.Send("Affects:")
		for _, aff := range o.Affects {
			s.Send(fmt.Sprintf("  Apply: %d  Modifier: %d\r\n", aff.Location, aff.Modifier))
		}
	}
	if o.ScriptName != "" {
		s.Send(fmt.Sprintf("Script: %s\r\n", o.ScriptName))
	}
}

// cmdVnum — find vnums by keyword (LVL_IMMORT)
func cmdVnum(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: vnum { obj | mob } <name>\r\n")
		return nil
	}

	category := strings.ToLower(args[0])
	keyword := strings.ToLower(strings.Join(args[1:], " "))
	if category != "mob" && category != "obj" && category != "object" {
		s.Send("Usage: vnum { obj | mob } <name>\r\n")
		return nil
	}

	if s.manager == nil || s.manager.world == nil {
		s.Send("World not available.")
		return nil
	}

	parsed := s.manager.world.GetParsedWorld()
	if parsed == nil {
		s.Send("Parsed world data not available.")
		return nil
	}

	results := make([]string, 0, 30)
	switch category {
	case "mob":
		for i := range parsed.Mobs {
			m := &parsed.Mobs[i]
			if strings.Contains(strings.ToLower(m.Keywords), keyword) || strings.Contains(strings.ToLower(m.ShortDesc), keyword) {
				results = append(results, fmt.Sprintf("[%5d] %s", m.VNum, m.ShortDesc))
				if len(results) >= 30 {
					results = append(results, fmt.Sprintf("... %d more matching mobs", len(parsed.Mobs)-i))
					break
				}
			}
		}
	case "obj", "object":
		for i := range parsed.Objs {
			o := &parsed.Objs[i]
			if strings.Contains(strings.ToLower(o.Keywords), keyword) || strings.Contains(strings.ToLower(o.ShortDesc), keyword) {
				results = append(results, fmt.Sprintf("[%5d] %s", o.VNum, o.ShortDesc))
				if len(results) >= 30 {
					results = append(results, fmt.Sprintf("... %d more matching objects", len(parsed.Objs)-i))
					break
				}
			}
		}
	}

	if len(results) == 0 {
		if category == "mob" {
			s.Send("No mobiles by that name.\r\n")
		} else {
			s.Send("No objects by that name.\r\n")
		}
		return nil
	}
	s.Send(fmt.Sprintf("%s matching %q (%d found):", category, keyword, len(results)))
	for _, r := range results {
		s.Send(r)
	}
	return nil
}

// cmdVstat — detailed vnum info for prototypes (LVL_IMMORT)
func cmdVstat(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Usage: vstat { obj | mob } <number>\r\n")
		return nil
	}

	cat := strings.ToLower(args[0])
	vnum, err := strconv.Atoi(args[1])
	if err != nil {
		s.Send("Usage: vstat { obj | mob } <number>\r\n")
		return nil
	}

	w := s.manager.world

	switch cat {
	case "mob":
		proto, ok := w.GetMobPrototype(vnum)
		if !ok {
			s.Send("There is no monster with that number.\r\n")
			return nil
		}
		s.Send(fmt.Sprintf("Mob: [%5d] %-30s\r\n", proto.VNum, proto.ShortDesc))
		s.Send(fmt.Sprintf("Keywords: %s\r\n", proto.Keywords))
		s.Send(fmt.Sprintf("Level: %d  HP: %s  AC: %d  THAC0: %d\r\n", proto.Level, proto.HP.String(), proto.AC, proto.THAC0))
		s.Send(fmt.Sprintf("Damage: %s  Alignment: %d  Exp: %d  Gold: %d\r\n", proto.Damage.String(), proto.Alignment, proto.Exp, proto.Gold))
		s.Send(fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d  Cha: %d\r\n",
			proto.Str, proto.Int, proto.Wis, proto.Dex, proto.Con, proto.Cha))
		s.Send(fmt.Sprintf("Sex: %d  Weight: %d  Height: %d  Race: %s\r\n",
			proto.Sex, proto.Weight, proto.Height, proto.RaceStr))
		if proto.ScriptName != "" {
			s.Send(fmt.Sprintf("Script: %s\r\n", proto.ScriptName))
		}

	case "obj":
		proto, ok := w.GetObjPrototype(vnum)
		if !ok {
			s.Send("There is no object with that number.\r\n")
			return nil
		}
		s.Send(fmt.Sprintf("Object: [%5d] %s\r\n", proto.VNum, proto.ShortDesc))
		s.Send(fmt.Sprintf("Keywords: %s\r\n", proto.Keywords))
		s.Send(fmt.Sprintf("Type: %d  Weight: %d  Cost: %d\r\n", proto.TypeFlag, proto.Weight, proto.Cost))
		s.Send(fmt.Sprintf("ExtraFlags: %v  WearFlags: %v\r\n", proto.ExtraFlags, proto.WearFlags))
		s.Send(fmt.Sprintf("Values: [%d] [%d] [%d] [%d]\r\n", proto.Values[0], proto.Values[1], proto.Values[2], proto.Values[3]))
		if len(proto.Affects) > 0 {
			s.Send("Affects:")
			for _, aff := range proto.Affects {
				s.Send(fmt.Sprintf("  Apply: %d  Modifier: %d\r\n", aff.Location, aff.Modifier))
			}
		}

	default:
		s.Send("That'll have to be either 'obj' or 'mob'.\r\n")
	}
	return nil
}

// cmdWizlock — toggle wizard-only login (LVL_IMPL)
