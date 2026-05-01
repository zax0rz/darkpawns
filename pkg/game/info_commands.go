package game

import "fmt"

func (w *World) doScore(ch *Player, me *MobInstance, cmd string, arg string) bool {
	classNames := map[int]string{
		0: "Magic-User", 1: "Cleric", 2: "Thief", 3: "Warrior",
	}
	raceNames := map[int]string{
		0: "Human", 1: "Elf", 2: "Dwarf", 3: "Halfling", 4: "Gnome", 5: "Kender",
	}
	cn := classNames[ch.Class]
	if cn == "" {
		cn = "Unknown"
	}
	rn := raceNames[ch.Race]
	if rn == "" {
		rn = "Unknown"
	}
	ch.SendMessage(fmt.Sprintf("You are %s.\r\n", ch.GetName()))
	ch.SendMessage(fmt.Sprintf("Level %d %s %s.\r\n", ch.Level, rn, cn))
	ch.SendMessage(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d\r\n",
		ch.Health, ch.MaxHealth,
		ch.Mana, ch.MaxMana,
		ch.Move, ch.MaxMove))
	ch.SendMessage(fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d  Cha: %d\r\n",
		ch.Stats.Str, ch.Stats.Int, ch.Stats.Wis,
		ch.Stats.Dex, ch.Stats.Con, ch.Stats.Cha))
	ch.SendMessage(fmt.Sprintf("AC: %d  Experience: %d  Gold: %d\r\n",
		ch.AC, ch.Exp, ch.Gold))
	return true
}

// ---------------------------------------------------------------------------
// doWho — shows connected players (ACMD(do_who) in C)
// ---------------------------------------------------------------------------

func (w *World) doWho(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Players currently connected:\r\n")
	for _, p := range w.players {
		if p.Flags&PLR_INVISIBLE != 0 && ch.Level < LVL_IMMORT {
			continue
		}
		if p.Level >= 31 {
			ch.SendMessage(fmt.Sprintf("[%5s] %-12s %s\r\n", w.getWhoTitle(p), p.GetName(), "God"))
		} else {
			ch.SendMessage(fmt.Sprintf("[%5s] %-12s %s\r\n", w.getWhoTitle(p), p.GetName(), "Adventurer"))
		}
	}
	ch.SendMessage("\r\n")
	return true
}

func (w *World) getWhoTitle(ch *Player) string {
	return fmt.Sprintf("%2d", ch.Level)
}

// ---------------------------------------------------------------------------
// doInventory — shows carried items (ACMD(do_inventory) in C)
// ---------------------------------------------------------------------------

func (w *World) doInventory(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("You are carrying:\r\n")
	for i, item := range ch.Inventory.Items {
		if item == nil {
			continue
		}
		ch.SendMessage(fmt.Sprintf("[%2d] %s\r\n", i+1, item.Prototype.ShortDesc))
	}
	return true
}

// ---------------------------------------------------------------------------
// doEquipment — shows worn items (ACMD(do_equipment) in C)
// ---------------------------------------------------------------------------

func (w *World) doEquipment(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("You are using:\r\n")
	for slot := EquipmentSlot(0); slot < SlotMax; slot++ {
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if !ok {
			continue
		}
		ch.SendMessage(fmt.Sprintf("%-15s : %s\r\n", slot.String(), item.Prototype.ShortDesc))
	}
	return true
}

// ---------------------------------------------------------------------------
// doWhere — shows mobs in world (ACMD(do_where) in C)
// ---------------------------------------------------------------------------

func (w *World) doWhere(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Players in your area:\r\n")
	for _, p := range w.players {
		if p.RoomVNum == ch.RoomVNum {
			ch.SendMessage(fmt.Sprintf("%-20s : here\r\n", p.GetName()))
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// doLevels — shows class level titles (ACMD(do_levels) in C)
// ---------------------------------------------------------------------------

func (w *World) doLevels(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Level progression:\r\n")
	for lvl := 1; lvl <= ch.Level && lvl <= 50; lvl++ {
		ch.SendMessage(fmt.Sprintf("Level %2d: %d exp\r\n", lvl, 1000*lvl*lvl))
	}
	return true
}

// ---------------------------------------------------------------------------
// doColor / doToggle — configuration commands
// ---------------------------------------------------------------------------

func (w *World) doColor(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Color is not yet implemented.\r\n")
	return true
}

func (w *World) doToggle(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Toggles are not yet implemented.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doAbils / doSkills — shows abilities and learned skills
// ---------------------------------------------------------------------------

func (w *World) doAbils(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Abilities:\r\n")
	abilNames := []string{"Str", "Int", "Wis", "Dex", "Con", "Cha"}
	abilVals := []int{ch.Stats.Str, ch.Stats.Int, ch.Stats.Wis,
		ch.Stats.Dex, ch.Stats.Con, ch.Stats.Cha}
	for i := range abilNames {
		ch.SendMessage(fmt.Sprintf("%s: %d\r\n", abilNames[i], abilVals[i]))
	}
	return true
}

func (w *World) doSkills(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Skills:\r\n")
	for _, skill := range ch.SkillManager.GetLearnedSkills() {
		ch.SendMessage(fmt.Sprintf("%-20s %3d%%\r\n", skill.Name, skill.Level))
	}
	return true
}

// ---------------------------------------------------------------------------
// doUsers — shows all descriptors (ACMD(do_users) in C)
// ---------------------------------------------------------------------------

func (w *World) doUsers(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Connected users:\r\n")
	for _, p := range w.players {
		ch.SendMessage(fmt.Sprintf("%-20s %6d hp  Room %d\r\n",
			p.GetName(), p.Health, p.RoomVNum))
	}
	return true
}

// ---------------------------------------------------------------------------
// doExamine — examine objects or mobs
// ---------------------------------------------------------------------------

func (w *World) doExamine(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Examine what?\r\n")
		return true
	}
	w.lookAtTarget(ch, arg)

	// If it's a container, show contents
	obj := w.findObjNear(ch, arg)
	if obj != nil && (obj.Prototype.TypeFlag == ITEM_DRINKCON ||
		obj.Prototype.TypeFlag == ITEM_FOUNTAIN ||
		obj.Prototype.TypeFlag == ITEM_CONTAINER) {
		ch.SendMessage("When you look inside, you see:\r\n")
		w.lookInObj(ch, arg)
	}
	return true
}

// ---------------------------------------------------------------------------
// doCoins — shows money on person
// ---------------------------------------------------------------------------

func (w *World) doCoins(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage(fmt.Sprintf("You have %d gold coins.\r\n", ch.Gold))
	return true
}

// ---------------------------------------------------------------------------
// doDescription — sets character description
// ---------------------------------------------------------------------------

func (w *World) doDescription(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.Description = arg
	ch.SendMessage("Description set.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doCommands — shows available commands
// ---------------------------------------------------------------------------

func (w *World) doCommands(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Commands available:\r\n")
	ch.SendMessage("look, read, examine, score, who, inventory, equipment\r\n")
	ch.SendMessage("time, weather, exits, consider, where\r\n")
	ch.SendMessage("north, east, south, west, up, down\r\n")
	ch.SendMessage("get, drop, give, put, wear, wield, remove\r\n")
	ch.SendMessage("open, close, lock, unlock\r\n")
	ch.SendMessage("kill, backstab, flee, assist, hit, murder\r\n")
	ch.SendMessage("sneak, hide, steal, practice, train, level\r\n")
	ch.SendMessage("skills, abilities, help, save, quit\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doDiagnose — check a mob's condition
// ---------------------------------------------------------------------------

func (w *World) doDiagnose(ch *Player, me *MobInstance, cmd string, arg string) bool {
	ch.SendMessage("Diagnose not yet fully implemented.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doConsider — compare target to player (ACMD(do_consider) in C)
// ---------------------------------------------------------------------------

func (w *World) doConsider(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Consider who?\r\n")
		return true
	}
	ch.SendMessage("You consider your options...\r\n")
	return true
}

// ---------------------------------------------------------------------------
// doHelp — shows help topics (ACMD(do_help) in C)
// ---------------------------------------------------------------------------

func (w *World) doHelp(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if arg == "" {
		ch.SendMessage("Usage: help <topic>\r\n")
		return true
	}
	ch.SendMessage(fmt.Sprintf("No help on '%s' available.\r\n", arg))
	return true
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

