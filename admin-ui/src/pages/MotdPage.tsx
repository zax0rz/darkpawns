import { useEffect, useRef, useState } from 'react';
import { useBlocker } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { fileEditApi, type EditableFile } from '../api/fileedit';
import { ApiError } from '../api/client';
import { Input } from '../components/ui/input';
import { Button } from '../components/ui/button';
import { NativeSelect } from '../components/ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../components/ui/tabs';
import { MotdPreview } from '../components/MotdPreview';
import { Skeleton } from '../components/Skeleton';
import { useToast } from '../hooks/useToast';
import { motdBytes, discordSnippet, gossipNotice } from '../lib/motd';
import { lineDiff } from '../lib/lineDiff';

type MotdFile = 'motd' | 'imotd';
type Document = { file: EditableFile; draft: string };
const names = { motd: 'MOTD', imotd: 'Immortal MOTD' };

export function MotdPage() {
  const [selected, setSelected] = useState<MotdFile>('motd');
  const [documents, setDocuments] = useState<Partial<Record<MotdFile, Document>>>({});
  const [saving, setSaving] = useState<MotdFile | null>(null);
  const [error, setError] = useState('');
  const [conflict, setConflict] = useState<{ path: MotdFile; server: EditableFile | null } | null>(null);
  const [columns, setColumns] = useState(80);
  const [colour, setColour] = useState(true);
  const [showInvite, setShowInvite] = useState(false);
  const [invite, setInvite] = useState('');
  const [inviteError, setInviteError] = useState('');
  const [insertion, setInsertion] = useState<{ start: number; end: number } | null>(null);
  const editor = useRef<HTMLTextAreaElement>(null);
  const client = useQueryClient();
  const { showToast } = useToast();
  const listing = useQuery({ queryKey: ['file-listing', 'tedit', ''], queryFn: () => fileEditApi.listing('tedit'), retry: false });
  const file = useQuery({ queryKey: ['editable-file', 'tedit', selected], queryFn: () => fileEditApi.read('tedit', selected), retry: false, refetchOnWindowFocus: false });
  // Keep each tab's draft and original ETag until the operator saves or reloads.
  if (file.data && !documents[selected]) setDocuments(prev => ({ ...prev, [selected]: { file: file.data!, draft: file.data!.content } }));
  const dirty = Object.values(documents).some(doc => doc && doc.draft !== doc.file.content);
  const blocker = useBlocker(({ currentLocation, nextLocation }) => dirty && currentLocation.pathname !== nextLocation.pathname);
  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [dirty]);
  useEffect(() => {
    if (blocker.state !== 'blocked') return;
    if (window.confirm('Discard unsaved MOTD changes?')) blocker.proceed(); else blocker.reset();
  }, [blocker]);
  const doc = documents[selected];
  const entry = listing.data?.entries?.find(e => e.path === selected);
  const allowed = entry?.allowed ?? false;
  const bytes = doc ? motdBytes(doc.draft) : 0;
  const limit = entry?.max_bytes ?? 2400;
  function change(draft: string) {
    setDocuments(prev => ({ ...prev, [selected]: { ...prev[selected]!, draft } }));
  }
  function insert(snippet: string, selection?: { start: number; end: number } | null) {
    if (!doc || !allowed) return;
    const start = selection?.start ?? editor.current?.selectionStart ?? doc.draft.length;
    const end = selection?.end ?? editor.current?.selectionEnd ?? start;
    change(doc.draft.slice(0, start) + snippet + doc.draft.slice(end));
    requestAnimationFrame(() => { editor.current?.focus(); editor.current?.setSelectionRange(start + snippet.length, start + snippet.length); });
  }
  async function save() {
    if (!doc || !allowed || saving || conflict?.path === selected) return;
    const path = selected;
    setSaving(path); setError('');
    try {
      const saved = await fileEditApi.save('tedit', path, doc.draft, doc.file.etag);
      // Preserve typing done while the save was in flight.
      setDocuments(prev => ({ ...prev, [path]: { file: saved, draft: prev[path]?.draft === doc.draft ? saved.content : prev[path]!.draft } }));
      client.setQueryData(['editable-file', 'tedit', path], saved);
      showToast(`${names[path]} saved. Players will see it at their next login.`, 'success');
    } catch (err) {
      if (err instanceof ApiError && err.status === 412) {
        const server = await fileEditApi.read('tedit', path).catch(() => null);
        setConflict({ path, server });
      } else setError("Couldn't save. Your draft is still here. Check your connection and access, then try again.");
    } finally { setSaving(null); }
  }
  async function reload() {
    if (doc && doc.draft !== doc.file.content && !window.confirm('Discard this draft and load the saved MOTD?')) return;
    try {
      const saved = await fileEditApi.read('tedit', selected);
      setDocuments(prev => ({ ...prev, [selected]: { file: saved, draft: saved.content } }));
      client.setQueryData(['editable-file', 'tedit', selected], saved);
      setConflict(null); setError('');
    } catch { setError("Couldn't load the saved MOTD. Check your connection, then try again."); }
  }
  const difference = conflict?.path === selected && conflict.server && doc ? lineDiff(conflict.server.content, doc.draft) : null;
  return <div className="space-y-5">
    <header className="border-b border-rule pb-4"><h1 className="text-2xl">MOTD</h1><p className="mt-2 max-w-prose text-sm text-ink-muted">The first words players see when they sign in. Edit either welcome message and check how it looks before saving.</p></header>
    <Tabs value={selected} onValueChange={value => { setSelected(value as MotdFile); setError(''); setShowInvite(false); setInviteError(''); }}>
      <TabsList variant="line" aria-label="Welcome messages">{(['motd', 'imotd'] as const).map(path => <TabsTrigger key={path} value={path} disabled={saving !== null}>{names[path]}{documents[path] && documents[path]!.draft !== documents[path]!.file.content ? ' · Unsaved' : ''}</TabsTrigger>)}</TabsList>
      <TabsContent value={selected} className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3"><p className="text-sm text-ink-muted">{selected === 'motd' ? 'Shown to mortal characters.' : 'Shown to immortal characters instead of the regular MOTD.'} Saving updates the welcome message at once.</p><div className="flex gap-2"><Button variant="outline" onClick={reload} disabled={!doc || saving !== null}>Load saved copy</Button><Button onClick={save} disabled={!allowed || !doc || doc.draft === doc.file.content || bytes > limit || saving !== null || conflict?.path === selected}>{saving === selected ? 'Saving…' : `Save ${names[selected]}`}</Button></div></div>
        {listing.isError || file.isError ? <div role="alert" className="border border-accent p-4 text-accent">Couldn't load this message. Check your connection and access, then try again. <Button variant="outline" onClick={() => { listing.refetch(); file.refetch(); }}>Try again</Button></div> : !doc ? <Skeleton className="h-96 w-full" /> : <>
          {!allowed && <p className="border border-rule bg-paper-deep p-3 text-sm">Read only. Saving requires {entry?.required_label ?? 'a higher staff level'}{entry?.required_level ? ` (level ${entry.required_level})` : ''}.</p>}
          {error && <p role="alert" className="border border-accent p-3 text-accent">{error}</p>}
          <div className="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
            <section className="min-w-0 space-y-3" aria-label="Message editor">
              <div className="flex flex-wrap items-center justify-between gap-2"><label htmlFor="motd-draft" className="text-lg">{names[selected]} text</label><span className={`font-mono text-xs ${bytes > limit ? 'text-accent' : 'text-ink-muted'}`} role="status">{bytes.toLocaleString()} / {limit.toLocaleString()} bytes</span></div>
              <textarea ref={editor} id="motd-draft" value={doc.draft} onChange={e => change(e.target.value)} readOnly={!allowed} spellCheck={false} className="min-h-80 w-full resize-y border border-rule bg-paper p-3 font-mono text-sm leading-relaxed focus-visible:outline-2 focus-visible:outline-accent" />
              {bytes > limit && <p role="alert" className="text-sm text-accent">This message is too long. Shorten it before saving.</p>}
              <div className="flex flex-wrap items-center gap-2"><span className="text-sm text-ink-muted">Insert at cursor</span><Button variant="outline" disabled={!allowed} onClick={() => { setInsertion({ start: editor.current?.selectionStart ?? doc.draft.length, end: editor.current?.selectionEnd ?? doc.draft.length }); setShowInvite(value => !value); }}>Discord block</Button><Button variant="outline" disabled={!allowed} onClick={() => insert(gossipNotice + '\n')}>Gossip notice</Button></div>
              {showInvite && <form className="space-y-3 border border-rule bg-paper-deep p-3" onSubmit={event => {
                event.preventDefault();
                const snippet = discordSnippet(invite);
                if (!snippet) { setInviteError('Enter an HTTPS invite from discord.gg or discord.com/invite.'); return; }
                insert(snippet, insertion); setShowInvite(false); setInviteError(''); setInvite('');
              }}><label htmlFor="motd-invite" className="block text-sm">Discord invite</label><Input autoFocus id="motd-invite" type="url" required value={invite} onChange={event => setInvite(event.target.value)} placeholder="https://discord.gg/your-invite" aria-invalid={!!inviteError} aria-describedby={inviteError ? 'motd-invite-error' : undefined} />{inviteError && <p id="motd-invite-error" role="alert" className="text-sm text-accent">{inviteError}</p>}<div className="flex gap-2"><Button type="submit">Insert Discord block</Button><Button type="button" variant="outline" onClick={() => setShowInvite(false)}>Cancel</Button></div></form>}
              <p className="text-xs text-ink-muted">Snippets only change your draft. Check the Discord invite before saving. If Notifications already adds the gossip notice, leave it out here.</p>
              <details className="text-sm"><summary className="cursor-pointer text-accent underline underline-offset-4">Color codes</summary><p className="mt-2 font-mono text-xs leading-relaxed">&amp;r red · &amp;g green · &amp;b blue · &amp;c cyan · &amp;m magenta · &amp;y yellow · &amp;w white · &amp;d black. Uppercase makes colors bright. &amp;n resets; &amp;&amp; prints &amp;.</p></details>
            </section>
            <section className="min-w-0 space-y-3" aria-label="Live preview"><div className="flex flex-wrap items-center justify-between gap-2"><h2 className="text-lg">Telnet preview</h2><div className="flex flex-wrap items-center gap-3"><label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={colour} onChange={e => setColour(e.target.checked)} />Color</label><label className="sr-only" htmlFor="motd-columns">Preview width</label><NativeSelect id="motd-columns" value={columns} onChange={e => setColumns(Number(e.target.value))}>{[40, 80, 100].map(n => <option key={n} value={n}>{n} columns</option>)}</NativeSelect></div></div>
              <MotdPreview text={doc.draft} columns={columns} colour={colour} /><p className="text-xs text-ink-muted">Lines wrap at the selected width, as in a telnet client. Scroll sideways on a narrow screen. Automatic notices are not included.</p>
            </section>
          </div>
          {conflict?.path === selected && <section role="alert" className="space-y-3 border border-accent bg-paper-deep p-4"><h2 className="text-lg">Someone else saved this message first</h2><p>Nothing was overwritten. Your draft is still here. Copy what you need, then load the saved copy and make your changes again.</p>{difference ? <pre className="max-h-64 overflow-auto border border-rule bg-paper p-3 font-mono text-xs" aria-label="Saved copy compared with your draft">{difference.map((line, i) => <div key={i}>{line.kind === 'server' ? '− ' : line.kind === 'draft' ? '+ ' : '  '}{line.text || ' '}</div>)}</pre> : conflict.server && <pre className="max-h-64 overflow-auto whitespace-pre-wrap border border-rule bg-paper p-3 font-mono text-xs">{conflict.server.content}</pre>}<p className="text-xs">− saved copy · + your draft</p><Button variant="outline" onClick={reload}>Load saved copy</Button></section>}
        </>}
      </TabsContent>
    </Tabs>
  </div>;
}
