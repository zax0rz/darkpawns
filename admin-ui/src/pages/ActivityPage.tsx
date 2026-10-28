import { useQuery } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { BarChart, Bar, CartesianGrid, XAxis, YAxis } from 'recharts';
import { activityEvents, dashboard } from '../api/observability';
import type { components } from '../api/schema';
import { Card } from '../components/ui/card';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '../components/ui/table';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '../components/ui/chart';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Skeleton } from '../components/Skeleton';
import { RangeControl } from './DashboardPage';
import { dateTime } from '../lib/observability';

const metrics = ['online', 'logins', 'characters', 'deaths', 'errors', 'warnings', 'release'] as const;
type Metric = typeof metrics[number];
const labels: Record<Metric, string> = { online: 'Players online', logins: 'Logins', characters: 'New characters', deaths: 'Player deaths', errors: 'Errors', warnings: 'Warnings', release: 'Release & uptime' };
type Activity = components['schemas']['Activity'];

function deathCause(e: Activity) {
  switch (e.cause) {
    case 'mob': return `${e.mob_name || 'A mob'}${e.mob_vnum ? ` · #${e.mob_vnum}` : ''}`;
    case 'player': return 'Another player';
    case 'trap': return 'A trap';
    case 'other': return 'Other damage';
    default: return 'Cause not recorded';
  }
}

function CharacterEntries({ events, deaths }: { events: Activity[]; deaths: boolean }) {
  return <>
    <ul className="divide-y divide-rule border-y border-rule md:hidden">
      {events.map(e => <li key={e.id} className="space-y-3 py-4">
        <div><div className="flex flex-wrap items-baseline justify-between gap-2"><span className="text-lg">{e.player}</span><span className="text-sm text-ink-muted">{e.class || 'Class not recorded'}</span></div><time dateTime={e.time} className="font-mono text-xs text-ink-muted">{dateTime(e.time)} UTC</time></div>
        {deaths && <dl className="grid grid-cols-[4rem_minmax(0,1fr)] gap-x-3 gap-y-2 text-sm"><dt className="text-ink-muted">Where</dt><dd className="min-w-0 break-words">{e.room_name || 'Room not recorded'}{e.room_vnum >= 0 && <span className="ml-2 font-mono text-xs text-ink-muted">#{e.room_vnum}</span>}</dd><dt className="text-ink-muted">To what</dt><dd className="min-w-0 break-words">{deathCause(e)}</dd></dl>}
      </li>)}
    </ul>
    <div className="hidden md:block"><Table><TableHeader><TableRow><TableHead>Time (UTC)</TableHead><TableHead>Character</TableHead><TableHead>Class</TableHead>{deaths && <><TableHead>Where</TableHead><TableHead>To what</TableHead></>}</TableRow></TableHeader><TableBody>{events.map(e => <TableRow key={e.id}><TableCell className="font-mono text-xs">{dateTime(e.time)}</TableCell><TableCell>{e.player}</TableCell><TableCell>{e.class || 'Not recorded'}</TableCell>{deaths && <><TableCell className="min-w-44 whitespace-normal">{e.room_name || 'Room not recorded'}{e.room_vnum >= 0 && <span className="block font-mono text-xs text-ink-muted">#{e.room_vnum}</span>}</TableCell><TableCell className="min-w-40 whitespace-normal">{deathCause(e)}</TableCell></>}</TableRow>)}</TableBody></Table></div>
  </>;
}

export function ActivityPage() {
  const [params, setParams] = useSearchParams();
  const range = params.get('range') === '7d' ? '7d' : '24h';
  const candidate = params.get('metric') || 'logins';
  const metric: Metric = metrics.includes(candidate as Metric) ? candidate as Metric : 'logins';
  const kind = metric === 'logins' ? 'login' : metric === 'characters' ? 'character' : 'death';
  const named = metric === 'logins' || metric === 'characters' || metric === 'deaths';
  const search = (params.get('search') || '').slice(0, 32);
  const before = /^\d+$/.test(params.get('before') || '') ? Number(params.get('before')) : undefined;
  const change = (key: string, value: string) => {
    const next = new URLSearchParams(params);
    if (value) next.set(key, value); else next.delete(key);
    if (key !== 'before') next.delete('before');
    setParams(next, { replace: key === 'search' });
  };
  const query = useQuery({ queryKey: ['dashboard', range], queryFn: () => dashboard(range), refetchInterval: 10000 });
  const entries = useQuery({ queryKey: ['activity-events', range, kind, search, before], queryFn: () => activityEvents({ range, kind, search, before, limit: 50 }), enabled: named, refetchInterval: 10000 });
  const data = query.data;
  const events = entries.data?.entries || [];
  const binWidth = (range === '24h' ? 3600 : 86400) * 1000;
  const knownDeath = (time: string) => !!data?.activity_since && Date.parse(time) + binWidth > Date.parse(data.activity_since);
  const chart = (data?.points || []).map(p => ({ label: dateTime(p.time), value: !p.samples || metric === 'release' || (metric === 'deaths' && !knownDeath(p.time)) ? null : metric === 'deaths' ? p.player_deaths : p[metric] }));

  return <div className="space-y-6">
    <Link to="/admin/" className="text-sm text-accent underline underline-offset-4">Dashboard</Link>
    <div className="flex flex-wrap justify-between gap-4"><div><h1 className="text-2xl">Activity & release</h1><p className="mt-1 text-sm text-ink-muted">See who arrived and what happened. All times are UTC.</p></div><RangeControl value={range} onChange={v => change('range', v)} /></div>
    <div className="flex flex-wrap gap-2" aria-label="Activity metric">{metrics.map(key => <Button key={key} variant={key === metric ? 'default' : 'outline'} aria-pressed={key === metric} onClick={() => change('metric', key)}>{labels[key]}</Button>)}</div>
    {query.isError && <p role="alert" className="text-accent">Couldn't refresh activity. Check that the server is running, then <Button variant="outline" onClick={() => query.refetch()}>Try again</Button></p>}

    {metric === 'release' ? <section className="border-y border-rule py-4"><h2 className="text-xl">Running release</h2><dl className="mt-3 grid gap-4 text-sm md:grid-cols-3"><div><dt className="text-ink-muted">Release / commit</dt><dd className="mt-1 break-all font-mono">{data?.release.version || 'Unknown'} · {data?.release.commit || 'No VCS metadata'}</dd></div><div><dt className="text-ink-muted">Process started</dt><dd className="mt-1 font-mono">{data ? `${dateTime(data.started_at)} UTC` : '—'}</dd></div><div><dt className="text-ink-muted">Last observed update</dt><dd className="mt-1">{data?.release.updated_at ? `${dateTime(data.release.updated_at)} UTC` : 'Unknown until a running executable change is recorded.'}</dd></div></dl><p className="mt-3 text-xs text-ink-muted">{data?.release.observed_at && !data.release.observed_at.startsWith('0001') ? `This executable first observed ${dateTime(data.release.observed_at)} UTC. ` : ''}An observed update records a changed executable at startup; it is not a build timestamp.</p></section> : <>
      {named && <section className="space-y-4" aria-labelledby="activity-entries-heading">
        <div className="flex flex-wrap items-end justify-between gap-4"><div><h2 id="activity-entries-heading" className="text-xl">{labels[metric]}</h2><p className="mt-1 max-w-prose text-sm text-ink-muted">{metric === 'logins' ? 'Successful returning-character sign-ins, including reconnects. New characters and guest visits are separate.' : metric === 'characters' ? 'Characters saved after their creation choices were accepted.' : 'Player deaths, including death traps. Mob deaths do not count here.'}</p></div><label className="block w-full text-sm sm:w-64">Find a character<Input className="mt-1" value={search} maxLength={32} onChange={e => change('search', e.target.value)} placeholder="Character name" /></label></div>
        {entries.isLoading ? <Skeleton className="h-32 w-full" /> : entries.isError ? <p role="alert" className="text-accent">Couldn't load character activity. Check your admin access and connection, then <Button variant="outline" onClick={() => entries.refetch()}>Try again</Button></p> : !entries.data?.recording ? <p className="border-y border-rule py-4 text-sm">Character activity recording is not configured.</p> : <>
          <p className="text-xs text-ink-muted">Details recorded since {entries.data.since ? `${dateTime(entries.data.since)} UTC` : 'this recorder started'}. Saved for seven days, up to 20,000 entries across all activity. Hidden staff names follow your immortal level. Earlier counts may have no character details.</p>
          {(entries.data.dropped > 0 || entries.data.trimmed > 0) && <p role="status" className="text-sm text-accent">Since recording began, some activity details were missed or removed at the entry limit. Totals can be higher than this list.</p>}
          {events.length ? <CharacterEntries events={events} deaths={metric === 'deaths'} /> : <p className="border-y border-rule py-6 text-sm">{search ? 'No matching characters in this window. Try another name or clear the search.' : 'No entries in this window. Details appear as new activity is recorded.'}</p>}
          <div className="flex flex-wrap items-center justify-between gap-3 text-sm"><p className="text-ink-muted">{events.length} shown · {entries.data.total} visible {entries.data.total === 1 ? 'entry' : 'entries'} in this window · refreshes every 10s</p><div className="flex gap-2">{before !== undefined && <Button variant="outline" onClick={() => change('before', '')}>Newest entries</Button>}<Button variant="outline" disabled={!entries.data.next_before} onClick={() => change('before', String(entries.data?.next_before || ''))}>Older entries</Button></div></div>
        </>}
      </section>}
      {metric === 'deaths' && <p className="max-w-prose text-sm text-ink-muted">Player deaths have been counted separately since {data?.activity_since ? `${dateTime(data.activity_since)} UTC` : 'this recorder started'}. Earlier combined counts include mobs and are not used here. A cause the game did not supply stays unrecorded.</p>}
      {query.isLoading ? <Skeleton className="h-56 w-full" /> : <Card className="gap-3 p-4"><h2 className="text-xl">{labels[metric]} over time</h2><p className="text-sm text-ink-muted">Recorded observations by UTC {range === '24h' ? 'hour' : 'day'}.</p><ChartContainer className="h-56 w-full aspect-auto" config={{ value: { label: labels[metric], color: 'var(--color-accent)' } }}><BarChart data={chart} accessibilityLayer><CartesianGrid vertical={false} /><XAxis dataKey="label" minTickGap={60} tickLine={false} /><YAxis allowDecimals={false} width={32} /><ChartTooltip content={<ChartTooltipContent />} /><Bar dataKey="value" fill="var(--color-accent)" isAnimationActive={false} /></BarChart></ChartContainer>{data && !(data.points || []).some(p => p.samples > 0) && <p className="text-sm text-ink-muted">{data.recording ? 'No observations in this window. History appears as the server records activity.' : 'History recording is not configured.'}</p>}</Card>}
      {(metric === 'errors' || metric === 'warnings') && <p className="border-y border-rule py-3 text-sm text-ink-muted">These are historical totals for the selected window. <Link className="text-accent underline underline-offset-4" to={`/admin/operations/logs?severity=${metric === 'errors' ? 'ERROR' : 'WARN'}&after=${encodeURIComponent(data?.points?.[0]?.time || '')}`}>Inspect saved {metric}</Link>. The log holds up to 1,000 entries and can contain fewer records than these totals.</p>}
      <details className="border-t border-rule pt-4"><summary className="cursor-pointer text-lg">Observation detail</summary><p className="my-3 max-w-prose text-sm text-ink-muted">Peak is the largest ten-second sample. Counts cover observed time only. Error and warning counts come from server logs.</p><Table><TableHeader><TableRow>{['UTC interval start', 'Online / peak', 'Logins', 'New characters', 'Player deaths', 'Errors', 'Warnings', 'Coverage'].map(s => <TableHead key={s}>{s}</TableHead>)}</TableRow></TableHeader><TableBody>{(data?.points || []).slice().reverse().map(p => <TableRow key={p.time}><TableCell className="font-mono text-xs">{dateTime(p.time)}</TableCell><TableCell className="font-mono">{p.samples ? `${p.online} / ${p.peak}` : '—'}</TableCell>{(['logins', 'characters', 'deaths', 'errors', 'warnings'] as const).map(k => <TableCell key={k} className="font-mono">{p.samples && (k !== 'deaths' || knownDeath(p.time)) ? k === 'deaths' ? p.player_deaths : p[k] : '—'}</TableCell>)}<TableCell className="text-xs">{p.samples ? `${Math.min(100, Math.round(p.samples / (range === '24h' ? 360 : 8640) * 100))}% · ${p.samples} ${p.samples === 1 ? 'sample' : 'samples'}` : 'No observations'}</TableCell></TableRow>)}</TableBody></Table></details>
    </>}
    {data?.persistence_error && <p role="alert" className="text-accent">History could not be saved. Check server storage permissions.</p>}
  </div>;
}
