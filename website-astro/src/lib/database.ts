import database from '../../../website/static/data/database.json';
import mapIndex from '../../../website/static/map/map-index.json';

export interface Spawn {
  room: number;
  name: string;
  zone: number;
}

export interface Drop {
  obj_vnum: number;
  name: string;
  slot: string;
}

export interface ShopItem {
  vnum: number;
  name: string;
}

export interface Mob {
  v: number;
  k: string;
  s: string;
  l: string;
  d: string;
  lvl: number;
  thac0?: number;
  rc: number;
  rcn?: string;
  alg: number;
  sex: number;
  pos?: string;
  dpos?: string;
  hp: string;
  dmg: string;
  bha?: number;
  atk?: string;
  gld: number;
  exp: number;
  ac: number;
  stat: number[];
  act?: string[];
  aff?: string[];
  noise: string;
  scr?: string;
  spw: Spawn[];
  drp: Drop[];
  shop: null | {
    shop_vnum: number;
    sell_mult: number;
    buy_mult: number;
    open_hours: string;
    items_sold: ShopItem[];
  };
}

export interface Affect {
  location: string;
  modifier: number;
}

export interface ExtraDescription {
  keywords: string;
  desc: string;
}

export interface Item {
  v: number;
  k: string;
  s: string;
  l: string;
  ad?: string;
  type: string;
  wear: string[];
  extra: string[];
  val: number[];
  wt: number;
  cst: number;
  load: number;
  aff: Affect[];
  edesc: ExtraDescription[];
  scr?: string;
  mobs: Array<{ mob_vnum: number; name: string; slot: string }>;
  rms: Array<{ room?: number; zone?: number; name: string; container_vnum?: number }>;
  shp: Array<{ keeper_vnum: number; keeper_name: string; price: number }>;
}

export const mobs = Object.values(database.mobs) as Mob[];
export const items = Object.values(database.items) as Item[];
export const mobByVnum = database.mobs as Record<string, Mob>;
export const itemByVnum = database.items as Record<string, Item>;
export const zoneNames = Object.fromEntries(mapIndex.zones.map((zone) => [zone.id, zone.name])) as Record<number, string>;

export const sexName = (value: number) => ({ 1: 'Male', 2: 'Female' })[value] ?? 'Neutral';
export const alignmentName = (value: number) => value >= 300 ? 'Good' : value <= -300 ? 'Evil' : 'Neutral';
export const signed = (value: number) => value > 0 ? `+${value}` : String(value);

export const descriptionFor = (name: string, description: string, fallback: string) => {
  const text = description.trim() || fallback.trim();
  return text ? `${name}: ${text}`.slice(0, 155) : `${name}, a record from the Dark Pawns world files.`;
};

export const MOB_RACES = [
  'Human', 'Elf', 'Dwarf', 'Kender', 'Centaur', 'Rakshasa', 'Troll',
  'Lycanthrope', 'Vampire', 'Undead', 'Dragon', 'Demon', 'Horse', 'Reptile',
  'Arachnid', 'Rodent', 'Other', 'Vegetable', 'Giant', 'Demi-god', 'Ogre',
  'Insect', 'Mammal', 'Fish', 'Avian', 'Magical Construct', 'Amphibian',
  'Humanoid', 'Faery', 'Ssaur', 'Minotaur',
];

export const AFFECT_BIT_NAMES = [
  'BLIND', 'INVIS', 'DET-ALIGN', 'DET-INVIS', 'DET-MAGIC',
  'SENSE-LIFE', 'WATERWALK', 'SANCT', 'GROUP', 'CURSE', 'INFRA',
  'POISON', 'PROT-EVIL', 'PROT-GOOD', 'SLEEP', '!TRACK',
  'FLESH-ALTER', 'DODGE', 'SNEAK', 'HIDE', 'BERSERK', 'CHARM',
  'FOLLOW', 'WIMPY', 'KUJI-KIRI', 'CUTTHROAT', 'FLY', 'WEREWOLF',
  'VAMPIRE', 'MOUNTED', 'INVULN', 'FLAMING', 'NOTHING', 'HASTE',
  'SLOW', 'DREAM', 'WATERBREATHE', 'METALSKIN', 'ROBBED',
];

export const ATTACK_TYPES = [
  'hit', 'sting', 'whip', 'slash', 'bite', 'bludgeon', 'crush',
  'pound', 'claw', 'maul', 'thrash', 'pierce', 'blast', 'punch', 'stab',
];

export const DRINK_TYPES = [
  'water', 'beer', 'wine', 'ale', 'dark ale', 'whiskey', 'lemonade',
  'firebreather', 'local speciality', 'slime mold juice', 'milk',
  'tea', 'coffee', 'blood', 'salt water', 'clear water',
];

export const SPELL_NAMES: Record<number, string> = {
  1: 'holy ward', 2: 'shift reality', 3: 'bless', 4: 'blindness', 5: 'burning hands',
  6: 'call lightning', 7: 'charm person', 8: 'chill touch', 9: 'clone', 10: 'color spray',
  11: 'control weather', 12: 'create food', 13: 'create water', 14: 'cure blind', 15: 'cure critic',
  16: 'cure light', 17: 'curse', 18: 'detect alignment', 19: 'detect invisibility', 20: 'detect magic',
  21: 'detect poison', 22: 'dispel evil', 23: 'earthquake', 24: 'enchant weapon', 25: 'energy drain',
  26: 'fireball', 27: 'harm', 28: 'heal', 29: 'invisibility', 30: 'lightning bolt',
  31: 'locate object', 32: 'flame arrow', 33: 'poison', 34: 'protection from evil', 35: 'remove curse',
  36: 'sanctuary', 37: 'shocking grasp', 38: 'sleep', 39: 'strength', 40: 'summon',
  41: 'meteor swarm', 42: 'word of recall', 43: 'remove poison', 44: 'sense life', 45: 'animate dead',
  46: 'dispel good', 47: 'holy shield', 48: 'group heal', 49: 'group recall', 50: 'infravision',
  51: 'waterwalk', 52: 'mass heal', 53: 'fly', 54: 'lycanthropy', 55: 'vampirism',
  56: 'sobriety', 57: 'group invisibility', 58: 'hellfire', 59: 'enchant armor', 60: 'identify',
  61: 'mind poke', 62: 'mind blast', 63: 'chameleon', 64: 'levitate', 65: 'metalskin',
  66: 'globe of invulnerability', 67: 'vitality', 68: 'invigorate', 69: 'lesser perception', 70: 'greater perception',
  71: 'mind attack', 72: 'adrenaline boost', 73: 'psychic shield', 74: 'change density', 75: 'acid blast',
  76: 'dominate', 77: 'cell adjustment', 78: 'zen', 79: 'mirror image', 80: 'mass dominate',
  81: 'divine intervention', 82: 'mind bar', 83: 'soul leech', 84: 'mindsight', 85: 'transparency',
  86: 'know align', 87: 'gate', 88: 'word of intellect', 89: 'lay hands', 90: 'mental lapse',
  91: 'smokescreen', 92: 'ray of disruption', 93: 'disintegration', 94: 'calliope', 95: 'protection from good',
  96: 'flame strike', 97: 'haste', 98: 'slow', 99: 'dream travel', 100: 'psiblast',
  101: 'glyph of summoning', 102: 'waterbreathe', 105: 'conjure elemental',
  202: 'fire breath', 203: 'gas breath', 204: 'frost breath', 205: 'acid breath', 206: 'lightning breath',
};

export const raceName = (rc: number, rcn?: string) => rcn || MOB_RACES[rc] || `Race ${rc}`;

export function formatItemAffect(aff: Affect): string {
  if (aff.location === 'PERM_SPELL') {
    const bitName = AFFECT_BIT_NAMES[aff.modifier] ?? `BIT_${aff.modifier}`;
    return `Permanent ${bitName}`;
  }
  if (aff.location === 'RACE_HATE') {
    const rName = MOB_RACES[aff.modifier] ?? `Race ${aff.modifier}`;
    return `Racial Hatred (${rName})`;
  }
  return `${signed(aff.modifier)} ${aff.location}`;
}

export function describeItemValues(item: Item): string {
  const [v0, v1, v2, v3] = item.val ?? [0, 0, 0, 0];
  switch (item.type) {
    case 'WEAPON': {
      const avg = (((v2 + 1) / 2.0) * v1).toFixed(1);
      const atk = ATTACK_TYPES[v3] ?? `type ${v3}`;
      return `${v1}d${v2} (${avg} avg, ${atk})`;
    }
    case 'ARMOR':
      return `AC apply ${v0}`;
    case 'LIGHT':
      return v2 === -1 ? 'Infinite hours' : `${v2} hours`;
    case 'SCROLL':
    case 'POTION': {
      const spells = [v1, v2, v3]
        .filter((s) => s > 0)
        .map((s) => SPELL_NAMES[s] ?? `spell #${s}`);
      return `Level ${v0}: ${spells.length ? spells.join(', ') : 'none'}`;
    }
    case 'WAND':
    case 'STAFF': {
      const spell = v3 > 0 ? (SPELL_NAMES[v3] ?? `spell #${v3}`) : 'none';
      return `Level ${v0} ${spell} (${v2}/${v1} charges)`;
    }
    case 'CONTAINER':
      return `Capacity ${v0}${v2 > 0 ? `, key #${v2}` : ''}`;
    case 'DRINKCON':
    case 'FOUNTAIN': {
      const liq = DRINK_TYPES[v2] ?? `liquid #${v2}`;
      return `${v1}/${v0} units of ${liq}${v3 ? ' (poisoned)' : ''}`;
    }
    case 'FOOD':
      return `Fills ${v0} hours${v3 ? ' (poisoned)' : ''}`;
    case 'MONEY':
      return `${v0} coins`;
    default:
      return `[${v0}, ${v1}, ${v2}, ${v3}]`;
  }
}
