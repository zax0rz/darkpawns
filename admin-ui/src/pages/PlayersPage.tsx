import { Tabs, TabsList, TabsTrigger, TabsContent } from '../components/ui/tabs';
import { Button } from '../components/ui/button';
import { Input } from '../components/ui/input';
import { NativeSelect } from '../components/ui/select';
import { useEffect, useState } from 'react';
import { Link, useBlocker } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { playerEditor, type SavedPlayer, type SavedPlayerDeletion, type SavedPlayerItem } from '../api/client';
import { useAuth } from '../hooks/useAuth';
import { Switch } from '../components/ui/switch';
import { ConfirmDialog } from '../components/ConfirmDialog';

const fieldClass = 'w-full border border-rule bg-paper px-3 py-2 font-mono text-sm focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50';
const buttonClass = 'border border-rule px-4 py-2 text-sm hover:bg-paper-deep focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50 disabled:cursor-not-allowed';

export function PlayersPage() {
  const { hasRole } = useAuth();
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState('');
  const [dirty, setDirty] = useState(false);
  const [listOpen, setListOpen] = useState(false);
  const [pageNotice, setPageNotice] = useState('');
  const names = useQuery({ queryKey: ['saved-players'], queryFn: () => playerEditor.list(''), enabled: hasRole('admin') });
  const player = useQuery({ queryKey: ['saved-player', selected], queryFn: () => playerEditor.get(selected), enabled: !!selected, refetchOnWindowFocus: false, retry: false });
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname);
  const visible = names.data?.filter((name) => name.toLowerCase().includes(search.toLowerCase())) ?? [];

  useEffect(() => {
    if (!dirty) return;
    const unload = (event: BeforeUnloadEvent) => { event.preventDefault(); };
    window.addEventListener('beforeunload', unload);
    return () => window.removeEventListener('beforeunload', unload);
  }, [dirty]);
  useEffect(() => {
    if (blocker.state !== 'blocked') return;
    if (window.confirm('Discard unsaved character changes?')) blocker.proceed();
    else blocker.reset();
  }, [blocker]);

  if (!hasRole('admin')) return <p role="alert">Player editing requires admin access.</p>;
  return <div className="max-w-6xl mx-auto space-y-6">
    <header className="border-b border-rule pb-5">
      <h1 className="text-3xl">Players</h1>
      <p className="text-ink-muted mt-2 max-w-prose">Edit a character's details, items, flags, or password. The player must be fully disconnected. Admin access required.</p>
    </header>
    <div className="grid gap-6 lg:grid-cols-[15rem_minmax(0,1fr)]">
      <aside aria-label="Saved characters" className="space-y-3">
        {selected && <Button variant="ghost" size="plain" type="button" className={`${buttonClass} lg:hidden`} aria-expanded={listOpen} aria-controls="saved-player-list" onClick={() => setListOpen(!listOpen)}>{listOpen ? 'Close character list' : 'Find another character'}</Button>}
        <div id="saved-player-list" className={`space-y-3 ${selected && !listOpen ? 'hidden lg:block' : ''}`}>
        <label className="block text-sm" htmlFor="player-search">Find a character</label>
        <Input id="player-search" type="search" placeholder="Search by name" value={search} onChange={(event) => setSearch(event.target.value)} className={fieldClass} />
        {names.isLoading && <p role="status" className="animate-pulse text-sm">Loading characters…</p>}
        {names.error && <div role="alert"><p>{names.error.message}</p><Button variant="ghost" size="plain" className={buttonClass} onClick={() => names.refetch()}>Try again</Button></div>}
        {names.data && <p className="text-xs text-ink-muted">{search ? `${visible.length} of ${names.data.length}` : names.data.length} {names.data.length === 1 ? 'character' : 'characters'}</p>}
        <div className="max-h-64 lg:max-h-[65vh] overflow-y-auto border-t border-rule">
          {visible.map((name) => <Button variant="ghost" size="plain" key={name} aria-pressed={name === selected} onClick={() => {
            if (name === selected || (dirty && !window.confirm('Discard unsaved character changes?'))) return;
            setDirty(false); setSelected(name); setListOpen(false);
          }} className={`block w-full text-left px-3 py-3 border-b border-rule focus-visible:outline-2 focus-visible:outline-accent ${name === selected ? 'bg-paper-deep text-accent font-semibold' : 'hover:bg-paper-deep'}`}>{name}</Button>)}
        </div>
        {names.data && !visible.length && <p className="text-sm text-ink-muted">No characters match. Try a different name.</p>}
        </div>
      </aside>
      <section aria-label="Character editor" className="min-w-0">
        {!selected && <div className="py-10 border-y border-rule"><h2 className="text-2xl">Choose a character</h2><p className="text-ink-muted mt-2">Search the list to open their details.</p></div>}
        {selected && player.isLoading && <p role="status" className="animate-pulse py-8">Loading {selected}…</p>}
        {player.error && <div role="alert" className="border-y border-rule py-6 space-y-3"><h2 className="text-xl">Cannot open {selected}</h2><p>{player.error.message}</p><Button variant="ghost" size="plain" className={buttonClass} onClick={() => player.refetch()}>Reload character</Button></div>}
        {player.data && !player.error && <CharacterEditor key={selected} player={player.data} onDirty={setDirty}
          onRenamed={(name) => { setDirty(false); setSelected(name); setListOpen(false); void names.refetch(); }}
          onDeleted={(result) => {
            setDirty(false); setSelected(''); setListOpen(false); void names.refetch();
            setPageNotice(result.marked_deleted
              ? `${result.name} was deleted. ${result.aliases_removed ? 'Saved items and aliases were removed, and the name is available again.' : 'Saved items were removed. The alias file needs attention in the server log.'}${result.clan_cleared ? ' Clan membership was cleared.' : ''}`
              : `${result.name} was deleted. Saved items and aliases were removed, and the name is available again.`);
          }} />}
      </section>
    </div>
    {pageNotice && <p role="status" className="border-y border-rule py-3">{pageNotice}</p>}
  </div>;
}

function CharacterEditor({ player, onDirty, onRenamed, onDeleted }: { player: SavedPlayer; onDirty: (dirty: boolean) => void; onRenamed: (name: string) => void; onDeleted: (result: SavedPlayerDeletion) => void }) {
  const cache = useQueryClient();
  const [tab, setTab] = useState<'stats' | 'items' | 'password' | 'flags' | 'identity'>('stats');
  const [values, setValues] = useState(player.values);
  const [loadedRevision, setLoadedRevision] = useState(player.revision);
  if (loadedRevision !== player.revision) {
    setLoadedRevision(player.revision);
    setValues(player.values);
  }
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [vnum, setVnum] = useState('');
  const [notice, setNotice] = useState('');
  const [actionError, setActionError] = useState('');
  const [newName, setNewName] = useState(player.name);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [confirmName, setConfirmName] = useState('');
  const needsNameConfirmation = player.values.level >= 31;
  const changes = Object.fromEntries(Object.entries(values).filter(([key, value]) => value !== player.values[key]));
  const dirty = Object.keys(changes).length > 0;
  const mutation = useMutation({
    mutationFn: (action: () => Promise<SavedPlayer>) => action(),
    onSuccess: (next) => {
      onDirty(false);
      cache.setQueryData(['saved-player', player.name], next);
    },
    onError: (error: Error) => { setActionError(error.message); setNotice(''); },
  });
  const renameMutation = useMutation({
    mutationFn: () => playerEditor.rename(player.name, player.revision, newName),
    onSuccess: (next) => {
      onDirty(false);
      cache.removeQueries({ queryKey: ['saved-player', player.name] });
      cache.setQueryData(['saved-player', next.name], next);
      void cache.invalidateQueries({ queryKey: ['saved-players'] });
      setNotice(`Renamed ${player.name} to ${next.name}.`); setActionError('');
      onRenamed(next.name);
    },
    onError: (error: Error) => { setActionError(error.message); setNotice(''); },
  });
  const deleteMutation = useMutation({
    mutationFn: () => playerEditor.delete(player.name, player.revision, confirmName),
    onSuccess: (result) => {
      onDirty(false);
      cache.removeQueries({ queryKey: ['saved-player', player.name] });
      void cache.invalidateQueries({ queryKey: ['saved-players'] });
      onDeleted(result);
    },
    onError: (error: Error) => { setActionError(error.message); setNotice(''); },
  });
  const run = (action: () => Promise<SavedPlayer>, success: string) => {
    setActionError(''); setNotice('');
    mutation.mutate(async () => { const next = await action(); setNotice(success); return next; });
  };
  const itemAction = (action: 'grant' | 'move' | 'remove', source: 'inventory' | 'equipment', index: number, locate: number) => {
    run(() => playerEditor.items(player.name, player.revision, { action, source, index, locate, vnum: Number(vnum) || 0 }), 'Equipment and inventory saved.');
  };

  return <Tabs value={tab} onValueChange={(value) => setTab(value as typeof tab)} className="space-y-5" aria-busy={mutation.isPending || renameMutation.isPending || deleteMutation.isPending}>
    <div className="flex flex-wrap gap-3 items-baseline justify-between"><div><h2 className="text-2xl">{player.name}</h2><LastSaved timestamp={player.last_saved_at} /></div><span className="text-sm text-ink-muted">Saved character · Level {player.values.level}</span></div>
    <TabsList className="flex h-auto w-full flex-wrap justify-start" aria-label="Editor sections">
      {(['stats', 'items', 'flags', 'identity', 'password'] as const).map((key) => <TabsTrigger value={key} key={key} disabled={mutation.isPending || renameMutation.isPending || deleteMutation.isPending} className={`px-4 py-3 text-sm border-b-2 focus-visible:outline-2 focus-visible:outline-accent ${tab === key ? 'border-accent text-accent' : 'border-transparent hover:bg-paper-deep'}`}>{key === 'stats' ? 'Character stats' : key === 'items' ? 'Equipment & inventory' : key === 'flags' ? 'Flags' : key === 'identity' ? 'Identity' : 'Password'}</TabsTrigger>)}
    </TabsList>
    {notice && <p role="status" className="border border-rule bg-paper-deep p-3">{notice}</p>}
    {actionError && <div role="alert" className="text-accent border border-accent p-3"><p>{actionError}</p><p className="mt-2 text-sm">If someone changed this character, reload before trying again.</p></div>}
    <TabsContent value="stats"><form onSubmit={(event) => { event.preventDefault(); run(() => playerEditor.stats(player.name, player.revision, changes), 'Character stats saved.'); }} className="space-y-6">
      {['Character', 'Vitals', 'Attributes'].map((group) => <fieldset key={group} disabled={mutation.isPending} className="border-t border-rule pt-3"><legend className="text-lg pr-3">{group}</legend><div className={`grid grid-cols-2 ${group === 'Attributes' ? 'sm:grid-cols-3' : ''} gap-4 mt-2`}>
        {(player.fields ?? []).filter((field) => field.group === group).map((field) => <label key={field.key} className="block text-sm">{field.label}<Input className={`${fieldClass} mt-1`} type="number" required min={field.min} max={field.max} step="1" value={values[field.key]} onChange={(event) => {
          const next = { ...values, [field.key]: event.target.valueAsNumber };
          setValues(next); onDirty(Object.keys(next).some((key) => next[key] !== player.values[key]));
        }} /></label>)}
      </div></fieldset>)}
      <div className="flex flex-wrap gap-3 items-center border-t border-rule pt-4">
        <Button variant="ghost" size="plain" type="submit" disabled={!dirty || mutation.isPending} className={`${buttonClass} bg-accent text-paper hover:bg-accent-deep`}>{mutation.isPending ? 'Saving…' : 'Save stat changes'}</Button>
        <Button variant="ghost" size="plain" type="button" disabled={!dirty || mutation.isPending} className={buttonClass} onClick={() => { setValues(player.values); onDirty(false); }}>Discard changes</Button>
        <span className="text-sm text-ink-muted">{dirty ? `${Object.keys(changes).length} unsaved fields` : 'No unsaved stat changes'}</span>
      </div>
    </form></TabsContent>
    <TabsContent value="items"><div className="space-y-6">
      <p className="text-sm text-ink-muted">Each change saves right away. Containers keep their contents. Equipment slots are staff assignments; choose a suitable item for each slot. The save setting is shared by every copy of an item.</p>
      {dirty && <p role="status" className="text-accent">Save or discard stat changes before editing items.</p>}
      <fieldset disabled={mutation.isPending || dirty} className="space-y-6">
        <form onSubmit={(event) => { event.preventDefault(); itemAction('grant', 'inventory', 0, 0); }} className="flex flex-wrap gap-3 items-end border-b border-rule pb-5"><label className="text-sm">Item number<Input type="number" min="1" step="1" required value={vnum} onChange={(event) => setVnum(event.target.value)} className={`${fieldClass} mt-1 max-w-48`} /></label><Button variant="ghost" size="plain" className={buttonClass}>Add item</Button></form>
        {(['equipment', 'inventory'] as const).map((source) => <section key={source}><h3 className="text-xl mb-3 capitalize">{source}</h3>
          {!(player[source] ?? []).length ? <p className="text-sm text-ink-muted">{source === 'equipment' ? 'Nothing equipped. Move an item into a slot below.' : 'No items carried. Add one by item number above.'}</p> : <ul className="border-t border-rule">{(player[source] ?? []).map((item) => <ItemRow key={item.index} item={item} player={player} source={source} onAction={(action, locate) => itemAction(action, source, item.index, locate)} />)}</ul>}
        </section>)}
      </fieldset>
    </div></TabsContent>
    <TabsContent value="flags"><div className="space-y-5">
      <p className="text-sm text-ink-muted max-w-prose">Each switch saves right away. Your staff level controls which switches you can change.</p>
      <ul className="divide-y divide-rule border-y border-rule">
        {(player.flags ?? []).map((flag) => <li key={flag.key} className="flex flex-wrap items-center justify-between gap-4 py-3">
          <div className="max-w-prose"><p className="font-medium">{flag.label}<span className="ml-2 text-xs text-ink-muted">Level {flag.min_level}+</span></p><p className="text-sm text-ink-muted mt-1">{flag.description}</p></div>
          <label className="flex min-h-11 items-center gap-3 text-sm"><span>{flag.enabled ? 'On' : 'Off'}<span className="sr-only"> {flag.label}</span></span><Switch checked={flag.enabled} disabled={!flag.editable || mutation.isPending || dirty} aria-label={flag.label} onCheckedChange={(enabled) => run(() => playerEditor.flags(player.name, player.revision, { [flag.key]: enabled }), `${flag.label} ${enabled ? 'enabled' : 'disabled'}.`)} /></label>
        </li>)}
      </ul>
      <p className="text-sm text-ink-muted max-w-prose">Item save settings are changed on the Items tab and affect every copy.</p>
      {dirty && <p role="status" className="text-accent">Save or discard stat changes before editing flags.</p>}
    </div></TabsContent>
    <TabsContent value="identity"><div className="space-y-8">
      <section className="max-w-xl space-y-3">
        <h3 className="text-xl">Rename character</h3>
        <p className="text-sm text-ink-muted">Changes the character's name everywhere it's used: aliases, saved items, board posts. Mail, clan and house access stay with the character. The player must be offline.</p>
        <form className="flex flex-wrap items-end gap-3" onSubmit={(event) => { event.preventDefault(); setActionError(''); setNotice(''); renameMutation.mutate(); }}>
          <label htmlFor="player-new-name" className="block flex-1 text-sm">New name<Input id="player-new-name" autoComplete="off" required minLength={2} maxLength={20} pattern="[A-Za-z]{2,20}" value={newName} onChange={(event) => setNewName(event.target.value)} className={`${fieldClass} mt-1`} /></label>
          <Button variant="ghost" size="plain" type="submit" disabled={dirty || renameMutation.isPending || newName === player.name || !newName.trim()} className={`${buttonClass} bg-accent text-paper hover:bg-accent-deep`}>{renameMutation.isPending ? 'Renaming…' : 'Rename character'}</Button>
        </form>
        {dirty && <p role="status" className="text-accent">Save or discard stat changes before renaming.</p>}
      </section>
      <section className="max-w-xl border-t border-rule pt-5 space-y-3">
        <h3 className="text-xl">Delete character</h3>
        <p className="text-sm text-ink-muted">Deletes this character. Their saved items and aliases are removed, and the name becomes available again. This can't be undone.</p>
        {dirty && <p role="status" className="text-accent">Save or discard stat changes before deleting.</p>}
        <Button variant="ghost" size="plain" type="button" disabled={dirty || deleteMutation.isPending || renameMutation.isPending} className={`${buttonClass} text-accent border-accent`} onClick={() => { setConfirmName(''); setDeleteOpen(true); }}>Delete character…</Button>
      </section>
      <ConfirmDialog open={deleteOpen} onOpenChange={setDeleteOpen} title={`Delete ${player.name}?`}
        description={needsNameConfirmation ? `Deletes this character. Their saved items and aliases are removed, and the name becomes available again. This can't be undone. Type ${player.name} to confirm.` : 'Deletes this character. Their saved items and aliases are removed, and the name becomes available again. This can\'t be undone.'}
        action={deleteMutation.isPending ? 'Deleting…' : 'Delete character'} disabled={needsNameConfirmation && confirmName !== player.name} onConfirm={() => deleteMutation.mutate()}>
        {needsNameConfirmation && <label className="block space-y-1 text-sm">Type {player.name}<Input autoComplete="off" value={confirmName} onChange={(event) => setConfirmName(event.target.value)} className={fieldClass} /></label>}
      </ConfirmDialog>
    </div></TabsContent>
    <TabsContent value="password"><form className="space-y-4 max-w-md" onSubmit={(event) => {
      event.preventDefault();
      if (password !== confirmPassword) { setActionError('Passwords do not match.'); return; }
      const secret = password;
      run(() => playerEditor.password(player.name, player.revision, secret), 'Password reset. Login lockout cleared.');
      setPassword(''); setConfirmPassword('');
    }}>
      <h3 className="text-xl">Reset password</h3><p className="text-sm text-ink-muted">Set a new password from 8 to 72 characters. This also clears failed login attempts and the account lockout.</p>
      {dirty && <p role="status" className="text-accent">Save or discard stat changes before resetting the password.</p>}
      <label className="block text-sm">New password<Input type="password" autoComplete="new-password" minLength={8} maxLength={72} required value={password} onChange={(event) => setPassword(event.target.value)} className={`${fieldClass} mt-1`} /></label>
      <label className="block text-sm">Confirm new password<Input type="password" autoComplete="new-password" minLength={8} maxLength={72} required value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} className={`${fieldClass} mt-1`} /></label>
      <Button variant="ghost" size="plain" disabled={dirty || mutation.isPending} className={`${buttonClass} bg-accent text-paper hover:bg-accent-deep`}>{mutation.isPending ? 'Resetting…' : 'Reset password'}</Button>
    </form></TabsContent>
    <Button variant="ghost" size="plain" type="button" disabled={mutation.isPending} className={`${buttonClass} text-ink-muted`} onClick={() => {
    if (dirty && !window.confirm('Discard character changes and reload?')) return;
      setValues(player.values); onDirty(false); cache.invalidateQueries({ queryKey: ['saved-player', player.name] });
    }}>Reload character</Button>
  </Tabs>;
}

function ItemRow({ item, player, source, onAction }: { item: SavedPlayerItem; player: SavedPlayer; source: 'inventory' | 'equipment'; onAction: (action: 'move' | 'remove', locate: number) => void }) {
  const [locate, setLocate] = useState(0);
  return <li className="border-b border-rule py-4 space-y-3">
    <div className="flex flex-wrap justify-between gap-2"><div><span className="font-medium">{item.name}</span><span className="font-mono text-xs text-ink-muted ml-2">#{item.vnum}{item.count > 1 ? ` × ${item.count}` : ''}</span><p className="text-xs text-ink-muted mt-1">Kept after logout: {item.no_rent ? 'No' : 'Yes'}</p>{item.vnum > 0 && <Link className="text-xs text-accent underline underline-offset-2" to={`/admin/game/objects/${item.vnum}/edit`}>Change setting for all copies</Link>}</div><span className="text-sm text-ink-muted">{item.parent ? 'Inside container' : source === 'equipment' ? (player.slots ?? []).find((slot) => slot.locate === item.locate)?.name ?? 'Unknown slot' : 'Carried'}</span></div>
    <div className="flex flex-wrap gap-2"><NativeSelect aria-label={`Destination for ${item.name} ${item.index + 1}`} className={`${fieldClass} sm:max-w-52`} value={locate} onChange={(event) => setLocate(Number(event.target.value))}><option value={0}>Inventory</option>{(player.slots ?? []).map((slot) => <option key={slot.locate} value={slot.locate}>{slot.name}</option>)}</NativeSelect><Button variant="ghost" size="plain" type="button" className={buttonClass} onClick={() => onAction('move', locate)}>Move item</Button><Button variant="ghost" size="plain" type="button" className={`${buttonClass} text-accent`} onClick={() => { if (window.confirm(`Remove ${item.name} and anything inside it from ${player.name}?`)) onAction('remove', 0); }}>Remove</Button></div>
  </li>;
}

function LastSaved({ timestamp }: { timestamp: string | null }) {
  const [now, setNow] = useState(Date.now);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 60000);
    return () => window.clearInterval(timer);
  }, []);
  if (!timestamp) return <p className="text-sm text-ink-muted mt-1">Last saved: unknown</p>;
  const date = new Date(timestamp);
  const elapsed = Math.max(0, (now - date.getTime()) / 1000);
  const formatter = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' });
  const [seconds, unit]: [number, Intl.RelativeTimeFormatUnit] = elapsed < 60 ? [1, 'second'] : elapsed < 3600 ? [60, 'minute'] : elapsed < 86400 ? [3600, 'hour'] : elapsed < 2592000 ? [86400, 'day'] : elapsed < 31536000 ? [2592000, 'month'] : [31536000, 'year'];
  const relative = elapsed < 60 ? 'just now' : formatter.format(-Math.floor(elapsed / seconds), unit);
  return <p className="text-sm text-ink-muted mt-1">Last saved <time dateTime={timestamp} title={date.toLocaleString()}>{relative}</time></p>;
}
