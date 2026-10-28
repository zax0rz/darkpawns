import type { SavedPlayer, SavedPlayerItem, PlayerItemChange } from '../api/client';

export type ItemSource = 'inventory' | 'equipment';
export type ItemDraft = Pick<SavedPlayer, ItemSource> & { changes: PlayerItemChange[] };

export function freshItems(player: SavedPlayer): ItemDraft {
  return { inventory: player.inventory ?? [], equipment: player.equipment ?? [], changes: [] };
}

// Match the server's ordered subtree edits. Original instance state never leaves
// the server; this projection carries only labels and locations for the preview.
export function changeItems(draft: ItemDraft, change: PlayerItemChange, granted?: SavedPlayerItem): ItemDraft {
  let inventory = [...(draft.inventory ?? [])];
  let equipment = [...(draft.equipment ?? [])];
  let tree: SavedPlayerItem[] = [];
  if (change.action === 'grant') {
    if (!granted) throw new Error('Choose an item to add.');
    tree = [{ ...granted, parent: 0 }];
  } else {
    const source = change.source === 'equipment' ? equipment : inventory;
    if (!source[change.index]) throw new Error('This item changed. Discard your changes and reload.');
    const selected = new Set([change.index + 1]);
    const keep: SavedPlayerItem[] = [];
    const keptIndices = new Map<number, number>();
    const treeIndices = new Map<number, number>();
    source.forEach((item, index) => {
      if (selected.has(item.parent)) selected.add(index + 1);
      if (selected.has(index + 1)) {
        treeIndices.set(index + 1, tree.length + 1);
        tree.push({ ...item, parent: index === change.index ? 0 : treeIndices.get(item.parent) ?? 0 });
      } else {
        keptIndices.set(index + 1, keep.length + 1);
        keep.push({ ...item, parent: keptIndices.get(item.parent) ?? 0 });
      }
    });
    if (change.source === 'equipment') equipment = keep;
    else inventory = keep;
  }
  if (change.action !== 'remove') {
    if (change.locate > 0 && equipment.some(item => item.parent === 0 && item.locate === change.locate)) throw new Error('Unequip the current item first.');
    if (change.locate > 0 && tree[0].count > 1) throw new Error('A stack cannot fill one wear slot.');
    if (inventory.length + equipment.length + tree.length > 2000) throw new Error('This character has reached the 2,000-item limit. Remove an item first.');
    const target = change.locate > 0 ? equipment : inventory;
    const offset = target.length;
    tree.forEach((item, index) => target.push({ ...item, locate: index === 0 ? change.locate : item.locate, parent: item.parent > 0 ? item.parent + offset : 0 }));
  }
  return {
    inventory: inventory.map((item, index) => ({ ...item, index })),
    equipment: equipment.map((item, index) => ({ ...item, index })),
    changes: [...draft.changes, change],
  };
}

export function itemSearch(name: string, vnum: number, keywords: string, query: string): boolean {
  return query.trim().toLowerCase().split(/\s+/).every(word =>
    name.toLowerCase().includes(word) || keywords.toLowerCase().includes(word) || String(vnum).includes(word.replace(/^#/, '')));
}
