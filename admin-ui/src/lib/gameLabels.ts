// Display names for the game's enums, sourced from src/structs.h.
//
// Every one of these was hand-rolled in the page that used it, and four of the
// five disagreed with the header they mirror:
//
//   item types  started at 0 with Light; C starts ITEM_LIGHT at 1, so every
//               object rendered one type low — a scroll showed as Wand, a boat
//               as Campfire, a fountain as Corpse
//   positions   were roughly reversed, so POS_STANDING (8) read as "Dead" and
//               a standing dragon looked like a corpse; "Hanging" and "Prone"
//               are not C positions at all
//   sex         was shifted: C is NEUTRAL 0, MALE 1, FEMALE 2, so every male
//               mob displayed as female
//   race        was never labelled, only the raw number
//   sectors     were right as far as they went but stopped at 9, leaving the
//               six elemental and desert sectors as "Sector 10".."Sector 15"
//
// One home, and each table names the header lines it came from. When a value
// has no sourced name, these fall back to the raw number rather than invent
// one: an unlabelled value is honest, a wrong label is not.

/** src/structs.h:420-442, extended 24-37; mirrored in pkg/game/item_helpers.go. 34 is unused. */
export const ITEM_TYPE_LABELS: Record<number, string> = {
  0: 'Undefined', 1: 'Light', 2: 'Scroll', 3: 'Wand', 4: 'Staff',
  5: 'Weapon', 6: 'Fire Weapon', 7: 'Missile', 8: 'Treasure', 9: 'Armor',
  10: 'Potion', 11: 'Worn', 12: 'Other', 13: 'Trash', 14: 'Trap',
  15: 'Container', 16: 'Note', 17: 'Drink Container', 18: 'Key', 19: 'Food',
  20: 'Money', 21: 'Pen', 22: 'Boat', 23: 'Fountain', 24: 'Vehicle',
  25: 'Onion', 26: 'Armor Piece', 27: 'Tattoo', 28: 'Raw Material',
  29: 'Weapon Part', 30: 'Tool', 31: 'Gem', 32: 'Jewelry', 33: 'Furniture',
  35: 'Bag', 36: 'Backpack', 37: 'Corpse',
};

/** src/structs.h:209-217. Position 0 is dead, 8 is standing — not the reverse. */
export const POSITION_LABELS: Record<number, string> = {
  0: 'Dead', 1: 'Mortally Wounded', 2: 'Incapacitated', 3: 'Stunned',
  4: 'Sleeping', 5: 'Resting', 6: 'Sitting', 7: 'Fighting', 8: 'Standing',
};

/** src/structs.h:203-205. */
export const SEX_LABELS: Record<number, string> = {
  0: 'Neutral', 1: 'Male', 2: 'Female',
};

/** src/structs.h:131-161. NPC races; distinct from the seven playable races. */
export const RACE_LABELS: Record<number, string> = {
  0: 'Human', 1: 'Elf', 2: 'Dwarf', 3: 'Kender', 4: 'Centaur',
  5: 'Rakshasa', 6: 'Troll', 7: 'Lycanthrope', 8: 'Vampire', 9: 'Undead',
  10: 'Dragon', 11: 'Demon', 12: 'Horse', 13: 'Reptile', 14: 'Arachnid',
  15: 'Rodent', 16: 'Other', 17: 'Veggie', 18: 'Giant', 19: 'Demigod',
  20: 'Ogre', 21: 'Insect', 22: 'Mammal', 23: 'Fish', 24: 'Avian',
  25: 'Magical', 26: 'Amphibian', 27: 'Humanoid', 28: 'Faery',
  29: 'Ssaur', 30: 'Minotaur',
};

/** src/structs.h:93-108. */
export const SECTOR_LABELS: Record<number, string> = {
  0: 'Inside', 1: 'City', 2: 'Field', 3: 'Forest', 4: 'Hills',
  5: 'Mountain', 6: 'Water (Swim)', 7: 'Water (No Swim)', 8: 'Underwater',
  9: 'Flying', 10: 'Desert', 11: 'Fire', 12: 'Earth', 13: 'Wind',
  14: 'Water', 15: 'Swamp',
};

function label(table: Record<number, string>, value: number, noun: string): string {
  return table[value] ?? `${noun} ${value}`;
}

export const itemTypeLabel = (v: number) => label(ITEM_TYPE_LABELS, v, 'Type');
export const positionLabel = (v: number) => label(POSITION_LABELS, v, 'Position');
export const sexLabel = (v: number) => label(SEX_LABELS, v, 'Sex');
export const raceLabel = (v: number) => label(RACE_LABELS, v, 'Race');
export const sectorLabel = (v: number) => label(SECTOR_LABELS, v, 'Sector');
