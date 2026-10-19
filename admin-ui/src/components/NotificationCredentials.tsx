import { useEffect, useState } from 'react';
import { ApiError, notifications, type CredentialProvider, type NotificationCredentials, type NotificationStatus, type NotificationView } from '../api/client';

const button = 'border border-rule px-4 py-2 text-sm hover:bg-paper-deep focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50 disabled:cursor-not-allowed';
const field = 'block mt-2 w-full border border-rule bg-paper px-3 py-2 text-sm focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50';
const names = { pushover: 'Pushover', discord: 'Discord', heartbeat: 'Healthchecks' };

export function NotificationCredentialsForm({ provider, revision, status, busy, onSaved, onDirty, onBusy }: {
  provider: CredentialProvider;
  revision: number;
  status: NotificationStatus;
  busy: boolean;
  onSaved: (view: NotificationView, provider: CredentialProvider, removed: boolean) => void;
  onDirty: (provider: CredentialProvider, dirty: boolean) => void;
  onBusy: (busy: boolean) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [values, setValues] = useState<NotificationCredentials>({});
  const [removing, setRemoving] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const dirty = Object.values(values).some(Boolean);
  useEffect(() => {
    onDirty(provider, dirty);
    return () => onDirty(provider, false);
  }, [provider, dirty, onDirty]);
  const close = () => { setValues({}); setEditing(false); setRemoving(false); setError(''); onDirty(provider, false); };
  // Deliberately not a mutation with variables cached by React Query: secrets
  // live only in the current inputs/request, and are cleared after saving.
  const submit = async (remove: boolean) => {
    setPending(true); onBusy(true); setError(''); setNotice('');
    try {
      const view = remove ? await notifications.removeCredentials(provider, revision) : await notifications.credentials(provider, revision, values);
      close(); onSaved(view, provider, remove);
      setNotice(remove ? `${names[provider]} credentials removed. This destination is disabled.` : `${names[provider]} credentials saved. Changes take effect immediately.`);
    } catch (failure) {
      // Never display a transport body that might contain a secret. Ordinary
      // validation failures also use a generic write-only API response.
      setError(failure instanceof ApiError && failure.status === 409 ? 'Settings changed elsewhere. Reload settings, then try again.' : failure instanceof ApiError && failure.status === 503 ? 'Credential storage is unavailable. Check the server encryption key and storage permissions.' : 'Could not save credentials. Check the credential format and try again.');
    } finally { setPending(false); onBusy(false); }
  };
  const input = (key: keyof NotificationCredentials, label: string, maxLength: number) => <label className="block text-sm" key={key}>
    {label}
    <input className={field} type="password" inputMode={key === 'url' ? 'url' : 'text'} name={`notification-${provider}-${key}`} autoComplete="new-password" autoCapitalize="none" spellCheck={false} maxLength={maxLength} required value={values[key] ?? ''} disabled={busy || pending} onChange={(event) => { setError(''); setValues({ ...values, [key]: event.target.value.trim() }); }} />
  </label>;
  return <div className="space-y-3">
    <p className="text-sm">{status.ready ? <><span className="font-semibold">Configured</span> <span className="font-mono" aria-label="Saved credentials are hidden">••••••••</span></> : 'Not configured'}</p>
    {status.ready && status.source === 'environment' && <p className="text-sm text-ink-muted">Using server environment credentials. Replace them here to manage this destination in admin.</p>}
    {(!status.ready || editing || dirty) && <form className="space-y-4 max-w-xl" onSubmit={(event) => { event.preventDefault(); void submit(false); }}>
      {provider === 'pushover' ? <>{input('token', 'Application API token', 30)}{input('user', 'User or group key', 30)}<p className="text-sm text-ink-muted">Use the token from your Pushover application and the user or group key that should receive alerts.</p></> : <>{input('url', provider === 'discord' ? 'Webhook URL' : 'Ping URL', 2000)}<p className="text-sm text-ink-muted">{provider === 'discord' ? 'Paste the channel webhook URL from Discord. Use discord.com without a query string.' : 'Paste the hosted Healthchecks ping URL beginning with https://hc-ping.com/.'}</p></>}
      <div className="flex flex-wrap gap-3"><button className={button} type="submit" disabled={busy || pending || !dirty}>{pending ? 'Saving…' : `Save ${names[provider]} credentials`}</button>{(editing || dirty) && <button type="button" className={button} disabled={busy || pending} onClick={close}>{status.ready ? 'Cancel' : 'Clear fields'}</button>}</div>
    </form>}
    {status.ready && !editing && !removing && !dirty && <div className="flex flex-wrap gap-3">
      <button type="button" className={button} disabled={busy || pending} onClick={() => { setNotice(''); setEditing(true); }}>Replace {names[provider]} credentials</button>
      <button type="button" className={button} disabled={busy || pending} onClick={() => { setNotice(''); setRemoving(true); }}>Remove {names[provider]} credentials</button>
    </div>}
    {removing && <div className="border-y border-rule py-3 space-y-3"><p className="text-sm">Remove {names[provider]} credentials and disable this destination? Event selections will be kept.</p><div className="flex flex-wrap gap-3"><button type="button" className={`${button} text-accent`} disabled={busy || pending} onClick={() => void submit(true)}>{pending ? 'Removing…' : `Confirm remove ${names[provider]}`}</button><button type="button" className={button} disabled={busy || pending} onClick={close}>Cancel</button></div></div>}
    {error && <p role="alert" className="text-sm text-accent">{error}</p>}
    {notice && <p role="status" className="text-sm">{notice}</p>}
  </div>;
}
