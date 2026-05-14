import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Mob } from '../api/client';
import { TableSkeleton } from '../components/Skeleton';

export function MobsPage() {
  const [search, setSearch] = useState('');
  const [sortKey, setSortKey] = useState<keyof Mob>('level');
  const [sortAsc, setSortAsc] = useState(false);

  const { data: mobs, isLoading, error } = useQuery({
    queryKey: ['mobs'],
    queryFn: api.mobs,
  });

  const filtered = useMemo(() => {
    if (!mobs) return [];
    const q = search.toLowerCase();
    let list = q
      ? mobs.filter(
          (m) =>
            String(m.vnum).includes(q) ||
            m.short_desc.toLowerCase().includes(q) ||
            m.keywords.toLowerCase().includes(q)
        )
      : [...mobs];

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
  }, [mobs, search, sortKey, sortAsc]);

  const toggleSort = (key: keyof Mob) => {
    if (sortKey === key) {
      setSortAsc(!sortAsc);
    } else {
      setSortKey(key);
      setSortAsc(false);
    }
  };

  const SortHeader = ({
    label,
    field,
  }: {
    label: string;
    field: keyof Mob;
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
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Mobs</h1>
        {mobs && (
          <span className="text-sm text-slate-500 dark:text-slate-400">
            {filtered.length} of {mobs.length} mobs
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

      {isLoading && <TableSkeleton rows={6} cols={6} />}

      {error && (
        <div className="bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-700 rounded p-4 text-sm text-red-700 dark:text-red-300">
          Failed to load mobs.
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
                  <SortHeader label="Level" field="level" />
                  <SortHeader label="AC" field="ac" />
                  <SortHeader label="Gold" field="gold" />
                  <SortHeader label="EXP" field="exp" />
                </tr>
              </thead>
              <tbody>
                {filtered.map((mob) => (
                  <tr
                    key={mob.vnum}
                    className="border-b border-slate-100 dark:border-slate-700/50 hover:bg-slate-50 dark:hover:bg-slate-700/30 transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/game/mobs/${mob.vnum}`}
                        className="text-amber-600 dark:text-amber-400 hover:text-amber-500 dark:hover:text-amber-300 font-mono"
                      >
                        {mob.vnum}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-slate-900 dark:text-white">{mob.short_desc}</td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300 font-mono">
                      {mob.level}
                    </td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300 font-mono">
                      {mob.ac}
                    </td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300 font-mono">
                      {mob.gold.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-slate-600 dark:text-slate-300 font-mono">
                      {mob.exp.toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {mobs && mobs.length === 0 && (
        <div className="text-center text-slate-400 dark:text-slate-500 py-8">
          No mobs loaded. Check world files.
        </div>
      )}
    </div>
  );
}
