import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { CardSkeleton } from '../components/Skeleton';
import { Can } from '../components/Can';

const sexLabels = ['Male', 'Female', 'Neutral'];
const positionLabels = [
  'Standing', 'Sitting', 'Fighting', 'Sleeping', 'Resting',
  'Stunned', 'Hanging', 'Prone', 'Dead', 'Incapacitated', 'Stunned',
];

export function MobDetailPage() {
  const { vnum } = useParams<{ vnum: string }>();

  const { data: mob, isLoading, error } = useQuery({
    queryKey: ['mob', vnum],
    queryFn: () => api.mob(Number(vnum)),
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

  if (error || !mob) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/mobs" className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 text-sm">
          ← Back to Mobs
        </Link>
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-4 text-sm text-red-700 dark:text-red-300">
          Mob not found or failed to load.
          <div className="mt-1 text-red-500/70 dark:text-red-400/70 text-xs">
            {(error as Error)?.message || 'Not found'}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to="/admin/game/mobs" className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 text-sm">
        ← Back to Mobs
      </Link>

      {/* Header */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-amber-600 dark:text-amber-400">#{mob.vnum}</span>
          <h1 className="text-xl font-bold text-slate-900 dark:text-white">{mob.short_desc}</h1>
          <Can role="builder">
            <Link
              to={`/admin/game/mobs/${mob.vnum}/edit`}
              className="bg-amber-600 hover:bg-amber-500 text-white px-3 py-1 rounded text-sm ml-auto"
            >
              Edit
            </Link>
          </Can>
        </div>
        <p className="text-xs text-slate-400 mt-1 font-mono">Keywords: {mob.keywords}</p>
      </div>

      {/* Stats grid */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">Stats</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatBlock label="Level" value={mob.level} />
          <StatBlock label="AC" value={mob.ac} />
          <StatBlock label="HP" value={mob.hp} />
          <StatBlock label="EXP" value={mob.exp.toLocaleString()} />
          <StatBlock label="Gold" value={mob.gold.toLocaleString()} />
          <StatBlock label="Alignment" value={mob.alignment === 0 ? 'Neutral' : mob.alignment > 0 ? `Good (+${mob.alignment})` : `Evil (${mob.alignment})`} />
          <StatBlock label="Sex" value={sexLabels[mob.sex] || `Unknown (${mob.sex})`} />
          <StatBlock label="Position" value={positionLabels[mob.position] || `Unknown (${mob.position})`} />
          <StatBlock label="Race" value={String(mob.race)} />
        </div>
      </div>

      {/* Attributes */}
      <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">Attributes</h2>
        <div className="grid grid-cols-3 md:grid-cols-6 gap-4">
          <StatBlock label="STR" value={mob.str} />
          <StatBlock label="INT" value={mob.int} />
          <StatBlock label="WIS" value={mob.wis} />
          <StatBlock label="DEX" value={mob.dex} />
          <StatBlock label="CON" value={mob.con} />
          <StatBlock label="CHA" value={mob.cha} />
        </div>
      </div>

      {/* Flags */}
      {(mob.action_flags.length > 0 || mob.affect_flags.length > 0) && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-4">Flags</h2>
          {mob.action_flags.length > 0 && (
            <div className="mb-3">
              <span className="text-xs text-slate-400 mr-2">Action:</span>
              <div className="inline-flex flex-wrap gap-1">
                {mob.action_flags.map((flag) => (
                  <Badge key={flag} text={flag} color="red" />
                ))}
              </div>
            </div>
          )}
          {mob.affect_flags.length > 0 && (
            <div>
              <span className="text-xs text-slate-400 mr-2">Affect:</span>
              <div className="inline-flex flex-wrap gap-1">
                {mob.affect_flags.map((flag) => (
                  <Badge key={flag} text={flag} color="blue" />
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Script */}
      {mob.script_name && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Script</h2>
          <p className="text-sm text-slate-700 dark:text-slate-200 font-mono">{mob.script_name}</p>
        </div>
      )}

      {/* Long description */}
      {mob.long_desc && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Long Description</h2>
          <p className="text-sm text-slate-700 dark:text-slate-200 italic whitespace-pre-wrap">{mob.long_desc}</p>
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

function Badge({ text, color }: { text: string; color: 'red' | 'blue' }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
      color === 'red'
        ? 'bg-red-100 dark:bg-red-900/50 text-red-700 dark:text-red-300'
        : 'bg-blue-100 dark:bg-blue-900/50 text-blue-700 dark:text-blue-300'
    }`}>
      {text}
    </span>
  );
}
