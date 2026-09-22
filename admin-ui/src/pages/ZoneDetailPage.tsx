import { useState } from 'react';
import { useQueries, useQuery, useQueryClient } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { olcApi, type OlcSchema } from '../api/olc';
import { Skeleton, CardSkeleton } from '../components/Skeleton';
import { resetModeLabel } from '../lib/zoneLabels';


export function ZoneDetailPage() {
  const { id } = useParams<{ id: string }>();
  const queryClient = useQueryClient();

  const { data: zone, isLoading, error } = useQuery({
    queryKey: ['zone', id],
    queryFn: () => api.zone(Number(id)),
    enabled: !!id,
  });
  const zoneNumber = Number(id);
  const vnumMapQuery = useQuery({ queryKey: ['olc-vnum-map', zoneNumber], queryFn: () => olcApi.vnumMap(zoneNumber), enabled: Number.isInteger(zoneNumber), retry: false });
  const schemaQueries = useQueries({
    queries: ['room', 'mob', 'obj', 'shop'].map((kind) => ({ queryKey: ['olc-schema', kind], queryFn: () => olcApi.schema(kind), staleTime: 30 * 60 * 1000, retry: false })),
  });
  const zoneSchemaQuery = useQuery({ queryKey: ['olc-schema', 'zone'], queryFn: () => olcApi.schema('zone'), staleTime: 30 * 60 * 1000, retry: false });
  const [resetting, setResetting] = useState(false);
  const [resetResult, setResetResult] = useState('');

  const handleReset = async () => {
    if (!id) return;
    setResetting(true);
    setResetResult('');
    try {
      await api.resetZone(Number(id));
      setResetResult('Zone reset triggered successfully.');
      queryClient.invalidateQueries({ queryKey: ['zone', id] });
    } catch (err) {
      setResetResult(`Reset failed: ${(err as Error).message}`);
    } finally {
      setResetting(false);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-4 w-24" />
        <CardSkeleton />
        <CardSkeleton />
      </div>
    );
  }

  if (error || !zone) {
    return (
      <div className="space-y-4">
        <Link
          to="/admin/game/zones"
          className="text-accent hover:text-accent text-sm"
        >
          ← Back to Zones
        </Link>
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Zone not found or failed to load.
          <div className="mt-1 text-accent text-xs">
            {(error as Error)?.message || 'Not found'}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link
        to="/admin/game/zones"
        className="text-accent hover:text-accent text-sm"
      >
        ← Back to Zones
      </Link>

      {/* Header */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-accent">
            #{zone.number}
          </span>
          <h1 className="text-xl font-bold text-ink">{zone.name}</h1>
        </div>
        <Link
          to={`/admin/game/zones/${zone.number}/edit`}
          className="mt-4 inline-block border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep"
        >
          Edit zone
        </Link>
      </div>

      {/* Properties */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-medium text-ink-muted">
            Properties
          </h2>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatBlock label="Zone Number" value={zone.number} />
          <StatBlock label="Top Room" value={zone.top_room} />
          <StatBlock label="Lifespan" value={`${zone.lifespan} min`} />
              <StatBlock label="Reset Mode" value={resetModeLabel(zone.reset_mode, zoneSchemaQuery.data)} />
        </div>
      </div>

      <VNumWorkshop map={vnumMapQuery.data} schemas={schemaQueries.map((query) => query.data as OlcSchema | undefined)} loading={vnumMapQuery.isLoading || schemaQueries.some((query) => query.isLoading)} />

      {/* Reset Zone */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <h2 className="text-sm font-medium text-ink-muted mb-3">
          Zone Reset
        </h2>
        <button
          onClick={handleReset}
          disabled={resetting}
          className="bg-accent hover:bg-accent disabled:opacity-50 text-ink px-4 py-2 rounded text-sm font-medium"
        >
          {resetting ? 'Resetting...' : 'Reset Zone'}
        </button>
        {resetResult && (
          <div className="mt-2 text-sm text-ink-muted">{resetResult}</div>
        )}
      </div>

      {/* Room range */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <h2 className="text-sm font-medium text-ink-muted mb-2">
          Room Range
        </h2>
        <p className="text-sm text-ink-muted">
          Rooms{' '}
          <span className="font-mono text-accent">
            {zone.number * 100}
          </span>{' '}
          –{' '}
          <span className="font-mono text-accent">{zone.top_room}</span>
        </p>
      </div>
    </div>
  );
}

function VNumWorkshop({ map, schemas, loading }: { map?: { kinds: { kind: string; used: number[]; free: { start: number; end: number }[]; suggested: number }[] }; schemas: (OlcSchema | undefined)[]; loading: boolean }) {
  const routes: Record<string, string> = { room: '/admin/game/rooms/', mob: '/admin/game/mobs/', obj: '/admin/game/objects/', shop: '/admin/game/shops/' };
  const labels: Record<string, string> = { room: 'Rooms', mob: 'Mobs', obj: 'Objects', shop: 'Shops' };
  const actions: Record<string, string> = { room: 'new_room', mob: 'new_mob', obj: 'new_obj', shop: 'new_shop' };
  return (
    <section className="border border-rule bg-paper-deep p-6">
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <div><h2 className="text-lg text-ink">Builder workshop</h2><p className="mt-1 text-sm text-ink-muted">Read-only free-VNUM map for this zone. Opening a suggested VNUM starts the draft; this map does not claim it.</p></div>
        <Link to="/admin/workshop" className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep">Claims and saves →</Link>
      </div>
      {loading && <div className="mt-4 text-sm text-ink-muted">Loading free-VNUM map…</div>}
      {!loading && map && (
        <div className="mt-4 grid gap-3 md:grid-cols-2">
          {map.kinds.map((entry) => {
            const schema = schemas.find((candidate) => candidate?.kind === entry.kind);
            const action = schema?.actions.find((candidate) => candidate.key === actions[entry.kind]);
            const suggested = entry.suggested;
            return (
              <div key={entry.kind} className="border border-rule bg-paper p-4">
                <div className="flex items-baseline justify-between gap-2"><h3 className="text-sm font-semibold text-ink">{labels[entry.kind] || entry.kind}</h3><span className="font-mono text-xs text-ink-muted">{entry.used.length} used</span></div>
                <p className="mt-2 text-xs text-ink-muted">Free: {entry.free.length ? entry.free.map((range) => range.start === range.end ? String(range.start) : range.start + '–' + range.end).join(', ') : 'none'}</p>
                {suggested > 0 && action?.allowed ? <Link to={(routes[entry.kind] || '/') + suggested + '/edit'} className="mt-3 inline-block border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">{action.label} · #{suggested}</Link> : <p className="mt-3 text-xs text-ink-muted">{action && !action.allowed ? 'Requires ' + action.requiredLabel + ' (level ' + action.requiredLevel + ').' : 'No free VNUM is available.'}</p>}
              </div>
            );
          })}
        </div>
      )}
      {!loading && !map && <p className="mt-4 text-sm text-accent">The free-VNUM map is unavailable for this zone.</p>}
      <p className="mt-4 text-xs text-ink-muted">Suggested values are advisory. The server revalidates existence, zone ownership, and capability when a draft opens.</p>
    </section>
  );
}

function StatBlock({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <div className="text-xs text-ink-muted mb-1">{label}</div>
      <div className="text-sm text-ink font-mono">{value}</div>
    </div>
  );
}
