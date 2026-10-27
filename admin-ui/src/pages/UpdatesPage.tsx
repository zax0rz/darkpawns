import { useState } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { updatesApi, api } from '../api/client';
import { useToast } from '../hooks/useToast';
import { useAuth } from '../hooks/useAuth';
import { Button } from '../components/ui/button';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { Icon } from '../components/Icon';

// Updates (admin-configuration design): a read-only Check, and Update now,
// the same transaction `darkpawns update` runs, handed to the launcher. The
// dying game process never waits on its own update; this page polls the
// journal-backed status, which survives both the restart and a refresh.

interface UpdateState {
  state: string;
  runtime_state?: string;
  current_version?: string;
  transaction_id?: string;
  phase?: string;
  previous_version?: string;
  candidate_version?: string;
  error?: string;
  recover_command?: string;
  requested_at?: string;
}

interface StatusBody {
  managed: boolean;
  server_version: string;
  commit?: string;
  online_players: number;
  update?: UpdateState;
}

interface CheckBody {
  current_version: string;
  latest_version?: string;
  up_to_date: boolean;
  release_url?: string;
  asset_size?: number;
  error?: string;
}

// Journal-backed phases disable the Update button: the launcher's own locks
// serialize transactions, and a staging marker alone (the CLI is still
// running, or died before writing a journal) must not wedge the page — the
// button stays available and the launcher refuses a true double-run.
const ACTIVE_STATES = new Set(['applying', 'testing', 'recovering']);
const FAST_POLL_STATES = new Set(['staging', 'applying', 'testing', 'recovering']);

function stateText(state: UpdateState | undefined): { title: string; detail?: string; tone: 'idle' | 'busy' | 'good' | 'bad' } {
  switch (state?.state) {
    case undefined: return { title: 'No transaction', tone: 'idle' };
    case 'idle': return { title: 'Idle', tone: 'idle' };
    case 'staging': return { title: 'Staging', detail: 'Downloading and verifying the new release; the world keeps running.', tone: 'busy' };
    case 'applying': return { title: 'Applying', detail: 'The world is stopped: snapshot, swap, candidate start. This page reconnects automatically.', tone: 'busy' };
    case 'testing': return { title: 'Health-checking', detail: 'The candidate release is up; the launcher gates on readiness and bundle identity.', tone: 'busy' };
    case 'recovering': return { title: 'Recovering', detail: 'Rolling back to the verified snapshot.', tone: 'busy' };
    case 'committed': return { title: 'Committed', detail: `Now running ${state.current_version ?? 'the new release'}.`, tone: 'good' };
    case 'rolled-back': return { title: 'Rolled back', detail: 'The previous release was restored automatically; the backup is retained.', tone: 'bad' };
    case 'pending-recovery': return { title: 'Recovery needed', detail: state.recover_command ?? 'Run darkpawns update recover', tone: 'bad' };
    default: return { title: state?.state ?? 'unknown', detail: state?.error, tone: 'bad' };
  }
}

export function UpdatesPage() {
  const { hasRole } = useAuth();
  const { showToast } = useToast();
  const [check, setCheck] = useState<CheckBody | null>(null);
  const [applyOpen, setApplyOpen] = useState(false);
  const [requested, setRequested] = useState(() => sessionStorage.getItem('dp-update-requested'));

  const { data: status, error: statusError } = useQuery({
    queryKey: ['updates-status'],
    queryFn: updatesApi.status,
    enabled: hasRole('admin'),
    retry: false,
    refetchInterval: (query) => {
      const state = (query.state.data as StatusBody | undefined)?.update?.state;
      return state && FAST_POLL_STATES.has(state) ? 2000 : 15000;
    },
  });

  const { data: players } = useQuery({ queryKey: ['players'], queryFn: api.players, enabled: hasRole('admin') });

  const txState = (status as StatusBody | undefined)?.update?.state;
  if (requested && txState && (txState === 'committed' || txState === 'rolled-back' || txState === 'pending-recovery')) {
    // Terminal journal state proves the transaction ended; clearing during
    // render keeps the flag consistent across refreshes without a re-render
    // loop (the derived value below reads the same condition).
    sessionStorage.removeItem('dp-update-requested');
  }

  const checkMutation = useMutation({
    mutationFn: updatesApi.check,
    onSuccess: (body) => setCheck(body as CheckBody),
    onError: (err: Error) => showToast(`Check failed: ${err.message}`, 'error'),
  });

  const applyMutation = useMutation({
    mutationFn: updatesApi.apply,
    onSuccess: () => {
      sessionStorage.setItem('dp-update-requested', new Date().toISOString());
      setRequested(new Date().toISOString());
      showToast('Update requested. The launcher stages first; the world keeps running until the swap.', 'success');
    },
    onError: (err: Error) => showToast(`Update failed to start: ${err.message}`, 'error'),
  });

  if (!hasRole('admin')) return <p className="text-ink-muted">Updates require implementor authority.</p>;

  const body = status as StatusBody | undefined;
  const waiting = requested !== null && statusError !== null;
  const state = body?.update;
  const info = stateText(state);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-ink">Updates</h1>

      <section className="border border-rule bg-paper-deep p-4">
        <h2 className="mb-1 text-base font-semibold text-ink">Current release</h2>
        <p className="text-sm text-ink">
          <span className="font-mono">{body?.server_version ?? '…'}</span>
          {body?.commit ? <span className="text-ink-muted"> ({body.commit.slice(0, 7)})</span> : null}
          <span className="text-ink-muted"> · {body?.managed ? 'managed instance' : 'unmanaged (checkout deployment)'}</span>
          <span className="text-ink-muted"> · {body?.online_players ?? 0} online</span>
        </p>
        {!body?.managed && (
          <p className="mt-2 text-sm text-ink-muted">
            This server runs outside the portable launcher, so browser updates are unavailable.
            Deploy with <span className="font-mono">darkpawns</span> or your existing procedure (DEPLOYMENT.md).
          </p>
        )}
      </section>

      <section className={`border p-4 ${info.tone === 'bad' ? 'border-accent' : 'border-rule'} bg-paper-deep`}>
        <h2 className="mb-1 text-base font-semibold text-ink">Update transaction</h2>
        {waiting ? (
          <p className="text-sm text-ink" role="status">Server unreachable: applying the update. This page reconnects automatically; a refresh is safe.</p>
        ) : (
          <div className="text-sm">
            <p className={info.tone === 'bad' ? 'text-accent' : 'text-ink'}>{info.title}{state?.phase ? ` (${state.phase})` : ''}</p>
            {info.detail && <p className="text-ink-muted">{info.detail}</p>}
            {state?.previous_version && state?.candidate_version && (
              <p className="text-ink-muted">{state.previous_version} → {state.candidate_version}</p>
            )}
            {state?.state === 'pending-recovery' && state.recover_command && (
              <p className="mt-2">Run <span className="bg-paper px-1 font-mono">{state.recover_command}</span> on the server, or paste it into your session.</p>
            )}
          </div>
        )}
        {body?.managed && (
          <div className="mt-3 flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => checkMutation.mutate()} disabled={checkMutation.isPending}>
              <Icon name="refresh" /> Check for updates
            </Button>
            <Button onClick={() => setApplyOpen(true)} disabled={!state || ACTIVE_STATES.has(state.state)}>
              Update now
            </Button>
          </div>
        )}
        {check && (
          <div className="mt-3 border-t border-rule pt-3 text-sm" role="status">
            {check.error ? (
              <p className="text-accent">{check.error}</p>
            ) : check.up_to_date ? (
              <p className="text-ink">Up to date ({check.current_version}).</p>
            ) : (
              <p className="text-ink">
                {check.current_version || 'unknown'} → <span className="font-mono">{check.latest_version}</span>
                {check.asset_size ? <span className="text-ink-muted"> ({Math.ceil(check.asset_size / 1e6)} MB)</span> : null}
                {check.release_url && (
                  <> · <a className="underline" href={check.release_url} target="_blank" rel="noreferrer">release notes</a></>
                )}
              </p>
            )}
          </div>
        )}
      </section>

      <ConfirmDialog
        open={applyOpen}
        onOpenChange={setApplyOpen}
        title="Update now?"
        description={[
          `${players?.length ?? 0} player(s) online will be disconnected; every character is saved first.`,
          'The launcher stages and verifies first (the world keeps running), then stops, snapshots, swaps and health-checks the new release.',
          'A failed health check rolls back to the previous release automatically.',
          'Local success says nothing about LAN/WAN reachability; that stays your reverse proxy and firewall.',
        ].join(' ')}
        action="Update now"
        onConfirm={() => applyMutation.mutate()}
      />
    </div>
  );
}
