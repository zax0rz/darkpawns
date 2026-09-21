import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { api, type Shop } from '../api/client';
import { TableSkeleton } from '../components/Skeleton';
import { NewEntityPanel } from '../components/olc/NewEntityPanel';

export function ShopsPage() {
  const [search, setSearch] = useState('');
  const { data: shops, isLoading, error } = useQuery({
    queryKey: ['shops'],
    queryFn: api.shops,
  });
  const filtered = useMemo(() => {
    if (!shops) return [];
    const query = search.trim().toLowerCase();
    if (!query) return shops;
    return shops.filter((shop) => [shop.vnum, shop.keeper_vnum, shop.keeper_name || ''].some((value) => String(value).toLowerCase().includes(query)));
  }, [search, shops]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-2xl font-bold text-ink">Shops</h1>
        {shops && <span className="text-sm text-ink-muted">{filtered.length} of {shops.length} shops</span>}
      </div>

      <NewEntityPanel kind="shop" />

      <input
        type="search"
        value={search}
        onChange={(event) => setSearch(event.currentTarget.value)}
        placeholder="Filter by shop, keeper, or name..."
        className="w-full border border-rule bg-paper-deep px-4 py-2 text-sm text-ink placeholder-ink-muted focus:border-accent focus:outline-none"
        aria-label="Filter shops"
      />

      {isLoading && <TableSkeleton rows={6} cols={6} />}
      {error && <div className="border border-accent bg-paper-deep p-4 text-sm text-accent" role="alert">Failed to load shops.<div className="mt-1 text-xs">{(error as Error).message}</div></div>}

      {shops && filtered.length > 0 && (
        <div className="overflow-hidden border border-rule bg-paper-deep">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-rule text-left text-xs uppercase tracking-wider text-ink-muted">
                  <th className="px-4 py-3">Shop #</th>
                  <th className="px-4 py-3">Keeper</th>
                  <th className="px-4 py-3">Room</th>
                  <th className="px-4 py-3">Buy rate</th>
                  <th className="px-4 py-3">Sell rate</th>
                  <th className="px-4 py-3">Edit</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((shop) => <ShopRow key={`${shop.vnum}-${shop.keeper_vnum}`} shop={shop} />)}
              </tbody>
            </table>
          </div>
        </div>
      )}
      {shops && filtered.length === 0 && <p className="py-8 text-center text-ink-muted">No matching shops.</p>}
    </div>
  );
}

function ShopRow({ shop }: { shop: Shop }) {
  return (
    <tr className="border-b border-rule hover:bg-paper-deep">
      <td className="px-4 py-3 font-mono text-accent">{shop.vnum || '—'}</td>
      <td className="px-4 py-3 text-ink">#{shop.keeper_vnum} {shop.keeper_name || 'Unknown keeper'}</td>
      <td className="px-4 py-3 font-mono text-ink-muted">{shop.room_vnum || '—'}</td>
      <td className="px-4 py-3 font-mono text-ink-muted">{shop.profit_buy.toFixed(2)}×</td>
      <td className="px-4 py-3 font-mono text-ink-muted">{shop.profit_sell.toFixed(2)}×</td>
      <td className="px-4 py-3"><Link to={`/admin/game/shops/${shop.vnum}/edit`} className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep">Edit</Link></td>
    </tr>
  );
}
