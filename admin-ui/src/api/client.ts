import { http, read, prometheus } from './http';
import type { components } from './schema';
export { ApiError } from './http';

export type Zone = components['schemas']['ZoneResponse'];
export type ServerInfo = components['schemas']['ServerInfoResponse'];
export interface Health { status: 'ok' | 'unknown' }
export type Mob = components['schemas']['MobResponse'];
export type Obj = components['schemas']['ObjResponse'];
export type Room = components['schemas']['RoomResponse'];
export type AgentStatus = components['schemas']['AgentStatus'];
export type Finding = components['schemas']['Finding'];
export type TriageSummary = components['schemas']['TriageSummary'];
export type PlayerInfo = components['schemas']['PlayerResponse'];
export type PlayerItem = components['schemas']['PlayerItemResponse'];
export type PlayerDetail = components['schemas']['PlayerDetailResponse'];
export type ServerMetrics = components['schemas']['MetricsResponse'];
export type Shop = components['schemas']['ShopResponse'];
export type WorldSearchHit = components['schemas']['WorldSearchHit'];
export type WorldSearchResponse = components['schemas']['WorldSearchResponse'];
export type PlayerItemChange = components['schemas']['PlayerItemChange'];
export type SavedPlayerItem = components['schemas']['SavedPlayerItem'];
export type SavedPlayer = components['schemas']['SavedPlayerResponse'];
export type SavedPlayerFlag = components['schemas']['PlayerFlagField'];
export type SavedPlayerDeletion = components['schemas']['DeleteSavedPlayerResponse'];
export type NotificationConfig = components['schemas']['Config'];
export type NotificationStatus = components['schemas']['Status'];
export type NotificationView = components['schemas']['View'];
export type NotificationDestination = components['schemas']['Destination'];
// One identifier per outbound channel: the two event destinations, the public
// gossip mirror, and the hosted heartbeat. It is the path parameter of the
// credential and test operations, so it stays a single name everywhere.
export type NotificationChannel = 'pushover' | 'discord' | 'gossip' | 'heartbeat';
export interface NotificationCredentials { token?: string; user?: string; url?: string }

export const api = {
  health: async (): Promise<Health> => ({ status: (await read(http.GET('/health', { parseAs: 'text' }))).trim() === 'OK' ? 'ok' : 'unknown' }),
  zones: () => read(http.GET('/admin/zones')),
  zone: (id: number) => read(http.GET('/admin/zones/{number}', { params: { path: { number: String(id) } } })),
  server: () => read(http.GET('/admin/server')),
  mobs: () => read(http.GET('/admin/mobs')),
  mob: (vnum: number) => read(http.GET('/admin/mobs/{vnum}', { params: { path: { vnum: String(vnum) } } })),
  objects: () => read(http.GET('/admin/objects')),
  object: (vnum: number) => read(http.GET('/admin/objects/{vnum}', { params: { path: { vnum: String(vnum) } } })),
  room: (vnum: number) => read(http.GET('/admin/rooms/{vnum}', { params: { path: { vnum: String(vnum) } } })),

  searchWorld: (query: string) => read(http.GET('/admin/search', { params: { query: { q: query } } })),




  logs: (lines?: number) => read(http.GET('/admin/logs', { params: { query: { lines: String(lines || 50) } } })),
  players: () => read(http.GET('/admin/players')),
  agents: () => read(http.GET('/admin/agents')),
  updateAgentStatus: (data: { agent_id: string; status: string }) =>
    read(http.POST('/admin/agents/status', { body: data })),
  findings: (params?: { status?: string; severity?: string; source?: string }) => {
    return read(http.GET('/admin/findings', { params: { query: params } }));
  },
  createFinding: (data: Pick<Finding, 'source' | 'severity' | 'title' | 'file' | 'line' | 'description'>) =>
    read(http.POST('/admin/findings', { body: data })),
  updateFinding: (id: number, data: { status: string }) =>
    read(http.PUT('/admin/findings/{id}', { params: { path: { id: String(id) } }, body: data })),
  triageSummaries: () => read(http.GET('/admin/triage/summaries')),
  createTriageSummary: (data: Omit<TriageSummary, 'id' | 'created_at'>) =>
    read(http.POST('/admin/triage/summaries', { body: data })),

  // Shops (Phase 4)
  shops: () => read(http.GET('/admin/shops')),
  shop: (keeperVnum: number) => read(http.GET('/admin/shops/{keeper}', { params: { path: { keeper: String(keeperVnum) } } })),
  // Zone reset
  resetZone: (zoneNumber: number) =>
    read(http.POST('/admin/zones/{number}/reset', { params: { path: { number: String(zoneNumber) } } })),

  // Phase 5 — Operations
  playerDetail: (name: string) => read(http.GET('/admin/players/{name}', { params: { path: { name: name } } })),
  savePlayer: (name: string) => read(http.POST('/admin/players/{name}/save', { params: { path: { name: name } } })),
  kickPlayer: (name: string) => read(http.POST('/admin/players/{name}/kick', { params: { path: { name: name } } })),
  metrics: () => read(http.GET('/admin/metrics')),
  prometheus,
  resetAllZones: () => read(http.POST('/admin/reset-all-zones')),
};


export const playerEditor = {
  list: (q: string) => read(http.GET('/admin/characters', { params: { query: { q } } })),
  get: (name: string) => read(http.GET('/admin/characters/{name}', { params: { path: { name: name } } })),
  stats: (name: string, revision: string, values: Record<string, number>, item_changes: PlayerItemChange[] = []) =>
    read(http.PATCH('/admin/characters/{name}', { params: { path: { name: name } }, body: { revision, values, item_changes } })),
  flags: (name: string, revision: string, flags: Record<string, boolean>) =>
    read(http.PATCH('/admin/characters/{name}/flags', { params: { path: { name: name } }, body: { revision, flags } })),
  rename: (name: string, revision: string, newName: string) =>
    read(http.POST('/admin/characters/{name}/rename', { params: { path: { name: name } }, body: { revision, new_name: newName } })),
  delete: (name: string, revision: string, confirmName = '') =>
    read(http.POST('/admin/characters/{name}/delete', { params: { path: { name: name } }, body: { revision, confirm_name: confirmName } })),
  password: (name: string, revision: string, password: string) =>
    read(http.POST('/admin/characters/{name}/password', { params: { path: { name: name } }, body: { revision, password } })),
  items: (name: string, revision: string, data: { action: 'grant' | 'move' | 'remove'; source: 'inventory' | 'equipment'; index: number; vnum: number; locate: number }) =>
    read(http.POST('/admin/characters/{name}/items', { params: { path: { name: name } }, body: { revision, ...data } })),
};


export const notifications = {
  get: () => read(http.GET('/admin/notifications')),
  save: (config: NotificationConfig) => read(http.PUT('/admin/notifications', { body: config })),
  credentials: (channel: NotificationChannel, revision: number, values: NotificationCredentials) => read(http.PUT('/admin/notifications/{provider}/credentials', { params: { path: { provider: channel } }, body: { revision, ...values } })),
  removeCredentials: (channel: NotificationChannel, revision: number) => read(http.DELETE('/admin/notifications/{provider}/credentials', { params: { path: { provider: channel } }, body: { revision } })),
  test: (channel: NotificationChannel) => read(http.POST('/admin/notifications/{provider}/test', { params: { path: { provider: channel } } })),
};

// Settings and updates (admin-configuration design): typed end to end. Secret
// fields travel only in PUT bodies — nil keeps the sealed value, '' clears.
export const settingsApi = {
  get: () => read(http.GET('/admin/settings')),
  saveSessions: (body: Record<string, unknown>) =>
    read(http.PUT('/admin/settings/sessions', { body: body as never })),
  saveAccess: (body: Record<string, unknown>) =>
    read(http.PUT('/admin/settings/access', { body: body as never })),
  saveGrapevine: (body: Record<string, unknown>) =>
    read(http.PUT('/admin/settings/grapevine', { body: body as never })),
  saveStaffPush: (body: Record<string, unknown>) =>
    read(http.PUT('/admin/settings/staff_push', { body: body as never })),
  saveContact: (body: Record<string, unknown>) =>
    read(http.PUT('/admin/settings/contact', { body: body as never })),
  exportUrl: '/admin/settings/export',
  restoreValidate: (bundle: unknown) =>
    read(http.POST('/admin/settings/restore/validate', { body: bundle as never })),
  restoreApply: (revision: number, bundle: unknown) =>
    read(http.POST('/admin/settings/restore/apply', { body: { revision, bundle } as never })),
  audit: () => read(http.GET('/admin/settings/audit')),
  restart: () => read(http.POST('/admin/restart')),
};

export const updatesApi = {
  status: () => read(http.GET('/admin/updates/status')),
  check: () => read(http.POST('/admin/updates/check')),
  apply: () => read(http.POST('/admin/updates/apply')),
};

// The Network card (admin-network-settings design): drafts go to the
// launcher through the game; applying runs the restart transaction whose
// receipt the console polls across the downtime.
export const networkApi = {
  get: () => read(http.GET('/admin/network')),
  save: (body: Record<string, unknown>) =>
    read(http.PUT('/admin/network', { body: body as never })),
  apply: () => read(http.POST('/admin/network/apply')),
  status: () => read(http.GET('/admin/network/status')),
};
