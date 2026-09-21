import { request } from './client';

export interface OlcBounds {
  min: number;
  max: number;
}

export interface OlcOption {
  value: number;
  label: string;
  storage?: string;
}

export interface OlcSchemaField {
  key: string;
  label: string;
  control: 'text' | 'textarea' | 'number' | 'select' | 'checkbox_group' | string;
  bounds?: OlcBounds;
  options?: OlcOption[];
}

export interface OlcValueField {
  label: string;
  control: string;
  min: number;
  max: number;
  visible: boolean;
  options?: OlcOption[];
}

export interface OlcValueMatrixEntry {
  itemType: number;
  itemTypeLabel: string;
  values: OlcValueField[];
}

export interface OlcAppliesDescriptor {
  max: number;
  options: OlcOption[];
  addOperation: string;
  removeOperation: string;
}

export interface OlcExitDescriptor {
  directions: string[];
  doorOptions: OlcOption[];
}

export interface OlcSchemaAction {
  key: string;
  label: string;
  requiredLevel: number;
  requiredLabel: string;
  allowed: boolean;
}

export interface OlcZoneCommandArgument {
  key: string;
  label: string;
  control: string;
  bounds?: OlcBounds;
  options: OlcOption[];
  visible: boolean;
}

export interface OlcZoneCommandDescriptor {
  command: string;
  label: string;
  arguments: OlcZoneCommandArgument[];
}

export interface OlcSchema {
  kind: string;
  fields: OlcSchemaField[];
  bespoke: string[];
  actions: OlcSchemaAction[];
  valueMatrix: OlcValueMatrixEntry[];
  applies?: OlcAppliesDescriptor;
  exits?: OlcExitDescriptor;
  zoneCommands: OlcZoneCommandDescriptor[];
}

export interface OlcExit {
  direction: string;
  toRoom: number;
  doorState: number;
  exitInfo: number;
  key: number;
  keywords: string;
  description: string;
}

export interface OlcExtraDescription {
  keywords: string;
  description: string;
}

export interface OlcRoom {
  vnum: number;
  name: string;
  description: string;
  zone: number;
  flags: string[];
  sector: number;
  exits: Record<string, OlcExit>;
  extraDescs: OlcExtraDescription[];
  scriptName: string;
  scriptFunctions: number;
}

export interface OlcRoomDraft {
  kind: string;
  vnum: number;
  room: OlcRoom;
  dirty: string[];
  leaseExpiresAt: string;
  leaseRemainingSeconds: number;
}

export interface OlcDiceRoll {
  num: number;
  sides: number;
  plus: number;
}

export interface OlcMob {
  vnum: number;
  keywords: string;
  shortDesc: string;
  longDesc: string;
  detailedDesc: string;
  actionFlags: string[];
  affectFlags: string[];
  alignment: number;
  race: number;
  level: number;
  thac0: number;
  ac: number;
  hp: OlcDiceRoll;
  damage: OlcDiceRoll;
  gold: number;
  exp: number;
  position: number;
  defaultPos: number;
  sex: number;
  noise: string;
  bareHandAttack: number;
  scriptName: string;
  luaFunctions: number;
}

export interface OlcObjectAffect {
  location: number;
  modifier: number;
}

export interface OlcObject {
  vnum: number;
  keywords: string;
  shortDesc: string;
  longDesc: string;
  actionDesc: string;
  typeFlag: number;
  extraFlags: number[];
  wearFlags: number[];
  values: number[];
  weight: number;
  cost: number;
  loadPercent: number;
  affects: OlcObjectAffect[];
  extraDescs: OlcExtraDescription[];
  scriptName: string;
  luaFunctions: number;
}

export interface OlcShop {
  vnum: number;
  products: number[];
  buyProfit: number;
  sellProfit: number;
  keeperVnum: number;
  flags: number;
  withWho: number;
  rooms: number[];
  openHour1: number;
  closeHour1: number;
  openHour2: number;
  closeHour2: number;
  messages: string[];
  buyTypes: number[];
  buyWords: string[];
}

export interface OlcZoneCommand {
  position: number;
  command: string;
  ifFlag: number;
  arg1: number;
  arg2: number;
  arg3: number;
}

export interface OlcZone {
  number: number;
  name: string;
  topRoom: number;
  lifespan: number;
  resetMode: number;
  commands: OlcZoneCommand[];
}

export type OlcEntityKind = 'mob' | 'obj' | 'shop' | 'zone';

export interface OlcEntityDraft {
  kind: string;
  vnum: number;
  mob?: OlcMob;
  object?: OlcObject;
  shop?: OlcShop;
  zone?: OlcZone;
  dirty: string[];
  leaseExpiresAt: string;
  leaseRemainingSeconds: number;
}

export interface OlcPreview {
  kind: string;
  vnum: number;
  zoneNumber: number;
  room?: OlcRoom;
  mob?: OlcMob;
  object?: OlcObject;
  shop?: OlcShop;
  zone?: OlcZone;
}

export interface OlcClaimEntry {
  kind: string;
  number: number;
  ownerIdentity: string;
  ownerDisplayName: string;
  ownerFrontend: string;
  claimedAt: string;
  expiresAt: string;
}

export interface OlcDirtyEntry {
  kind: string;
  zone: number;
}

export interface RoomPatchOperation {
  kind: string;
  text?: string;
  keywords?: string;
  direction?: string;
  value?: number;
  float?: number;
  bit?: number;
  index?: number;
  enabled?: boolean;
  location?: number;
  modifier?: number;
  toIndex?: number;
  command?: string;
  ifFlag?: number;
  arg1?: number;
  arg2?: number;
  arg3?: number;
}

export type OlcPatchOperation = RoomPatchOperation;

interface JsonRecord {
  [key: string]: unknown;
}

function record(value: unknown): JsonRecord {
  return typeof value === 'object' && value !== null ? (value as JsonRecord) : {};
}

function value<T>(source: JsonRecord, ...keys: string[]): T | undefined {
  for (const key of keys) {
    if (source[key] !== undefined) return source[key] as T;
  }
  return undefined;
}

function numberValue(source: JsonRecord, ...keys: string[]): number {
  const candidate = value<unknown>(source, ...keys);
  return typeof candidate === 'number' ? candidate : Number(candidate || 0);
}

function stringValue(source: JsonRecord, ...keys: string[]): string {
  const candidate = value<unknown>(source, ...keys);
  return typeof candidate === 'string' ? candidate : '';
}

function stringArray(source: JsonRecord, ...keys: string[]): string[] {
  const candidate = value<unknown>(source, ...keys);
  return Array.isArray(candidate) ? candidate.map(String) : [];
}

function numberArray(source: JsonRecord, ...keys: string[]): number[] {
  const candidate = value<unknown>(source, ...keys);
  return Array.isArray(candidate) ? candidate.map((entry) => Number(entry) || 0) : [];
}

function normalizeDice(raw: unknown): OlcDiceRoll {
  const source = record(raw);
  return {
    num: numberValue(source, 'num', 'Num'),
    sides: numberValue(source, 'sides', 'Sides'),
    plus: numberValue(source, 'plus', 'Plus'),
  };
}

function normalizeMob(raw: unknown): OlcMob {
  const source = record(raw);
  return {
    vnum: numberValue(source, 'vnum', 'VNum'),
    keywords: stringValue(source, 'keywords', 'Keywords'),
    shortDesc: stringValue(source, 'shortDesc', 'ShortDesc', 'short_desc'),
    longDesc: stringValue(source, 'longDesc', 'LongDesc', 'long_desc'),
    detailedDesc: stringValue(source, 'detailedDesc', 'DetailedDesc', 'detailed_desc'),
    actionFlags: stringArray(source, 'actionFlags', 'ActionFlags', 'action_flags'),
    affectFlags: stringArray(source, 'affectFlags', 'AffectFlags', 'affect_flags'),
    alignment: numberValue(source, 'alignment', 'Alignment'),
    race: numberValue(source, 'race', 'Race'),
    level: numberValue(source, 'level', 'Level'),
    thac0: numberValue(source, 'thac0', 'THAC0', 'thaco'),
    ac: numberValue(source, 'ac', 'AC'),
    hp: normalizeDice(value(source, 'hp', 'HP')),
    damage: normalizeDice(value(source, 'damage', 'Damage')),
    gold: numberValue(source, 'gold', 'Gold'),
    exp: numberValue(source, 'exp', 'Exp'),
    position: numberValue(source, 'position', 'Position'),
    defaultPos: numberValue(source, 'defaultPos', 'DefaultPos', 'default_pos'),
    sex: numberValue(source, 'sex', 'Sex'),
    noise: stringValue(source, 'noise', 'Noise'),
    bareHandAttack: numberValue(source, 'bareHandAttack', 'BareHandAttack', 'bare_hand_attack'),
    scriptName: stringValue(source, 'scriptName', 'ScriptName', 'script_name'),
    luaFunctions: numberValue(source, 'luaFunctions', 'LuaFunctions', 'lua_functions'),
  };
}

function normalizeAffects(raw: unknown): OlcObjectAffect[] {
  return Array.isArray(raw)
    ? raw.map((entry) => {
        const source = record(entry);
        return {
          location: numberValue(source, 'location', 'Location'),
          modifier: numberValue(source, 'modifier', 'Modifier'),
        };
      })
    : [];
}

function normalizeExtras(raw: unknown): OlcExtraDescription[] {
  return Array.isArray(raw)
    ? raw.map((entry) => {
        const source = record(entry);
        return {
          keywords: stringValue(source, 'keywords', 'Keywords'),
          description: stringValue(source, 'description', 'Description'),
        };
      })
    : [];
}

function normalizeObject(raw: unknown): OlcObject {
  const source = record(raw);
  return {
    vnum: numberValue(source, 'vnum', 'VNum'),
    keywords: stringValue(source, 'keywords', 'Keywords'),
    shortDesc: stringValue(source, 'shortDesc', 'ShortDesc', 'short_desc'),
    longDesc: stringValue(source, 'longDesc', 'LongDesc', 'long_desc'),
    actionDesc: stringValue(source, 'actionDesc', 'ActionDesc', 'action_desc'),
    typeFlag: numberValue(source, 'typeFlag', 'TypeFlag', 'type_flag'),
    extraFlags: numberArray(source, 'extraFlags', 'ExtraFlags', 'extra_flags'),
    wearFlags: numberArray(source, 'wearFlags', 'WearFlags', 'wear_flags'),
    values: numberArray(source, 'values', 'Values'),
    weight: numberValue(source, 'weight', 'Weight'),
    cost: numberValue(source, 'cost', 'Cost'),
    loadPercent: numberValue(source, 'loadPercent', 'LoadPercent', 'load_percent'),
    affects: normalizeAffects(value(source, 'affects', 'Affects')),
    extraDescs: normalizeExtras(value(source, 'extraDescs', 'ExtraDescs', 'extra_descs')),
    scriptName: stringValue(source, 'scriptName', 'ScriptName', 'script_name'),
    luaFunctions: numberValue(source, 'luaFunctions', 'LuaFunctions', 'lua_functions'),
  };
}

function normalizeShop(raw: unknown): OlcShop {
  const source = record(raw);
  return {
    vnum: numberValue(source, 'vnum', 'VNum'),
    products: numberArray(source, 'products', 'Products'),
    buyProfit: numberValue(source, 'buyProfit', 'BuyProfit', 'buy_profit'),
    sellProfit: numberValue(source, 'sellProfit', 'SellProfit', 'sell_profit'),
    keeperVnum: numberValue(source, 'keeperVNum', 'KeeperVNum', 'keeper_vnum'),
    flags: numberValue(source, 'bitvector', 'Bitvector', 'flags', 'Flags'),
    withWho: numberValue(source, 'withWho', 'WithWho', 'with_who'),
    rooms: numberArray(source, 'rooms', 'Rooms'),
    openHour1: numberValue(source, 'openHour1', 'OpenHour1', 'open_hour_1'),
    closeHour1: numberValue(source, 'closeHour1', 'CloseHour1', 'close_hour_1'),
    openHour2: numberValue(source, 'openHour2', 'OpenHour2', 'open_hour_2'),
    closeHour2: numberValue(source, 'closeHour2', 'CloseHour2', 'close_hour_2'),
    messages: stringArray(source, 'messages', 'Messages'),
    buyTypes: numberArray(source, 'buyTypes', 'BuyTypes'),
    buyWords: stringArray(source, 'buyWords', 'BuyWords'),
  };
}

function normalizeZoneCommand(raw: unknown): OlcZoneCommand {
  const source = record(raw);
  return {
    position: numberValue(source, 'position', 'Position'),
    command: stringValue(source, 'command', 'Command'),
    ifFlag: numberValue(source, 'ifFlag', 'IfFlag', 'if_flag'),
    arg1: numberValue(source, 'arg1', 'Arg1'),
    arg2: numberValue(source, 'arg2', 'Arg2'),
    arg3: numberValue(source, 'arg3', 'Arg3'),
  };
}

function normalizeZone(raw: unknown): OlcZone {
  const source = record(raw);
  const commands = value<unknown>(source, 'commands', 'Commands');
  return {
    number: numberValue(source, 'number', 'Number'),
    name: stringValue(source, 'name', 'Name'),
    topRoom: numberValue(source, 'topRoom', 'TopRoom', 'top_room'),
    lifespan: numberValue(source, 'lifespan', 'Lifespan'),
    resetMode: numberValue(source, 'resetMode', 'ResetMode', 'reset_mode'),
    commands: Array.isArray(commands) ? commands.map(normalizeZoneCommand) : [],
  };
}

function normalizeExit(raw: unknown, direction: string): OlcExit {
  const source = record(raw);
  return {
    direction: stringValue(source, 'direction', 'Direction') || direction,
    toRoom: numberValue(source, 'toRoom', 'ToRoom', 'to_room'),
    doorState: numberValue(source, 'doorState', 'DoorState', 'door_state'),
    exitInfo: numberValue(source, 'exitInfo', 'ExitInfo', 'exit_info'),
    key: numberValue(source, 'key', 'Key'),
    keywords: stringValue(source, 'keywords', 'Keywords'),
    description: stringValue(source, 'description', 'Description'),
  };
}

function normalizeRoom(raw: unknown): OlcRoom {
  const source = record(raw);
  const rawExits = value<unknown>(source, 'exits', 'Exits');
  const exits: Record<string, OlcExit> = {};
  if (Array.isArray(rawExits)) {
    for (const rawExit of rawExits) {
      const exit = normalizeExit(rawExit, '');
      if (exit.direction) exits[exit.direction] = exit;
    }
  } else {
    for (const [direction, rawExit] of Object.entries(record(rawExits))) {
      exits[direction] = normalizeExit(rawExit, direction);
    }
  }

  const rawExtras = value<unknown>(source, 'extraDescs', 'ExtraDescs', 'extra_descs');
  const extraDescs = Array.isArray(rawExtras)
    ? rawExtras.map((rawExtra) => {
        const extra = record(rawExtra);
        return {
          keywords: stringValue(extra, 'keywords', 'Keywords'),
          description: stringValue(extra, 'description', 'Description'),
        };
      })
    : [];

  const rawFlags = value<unknown>(source, 'flags', 'Flags');
  return {
    vnum: numberValue(source, 'vnum', 'VNum'),
    name: stringValue(source, 'name', 'Name'),
    description: stringValue(source, 'description', 'Description'),
    zone: numberValue(source, 'zone', 'Zone'),
    flags: Array.isArray(rawFlags) ? rawFlags.map(String) : [],
    sector: numberValue(source, 'sector', 'Sector'),
    exits,
    extraDescs,
    scriptName: stringValue(source, 'scriptName', 'ScriptName', 'script_name'),
    scriptFunctions: numberValue(
      source,
      'scriptFunctions',
      'ScriptFunctions',
      'script_functions',
    ),
  };
}

function normalizeDraft(raw: unknown): OlcRoomDraft {
  const source = record(raw);
  const dirty = value<unknown>(source, 'dirty', 'Dirty');
  return {
    kind: stringValue(source, 'kind', 'Kind'),
    vnum: numberValue(source, 'vnum', 'VNum'),
    room: normalizeRoom(value(source, 'room', 'Room')),
    dirty: Array.isArray(dirty) ? dirty.map(String) : [],
    leaseExpiresAt: stringValue(source, 'leaseExpiresAt', 'LeaseExpiresAt', 'lease_expires_at'),
    leaseRemainingSeconds: numberValue(
      source,
      'leaseRemainingSeconds',
      'LeaseRemainingSeconds',
      'lease_remaining_seconds',
    ),
  };
}

function normalizeEntityDraft(raw: unknown): OlcEntityDraft {
  const source = record(raw);
  const dirty = value<unknown>(source, 'dirty', 'Dirty');
  const rawMob = value(source, 'mob', 'Mob');
  const rawObject = value(source, 'object', 'Object');
  const rawShop = value(source, 'shop', 'Shop');
  const rawZone = value(source, 'zone', 'Zone');
  return {
    kind: stringValue(source, 'kind', 'Kind'),
    vnum: numberValue(source, 'vnum', 'VNum'),
    ...(rawMob !== undefined ? { mob: normalizeMob(rawMob) } : {}),
    ...(rawObject !== undefined ? { object: normalizeObject(rawObject) } : {}),
    ...(rawShop !== undefined ? { shop: normalizeShop(rawShop) } : {}),
    ...(rawZone !== undefined ? { zone: normalizeZone(rawZone) } : {}),
    dirty: Array.isArray(dirty) ? dirty.map(String) : [],
    leaseExpiresAt: stringValue(source, 'leaseExpiresAt', 'LeaseExpiresAt', 'lease_expires_at'),
    leaseRemainingSeconds: numberValue(
      source,
      'leaseRemainingSeconds',
      'LeaseRemainingSeconds',
      'lease_remaining_seconds',
    ),
  };
}

function normalizeSchema(raw: unknown): OlcSchema {
  const source = record(raw);
  const fields = value<unknown>(source, 'fields', 'Fields');
  const bespoke = value<unknown>(source, 'bespoke', 'Bespoke');
  const actions = value<unknown>(source, 'actions', 'Actions');
  const zoneCommands = value<unknown>(source, 'zone_commands', 'ZoneCommands');
  return {
    kind: stringValue(source, 'kind', 'Kind'),
    bespoke: Array.isArray(bespoke) ? bespoke.map(String) : [],
    actions: Array.isArray(actions)
      ? actions.map((rawAction) => {
          const action = record(rawAction);
          return {
            key: stringValue(action, 'key', 'Key'),
            label: stringValue(action, 'label', 'Label'),
            requiredLevel: numberValue(action, 'required_level', 'RequiredLevel'),
            requiredLabel: stringValue(action, 'required_label', 'RequiredLabel'),
            allowed: Boolean(value<boolean>(action, 'allowed', 'Allowed')),
          };
        })
      : [],
    fields: Array.isArray(fields)
      ? fields.map((rawField) => {
          const field = record(rawField);
          const rawBounds = value<unknown>(field, 'bounds', 'Bounds');
          const bounds = rawBounds ? record(rawBounds) : undefined;
          const rawOptions = value<unknown>(field, 'options', 'Options');
          return {
            key: stringValue(field, 'key', 'Key'),
            label: stringValue(field, 'label', 'Label'),
            control: stringValue(field, 'control', 'Control'),
            ...(bounds
              ? { bounds: { min: numberValue(bounds, 'min', 'Min'), max: numberValue(bounds, 'max', 'Max') } }
              : {}),
            options: Array.isArray(rawOptions)
              ? rawOptions.map((rawOption) => {
                  const option = record(rawOption);
                  return {
                    value: numberValue(option, 'value', 'Value'),
                    label: stringValue(option, 'label', 'Label'),
                    ...(stringValue(option, 'storage', 'Storage') ? { storage: stringValue(option, 'storage', 'Storage') } : {}),
                  };
                })
              : [],
          };
        })
      : [],
    zoneCommands: Array.isArray(zoneCommands)
      ? zoneCommands.map((rawDescriptor) => {
          const descriptor = record(rawDescriptor);
          const args = value<unknown>(descriptor, 'arguments', 'Arguments');
          return {
            command: stringValue(descriptor, 'command', 'Command'),
            label: stringValue(descriptor, 'label', 'Label'),
            arguments: Array.isArray(args)
              ? args.map((rawArgument) => {
                  const argument = record(rawArgument);
                  const rawBounds = value<unknown>(argument, 'bounds', 'Bounds');
                  const bounds = rawBounds ? record(rawBounds) : undefined;
                  const rawOptions = value<unknown>(argument, 'options', 'Options');
                  return {
                    key: stringValue(argument, 'key', 'Key'),
                    label: stringValue(argument, 'label', 'Label'),
                    control: stringValue(argument, 'control', 'Control'),
                    visible: Boolean(value<boolean>(argument, 'visible', 'Visible')),
                    ...(bounds
                      ? { bounds: { min: numberValue(bounds, 'min', 'Min'), max: numberValue(bounds, 'max', 'Max') } }
                      : {}),
                    options: Array.isArray(rawOptions)
                      ? rawOptions.map((rawOption) => {
                          const option = record(rawOption);
                          return {
                            value: numberValue(option, 'value', 'Value'),
                            label: stringValue(option, 'label', 'Label'),
                          };
                        })
                      : [],
                  };
                })
              : [],
          };
        })
      : [],
    valueMatrix: Array.isArray(value(source, 'value_matrix', 'ValueMatrix'))
      ? (value(source, 'value_matrix', 'ValueMatrix') as unknown[]).map((rawEntry) => {
          const entry = record(rawEntry);
          const rawValues = value<unknown>(entry, 'values', 'Values');
          return {
            itemType: numberValue(entry, 'item_type', 'ItemType'),
            itemTypeLabel: stringValue(entry, 'item_type_label', 'ItemTypeLabel'),
            values: Array.isArray(rawValues)
              ? rawValues.map((rawField) => {
                  const field = record(rawField);
                  const rawOptions = value<unknown>(field, 'options', 'Options');
                  return {
                    label: stringValue(field, 'label', 'Label'),
                    control: stringValue(field, 'control', 'Control'),
                    min: numberValue(field, 'min', 'Min'),
                    max: numberValue(field, 'max', 'Max'),
                    visible: Boolean(value<boolean>(field, 'visible', 'Visible')),
                    options: Array.isArray(rawOptions)
                      ? rawOptions.map((rawOption) => {
                          const option = record(rawOption);
                          return {
                            value: numberValue(option, 'value', 'Value'),
                            label: stringValue(option, 'label', 'Label'),
                          };
                        })
                      : [],
                  };
                })
              : [],
          };
        })
      : [],
    ...(value(source, 'applies', 'Applies')
      ? (() => {
          const applies = record(value(source, 'applies', 'Applies'));
          const rawOptions = value<unknown>(applies, 'options', 'Options');
          return {
            applies: {
              max: numberValue(applies, 'max', 'Max'),
              options: Array.isArray(rawOptions)
                ? rawOptions.map((rawOption) => {
                    const option = record(rawOption);
                    return {
                      value: numberValue(option, 'value', 'Value'),
                      label: stringValue(option, 'label', 'Label'),
                    };
                  })
                : [],
              addOperation: stringValue(applies, 'add_operation', 'AddOperation'),
              removeOperation: stringValue(applies, 'remove_operation', 'RemoveOperation'),
            },
          };
        })()
      : {}),
    ...(value(source, 'exits', 'Exits')
      ? (() => {
          const exits = record(value(source, 'exits', 'Exits'));
          const rawDirections = value<unknown>(exits, 'directions', 'Directions');
          const rawOptions = value<unknown>(exits, 'door_options', 'DoorOptions');
          return {
            exits: {
              directions: Array.isArray(rawDirections) ? rawDirections.map(String) : [],
              doorOptions: Array.isArray(rawOptions)
                ? rawOptions.map((rawOption) => {
                    const option = record(rawOption);
                    return {
                      value: numberValue(option, 'value', 'Value'),
                      label: stringValue(option, 'label', 'Label'),
                    };
                  })
                : [],
            },
          };
        })()
      : {}),
  };
}

function normalizePreview(raw: unknown): OlcPreview {
  const source = record(raw);
  const rawRoom = value(source, 'room', 'Room');
  const rawMob = value(source, 'mob', 'Mob');
  const rawObject = value(source, 'object', 'Object');
  const rawShop = value(source, 'shop', 'Shop');
  const rawZone = value(source, 'zone', 'Zone');
  return {
    kind: stringValue(source, 'kind', 'Kind'),
    vnum: numberValue(source, 'vnum', 'VNum'),
    zoneNumber: numberValue(source, 'zone_number', 'ZoneNumber'),
    ...(rawRoom !== undefined ? { room: normalizeRoom(rawRoom) } : {}),
    ...(rawMob !== undefined ? { mob: normalizeMob(rawMob) } : {}),
    ...(rawObject !== undefined ? { object: normalizeObject(rawObject) } : {}),
    ...(rawShop !== undefined ? { shop: normalizeShop(rawShop) } : {}),
    ...(rawZone !== undefined ? { zone: normalizeZone(rawZone) } : {}),
  };
}

function normalizeClaim(raw: unknown): OlcClaimEntry {
  const source = record(raw);
  return {
    kind: stringValue(source, 'kind', 'Kind'),
    number: numberValue(source, 'number', 'Number'),
    ownerIdentity: stringValue(source, 'ownerIdentity', 'OwnerIdentity', 'owner_identity'),
    ownerDisplayName: stringValue(source, 'ownerDisplayName', 'OwnerDisplayName', 'owner_display_name'),
    ownerFrontend: stringValue(source, 'ownerFrontend', 'OwnerFrontend', 'owner_frontend'),
    claimedAt: stringValue(source, 'claimedAt', 'ClaimedAt', 'claimed_at'),
    expiresAt: stringValue(source, 'expiresAt', 'ExpiresAt', 'expires_at'),
  };
}

export const olcApi = {
  schema: async (kind: string) => normalizeSchema(await request<unknown>(`/olc/schema/${kind}`)),
  preview: async (kind: string, vnum: number) => normalizePreview(await request<unknown>(`/olc/${kind}/${vnum}/preview`)),
  openRoomDraft: async (vnum: number) => normalizeDraft(await request<unknown>(`/olc/room/${vnum}`, { method: 'POST' })),
  getRoomDraft: async (vnum: number) => normalizeDraft(await request<unknown>(`/olc/room/${vnum}/draft`)),
  patchRoomDraft: async (vnum: number, operations: RoomPatchOperation[]) =>
    normalizeDraft(
      await request<unknown>(`/olc/room/${vnum}/draft`, {
        method: 'PATCH',
        body: JSON.stringify(operations),
      }),
    ),
  // P7 transports these operation kinds through the draft route, but the
  // server applies them directly to the live prototype and does not add them
  // to draft dirty state. Keep these methods separate so callers cannot treat
  // script changes as reversible draft edits.
  setRoomScriptName: async (vnum: number, text: string) =>
    olcApi.patchRoomDraft(vnum, [{ kind: 'set_script_name', text }]),
  setRoomScriptFlag: async (vnum: number, bit: number, enabled: boolean) =>
    olcApi.patchRoomDraft(vnum, [{ kind: 'set_script_flag', bit, enabled }]),
  commitRoomDraft: async (vnum: number) =>
    normalizeDraft(await request<unknown>(`/olc/room/${vnum}/draft/commit`, { method: 'POST' })),
  discardRoomDraft: (vnum: number) =>
    request<void>(`/olc/room/${vnum}/draft`, { method: 'DELETE' }),
  openEntityDraft: async (kind: OlcEntityKind, vnum: number) =>
    normalizeEntityDraft(await request<unknown>(`/olc/${kind}/${vnum}`, { method: 'POST' })),
  getEntityDraft: async (kind: OlcEntityKind, vnum: number) =>
    normalizeEntityDraft(await request<unknown>(`/olc/${kind}/${vnum}/draft`)),
  patchEntityDraft: async (kind: OlcEntityKind, vnum: number, operations: OlcPatchOperation[]) =>
    normalizeEntityDraft(
      await request<unknown>(`/olc/${kind}/${vnum}/draft`, {
        method: 'PATCH',
        body: JSON.stringify(operations),
      }),
    ),
  commitEntityDraft: async (kind: OlcEntityKind, vnum: number) =>
    normalizeEntityDraft(await request<unknown>(`/olc/${kind}/${vnum}/draft/commit`, { method: 'POST' })),
  discardEntityDraft: (kind: OlcEntityKind, vnum: number) =>
    request<void>(`/olc/${kind}/${vnum}/draft`, { method: 'DELETE' }),
  setMobScriptName: async (vnum: number, text: string) =>
    olcApi.patchEntityDraft('mob', vnum, [{ kind: 'set_script_name', text }]),
  setMobScriptFlag: async (vnum: number, bit: number, enabled: boolean) =>
    olcApi.patchEntityDraft('mob', vnum, [{ kind: 'set_script_flag', bit, enabled }]),
  setObjectScriptName: async (vnum: number, text: string) =>
    olcApi.patchEntityDraft('obj', vnum, [{ kind: 'set_script_name', text }]),
  setObjectScriptFlag: async (vnum: number, bit: number, enabled: boolean) =>
    olcApi.patchEntityDraft('obj', vnum, [{ kind: 'set_script_flag', bit, enabled }]),
  createZone: async (zone: number) =>
    (await request<{ zone: number; name: string; top_room: number; lifespan: number; reset_mode: number; created: boolean }>(`/olc/zones/${zone}`, { method: 'POST' })),
  saveZone: (zone: number) =>
    request<{ zone: number; saved: boolean }>(`/olc/zones/${zone}/save`, { method: 'POST' }),
  held: async () => {
    const raw = await request<unknown[]>(`/olc/held`);
    return Array.isArray(raw) ? raw.map(normalizeClaim) : [];
  },
  pending: async () => {
    const raw = await request<unknown[]>(`/olc/pending`);
    return Array.isArray(raw)
      ? raw.map((entry) => {
          const source = record(entry);
          return {
            kind: stringValue(source, 'kind', 'Kind'),
            zone: numberValue(source, 'zone', 'Zone'),
          };
        })
      : [];
  },
};
