import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';
import { StatCardSkeleton } from '../components/Skeleton';

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

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Dashboard</h1>

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
              value={health?.status || '...'}
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
              label="Players"
              value={server?.player_count?.toString() || '...'}
              error={!!serverError}
            />
          </>
        )}
      </div>

      {/* Uptime */}
      {server?.uptime && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4">
          <span className="text-sm text-slate-500 dark:text-slate-400">Uptime: </span>
          <span className="text-sm text-slate-900 dark:text-white font-mono">{server.uptime}</span>
        </div>
      )}

      {/* Placeholder cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <PlaceholderCard title="Recent Findings" icon="🔍" phase={6} />
        <PlaceholderCard title="Agent Status" icon="🤖" phase={6} />
      </div>

      {/* Dev note */}
      <div className="bg-white/50 dark:bg-slate-800/50 rounded border border-dashed border-slate-300 dark:border-slate-600 p-4 text-sm text-slate-500 dark:text-slate-400">
        💡 If server stats show "...", the Go server may not be running on port
        4350. Start it with{' '}
        <code className="bg-slate-100 dark:bg-slate-700 px-1 rounded">go run ./cmd/server</code>
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  error,
  color,
}: {
  label: string;
  value: string;
  error?: boolean;
  color?: 'green' | 'slate';
}) {
  return (
    <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4">
      <div className="text-xs text-slate-500 dark:text-slate-400 mb-1">{label}</div>
      <div
        className={`text-2xl font-bold ${
          error
            ? 'text-red-500 dark:text-red-400'
            : color === 'green'
              ? 'text-green-600 dark:text-green-400'
              : 'text-slate-900 dark:text-white'
        }`}
      >
        {error ? '—' : value}
      </div>
    </div>
  );
}

function PlaceholderCard({
  title,
  icon,
  phase,
}: {
  title: string;
  icon: string;
  phase: number;
}) {
  return (
    <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
      <div className="flex items-center gap-2 mb-3">
        <span className="text-xl">{icon}</span>
        <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300">{title}</h3>
      </div>
      <div className="text-xs text-slate-400 dark:text-slate-500">
        Coming in Phase {phase}
      </div>
    </div>
  );
}
