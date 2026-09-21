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
    <div className="space-y-6">
      <div>
        <Link to="/admin/workshop" className="text-sm text-accent hover:text-accent-deep">← Back to workshop</Link>
        <p className="mt-5 font-mono text-xs uppercase tracking-widest text-accent">Server capabilities</p>
        <h1 className="mt-1 text-2xl text-ink">OLC help</h1>
        <p className="mt-2 max-w-2xl text-sm text-ink-muted">Action labels and access requirements below are supplied by the server schema. The browser does not maintain a separate level ladder.</p>
      </div>
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
    </div>
  );
}
