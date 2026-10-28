import { useAuth } from '../hooks/useAuth';
import { useQuery } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { ArrowUpRight, CircleCheck, RefreshCw } from 'lucide-react';
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Legend } from 'recharts';
import { api } from '../api/client';
import { dashboard } from '../api/observability';
import { Button } from '../components/ui/button';
import { Card } from '../components/ui/card';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '../components/ui/chart';
import { Skeleton } from '../components/Skeleton';

import { dateTime } from '../lib/observability';
export function RangeControl({ value, onChange }: { value: '24h' | '7d'; onChange: (value: '24h' | '7d') => void }) {
  return <div className="flex gap-1" aria-label="History window">{(['24h', '7d'] as const).map(v => <Button key={v} variant={v === value ? 'default' : 'outline'} aria-pressed={v === value} onClick={() => onChange(v)}>{v === '24h' ? '24 hours' : '7 days'}</Button>)}</div>;
}
export function DashboardPage() {
  const { hasRole } = useAuth();
  const [params, setParams] = useSearchParams(); const range = params.get('range') === '7d' ? '7d' : '24h';
  const query = useQuery({ queryKey: ['dashboard', range], queryFn: () => dashboard(range), refetchInterval: 10000 });
  const daily = useQuery({ queryKey: ['dashboard', '7d'], queryFn: () => dashboard('7d'), refetchInterval: 10000 });
  const health = useQuery({ queryKey: ['health'], queryFn: api.health, refetchInterval: 10000 });
  const data = query.data; const points = data?.points || []; const observed = points.some(p => p.samples > 0);
  const total = (key: 'logins' | 'characters' | 'deaths' | 'errors' | 'warnings') => observed ? points.reduce((sum, p) => sum + p[key], 0).toLocaleString() : '—';
  const samples = points.reduce((sum, p) => sum + p.samples, 0);
  const covered = points.filter(p => p.samples > 0).length;
  const historyMessage = !data ? query.isError ? 'History unavailable. Recording status is unknown.' : 'Loading history…' : !data.recording ? 'Historical recording is not configured.' : !observed ? 'No online history yet. Waiting for the first server observation.' : 'Lines join aggregated observations. Uncovered intervals break the line; partially sampled intervals can contain gaps.';
  const activity = `/admin/activity?range=${range}`;
  const chartData = points.map(p => ({ ...p, label: range === '24h' ? new Date(p.time).toISOString().slice(11, 16) : new Date(p.time).toISOString().slice(5, 10) }));
  const dailyData = (daily.data?.points || []).map(p => ({ ...p, label: new Date(p.time).toISOString().slice(5, 10), logins: p.samples ? p.logins : null, characters: p.samples ? p.characters : null }));
  return <div className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4"><div><h1 className="text-2xl">Dashboard</h1><p className="mt-1 text-sm text-ink-muted">Server health and the people playing. All history in UTC.</p></div><RangeControl value={range} onChange={v => setParams({ range: v })} /></div>
    {query.isError && <div role="alert" className="border border-accent p-4 text-accent">Dashboard refresh failed. {data ? 'Showing the last response.' : 'No observations available.'} <Button variant="outline" onClick={() => query.refetch()}>Retry</Button></div>}
    <div className="flex flex-wrap items-center justify-between gap-3 border-y border-rule py-3 text-sm"><span className="flex items-center gap-2">{health.data?.status === 'ok' && !health.isError ? <CircleCheck className="size-4 text-online" /> : <RefreshCw className="size-4" />} {health.isError ? 'Health check failed' : health.data?.status === 'ok' ? 'Server responding' : 'Checking server…'}</span><span className="text-ink-muted">{data ? `Updated ${dateTime(data.now)} UTC · refreshes every 10s` : 'Waiting for observations'}</span></div>
    {query.isLoading ? <Skeleton className="h-44 w-full" /> : <div className="grid grid-cols-2 gap-px border border-rule bg-rule lg:grid-cols-4">
      <Metric label="Players online now" value={data?.online.toLocaleString() ?? '—'} detail="Characters in the world" to="/admin/operations" />
      <Metric label="Logins" value={total('logins')} detail={`Recorded · ${range}`} to={`${activity}&metric=logins`} />
      <Metric label="New characters" value={total('characters')} detail={`Accepted creations · ${range}`} to={`${activity}&metric=characters`} />
      <Metric label="Deaths" value={total('deaths')} detail={`Players + mobs · ${range}`} to={`${activity}&metric=deaths`} />
      <Metric label="Errors" value={total('errors')} detail={`Server records · ${range}`} to={`${activity}&metric=errors`} attention={observed && total('errors') !== '0'} />
      <Metric label="Warnings" value={total('warnings')} detail={`Server records · ${range}`} to={`${activity}&metric=warnings`} />
      <Metric label="Uptime" value={data ? uptime(data.uptime_seconds) : '—'} detail="Current process" to={`${activity}&metric=release`} />
      <Metric label="Current release" value={data?.release.version || 'Unknown'} detail={data?.release.updated_at ? `Updated ${dateTime(data.release.updated_at)} UTC` : 'Last update not yet observed'} to={`${activity}&metric=release`} />
    </div>}
    <div className="grid gap-6 xl:grid-cols-[3fr_2fr]">
      <Card className="gap-4 p-4 md:p-6"><div className="flex justify-between gap-3"><div><h2 className="text-xl">Players over time</h2><p className="mt-1 text-sm text-ink-muted">Latest observation and peak in each {range === '24h' ? 'hour' : 'UTC day'}.</p></div><Link className="text-accent underline underline-offset-4" to={`${activity}&metric=online`}>Details</Link></div>
        <ChartContainer className="h-64 w-full aspect-auto" config={{ online: { label: 'Latest', color: 'var(--color-accent)' }, peak: { label: 'Peak', color: 'var(--color-ink-muted)' } }}><LineChart data={chartData} accessibilityLayer><CartesianGrid vertical={false} /><XAxis dataKey="label" minTickGap={30} tickLine={false} axisLine={false} /><YAxis allowDecimals={false} width={32} tickLine={false} axisLine={false} domain={[0, 'auto']} /><ChartTooltip content={<ChartTooltipContent />} /><Legend /><Line name="Latest" type="stepAfter" dataKey="online" stroke="var(--color-accent)" strokeWidth={2} dot={{ r: 2 }} isAnimationActive={false} connectNulls={false} /><Line name="Peak" dataKey="peak" stroke="var(--color-ink-muted)" strokeDasharray="4 4" dot={false} isAnimationActive={false} connectNulls={false} /></LineChart></ChartContainer>
        <p className="text-xs text-ink-muted">{data && <span className="block mb-1">{covered} / {points.length} intervals observed · {samples.toLocaleString()} ten-second samples. <Link to={`${activity}&metric=online`} className="text-accent underline underline-offset-4">Inspect partial coverage</Link></span>}{historyMessage}</p>
      </Card>
      <Card className="gap-4 p-4 md:p-6"><div className="flex justify-between gap-3"><div><h2 className="text-xl">Daily arrivals</h2><p className="mt-1 text-sm text-ink-muted">Logins and accepted characters over seven UTC days.</p></div><Link className="text-accent underline underline-offset-4" to="/admin/activity?range=7d">Details</Link></div>
        <ChartContainer className="h-64 w-full aspect-auto" config={{ logins: { label: 'Logins', color: 'var(--color-accent)' }, characters: { label: 'New characters', color: 'var(--color-ink)' } }}><BarChart data={dailyData} accessibilityLayer><CartesianGrid vertical={false} /><XAxis dataKey="label" tickLine={false} axisLine={false} /><YAxis allowDecimals={false} width={32} tickLine={false} axisLine={false} /><ChartTooltip content={<ChartTooltipContent />} /><Legend /><Bar name="Logins" dataKey="logins" fill="var(--color-accent)" isAnimationActive={false} /><Bar name="New characters" dataKey="characters" fill="var(--color-ink)" isAnimationActive={false} /></BarChart></ChartContainer>
        <p className="text-xs text-ink-muted">{daily.isError ? 'Daily history unavailable. Retry by refreshing the page.' : 'Only observed periods count. Partial days are not full-day totals.'}</p>
      </Card>
    </div>
    {hasRole('builder') && <Link to="/admin/staff/motd" className="group flex items-center justify-between gap-4 border border-rule bg-paper-deep p-5 hover:bg-paper focus-visible:outline-2 focus-visible:outline-accent"><div><h2 className="text-xl">MOTD</h2><p className="mt-1 text-sm text-ink-muted">Edit the welcome messages players see at login. Preview colours and line wrapping before saving.</p></div><ArrowUpRight aria-hidden className="size-5 shrink-0 text-accent" /></Link>}
    <div className="flex flex-wrap justify-between gap-3 border-t border-rule pt-4 text-sm text-ink-muted"><p>{data?.recording_since ? `Recording since ${dateTime(data.recording_since)} UTC. Retains 7 days.` : data ? 'Historical recording is not configured.' : query.isError ? 'Recording status unknown while the dashboard is unavailable.' : 'Loading recording status…'}</p><Link to={activity} className="text-accent underline underline-offset-4">Inspect coverage and activity</Link></div>
    {data?.persistence_error && <p role="alert" className="text-accent">History is recording in memory but cannot be saved. Check server storage permissions.</p>}
  </div>;
}
function Metric({ label, value, detail, to, attention }: { label: string; value: string; detail: string; to: string; attention?: boolean }) {
  return <Link to={to} className="group bg-paper p-4 transition-colors hover:bg-paper-deep focus-visible:outline-2 focus-visible:outline-accent"><div className="flex items-center justify-between gap-2 text-sm text-ink-muted">{label}<ArrowUpRight aria-hidden className="size-4 shrink-0 group-hover:text-accent" /></div><div className={`mt-2 break-words font-mono text-2xl tabular-nums ${attention ? 'text-accent' : 'text-ink'}`}>{value}</div><p className="mt-2 text-xs text-ink-muted">{detail}</p></Link>;
}
function uptime(seconds: number) { const d = Math.floor(seconds / 86400); const h = Math.floor(seconds % 86400 / 3600); const m = Math.floor(seconds % 3600 / 60); return d ? `${d}d ${h}h` : h ? `${h}h ${m}m` : `${m}m`; }
