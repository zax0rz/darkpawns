import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { useBlocker } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Activity, MessageSquare, Radio, Smartphone, type LucideIcon } from 'lucide-react';
import { notifications, type NotificationChannel, type NotificationConfig, type NotificationDestination, type NotificationStatus, type NotificationView } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../hooks/useToast';
import { Button } from '../components/ui/button';
import { Card, CardAction, CardContent, CardHeader } from '../components/ui/card';
import { Switch } from '../components/ui/switch';
import { ChannelCredentials } from '../components/NotificationCredentials';
import { Skeleton } from '../components/Skeleton';

// Operator reference for the channels this server can configure.
const DOCS_URL = 'https://darkpawns.org/docs/server/notifications/';

const channelNames: Record<NotificationChannel, string> = {
  discord: 'Discord alerts',
  pushover: 'Pushover',
  gossip: 'Gossip mirror',
  heartbeat: 'Healthchecks heartbeat',
};

const testLabels: Record<NotificationChannel, string> = {
  discord: 'Send test notification',
  pushover: 'Send test notification',
  gossip: 'Send a test line',
  heartbeat: 'Send a check-in now',
};

// A character name and, for level gains, the new level. The example is the whole
// payload, so an operator sees what a public channel would receive.
const events = [
  { id: 'player.created', name: 'Character created', example: 'Avery created a character.' },
  { id: 'player.login', name: 'Login', example: 'Avery entered the game.' },
  { id: 'player.logout', name: 'Logout', example: 'Avery left the game.' },
  { id: 'player.death', name: 'Death', example: 'Avery died.' },
  { id: 'player.level', name: 'Level gained', example: 'Avery reached level 20.' },
];

function sameEvents(left: string[] | null, right: string[] | null) {
  return JSON.stringify(left ?? []) === JSON.stringify(right ?? []);
}

// Which channels the save bar is about to change, named the way the blocks are.
function unsavedChannels(config: NotificationConfig, saved: NotificationConfig) {
  return [
    config.discord.enabled !== saved.discord.enabled || !sameEvents(config.discord.events, saved.discord.events) ? channelNames.discord : '',
    config.pushover.enabled !== saved.pushover.enabled || !sameEvents(config.pushover.events, saved.pushover.events) ? channelNames.pushover : '',
    config.gossip !== saved.gossip || config.gossip_notice !== saved.gossip_notice ? channelNames.gossip : '',
    config.heartbeat !== saved.heartbeat ? channelNames.heartbeat : '',
  ].filter(Boolean);
}

export function NotificationsPage() {
  const { hasRole } = useAuth();
  const query = useQuery({ queryKey: ['notifications'], queryFn: notifications.get, enabled: hasRole('admin'), retry: false, refetchInterval: 5000, refetchOnWindowFocus: false });
  return <div className="mx-auto max-w-5xl space-y-6">
    <header className="border-b border-rule pb-5">
      <h1 className="text-3xl">Notifications</h1>
      <p className="mt-2 max-w-prose text-ink-muted">Every outbound channel in one place. Each block holds its own credentials, event selection and test send, and nothing leaves the server until a channel is enabled.</p>
    </header>
    {!hasRole('admin') ? <p role="alert">Notification settings require HIGOD access.</p> : query.isLoading ? <Skeleton className="h-96 w-full" /> : query.error && !query.data ? <div role="alert"><p>{query.error.message}</p><Button variant="outline" className="mt-3" onClick={() => query.refetch()}>Try again</Button></div> : query.data && <NotificationSettings initial={query.data} latest={query.data} refreshError={query.error?.message} />}
  </div>;
}

function NotificationSettings({ initial, latest, refreshError }: { initial: NotificationView; latest: NotificationView; refreshError?: string }) {
  const { showToast } = useToast();
  const [config, setConfig] = useState(initial.config);
  const [saved, setSaved] = useState(initial.config);
  const [secretDirty, setSecretDirty] = useState<Record<string, boolean>>({});
  const [credentialBusy, setCredentialBusy] = useState(false);
  const [credentialReset, setCredentialReset] = useState(0);
  const onCredentialDirty = useCallback((channel: NotificationChannel, value: boolean) => setSecretDirty((previous) => previous[channel] === value ? previous : { ...previous, [channel]: value }), []);
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
    setConfig(view.config); setSaved(view.config); cache.setQueryData(['notifications'], view);
    showToast('Channel settings saved.', 'success');
  } });
  const onCredentialSaved = (view: NotificationView, channel: NotificationChannel, removed: boolean) => {
    setSaved(view.config);
    setConfig((current) => {
      const next: NotificationConfig = { ...current, revision: view.config.revision };
      // Removing a credential disables that channel server-side, so the draft
      // follows and the switches never disagree with the saved configuration.
      if (removed) {
        if (channel === 'heartbeat') next.heartbeat = false;
        else if (channel === 'gossip') next.gossip = false;
        else next[channel] = { ...next[channel], enabled: false };
      }
      return next;
    });
    onCredentialDirty(channel, false);
    cache.setQueryData(['notifications'], view);
  };
  const credentials = (channel: NotificationChannel) => <ChannelCredentials key={`${channel}-${credentialReset}`} channel={channel} revision={saved.revision} status={latest.status[channel]} busy={save.isPending || credentialBusy} onSaved={onCredentialSaved} onDirty={onCredentialDirty} onBusy={setCredentialBusy} />;
  const setDestination = (channel: 'discord' | 'pushover', patch: Partial<NotificationDestination>) => setConfig({ ...config, [channel]: { ...config[channel], ...patch } });
  const stale = latest.config.revision !== saved.revision;
  const unsaved = unsavedChannels(config, saved);
  const controlsDisabled = save.isPending || credentialBusy;
  const discard = () => { setConfig(saved); setCredentialReset((value) => value + 1); setSecretDirty({}); save.reset(); showToast('Changes discarded.', 'info'); };
  const reload = () => {
    if (dirty && !window.confirm('Discard your changes and load the latest settings?')) return;
    setConfig(latest.config); setSaved(latest.config); setCredentialReset((value) => value + 1); setSecretDirty({}); save.reset();
    showToast('Latest settings loaded.', 'info');
  };
  return <div className="space-y-6">
    {refreshError && <p role="alert" className="text-sm text-accent">Delivery status could not refresh: {refreshError}</p>}
    <div className="sticky -top-4 z-20 -mx-4 border-b border-rule bg-paper px-4 py-3 md:-top-6 md:-mx-6 md:px-6">
      <div className="flex flex-wrap items-center gap-3">
        <Button type="button" disabled={!settingsDirty || stale || controlsDisabled} onClick={() => save.mutate(config)}>{save.isPending ? 'Saving…' : 'Save settings'}</Button>
        <Button type="button" variant="outline" disabled={!dirty || controlsDisabled} onClick={discard}>Discard changes</Button>
        <p className="text-sm text-ink-muted">{settingsDirty ? `Unsaved: ${unsaved.join(', ')}` : dirty ? `Unsaved credentials: ${Object.entries(secretDirty).filter(([, value]) => value).map(([channel]) => channelNames[channel as NotificationChannel]).join(', ')}` : 'All settings saved'}</p>
      </div>
      {save.error && <p role="alert" className="mt-2 text-sm text-accent">{save.error.message}</p>}
    </div>
    {stale && <div role="alert" className="border border-accent bg-paper-deep p-4">
      <p className="text-sm">Another administrator changed these settings. Reload before saving, or your save is refused.</p>
      <Button type="button" variant="outline" className="mt-3" onClick={reload}>Reload settings</Button>
    </div>}
    <ChannelBlock channel="discord" icon={MessageSquare} description="Posts the selected character events to a Discord channel webhook. Point it at a staff channel or a public one, whichever the events belong in." enabled={config.discord.enabled} onEnabled={(value) => setDestination('discord', { enabled: value })} status={latest.status.discord} controlsDisabled={controlsDisabled}>
      <Group title="Events">
        <p className="max-w-prose text-sm text-ink-muted">Guests and invisible staff are excluded. A message carries a character name and, for level gains, the new level: no IP address, room detail or raw log line.</p>
        <EventSelection channel="discord" selected={config.discord.events ?? []} disabled={controlsDisabled} onChange={(list) => setDestination('discord', { events: list })} />
      </Group>
      {credentials('discord')}
      <Delivery channel="discord" status={latest.status.discord} disabled={controlsDisabled} />
    </ChannelBlock>
    <ChannelBlock channel="pushover" icon={Smartphone} description="Pushes the selected character events to a phone or a group through Pushover. Use it for the events you want in your pocket rather than in a channel." enabled={config.pushover.enabled} onEnabled={(value) => setDestination('pushover', { enabled: value })} status={latest.status.pushover} controlsDisabled={controlsDisabled}>
      <Group title="Events">
        <p className="max-w-prose text-sm text-ink-muted">The same five observations the Discord alerts use, selected independently. A message carries a character name and, for level gains, the new level.</p>
        <EventSelection channel="pushover" selected={config.pushover.events ?? []} disabled={controlsDisabled} onChange={(list) => setDestination('pushover', { events: list })} />
      </Group>
      {credentials('pushover')}
      <Delivery channel="pushover" status={latest.status.pushover} disabled={controlsDisabled} />
    </ChannelBlock>
    <ChannelBlock channel="gossip" icon={Radio} description="Mirrors the public gossip channel to its own Discord webhook. Names appear as a level one mortal would see them, and nothing said in Discord returns to the game." enabled={config.gossip} onEnabled={(value) => setConfig({ ...config, gossip: value })} status={latest.status.gossip} controlsDisabled={controlsDisabled}>
      <Group title="Mirroring">
        <label className="flex items-start gap-3 text-sm">
          <Switch className="mt-0.5" checked={config.gossip_notice} disabled={controlsDisabled} onCheckedChange={(value) => setConfig({ ...config, gossip_notice: value })} aria-label="Show the automatic gossip notice in the MOTD" />
          <span><span className="block">Show the automatic gossip notice in the MOTD</span><span className="mt-1 block max-w-prose text-ink-muted">On while mirroring is active: “What's said on gossip echoes to our Discord.” Turn it off if your own MOTD already discloses the mirror.</span></span>
        </label>
        <p className="max-w-prose text-sm text-ink-muted">Every public gossip line is mirrored. There is no event selection here: the mirror carries what players say, not the activity events above.</p>
      </Group>
      {credentials('gossip')}
      <Delivery channel="gossip" status={latest.status.gossip} disabled={controlsDisabled} />
    </ChannelBlock>
    <ChannelBlock channel="heartbeat" icon={Activity} description="Checks in with Healthchecks once a minute while the game loop is progressing, and stops checking in when it stalls or the process dies." enabled={config.heartbeat} onEnabled={(value) => setConfig({ ...config, heartbeat: value })} status={latest.status.heartbeat} controlsDisabled={controlsDisabled}>
      <Group title="Check-in">
        <p className="max-w-prose text-sm text-ink-muted"><a className="text-accent underline underline-offset-2" href="https://healthchecks.io/" target="_blank" rel="noreferrer">Healthchecks.io</a> alerts you when this server stops checking in.</p>
      </Group>
      {credentials('heartbeat')}
      <Delivery channel="heartbeat" status={latest.status.heartbeat} disabled={controlsDisabled} />
    </ChannelBlock>
    <p className="max-w-prose text-sm text-ink-muted">Tests use the saved credentials and run even while a channel is disabled. Delivery is best effort: nothing is replayed after an outage or restart, and anything a queue or rate limit skipped is counted under that channel. <a className="text-accent underline underline-offset-2" href={DOCS_URL}>Notification reference</a></p>
  </div>;
}


// Every channel block has the same shape: a ruled header that names the channel
// and owns its switch, then that channel's own controls, its write-only
// credentials, and the delivery evidence its test send produces.
function ChannelBlock({ channel, icon: Glyph, description, enabled, onEnabled, status, controlsDisabled, children }: {
  channel: NotificationChannel;
  icon: LucideIcon;
  description: string;
  enabled: boolean;
  onEnabled: (value: boolean) => void;
  status: NotificationStatus;
  controlsDisabled: boolean;
  children: ReactNode;
}) {
  const name = channelNames[channel];
  return <Card className="gap-0 py-0">
    <CardHeader className="border-b border-rule bg-paper-deep/50 py-4">
      <h2 className="flex items-center gap-2 font-display text-lg">
        <Glyph className="size-4 shrink-0 text-ink-muted" strokeWidth={1.5} aria-hidden="true" />
        {name}
      </h2>
      <p className="max-w-prose text-sm text-ink-muted">{description}</p>
      <CardAction className="flex flex-col items-start gap-1.5 sm:items-end">
        <label className="flex cursor-pointer items-center gap-2 text-sm">
          <Switch checked={enabled} disabled={controlsDisabled || !status.ready} onCheckedChange={onEnabled} aria-label={`Enable ${name}`} />
          <span>{enabled ? 'Enabled' : 'Disabled'}</span>
        </label>
        <p className={status.ready ? 'text-sm text-ink-muted' : 'text-sm text-accent'}>{status.ready ? (enabled ? 'Delivering' : 'Ready, paused') : 'Setup needed'}</p>
      </CardAction>
    </CardHeader>
    <CardContent className="space-y-5 py-5">{children}</CardContent>
  </Card>;
}

function Group({ title, children }: { title: string; children: ReactNode }) {
  return <section className="space-y-3">
    <h3 className="text-sm font-semibold">{title}</h3>
    {children}
  </section>;
}

function EventSelection({ channel, selected, disabled, onChange }: { channel: 'discord' | 'pushover'; selected: string[]; disabled: boolean; onChange: (events: string[]) => void }) {
  return <ul className="divide-y divide-rule border-y border-rule">
    {events.map((item) => <li key={item.id}>
      <label className="flex cursor-pointer items-start gap-3 py-2.5">
        <input className="mt-0.5 size-4 shrink-0 accent-accent focus-visible:outline-2 focus-visible:outline-accent" type="checkbox" checked={selected.includes(item.id)} disabled={disabled} onChange={(event) => onChange(event.target.checked ? [...selected, item.id] : selected.filter((id) => id !== item.id))} />
        <span className="min-w-0">
          <span className="block text-sm">{item.name}</span>
          <span className="block text-sm text-ink-muted">{item.example}</span>
        </span>
        <span className="sr-only">to {channelNames[channel]}</span>
      </label>
    </li>)}
  </ul>;
}
// The delivery group answers "is it working": the provider's last answer, what
// the queue dropped, and the button that produces the next one.
function Delivery({ channel, status, disabled }: { channel: NotificationChannel; status: NotificationStatus; disabled: boolean }) {
  return <Group title="Delivery">
    <dl className="grid gap-x-6 gap-y-1.5 text-sm sm:grid-cols-[9rem_minmax(0,1fr)]">
      <dt className="text-ink-muted">Last accepted</dt>
      <dd>{status.last_success ? <time className="font-mono text-xs" dateTime={status.last_success}>{new Date(status.last_success).toLocaleString()}</time> : 'Nothing delivered this session'}</dd>
      {status.last_failure && <>
        <dt className="text-ink-muted">Last failure</dt>
        <dd className={status.last_error ? 'text-accent' : undefined}><time className="font-mono text-xs" dateTime={status.last_failure}>{new Date(status.last_failure).toLocaleString()}</time>{status.last_error ? `, ${status.last_error}` : ', recovered'}</dd>
      </>}
      {status.dropped > 0 && <>
        <dt className="text-ink-muted">Skipped</dt>
        <dd>{status.dropped} by queue or rate limit</dd>
      </>}
    </dl>
    <TestSend channel={channel} ready={status.ready} disabled={disabled} />
  </Group>;
}

function TestSend({ channel, ready, disabled }: { channel: NotificationChannel; ready: boolean; disabled: boolean }) {
  const [notice, setNotice] = useState('');
  const test = useMutation({ mutationFn: () => notifications.test(channel), onSuccess: () => setNotice('Queued. The provider’s answer appears above once it arrives.') });
  return <div className="flex flex-wrap items-center gap-3">
    <Button type="button" variant="outline" size="sm" disabled={!ready || disabled || test.isPending} onClick={() => { setNotice(''); test.mutate(); }}>{test.isPending ? 'Queuing…' : testLabels[channel]}</Button>
    {!ready && <p className="text-sm text-ink-muted">Save credentials before testing this channel.</p>}
    {test.error && <p role="alert" className="text-sm text-accent">{test.error.message}</p>}
    {notice && <p role="status" className="text-sm text-ink-muted">{notice}</p>}
  </div>;
}
