import { useCallback, useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import {
  olcApi,
  type OlcEntityDraft,
  type OlcMob,
  type OlcObject,
  type OlcPatchOperation,
} from '../api/olc';
import { ClaimSaveFrame } from '../components/olc/ClaimSaveFrame';
import { EntitySchemaFields } from '../components/olc/EntitySchemaFields';
import { ExtraDescriptionList } from '../components/olc/ExtraDescriptionList';
import { LiveScriptControls } from '../components/olc/LiveScriptControls';
import { ObjectAppliesEditor } from '../components/olc/ObjectAppliesEditor';
import { ObjectValueMatrixEditor } from '../components/olc/ObjectValueMatrixEditor';
import { Skeleton } from '../components/Skeleton';

type EntityKind = 'mob' | 'obj';

interface EntityEditorPageProps {
  kind: EntityKind;
}

const draftKey = (kind: EntityKind, vnum: number) => ['olc-entity-draft', kind, vnum];

function draftKind(kind: EntityKind): string {
  return kind === 'obj' ? 'object' : kind;
}

function conflictSummary(error: unknown): string {
  if (!(error instanceof ApiError)) return (error as Error)?.message || 'The entity could not be claimed.';
  if (typeof error.payload !== 'object' || error.payload === null) return error.message;
  const payload = error.payload as Record<string, unknown>;
  const holder = typeof payload.holder === 'string' ? payload.holder : 'another editor';
  const frontend = typeof payload.frontend === 'string' ? payload.frontend : 'unknown frontend';
  const idle = typeof payload.idle === 'string' ? payload.idle : 'unknown idle time';
  return `${error.message} Holder: ${holder}. Frontend: ${frontend}. Idle: ${idle}.`;
}

export function EntityEditorPage({ kind }: EntityEditorPageProps) {
  const { vnum: rawVnum } = useParams<{ vnum: string }>();
  const vnum = Number(rawVnum);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const openedRef = useRef(false);
  const [claimed, setClaimed] = useState(false);
  const [committedDraft, setCommittedDraft] = useState<OlcEntityDraft | null>(null);
  const resourceKind = draftKind(kind);
  const listPath = kind === 'mob' ? '/admin/game/mobs' : '/admin/game/objects';
  const title = kind === 'mob' ? 'Mob editor' : 'Object editor';

  const schemaQuery = useQuery({
    queryKey: ['olc-schema', kind],
    queryFn: () => olcApi.schema(kind),
    staleTime: 30 * 60 * 1000,
    retry: false,
  });
  const previewQuery = useQuery({
    queryKey: ['olc-entity-preview', kind, vnum],
    queryFn: () => olcApi.preview(kind, vnum),
    enabled: Number.isInteger(vnum),
    staleTime: 0,
    retry: false,
  });
  const pendingQuery = useQuery({
    queryKey: ['olc-pending'],
    queryFn: olcApi.pending,
    refetchInterval: 30_000,
    refetchIntervalInBackground: true,
    retry: false,
  });
  const heldQuery = useQuery({
    queryKey: ['olc-held'],
    queryFn: olcApi.held,
    enabled: claimed && !committedDraft,
    refetchInterval: 30_000,
    refetchIntervalInBackground: true,
    retry: false,
  });
  const openMutation = useMutation({
    mutationFn: () => olcApi.openEntityDraft(kind, vnum),
    onSuccess: (draft) => {
      setClaimed(true);
      queryClient.setQueryData(draftKey(kind, vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
    },
  });
  const draftQuery = useQuery({
    queryKey: draftKey(kind, vnum),
    queryFn: () => olcApi.getEntityDraft(kind, vnum),
    enabled: claimed && !committedDraft,
    refetchInterval: 30_000,
    refetchIntervalInBackground: true,
    retry: false,
  });

  useEffect(() => {
    if (!Number.isInteger(vnum) || openedRef.current || committedDraft) return;
    openedRef.current = true;
    openMutation.mutate();
  }, [committedDraft, openMutation, vnum]);

  const patchMutation = useMutation({
    mutationFn: (operations: OlcPatchOperation[]) => olcApi.patchEntityDraft(kind, vnum, operations),
    onSuccess: (draft) => {
      queryClient.setQueryData(draftKey(kind, vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
    },
  });
  const scriptNameMutation = useMutation({
    mutationFn: (text: string) => kind === 'mob' ? olcApi.setMobScriptName(vnum, text) : olcApi.setObjectScriptName(vnum, text),
    onSuccess: (draft) => {
      queryClient.setQueryData(draftKey(kind, vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-entity-preview', kind, vnum] });
    },
  });
  const scriptFlagMutation = useMutation({
    mutationFn: ({ bit, enabled }: { bit: number; enabled: boolean }) => kind === 'mob'
      ? olcApi.setMobScriptFlag(vnum, bit, enabled)
      : olcApi.setObjectScriptFlag(vnum, bit, enabled),
    onSuccess: (draft) => {
      queryClient.setQueryData(draftKey(kind, vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-entity-preview', kind, vnum] });
    },
  });
  const commitMutation = useMutation({
    mutationFn: () => olcApi.commitEntityDraft(kind, vnum),
    onSuccess: (draft) => {
      setCommittedDraft(draft);
      setClaimed(false);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
      queryClient.invalidateQueries({ queryKey: ['olc-pending'] });
      queryClient.invalidateQueries({ queryKey: [kind === 'mob' ? 'mob' : 'object', String(vnum)] });
      queryClient.invalidateQueries({ queryKey: [kind === 'mob' ? 'mobs' : 'objects'] });
    },
  });
  const discardMutation = useMutation({
    mutationFn: () => olcApi.discardEntityDraft(kind, vnum),
    onSuccess: () => navigate(listPath),
  });
  const saveMutation = useMutation({
    mutationFn: (zone: number) => olcApi.saveZone(zone),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['olc-pending'] }),
  });

  const currentDraft = committedDraft || draftQuery.data || openMutation.data;
  const entity = kind === 'mob' ? currentDraft?.mob : currentDraft?.object;
  const liveEntity = kind === 'mob' ? previewQuery.data?.mob : previewQuery.data?.object;
  const zone = previewQuery.data?.zoneNumber || 0;
  const busy = patchMutation.isPending || scriptNameMutation.isPending || scriptFlagMutation.isPending || commitMutation.isPending || discardMutation.isPending || saveMutation.isPending;
  const handleOperation = useCallback((operations: OlcPatchOperation[]) => patchMutation.mutate(operations), [patchMutation]);
  const retryClaim = () => {
    openedRef.current = false;
    openMutation.reset();
    setClaimed(false);
  };

  if (!Number.isInteger(vnum)) return <EditorError message={`The ${kind} VNUM is invalid.`} />;
  if (schemaQuery.isLoading || previewQuery.isLoading || openMutation.isPending || (claimed && !currentDraft)) return <EditorLoading />;
  if (openMutation.error && !currentDraft) {
    return (
      <div className="space-y-5">
        <Link to={listPath} className="text-sm text-accent hover:text-accent-deep">← Back</Link>
        <div className="border border-accent bg-paper-deep px-5 py-5" role="alert">
          <h1 className="text-xl text-ink">{title} unavailable</h1>
          <p className="mt-2 text-sm text-ink">{conflictSummary(openMutation.error)}</p>
          <button type="button" onClick={retryClaim} className="mt-4 border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">Try again</button>
        </div>
      </div>
    );
  }
  const loadError = schemaQuery.error || previewQuery.error || draftQuery.error;
  if (loadError || !currentDraft || !entity || !liveEntity || !schemaQuery.data || !zone) return <EditorError message={(loadError as Error)?.message || 'The editor could not load.'} />;

  const claim = heldQuery.data?.find((entry) => entry.kind === resourceKind && entry.number === vnum);
  const isCommitted = Boolean(committedDraft);
  const saveError = saveMutation.error instanceof ApiError ? saveMutation.error : null;

  return (
    <div className="space-y-4">
      <Link to={listPath} className="text-sm text-accent hover:text-accent-deep">← Back</Link>
      <ClaimSaveFrame
        draft={currentDraft}
        kind={resourceKind}
        title={title}
        subtitle={kind === 'mob' ? (entity as OlcMob).shortDesc : (entity as OlcObject).shortDesc}
        zone={zone}
        claim={claim}
        pending={pendingQuery.data || []}
        leaseRemainingSeconds={currentDraft.leaseRemainingSeconds}
        committed={isCommitted}
        busy={busy}
        saveError={saveError}
        onCommit={() => commitMutation.mutate()}
        onDiscard={() => discardMutation.mutate()}
        onSave={() => saveMutation.mutate(zone)}
      >
        <div className="border border-rule bg-paper-deep px-4 py-5">
          <EntitySchemaFields
            schema={schemaQuery.data}
            kind={kind}
            entity={entity}
            dirty={currentDraft.dirty}
            disabled={busy || isCommitted}
            onOperation={handleOperation}
          />
          {kind === 'obj' && (
            <>
              <ObjectValueMatrixEditor schema={schemaQuery.data} object={entity as OlcObject} disabled={busy || isCommitted} onOperation={handleOperation} />
              <ObjectAppliesEditor schema={schemaQuery.data} object={entity as OlcObject} disabled={busy || isCommitted} onOperation={handleOperation} />
              <ExtraDescriptionList entries={(entity as OlcObject).extraDescs} kind="object" dirty={currentDraft.dirty} disabled={busy || isCommitted} onOperation={(operation) => handleOperation([operation])} />
            </>
          )}
          <LiveScriptControls
            schema={schemaQuery.data}
            value={{ scriptName: liveEntity.scriptName, scriptFunctions: liveEntity.luaFunctions }}
            disabled={busy || isCommitted}
            onScriptName={(text) => scriptNameMutation.mutate(text)}
            onScriptFlag={(bit, enabled) => scriptFlagMutation.mutate({ bit, enabled })}
          />
        </div>
      </ClaimSaveFrame>
      {(patchMutation.error || scriptNameMutation.error || scriptFlagMutation.error || commitMutation.error || discardMutation.error) && (
        <div className="border border-accent bg-paper-deep px-4 py-3 text-sm text-accent" role="alert">
          {conflictSummary(patchMutation.error || scriptNameMutation.error || scriptFlagMutation.error || commitMutation.error || discardMutation.error)}
        </div>
      )}
    </div>
  );
}

function EditorLoading() {
  return <div className="space-y-5"><Skeleton className="h-4 w-28" /><Skeleton className="h-28 w-full" /><Skeleton className="h-96 w-full" /></div>;
}

function EditorError({ message }: { message: string }) {
  return <div className="border border-accent bg-paper-deep px-5 py-5" role="alert"><h1 className="text-xl text-ink">Editor unavailable</h1><p className="mt-2 text-sm text-accent">{message}</p></div>;
}
