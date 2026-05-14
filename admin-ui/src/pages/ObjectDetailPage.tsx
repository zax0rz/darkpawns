import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { CardSkeleton } from '../components/Skeleton';
import { Can } from '../components/Can';

const itemTypeLabels: Record<number, string> = {
  0: 'Light', 1: 'Scroll', 2: 'Wand', 3: 'Staff', 4: 'Weapon',
  5: 'Fire Weapon', 6: 'Missile', 7: 'Treasure', 8: 'Armor', 9: 'Potion',
  10: 'Worn', 11: 'Furniture', 12: 'Trash', 13: 'Container', 14: 'Note',
  15: 'Drink Container', 16: 'Key', 17: 'Food', 18: 'Money', 19: 'Pen',
  20: 'Boat', 21: 'Fountain', 22: 'Campfire', 23: 'Corpse',
};

function itemTypeLabel(flag: number): string {
  return itemTypeLabels[flag] || `Type ${flag}`;
}

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
        <div className="h-4 w-24 animate-pulse bg-slate-200 dark:bg-slate-700 rounded" />
        <CardSkeleton />
        <CardSkeleton />
      </div>
    );
  }

  if (error || !obj) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/objects" className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 text-sm">
          ← Back to Objects
        </Link>
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-4 text-sm text-red-700 dark:text-red-300">
          Object not found or failed to load.
          <div className="mt-1 text-red-500/70 dark:text-red-400/70 text-xs">
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
      <Link to="/admin/game/objects" className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 text-sm">
        ← Back to Objects
      </Link>

      {/* Header */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-amber-600 dark:text-amber-400">#{obj.vnum}</span>
          <h1 className="text-xl font-bold text-slate-900 dark:text-white">{obj.short_desc}</h1>
          <Can role="builder">
            <Link
              to={`/admin/game/objects/${obj.vnum}/edit`}
              className="bg-amber-600 hover:bg-amber-500 text-white px-3 py-1 rounded text-sm ml-auto"
            >
              Edit
            </Link>
          </Can>
        </div>
        <p className="text-xs text-slate-400 mt-1 font-mono">Keywords: {obj.keywords}</p>
      </div>

      {/* Stats grid */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">Properties</h2>
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
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">Flags</h2>
          {extraBits.length > 0 && (
            <div className="mb-3">
              <span className="text-xs text-slate-400 mr-2">Extra Flags ({obj.extra_flags.join(', ')}):</span>
              <div className="inline-flex flex-wrap gap-1">
                {extraBits.map((b) => <Badge key={`e-${b}`} text={b} color="amber" />)}
              </div>
            </div>
          )}
          {wearBits.length > 0 && (
            <div>
              <span className="text-xs text-slate-400 mr-2">Wear Flags ({obj.wear_flags.join(', ')}):</span>
              <div className="inline-flex flex-wrap gap-1">
                {wearBits.map((b) => <Badge key={`w-${b}`} text={b} color="blue" />)}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Script */}
      {obj.script_name && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Script</h2>
          <p className="text-sm text-slate-700 dark:text-slate-200 font-mono">{obj.script_name}</p>
        </div>
      )}

      {/* Long description */}
      {obj.long_desc && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Long Description</h2>
          <p className="text-sm text-slate-700 dark:text-slate-200 italic whitespace-pre-wrap">{obj.long_desc}</p>
        </div>
      )}
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

function Badge({ text, color }: { text: string; color: 'amber' | 'blue' }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
      color === 'amber'
        ? 'bg-amber-100 dark:bg-amber-900/50 text-amber-700 dark:text-amber-300'
        : 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
    }`}>
      {text}
    </span>
  );
}
