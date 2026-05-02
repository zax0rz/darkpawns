package session

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdEat implements the eat command.
// Source: src/act.item.c ACMD(do_eat)
func cmdEat(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Eat what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find item in inventory by keyword
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.Send(fmt.Sprintf("You don't seem to have %s.", itemName))
		return nil
	}

	// Check it's actually food
	if item.GetTypeFlag() != 19 { // ITEM_FOOD
		s.Send("You can't eat THAT!")
		return nil
	}

	// Check stomach isn't overfull — C: GET_COND(ch, FULL) > 40
	if s.player.Hunger > 40 {
		s.Send("You are too full to eat more!")
		return nil
	}

	// Eat message
	s.Send(fmt.Sprintf("You eat %s.", item.GetShortDesc()))
	broadcastToRoom(s, fmt.Sprintf("%s eats %s.", s.player.Name, item.GetShortDesc()))

	// Apply food effect via game.EatFood (which calls GainCondition for CondFull)
	amount, err := game.EatFood(s.player, item)
	if err != nil {
		return fmt.Errorf("game.EatFood: %w", err)
	}

	// Full message after eating
	if s.player.Hunger > 20 {
		s.Send("You are full.")
	}

	// Check for poison — Values[3] is poison flag (C: GET_OBJ_VAL(food, 3))
	isPoisoned := item.Prototype != nil && item.Prototype.Values[3] != 0

	if isPoisoned {
		s.Send("Oops, that tasted rather strange!")
		broadcastToRoom(s, fmt.Sprintf("%s coughs and utters some strange sounds.", s.player.Name))

		// Apply poison affect (C: af.type = SPELL_POISON, duration = amount * 2)
		poisonAffect := engine.NewAffect(engine.AffectPoison, amount*2, 0, item.GetShortDesc())
		s.player.ActiveAffects = append(s.player.ActiveAffects, poisonAffect)
	}

	// Remove food from inventory
	s.player.Inventory.RemoveItem(item)
	s.markDirty(VarInventory)

	return nil
}

// cmdDrink implements the drink command.
// Source: src/act.item.c ACMD(do_drink)
func cmdDrink(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Drink from what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find container in inventory first, then room
	item, found := s.player.Inventory.FindItem(itemName)
	var onGround bool

	if !found {
		roomVNum := s.player.GetRoom()
		roomItems := s.manager.world.GetItemsInRoom(roomVNum)
		for _, roomItem := range roomItems {
			keywords := strings.ToLower(roomItem.GetKeywords())
			shortDesc := strings.ToLower(roomItem.GetShortDesc())
			search := strings.ToLower(itemName)
			if strings.Contains(keywords, search) || strings.Contains(shortDesc, search) {
				item = roomItem
				onGround = true
				break
			}
		}
	}

	if item == nil {
		s.Send("You can't find it!")
		return nil
	}

	// Check it's a drink container or fountain
	objType := item.GetTypeFlag()
	if objType != 17 && objType != 23 { // ITEM_DRINKCON, ITEM_FOUNTAIN
		s.Send("You can't drink from that!")
		return nil
	}

	// Ground drink containers must be picked up first
	if onGround && objType == 17 { // ITEM_DRINKCON
		s.Send("You have to be holding that to drink from it.")
		return nil
	}

	// Drunk check — C: (GET_COND(ch, DRUNK) > 10) && (GET_COND(ch, THIRST) > 0)
	if s.player.Drunk > 10 && s.player.Thirst > 0 {
		s.Send("You can't seem to get close enough to your mouth.")
		broadcastToRoom(s, fmt.Sprintf("%s tries to drink but misses %s mouth!", s.player.Name, "their"))
		return nil
	}

	// Full check — C: (GET_COND(ch, FULL) > 20) && (GET_COND(ch, THIRST) > 0)
	if s.player.Hunger > 20 && s.player.Thirst > 0 {
		s.Send("Your stomach can't contain anymore!")
		return nil
	}

	// Thirst max check — C: GET_COND(ch, THIRST) > 40
	if s.player.Thirst > 40 {
		s.Send("If you drink any more, you'll explode!")
		return nil
	}

	// Check container has liquid — Values[1] = drinks left
	if item.Prototype == nil || item.Prototype.Values[1] <= 0 {
		s.Send("It's empty.")
		return nil
	}

	// Get liquid type
	liqIndex := item.Prototype.Values[2] // Values[2] = liquid type

	// Drink message
	liq := game.DrinkName(liqIndex)
	s.Send(fmt.Sprintf("You drink the %s.", liq))
	broadcastToRoom(s, fmt.Sprintf("%s drinks from %s.", s.player.Name, item.GetShortDesc()))

	// Call game.DrinkLiquid which handles condition updates
	amount, _, err := game.DrinkLiquid(s.player, item)
	if err != nil {
		slog.Error("drink failed", "error", err)
		return nil
	}

	// Condition messages (C: checks after drinking)
	if s.player.Drunk > 10 {
		s.Send("You feel drunk.")
	}
	if s.player.Thirst > 20 {
		s.Send("You don't feel thirsty any more.")
	}
	if s.player.Hunger > 20 {
		s.Send("You are full.")
	}

	// Check for poison — Values[3] = poison flag
	if item.Prototype.Values[3] != 0 {
		s.Send("Oops, it tasted rather strange!")
		broadcastToRoom(s, fmt.Sprintf("%s chokes and utters some strange sounds.", s.player.Name))

		// Apply poison affect (C: duration = amount * 3)
		poisonAffect := engine.NewAffect(engine.AffectPoison, amount*3, 0, "poisoned drink")
		s.player.ActiveAffects = append(s.player.ActiveAffects, poisonAffect)
	}

	// Update container liquid amount (DrinkLiquid doesn't modify prototype values)
	if objType == 17 { // ITEM_DRINKCON — reduce liquid, fountains are infinite
		item.Prototype.Values[1] -= amount
		if item.Prototype.Values[1] <= 0 {
			item.Prototype.Values[1] = 0
			item.Prototype.Values[2] = 0 // reset liquid type
			item.Prototype.Values[3] = 0 // reset poison
		}
	}

	return nil
}

// cmdQuaff implements the quaff command.
// Source: src/act.other.c (do_use SCMD_QUAFF) + src/spell_parser.c (mag_objectmagic ITEM_POTION)
func cmdQuaff(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Quaff what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find potion in inventory
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.Send("You don't seem to have that.")
		return nil
	}

	// Check it's a potion
	if item.GetTypeFlag() != 10 { // ITEM_POTION
		s.Send("You can only quaff potions.")
		return nil
	}

	// Can't quaff while sitting (C: spell_parser.c line check)
	if s.player.GetPosition() == combat.PosSitting {
		s.Send("You can't do this sitting!")
		return nil
	}

	// Quaff message
	s.Send(fmt.Sprintf("You quaff %s.", item.GetShortDesc()))
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		broadcastToRoom(s, fmt.Sprintf("%s %s", s.player.Name, item.Prototype.ActionDesc))
	} else {
		broadcastToRoom(s, fmt.Sprintf("%s quaffs %s.", s.player.Name, item.GetShortDesc()))
	}

	// Apply potion effects
	// In C: for i=1; i<4; i++ call_magic(ch, ch, NULL, GET_OBJ_VAL(obj, i), GET_OBJ_VAL(obj, 0), CAST_POTION)
	// Values[0] = spell level, Values[1-3] = spell numbers
	// For now, apply object affects as stat modifiers
	if item.Prototype != nil {
		for _, aff := range item.Prototype.Affects {
			applyAffect(s.player, aff.Location, aff.Modifier, item.GetShortDesc())
		}
	}

	// Remove potion from inventory
	s.player.Inventory.RemoveItem(item)
	s.markDirty(VarInventory)

	return nil
}

// applyAffect applies a stat/HP/mana/move modifier from a potion affect.
// Location values from CircleMUD structs.h APPLY_* constants.
func applyAffect(p *game.Player, location, modifier int, source string) {
	switch location {
	case 1: // APPLY_STR
		p.Stats.Str += modifier
	case 2: // APPLY_DEX
		p.Stats.Dex += modifier
	case 3: // APPLY_INT
		p.Stats.Int += modifier
	case 4: // APPLY_WIS
		p.Stats.Wis += modifier
	case 5: // APPLY_CON
		p.Stats.Con += modifier
	case 12: // APPLY_HIT (HP)
		p.Health += modifier
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
		if p.Health < 0 {
			p.Health = 0
		}
	case 13: // APPLY_MANA
		p.Mana += modifier
		if p.Mana > p.MaxMana {
			p.Mana = p.MaxMana
		}
		if p.Mana < 0 {
			p.Mana = 0
		}
	case 14: // APPLY_MOVE
		p.Move += modifier
		if p.Move > p.MaxMove {
			p.Move = p.MaxMove
		}
		if p.Move < 0 {
			p.Move = 0
		}
	case 17: // APPLY_HITROLL
		p.Hitroll += modifier
	case 18: // APPLY_DAMROLL
		p.Damroll += modifier
	}
}
