import { useState, useMemo } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Obj } from '../api/client';
import { TableSkeleton } from '../components/Skeleton';
import { itemTypeLabel } from '../lib/gameLabels';

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
    const list = q
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-ink">Objects</h1>
        {objects && (
          <span className="text-sm text-ink-muted">
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
        className="w-full px-4 py-2 rounded-none bg-paper-deep border border-rule text-ink placeholder-slate-400 focus:outline-none focus:border-accent text-sm"
      />

      {isLoading && <TableSkeleton rows={6} cols={5} />}

      {error && (
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Failed to load objects.
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
                  <SortHeader label="Type" field="type_flag" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="Weight" field="weight" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                  <SortHeader label="Cost" field="cost" sortKey={sortKey} sortAsc={sortAsc} onSort={toggleSort} />
                </tr>
              </thead>
              <tbody>
                {filtered.map((obj) => (
                  <tr
                    key={obj.vnum}
                    className="border-b border-rule hover:bg-paper-deep transition-colors"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/admin/game/objects/${obj.vnum}`}
                        className="text-accent hover:text-accent font-mono"
                      >
                        {obj.vnum}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-ink">{obj.short_desc}</td>
                    <td className="px-4 py-3 text-ink-muted">
                      {itemTypeLabel(obj.type_flag)}
                    </td>
                    <td className="px-4 py-3 text-ink-muted font-mono">
                      {obj.weight}
                    </td>
                    <td className="px-4 py-3 text-ink-muted font-mono">
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
        <div className="text-center text-ink-muted py-8">
          No objects loaded. Check world files.
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
  field: keyof Obj;
  sortKey: keyof Obj;
  sortAsc: boolean;
  onSort: (field: keyof Obj) => void;
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
