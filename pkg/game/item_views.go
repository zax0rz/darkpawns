package game

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Inventory / equipment display — ported from C act.informative.c
// ---------------------------------------------------------------------------

// cWearToGoSlot maps each C WEAR_ index to the Go EquipmentSlot that stores
// that item. A negative value means the C position has no Go equivalent.
var cWearToGoSlot = []EquipmentSlot{
	0:  SlotLight,
	1:  SlotFingerR,
	2:  SlotFingerL,
	3:  SlotNeck1,
	4:  SlotNeck2,
	5:  SlotBody,
	6:  SlotHead,
	7:  SlotLegs,
	8:  SlotFeet,
	9:  SlotHands,
	10: SlotArms,
	11: SlotShield,
	12: SlotAbout,
	13: SlotWaist,
	14: SlotWristR,
	15: SlotWristL,
	16: SlotWield,
	17: SlotHold,
	18: -1, // THROW — no Go slot
	19: -1, // ABLEGS — no Go slot
	20: -1, // FACE — no Go slot
	21: -1, // HOVER — no Go slot
}

// cWearSlot maps a C WEAR_ position to the Go EquipmentSlot used for storage.
// THROW, ABLEGS, FACE, and HOVER do not yet have Go storage slots.
func cWearSlot(where int) (EquipmentSlot, bool) {
	if where < 0 || where >= len(cWearToGoSlot) {
		return 0, false
	}
	slot := cWearToGoSlot[where]
	if slot < 0 {
		return 0, false
	}
	return slot, true
}

// cWearWhere is the fixed-width label printed by C's do_equipment().
// Trailing spaces are load-bearing and copied from src/constants.c.
var cWearWhere = []string{
	0:  "<used as light>      ",
	1:  "<worn on finger>     ",
	2:  "<worn on finger>     ",
	3:  "<worn around neck>   ",
	4:  "<worn around neck>   ",
	5:  "<worn on body>       ",
	6:  "<worn on head>       ",
	7:  "<worn on legs>       ",
	8:  "<worn on feet>       ",
	9:  "<worn on hands>      ",
	10: "<worn on arms>       ",
	11: "<worn as shield>     ",
	12: "<worn about body>    ",
	13: "<worn about waist>   ",
	14: "<worn around wrist>  ",
	15: "<worn around wrist>  ",
	16: "<wielded>            ",
	17: "<held>               ",
	18: "<held>               ",
	19: "<worn about legs>    ",
	20: "<worn on face>       ",
	21: "<hovering near head> ",
}

// DoEquipment prints the player's equipped items in C WEAR_ order.
// Source: C do_equipment() (src/act.informative.c:1470-1495)
func (w *World) DoEquipment(ch *Player) {
	var b strings.Builder
	b.WriteString("You are using:\r\n")

	found := false
	for i := 0; i < len(cWearToGoSlot); i++ {
		slot := cWearToGoSlot[i]
		if slot < 0 {
			continue
		}
		item, ok := ch.Equipment.GetItemInSlot(slot)
		if !ok || item == nil {
			continue
		}
		found = true
		b.WriteString(cWearWhere[i])
		if chCanSeeObj(ch, item) {
			b.WriteString(item.GetShortDesc())
			b.WriteString(coloredObjectVisibleFlags(ch, item))
			b.WriteString("\r\n")
		} else {
			b.WriteString("Something.\r\n")
		}
	}

	if !found {
		b.WriteString(" Nothing.\r\n")
	}
	ch.SendMessage(b.String())
}

// DoInventory prints the player's inventory with C's object-clumping format.
// Source: C do_inventory() + list_obj_to_char() + oc_show_list() (mode 15)
func (w *World) DoInventory(ch *Player) {
	var b strings.Builder
	b.WriteString("You are carrying:\r\n")

	var items []*ObjectInstance
	if ch.Inventory != nil {
		items = ch.Inventory.Items
	}
	b.WriteString(w.renderObjectListMode15(ch, items))
	ch.SendMessage(b.String())
}

// renderObjectListMode15 renders a slice of objects the way C list_obj_to_char
// does for mode 15 (short desc, weights, wide list, indent, Num/Item/
// Encumbrance header). An empty/all-invisible list renders "Nothing.". Grouping
// and reverse discovery order mirror oc_add_front + oc_show_list. This block is
// sent raw (like C send_to_char), so it is NOT routed through the capitalizing
// act() path used for ordinary observation lines.
func (w *World) renderObjectListMode15(ch *Player, items []*ObjectInstance) string {
	type group struct {
		line  string
		count int
		item  *ObjectInstance
	}
	groups := make(map[string]*group)
	order := make([]string, 0)

	for _, item := range items {
		if item == nil || !chCanSeeObj(ch, item) {
			continue
		}
		text := item.GetShortDesc()
		// C GET_OBJ_WEIGHT for containers includes contents after weight_change_object().
		weight := item.GetTotalWeight()
		key := fmt.Sprintf("%d:%d:%d:%s", item.GetVNum(), item.GetExtraFlags()[0], weight, text)
		if existing := groups[key]; existing != nil {
			existing.count++
			continue
		}
		groups[key] = &group{line: text, count: 1, item: item}
		order = append(order, key)
	}

	if len(groups) == 0 {
		return "Nothing.\r\n"
	}

	var b strings.Builder
	// mode = 15 -> short descr, show weights, wide list, indent, header
	b.WriteString("\r\n Num  Item   " + strings.Repeat(" ", 51) + "Encumbrance\r\n")
	b.WriteString("-------------------------------------------------------------------------------\r\n")

	// C builds the list via oc_add_front(), so oc_show_list() renders reverse
	// discovery order.
	for i := len(order) - 1; i >= 0; i-- {
		g := groups[order[i]]
		b.WriteString(formatOCShowListLine(g.line, g.count, g.item.GetTotalWeight(), g.item, ch))
	}

	return b.String()
}

// formatOCShowListLine renders one entry the way C oc_show_list() does for
// mode 15 (wide, indented, with weights).
func formatOCShowListLine(text string, count, weight int, item *ObjectInstance, ch *Player) string {
	var b strings.Builder

	if count == 1 {
		b.WriteString("  1   ")
		fmt.Fprintf(&b, "%-63s", text)
		fmt.Fprintf(&b, "%2d pt%s", weight, pluralPT(weight))
	} else {
		fmt.Fprintf(&b, " %2d   ", count)
		fmt.Fprintf(&b, "%-63s", text)
		fmt.Fprintf(&b, "%2d pt%s ea.", weight, pluralPT(weight))
	}

	b.WriteString("\r\n")

	extras := item.GetExtraFlags()[0]
	firstExtra := true
	appendExtra := func(desc string) {
		if firstExtra {
			b.WriteString("          ")
			firstExtra = false
		}
		b.WriteString(desc)
	}

	if extras&(1<<itemExtraInvisible) != 0 {
		appendExtra("...it is invisible ")
	}
	if extras&(1<<itemExtraBless) != 0 && ch.IsAffected(affDetectAlign) {
		appendExtra("...it glows blue")
	}
	if extras&(1<<itemExtraMagic) != 0 && ch.IsAffected(affDetectMagic) {
		appendExtra("...it glows gold")
	}
	if extras&(1<<itemExtraGlow) != 0 {
		appendExtra("...it glows white")
	}
	if extras&(1<<itemExtraHum) != 0 {
		appendExtra("...it is humming")
	}

	if !firstExtra {
		b.WriteString("\r\n")
	}

	return b.String()
}

func pluralPT(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Item extra-flag bits used by the view helpers (src/structs.h).
const (
	itemExtraGlow      = 0
	itemExtraHum       = 1
	itemExtraInvisible = 5
	itemExtraMagic     = 6
	itemExtraBless     = 8
)
