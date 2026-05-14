const API_BASE = '/admin';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('admin_token');
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  });
  if (res.status === 401) {
    localStorage.removeItem('admin_token');
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API error ${res.status}: ${body}`);
  }
  return res.json();
}

export interface Zone {
  number: number;
  name: string;
  top_room: number;
  lifespan: number;
  reset_mode: number;
}

export interface ServerInfo {
  uptime: string;
  room_count: number;
  player_count: number;
  zone_count: number;
}

export interface Health {
  status: string;
}

export interface Mob {
  vnum: number;
  keywords: string;
  short_desc: string;
  long_desc: string;
  level: number;
  alignment: number;
  ac: number;
  hp: string;
  gold: number;
  exp: number;
  position: number;
  default_pos: number;
  sex: number;
  race: number;
  action_flags: string[];
  affect_flags: string[];
  script_name: string;
  str: number;
  int: number;
  wis: number;
  dex: number;
  con: number;
  cha: number;
}

export interface Obj {
  vnum: number;
  keywords: string;
  short_desc: string;
  long_desc: string;
  type_flag: number;
  weight: number;
  cost: number;
  extra_flags: number[];
  wear_flags: number[];
  values: number[];
  script_name: string;
}

export interface Room {
  vnum: number;
  name: string;
  description: string;
  zone: number;
  sector: number;
  flags: string[];
}

export interface AgentStatus {
  agent_id: string;
  name: string;
  status: string;
  last_run: string;
  model: string;
  description: string;
}

export interface Finding {
  id: number;
  source: string;
  severity: string;
  status: string;
  title: string;
  file: string;
  line: number;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface TriageSummary {
  id: number;
  date: string;
  confirmed: number;
  rejected: number;
  pending: number;
  summary: string;
  created_at: string;
}

export interface PlayerInfo {
  name: string;
  level: number;
  room: number;
}

export interface PlayerItem {
  vnum: number;
  name: string;
  short_desc: string;
  type: number;
  wear_location: string;
}

export interface PlayerDetail {
  name: string;
  level: number;
  class: number;
  race: number;
  sex: number;
  health: number;
  max_health: number;
  mana: number;
  max_mana: number;
  move: number;
  max_move: number;
  alignment: number;
  gold: number;
  bank_gold: number;
  exp: number;
  room: number;
  ac: number;
  thac0: number;
  hitroll: number;
  damroll: number;
  stats: Record<string, number>;
  affects: number;
  connected_at: string;
  last_active: string;
  inventory: PlayerItem[];
  equipment: PlayerItem[];
}

export interface ServerMetrics {
  memory_alloc: number;
  memory_sys: number;
  memory_heap: number;
  goroutines: number;
  gc_cycles: number;
  last_gc: string;
  pause_total_ns: number;
  uptime: string;
  player_count: number;
  room_count: number;
  zone_count: number;
}

export interface Shop {
  keeper_vnum: number;
  keeper_name?: string;
  room_vnum?: number;
  buy_types: number[];
  sell_types: number[];
  profit_buy: number;
  profit_sell: number;
}

export const api = {
  health: () => request<Health>('/health'),
  zones: () => request<Zone[]>('/zones'),
  zone: (id: number) => request<Zone>(`/zones/${id}`),
  updateZone: (id: number, data: { lifespan?: number; reset_mode?: number }) =>
    request<Zone>(`/zones/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  server: () => request<ServerInfo>('/server'),
  mobs: () => request<Mob[]>('/mobs'),
  mob: (vnum: number) => request<Mob>(`/mobs/${vnum}`),
  objects: () => request<Obj[]>('/objects'),
  object: (vnum: number) => request<Obj>(`/objects/${vnum}`),
  room: (vnum: number) => request<Room>(`/rooms/${vnum}`),

  // Write methods (Phase 4)
  updateRoom: (vnum: number, data: { name?: string; description?: string }) =>
    request<Room>(`/rooms/${vnum}`, { method: 'PUT', body: JSON.stringify(data) }),

  updateMob: (vnum: number, data: Record<string, unknown>) =>
    request<Mob>(`/mobs/${vnum}`, { method: 'PUT', body: JSON.stringify(data) }),

  updateObject: (vnum: number, data: Record<string, unknown>) =>
    request<Obj>(`/objects/${vnum}`, { method: 'PUT', body: JSON.stringify(data) }),

  logs: (lines?: number) => request<string[]>(`/logs?lines=${lines || 50}`),
  players: () => request<PlayerInfo[]>('/players'),
  agents: () => request<AgentStatus[]>('/agents'),
  updateAgentStatus: (data: { agent_id: string; status: string }) =>
    request<AgentStatus>('/agents/status', { method: 'POST', body: JSON.stringify(data) }),
  findings: (params?: { status?: string; severity?: string; source?: string }) => {
    const qs = new URLSearchParams(params).toString();
    return request<Finding[]>(`/findings${qs ? '?' + qs : ''}`);
  },
  createFinding: (data: Omit<Finding, 'id' | 'created_at' | 'updated_at'>) =>
    request<Finding>('/findings', { method: 'POST', body: JSON.stringify(data) }),
  updateFinding: (id: number, data: { status: string }) =>
    request<Finding>(`/findings/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  triageSummaries: () => request<TriageSummary[]>('/triage/summaries'),

  // Shops (Phase 4)
  shops: () => request<Shop[]>('/shops'),
  shop: (keeperVnum: number) => request<Shop>(`/shops/${keeperVnum}`),
  updateShop: (keeperVnum: number, data: Record<string, unknown>) =>
    request<Shop>(`/shops/${keeperVnum}`, { method: 'PUT', body: JSON.stringify(data) }),

  // Zone reset
  resetZone: (zoneNumber: number) =>
    request<void>(`/zones/${zoneNumber}/reset`, { method: 'POST' }),

  // Phase 5 — Operations
  playerDetail: (name: string) => request<PlayerDetail>(`/players/${encodeURIComponent(name)}`),
  savePlayer: (name: string) => request<{ status: string }>(`/players/${encodeURIComponent(name)}/save`, { method: 'POST' }),
  kickPlayer: (name: string) => request<{ status: string }>(`/players/${encodeURIComponent(name)}/kick`, { method: 'POST' }),
  metrics: () => request<ServerMetrics>('/metrics'),
  saveWorld: () => request<{ status: string }>('/save-world', { method: 'POST' }),
  resetAllZones: () => request<{ status: string; count: number }>('/reset-all-zones', { method: 'POST' }),
};
