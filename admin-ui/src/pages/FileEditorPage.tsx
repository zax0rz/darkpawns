import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useSearchParams } from 'react-router-dom';
import { ApiError } from '../api/client';
import { fileEditApi, type EditableFile, type FileAction, type FileEntry, type FileRoot } from '../api/fileedit';
import { CodeEditor } from '../components/olc/CodeEditor';
import { Skeleton } from '../components/Skeleton';
import { useToast } from '../hooks/useToast';
import { lineDiff } from '../lib/lineDiff';

// The page's open document. etag null means the file does not exist yet and
// the first save creates it (If-None-Match: *).
interface OpenFile {
  path: string;
  etag: string | null;
  serverContent: string;
}

interface Conflict {
  draft: string;
  server: EditableFile | null;
}

const USAGE_ROUTES: Record<string, string> = { room: 'rooms', mob: 'mobs', obj: 'objects' };

function requirement(level?: number, label?: string): string {
  return label && level ? `${label} (level ${level})` : 'a higher level';
}

function parentDirectory(path: string): string {
  return path.split('/').slice(0, -1).join('/');
}

export function FileEditorPage({ root }: { root: FileRoot }) {
  const isLua = root === 'lua';
  const [searchParams, setSearchParams] = useSearchParams();
  const [directory, setDirectory] = useState(() => parentDirectory(searchParams.get('file') || ''));
  const [open, setOpen] = useState<OpenFile | null>(null);
  const [draft, setDraft] = useState('');
  const [refusal, setRefusal] = useState('');
  const [conflict, setConflict] = useState<Conflict | null>(null);
  const [newName, setNewName] = useState('');
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const requestedFile = searchParams.get('file') || '';
  const requestedScript = isLua ? searchParams.get('script') || '' : '';
  const requestedKind = searchParams.get('kind') || '';

  const listing = useQuery({
    queryKey: ['file-listing', root, directory],
    queryFn: () => fileEditApi.listing(root, directory),
  });
  const resolution = useQuery({
    queryKey: ['script-resolve', requestedScript, requestedKind],
    queryFn: () => fileEditApi.resolve(requestedScript, requestedKind),
    enabled: requestedScript !== '',
    retry: false,
  });
  const selectedPath = requestedFile || (resolution.data?.exists ? resolution.data.path : '');
  const file = useQuery({
    queryKey: ['editable-file', root, selectedPath],
    queryFn: () => fileEditApi.read(root, selectedPath),
    enabled: selectedPath !== '',
    retry: false,
  });
  const usagePath = open?.path || '';
  const usage = useQuery({
    queryKey: ['script-usage', usagePath],
    queryFn: () => fileEditApi.usage(usagePath),
    enabled: isLua && usagePath !== '' && open?.etag !== null,
    retry: false,
  });

  // Loading a file resets the draft; a script_name with no file yet opens an
  // empty buffer at the engine's proposed path, and saving creates it. Both
  // adjust state during render, keyed on the query result they came from.
  const [seededFrom, setSeededFrom] = useState<unknown>(null);
  if (file.data && file.data !== seededFrom) {
    setSeededFrom(file.data);
    if (isLua) setDirectory(parentDirectory(file.data.path));
    setOpen({ path: file.data.path, etag: file.data.etag, serverContent: file.data.content });
    setDraft(file.data.content);
    setConflict(null);
    setRefusal('');
  } else if (resolution.data && !resolution.data.exists && resolution.data !== seededFrom) {
    setSeededFrom(resolution.data);
    setDirectory(parentDirectory(resolution.data.path));
    setOpen({ path: resolution.data.path, etag: null, serverContent: '' });
    setDraft('');
  }

  const writeAction: FileAction | undefined = listing.data?.actions.find((action) => action.key === 'write');
  const openEntry: FileEntry | undefined = listing.data?.entries.find((entry) => entry.path === open?.path);
  // Tedit rows carry their own level (credits sits above the web door); Lua files share the
  // write action's level.
  const canWrite = isLua ? writeAction?.allowed ?? false : openEntry?.allowed ?? false;
  const writeRequirement = isLua
    ? requirement(writeAction?.required_level, writeAction?.required_label)
    : requirement(openEntry?.required_level, openEntry?.required_label);
  const dirty = open !== null && draft !== open.serverContent;
  const isNew = open?.etag === null;

  useEffect(() => {
    if (!dirty) return;
    const warn = (event: BeforeUnloadEvent) => event.preventDefault();
    window.addEventListener('beforeunload', warn);
    return () => window.removeEventListener('beforeunload', warn);
  }, [dirty]);

  function selectFile(path: string) {
    if (dirty && !window.confirm('Discard unsaved changes?')) return;
    setSeededFrom(null);
    setSearchParams({ file: path });
  }

  const save = useMutation({
    mutationFn: () => fileEditApi.save(root, open!.path, draft, open!.etag),
    onSuccess: (saved) => {
      queryClient.setQueryData(['editable-file', root, saved.path], saved);
      setOpen({ path: saved.path, etag: saved.etag, serverContent: saved.content });
      setDraft(saved.content);
      setConflict(null);
      setRefusal('');
      if (isNew) {
        setSearchParams({ file: saved.path });
        queryClient.invalidateQueries({ queryKey: ['file-listing', root] });
      }
      showToast(`${saved.path} saved`, 'success');
    },
    onError: async (error) => {
      if (error instanceof ApiError && error.status === 412) {
        const server = await fileEditApi.read(root, open!.path).catch(() => null);
        setConflict({ draft, server });
        return;
      }
      if (error instanceof ApiError && error.status === 422) {
        setRefusal(error.message);
        return;
      }
      showToast(`Save failed: ${(error as Error).message}`, 'error');
    },
  });

  const remove = useMutation({
    mutationFn: () => fileEditApi.remove(root, open!.path, open!.etag || ''),
    onSuccess: () => {
      const path = open!.path;
      setOpen(null);
      setDraft('');
      setSearchParams({});
      queryClient.invalidateQueries({ queryKey: ['file-listing', root] });
      showToast(`${path} deleted`, 'success');
    },
    onError: async (error) => {
      if (error instanceof ApiError && error.status === 412) {
        const server = await fileEditApi.read(root, open!.path).catch(() => null);
        setConflict({ draft, server });
        return;
      }
      showToast(`Delete failed: ${(error as Error).message}`, 'error');
    },
  });

  function startNewScript(event: React.FormEvent) {
    event.preventDefault();
    const name = newName.trim().replace(/^\/+/, '');
    if (!name) return;
    const file = name.endsWith('.lua') ? name : `${name}.lua`;
    const path = directory ? `${directory}/${file}` : file;
    if (dirty && !window.confirm('Discard unsaved changes?')) return;
    setSeededFrom(null);
    setSearchParams({});
    setOpen({ path, etag: null, serverContent: '' });
    setDraft('');
    setNewName('');
    setRefusal('');
  }

  function reloadServerCopy() {
    if (!conflict) return;
    if (conflict.server) {
      queryClient.setQueryData(['editable-file', root, conflict.server.path], conflict.server);
      setOpen({ path: conflict.server.path, etag: conflict.server.etag, serverContent: conflict.server.content });
      setDraft(conflict.server.content);
    } else {
      setOpen(null);
      setDraft('');
      setSearchParams({});
      queryClient.invalidateQueries({ queryKey: ['file-listing', root] });
    }
    setConflict(null);
  }

  const entries = listing.data?.entries || [];
  const title = isLua ? 'Lua scripts' : 'Server text';
  const description = isLua
    ? 'The live script tree. A save is parse-checked first and runs on the next trigger.'
    : 'News, MOTD, help, credits, and the rest of the tedit table. Saves go live at once.';
  const usageByKind = useMemo(() => {
    const result: Record<string, number> = {};
    for (const item of usage.data || []) result[item.kind] = (result[item.kind] || 0) + 1;
    return result;
  }, [usage.data]);
  const loadError = file.error || resolution.error;

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-3 border-b border-rule pb-4">
        <div>
          <h1 className="text-2xl text-ink">{title}</h1>
          <p className="mt-2 max-w-3xl text-sm text-ink-muted">{description}</p>
        </div>
        <nav className="flex gap-2 text-xs font-semibold uppercase tracking-wider" aria-label="File roots">
          <Link to="/admin/workshop/scripts" className={`border px-3 py-2 ${isLua ? 'border-accent bg-accent text-paper' : 'border-rule bg-paper-deep text-ink'}`}>Scripts</Link>
          <Link to="/admin/workshop/text" className={`border px-3 py-2 ${!isLua ? 'border-accent bg-accent text-paper' : 'border-rule bg-paper-deep text-ink'}`}>Text files</Link>
        </nav>
      </div>

      <div className="grid gap-4 xl:grid-cols-[17rem_minmax(0,1fr)_18rem]">
        <aside className="border border-rule bg-paper-deep p-3">
          <div className="flex items-center justify-between border-b border-rule pb-3">
            <h2 className="text-sm font-semibold text-ink">Files</h2>
            {isLua && directory && <button type="button" className="text-xs text-accent hover:underline" onClick={() => setDirectory(parentDirectory(directory))}>Up</button>}
          </div>
          {isLua && <p className="mt-2 break-all font-mono text-[11px] text-ink-muted">scripts/{directory}</p>}
          {listing.isLoading ? <Skeleton className="mt-3 h-40 w-full" /> : listing.error ? (
            <p className="mt-3 text-sm text-accent" role="alert">Could not list files: {(listing.error as Error).message}</p>
          ) : (
            <ul className="mt-3 max-h-[60vh] space-y-1 overflow-y-auto">
              {entries.map((entry) => (
                <li key={entry.path}>
                  <button
                    type="button"
                    onClick={() => entry.directory ? setDirectory(entry.path) : selectFile(entry.path)}
                    aria-current={open?.path === entry.path ? 'true' : undefined}
                    className={`block w-full border px-2 py-2 text-left text-xs ${open?.path === entry.path ? 'border-accent bg-paper text-accent' : 'border-transparent text-ink hover:border-rule hover:bg-paper'}`}
                  >
                    <span className="font-mono">{entry.directory ? `${entry.name}/` : entry.name}</span>
                    {!isLua && (
                      <span className="mt-1 block text-[10px] text-ink-muted">
                        {entry.max_bytes?.toLocaleString()} bytes{entry.allowed ? '' : ` · ${entry.required_label} to edit`}
                      </span>
                    )}
                  </button>
                </li>
              ))}
              {!entries.length && <li className="py-4 text-sm text-ink-muted">No scripts here.</li>}
            </ul>
          )}
          {isLua && (
            <form onSubmit={startNewScript} className="mt-4 border-t border-rule pt-3">
              <label htmlFor="new-script" className="block text-xs font-semibold text-ink">New script here</label>
              <div className="mt-2 flex gap-2">
                <input id="new-script" value={newName} onChange={(event) => setNewName(event.target.value)} disabled={!writeAction?.allowed} placeholder="name.lua" className="min-w-0 flex-1 border border-rule bg-paper px-2 py-1 font-mono text-xs text-ink disabled:opacity-50" />
                <button type="submit" disabled={!writeAction?.allowed || !newName.trim()} className="border border-rule bg-paper px-2 py-1 text-xs font-semibold text-ink hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40">Open</button>
              </div>
              {writeAction && !writeAction.allowed && <p className="mt-2 text-[11px] text-ink-muted">Requires {writeRequirement}.</p>}
            </form>
          )}
        </aside>

        <main className="min-w-0">
          {loadError ? (
            <div className="border border-accent bg-paper-deep p-4 text-sm text-accent" role="alert">Could not open {selectedPath || requestedScript}: {(loadError as Error).message}</div>
          ) : (file.isLoading || resolution.isLoading) && !open ? (
            <Skeleton className="h-[38rem] w-full" />
          ) : !open ? (
            <div className="flex min-h-[30rem] items-center justify-center border border-dashed border-rule bg-paper-deep p-8 text-center text-sm text-ink-muted">Choose a file to read or edit.</div>
          ) : (
            <>
              <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
                <div>
                  <h2 className="break-all font-mono text-sm text-ink">{isLua ? `scripts/${open.path}` : open.path}</h2>
                  <p className="mt-1 text-xs text-ink-muted">{isNew ? 'New file; saving creates it' : dirty ? 'Unsaved changes' : 'Matches the server copy'}</p>
                </div>
                <div className="flex gap-2">
                  <button type="button" disabled={!canWrite || (!dirty && !isNew) || save.isPending} title={canWrite ? undefined : `Requires ${writeRequirement}`} onClick={() => save.mutate()} className="border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:cursor-not-allowed disabled:opacity-40">{save.isPending ? 'Saving…' : isNew ? 'Create' : 'Save'}</button>
                  {isLua && !isNew && <button type="button" disabled={!canWrite || remove.isPending} title={canWrite ? undefined : `Requires ${writeRequirement}`} onClick={() => window.confirm(`Delete scripts/${open.path}? Anything that runs it will fail to load.`) && remove.mutate()} className="border border-rule bg-paper px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40">Delete</button>}
                </div>
              </div>
              {!canWrite && <div className="mb-3 border border-rule bg-paper-deep px-4 py-3 text-sm text-ink-muted">Read only. Editing this file requires <strong className="text-ink">{writeRequirement}</strong>.</div>}
              {canWrite && <div className="mb-3 border border-accent bg-paper-deep px-4 py-3 text-sm text-ink"><strong className="text-accent">Applies immediately.</strong> {isLua ? 'The next trigger runs the saved bytes; there is no draft or undo.' : 'Players see the saved text at once; there is no draft or undo.'}</div>}
              {refusal && <div className="mb-3 border border-accent bg-paper-deep px-4 py-3 font-mono text-xs text-accent" role="alert">Not saved. {refusal}</div>}
              <CodeEditor key={`${root}:${open.path}`} label={open.path} value={draft} language={isLua ? 'lua' : 'text'} readOnly={!canWrite} onChange={setDraft} />
            </>
          )}
        </main>

        <aside className="space-y-4">
          {isLua && open && !isNew && (
            <section className="border border-rule bg-paper-deep p-4">
              <h2 className="text-sm font-semibold text-ink">Used by</h2>
              {usage.isLoading ? <Skeleton className="mt-3 h-16 w-full" /> : (
                <>
                  <p className="mt-2 font-mono text-xs text-ink-muted">{Object.entries(usageByKind).map(([kind, count]) => `${count} ${kind}`).join(' · ') || 'Nothing in the world runs this file.'}</p>
                  <ul className="mt-3 space-y-2">
                    {(usage.data || []).map((item) => (
                      <li key={`${item.kind}-${item.vnum}`}>
                        <Link to={`/admin/game/${USAGE_ROUTES[item.kind]}/${item.vnum}`} className="block border-t border-rule pt-2 text-xs text-accent hover:underline">{item.kind} #{item.vnum}: {item.name}</Link>
                      </li>
                    ))}
                  </ul>
                </>
              )}
            </section>
          )}
          <section className="border border-rule bg-paper-deep p-4">
            <h2 className="text-sm font-semibold text-ink">Conflicts</h2>
            <p className="mt-2 text-sm text-ink-muted">A save only lands on the exact copy you opened. If someone else saved first, nothing is overwritten and you get a diff to work from.</p>
          </section>
        </aside>
      </div>

      {conflict && <ConflictPanel conflict={conflict} onReload={reloadServerCopy} />}
    </div>
  );
}

function ConflictPanel({ conflict, onReload }: { conflict: Conflict; onReload: () => void }) {
  const diff = useMemo(() => conflict.server ? lineDiff(conflict.server.content, conflict.draft) : null, [conflict]);
  return (
    <section className="border border-accent bg-paper-deep p-4" role="alert">
      <h2 className="text-lg text-accent">{conflict.server ? 'Someone else saved this file while you were editing' : 'This file was deleted while you were editing'}</h2>
      <p className="mt-1 text-sm text-ink-muted">Nothing was overwritten. Your draft is still in the editor. Copy what you need, then load the server copy and apply your changes to it.</p>
      {conflict.server && (diff ? (
        <pre className="mt-4 max-h-96 overflow-auto border border-rule bg-paper p-3 font-mono text-xs leading-relaxed" aria-label="Differences between the server copy and your draft">
          {diff.map((line, index) => (
            <div key={index} className={line.kind === 'server' ? 'bg-paper-deep text-ink-muted line-through' : line.kind === 'draft' ? 'text-accent' : 'text-ink'}>
              {line.kind === 'server' ? '- ' : line.kind === 'draft' ? '+ ' : '  '}{line.text || ' '}
            </div>
          ))}
        </pre>
      ) : (
        <div className="mt-4 grid gap-3 lg:grid-cols-2">
          <WholeCopy label="Your draft" value={conflict.draft} />
          <WholeCopy label="Server copy" value={conflict.server.content} />
        </div>
      ))}
      {conflict.server && diff && <p className="mt-2 text-xs text-ink-muted"><span className="line-through">Struck lines</span> are only on the server; <span className="text-accent">+ lines</span> are only in your draft.</p>}
      <button type="button" onClick={onReload} className="mt-4 border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep">{conflict.server ? 'Load server copy' : 'Close file'}</button>
    </section>
  );
}

function WholeCopy({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <h3 className="mb-2 text-xs font-semibold uppercase tracking-wider text-ink">{label}</h3>
      <pre className="max-h-80 overflow-auto whitespace-pre-wrap border border-rule bg-paper p-3 font-mono text-xs text-ink">{value}</pre>
    </div>
  );
}
