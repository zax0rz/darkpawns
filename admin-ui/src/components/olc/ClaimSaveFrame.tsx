import type { ReactNode } from 'react';
import { useEffect, useState } from 'react';
import type { ApiError } from '../../api/client';
import type { OlcClaimEntry, OlcDirtyEntry, OlcEntityDraft, OlcRoomDraft } from '../../api/olc';

type OlcEditorDraft = OlcRoomDraft | OlcEntityDraft;

interface ClaimSaveFrameProps {
  draft: OlcEditorDraft;
  kind: string;
  title: string;
  subtitle: string;
  zone: number;
  claim?: OlcClaimEntry;
  pending: OlcDirtyEntry[];
  leaseRemainingSeconds: number;
  committed: boolean;
  busy: boolean;
  saveError: ApiError | null;
  children: ReactNode;
  onCommit: () => void;
  onDiscard: () => void;
  onSave: () => void;
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.max(0, seconds)}s`;
  return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
}

function conflictDetails(error: ApiError | null): Record<string, string> {
  if (!error || typeof error.payload !== 'object' || error.payload === null) return {};
  const payload = error.payload as Record<string, unknown>;
  return {
    holder: typeof payload.holder === 'string' ? payload.holder : '',
    frontend: typeof payload.frontend === 'string' ? payload.frontend : '',
    idle: typeof payload.idle === 'string' ? payload.idle : '',
  };
}

export function ClaimSaveFrame({
  draft,
  kind,
  title,
  subtitle,
  zone,
  claim,
  pending,
  leaseRemainingSeconds,
  committed,
  busy,
  saveError,
  children,
  onCommit,
  onDiscard,
  onSave,
}: ClaimSaveFrameProps) {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (committed) return undefined;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [committed]);

  const idleSeconds = claim?.claimedAt
    ? Math.max(0, Math.floor((now - Date.parse(claim.claimedAt)) / 1000))
    : 0;
  const zoneDirty = pending.some((entry) => entry.kind === kind && entry.zone === zone);
  const details = conflictDetails(saveError);

  return (
    <div className="space-y-5">
      <div className="border border-rule bg-paper-deep">
        <div className="flex flex-col gap-4 border-b border-rule px-4 py-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
              <h1 className="text-2xl text-ink">{title}</h1>
              <span className="font-mono text-sm text-accent">#{draft.vnum}</span>
            </div>
            <p className="mt-1 text-sm text-ink-muted">{subtitle || `#${draft.vnum}`}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            {!committed && (
              <>
                <button
                  type="button"
                  disabled={busy}
                  onClick={onDiscard}
                  className="border border-rule bg-paper px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
                >
                  Discard
                </button>
                <button
                  type="button"
                  disabled={busy}
                  onClick={onCommit}
                  className="border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:cursor-not-allowed disabled:opacity-50"
                >
                  Commit draft
                </button>
              </>
            )}
            {committed && (
              <span className="border border-online px-3 py-2 font-mono text-xs uppercase tracking-wider text-online">Committed</span>
            )}
          </div>
        </div>

        <div className="grid gap-3 px-4 py-3 text-xs text-ink-muted md:grid-cols-3">
          <div>
            <span className="font-semibold text-ink">Holder:</span>{' '}
            {committed ? 'released' : claim?.ownerDisplayName || 'this editor'}
          </div>
          <div>
            <span className="font-semibold text-ink">Frontend:</span>{' '}
            {committed ? 'not held' : claim?.ownerFrontend || 'web'}
          </div>
          <div>
            <span className="font-semibold text-ink">Idle:</span>{' '}
            {committed ? 'not held' : formatDuration(idleSeconds)}
          </div>
          {!committed && (
            <div className={leaseRemainingSeconds < 60 ? 'font-semibold text-accent' : ''}>
              <span className="font-semibold text-ink">Lease:</span>{' '}
              {formatDuration(leaseRemainingSeconds)}
            </div>
          )}
          <div className={zoneDirty ? 'font-semibold text-accent' : ''}>
            <span className="font-semibold text-ink">Zone {zone}:</span>{' '}
            {zoneDirty ? 'dirty; save required' : 'clean'}
          </div>
        </div>
      </div>

      {saveError && (
        <div className="border border-accent bg-paper-deep px-4 py-3 text-sm text-ink" role="alert">
          <p className="font-semibold text-accent">Save refused</p>
          <p className="mt-1">{saveError.message}</p>
          {(details.holder || details.frontend || details.idle) && (
            <p className="mt-2 font-mono text-xs text-ink-muted">
              Holder: {details.holder || 'unknown'} · Frontend: {details.frontend || 'unknown'} · Idle: {details.idle || 'unknown'}
            </p>
          )}
        </div>
      )}

      {children}

      <div className="flex flex-col gap-3 border-t border-rule pt-4 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-ink-muted">
              {zoneDirty ? `Zone ${zone} has committed changes waiting for a file save.` : 'Commit changes before saving the zone file.'}
        </p>
        <button
          type="button"
          disabled={busy || !zoneDirty || !committed}
          onClick={onSave}
          className="border border-accent bg-accent px-4 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:cursor-not-allowed disabled:opacity-50"
        >
          Save zone file
        </button>
      </div>
    </div>
  );
}
