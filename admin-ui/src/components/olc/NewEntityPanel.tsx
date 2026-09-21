import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api } from '../../api/client';
import { olcApi, type OlcSchema } from '../../api/olc';

type NewKind = 'mob' | 'obj' | 'shop';

const kindLabels: Record<NewKind, string> = { mob: 'mob', obj: 'object', shop: 'shop' };
const actionKeys: Record<NewKind, string> = { mob: 'new_mob', obj: 'new_obj', shop: 'new_shop' };
const editorPaths: Record<NewKind, string> = { mob: '/admin/game/mobs/', obj: '/admin/game/objects/', shop: '/admin/game/shops/' };

export function NewEntityPanel({ kind }: { kind: NewKind }) {
  const [zone, setZone] = useState<number | null>(null);
  const zonesQuery = useQuery({ queryKey: ['zones'], queryFn: api.zones, staleTime: 30 * 60 * 1000 });
  const schemaQuery = useQuery({ queryKey: ['olc-schema', kind], queryFn: () => olcApi.schema(kind), staleTime: 30 * 60 * 1000, retry: false });
  const mapQuery = useQuery({ queryKey: ['olc-vnum-map', zone], queryFn: () => olcApi.vnumMap(zone as number), enabled: zone !== null, retry: false });
  const entry = mapQuery.data?.kinds.find((candidate) => candidate.kind === kind);
  const action = (schemaQuery.data as OlcSchema | undefined)?.actions.find((candidate) => candidate.key === actionKeys[kind]);
  const label = kindLabels[kind];

  return (
    <section className="border border-rule bg-paper-deep p-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div><h2 className="text-lg text-ink">New {label}</h2><p className="mt-1 text-sm text-ink-muted">Choose a zone to get the next advisory free VNUM. The picker never claims it.</p></div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
          <label className="block"><span className="mb-1 block text-[11px] font-semibold uppercase tracking-wider text-ink-muted">Zone</span><select value={zone === null ? '' : String(zone)} onChange={(event) => setZone(event.currentTarget.value ? Number(event.currentTarget.value) : null)} className="border border-rule bg-paper px-3 py-2 text-sm text-ink"><option value="">Choose a zone</option>{(zonesQuery.data || []).map((candidate) => <option key={candidate.number} value={candidate.number}>{candidate.number} — {candidate.name}</option>)}</select></label>
          {entry && entry.suggested > 0 && action?.allowed ? <Link to={editorPaths[kind] + entry.suggested + '/edit'} className="border border-accent bg-accent px-3 py-2 text-center text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">{action.label} · #{entry.suggested}</Link> : <span className="border border-rule px-3 py-2 text-xs text-ink-muted">{action && !action.allowed ? 'Requires ' + action.requiredLabel + ' (level ' + action.requiredLevel + ')' : entry ? 'No free VNUM' : 'Select a zone'}</span>}
        </div>
      </div>
      {entry && <p className="mt-3 text-xs text-ink-muted">Free range: {entry.free.length ? entry.free.map((range) => range.start === range.end ? String(range.start) : range.start + '–' + range.end).join(', ') : 'none'} · suggested #{entry.suggested || '—'}</p>}
      {(zonesQuery.error || mapQuery.error || schemaQuery.error) && <p className="mt-3 text-xs text-accent" role="alert">The free-VNUM picker could not load its server data.</p>}
    </section>
  );
}
