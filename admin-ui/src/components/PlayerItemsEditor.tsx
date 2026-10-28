import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Obj, type PlayerItemChange, type SavedPlayer, type SavedPlayerItem } from '../api/client';
import { itemSearch, type ItemDraft, type ItemSource } from '../lib/playerItems';
import { itemTypeLabel } from '../lib/gameLabels';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { NativeSelect } from './ui/select';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from './ui/dialog';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { ConfirmDialog } from './ConfirmDialog';

interface Props {
  player: SavedPlayer;
  draft: ItemDraft;
  disabled: boolean;
  onChange: (change: PlayerItemChange, granted?: SavedPlayerItem) => void;
}
const emptyChange = { action: 'move' as const, source: 'inventory' as const, index: 0, vnum: 0, locate: 0 };

export function PlayerItemsEditor({ player, draft, disabled, onChange }: Props) {
  const [picker, setPicker] = useState<number | null>(null);
  const [removal, setRemoval] = useState<{ source: ItemSource; item: SavedPlayerItem } | null>(null);
  const equipment = draft.equipment ?? [];
  const worn = equipment.filter(item => item.parent === 0);
  const slots = player.slots ?? [];
  const occupied = new Set(worn.map(item => item.locate));
  const remove = (source: ItemSource, item: SavedPlayerItem) => setRemoval({ source, item });
  const change = (source: ItemSource, item: SavedPlayerItem, locate: number) => onChange({ ...emptyChange, source, index: item.index, locate });

  return <div className="space-y-8">
    <p className="text-sm text-ink-muted max-w-prose">Changes wait until you save. Containers travel with their contents. Staff can assign any single item to a wear slot.</p>
    <section aria-labelledby="equipment-heading">
      <div className="flex flex-wrap items-baseline justify-between gap-2 mb-3"><h3 id="equipment-heading" className="text-xl">Equipment</h3><p className="text-sm text-ink-muted">{worn.length} of {slots.length} slots filled</p></div>
      <ul className="grid sm:grid-cols-2 border-t border-rule sm:gap-x-6">
        {slots.map(slot => {
          const item = worn.find(item => item.locate === slot.locate);
          return <li key={slot.locate} className="min-w-0 border-b border-rule py-3 space-y-2">
            <div className="flex items-baseline justify-between gap-2"><h4 className="text-sm font-semibold capitalize">{slot.name}</h4>{!item && <Button variant="outline" size="sm" disabled={disabled} aria-label={`Equip ${slot.name}`} onClick={() => setPicker(slot.locate)}>Equip</Button>}</div>
            {item ? <><ItemName item={item} /><div className="flex flex-wrap gap-2"><Button variant="outline" size="sm" disabled={disabled} aria-label={`Unequip ${slot.name}`} onClick={() => change('equipment', item, 0)}>Unequip</Button><Button variant="ghost" size="sm" disabled={disabled} aria-label={`Remove ${item.name} from ${slot.name}`} onClick={() => remove('equipment', item)}>Remove</Button></div>
              {equipment.some(child => child.parent === item.index + 1) && <p className="text-xs text-ink-muted">Contents listed below.</p>}
            </> : <p className="text-sm text-ink-muted">Empty</p>}
          </li>;
        })}
      </ul>
    </section>
    <section aria-labelledby="inventory-heading">
      <div className="flex items-center justify-between gap-3 mb-3"><div><h3 id="inventory-heading" className="text-xl">Inventory</h3><p className="text-sm text-ink-muted mt-1">Carried items and container contents</p></div><Button variant="outline" disabled={disabled} onClick={() => setPicker(0)}>Give item</Button></div>
      {!(draft.inventory ?? []).length ? <p className="border-y border-rule py-5 text-sm text-ink-muted">Nothing carried. Give an item to add it here.</p> : <InventoryTree key={`inventory-${draft.changes.length}`} items={draft.inventory ?? []} source="inventory" disabled={disabled} slots={slots} occupied={occupied} onMove={change} onRemove={remove} />}
    </section>
    {equipment.some(item => item.parent > 0) && <section aria-labelledby="worn-contents-heading"><h3 id="worn-contents-heading" className="text-xl mb-3">Worn containers</h3><p className="text-sm text-ink-muted mb-3">Contents stay with the worn item when you unequip it.</p><InventoryTree key={`equipment-${draft.changes.length}`} items={equipment} source="equipment" disabled={disabled} slots={slots} occupied={occupied} onMove={change} onRemove={remove} /></section>}
    <p className="text-sm text-ink-muted">Items marked “Not saved after logout” share that setting with every copy. <Link className="underline underline-offset-2 text-accent" to="/admin/game/objects">Change item definitions</Link>.</p>
    {picker !== null && <ItemPicker locate={picker} slotName={slots.find(slot => slot.locate === picker)?.name ?? ''} inventory={draft.inventory ?? []} disabled={disabled} onClose={() => setPicker(null)} onPick={(action, item) => { onChange(action, item); setPicker(null); }} />}
    <ConfirmDialog open={removal !== null} onOpenChange={open => { if (!open) setRemoval(null); }} title={`Remove ${removal?.item.name ?? 'item'}?`} description="This item and anything inside it will be deleted when you save. Discard your changes to bring it back before saving." action="Remove item" onConfirm={() => { if (removal) onChange({ ...emptyChange, action: 'remove', source: removal.source, index: removal.item.index }); setRemoval(null); }} />
  </div>;
}

function ItemName({ item }: { item: SavedPlayerItem }) {
  return <div className="min-w-0"><p className="text-sm break-words">{item.name}{item.count > 1 && <span className="font-mono text-xs"> × {item.count}</span>}</p><p className="text-xs text-ink-muted mt-0.5"><span className="font-mono">#{item.vnum}</span>{item.no_rent && <span> · Not saved after logout</span>}</p></div>;
}

function InventoryTree({ items, source, slots, occupied, disabled, onMove, onRemove }: {
  items: SavedPlayerItem[]; source: ItemSource; slots: NonNullable<SavedPlayer['slots']>; occupied: Set<number>; disabled: boolean;
  onMove: (source: ItemSource, item: SavedPlayerItem, locate: number) => void; onRemove: (source: ItemSource, item: SavedPlayerItem) => void;
}) {
  const [collapsed, setCollapsed] = useState(new Set<number>());
  const parents = new Set(items.map(item => item.parent));
  const depths: number[] = [];
  const hidden: boolean[] = [];
  return <ul className="border-t border-rule" aria-label={source === 'equipment' ? 'Worn container contents' : 'Carried items'}>
    {items.map(item => {
      depths[item.index] = item.parent ? (depths[item.parent - 1] ?? 0) + 1 : 0;
      hidden[item.index] = item.parent > 0 && (hidden[item.parent - 1] || collapsed.has(item.parent - 1));
      if (hidden[item.index]) return null;
      const container = item.container || parents.has(item.index + 1);
      return <li key={item.index} className="border-b border-rule py-3" style={{ paddingInlineStart: `${Math.min(depths[item.index], 4) * 1.25}rem` }}>
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 flex-1 basis-48 space-y-1"><ItemName item={item} />
            {item.parent > 0 && <p className="text-xs text-ink-muted break-words">In {items[item.parent - 1]?.name ?? 'a container'}</p>}
            {source === 'equipment' && item.parent === 0 && <p className="text-xs text-ink-muted capitalize">Worn: {slots.find(slot => slot.locate === item.locate)?.name ?? 'unknown slot'}</p>}
            {container && <Button variant="link" size="sm" className="px-0" aria-label={`${collapsed.has(item.index) ? 'Show' : 'Hide'} contents of ${item.name}`} aria-expanded={!collapsed.has(item.index)} onClick={() => setCollapsed(current => { const next = new Set(current); if (next.has(item.index)) next.delete(item.index); else next.add(item.index); return next; })}>{parents.has(item.index + 1) ? collapsed.has(item.index) ? 'Show contents' : 'Hide contents' : 'Empty container'}</Button>}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {source === 'inventory' || item.parent > 0 ? <ItemMove item={item} slots={slots} occupied={occupied} disabled={disabled} nested={item.parent > 0} onMove={locate => onMove(source, item, locate)} /> : <Button variant="outline" size="sm" disabled={disabled} onClick={() => onMove(source, item, 0)}>Unequip</Button>}
            <Button variant="ghost" size="sm" disabled={disabled} aria-label={`Remove ${item.name} ${item.index + 1}`} onClick={() => onRemove(source, item)}>Remove</Button>
          </div>
        </div>
      </li>;
    })}
  </ul>;
}

function ItemMove({ item, slots, occupied, disabled, nested, onMove }: { item: SavedPlayerItem; slots: NonNullable<SavedPlayer['slots']>; occupied: Set<number>; disabled: boolean; nested: boolean; onMove: (locate: number) => void }) {
  const [destination, setDestination] = useState('');
  return <><NativeSelect className="w-auto max-w-48 text-sm" aria-label={`Move ${item.name} ${item.index + 1} to`} value={destination} disabled={disabled} onChange={event => setDestination(event.target.value)}><option value="">Move to…</option>{nested && <option value="0">Carried inventory</option>}{slots.map(slot => <option key={slot.locate} value={slot.locate} disabled={occupied.has(slot.locate) || item.count > 1}>{slot.name}{occupied.has(slot.locate) ? ' (filled)' : ''}</option>)}</NativeSelect><Button variant="outline" size="sm" disabled={disabled || destination === ''} onClick={() => onMove(Number(destination))}>{destination === '0' ? 'Take out' : 'Equip'}</Button></>;
}

function ItemPicker({ locate, slotName, inventory, disabled, onClose, onPick }: { locate: number; slotName: string; inventory: SavedPlayerItem[]; disabled: boolean; onClose: () => void; onPick: (change: PlayerItemChange, item?: SavedPlayerItem) => void }) {
  const [query, setQuery] = useState('');
  const [page, setPage] = useState(0);
  const [source, setSource] = useState(locate > 0 && inventory.length ? 'carried' : 'new');
  const objects = useQuery({ queryKey: ['objects'], queryFn: api.objects });
  const carried = inventory.filter(item => item.count === 1 && itemSearch(item.name, item.vnum, '', query));
  const found = useMemo(() => (objects.data ?? []).filter(item => itemSearch(item.short_desc, item.vnum, item.keywords, query)).sort((a, b) => a.vnum - b.vnum), [objects.data, query]);
  const list = source === 'carried' ? carried : found;
  const lastPage = Math.max(0, Math.ceil(list.length / 25) - 1);
  const currentPage = Math.min(page, lastPage);
  const pickNew = (item: Obj) => onPick({ ...emptyChange, action: 'grant', vnum: item.vnum, locate }, { index: 0, vnum: item.vnum, name: item.short_desc, count: 1, parent: 0, locate, no_rent: ((item.extra_flags?.[0] ?? 0) & 4) !== 0, container: item.type_flag === 15 });
  return <Dialog open onOpenChange={open => { if (!open) onClose(); }}><DialogContent className="sm:max-w-xl max-h-[85dvh] flex flex-col overflow-hidden"><DialogHeader><DialogTitle>{locate ? `Equip ${slotName}` : 'Give an item'}</DialogTitle><DialogDescription>{locate ? 'Move a carried item here or add a new copy. This waits until you save.' : 'Adds one new copy to carried inventory when you save.'}</DialogDescription></DialogHeader>
    <Tabs value={source} onValueChange={value => { setSource(value); setPage(0); }} className="min-h-0 flex flex-col gap-3">
      {locate > 0 && <TabsList aria-label="Item source"><TabsTrigger value="carried">Carried items</TabsTrigger><TabsTrigger value="new">New copy</TabsTrigger></TabsList>}
      <label className="text-sm" htmlFor="item-picker-search">Find an item</label><Input id="item-picker-search" type="search" placeholder="Name or vnum" value={query} onChange={event => { setQuery(event.target.value); setPage(0); }} />
      <p role="status" className="text-xs text-ink-muted">{list.length} {list.length === 1 ? 'match' : 'matches'}{source === 'carried' ? ' · Single carried items' : ' · New copies'}</p>
      {source === 'new' && objects.isLoading && <p role="status">Loading items…</p>}
      {source === 'new' && objects.error && <div role="alert"><p>Couldn't load items. Try again.</p><Button variant="outline" onClick={() => objects.refetch()}>Try again</Button></div>}
      <div className="overflow-y-auto min-h-0 max-h-[40dvh] border-y border-rule">
        <TabsContent value="carried" className="mt-0"><ul>{carried.slice(currentPage * 25, (currentPage + 1) * 25).map(item => <li key={item.index} className="flex items-center justify-between gap-3 py-3 border-b border-rule"><div className="min-w-0"><ItemName item={item} />{item.parent > 0 && <p className="text-xs text-ink-muted">In {inventory[item.parent - 1]?.name}</p>}</div><Button variant="outline" disabled={disabled} onClick={() => onPick({ ...emptyChange, index: item.index, locate })}>Equip</Button></li>)}</ul></TabsContent>
        <TabsContent value="new" className="mt-0"><ul>{found.slice(currentPage * 25, (currentPage + 1) * 25).map(item => <li key={item.vnum} className="flex items-center justify-between gap-3 py-3 border-b border-rule"><div className="min-w-0"><p className="text-sm break-words">{item.short_desc}</p><p className="text-xs text-ink-muted"><span className="font-mono">#{item.vnum}</span> · {itemTypeLabel(item.type_flag)}</p></div><Button variant="outline" disabled={disabled} aria-label={`${locate ? 'Equip' : 'Give'} ${item.short_desc} ${item.vnum}`} onClick={() => pickNew(item)}>{locate ? 'Equip new copy' : 'Give item'}</Button></li>)}</ul></TabsContent>
        {!list.length && !(source === 'new' && (objects.isLoading || objects.error)) && <p className="py-5 text-sm text-ink-muted">{source === 'carried' && !inventory.length ? 'Nothing carried. Choose New copy to add an item.' : 'No items match. Try a different name or vnum.'}</p>}
      </div>
      {lastPage > 0 && <div className="flex items-center justify-between gap-2"><Button variant="outline" size="sm" disabled={currentPage === 0} onClick={() => setPage(currentPage - 1)}>Previous</Button><span className="text-sm">Page {currentPage + 1} of {lastPage + 1}</span><Button variant="outline" size="sm" disabled={currentPage === lastPage} onClick={() => setPage(currentPage + 1)}>Next</Button></div>}
    </Tabs>
  </DialogContent></Dialog>;
}
