import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { CardSkeleton } from '../components/Skeleton';
import { itemTypeLabel } from '../lib/gameLabels';

function flagBits(value: number): string[] {
  const bits: string[] = [];
  for (let i = 0; i < 32; i++) {
    if (value & (1 << i)) bits.push(String(i));
  }
  return bits;
}

export function ObjectDetailPage() {
  const { vnum } = useParams<{ vnum: string }>();

  const { data: obj, isLoading, error } = useQuery({
    queryKey: ['object', vnum],
    queryFn: () => api.object(Number(vnum)),
    enabled: !!vnum,
  });

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="h-4 w-24 animate-pulse bg-paper-deep rounded" />
        <CardSkeleton />
        <CardSkeleton />
      </div>
    );
  }

  if (error || !obj) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/objects" className="text-accent hover:text-accent text-sm">
          ← Back to Objects
        </Link>
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Object not found or failed to load.
          <div className="mt-1 text-accent text-xs">
            {(error as Error)?.message || 'Not found'}
          </div>
        </div>
      </div>
    );
  }

  const extraBits = obj.extra_flags.flatMap((v, i) => flagBits(v).map((b) => `${i}:${b}`));
  const wearBits = obj.wear_flags.flatMap((v, i) => flagBits(v).map((b) => `${i}:${b}`));

  return (
    <div className="space-y-6">
      <Link to="/admin/game/objects" className="text-accent hover:text-accent text-sm">
        ← Back to Objects
      </Link>

      {/* Header */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-accent">#{obj.vnum}</span>
          <h1 className="text-xl font-bold text-ink">{obj.short_desc}</h1>
        </div>
        <p className="text-xs text-ink-muted mt-1 font-mono">Keywords: {obj.keywords}</p>
        <Link to={`/admin/game/objects/${obj.vnum}/edit`} className="mt-4 inline-block border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">Edit object</Link>
      </div>

      {/* Stats grid */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <h2 className="text-sm font-medium text-ink-muted mb-4">Properties</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatBlock label="Type" value={itemTypeLabel(obj.type_flag)} />
          <StatBlock label="Weight" value={`${obj.weight} lbs`} />
          <StatBlock label="Cost" value={`${obj.cost.toLocaleString()} gold`} />
          <StatBlock label="Value[0]" value={obj.values[0]} />
          <StatBlock label="Value[1]" value={obj.values[1]} />
          <StatBlock label="Value[2]" value={obj.values[2]} />
          <StatBlock label="Value[3]" value={obj.values[3]} />
        </div>
      </div>

      {/* Flags */}
      {(extraBits.length > 0 || wearBits.length > 0) && (
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-4">Flags</h2>
          {extraBits.length > 0 && (
            <div className="mb-3">
              <span className="text-xs text-ink-muted mr-2">Extra Flags ({obj.extra_flags.join(', ')}):</span>
              <div className="inline-flex flex-wrap gap-1">
                {extraBits.map((b) => <Badge key={`e-${b}`} text={b} color="amber" />)}
              </div>
            </div>
          )}
          {wearBits.length > 0 && (
            <div>
              <span className="text-xs text-ink-muted mr-2">Wear Flags ({obj.wear_flags.join(', ')}):</span>
              <div className="inline-flex flex-wrap gap-1">
                {wearBits.map((b) => <Badge key={`w-${b}`} text={b} color="blue" />)}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Script */}
      {obj.script_name && (
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-2">Script</h2>
          <p className="text-sm text-ink-muted font-mono">{obj.script_name}</p>
        </div>
      )}

      {/* Long description */}
      {obj.long_desc && (
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-2">Long Description</h2>
          <p className="text-sm text-ink-muted italic whitespace-pre-wrap">{obj.long_desc}</p>
        </div>
      )}
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

function Badge({ text, color }: { text: string; color: 'amber' | 'blue' }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
      color === 'amber'
        ? 'bg-paper-deep text-accent'
        : 'bg-paper-deep text-ink-muted'
    }`}>
      {text}
    </span>
  );
}
