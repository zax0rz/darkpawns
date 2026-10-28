import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { Download } from 'lucide-react';
import { exportLogs, logbook, observeMudlog, type LogFilters } from '../api/observability';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { NativeSelect } from '../components/ui/select';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table';
import { dateTime } from '../lib/observability';

export function LogsPage() {
  const [params, setParams] = useSearchParams(); const source = params.get('source') === 'mudlog' ? 'mudlog' : 'server';
  const [paused, setPaused] = useState(false); const [stream, setStream] = useState('Connecting'); const [exporting, setExporting] = useState(false); const [exportError, setExportError] = useState(''); const cache = useQueryClient();
  const severity = params.get('severity') || ''; const search = params.get('search') || '';
  const filters: LogFilters = { source, search, ...(source === 'server' ? { severity: severity as LogFilters['severity'] } : { type: Number(params.get('type') ?? -1), level: Number(params.get('level') ?? -1), key: params.get('key') || '' }), after: params.get('after') || undefined, before: params.get('before') || undefined };
  const query = useQuery({ queryKey: ['logbook', filters], queryFn: () => logbook(filters), refetchInterval: paused ? false : 3000 });
  useEffect(() => {
    if (paused || source !== 'mudlog') return;
    const controller = new AbortController(); let retry: ReturnType<typeof setTimeout>;
    const connect = () => { void observeMudlog(controller.signal, () => setStream('Connected'), () => { void cache.invalidateQueries({ queryKey: ['logbook'] }); }).catch(() => {
      if (!controller.signal.aborted) { setStream('Reconnecting'); retry = setTimeout(connect, 3000); }
    }); };
    connect();
    return () => { controller.abort(); clearTimeout(retry); };
  }, [source, paused, cache]);
  const change = (key: string, value: string) => { const next = new URLSearchParams(params); if (value) next.set(key, value); else next.delete(key); setParams(next, { replace: true }); };
  async function download(format: 'log' | 'csv') { setExporting(true); setExportError(''); try { await exportLogs(filters, format); } catch (error) { setExportError(error instanceof Error ? error.message : 'Export failed'); } finally { setExporting(false); } }
  const entries = query.data?.entries || [];
  return <div className="space-y-5"><div className="flex flex-wrap justify-between gap-4"><div><h1 className="text-2xl">Logs</h1><p className="mt-1 text-sm text-ink-muted">Live retained logs. Filter here, then download the matching entries.</p></div><div className="flex flex-wrap gap-2"><Button variant="outline" disabled={exporting || query.isError || !query.data} onClick={() => download('log')}><Download aria-hidden className="size-4" />Raw .log</Button><Button variant="outline" disabled={exporting || query.isError || !query.data} onClick={() => download('csv')}><Download aria-hidden className="size-4" />Filtered CSV</Button></div></div>
    <div className="flex flex-wrap items-center gap-2 border-b border-rule pb-3"><Button variant={source === 'server' ? 'default' : 'outline'} aria-pressed={source === 'server'} onClick={() => setParams({ source: 'server' })}>Server log</Button><Button variant={source === 'mudlog' ? 'default' : 'outline'} aria-pressed={source === 'mudlog'} onClick={() => setParams({ source: 'mudlog' })}>Mudlog</Button><span className="ml-auto text-sm text-ink-muted" role="status">{paused ? 'Paused' : query.isError ? 'Refresh failed' : source === 'server' ? 'Live · refreshes every 3s' : `Live · ${stream.toLowerCase()}`}</span><Button variant="outline" onClick={() => setPaused(v => !v)}>{paused ? 'Resume live' : 'Pause live'}</Button></div>
    <p className="text-sm text-ink-muted">{source === 'server' ? 'The server log shows startup, connections, storage and service errors.' : 'Mudlog shows in-game events recorded for staff, limited to your immortal level.'}</p>
    <div className="grid items-end gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <label className="text-sm">Message contains<Input className="mt-1" value={search} onChange={e => change('search', e.target.value)} placeholder="Search messages" /></label>
      {source === 'server' ? <label className="text-sm">Level<NativeSelect className="mt-1 block w-full" value={severity} onChange={e => change('severity', e.target.value)}><option value="">All levels</option>{['ERROR', 'WARN', 'INFO', 'DEBUG'].map(l => <option key={l}>{l}</option>)}</NativeSelect></label> : <><label className="text-sm">Syslog type<NativeSelect className="mt-1 block w-full" value={params.get('type') || '-1'} onChange={e => change('type', e.target.value)}><option value="-1">All types</option><option value="0">OFF (0)</option><option value="1">BRF (≤1)</option><option value="2">NRM (≤2)</option><option value="3">CMP (≤3)</option></NativeSelect></label><label className="text-sm">Immortal level ceiling<Input className="mt-1" type="number" min={0} max={40} placeholder="Your actual level" value={params.get('level') || ''} onChange={e => change('level', e.target.value)} /></label><label className="text-sm">Site key<Input className="mt-1" value={params.get('key') || ''} onChange={e => change('key', e.target.value)} placeholder="Exact key" /></label></>}
      <label className="text-sm">From (UTC)<Input className="mt-1" type="datetime-local" value={(params.get('after') || '').slice(0, 16)} onChange={e => change('after', e.target.value ? new Date(`${e.target.value}Z`).toISOString() : '')} /></label>
      <label className="text-sm">Through (UTC)<Input className="mt-1" type="datetime-local" value={(params.get('before') || '').slice(0, 16)} onChange={e => change('before', e.target.value ? new Date(`${e.target.value}Z`).toISOString() : '')} /></label>
    </div>
    <div className="flex flex-wrap justify-between gap-2 text-sm text-ink-muted"><p>{entries.length} matching / {query.data?.retained ?? '—'} retained {source === 'mudlog' ? 'visible events' : 'lines'} · oldest first</p><Button variant="link" className="h-auto p-0" onClick={() => setParams({ source })}>Clear filters</Button></div>
    <p className="text-xs text-ink-muted">{source === 'mudlog' ? 'Type is OFF/BRF/NRM/CMP; level is minimum immortal visibility. Your actual level always caps access. Raw export preserves message text.' : 'Raw export preserves stored log lines. CSV columns: time, level, type, message.'} Downloads include the filtered retained ring only, up to 1,000 entries. Live activity can arrive between viewing and exporting.</p>
    {(query.isError || exportError) && <div role="alert" className="border border-accent p-3 text-sm text-accent">{exportError || (query.error instanceof Error ? query.error.message : 'Log refresh failed.')} <Button variant="outline" onClick={() => query.refetch()}>Retry</Button></div>}
    {query.data && query.data.dropped > 0 && <p role="status" className="text-accent">Observer dropped {query.data.dropped} events. This is a partial record.</p>}
    <div className="max-h-[36rem] overflow-auto border-y border-rule"><Table><TableHeader><TableRow><TableHead>Time (UTC)</TableHead><TableHead>Level</TableHead><TableHead>Type</TableHead><TableHead>Message</TableHead></TableRow></TableHeader><TableBody>{entries.map((e, i) => <TableRow key={e.seq || i}><TableCell title={e.time} className="whitespace-nowrap font-mono text-xs">{e.time ? dateTime(e.time) : 'Unstamped'}</TableCell><TableCell className={`font-mono text-xs ${e.level === 'ERROR' ? 'text-accent font-bold' : ''}`}>{e.level || '—'}</TableCell><TableCell className="font-mono text-xs">{e.type}</TableCell><TableCell className="min-w-64 whitespace-pre-wrap break-words font-mono text-xs">{e.message}</TableCell></TableRow>)}</TableBody></Table>{entries.length === 0 && <p className="p-6 text-sm text-ink-muted">{query.isLoading ? 'Loading retained entries…' : 'No retained entries match these filters. Clear filters or wait for activity.'}</p>}</div>
    <Link to="/admin/operations" className="text-sm text-accent underline underline-offset-4">Open operations and the original server tail</Link>
  </div>;
}
