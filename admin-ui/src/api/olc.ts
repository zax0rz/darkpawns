import { request } from './client';

export interface OlcBounds {
  min: number;
  max: number;
}

export interface OlcOption {
  value: number;
  label: string;
}

export interface OlcSchemaField {
  key: string;
  label: string;
  control: 'text' | 'textarea' | 'number' | 'select' | 'checkbox_group' | string;
  bounds?: OlcBounds;
  options?: OlcOption[];
}

export interface OlcSchema {
  kind: string;
  fields: OlcSchemaField[];
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
  bit?: number;
  index?: number;
  enabled?: boolean;
}

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

function normalizeSchema(raw: unknown): OlcSchema {
  const source = record(raw);
  const fields = value<unknown>(source, 'fields', 'Fields');
  return {
    kind: stringValue(source, 'kind', 'Kind'),
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
                  };
                })
              : [],
          };
        })
      : [],
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
  preview: async (kind: string, vnum: number) => {
    const source = record(await request<unknown>(`/olc/${kind}/${vnum}/preview`));
    return { room: normalizeRoom(value(source, 'room', 'Room')) };
  },
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
