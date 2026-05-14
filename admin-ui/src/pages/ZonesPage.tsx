import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Zone } from '../api/client';
import { TableSkeleton } from '../components/Skeleton';

export function ZonesPage() {
  const {
    data: zones,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['zones'],
    queryFn: api.zones,
  });

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Zones</h1>
        {zones && (
          <span className="text-sm text-slate-500 dark:text-slate-400">{zones.length} zones</span>
        )}
      </div>

      {isLoading && <TableSkeleton rows={8} cols={5} />}

      {error && (
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-4 text-sm text-red-700 dark:text-red-300">
          Failed to load zones. Is the server running on port 4350?
          <div className="mt-1 text-red-500/70 dark:text-red-400/70 text-xs">
            {(error as Error).message}
          </div>
        </div>
      )}

      {zones && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700 text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  <th className="text-left px-4 py-3">Zone #</th>
                  <th className="text-left px-4 py-3">Name</th>
                  <th className="text-right px-4 py-3">Top Room</th>
                  <th className="text-right px-4 py-3">Lifespan</th>
                  <th className="text-right px-4 py-3">Reset Mode</th>
                </tr>
              </thead>
              <tbody>
                {zones.map((zone: Zone) => (
                  <tr
                    key={zone.number}
                    className="border-b border-slate-100 dark:border-slate-700/50 hover:bg-slate-50 dark:hover:bg-slate-700/30 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/game/zones/${zone.number}`}
                        className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 font-mono"
                      >
                        {zone.number}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-slate-900 dark:text-white">{zone.name}</td>
                    <td className="px-4 py-3 text-right text-slate-600 dark:text-slate-300 font-mono">
                      {zone.top_room}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-600 dark:text-slate-300 font-mono">
                      {zone.lifespan}
                    </td>
                    <td className="px-4 py-3 text-right text-slate-600 dark:text-slate-300">
                      {resetModeLabel(zone.reset_mode)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {zones && zones.length === 0 && (
        <div className="text-center text-slate-400 dark:text-slate-500 py-8">
          No zones loaded. Check server configuration.
        </div>
      )}
    </div>
  );
}

function resetModeLabel(mode: number): string {
  switch (mode) {
    case 0:
      return 'Never';
    case 1:
      return 'When empty';
    case 2:
      return 'Always';
    case 3:
      return 'Force reset';
    default:
      return `Mode ${mode}`;
  }
}
