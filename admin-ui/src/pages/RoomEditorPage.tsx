import { useCallback, useEffect, useRef, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { olcApi, type OlcRoomDraft, type RoomPatchOperation } from '../api/olc';
import { ClaimSaveFrame } from '../components/olc/ClaimSaveFrame';
import { ExitEditor } from '../components/olc/ExitEditor';
import { ExtraDescriptionList } from '../components/olc/ExtraDescriptionList';
import { LiveScriptControls } from '../components/olc/LiveScriptControls';
import { SchemaFields } from '../components/olc/SchemaFields';
import { Skeleton } from '../components/Skeleton';
import { useAuth } from '../hooks/useAuth';

const draftKey = (vnum: number) => ['olc-room-draft', vnum];

function conflictSummary(error: unknown): string {
  if (!(error instanceof ApiError)) return (error as Error)?.message || 'The room could not be claimed.';
  if (typeof error.payload !== 'object' || error.payload === null) return error.message;
  const payload = error.payload as Record<string, unknown>;
  const holder = typeof payload.holder === 'string' ? payload.holder : 'another editor';
  const frontend = typeof payload.frontend === 'string' ? payload.frontend : 'unknown frontend';
  const idle = typeof payload.idle === 'string' ? payload.idle : 'unknown idle time';
  return `${error.message} Holder: ${holder}. Frontend: ${frontend}. Idle: ${idle}.`;
}

export function RoomEditorPage() {
  const { vnum: rawVnum } = useParams<{ vnum: string }>();
  const vnum = Number(rawVnum);
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { playerName } = useAuth();
  const openedRef = useRef(false);
  const [claimed, setClaimed] = useState(false);
  const [committedDraft, setCommittedDraft] = useState<OlcRoomDraft | null>(null);

  const schemaQuery = useQuery({
    queryKey: ['olc-schema', 'room'],
    queryFn: () => olcApi.schema('room'),
    staleTime: 30 * 60 * 1000,
    retry: false,
  });
  const previewQuery = useQuery({
    queryKey: ['olc-room-preview', vnum],
    queryFn: () => olcApi.preview('room', vnum),
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
    mutationFn: () => olcApi.openRoomDraft(vnum),
    onSuccess: (draft) => {
      setClaimed(true);
      queryClient.setQueryData(draftKey(vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
    },
  });
  const draftQuery = useQuery({
    queryKey: draftKey(vnum),
    queryFn: () => olcApi.getRoomDraft(vnum),
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
    mutationFn: (operations: RoomPatchOperation[]) => olcApi.patchRoomDraft(vnum, operations),
    onSuccess: (draft) => {
      queryClient.setQueryData(draftKey(vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
    },
  });
  const scriptNameMutation = useMutation({
    mutationFn: (text: string) => olcApi.setRoomScriptName(vnum, text),
    onSuccess: (draft) => {
      queryClient.setQueryData(draftKey(vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-room-preview', vnum] });
    },
  });
  const scriptFlagMutation = useMutation({
    mutationFn: ({ bit, enabled }: { bit: number; enabled: boolean }) => olcApi.setRoomScriptFlag(vnum, bit, enabled),
    onSuccess: (draft) => {
      queryClient.setQueryData(draftKey(vnum), draft);
      queryClient.invalidateQueries({ queryKey: ['olc-room-preview', vnum] });
    },
  });
  const commitMutation = useMutation({
    mutationFn: () => olcApi.commitRoomDraft(vnum),
    onSuccess: (draft) => {
      setCommittedDraft(draft);
      setClaimed(false);
      queryClient.invalidateQueries({ queryKey: ['olc-held'] });
      queryClient.invalidateQueries({ queryKey: ['olc-pending'] });
      queryClient.invalidateQueries({ queryKey: ['room', String(vnum)] });
    },
  });
  const discardMutation = useMutation({
    mutationFn: () => olcApi.discardRoomDraft(vnum),
    onSuccess: () => navigate(`/admin/game/rooms/${vnum}`),
  });
  const saveMutation = useMutation({
    mutationFn: (zone: number) => olcApi.saveZone(zone),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['olc-pending'] }),
  });

  const currentDraft = committedDraft || draftQuery.data || openMutation.data;
  const liveRoom = previewQuery.data?.room || currentDraft?.room;
  const busy = patchMutation.isPending || scriptNameMutation.isPending || scriptFlagMutation.isPending || commitMutation.isPending || discardMutation.isPending || saveMutation.isPending;
  const handleOperation = useCallback((operation: RoomPatchOperation) => {
    patchMutation.mutate([operation]);
  }, [patchMutation]);
  const retryClaim = () => {
    openedRef.current = false;
    openMutation.reset();
    setClaimed(false);
  };

  if (!Number.isInteger(vnum)) return <EditorError message="The room VNUM is invalid." />;
  if (schemaQuery.isLoading || previewQuery.isLoading || openMutation.isPending || (claimed && !currentDraft)) {
    return <EditorLoading />;
  }
  if (openMutation.error && !currentDraft) {
    return (
      <div className="space-y-5">
        <Link to={`/admin/game/rooms/${vnum}`} className="text-sm text-accent hover:text-accent-deep">← Back to room</Link>
        <div className="border border-accent bg-paper-deep px-5 py-5" role="alert">
          <h1 className="text-xl text-ink">Room unavailable</h1>
          <p className="mt-2 text-sm text-ink">{conflictSummary(openMutation.error)}</p>
          <button type="button" onClick={retryClaim} className="mt-4 border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">Try again</button>
        </div>
      </div>
    );
  }
  const loadError = schemaQuery.error || previewQuery.error || draftQuery.error;
  if (loadError || !currentDraft || !liveRoom || !schemaQuery.data) {
    return <EditorError message={(loadError as Error)?.message || 'The editor could not load.'} />;
  }

  const claim = heldQuery.data?.find((entry) => entry.kind === 'room' && entry.number === vnum);
  const leaseRemainingSeconds = currentDraft.leaseRemainingSeconds;
  const isCommitted = Boolean(committedDraft);
  const saveError = saveMutation.error instanceof ApiError ? saveMutation.error : null;

  return (
    <div className="space-y-4">
      <Link to={`/admin/game/rooms/${vnum}`} className="text-sm text-accent hover:text-accent-deep">← Back to room</Link>
      <ClaimSaveFrame
        draft={currentDraft}
        claim={claim}
        pending={pendingQuery.data || []}
        leaseRemainingSeconds={leaseRemainingSeconds}
        committed={isCommitted}
        busy={busy}
        saveError={saveError}
        onCommit={() => commitMutation.mutate()}
        onDiscard={() => discardMutation.mutate()}
        onSave={() => saveMutation.mutate(currentDraft.room.zone)}
      >
        <div className="border border-rule bg-paper-deep px-4 py-5">
          <SchemaFields
            schema={schemaQuery.data}
            room={currentDraft.room}
            dirty={currentDraft.dirty}
            disabled={busy || isCommitted}
            onOperation={handleOperation}
          />
          <ExitEditor
            room={currentDraft.room}
            dirty={currentDraft.dirty}
            disabled={busy || isCommitted}
            onOperation={handleOperation}
          />
          <ExtraDescriptionList
            room={currentDraft.room}
            dirty={currentDraft.dirty}
            disabled={busy || isCommitted}
            onOperation={handleOperation}
          />
          <LiveScriptControls
            schema={schemaQuery.data}
            room={liveRoom}
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
      {playerName && <span className="sr-only">Editing as {playerName}</span>}
    </div>
  );
}

function EditorLoading() {
  return (
    <div className="space-y-5">
      <Skeleton className="h-4 w-28" />
      <Skeleton className="h-28 w-full" />
      <Skeleton className="h-96 w-full" />
    </div>
  );
}

function EditorError({ message }: { message: string }) {
  return (
    <div className="border border-accent bg-paper-deep px-5 py-5" role="alert">
      <h1 className="text-xl text-ink">Editor unavailable</h1>
      <p className="mt-2 text-sm text-accent">{message}</p>
    </div>
  );
}
