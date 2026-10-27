import { useRef, useState, useEffect } from 'react';
import { useBlocker } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { settingsApi, api } from '../api/client';
import { useToast } from '../hooks/useToast';
import { useAuth } from '../hooks/useAuth';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select';
import { ConfirmDialog } from '../components/ConfirmDialog';
import { Icon } from '../components/Icon';

// Configuration (admin-configuration design): every game-owned runtime
// setting, typed end to end. Secrets are write-only (empty keeps the sealed
// value), saves are revision-checked, and a save that touches restart-marked
// fields raises the banner with the launcher's restart transaction.

interface FieldMeta {
  restart_fields?: string[] | null;
  env_overridden?: string[] | null;
}

interface SettingsView {
  format: number;
  revision: number;
  sessions: Record<string, number>;
  access: { trusted_proxies?: string[]; cors_allowed_origins?: string[]; admin_cors_origin?: string };
  grapevine: { mode: string; url: string; client_id: string; client_secret_set: boolean };
  staff_push: { url: string; min_level: number; min_type: number; rate_per_minute: number; token_set: boolean };
  contact: {
    to: string; smtp_host: string; smtp_port: number; smtp_user: string;
    turnstile_hostnames?: string[]; smtp_password_set: boolean; turnstile_secret_set: boolean; effective: boolean;
  };
  meta: Record<string, FieldMeta>;
}

interface SaveResult {
  revision: number;
  restart_required?: string[] | null;
  env_overridden?: string[] | null;
  warning?: string;
}

function NumberField({ label, value, onChange, hint }: { label: string; value: number; onChange: (v: number) => void; hint?: string }) {
  return (
    <label className="flex items-center justify-between gap-4 py-1.5">
      <span className="text-sm text-ink">{label}{hint && <span className="block text-xs text-ink-muted">{hint}</span>}</span>
      <Input type="number" className="w-28 text-right font-mono" value={value} onChange={(e) => onChange(Number(e.target.value))} />
    </label>
  );
}

function TextField({ label, value, onChange, placeholder, mono }: { label: string; value: string; onChange: (v: string) => void; placeholder?: string; mono?: boolean }) {
  return (
    <label className="flex items-center justify-between gap-4 py-1.5">
      <span className="text-sm text-ink">{label}</span>
      <Input className={mono ? 'font-mono' : ''} value={value} placeholder={placeholder} onChange={(e) => onChange(e.target.value)} />
    </label>
  );
}

function SecretField({ label, set, onSet, onClear }: { label: string; set: boolean; onSet: (v: string) => void; onClear: () => void }) {
  const [value, setValue] = useState('');
  const [cleared, setCleared] = useState(false);
  const status = cleared ? 'will be cleared' : value !== '' ? 'will be replaced' : set ? 'set, unchanged unless replaced' : 'not set';
  return (
    <div className="py-1.5">
      <div className="flex items-center justify-between gap-4">
        <span className="text-sm text-ink">{label}</span>
        <div className="flex items-center gap-2">
          <Input type="password" className="w-56 font-mono" value={value} placeholder={set ? 'unchanged' : 'not set'}
            onChange={(e) => { setValue(e.target.value); setCleared(false); if (e.target.value !== '') onSet(e.target.value); }} />
          {set && (
            <Button variant="outline" size="sm" onClick={() => { setCleared(true); setValue(''); onClear(); }}>Clear</Button>
          )}
        </div>
      </div>
      <p className="text-xs text-ink-muted text-right">{status}</p>
    </div>
  );
}

function GroupCard({ title, subtitle, children, envOverridden }: { title: string; subtitle: string; children: React.ReactNode; envOverridden?: string[] }) {
  return (
    <section className="border border-rule bg-paper-deep p-4">
      <header className="mb-2 flex items-baseline justify-between gap-2">
        <h2 className="text-base font-semibold text-ink">{title}</h2>
        {envOverridden && envOverridden.length > 0 && (
          <span className="border border-ink-muted/40 px-2 py-0.5 text-xs text-ink-muted" title="An exported environment variable still wins over the file for these fields.">
            env overrides: {envOverridden.join(', ')}
          </span>
        )}
      </header>
      <p className="mb-3 text-xs text-ink-muted">{subtitle}</p>
      {children}
    </section>
  );
}

export function SettingsPage() {
  const { hasRole } = useAuth();
  const { showToast } = useToast();
  const queryClient = useQueryClient();
  const [drafts, setDrafts] = useState<Record<string, unknown>>({});
  const [conflict, setConflict] = useState(false);
  const [pendingRestart, setPendingRestart] = useState<string[]>([]);
  const [restartOpen, setRestartOpen] = useState(false);
  const [restorePreview, setRestorePreview] = useState<{ diff: { field: string; current: unknown; incoming: unknown }[]; unknown_keys?: string[]; secrets_untouched?: string[]; restart_required?: string[] } | null>(null);
  const [restoreBundle, setRestoreBundle] = useState<unknown>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const secretUpdates = useRef<Record<string, string | ''>>({});

  const { data: rawView, isLoading, error } = useQuery({
    queryKey: ['settings'],
    queryFn: settingsApi.get,
    enabled: hasRole('admin'),
    retry: false,
  });

  const { data: players } = useQuery({ queryKey: ['players'], queryFn: api.players, enabled: hasRole('admin') });
  const { data: audit } = useQuery({ queryKey: ['settings-audit'], queryFn: settingsApi.audit, enabled: hasRole('admin'), refetchInterval: 30000 });

  const view = rawView as SettingsView | undefined;

  const reload = () => {
    setDrafts({});
    setConflict(false);
    secretUpdates.current = {};
    queryClient.invalidateQueries({ queryKey: ['settings'] });
  };

  const dirty = view !== undefined && Object.keys(drafts).length > 0;

  useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname);
  useEffect(() => {
    const handler = (e: BeforeUnloadEvent) => { if (dirty) e.preventDefault(); };
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [dirty]);

  const saveMutation = useMutation({
    mutationFn: async () => {
      if (!view) throw new Error('settings not loaded');
      const revision = view.revision;
      const savers: Record<string, (body: Record<string, unknown>) => Promise<{ result: SaveResult }>> = {
        sessions: settingsApi.saveSessions,
        access: settingsApi.saveAccess,
        grapevine: settingsApi.saveGrapevine,
        staff_push: settingsApi.saveStaffPush,
        contact: settingsApi.saveContact,
      };
      let last: { result: SaveResult } | undefined;
      let restart: string[] = [];
      for (const group of Object.keys(drafts)) {
        if (!(group in savers)) continue;
        const groupDraft = drafts[group] as Record<string, unknown> | undefined;
        if (!groupDraft) continue;
        const current = (view as unknown as Record<string, unknown>)[group] as Record<string, unknown> | undefined;
        const body: Record<string, unknown> = { ...(current ?? {}), ...groupDraft };
        if (group === 'grapevine' || group === 'staff_push' || group === 'contact') {
          for (const [field, value] of Object.entries(secretUpdates.current)) {
            if (field.startsWith(group + '.')) body[field.slice(group.length + 1)] = value;
          }
        }
        const saved = await savers[group]({ revision, ...body });
        last = saved;
        restart = [...restart, ...(saved.result.restart_required ?? [])];
      }
      if (!last) throw new Error('nothing to save');
      return restart;
    },
    onSuccess: (restart) => {
      setDrafts({});
      secretUpdates.current = {};
      setPendingRestart(restart);
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      showToast(restart.length > 0 ? 'Saved. Some changes need a restart.' : 'Saved. Changes are live.', 'success');
    },
    onError: (err: { status?: number; message?: string }) => {
      if (err.status === 409) {
        setConflict(true);
        showToast('Another administrator changed these settings. Reload before saving.', 'error');
      } else {
        showToast(`Save failed: ${err.message ?? 'unknown error'}`, 'error');
      }
    },
  });

  const restartMutation = useMutation({
    mutationFn: settingsApi.restart,
    onSuccess: () => {
      showToast('Restart requested. The console will reconnect when the server is back.', 'success');
      setTimeout(() => { window.location.reload(); }, 4000);
    },
    onError: (err: Error) => showToast(`Restart failed: ${err.message}`, 'error'),
  });

  const validateRestore = useMutation({
    mutationFn: (bundle: unknown) => settingsApi.restoreValidate(bundle),
    onSuccess: (preview) => { setRestorePreview(preview as never); showToast('Bundle validated. Review the diff before applying.', 'success'); },
    onError: (err: Error) => showToast(`Bundle rejected: ${err.message}`, 'error'),
  });

  const applyRestore = useMutation({
    mutationFn: () => settingsApi.restoreApply(view?.revision ?? 0, restoreBundle),
    onSuccess: (result) => {
      setRestorePreview(null);
      setRestoreBundle(null);
      setPendingRestart((result as { result: SaveResult }).result.restart_required ?? []);
      queryClient.invalidateQueries({ queryKey: ['settings'] });
      showToast('Bundle applied.', 'success');
    },
    onError: (err: Error) => showToast(`Restore failed: ${err.message}`, 'error'),
  });

  const onFile = async (file: File) => {
    const text = await file.text();
    let parsed: unknown;
    try { parsed = JSON.parse(text); } catch { showToast('That file is not valid JSON.', 'error'); return; }
    setRestoreBundle(parsed);
    validateRestore.mutate(parsed);
  };

  if (!hasRole('admin')) return <p className="text-ink-muted">Configuration requires implementor authority.</p>;
  if (isLoading) return <p className="text-ink-muted animate-pulse">Loading settings…</p>;
  if (error || !view) return <p className="text-accent" role="alert">Settings unavailable: {(error as Error | null)?.message ?? 'unknown error'}</p>;

  const sessions = { ...view.sessions, ...((drafts.sessions as object) ?? {}) };
  const access = { ...view.access, ...((drafts.access as object) ?? {}) };
  const grapevine = { ...view.grapevine, ...((drafts.grapevine as object) ?? {}) };
  const staffPush = { ...view.staff_push, ...((drafts.staff_push as object) ?? {}) };
  const contact = { ...view.contact, ...((drafts.contact as object) ?? {}) };
  const setDraft = (group: string, patch: Record<string, unknown>) =>
    setDrafts((prev) => ({ ...prev, [group]: { ...(prev[group] as object ?? {}), ...patch } }));

  const exportDownload = async () => {
    const response = await fetch(settingsApi.exportUrl, { credentials: 'same-origin', headers: { 'X-Requested-With': 'darkpawns-admin' } });
    if (!response.ok) { showToast('Export failed.', 'error'); return; }
    const blob = await response.blob();
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = `darkpawns-settings-r${view.revision}.json`;
    anchor.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-ink">Configuration</h1>
      <p className="text-sm text-ink-muted">Revision {view.revision}. Engine rules stay code (R4); everything here is operator surface. Secrets are stored encrypted and never returned.</p>

      {conflict && (
        <div className="border border-accent bg-accent/5 p-3 text-sm text-ink" role="alert">
          Another administrator changed these settings.{' '}
          <Button variant="outline" size="sm" onClick={reload}>Reload</Button>
        </div>
      )}

      {pendingRestart.length > 0 && (
        <div className="border border-ink bg-paper p-3 text-sm" role="status">
          <p className="mb-2"><strong>Restart required</strong> for: <span className="font-mono">{pendingRestart.join(', ')}</span></p>
          <p className="mb-2 text-ink-muted">{players?.length ?? 0} player(s) online will be disconnected; every character is saved first.</p>
          <Button onClick={() => setRestartOpen(true)}><Icon name="refresh" /> Apply and restart</Button>
        </div>
      )}

      <ConfirmDialog
        open={restartOpen}
        onOpenChange={setRestartOpen}
        title="Apply and restart?"
        description={`The launcher restarts the world gracefully. ${players?.length ?? 0} player(s) online will be disconnected after a forced save.`}
        action="Restart now"
        onConfirm={() => restartMutation.mutate()}
      />

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <GroupCard title="Sessions & limits" subtitle="Applies live: the next connection sees the new limits." envOverridden={view.meta.sessions?.env_overridden ?? undefined}>
          <NumberField label="Login idle timeout (s)" value={sessions.login_idle_timeout_seconds as number} onChange={(v) => setDraft('sessions', { login_idle_timeout_seconds: v })} />
          <NumberField label="Telnet connections (total)" value={sessions.telnet_max_conns as number} onChange={(v) => setDraft('sessions', { telnet_max_conns: v })} />
          <NumberField label="Telnet per address" value={sessions.telnet_max_conns_per_ip as number} onChange={(v) => setDraft('sessions', { telnet_max_conns_per_ip: v })} />
          <NumberField label="WebSocket per address" value={sessions.websocket_max_conns_per_ip as number} onChange={(v) => setDraft('sessions', { websocket_max_conns_per_ip: v })} />
          <NumberField label="WebSocket total" value={sessions.websocket_max_total_conns as number} onChange={(v) => setDraft('sessions', { websocket_max_total_conns: v })} />
          <NumberField label="Account failure budget" hint="0 disables" value={sessions.account_failure_budget as number} onChange={(v) => setDraft('sessions', { account_failure_budget: v })} />
          <NumberField label="Failure window (min)" value={sessions.account_failure_window_minutes as number} onChange={(v) => setDraft('sessions', { account_failure_window_minutes: v })} />
        </GroupCard>

        <GroupCard title="Security & access" subtitle="CORS applies live; trusted proxies apply after a restart." envOverridden={view.meta.access?.env_overridden ?? undefined}>
          <TextField label="Trusted proxies" mono value={(access.trusted_proxies ?? []).join(', ')} placeholder="127.0.0.0/8, ::1/128" onChange={(v) => setDraft('access', { trusted_proxies: v.split(',').map((s) => s.trim()).filter(Boolean) })} />
          <TextField label="CORS origins" mono value={(access.cors_allowed_origins ?? []).join(', ')} placeholder="empty = built-in defaults" onChange={(v) => setDraft('access', { cors_allowed_origins: v.split(',').map((s) => s.trim()).filter(Boolean) })} />
          <TextField label="Admin SPA origin" mono value={access.admin_cors_origin ?? ''} placeholder="empty = built-in default" onChange={(v) => setDraft('access', { admin_cors_origin: v })} />
        </GroupCard>

        <GroupCard title="Grapevine intermud" subtitle="Needs a restart; the relay connects at boot." envOverridden={view.meta.grapevine?.env_overridden ?? undefined}>
          <label className="flex items-center justify-between gap-4 py-1.5">
            <span className="text-sm text-ink">Mode</span>
            <Select value={grapevine.mode as string} onValueChange={(v) => setDraft('grapevine', { mode: v })}>
              <SelectTrigger className="w-36"><SelectValue /></SelectTrigger>
              <SelectContent>
                <SelectItem value="off">off</SelectItem>
                <SelectItem value="presence">presence</SelectItem>
                <SelectItem value="full">full</SelectItem>
              </SelectContent>
            </Select>
          </label>
          <TextField label="Relay URL" mono value={grapevine.url as string} onChange={(v) => setDraft('grapevine', { url: v })} />
          <TextField label="Client ID" mono value={grapevine.client_id as string} onChange={(v) => setDraft('grapevine', { client_id: v })} />
          <SecretField label="Client secret" set={view.grapevine.client_secret_set}
            onSet={(v) => { secretUpdates.current['grapevine.client_secret'] = v; }}
            onClear={() => { secretUpdates.current['grapevine.client_secret'] = ''; }} />
        </GroupCard>

        <GroupCard title="Staff-log push (ntfy)" subtitle="Needs a restart; the pusher starts at boot." envOverridden={view.meta.staff_push?.env_overridden ?? undefined}>
          <TextField label="Topic URL" mono value={staffPush.url as string} placeholder="empty disables push" onChange={(v) => setDraft('staff_push', { url: v })} />
          <NumberField label="Minimum level" value={staffPush.min_level as number} onChange={(v) => setDraft('staff_push', { min_level: v })} />
          <NumberField label="Minimum type" value={staffPush.min_type as number} onChange={(v) => setDraft('staff_push', { min_type: v })} />
          <NumberField label="Rate cap (per minute)" value={staffPush.rate_per_minute as number} onChange={(v) => setDraft('staff_push', { rate_per_minute: v })} />
          <SecretField label="Bearer token" set={view.staff_push.token_set}
            onSet={(v) => { secretUpdates.current['staff_push.token'] = v; }}
            onClear={() => { secretUpdates.current['staff_push.token'] = ''; }} />
        </GroupCard>

        <GroupCard title="Contact form" subtitle={`Needs a restart. ${contact.effective ? 'Delivery is configured.' : 'The form stays disabled until every field is set.'}`} envOverridden={view.meta.contact?.env_overridden ?? undefined}>
          <TextField label="Recipient" mono value={contact.to as string} onChange={(v) => setDraft('contact', { to: v })} />
          <TextField label="SMTP host" mono value={contact.smtp_host as string} onChange={(v) => setDraft('contact', { smtp_host: v })} />
          <NumberField label="SMTP port" value={contact.smtp_port as number} onChange={(v) => setDraft('contact', { smtp_port: v })} />
          <TextField label="SMTP user" mono value={contact.smtp_user as string} onChange={(v) => setDraft('contact', { smtp_user: v })} />
          <TextField label="Turnstile hostnames" mono value={(contact.turnstile_hostnames ?? []).join(', ')} onChange={(v) => setDraft('contact', { turnstile_hostnames: v.split(',').map((s) => s.trim()).filter(Boolean) })} />
          <SecretField label="SMTP password" set={view.contact.smtp_password_set}
            onSet={(v) => { secretUpdates.current['contact.smtp_password'] = v; }}
            onClear={() => { secretUpdates.current['contact.smtp_password'] = ''; }} />
          <SecretField label="Turnstile secret" set={view.contact.turnstile_secret_set}
            onSet={(v) => { secretUpdates.current['contact.turnstile_secret'] = v; }}
            onClear={() => { secretUpdates.current['contact.turnstile_secret'] = ''; }} />
        </GroupCard>

        <GroupCard title="Backup & restore" subtitle="Exports never contain secrets; restoring keeps current secrets.">
          <div className="flex flex-wrap gap-2 py-1">
            <Button variant="outline" onClick={exportDownload}><Icon name="check" /> Download bundle</Button>
            <Button variant="outline" onClick={() => fileRef.current?.click()}>Upload bundle…</Button>
            <input ref={fileRef} type="file" accept="application/json" className="hidden"
              onChange={(e) => { const file = e.target.files?.[0]; if (file) onFile(file); e.target.value = ''; }} />
          </div>
          {restorePreview && (
            <div className="mt-3 border border-rule bg-paper p-3">
              <h3 className="mb-2 text-sm font-semibold">Restore preview</h3>
              {restorePreview.diff.length === 0 ? (
                <p className="text-sm text-ink-muted">No differences from current values.</p>
              ) : (
                <table className="w-full text-sm">
                  <thead><tr className="text-left text-ink-muted"><th className="py-1">Field</th><th>Current</th><th>Incoming</th></tr></thead>
                  <tbody>
                    {restorePreview.diff.map((entry) => (
                      <tr key={entry.field} className="border-t border-rule font-mono">
                        <td className="py-1">{entry.field}</td>
                        <td>{JSON.stringify(entry.current)}</td>
                        <td>{JSON.stringify(entry.incoming)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
              {restorePreview.unknown_keys && restorePreview.unknown_keys.length > 0 && (
                <p className="mt-2 text-xs text-ink-muted">Unknown keys ignored: {restorePreview.unknown_keys.join(', ')}</p>
              )}
              {restorePreview.secrets_untouched && restorePreview.secrets_untouched.length > 0 && (
                <p className="mt-1 text-xs text-ink-muted">Secrets are never exported; existing ones stay: {restorePreview.secrets_untouched.join(', ')}</p>
              )}
              <div className="mt-3 flex gap-2">
                <Button onClick={() => applyRestore.mutate()}>Apply bundle</Button>
                <Button variant="outline" onClick={() => { setRestorePreview(null); setRestoreBundle(null); }}>Discard</Button>
              </div>
            </div>
          )}
        </GroupCard>
      </div>

      <section className="border border-rule bg-paper-deep p-4">
        <h2 className="mb-2 text-base font-semibold text-ink">History</h2>
        {(audit as { timestamp: string; user: string; action: string; details: string; success: boolean }[] | undefined)?.length ? (
          <table className="w-full text-sm">
            <thead><tr className="text-left text-ink-muted"><th className="py-1">When</th><th>Who</th><th>Action</th><th>Detail</th></tr></thead>
            <tbody>
              {(audit as { timestamp: string; user: string; action: string; details: string; success: boolean }[]).slice(0, 20).map((entry, index) => (
                <tr key={index} className="border-t border-rule">
                  <td className="py-1 font-mono text-xs">{new Date(entry.timestamp).toLocaleString()}</td>
                  <td>{entry.user || '(system)'}</td>
                  <td className="font-mono text-xs">{entry.action}</td>
                  <td className="text-ink-muted">{entry.details}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-sm text-ink-muted">No settings changes recorded yet.</p>
        )}
      </section>

      {dirty && (
        <div className="sticky bottom-4 border border-ink bg-paper p-3 shadow-md">
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-ink-muted">Unsaved changes in {Object.keys(drafts).join(', ')}</span>
            <div className="flex gap-2">
              <Button variant="outline" onClick={reload}>Discard</Button>
              <Button onClick={() => saveMutation.mutate()} disabled={saveMutation.isPending}>Save</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
