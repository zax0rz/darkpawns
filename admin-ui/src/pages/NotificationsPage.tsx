import { useCallback, useEffect, useState } from 'react';
import { useBlocker } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { notifications, type NotificationConfig, type NotificationProvider, type NotificationStatus, type NotificationView, type CredentialProvider } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { NotificationCredentialsForm } from '../components/NotificationCredentials';
import { Skeleton } from '../components/Skeleton';

const button = 'border border-rule px-4 py-2 text-sm hover:bg-paper-deep focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50 disabled:cursor-not-allowed';
const checkbox = 'size-5 shrink-0 accent-accent focus-visible:outline-2 focus-visible:outline-accent';
const providers: NotificationProvider[] = ['pushover', 'discord'];
const names = { pushover: 'Pushover', discord: 'Discord', heartbeat: 'Healthchecks', gossip: 'Discord gossip' };
const events = [
  { id: 'player.created', name: 'Character created', description: 'After a new character is saved.', example: 'Avery created a character.' },
  { id: 'player.login', name: 'Login', description: 'When a character enters the world.', example: 'Avery entered the game.' },
  { id: 'player.logout', name: 'Logout', description: 'When a character leaves the world.', example: 'Avery left the game.' },
  { id: 'player.death', name: 'Death', description: 'Combat and environmental deaths.', example: 'Avery died.' },
  { id: 'player.level', name: 'Level gained', description: 'Each level gained through play.', example: 'Avery reached level 20.' },
];

export function NotificationsPage() {
  const { hasRole } = useAuth();
  const query = useQuery({ queryKey: ['notifications'], queryFn: notifications.get, enabled: hasRole('admin'), retry: false, refetchInterval: 5000, refetchOnWindowFocus: false });
  return <div className="max-w-5xl mx-auto space-y-6">
    <header className="border-b border-rule pb-5">
      <h1 className="text-3xl">Notifications</h1>
      <p className="mt-2 max-w-prose text-ink-muted">Choose which game events reach each destination. Subscriptions start off.</p>
    </header>
    {!hasRole('admin') ? <p role="alert">Notification settings require HIGOD access.</p> : query.isLoading ? <Skeleton className="h-96 w-full" /> : query.error && !query.data ? <div role="alert"><p>{query.error.message}</p><button className={`${button} mt-3`} onClick={() => query.refetch()}>Try again</button></div> : query.data && <NotificationSettings initial={query.data} latest={query.data} refreshError={query.error?.message} />}
  </div>;
}

function NotificationSettings({ initial, latest, refreshError }: { initial: NotificationView; latest: NotificationView; refreshError?: string }) {
  const [config, setConfig] = useState(initial.config);
  const [saved, setSaved] = useState(initial.config);
  const [notice, setNotice] = useState('');
  const [secretDirty, setSecretDirty] = useState<Record<string, boolean>>({});
  const [credentialBusy, setCredentialBusy] = useState(false);
  const [credentialReset, setCredentialReset] = useState(0);
  const onCredentialDirty = useCallback((provider: CredentialProvider, value: boolean) => setSecretDirty((previous) => previous[provider] === value ? previous : { ...previous, [provider]: value }), []);
  const cache = useQueryClient();
  const settingsDirty = JSON.stringify(config) !== JSON.stringify(saved);
  const dirty = settingsDirty || Object.values(secretDirty).some(Boolean);
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname);
  useEffect(() => {
    if (!dirty) return;
    const handler = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener('beforeunload', handler);
    return () => window.removeEventListener('beforeunload', handler);
  }, [dirty]);
  useEffect(() => {
    if (blocker.state !== 'blocked') return;
    if (window.confirm('Discard unsaved notification settings?')) blocker.proceed();
    else blocker.reset();
  }, [blocker]);
  const save = useMutation({ mutationFn: notifications.save, onSuccess: (view) => {
    setConfig(view.config); setSaved(view.config); setNotice('Notification settings saved.');
    cache.setQueryData(['notifications'], view);
  } });
  const onCredentialSaved = (view: NotificationView, provider: CredentialProvider, removed: boolean) => {
    setSaved(view.config);
    setConfig((current) => ({ ...current, revision: view.config.revision, ...(removed ? provider === 'heartbeat' ? { heartbeat: false } : provider === 'gossip' ? { gossip: false } : { [provider]: { ...current[provider], enabled: false } } : {}) }));
    onCredentialDirty(provider, false);
    cache.setQueryData(['notifications'], view);
  };
  const credentialForm = (provider: CredentialProvider) => <NotificationCredentialsForm key={`${provider}-${credentialReset}`} provider={provider} revision={saved.revision} status={latest.status[provider]} busy={save.isPending || credentialBusy} onSaved={onCredentialSaved} onDirty={onCredentialDirty} onBusy={setCredentialBusy} />;
  const update = (provider: NotificationProvider, patch: Partial<NotificationConfig[NotificationProvider]>) => {
    setNotice(''); setConfig({ ...config, [provider]: { ...config[provider], ...patch } });
  };
  const stale = latest.config.revision !== saved.revision;
  return <div className="space-y-8">
    {refreshError && <p role="alert" className="text-accent">Delivery status could not refresh: {refreshError}</p>}
    {stale && <div role="alert" className="border-y border-rule py-3"><p>Another administrator changed these settings. Reload before saving.</p><button className={`${button} mt-2`} onClick={() => { if (!dirty || window.confirm('Discard your changes and load the latest settings?')) { setConfig(latest.config); setSaved(latest.config); save.reset(); setNotice('Latest settings loaded.'); } }}>Reload settings</button></div>}
    <form onSubmit={(event) => { event.preventDefault(); save.mutate(config); }} className="space-y-8">
      <section aria-labelledby="events-heading">
        <h2 id="events-heading" className="text-xl mb-2">Event subscriptions</h2>
        <p className="text-sm text-ink-muted mb-4 max-w-prose">Guests and invisible staff are excluded. HIGOD access required.</p>
        <div role="group" aria-label="Events by destination">
          <div className="grid grid-cols-[minmax(0,1fr)_5.5rem_5.5rem] sm:grid-cols-[minmax(0,1fr)_9rem_9rem] border-y border-rule bg-paper-deep py-3">
            <p className="self-center pl-3 text-sm font-semibold">Send event to</p>
            {providers.map((provider) => <div key={provider} className="text-center space-y-2">
              <p className="font-semibold">{names[provider]}</p>
              {!latest.status[provider].ready && <a href="#destinations-heading" className="text-xs underline text-accent">Setup needed</a>}
              <label className="flex justify-center items-center gap-2 text-sm cursor-pointer min-h-11"><input className={checkbox} type="checkbox" checked={config[provider].enabled} disabled={save.isPending || credentialBusy} onChange={(event) => update(provider, { enabled: event.target.checked })} /><span>Enable<span className="sr-only"> {names[provider]}</span></span></label>
            </div>)}
          </div>
          {events.map((item) => <div key={item.id} className="grid grid-cols-[minmax(0,1fr)_5.5rem_5.5rem] sm:grid-cols-[minmax(0,1fr)_9rem_9rem] border-b border-rule py-2">
            <div className="pl-3 pr-2"><p className="font-semibold">{item.name}</p><p className="text-sm text-ink-muted mt-1">{item.description}</p></div>
            {providers.map((provider) => <label key={provider} className="flex items-center justify-center cursor-pointer"><span className="sr-only">{item.name} to {names[provider]}</span><input className={checkbox} type="checkbox" checked={config[provider].events.includes(item.id)} disabled={save.isPending || credentialBusy} onChange={(event) => update(provider, { events: event.target.checked ? [...config[provider].events, item.id] : config[provider].events.filter((id) => id !== item.id) })} /></label>)}
          </div>)}
        </div>
        <details className="mt-3 text-sm">
          <summary className="cursor-pointer underline underline-offset-2 focus-visible:outline-2 focus-visible:outline-accent">Message previews & privacy</summary>
          <p className="mt-3 text-ink-muted max-w-prose">Messages contain a character name and, for level gains, the new level. No IP addresses, room details, or raw logs are sent.</p>
          <ul className="list-disc pl-5 mt-3 space-y-2">{events.map((item) => <li key={item.id}><span className="font-semibold">{item.name}:</span> {item.example}</li>)}</ul>
        </details>
      </section>
      <section aria-labelledby="gossip-heading" className="border-y border-rule py-5 space-y-3">
        <h2 id="gossip-heading" className="text-xl">Discord gossip</h2>
        <p className="text-sm text-ink-muted max-w-prose">Mirror public gossip to a separate Discord #gossip webhook. Names appear as a level-1 mortal would see them. Discord messages do not return to the game.</p>
        <label className="flex items-center gap-3 min-h-11 text-sm"><input className={checkbox} type="checkbox" checked={config.gossip ?? false} disabled={save.isPending || credentialBusy} onChange={(event) => { setNotice(''); setConfig({ ...config, gossip: event.target.checked }); }} />Enable gossip mirroring</label>
        {!latest.status.gossip.ready && <p className="text-sm text-ink-muted">Configure the Discord gossip destination below before enabling.</p>}
      </section>
      <section aria-labelledby="heartbeat-heading" className="border-y border-rule py-5">
        <h2 id="heartbeat-heading" className="text-xl">External monitoring</h2>
        <label className="flex items-start gap-3 mt-4 cursor-pointer"><input className={`${checkbox} mt-0.5`} type="checkbox" checked={config.heartbeat} disabled={save.isPending || credentialBusy} onChange={(event) => { setNotice(''); setConfig({ ...config, heartbeat: event.target.checked }); }} /><span><span className="font-semibold">Enable Healthchecks heartbeat</span><span className="block text-sm text-ink-muted mt-1 max-w-prose">Check in every minute while the game loop is progressing. Configure outage and recovery alerts in Healthchecks so they can reach you when this server is down.</span></span></label>
        <p className="mt-3 text-sm"><a href="#heartbeat-credentials" className="underline underline-offset-2">{latest.status.heartbeat.ready ? 'Manage Healthchecks credentials' : 'Set up Healthchecks'}</a></p>
        <DeliveryStatus status={latest.status.heartbeat} />
      </section>
      <div className="sticky -bottom-4 md:-bottom-6 z-10 bg-paper border-t border-rule pt-3 pb-4 md:pb-6">
      <div className="flex flex-wrap items-center gap-3">
        <button className={`${button} bg-accent text-paper hover:bg-accent-deep`} disabled={!settingsDirty || stale || save.isPending || credentialBusy} type="submit">{save.isPending ? 'Saving…' : 'Save settings'}</button>
        <button className={button} type="button" disabled={!dirty || save.isPending || credentialBusy} onClick={() => { setConfig(saved); setCredentialReset((value) => value + 1); setSecretDirty({}); setNotice('Changes discarded.'); save.reset(); }}>Discard changes</button>
        <span className="text-sm text-ink-muted">{settingsDirty ? 'Unsaved settings' : dirty ? `Unsaved credentials: ${Object.entries(secretDirty).filter(([, value]) => value).map(([provider]) => names[provider as CredentialProvider]).join(', ')}` : 'All changes saved'}</span>
      </div>
      {save.error && <p role="alert" className="text-accent">{save.error.message}</p>}
      {notice && <p role="status" className="mt-2">{notice}</p>}
      </div>
    </form>
    <section aria-labelledby="destinations-heading" className="space-y-5">
      <div><h2 id="destinations-heading" className="text-xl">Destinations & delivery</h2><p className="mt-2 text-sm text-ink-muted max-w-prose">Paste credentials here to connect a destination. Saved values are encrypted and hidden; changes take effect immediately. Tests work even when a destination is disabled. Use Discord for a community channel or a private staff channel.</p></div>
      {providers.map((provider) => <DestinationDelivery key={provider} provider={provider} status={latest.status[provider]} dirty={Boolean(secretDirty[provider])} busy={save.isPending || credentialBusy} credentials={credentialForm(provider)} />)}
      <div className="border-t border-rule pt-5 space-y-3"><h3 className="text-lg">Discord gossip</h3>{credentialForm('gossip')}<DeliveryStatus status={latest.status.gossip} /></div>
      <div id="heartbeat-credentials" className="border-t border-rule pt-5 space-y-3"><h3 className="text-lg">Healthchecks</h3>{credentialForm('heartbeat')}</div>
      <p className="text-sm text-ink-muted max-w-prose">Delivery is best effort. Events are not replayed after an outage or restart. Queues and rate limits protect the game; skipped messages are counted below each destination.</p>
    </section>
  </div>;
}

function DestinationDelivery({ provider, status, dirty, busy, credentials }: { provider: NotificationProvider; status: NotificationStatus; dirty: boolean; busy: boolean; credentials: React.ReactNode }) {
  const [notice, setNotice] = useState('');
  const test = useMutation({ mutationFn: () => notifications.test(provider), onSuccess: () => setNotice('Test queued. Check the destination and delivery status for the result.') });
  return <div className="border-t border-rule pt-5 space-y-3">
    <div className="flex flex-wrap items-center justify-between gap-3"><h3 className="text-lg">{names[provider]}</h3><span className={`text-sm ${status.ready ? 'text-ink' : 'text-accent'}`}>{status.ready ? 'Credentials configured' : 'Setup needed'}</span></div>
    {credentials}
    <DeliveryStatus status={status} />
    <button className={button} type="button" disabled={!status.ready || test.isPending || dirty || busy} onClick={() => { setNotice(''); test.mutate(); }}>{test.isPending ? 'Queuing…' : `Send ${names[provider]} test`}</button>
    {dirty && <p className="text-sm text-ink-muted">Save or discard this destination’s credentials before sending a test.</p>}
    {test.error && <p role="alert" className="text-accent">{test.error.message}</p>}
    {notice && <p role="status" className="text-sm">{notice}</p>}
  </div>;
}
function DeliveryStatus({ status }: { status: NotificationStatus }) {
  return <div className="text-sm space-y-1 mt-3">
    <p>Last accepted by provider: {status.last_success ? <time dateTime={status.last_success}>{new Date(status.last_success).toLocaleString()}</time> : 'No successful delivery this session'}</p>
    {status.last_failure && <p className={status.last_error ? 'text-accent' : 'text-ink-muted'}>Last failure: {new Date(status.last_failure).toLocaleString()}{status.last_error ? `: ${status.last_error}` : ': recovered'}</p>}
    {status.dropped > 0 && <p className="text-ink-muted">Messages dropped: {status.dropped}</p>}
  </div>;
}
