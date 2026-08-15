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
  rc: number;
  alg: number;
  sex: number;
  hp: string;
  dmg: string;
  gld: number;
  exp: number;
  ac: number;
  stat: number[];
  noise: string;
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
  type: string;
  wear: string[];
  extra: string[];
  val: number[];
  wt: number;
  cst: number;
  load: number;
  aff: Affect[];
  edesc: ExtraDescription[];
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
