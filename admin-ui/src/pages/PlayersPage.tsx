import { useEffect, useState } from 'react';
import { useBlocker } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { playerEditor, type SavedPlayer, type SavedPlayerItem } from '../api/client';
import { useAuth } from '../hooks/useAuth';

const fieldClass = 'w-full border border-rule bg-paper px-3 py-2 font-mono text-sm focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50';
const buttonClass = 'border border-rule px-4 py-2 text-sm hover:bg-paper-deep focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50 disabled:cursor-not-allowed';

export function PlayersPage() {
  const { hasRole } = useAuth();
  const [search, setSearch] = useState('');
  const [selected, setSelected] = useState('');
  const [dirty, setDirty] = useState(false);
  const [listOpen, setListOpen] = useState(false);
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
    if (window.confirm('Discard unsaved stat changes?')) blocker.proceed();
    else blocker.reset();
  }, [blocker]);

  if (!hasRole('admin')) return <p role="alert">Player editing requires HIGOD access.</p>;
  return <div className="max-w-6xl mx-auto space-y-6">
    <header className="border-b border-rule pb-5">
      <h1 className="text-3xl">Players</h1>
      <p className="text-ink-muted mt-2 max-w-prose">Edit saved characters, manage their equipment, and reset passwords. Players must be fully disconnected. HIGOD access required.</p>
    </header>
    <div className="grid gap-6 lg:grid-cols-[15rem_minmax(0,1fr)]">
      <aside aria-label="Saved characters" className="space-y-3">
        {selected && <button type="button" className={`${buttonClass} lg:hidden`} aria-expanded={listOpen} aria-controls="saved-player-list" onClick={() => setListOpen(!listOpen)}>{listOpen ? 'Close character list' : 'Find another character'}</button>}
        <div id="saved-player-list" className={`space-y-3 ${selected && !listOpen ? 'hidden lg:block' : ''}`}>
        <label className="block text-sm" htmlFor="player-search">Find a character</label>
        <input id="player-search" type="search" placeholder="Search by name" value={search} onChange={(event) => setSearch(event.target.value)} className={fieldClass} />
        {names.isLoading && <p role="status" className="animate-pulse text-sm">Loading characters…</p>}
        {names.error && <div role="alert"><p>{names.error.message}</p><button className={buttonClass} onClick={() => names.refetch()}>Try again</button></div>}
        {names.data && <p className="text-xs text-ink-muted">{visible.length} matching {visible.length === 1 ? 'character' : 'characters'}</p>}
        <div className="max-h-64 lg:max-h-[65vh] overflow-y-auto border-t border-rule">
          {visible.map((name) => <button key={name} aria-pressed={name === selected} onClick={() => {
            if (name === selected || (dirty && !window.confirm('Discard unsaved stat changes?'))) return;
            setDirty(false); setSelected(name); setListOpen(false);
          }} className={`block w-full text-left px-3 py-3 border-b border-rule focus-visible:outline-2 focus-visible:outline-accent ${name === selected ? 'bg-paper-deep text-accent font-semibold' : 'hover:bg-paper-deep'}`}>{name}</button>)}
        </div>
        {names.data && !visible.length && <p className="text-sm text-ink-muted">No characters match. Try a different name.</p>}
        </div>
      </aside>
      <section aria-label="Character editor" className="min-w-0">
        {!selected && <div className="py-10 border-y border-rule"><h2 className="text-2xl">Choose a character</h2><p className="text-ink-muted mt-2">Search the saved-player list to open their record.</p></div>}
        {selected && player.isLoading && <p role="status" className="animate-pulse py-8">Loading {selected}…</p>}
        {player.error && <div role="alert" className="border-y border-rule py-6 space-y-3"><h2 className="text-xl">Cannot open {selected}</h2><p>{player.error.message}</p><button className={buttonClass} onClick={() => player.refetch()}>Reload character</button></div>}
        {player.data && !player.error && <CharacterEditor key={selected} player={player.data} onDirty={setDirty} />}
      </section>
    </div>
  </div>;
}

function CharacterEditor({ player, onDirty }: { player: SavedPlayer; onDirty: (dirty: boolean) => void }) {
  const cache = useQueryClient();
  const [tab, setTab] = useState<'stats' | 'items' | 'password'>('stats');
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
  const run = (action: () => Promise<SavedPlayer>, success: string) => {
    setActionError(''); setNotice('');
    mutation.mutate(async () => { const next = await action(); setNotice(success); return next; });
  };
  const itemAction = (action: 'grant' | 'move' | 'remove', source: 'inventory' | 'equipment', index: number, locate: number) => {
    run(() => playerEditor.items(player.name, player.revision, { action, source, index, locate, vnum: Number(vnum) || 0 }), 'Equipment and inventory saved.');
  };

  return <div className="space-y-5" aria-busy={mutation.isPending}>
    <div className="flex flex-wrap gap-3 items-baseline justify-between"><h2 className="text-2xl">{player.name}</h2><span className="text-sm text-ink-muted">Saved character · Level {player.values.level}</span></div>
    <div className="flex flex-wrap gap-1 border-b border-rule" aria-label="Editor sections">
      {(['stats', 'items', 'password'] as const).map((key) => <button key={key} type="button" aria-pressed={tab === key} disabled={mutation.isPending} onClick={() => setTab(key)} className={`px-4 py-3 text-sm border-b-2 focus-visible:outline-2 focus-visible:outline-accent ${tab === key ? 'border-accent text-accent' : 'border-transparent hover:bg-paper-deep'}`}>{key === 'stats' ? 'Character stats' : key === 'items' ? 'Equipment & inventory' : 'Password'}</button>)}
    </div>
    {notice && <p role="status" className="border border-rule bg-paper-deep p-3">{notice}</p>}
    {actionError && <div role="alert" className="text-accent border border-accent p-3"><p>{actionError}</p><p className="mt-2 text-sm">Stat drafts stay here; password fields are cleared after submission. If the record changed, reload before trying again.</p></div>}
    {tab === 'stats' && <form onSubmit={(event) => { event.preventDefault(); run(() => playerEditor.stats(player.name, player.revision, changes), 'Character stats saved.'); }} className="space-y-6">
      {['Character', 'Vitals', 'Attributes'].map((group) => <fieldset key={group} disabled={mutation.isPending} className="border-t border-rule pt-3"><legend className="text-lg pr-3">{group}</legend><div className={`grid grid-cols-2 ${group === 'Attributes' ? 'sm:grid-cols-3' : ''} gap-4 mt-2`}>
        {player.fields.filter((field) => field.group === group).map((field) => <label key={field.key} className="block text-sm">{field.label}<input className={`${fieldClass} mt-1`} type="number" required min={field.min} max={field.max} step="1" value={values[field.key]} onChange={(event) => {
          const next = { ...values, [field.key]: event.target.valueAsNumber };
          setValues(next); onDirty(Object.keys(next).some((key) => next[key] !== player.values[key]));
        }} /></label>)}
      </div></fieldset>)}
      <div className="flex flex-wrap gap-3 items-center border-t border-rule pt-4">
        <button type="submit" disabled={!dirty || mutation.isPending} className={`${buttonClass} bg-accent text-paper hover:bg-accent-deep`}>{mutation.isPending ? 'Saving…' : 'Save stat changes'}</button>
        <button type="button" disabled={!dirty || mutation.isPending} className={buttonClass} onClick={() => { setValues(player.values); onDirty(false); }}>Discard changes</button>
        <span className="text-sm text-ink-muted">{dirty ? `${Object.keys(changes).length} unsaved fields` : 'No unsaved stat changes'}</span>
      </div>
    </form>}
    {tab === 'items' && <div className="space-y-6">
      <p className="text-sm text-ink-muted">Each action saves immediately. Containers travel with their contents. Equipment slots are staff assignments; choose a suitable object for the slot.</p>
      {dirty && <p role="status" className="text-accent">Save or discard stat changes before editing items.</p>}
      <fieldset disabled={mutation.isPending || dirty} className="space-y-6">
        <form onSubmit={(event) => { event.preventDefault(); itemAction('grant', 'inventory', 0, 0); }} className="flex flex-wrap gap-3 items-end border-b border-rule pb-5"><label className="text-sm">Object vnum<input type="number" min="1" step="1" required value={vnum} onChange={(event) => setVnum(event.target.value)} className={`${fieldClass} mt-1 max-w-48`} /></label><button className={buttonClass}>Grant to inventory</button></form>
        {(['equipment', 'inventory'] as const).map((source) => <section key={source}><h3 className="text-xl mb-3 capitalize">{source}</h3>
          {!player[source].length ? <p className="text-sm text-ink-muted">{source === 'equipment' ? 'Nothing equipped. Move an inventory item into a slot below.' : 'Inventory is empty. Grant an object by vnum above.'}</p> : <ul className="border-t border-rule">{player[source].map((item) => <ItemRow key={item.index} item={item} player={player} source={source} onAction={(action, locate) => itemAction(action, source, item.index, locate)} />)}</ul>}
        </section>)}
      </fieldset>
    </div>}
    {tab === 'password' && <form className="space-y-4 max-w-md" onSubmit={(event) => {
      event.preventDefault();
      if (password !== confirmPassword) { setActionError('Passwords do not match.'); return; }
      const secret = password;
      run(() => playerEditor.password(player.name, player.revision, secret), 'Password reset. Login lockout cleared.');
      setPassword(''); setConfirmPassword('');
    }}>
      <h3 className="text-xl">Reset password</h3><p className="text-sm text-ink-muted">Set a new password of 8 to 72 bytes. This also clears failed login attempts and the account lockout.</p>
      {dirty && <p role="status" className="text-accent">Save or discard stat changes before resetting the password.</p>}
      <label className="block text-sm">New password<input type="password" autoComplete="new-password" minLength={8} maxLength={72} required value={password} onChange={(event) => setPassword(event.target.value)} className={`${fieldClass} mt-1`} /></label>
      <label className="block text-sm">Confirm new password<input type="password" autoComplete="new-password" minLength={8} maxLength={72} required value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} className={`${fieldClass} mt-1`} /></label>
      <button disabled={dirty || mutation.isPending} className={`${buttonClass} bg-accent text-paper hover:bg-accent-deep`}>{mutation.isPending ? 'Resetting…' : 'Reset password'}</button>
    </form>}
    <button type="button" disabled={mutation.isPending} className={`${buttonClass} text-ink-muted`} onClick={() => {
      if (dirty && !window.confirm('Discard stat changes and reload this character?')) return;
      setValues(player.values); onDirty(false); cache.invalidateQueries({ queryKey: ['saved-player', player.name] });
    }}>Reload saved record</button>
  </div>;
}

function ItemRow({ item, player, source, onAction }: { item: SavedPlayerItem; player: SavedPlayer; source: 'inventory' | 'equipment'; onAction: (action: 'move' | 'remove', locate: number) => void }) {
  const [locate, setLocate] = useState(0);
  return <li className="border-b border-rule py-4 space-y-3">
    <div className="flex flex-wrap justify-between gap-2"><div><span className="font-medium">{item.name}</span><span className="font-mono text-xs text-ink-muted ml-2">#{item.vnum}{item.count > 1 ? ` × ${item.count}` : ''}</span></div><span className="text-sm text-ink-muted">{item.parent ? 'Inside container' : source === 'equipment' ? player.slots.find((slot) => slot.locate === item.locate)?.name ?? 'Unknown slot' : 'Carried'}</span></div>
    <div className="flex flex-wrap gap-2"><select aria-label={`Destination for ${item.name} ${item.index + 1}`} className={`${fieldClass} sm:max-w-52`} value={locate} onChange={(event) => setLocate(Number(event.target.value))}><option value={0}>Inventory</option>{player.slots.map((slot) => <option key={slot.locate} value={slot.locate}>{slot.name}</option>)}</select><button type="button" className={buttonClass} onClick={() => onAction('move', locate)}>Move item</button><button type="button" className={`${buttonClass} text-accent`} onClick={() => { if (window.confirm(`Remove ${item.name} and anything inside it from ${player.name}?`)) onAction('remove', 0); }}>Remove</button></div>
  </li>;
}
