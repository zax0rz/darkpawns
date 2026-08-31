package session

import (
	"fmt"
	"io"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// These tables are the C constants.c tables consumed by do_idlist. Keep their
// ordering intact: sprinttype/sprintbitarray use numeric positions, not names.
var idlistItemTypes = []string{
	"UNDEFINED", "LIGHT", "SCROLL", "WAND", "STAFF", "WEAPON",
	"FIRE WEAPON", "MISSILE", "TREASURE", "ARMOR", "POTION", "WORN",
	"OTHER", "TRASH", "TRAP", "CONTAINER", "NOTE", "LIQ CONTAINER",
	"KEY", "FOOD", "MONEY", "PEN", "BOAT", "FOUNTAIN",
}

var idlistAffectedBits = []string{
	"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE",
	"WATERWALK", "SANCT", "GROUP", "CURSE", "INFRA", "POISON", "PROT-EVIL",
	"PROT-GOOD", "SLEEP", "!TRACK", "FLESH-ALTER", "DODGE", "SNEAK", "HIDE",
	"BERSERK", "CHARM", "FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY",
	"WEREWOLF", "VAMPIRE", "MOUNTED", "INVULN", "FLAMING", "NOTHING", "HASTE",
	"SLOW", "DREAM", "WATERBREATHE", "METALSKIN", "ROBBED",
}

var idlistExtraBits = []string{
	"GLOW", "HUM", "!RENT", "!DONATE", "!INVIS", "INVIS", "MAGIC", "!DROP",
	"BLESS", "!GOOD", "!EVIL", "!NEU", "!MAGE", "!CLE", "!THI", "!WAR",
	"!SELL", "NAMED", "!PSI", "!NIN", "!PAL", "!MAGUS", "!ASS", "!AVA",
	"RARE", "!LOCATE", "!RAN", "!MYS", "TWOHANDS",
}

var idlistApplyTypes = []string{
	"NONE", "STR", "DEX", "INT", "WIS", "CON", "CHA", "CLASS", "LEVEL", "AGE",
	"CHAR_WEIGHT", "CHAR_HEIGHT", "MAXMANA", "MAXHIT", "MAXMOVE", "GOLD", "EXP",
	"ARMOR", "HITROLL", "DAMROLL", "SAVING_PARA", "SAVING_ROD", "SAVING_PETRI",
	"SAVING_BREATH", "SAVING_SPELL", "RACE_HATE", "HIT_REGEN", "MANA_REGEN",
	"MOVE_REGEN", "PERM_SPELL",
}

var idlistMobRaces = []string{
	"Human", "Elf", "Dwarf", "Kender", "Centaur", "Rakshasa", "Troll", "Lycanthrope",
	"Vampire", "Undead", "Dragon", "Demon", "Horse", "Reptile", "Arachnid", "Rodent",
	"Other", "Vegetable", "Giant", "Demi-god", "Ogre", "Insect", "Mammal", "Fish",
	"Avian", "Magical Construct", "Amphibian", "Humanoid", "Faery", "Ssaur", "Minotaur",
}

func idlistTypeName(value int) string {
	if value >= 0 && value < len(idlistItemTypes) {
		return idlistItemTypes[value]
	}
	return "UNDEFINED"
}

func idlistApplyName(value int) string {
	if value >= 0 && value < len(idlistApplyTypes) {
		return idlistApplyTypes[value]
	}
	return "UNDEFINED"
}

func idlistMobRaceName(value int) string {
	if value >= 0 && value < len(idlistMobRaces) {
		return idlistMobRaces[value]
	}
	return "UNDEFINED"
}

// idlistBitArray mirrors C's sprintbitarray for the four-word object flag
// arrays. parser.Obj has no prototype bitvector field because the C object
// file format does not encode one; do_idlist therefore receives four zero
// words for "Item will give the following abilities".
func idlistBitArray(words [4]int, names []string) string {
	var out strings.Builder
	for word := range words {
		bits := uint32(words[word])
		for bit := 0; bits != 0; bit++ {
			if bits&1 != 0 {
				index := word*32 + bit
				name := "UNDEFINED"
				if index < len(names) {
					name = names[index]
				}
				out.WriteString(name)
				out.WriteByte(' ')
			}
			bits >>= 1
		}
	}
	if out.Len() == 0 {
		return "NOBITS "
	}
	return out.String()
}

func idlistSpellName(value int) string {
	return spells.SpellRawName(value)
}

// writeObjectIDList is the report body of C do_idlist. It returns the number
// of prototypes written so the caller can retain a useful audit log without
// inventing a player-facing count (src/act.wizard.c:3594-3690).
func writeObjectIDList(w io.Writer, objects []parser.Obj) (int, error) {
	for i := range objects {
		obj := &objects[i]
		if _, err := fmt.Fprintf(w, "Object: '%s', Item type: ", obj.ShortDesc); err != nil {
			return 0, err
		}
		if obj.TypeFlag == 3 || obj.TypeFlag == 4 {
			if _, err := fmt.Fprintln(w, "CASTING_EQ"); err != nil {
				return 0, err
			}
		} else if _, err := fmt.Fprintln(w, idlistTypeName(obj.TypeFlag)); err != nil {
			return 0, err
		}

		if _, err := fmt.Fprintf(w, "Item will give the following abilities:  %s\n", idlistBitArray([4]int{}, idlistAffectedBits)); err != nil {
			return 0, err
		}
		if _, err := fmt.Fprintf(w, "Item is: %s\n", idlistBitArray(obj.ExtraFlags, idlistExtraBits)); err != nil {
			return 0, err
		}
		if _, err := fmt.Fprintf(w, "Encumbrance: %d, Value: %d\n", obj.Weight, obj.Cost); err != nil {
			return 0, err
		}

		switch obj.TypeFlag {
		case 2, 10: // ITEM_SCROLL, ITEM_POTION
			if _, err := fmt.Fprintf(w, "This %s casts: ", idlistTypeName(obj.TypeFlag)); err != nil {
				return 0, err
			}
			for _, value := range obj.Values[1:] {
				if value >= 1 {
					if _, err := fmt.Fprint(w, idlistSpellName(value)); err != nil {
						return 0, err
					}
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return 0, err
			}
		case 3, 4: // ITEM_WAND, ITEM_STAFF
			if _, err := fmt.Fprintf(w, "This %s casts: %sIt has %d maximum charge%s and %d remaining.\n\n",
				idlistTypeName(obj.TypeFlag), idlistSpellName(obj.Values[3]), obj.Values[1], plural(obj.Values[1]), obj.Values[2]); err != nil {
				return 0, err
			}
		case 5: // ITEM_WEAPON
			average := (float64(obj.Values[2]+1) / 2.0) * float64(obj.Values[1])
			if _, err := fmt.Fprintf(w, "Damage Dice is '%dD%d' for an average per-round damage of %.1f.\n", obj.Values[1], obj.Values[2], average); err != nil {
				return 0, err
			}
		case 9: // ITEM_ARMOR
			if _, err := fmt.Fprintf(w, "AC-apply is %d\n", obj.Values[0]); err != nil {
				return 0, err
			}
		}

		found := false
		for _, affect := range obj.Affects {
			if affect.Location == 0 || affect.Modifier == 0 {
				continue
			}
			if !found {
				if _, err := fmt.Fprintln(w, "Can affect you as :"); err != nil {
					return 0, err
				}
				found = true
			}
			if affect.Location == 25 { // APPLY_RACE_HATE
				if _, err := fmt.Fprintf(w, "   Extra damage to: %ss.\n", idlistMobRaceName(affect.Modifier)); err != nil {
					return 0, err
				}
			} else if _, err := fmt.Fprintf(w, "   Affects: %s By %d\n", idlistApplyName(affect.Location), affect.Modifier); err != nil {
				return 0, err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return 0, err
		}
	}
	return len(objects), nil
}

func plural(value int) string {
	if value == 1 {
		return ""
	}
	return "s"
}
