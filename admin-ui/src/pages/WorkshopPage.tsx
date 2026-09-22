import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { olcApi } from '../api/olc';
import { Skeleton } from '../components/Skeleton';

function formatIdle(seconds: number): string {
  if (seconds < 60) return seconds + 's';
  return Math.floor(seconds / 60) + 'm ' + (seconds % 60) + 's';
}

export function WorkshopPage() {
  const queryClient = useQueryClient();
  const [now, setNow] = useState(() => Date.now());
  const heldQuery = useQuery({ queryKey: ['olc-held'], queryFn: olcApi.held, refetchInterval: 30_000, refetchIntervalInBackground: true, retry: false });
  const pendingQuery = useQuery({ queryKey: ['olc-pending'], queryFn: olcApi.pending, refetchInterval: 30_000, refetchIntervalInBackground: true, retry: false });
  const saveMutation = useMutation({
    mutationFn: (zone: number) => olcApi.saveZone(zone),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['olc-pending'] });
      queryClient.invalidateQueries({ queryKey: ['zones'] });
    },
  });

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const pendingZones = useMemo(() => Array.from(new Set((pendingQuery.data || []).map((entry) => entry.zone))).sort((a, b) => a - b), [pendingQuery.data]);
  const loading = heldQuery.isLoading || pendingQuery.isLoading;
  const error = heldQuery.error || pendingQuery.error;

  if (loading) {
    return <div className="space-y-5"><Skeleton className="h-8 w-48" /><Skeleton className="h-44 w-full" /><Skeleton className="h-44 w-full" /></div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <p className="font-mono text-xs uppercase tracking-widest text-accent">Builder workshop</p>
          <h1 className="mt-1 text-2xl text-ink">Claims and saves</h1>
          <p className="mt-2 max-w-2xl text-sm text-ink-muted">A live view of held OLC leases and committed zone changes waiting for their canonical file save.</p>
        </div>
        <Link to="/admin/workshop/help" className="border border-rule bg-paper-deep px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper">Capabilities and help</Link>
      </div>

      {error && <div className="border border-accent bg-paper-deep px-4 py-3 text-sm text-accent" role="alert">Workshop data could not be loaded: {(error as Error).message}</div>}

      <section className="grid gap-3 md:grid-cols-2">
        <Link to="/admin/workshop/scripts" className="border border-rule bg-paper-deep p-4 hover:border-accent">
          <h2 className="text-lg text-ink">Lua scripts</h2>
          <p className="mt-2 text-sm text-ink-muted">Browse references, parse-check, and safely publish live game scripts.</p>
        </Link>
        <Link to="/admin/workshop/text" className="border border-rule bg-paper-deep p-4 hover:border-accent">
          <h2 className="text-lg text-ink">Server text</h2>
          <p className="mt-2 text-sm text-ink-muted">Edit news, MOTD, help, credits, and the other canonical tedit files.</p>
        </Link>
      </section>

      <section className="border border-rule bg-paper-deep p-4">
        <div className="flex items-baseline justify-between gap-3">
          <div>
            <h2 className="text-lg text-ink">Held claims</h2>
            <p className="mt-1 text-sm text-ink-muted">Claims are server-owned leases; idle time is measured from the reported claim timestamp.</p>
          </div>
          <span className="font-mono text-xs text-ink-muted">{(heldQuery.data || []).length} held</span>
        </div>
        {(heldQuery.data || []).length === 0 ? (
          <p className="mt-4 border-t border-rule pt-4 text-sm text-ink-muted">No active claims.</p>
        ) : (
          <div className="mt-4 overflow-x-auto border-t border-rule">
            <table className="w-full min-w-[640px] text-left text-sm">
              <thead className="border-b border-rule text-[11px] uppercase tracking-wider text-ink-muted">
                <tr><th className="px-2 py-3">Resource</th><th className="px-2 py-3">Holder</th><th className="px-2 py-3">Frontend</th><th className="px-2 py-3 text-right">Idle</th></tr>
              </thead>
              <tbody>
                {(heldQuery.data || []).map((entry) => {
                  const idle = Math.max(0, Math.floor((now - Date.parse(entry.claimedAt)) / 1000));
                  return <tr key={entry.kind + '-' + entry.number} className="border-b border-rule last:border-b-0"><td className="px-2 py-3 font-mono text-accent">{entry.kind} #{entry.number}</td><td className="px-2 py-3 text-ink">{entry.ownerDisplayName}</td><td className="px-2 py-3 text-ink-muted">{entry.ownerFrontend}</td><td className="px-2 py-3 text-right font-mono text-ink-muted">{formatIdle(idle)}</td></tr>;
                })}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="border border-rule bg-paper-deep p-4">
        <div className="flex items-baseline justify-between gap-3">
          <div>
            <h2 className="text-lg text-ink">Pending zone saves</h2>
            <p className="mt-1 text-sm text-ink-muted">These zones have committed in-memory changes that still need the canonical zone-file save.</p>
          </div>
          <span className="font-mono text-xs text-ink-muted">{pendingZones.length} zones</span>
        </div>
        {pendingZones.length === 0 ? (
          <p className="mt-4 border-t border-rule pt-4 text-sm text-ink-muted">No pending zone saves.</p>
        ) : (
          <div className="mt-4 space-y-2 border-t border-rule pt-4">
            {pendingZones.map((zone) => (
              <div key={zone} className="flex flex-wrap items-center justify-between gap-3 border border-rule bg-paper px-3 py-3">
                <div><Link to={'/admin/game/zones/' + zone} className="font-mono text-accent hover:text-accent-deep">Zone #{zone}</Link><p className="mt-1 text-xs text-ink-muted">{(pendingQuery.data || []).filter((entry) => entry.zone === zone).map((entry) => entry.kind).join(', ')} dirty</p></div>
                <button type="button" disabled={saveMutation.isPending} onClick={() => saveMutation.mutate(zone)} className="border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:cursor-not-allowed disabled:opacity-50">{saveMutation.isPending && saveMutation.variables === zone ? 'Saving…' : 'Save zone file'}</button>
              </div>
            ))}
          </div>
        )}
        {saveMutation.error && <p className="mt-3 text-sm text-accent" role="alert">Save failed: {(saveMutation.error as Error).message}</p>}
      </section>
    </div>
  );
}
