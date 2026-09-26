import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { StatCardSkeleton } from '../components/Skeleton';
import { Link } from 'react-router-dom';
import { olcApi } from '../api/olc';

function counterTotal(exposition: string | undefined, name: string): number | null {
  if (!exposition) return null;
  let total = 0;
  for (const line of exposition.split('\n')) {
    if (!line.startsWith(name + ' ') && !line.startsWith(name + '{')) continue;
    const value = Number(line.slice(line.lastIndexOf(' ') + 1));
    if (Number.isFinite(value)) total += value;
  }
  return total;
}


export function DashboardPage() {
  const {
    data: server,
    isLoading: serverLoading,
    error: serverError,
  } = useQuery({
    queryKey: ['server'],
    queryFn: api.server,
  });

  const {
    data: health,
    isLoading: healthLoading,
  } = useQuery({
    queryKey: ['health'],
    queryFn: api.health,
    refetchInterval: 30000,
  });

  const { data: metrics, error: metricsError } = useQuery({ queryKey: ['metrics'], queryFn: api.metrics, refetchInterval: 15000 });
  const { data: exposition, error: prometheusError } = useQuery({ queryKey: ['prometheus'], queryFn: api.prometheus, refetchInterval: 15000 });
  const { data: logs, error: logsError } = useQuery({ queryKey: ['dashboard-logs'], queryFn: () => api.logs(25), refetchInterval: 10000 });
  const { data: pending } = useQuery({ queryKey: ['olc-pending'], queryFn: olcApi.pending, refetchInterval: 30000, retry: false });
  const pendingZones = Array.from(new Set((pending || []).map((entry) => entry.zone)));
  const recentLog = logs?.slice(-3).reverse() || [];

  return (
    <div className="space-y-6">
      <div>
        <p className="font-mono text-xs uppercase tracking-widest text-accent">World desk</p>
        <h1 className="mt-1 text-2xl font-bold text-ink">Dashboard</h1>
        <p className="mt-2 text-sm text-ink-muted">Find what needs attention, then open the world search to jump straight into an editor.</p>
      </div>

      {/* Server Status */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {(serverLoading || healthLoading) ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard
              label="Status"
              value={health?.status === 'ok' ? 'Online' : 'Unreachable'}
              detail={server?.uptime ? `up ${server.uptime}` : undefined}
              error={!!serverError}
              color={health?.status === 'ok' ? 'green' : 'slate'}
            />
            <StatCard
              label="Zones"
              value={server?.zone_count?.toString() || '...'}
              error={!!serverError}
            />
            <StatCard
              label="Rooms"
              value={server?.room_count?.toString() || '...'}
              error={!!serverError}
            />
            <StatCard
              label="Players online"
              value={server?.player_count?.toString() || '...'}
              error={!!serverError}
            />
          </>
        )}
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <section className="border border-rule bg-paper-deep p-4">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="text-lg text-ink">Builder work</h2>
            <Link to="/admin/workshop" className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep">Open workshop →</Link>
          </div>
          <p className="mt-2 text-sm text-ink-muted">Committed edits waiting for their zone files to be saved.</p>
          <div className="mt-4 font-mono text-3xl text-ink">{pending ? pendingZones.length : '—'}<span className="ml-2 text-sm text-ink-muted">zones pending</span></div>
          {pendingZones.length > 0 && <p className="mt-2 text-xs text-accent">Zones {pendingZones.slice(0, 6).join(', ')}{pendingZones.length > 6 ? '…' : ''}</p>}
        </section>
        <section className="border border-rule bg-paper-deep p-4">
          <div className="flex items-baseline justify-between gap-3">
            <h2 className="text-lg text-ink">Server pulse</h2>
            <Link to="/admin/operations" className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep">Operations →</Link>
          </div>
          {metricsError ? <p className="mt-4 text-sm text-accent">Metrics unavailable</p> : <div className="mt-4 grid grid-cols-2 gap-4 text-sm"><div><span className="block text-xs text-ink-muted">Heap</span><strong className="font-mono font-normal text-ink">{metrics ? `${(metrics.memory_heap / 1048576).toFixed(1)} MB` : '—'}</strong></div><div><span className="block text-xs text-ink-muted">Goroutines</span><strong className="font-mono font-normal text-ink">{metrics?.goroutines ?? '—'}</strong></div></div>}
          <div className="mt-4 grid grid-cols-2 gap-4 border-t border-rule pt-3 text-sm">
            <div><span className="block text-xs text-ink-muted">Combat rounds</span><strong className="font-mono font-normal text-ink">{counterTotal(exposition, 'darkpawns_combat_rounds_total') ?? '—'}</strong></div>
            <div><span className="block text-xs text-ink-muted">Deaths</span><strong className="font-mono font-normal text-ink">{counterTotal(exposition, 'darkpawns_deaths_total') ?? '—'}</strong></div>
            <div><span className="block text-xs text-ink-muted">Commands</span><strong className="font-mono font-normal text-ink">{counterTotal(exposition, 'darkpawns_commands_processed_total') ?? '—'}</strong></div>
            <div><span className="block text-xs text-ink-muted">Connection errors</span><strong className="font-mono font-normal text-ink">{counterTotal(exposition, 'darkpawns_connection_errors_total') ?? '—'}</strong></div>
          </div>
          <p className="mt-3 text-xs text-ink-muted">Prometheus counters since server start{prometheusError ? ' · telemetry unavailable' : ''}.</p>
        </section>
      </div>

      <section className="border border-rule bg-paper-deep p-4">
        <div className="flex items-baseline justify-between gap-3"><h2 className="text-lg text-ink">Recent server log</h2><Link to="/admin/operations" className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep">Full log →</Link></div>
        {logsError ? <p className="mt-3 text-sm text-accent">Log unavailable</p> : recentLog.length === 0 ? <p className="mt-3 text-sm text-ink-muted">No entries yet.</p> : <div className="mt-3 space-y-1">{recentLog.map((line, index) => <p key={`${index}-${line}`} className="truncate border-t border-rule pt-2 font-mono text-xs text-ink-muted" title={line}>{line}</p>)}</div>}
      </section>

      {serverError && (
        <div className="bg-paper-deep border border-accent p-4 text-sm text-ink" role="status">
          The game server is not answering on port 4350. Figures above are the
          last values it reported.
        </div>
      )}
    </div>
  );
}

function StatCard({
  label,
  value,
  detail,
  error,
  color,
}: {
  label: string;
  value: string;
  detail?: string;
  error?: boolean;
  color?: 'green' | 'slate';
}) {
  return (
    <div className="bg-paper-deep rounded-none border border-rule p-4">
      <div className="text-xs text-ink-muted mb-1">{label}</div>
      <div
        className={`text-2xl font-bold ${
          error
            ? 'text-accent'
            : color === 'green'
              ? 'text-online'
              : 'text-ink'
        }`}
      >
        {error ? '—' : value}
      </div>
      {detail && !error && (
        <div className="text-xs text-ink-muted font-mono mt-1">{detail}</div>
      )}
    </div>
  );
}
