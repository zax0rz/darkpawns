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
    const list = q
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-ink">Mobs</h1>
        {mobs && (
          <span className="text-sm text-ink-muted">
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
        className="w-full px-4 py-2 rounded-none bg-paper-deep border border-rule text-ink placeholder-slate-400 focus:outline-none focus:border-accent text-sm"
      />

      {isLoading && <TableSkeleton rows={6} cols={6} />}

      {error && (
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Failed to load mobs.
          <div className="mt-1 text-accent text-xs">
            {(error as Error).message}
          </div>
        </div>
      )}

      {filtered.length > 0 && (
        <div className="bg-paper-deep rounded-none border border-rule overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-rule text-xs text-ink-muted uppercase tracking-wider">
                  <SortHeader label="VNum" field="vnum" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="Name" field="short_desc" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="Level" field="level" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="AC" field="ac" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="Gold" field="gold" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="EXP" field="exp" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                </tr>
              </thead>
              <tbody>
                {filtered.map((mob) => (
                  <tr
                    key={mob.vnum}
                    className="border-b border-rule hover:bg-paper-deep transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/game/mobs/${mob.vnum}`}
                        className="text-accent hover:text-accent font-mono"
                      >
                        {mob.vnum}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-ink">{mob.short_desc}</td>
                    <td className="px-4 py-3 text-ink-muted font-mono">
                      {mob.level}
                    </td>
                    <td className="px-4 py-3 text-ink-muted font-mono">
                      {mob.ac}
                    </td>
                    <td className="px-4 py-3 text-ink-muted font-mono">
                      {mob.gold.toLocaleString()}
                    </td>
                    <td className="px-4 py-3 text-ink-muted font-mono">
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
        <div className="text-center text-ink-muted py-8">
          No mobs loaded. Check world files.
        </div>
      )}
    </div>
  );
}

function SortHeader({
  label,
  field,
  sortKey,
  sortAsc,
  onSort,
}: {
  label: string;
  field: keyof Mob;
  sortKey: keyof Mob;
  sortAsc: boolean;
  onSort: (field: keyof Mob) => void;
}) {
  return (
    <th
      className="cursor-pointer select-none px-4 py-3 text-left hover:text-accent"
      onClick={() => onSort(field)}
    >
      {label}
      {sortKey === field && <span className="ml-1 text-accent">{sortAsc ? '↑' : '↓'}</span>}
    </th>
  );
}
