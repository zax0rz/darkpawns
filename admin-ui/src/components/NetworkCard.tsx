import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { networkApi, api } from '../api/client';
import { useToast } from '../hooks/useToast';
import { useAuth } from '../hooks/useAuth';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Switch } from './ui/switch';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from './ui/select';
import { ConfirmDialog } from './ConfirmDialog';

// The Network card (docs/design/admin-network-settings.md): plaintext
// telnet, TLS telnet and WHOD with enable, port and exposure. HTTP stays
// read-only because it is the management route itself. Saving is a draft in
// the launcher's configuration; Apply and restart runs the launcher's
// transaction, which restores the previous settings automatically when a
// requested listener fails to bind.

interface ListenerView {
  enabled: boolean;
  port: number;
  scope: string;
  address?: string;
  observed?: string;
}

interface NetworkView {
  managed: boolean;
  revision: number;
  http: { port: number; bind: string; read_only: boolean; how_to_change: string };
  tls_certificate_configured: boolean;
  telnet: ListenerView;
  telnet_tls: ListenerView;
  whod: ListenerView;
}

interface Draft {
  enabled: boolean;
  port: number;
  publicAccess: boolean;
}

interface SaveResult {
  config: { revision: number };
  diff: { field: string; from: string; to: string }[];
  warnings: string[];
  restart_required: boolean;
}

interface StatusBody {
  managed: boolean;
  state: string;
  error?: string;
  failed_listener?: string;
}

const NAMES: Record<string, string> = { telnet: 'Telnet', telnet_tls: 'Telnet (TLS)', whod: 'WHOD' };

const APPLY_STATES = new Set(['applying', 'verifying']);

function stateText(state: string, error?: string, failed?: string): { title: string; tone: 'busy' | 'good' | 'bad' } {
  switch (state) {
    case 'applying': return { title: 'Applying. The world is stopping and starting with the new settings.', tone: 'busy' };
    case 'verifying': return { title: 'Verifying. Checking that every requested listener answered.', tone: 'busy' };
    case 'committed': return { title: 'Committed. The new settings are live.', tone: 'good' };
    case 'rolled-back': return { title: `Rolled back. ${failed ? 'The ' + failed + ' listener' : 'A listener'} failed, and the previous settings were restored. ${error ?? ''}`, tone: 'bad' };
    default: return { title: '', tone: 'busy' };
  }
}

function draftFrom(view: NetworkView): Record<string, Draft> {
  const one = (l: ListenerView): Draft => ({ enabled: l.enabled, port: l.port, publicAccess: l.scope === 'public' });
  return { telnet: one(view.telnet), telnet_tls: one(view.telnet_tls), whod: one(view.whod) };
}

export function NetworkCard() {
  const { hasRole } = useAuth();
  const { showToast } = useToast();
  const [drafts, setDrafts] = useState<Record<string, Draft> | null>(null);
  const [conflict, setConflict] = useState(false);
  const [saved, setSaved] = useState<SaveResult | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [applySent, setApplySent] = useState(false);

  const { data: raw, isLoading, error } = useQuery({
    queryKey: ['network'],
    queryFn: networkApi.get,
    enabled: hasRole('admin'),
    retry: false,
  });
  const view = raw as NetworkView | undefined;

  const { data: players } = useQuery({ queryKey: ['players'], queryFn: api.players, enabled: hasRole('admin') });

  const { data: statusRaw } = useQuery({
    queryKey: ['network-status'],
    queryFn: networkApi.status,
    enabled: hasRole('admin') && applySent,
    retry: false,
    refetchInterval: (query) => {
      const state = (query.state.data as StatusBody | undefined)?.state;
      return state && APPLY_STATES.has(state) ? 2000 : 4000;
    },
  });
  const status = statusRaw as StatusBody | undefined;

  const queryClient = useQueryClient();
  const reload = () => {
    setDrafts(null);
    setConflict(false);
    setSaved(null);
    queryClient.invalidateQueries({ queryKey: ['network'] });
  };

  const dirty = view !== undefined && drafts !== null && (
    JSON.stringify(draftFrom(view)) !== JSON.stringify(drafts)
  );

  const saveMutation = useMutation({
    mutationFn: () => {
      if (!view || !drafts) throw new Error('network not loaded');
      const body: Record<string, unknown> = { revision: view.revision };
      for (const key of ['telnet', 'telnet_tls', 'whod']) {
        const d = drafts[key];
        body[key] = { enabled: d.enabled, port: d.enabled ? d.port : 0, public: d.publicAccess };
      }
      return networkApi.save(body);
    },
    onSuccess: (result) => {
      setSaved(result as SaveResult);
      setDrafts(null);
      queryClient.invalidateQueries({ queryKey: ['network'] });
      showToast('Saved as a draft. Nothing changes until you apply and restart.', 'success');
    },
    onError: (err: { status?: number; message?: string }) => {
      if (err.status === 409) {
        setConflict(true);
        showToast('The configuration changed underneath you. Reload before saving.', 'error');
      } else {
        showToast(`Save failed: ${err.message ?? 'unknown error'}`, 'error');
      }
    },
  });

  const applyMutation = useMutation({
    mutationFn: networkApi.apply,
    onSuccess: () => {
      setApplySent(true);
      showToast('Applying. The world restarts; this page reconnects by itself.', 'success');
    },
    onError: (err: Error) => showToast(`Apply failed: ${err.message}`, 'error'),
  });

  if (!hasRole('admin')) return null;
  if (isLoading) return <section className="border border-rule bg-paper-deep p-4"><p className="text-sm text-ink-muted animate-pulse">Loading network settings…</p></section>;
  if (error || !view) return (
    <section className="border border-rule bg-paper-deep p-4">
      <h2 className="text-base font-semibold text-ink">Network</h2>
      <p className="text-sm text-accent" role="alert">Network settings unavailable: {(error as Error | null)?.message ?? 'unknown error'}</p>
    </section>
  );

  const effective = drafts ?? draftFrom(view);
  const setDraft = (key: string, patch: Partial<Draft>) =>
    setDrafts((prev) => ({ ...(prev ?? draftFrom(view)), [key]: { ...(prev ?? draftFrom(view))[key], ...patch } }));

  const listenerBlock = (key: string, l: ListenerView, d: Draft, tlsNote?: string) => (
    <div key={key} className="border-t border-rule py-3 first:border-t-0">
      <div className="flex items-center justify-between gap-4">
        <span className="text-sm font-medium text-ink">{NAMES[key]}</span>
        <div className="flex items-center gap-3">
          {l.observed && <span className="text-xs text-ink-muted">observed {l.observed}</span>}
          <Switch checked={d.enabled} disabled={!view.managed || (key === 'telnet_tls' && !view.tls_certificate_configured)}
            onCheckedChange={(checked) => setDraft(key, { enabled: checked })} />
        </div>
      </div>
      {tlsNote && !view.tls_certificate_configured && <p className="mt-1 text-xs text-ink-muted">{tlsNote}</p>}
      {d.enabled && l.scope !== 'address' && (
        <div className="mt-2 flex items-center justify-between gap-4">
          <label className="flex items-center gap-2 text-sm text-ink">
            Port
            <Input type="number" className="w-24 text-right font-mono" value={d.port} disabled={!view.managed}
              onChange={(e) => setDraft(key, { port: Number(e.target.value) })} />
          </label>
          <label className="flex items-center gap-2 text-sm text-ink">
            Reachable from
            <Select value={d.publicAccess ? 'public' : 'loopback'} onValueChange={(v) => setDraft(key, { publicAccess: v === 'public' })}>
              <SelectTrigger className="w-36" disabled={!view.managed}><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="loopback">This computer</SelectItem>
                <SelectItem value="public">Public</SelectItem>
              </SelectContent>
            </Select>
          </label>
        </div>
      )}
      {l.scope === 'address' && (
        <p className="mt-1 text-xs text-ink-muted">Bound to an explicit address set outside the console ({l.address}); saving replaces it with the choice above on apply.</p>
      )}
    </div>
  );

  const info = status ? stateText(status.state, status.error, status.failed_listener) : null;
  const online = (players as { name: string }[] | undefined) ?? [];

  return (
    <section className="border border-rule bg-paper-deep p-4">
      <header className="mb-2 flex items-baseline justify-between gap-2">
        <h2 className="text-base font-semibold text-ink">Network</h2>
        <span className="text-xs text-ink-muted">{view.managed ? 'Changes apply after a restart.' : 'Read-only on this server'}</span>
      </header>

      <div className="flex items-center justify-between gap-4 border-b border-rule py-2">
        <div>
          <p className="text-sm font-medium text-ink">Web console and API (HTTP)</p>
          <p className="text-xs text-ink-muted">{view.http.how_to_change}</p>
        </div>
        <span className="font-mono text-sm text-ink">{view.http.bind || 'all interfaces'}:{view.http.port}</span>
      </div>

      {listenerBlock('telnet', view.telnet, effective.telnet)}
      {listenerBlock('telnet_tls', view.telnet_tls, effective.telnet_tls, 'TLS needs a certificate configured on the server before it can be turned on.')}
      {listenerBlock('whod', view.whod, effective.whod)}

      {!view.managed && (
        <p className="mt-3 text-sm text-ink-muted">
          These settings are read only on this server. Change listeners by stopping the server and starting
          it again with different -telnet-port, -telnet-bind or -whod flags.
        </p>
      )}

      {conflict && (
        <div className="mt-3 border border-accent bg-accent/5 p-3 text-sm text-ink" role="alert">
          The configuration changed underneath you.
          <Button variant="outline" size="sm" className="ml-2" onClick={reload}>Reload</Button>
        </div>
      )}

      {saved && (
        <div className="mt-3 border border-ink bg-paper p-3 text-sm" role="status">
          <p className="mb-2 font-medium">Restart required. Draft saved; nothing has changed yet.</p>
          {saved.diff.length > 0 ? (
            <table className="mb-2 w-full">
              <tbody>
                {saved.diff.map((entry) => (
                  <tr key={entry.field} className="border-b border-rule font-mono text-xs">
                    <td className="py-1 pr-2">{NAMES[entry.field] ?? entry.field}</td>
                    <td className="py-1 pr-2 text-ink-muted">{entry.from}</td>
                    <td className="py-1">{entry.to}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : (
            <p className="mb-2 text-ink-muted">No exposure changes in this draft.</p>
          )}
          {saved.warnings.map((warning) => (
            <p key={warning} className="mb-1 text-xs text-accent">{warning}</p>
          ))}
          <p className="mb-2 text-xs text-ink-muted">
            {online.length === 0 ? 'No one is online.' :
              `${online.length} ${online.length === 1 ? 'player' : 'players'} online: ${online.slice(0, 5).map((p) => p.name).join(', ')}${online.length > 5 ? ` and ${online.length - 5} more` : ''}.`}
            {' '}Applying disconnects them; characters are saved first.
          </p>
          <Button onClick={() => setConfirmOpen(true)} disabled={applySent && status !== undefined && APPLY_STATES.has(status.state)}>Apply and restart</Button>
          <Button variant="outline" className="ml-2" onClick={reload}>Discard draft</Button>
        </div>
      )}

      {view.managed && dirty && !saved && (
        <div className="mt-3 flex items-center justify-between gap-4 border border-ink bg-paper p-3">
          <span className="text-sm text-ink-muted">Unsaved network changes</span>
          <div className="flex gap-2">
            <Button variant="outline" onClick={reload}>Discard</Button>
            <Button onClick={() => saveMutation.mutate()} disabled={saveMutation.isPending}>Save draft</Button>
          </div>
        </div>
      )}

      {info && info.title && (
        <p className={`mt-3 text-sm ${info.tone === 'bad' ? 'text-accent' : info.tone === 'good' ? 'text-ink' : 'text-ink-muted'}`} role="status">{info.title}</p>
      )}

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Apply network changes and restart?"
        description={
          <div className="space-y-2">
            <p>
              {online.length === 0 ? 'No one is online.' :
                `${online.length} ${online.length === 1 ? 'player' : 'players'} online: ${online.slice(0, 5).map((p) => p.name).join(', ')}${online.length > 5 ? ` and ${online.length - 5} more` : ''}.`}
              {' '}They will be disconnected; characters are saved first.
            </p>
            <ul className="list-disc pl-5">
              {saved?.diff.map((entry) => (
                <li key={entry.field} className="font-mono text-xs">{NAMES[entry.field] ?? entry.field}: {entry.from} to {entry.to}</li>
              ))}
            </ul>
            {saved?.warnings.map((warning) => (
              <p key={warning} className="text-xs text-accent">{warning}</p>
            ))}
            <p className="text-xs text-ink-muted">
              The web console at its current address keeps working through the restart. If any requested
              listener fails to bind, the previous settings come back automatically.
            </p>
          </div>
        }
        action="Apply and restart"
        onConfirm={() => applyMutation.mutate()}
      />
    </section>
  );
}
