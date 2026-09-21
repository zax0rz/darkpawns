import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Zone } from '../api/client';
import { TableSkeleton } from '../components/Skeleton';
import { olcApi } from '../api/olc';
import { useState } from 'react';

export function ZonesPage() {
  const queryClient = useQueryClient();
  const [newZone, setNewZone] = useState('');
  const [createMessage, setCreateMessage] = useState('');
  const {
    data: zones,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['zones'],
    queryFn: api.zones,
  });
  const schemaQuery = useQuery({ queryKey: ['olc-schema', 'zone'], queryFn: () => olcApi.schema('zone'), staleTime: 30 * 60 * 1000, retry: false });
  const createMutation = useMutation({
    mutationFn: () => olcApi.createZone(Number(newZone)),
    onSuccess: (created) => { setCreateMessage(`Created zone #${created.zone}.`); setNewZone(''); queryClient.invalidateQueries({ queryKey: ['zones'] }); },
    onError: (err) => setCreateMessage(`Could not create zone: ${(err as Error).message}`),
  });
  const createAction = schemaQuery.data?.actions.find((action) => action.key === 'create_zone');
  const createAllowed = Boolean(createAction?.allowed);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-ink">Zones</h1>
        {zones && (
          <span className="text-sm text-ink-muted">{zones.length} zones</span>
        )}
      </div>

      <section className="border border-rule bg-paper-deep p-4">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div>
            <h2 className="text-lg text-ink">New zone</h2>
            <p className="mt-1 text-sm text-ink-muted">Creates the C-compatible world files and adds the zone to the live world.</p>
          </div>
          <form className="flex flex-col gap-2 sm:flex-row" onSubmit={(event) => { event.preventDefault(); setCreateMessage(''); createMutation.mutate(); }}>
            <label className="sr-only" htmlFor="new-zone-number">Zone number</label>
            <input id="new-zone-number" type="number" min="0" max="326" value={newZone} onChange={(event) => setNewZone(event.currentTarget.value)} placeholder="Zone number" disabled={!createAllowed || createMutation.isPending} className="border border-rule bg-paper px-3 py-2 text-sm text-ink placeholder-ink-muted disabled:cursor-not-allowed disabled:opacity-50" />
            <button type="submit" disabled={!createAllowed || !newZone || createMutation.isPending} className="border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:cursor-not-allowed disabled:opacity-50">{createMutation.isPending ? 'Creating…' : 'Create zone'}</button>
          </form>
        </div>
        {!createAllowed && createAction && <p className="mt-3 text-xs text-accent">Requires {createAction.requiredLabel} (level {createAction.requiredLevel}).</p>}
        {createMessage && <p className="mt-3 text-sm text-ink" role="status">{createMessage}</p>}
      </section>

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
