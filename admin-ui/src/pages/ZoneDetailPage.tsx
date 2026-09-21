import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
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
  const queryClient = useQueryClient();

  const { data: zone, isLoading, error } = useQuery({
    queryKey: ['zone', id],
    queryFn: () => api.zone(Number(id)),
    enabled: !!id,
  });
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
              <StatBlock label="Reset Mode" value={resetModeLabel(zone.reset_mode)} />
        </div>
      </div>

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

function StatBlock({ label, value }: { label: string; value: string | number }) {
  return (
    <div>
      <div className="text-xs text-ink-muted mb-1">{label}</div>
      <div className="text-sm text-ink font-mono">{value}</div>
    </div>
  );
}
