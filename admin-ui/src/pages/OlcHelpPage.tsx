import { useQueries } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { olcApi, type OlcSchema } from '../api/olc';
import { Skeleton } from '../components/Skeleton';

const kinds = ['room', 'mob', 'obj', 'shop', 'zone'];

function kindLabel(kind: string): string {
  if (kind === 'obj') return 'Objects';
  return kind.charAt(0).toUpperCase() + kind.slice(1) + 's';
}

export function OlcHelpPage() {
  const schemaQueries = useQueries({
    queries: kinds.map((kind) => ({ queryKey: ['olc-schema', kind], queryFn: () => olcApi.schema(kind), staleTime: 30 * 60 * 1000, retry: false })),
  });
  const loading = schemaQueries.some((query) => query.isLoading);
  const error = schemaQueries.find((query) => query.error)?.error;
  const schemas = schemaQueries.map((query) => query.data).filter((schema): schema is OlcSchema => Boolean(schema));
  const actions = schemas.flatMap((schema) => schema.actions.map((action) => ({ ...action, kind: schema.kind })));

  return (
    <div className="mx-auto max-w-5xl space-y-8">
      <div>
        <h1 className="text-3xl text-ink">Builder guide</h1>
        <p className="mt-3 max-w-2xl text-base text-ink-muted">Use this guide to find a room, edit it, and save the zone file. Your character's assigned OLC zone controls where you may build.</p>
      </div>
      <nav aria-label="On this page" className="flex flex-wrap gap-x-5 gap-y-2 border-y border-rule py-3 text-sm">
        <a href="#first-room" className="text-accent hover:underline">Edit a room</a>
        <a href="#save-work" className="text-accent hover:underline">Commit and save</a>
        <a href="#access" className="text-accent hover:underline">Access and conflicts</a>
        <a href="#capabilities" className="text-accent hover:underline">Available actions</a>
      </nav>
      <section id="first-room" className="grid gap-6 lg:grid-cols-[14rem_1fr]">
        <div><h2 className="text-xl">Edit a room</h2><p className="mt-2 text-sm text-ink-muted">Start in your assigned zone.</p></div>
        <ol className="divide-y divide-rule border-t border-b border-rule">
          <li className="py-4"><strong>Find your zone.</strong> Open <Link to="/admin/game/zones" className="text-accent underline underline-offset-2">Zones & rooms</Link> and select its number. The zone page leads to its room range and creation controls.</li>
          <li className="py-4"><strong>Open the room.</strong> Follow its VNUM to the room detail page, then choose Edit. For a new room, use the zone page's offered free VNUM.</li>
          <li className="py-4"><strong>Work in the draft.</strong> Change the room's text, flags, sector, exits, or extra descriptions. The editor holds a temporary claim while you work. Review the fields before committing.</li>
        </ol>
      </section>
      <section id="save-work" className="grid gap-6 lg:grid-cols-[14rem_1fr]">
        <div><h2 className="text-xl">Commit and save</h2><p className="mt-2 text-sm text-ink-muted">These are two separate actions.</p></div>
        <div className="border-y border-rule py-4 space-y-3">
          <p><strong>Commit draft</strong> applies the edited room to the live world and releases its claim. <strong>Discard</strong> leaves the live room unchanged.</p>
          <p><strong>Save zone file</strong> writes committed changes to the canonical world file. A zone marked dirty still needs this step. You can save from the editor after committing or from <Link to="/admin/workshop" className="text-accent underline underline-offset-2">Workshop & saves</Link>.</p>
          <p className="text-sm text-ink-muted">Check the pending zone saves list before you finish. A successful commit alone does not clear that list.</p>
        </div>
      </section>
      <section id="access" className="grid gap-6 lg:grid-cols-[14rem_1fr]">
        <h2 className="text-xl">Access and conflicts</h2>
        <div className="border-y border-rule py-4 space-y-3">
          <p>Builders can edit resources in their assigned OLC zone. If a room is outside that zone, ask staff to check the zone assignment on your character.</p>
          <p>A room may already be claimed in the web editor or in-game OLC. The editor shows the holder when a claim blocks you. Wait for that edit to finish before trying again; do not start a second edit of the same resource.</p>
          <p>If a commit or save is refused, leave the error visible and share the resource VNUM and message with staff. The <Link to="/admin/workshop" className="text-accent underline underline-offset-2">workshop</Link> shows held claims and pending saves.</p>
        </div>
      </section>
      <section id="capabilities" className="space-y-3">
        <div><h2 className="text-xl">Available actions</h2><p className="mt-2 text-sm text-ink-muted">The server supplies these permissions for your current account. An unavailable action may require a different level or role.</p></div>
      {loading && <div className="space-y-3"><Skeleton className="h-16 w-full" /><Skeleton className="h-16 w-full" /></div>}
      {error && <div className="border border-accent bg-paper-deep px-4 py-3 text-sm text-accent" role="alert">Capability data could not be loaded: {(error as Error).message}</div>}
      {!loading && !error && (
        <div className="space-y-3">
          {actions.map((action) => (
            <section key={action.kind + '-' + action.key} className={'border border-rule bg-paper-deep p-4 ' + (action.allowed ? '' : 'opacity-60')}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div><p className="font-mono text-xs uppercase tracking-wider text-ink-muted">{kindLabel(action.kind)}</p><h2 className="mt-1 text-lg text-ink">{action.label}</h2></div>
                <span className={'border px-2 py-1 font-mono text-[10px] uppercase tracking-wider ' + (action.allowed ? 'border-online text-online' : 'border-rule text-ink-muted')}>{action.allowed ? 'available' : 'requires ' + action.requiredLabel + ' · level ' + action.requiredLevel}</span>
              </div>
              {!action.allowed && <p className="mt-3 text-sm text-ink-muted">This action is shown for orientation but is unavailable for the current server capability.</p>}
            </section>
          ))}
          {actions.length === 0 && <p className="border border-rule bg-paper-deep p-4 text-sm text-ink-muted">No additional actions are exposed by the server schema.</p>}
        </div>
      )}
      </section>
    </div>
  );
}
