import { useCallback, useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { olcApi, type OlcEntityDraft, type OlcPatchOperation, type OlcSchema, type OlcSchemaField, type OlcZone } from '../api/olc';
import { ClaimSaveFrame } from '../components/olc/ClaimSaveFrame';
import { ServerProposal } from '../components/olc/ServerProposal';
import { ZoneCommandTimeline } from '../components/olc/ZoneCommandTimeline';
import { Skeleton } from '../components/Skeleton';

const draftKey = (zone: number, room: number | null) => ['olc-entity-draft', 'zone', zone, room ?? 0];

function conflictSummary(error: unknown): string {
  if (!(error instanceof ApiError)) return (error as Error)?.message || 'The zone could not be claimed.';
  if (typeof error.payload !== 'object' || error.payload === null) return error.message;
  const payload = error.payload as Record<string, unknown>;
  const holder = typeof payload.holder === 'string' ? payload.holder : 'another editor';
  const frontend = typeof payload.frontend === 'string' ? payload.frontend : 'unknown frontend';
  const idle = typeof payload.idle === 'string' ? payload.idle : 'unknown idle time';
  return error.message + ' Holder: ' + holder + '. Frontend: ' + frontend + '. Idle: ' + idle + '.';
}

function label(field: OlcSchemaField, changed: boolean) {
  return (
    <div className="mb-2 flex items-baseline justify-between gap-3">
      <label htmlFor={'zone-' + field.key} className="text-sm font-semibold text-ink">{field.label}</label>
      {changed && <span className="font-mono text-[10px] uppercase tracking-wider text-accent">changed</span>}
    </div>
  );
}

function ZoneSettings({ schema, zone, dirty, disabled, onOperation }: { schema: OlcSchema; zone: OlcZone; dirty: string[]; disabled: boolean; onOperation: (operations: OlcPatchOperation[]) => void }) {
  const operation: Record<string, string> = { name: 'set_name', lifespan: 'set_lifespan', reset_mode: 'set_reset_mode', top_room: 'set_top_room' };
  return (
    <section className="border-t border-rule pt-5">
      <h2 className="mb-4 text-lg text-ink">Zone settings</h2>
      <div className="grid gap-5 lg:grid-cols-2">
        {schema.fields.map((field) => {
          const current: string | number = field.key === 'name' ? zone.name : field.key === 'lifespan' ? zone.lifespan : field.key === 'reset_mode' ? zone.resetMode : zone.topRoom;
          const op = operation[field.key];
          if (field.control === 'select') {
            return (
              <div key={field.key}>
                {label(field, dirty.includes(field.key))}
                <select id={'zone-' + field.key} value={String(current)} disabled={disabled} onChange={(event) => onOperation([{ kind: op, value: Number(event.currentTarget.value) }])} className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink">
                  {(field.options || []).map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                </select>
              </div>
            );
          }
          return (
            <div key={field.key}>
              {label(field, dirty.includes(field.key))}
              <ServerProposal value={current} identity={'zone-' + field.key} disabled={disabled} min={field.bounds?.min} max={field.bounds?.max} onCommit={(raw) => onOperation([{ kind: op, ...(field.control === 'text' ? { text: raw } : { value: Number(raw) }) }])} />
              {field.bounds && <p className="mt-1 font-mono text-[11px] text-ink-muted">{field.bounds.min}–{field.bounds.max}</p>}
            </div>
          );
        })}
      </div>
    </section>
  );
}

export function ZoneEditorPage() {
  const { zone: rawZone } = useParams<{ zone: string }>();
  const [searchParams] = useSearchParams();
  const zoneNumber = Number(rawZone);
  const rawRoom = searchParams.get('room');
  const parsedRoom = rawRoom === null ? null : Number(rawRoom);
  const anchorRoom = parsedRoom !== null && Number.isInteger(parsedRoom) ? parsedRoom : null;
  const invalidAnchor = rawRoom !== null && anchorRoom === null;
  const editVnum = anchorRoom ?? zoneNumber;
  const backPath = anchorRoom === null ? '/admin/game/zones' : '/admin/game/rooms/' + anchorRoom;
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const openedRef = useRef(false);
  const [claimed, setClaimed] = useState(false);
  const [committedDraft, setCommittedDraft] = useState<OlcEntityDraft | null>(null);
  const currentDraftKey = draftKey(zoneNumber, anchorRoom);

  const schemaQuery = useQuery({ queryKey: ['olc-schema', 'zone'], queryFn: () => olcApi.schema('zone'), staleTime: 30 * 60 * 1000, retry: false });
  const overviewQuery = useQuery({ queryKey: ['olc-zone-overview', zoneNumber], queryFn: () => olcApi.preview('zone', zoneNumber), enabled: Number.isInteger(zoneNumber) && anchorRoom !== null, retry: false });
  const pendingQuery = useQuery({ queryKey: ['olc-pending'], queryFn: olcApi.pending, refetchInterval: 30_000, refetchIntervalInBackground: true, retry: false });
  const heldQuery = useQuery({ queryKey: ['olc-held'], queryFn: olcApi.held, enabled: claimed && !committedDraft, refetchInterval: 30_000, refetchIntervalInBackground: true, retry: false });
  const openMutation = useMutation({
    mutationFn: () => olcApi.openEntityDraft('zone', editVnum),
    onSuccess: (draft) => {
      setClaimed(true);
      queryClient.setQueryData(currentDraftKey, draft);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
    },
  });
  const draftQuery = useQuery({
    queryKey: currentDraftKey,
    queryFn: () => olcApi.getEntityDraft('zone', editVnum),
    enabled: claimed && !committedDraft,
    refetchInterval: 30_000,
    refetchIntervalInBackground: true,
    retry: false,
  });

  useEffect(() => {
    if (!Number.isInteger(zoneNumber) || invalidAnchor || openedRef.current || committedDraft) return;
    openedRef.current = true;
    openMutation.mutate();
  }, [committedDraft, invalidAnchor, openMutation, zoneNumber]);

  const patchMutation = useMutation({
    mutationFn: (operations: OlcPatchOperation[]) => olcApi.patchEntityDraft('zone', editVnum, operations),
    onSuccess: (draft) => {
      queryClient.setQueryData(currentDraftKey, draft);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
    },
  });
  const commitMutation = useMutation({
    mutationFn: () => olcApi.commitEntityDraft('zone', editVnum),
    onSuccess: (draft) => {
      setCommittedDraft(draft);
      setClaimed(false);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
      queryClient.invalidateQueries({ queryKey: ['olc-pending'] });
      queryClient.invalidateQueries({ queryKey: ['olc-zone-overview', zoneNumber] });
      queryClient.invalidateQueries({ queryKey: ['zones'] });
      queryClient.invalidateQueries({ queryKey: ['zone', String(zoneNumber)] });
    },
  });
  const discardMutation = useMutation({ mutationFn: () => olcApi.discardEntityDraft('zone', editVnum), onSuccess: () => navigate(backPath) });
  const saveMutation = useMutation({ mutationFn: (zone: number) => olcApi.saveZone(zone), onSuccess: () => queryClient.invalidateQueries({ queryKey: ['olc-pending'] }) });

  const currentDraft = committedDraft || draftQuery.data || openMutation.data;
  const zone = currentDraft?.zone;
  const busy = patchMutation.isPending || commitMutation.isPending || discardMutation.isPending || saveMutation.isPending;
  const handleOperation = useCallback((operations: OlcPatchOperation[]) => patchMutation.mutate(operations), [patchMutation]);
  const retryClaim = () => { openedRef.current = false; openMutation.reset(); setClaimed(false); };

  if (!Number.isInteger(zoneNumber)) return <EditorError message="The zone number is invalid." />;
  if (invalidAnchor) return <EditorError message="The room anchor is invalid." />;
  if (schemaQuery.isLoading || overviewQuery.isLoading || (openMutation.isPending && !currentDraft) || (claimed && !currentDraft)) return <EditorLoading />;
  if (openMutation.error && !currentDraft) {
    return (
      <div className="space-y-5">
        <Link to={backPath} className="text-sm text-accent hover:text-accent-deep">← Back</Link>
        <div className="border border-accent bg-paper-deep px-5 py-5" role="alert">
          <h1 className="text-xl text-ink">Zone unavailable</h1>
          <p className="mt-2 text-sm text-ink">{conflictSummary(openMutation.error)}</p>
          <button type="button" onClick={retryClaim} className="mt-4 border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">Try again</button>
        </div>
      </div>
    );
  }
  const loadError = schemaQuery.error || overviewQuery.error || draftQuery.error;
  if (loadError || !currentDraft || !zone || !schemaQuery.data) return <EditorError message={(loadError as Error)?.message || 'The zone editor could not load.'} />;

  const claim = heldQuery.data?.find((entry) => entry.kind === 'zone' && entry.number === editVnum);
  const isCommitted = Boolean(committedDraft);
  const saveError = saveMutation.error instanceof ApiError ? saveMutation.error : null;
  const overviewCommands = anchorRoom === null ? zone.commands : overviewQuery.data?.zone?.commands || [];

  return (
    <div className="space-y-4">
      <Link to={backPath} className="text-sm text-accent hover:text-accent-deep">← Back</Link>
      <ClaimSaveFrame draft={currentDraft} kind="zone" title={anchorRoom === null ? 'Zone editor' : 'Zone editor · room #' + anchorRoom} subtitle={zone.name} zone={zone.number} claim={claim} pending={pendingQuery.data || []} leaseRemainingSeconds={currentDraft.leaseRemainingSeconds} committed={isCommitted} busy={busy} saveError={saveError} onCommit={() => commitMutation.mutate()} onDiscard={() => discardMutation.mutate()} onSave={() => saveMutation.mutate(zone.number)}>
        <div className="border border-rule bg-paper-deep px-4 py-5">
          <ZoneSettings schema={schemaQuery.data} zone={zone} dirty={currentDraft.dirty} disabled={busy || isCommitted} onOperation={handleOperation} />
          {anchorRoom !== null ? (
            <>
              <ZoneCommandTimeline schema={schemaQuery.data} title={'Room #' + anchorRoom + ' reset commands'} commands={zone.commands} disabled={busy || isCommitted} onOperation={handleOperation} />
              <ZoneCommandTimeline schema={schemaQuery.data} title="Whole-zone timeline" commands={overviewCommands} disabled readOnly onOperation={() => undefined} />
            </>
          ) : (
            <ZoneCommandTimeline schema={schemaQuery.data} title="Whole-zone timeline (read-only)" commands={overviewCommands} disabled readOnly onOperation={() => undefined} />
          )}
        </div>
      </ClaimSaveFrame>
      {(patchMutation.error || commitMutation.error || discardMutation.error) && <div className="border border-accent bg-paper-deep px-4 py-3 text-sm text-accent" role="alert">{conflictSummary(patchMutation.error || commitMutation.error || discardMutation.error)}</div>}
    </div>
  );
}

function EditorLoading() { return <div className="space-y-5"><Skeleton className="h-4 w-28" /><Skeleton className="h-28 w-full" /><Skeleton className="h-96 w-full" /></div>; }
function EditorError({ message }: { message: string }) { return <div className="border border-accent bg-paper-deep px-5 py-5" role="alert"><h1 className="text-xl text-ink">Editor unavailable</h1><p className="mt-2 text-sm text-accent">{message}</p></div>; }
