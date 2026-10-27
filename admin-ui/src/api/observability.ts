import { http, read } from './http';
import type { paths } from './schema';
export type LogFilters = NonNullable<paths['/admin/logbook']['get']['parameters']['query']>;
export const dashboard = (range: '24h' | '7d') => read(http.GET('/admin/dashboard', { params: { query: { range } } }));
export const logbook = (filters: LogFilters) => read(http.GET('/admin/logbook', { params: { query: filters } }));
export async function exportLogs(filters: LogFilters, format: 'log' | 'csv') {
  const result = await http.GET('/admin/logbook/export', { params: { query: { ...filters, format } }, parseAs: 'blob' });
  if (!result.data) throw new Error('Export returned no data');
  const url = URL.createObjectURL(result.data);
  const a = document.createElement('a'); a.href = url; a.download = `darkpawns-${filters.source || 'server'}.${format}`; a.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// Native EventSource cannot send the console's mandatory CSRF header. Keep the
// existing SSE transport through fetch rather than weakening cookie guards.
export async function observeMudlog(signal: AbortSignal, connected: () => void, changed: () => void) {
  const response = await fetch('/admin/mudlog/stream', { credentials: 'same-origin', headers: { 'X-Requested-With': 'darkpawns-admin', Accept: 'text/event-stream' }, signal });
  if (!response.ok || !response.body) throw new Error(`Mudlog stream unavailable (${response.status})`);
  connected();
  const reader = response.body.getReader(); const decoder = new TextDecoder(); let buffer = '';
  try {
    while (!signal.aborted) {
      const chunk = await reader.read(); if (chunk.done) throw new Error('Mudlog stream disconnected');
      buffer += decoder.decode(chunk.value, { stream: true });
      const frames = buffer.split('\n\n'); buffer = frames.pop() || '';
      if (frames.some(frame => frame.split('\n').includes('event: mudlog'))) changed();
    }
  } finally { reader.releaseLock(); }
}
