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

  const [lifespan, setLifespan] = useState(0);
  const [resetMode, setResetMode] = useState(0);
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [resetting, setResetting] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [resetResult, setResetResult] = useState('');

  const startEditing = () => {
    if (!zone) return;
    setLifespan(zone.lifespan);
    setResetMode(zone.reset_mode);
    setEditing(true);
    setSaveError('');
  };

  const handleSave = async () => {
    if (!id) return;
    setSaving(true);
    setSaveError('');
    try {
      await api.updateZone(Number(id), {
        lifespan,
        reset_mode: resetMode,
      });
      queryClient.invalidateQueries({ queryKey: ['zone', id] });
      queryClient.invalidateQueries({ queryKey: ['zones'] });
      setEditing(false);
    } catch (err) {
      setSaveError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

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
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300">
            Properties
          </h2>
          {!editing && (
            <button
              onClick={startEditing}
              className="text-xs text-amber-600 dark:text-amber-400 hover:text-amber-500 px-2 py-1 border border-amber-600/50 rounded"
            >
              Edit
            </button>
          )}
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatBlock label="Zone Number" value={zone.number} />
          <StatBlock label="Top Room" value={zone.top_room} />
          {editing ? (
            <>
              <div>
                <label className="block text-xs text-slate-400 mb-1">Lifespan (minutes)</label>
                <input
                  type="number"
                  value={lifespan}
                  onChange={(e) => setLifespan(Number(e.target.value))}
                  min={0}
                  className="w-full bg-slate-100 dark:bg-slate-900 border border-slate-300 dark:border-slate-600 rounded px-2 py-1 text-sm text-slate-900 dark:text-white focus:outline-none focus:border-amber-500"
                />
              </div>
              <div>
                <label className="block text-xs text-slate-400 mb-1">Reset Mode</label>
                <select
                  value={resetMode}
                  onChange={(e) => setResetMode(Number(e.target.value))}
                  className="w-full bg-slate-100 dark:bg-slate-900 border border-slate-300 dark:border-slate-600 rounded px-2 py-1 text-sm text-slate-900 dark:text-white focus:outline-none focus:border-amber-500"
                >
                  <option value={0}>0: Never</option>
                  <option value={1}>1: If Empty</option>
                  <option value={2}>2: Always</option>
                  <option value={3}>3: Force reset</option>
                </select>
              </div>
            </>
          ) : (
            <>
              <StatBlock label="Lifespan" value={`${zone.lifespan} min`} />
              <StatBlock label="Reset Mode" value={resetModeLabel(zone.reset_mode)} />
            </>
          )}
        </div>

        {saveError && (
          <div className="mt-3 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-3 text-sm text-red-700 dark:text-red-300">
            {saveError}
          </div>
        )}

        {editing && (
          <div className="flex gap-3 mt-4">
            <button
              onClick={handleSave}
              disabled={saving}
              className="bg-amber-600 hover:bg-amber-500 disabled:opacity-50 text-white px-3 py-1.5 rounded text-xs font-medium"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button
              onClick={() => setEditing(false)}
              className="bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600 text-slate-700 dark:text-white px-3 py-1.5 rounded text-xs"
            >
              Cancel
            </button>
          </div>
        )}
      </div>

      {/* Reset Zone */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-3">
          Zone Reset
        </h2>
        <button
          onClick={handleReset}
          disabled={resetting}
          className="bg-red-700 hover:bg-red-600 disabled:opacity-50 text-white px-4 py-2 rounded text-sm font-medium"
        >
          {resetting ? 'Resetting...' : 'Reset Zone'}
        </button>
        {resetResult && (
          <div className="mt-2 text-sm text-slate-600 dark:text-slate-400">{resetResult}</div>
        )}
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
