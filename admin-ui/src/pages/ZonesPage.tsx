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
        <h1 className="text-2xl font-bold text-ink">Zones</h1>
        {zones && (
          <span className="text-sm text-ink-muted">{zones.length} zones</span>
        )}
      </div>

      {isLoading && <TableSkeleton rows={8} cols={5} />}

      {error && (
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Failed to load zones. Is the server running on port 4350?
          <div className="mt-1 text-accent text-xs">
            {(error as Error).message}
          </div>
        </div>
      )}

      {zones && (
        <div className="bg-paper-deep rounded-none border border-rule overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-rule text-xs text-ink-muted uppercase tracking-wider">
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
                    className="border-b border-rule hover:bg-paper-deep transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/game/zones/${zone.number}`}
                        className="text-accent hover:text-accent font-mono"
                      >
                        {zone.number}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-ink">{zone.name}</td>
                    <td className="px-4 py-3 text-right text-ink-muted font-mono">
                      {zone.top_room}
                    </td>
                    <td className="px-4 py-3 text-right text-ink-muted font-mono">
                      {zone.lifespan}
                    </td>
                    <td className="px-4 py-3 text-right text-ink-muted">
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
        <div className="text-center text-ink-muted py-8">
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
