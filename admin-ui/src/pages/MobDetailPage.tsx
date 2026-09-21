import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';
import { CardSkeleton } from '../components/Skeleton';
import { positionLabel, sexLabel, raceLabel } from '../lib/gameLabels';


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
        <div className="h-4 w-24 animate-pulse bg-paper-deep rounded" />
        <CardSkeleton />
        <CardSkeleton />
      </div>
    );
  }

  if (error || !mob) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/mobs" className="text-accent hover:text-accent text-sm">
          ← Back to Mobs
        </Link>
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Mob not found or failed to load.
          <div className="mt-1 text-accent text-xs">
            {(error as Error)?.message || 'Not found'}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to="/admin/game/mobs" className="text-accent hover:text-accent text-sm">
        ← Back to Mobs
      </Link>

      {/* Header */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-accent">#{mob.vnum}</span>
          <h1 className="text-xl font-bold text-ink">{mob.short_desc}</h1>
        </div>
        <p className="text-xs text-ink-muted mt-1 font-mono">Keywords: {mob.keywords}</p>
        <Link to={`/admin/game/mobs/${mob.vnum}/edit`} className="mt-4 inline-block border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">Edit mob</Link>
      </div>

      {/* Stats grid */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <h2 className="text-sm font-medium text-ink-muted mb-4">Stats</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <StatBlock label="Level" value={mob.level} />
          <StatBlock label="AC" value={mob.ac} />
          <StatBlock label="HP" value={mob.hp} />
          <StatBlock label="EXP" value={mob.exp.toLocaleString()} />
          <StatBlock label="Gold" value={mob.gold.toLocaleString()} />
          <StatBlock label="Alignment" value={mob.alignment === 0 ? 'Neutral' : mob.alignment > 0 ? `Good (+${mob.alignment})` : `Evil (${mob.alignment})`} />
          <StatBlock label="Sex" value={sexLabel(mob.sex)} />
          <StatBlock label="Position" value={positionLabel(mob.position)} />
          <StatBlock label="Race" value={raceLabel(mob.race)} />
        </div>
      </div>

      {/* Attributes */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <h2 className="text-sm font-medium text-ink-muted mb-4">Attributes</h2>
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
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-4">Flags</h2>
          {mob.action_flags.length > 0 && (
            <div className="mb-3">
              <span className="text-xs text-ink-muted mr-2">Action:</span>
              <div className="inline-flex flex-wrap gap-1">
                {mob.action_flags.map((flag) => (
                  <Badge key={flag} text={flag} color="red" />
                ))}
              </div>
            </div>
          )}
          {mob.affect_flags.length > 0 && (
            <div>
              <span className="text-xs text-ink-muted mr-2">Affect:</span>
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
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-2">Script</h2>
          <p className="text-sm text-ink-muted font-mono">{mob.script_name}</p>
        </div>
      )}

      {/* Long description */}
      {mob.long_desc && (
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-2">Long Description</h2>
          <p className="text-sm text-ink-muted italic whitespace-pre-wrap">{mob.long_desc}</p>
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

function Badge({ text, color }: { text: string; color: 'red' | 'blue' }) {
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${
      color === 'red'
        ? 'bg-paper-deep text-accent'
        : 'bg-paper-deep text-ink-muted'
    }`}>
      {text}
    </span>
  );
}
