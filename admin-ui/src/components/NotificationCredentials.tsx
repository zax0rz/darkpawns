import { useEffect, useState } from 'react';
import { ApiError, notifications, type NotificationChannel, type NotificationCredentials, type NotificationStatus, type NotificationView } from '../api/client';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { ConfirmDialog } from './ConfirmDialog';
import { useToast } from '../hooks/useToast';

// One row of provider truth per channel, so the console never asks for a secret
// the destination does not use and the help text names the right dashboard.
const channels: Record<NotificationChannel, {
  label: string;
  fields: { key: keyof NotificationCredentials; label: string; maxLength: number }[];
  help: string;
  removal: string;
}> = {
  pushover: {
    label: 'Pushover',
    fields: [
      { key: 'token', label: 'Application API token', maxLength: 30 },
      { key: 'user', label: 'User or group key', maxLength: 30 },
    ],
    help: 'Use the token from your Pushover application and the user or group key that should receive alerts.',
    removal: 'Remove the Pushover credentials and stop sending events to this destination? Event selections are kept.',
  },
  discord: {
    label: 'Discord',
    fields: [{ key: 'url', label: 'Channel webhook URL', maxLength: 2000 }],
    help: 'Paste the channel webhook URL from Discord, with no query string.',
    removal: 'Remove the Discord credentials and stop posting events to this channel? Event selections are kept.',
  },
  gossip: {
    label: 'Discord gossip',
    fields: [{ key: 'url', label: 'Gossip channel webhook URL', maxLength: 2000 }],
    help: 'Paste the webhook URL for the channel that should carry public gossip. Use discord.com with no query string.',
    removal: 'Remove the gossip credentials and stop mirroring? Lines already queued are discarded.',
  },
  heartbeat: {
    label: 'Healthchecks',
    fields: [{ key: 'url', label: 'Ping URL', maxLength: 2000 }],
    help: 'Paste the hosted Healthchecks ping URL beginning with https://hc-ping.com/.',
    removal: 'Remove the Healthchecks ping URL and stop the check-in?',
  },
};

// Credentials are write-only: the server never returns a saved secret, so this
// panel reports only whether one exists and where it came from. Values live in
// this component's state until the request is sent, and are cleared after.
export function ChannelCredentials({ channel, revision, status, busy, onSaved, onDirty, onBusy }: {
  channel: NotificationChannel;
  revision: number;
  status: NotificationStatus;
  busy: boolean;
  onSaved: (view: NotificationView, channel: NotificationChannel, removed: boolean) => void;
  onDirty: (channel: NotificationChannel, dirty: boolean) => void;
  onBusy: (busy: boolean) => void;
}) {
  const meta = channels[channel];
  const { showToast } = useToast();
  const [editing, setEditing] = useState(false);
  const [values, setValues] = useState<NotificationCredentials>({});
  const [confirming, setConfirming] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState('');
  const dirty = Object.values(values).some(Boolean);
  useEffect(() => {
    onDirty(channel, dirty);
    return () => onDirty(channel, false);
  }, [channel, dirty, onDirty]);
  const close = () => { setValues({}); setEditing(false); setConfirming(false); setError(''); onDirty(channel, false); };
  // Never display a transport body that might contain a secret. Ordinary
  // validation failures also use a generic write-only API response.
  const failureText = (failure: unknown) => {
    if (failure instanceof ApiError && failure.status === 409) return 'Settings changed elsewhere. Reload settings, then try again.';
    if (failure instanceof ApiError && failure.status === 503) return 'Credential storage is unavailable. Check the server encryption key and storage permissions.';
    return 'Could not save credentials. Check the credential format and try again.';
  };
  // Deliberately not a mutation with variables cached by React Query: secrets
  // live only in the current inputs/request, and are cleared after saving.
  const submit = async (remove: boolean) => {
    setPending(true); onBusy(true); setError('');
    try {
      const view = remove ? await notifications.removeCredentials(channel, revision) : await notifications.credentials(channel, revision, values);
      close(); onSaved(view, channel, remove);
      showToast(remove ? `${meta.label} credentials removed. The channel is disabled.` : `${meta.label} credentials saved. Changes take effect immediately.`, 'success');
    } catch (failure) {
      setError(failureText(failure));
    } finally { setPending(false); onBusy(false); }
  };
  const showForm = !status.ready || editing || dirty;
  return <div className="space-y-3">
    <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
      <h3 className="text-sm font-semibold">Credentials</h3>
      <p className="text-sm">{status.ready
        ? <><span className="font-semibold">Configured</span> <span className="font-mono" aria-label="The saved credential is hidden and never displayed">••••••••</span></>
        : <span className="text-ink-muted">Not configured</span>}</p>
    </div>
    {status.ready && status.source === 'environment' && <p className="max-w-prose text-sm text-ink-muted">Using the server environment credentials. Save your own values here to manage this channel in the console.</p>}
    {showForm && <form className="max-w-xl space-y-3" onSubmit={(event) => { event.preventDefault(); void submit(false); }}>
      {meta.fields.map((field) => <label className="block text-sm" key={field.key}>
        {field.label}
        <Input className="mt-1" type="password" inputMode={field.key === 'url' ? 'url' : 'text'} name={`notification-${channel}-${field.key}`} autoComplete="new-password" autoCapitalize="none" spellCheck={false} maxLength={field.maxLength} required value={values[field.key] ?? ''} disabled={busy || pending} onChange={(event) => { setError(''); setValues({ ...values, [field.key]: event.target.value.trim() }); }} />
      </label>)}
      <p className="max-w-prose text-sm text-ink-muted">{meta.help}</p>
      <div className="flex flex-wrap gap-3">
        <Button type="submit" size="sm" disabled={busy || pending || !dirty}>{pending ? 'Saving…' : 'Save credentials'}</Button>
        {(editing || dirty) && <Button type="button" size="sm" variant="outline" disabled={busy || pending} onClick={close}>{status.ready ? 'Cancel' : 'Clear fields'}</Button>}
      </div>
    </form>}
    {status.ready && !editing && !dirty && <div className="flex flex-wrap gap-3">
      <Button type="button" size="sm" variant="outline" disabled={busy || pending} onClick={() => { setError(''); setEditing(true); }}>Replace credentials</Button>
      <Button type="button" size="sm" variant="outline" disabled={busy || pending} onClick={() => { setError(''); setConfirming(true); }}>Remove credentials</Button>
    </div>}
    {error && <p role="alert" className="max-w-prose text-sm text-accent">{error}</p>}
    <ConfirmDialog open={confirming} onOpenChange={setConfirming} title={`Remove ${meta.label} credentials?`} description={meta.removal} action="Remove credentials" onConfirm={() => void submit(true)} />
  </div>;
}
