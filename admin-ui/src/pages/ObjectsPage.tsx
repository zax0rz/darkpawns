import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Obj } from '../api/client';
import { TableSkeleton } from '../components/Skeleton';

const itemTypeLabels: Record<number, string> = {
  0: 'Light', 1: 'Scroll', 2: 'Wand', 3: 'Staff', 4: 'Weapon',
  5: 'Fire Weapon', 6: 'Missile', 7: 'Treasure', 8: 'Armor', 9: 'Potion',
  10: 'Worn', 11: 'Furniture', 12: 'Trash', 13: 'Container', 14: 'Note',
  15: 'Drink Container', 16: 'Key', 17: 'Food', 18: 'Money', 19: 'Pen',
  20: 'Boat', 21: 'Fountain', 22: 'Campfire', 23: 'Corpse',
};

function itemTypeLabel(flag: number): string {
  return itemTypeLabels[flag] || `Type ${flag}`;
}

export function ObjectsPage() {
  const [search, setSearch] = useState('');
  const [sortKey, setSortKey] = useState<keyof Obj>('vnum');
  const [sortAsc, setSortAsc] = useState(true);

  const { data: objects, isLoading, error } = useQuery({
    queryKey: ['objects'],
    queryFn: api.objects,
  });

  const filtered = useMemo(() => {
    if (!objects) return [];
    const q = search.toLowerCase();
    let list = q
      ? objects.filter(
          (o) =>
            String(o.vnum).includes(q) ||
            o.short_desc.toLowerCase().includes(q) ||
            o.keywords.toLowerCase().includes(q)
        )
      : [...objects];

    list.sort((a, b) => {
      const av = a[sortKey];
      const bv = b[sortKey];
      if (typeof av === 'number' && typeof bv === 'number') {
        return sortAsc ? av - bv : bv - av;
      }
      return sortAsc
        ? String(av).localeCompare(String(bv))
        : String(bv).localeCompare(String(av));
    });
    return list;
  }, [objects, search, sortKey, sortAsc]);

  const toggleSort = (key: keyof Obj) => {
    if (sortKey === key) {
      setSortAsc(!sortAsc);
    } else {
      setSortKey(key);
      setSortAsc(key === 'vnum' || key === 'short_desc');
    }
  };

  const SortHeader = ({
    label,
    field,
  }: {
    label: string;
    field: keyof Obj;
  }) => (
    <th
      className="text-left px-4 py-3 cursor-pointer hover:text-amber-600 dark:hover:text-amber-400 select-none"
      onClick={() => toggleSort(field)}
    >
      {label}
      {sortKey === field && (
        <span className="ml-1 text-amber-600 dark:text-amber-400">{sortAsc ? '↑' : '↓'}</span>
      )}
    </th>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Objects</h1>
        {objects && (
          <span className="text-sm text-slate-500 dark:text-slate-400">
            {filtered.length} of {objects.length} objects
          </span>
        )}
      </div>

      {/* Search */}
      <input
        type="text"
        placeholder="Filter by vnum, name, or keywords..."
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        className="w-full px-4 py-2 rounded-lg bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 text-slate-900 dark:text-white placeholder-slate-400 dark:placeholder-slate-500 focus:outline-none focus:border-amber-500 text-sm"
      />

      {isLoading && <TableSkeleton rows={6} cols={5} />}

      {error && (
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-4 text-sm text-red-700 dark:text-red-300">
          Failed to load objects.
          <div className="mt-1 text-red-500/70 dark:text-red-400/70 text-xs">
            {(error as Error).message}
          </div>
        </div>
      )}

      {filtered.length > 0 && (
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700 text-xs text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                  <SortHeader label="VNum" field="vnum" />
                  <SortHeader label="Name" field="short_desc" />
                  <SortHeader label="Type" field="type_flag" />
                  <SortHeader label="Weight" field="weight" />
                  <SortHeader label="Cost" field="cost" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((obj) => (
                  <tr
                    key={obj.vnum}
                    className="border-b border-slate-100 dark:border-slate-700/50 hover:bg-slate-50 dark:hover:bg-slate-700/30 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/game/objects/${obj.vnum}`}
                        className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 font-mono"
                      >
                        {obj.vnum}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-slate-900 dark:text-white">{obj.short_desc}</td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300">
                      {itemTypeLabel(obj.type_flag)}
                    </td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300 font-mono">
                      {obj.weight}
                    </td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300 font-mono">
                      {obj.cost.toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {objects && objects.length === 0 && (
        <div className="text-center text-slate-400 dark:text-slate-500 py-8">
          No objects loaded. Check world files.
        </div>
      )}
    </div>
  );
}
