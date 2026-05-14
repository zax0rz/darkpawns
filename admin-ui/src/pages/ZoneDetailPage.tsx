import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { Skeleton, CardSkeleton } from '../components/Skeleton';

function resetModeLabel(mode: number): string {
  switch (mode) {
    case 0: return 'Never';
    case 1: return 'When empty';
    case 2: return 'Always';
    case 3: return 'Force reset';
    default: return `Mode ${mode}`;
  }
}

export function ZoneDetailPage() {
  const { id } = useParams<{ id: string }>();

  const { data: zone, isLoading, error } = useQuery({
    queryKey: ['zone', id],
    queryFn: () => api.zone(Number(id)),
    enabled: !!id,
  });

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
          className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 text-sm"
        >
          ← Back to Zones
        </Link>
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-4 text-sm text-red-700 dark:text-red-300">
          Zone not found or failed to load.
          <div className="mt-1 text-red-500/70 dark:text-red-400/70 text-xs">
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
        className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 text-sm"
      >
        ← Back to Zones
      </Link>

      {/* Header */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-amber-600 dark:text-amber-400">
            #{zone.number}
          </span>
          <h1 className="text-xl font-bold text-slate-900 dark:text-white">{zone.name}</h1>
        </div>
      </div>

      {/* Properties */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">
          Properties
        </h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatBlock label="Zone Number" value={zone.number} />
          <StatBlock label="Top Room" value={zone.top_room} />
          <StatBlock label="Lifespan" value={`${zone.lifespan} min`} />
          <StatBlock label="Reset Mode" value={resetModeLabel(zone.reset_mode)} />
        </div>
      </div>

      {/* Room range */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
          Room Range
        </h2>
        <p className="text-sm text-slate-700 dark:text-slate-200">
          Rooms{' '}
          <span className="font-mono text-amber-600 dark:text-amber-400">
            {zone.number * 100}
          </span>{' '}
          –{' '}
          <span className="font-mono text-amber-600 dark:text-amber-400">{zone.top_room}</span>
        </p>
        <p className="text-xs text-slate-400 mt-2">
          Individual room details will be available in Phase 4.
        </p>
      </div>
    </div>
  );
}

function StatBlock({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <div className="text-xs text-slate-400 mb-1">{label}</div>
      <div className="text-sm text-slate-900 dark:text-white font-mono">{value}</div>
    </div>
  );
}
