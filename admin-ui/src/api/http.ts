import createClient from 'openapi-fetch';
import type { paths } from './schema';

export class ApiError extends Error {
  readonly status: number;
  readonly payload: unknown;
  constructor(status: number, payload: unknown, message: string) {
    super(message); this.status = status; this.payload = payload; this.name = 'ApiError';
  }
}

export const http = createClient<paths>({
  credentials: 'same-origin',
  headers: { 'X-Requested-With': 'darkpawns-admin' },
});

http.use({
  async onResponse({ request, response }) {
    if (response.ok) return;
    if (response.status === 401 && !['/admin/login', '/admin/session'].includes(new URL(request.url).pathname)) { window.location.href = '/login'; }
    const text = await response.clone().text();
    let payload: unknown = text;
    try { payload = JSON.parse(text); } catch { /* Preserve non-JSON failures. */ }
    const body = typeof payload === 'object' && payload !== null ? payload as Record<string, unknown> : {};
    const message = typeof body.error === 'string' ? body.error : typeof body.detail === 'string' ? body.detail : `API error ${response.status}: ${text}`;
    throw new ApiError(response.status, payload, message);
  },
});

// The generated client validates paths, methods, parameters and bodies at each
// call site. Successful no-content operations intentionally resolve undefined.
export async function read<T>(result: Promise<{ data?: T }>): Promise<T> {
  return (await result).data as T;
}

// Prometheus and SSE are explicitly outside the typed JSON API.
export async function prometheus(): Promise<string> {
  const response = await fetch('/admin/prometheus', { credentials: 'same-origin', headers: { 'X-Requested-With': 'darkpawns-admin' } });
  if (!response.ok) throw new ApiError(response.status, null, `API error ${response.status}`);
  return response.text();
}
