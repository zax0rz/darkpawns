import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

export function ShopEditPage() {
  const { keeperVnum } = useParams<{ keeperVnum: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: shop, isLoading, error } = useQuery({
    queryKey: ['shop', keeperVnum],
    queryFn: () => api.shop(Number(keeperVnum)),
    enabled: !!keeperVnum,
  });

  const [buyTypes, setBuyTypes] = useState<number[]>([]);
  const [sellTypes, setSellTypes] = useState<number[]>([]);
  const [profitBuy, setProfitBuy] = useState(0);
  const [profitSell, setProfitSell] = useState(0);
  const [newBuyType, setNewBuyType] = useState('');
  const [newSellType, setNewSellType] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (shop && !initialized) {
      setBuyTypes(shop.buy_types ?? []);
      setSellTypes(shop.sell_types ?? []);
      setProfitBuy(shop.profit_buy ?? 0);
      setProfitSell(shop.profit_sell ?? 0);
      setInitialized(true);
    }
  }, [shop, initialized]);

  const addBuyType = () => {
    const val = parseInt(newBuyType, 10);
    if (!isNaN(val) && !buyTypes.includes(val)) {
      setBuyTypes((prev) => [...prev, val]);
    }
    setNewBuyType('');
  };

  const removeBuyType = (val: number) => {
    setBuyTypes((prev) => prev.filter((v) => v !== val));
  };

  const addSellType = () => {
    const val = parseInt(newSellType, 10);
    if (!isNaN(val) && !sellTypes.includes(val)) {
      setSellTypes((prev) => [...prev, val]);
    }
    setNewSellType('');
  };

  const removeSellType = (val: number) => {
    setSellTypes((prev) => prev.filter((v) => v !== val));
  };

  const handleSave = async () => {
    if (!keeperVnum) return;
    setSaving(true);
    setSaveError('');
    try {
      await api.updateShop(Number(keeperVnum), {
        buy_types: buyTypes,
        sell_types: sellTypes,
        profit_buy: profitBuy,
        profit_sell: profitSell,
      });
      queryClient.invalidateQueries({ queryKey: ['shop', keeperVnum] });
      navigate('/admin/game/mobs');
    } catch (err) {
      setSaveError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (isLoading) {
    return <div className="text-slate-400 animate-pulse">Loading shop...</div>;
  }

  if (error || !shop) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/mobs" className="text-amber-400 hover:text-amber-300 text-sm">
          ← Back to Mobs
        </Link>
        <div className="bg-red-900/30 border border-red-700 rounded p-4 text-sm text-red-300">
          Shop not found or failed to load.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to="/admin/game/mobs" className="text-amber-400 hover:text-amber-300 text-sm">
        ← Back to Mobs
      </Link>

      <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
        <div className="flex items-baseline gap-3 mb-6">
          <span className="text-lg font-mono text-amber-400">#{shop.keeper_vnum}</span>
          <h1 className="text-xl font-bold text-white">
            Shop — Keeper #{shop.keeper_vnum}
          </h1>
        </div>

        {/* Keeper name / Room */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div>
            <label className="block text-sm font-medium text-slate-400 mb-1">Keeper Name</label>
            <div className="text-white text-sm font-mono bg-slate-900 rounded px-3 py-2 border border-slate-600">
              {shop.keeper_name || 'Unknown'}
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-400 mb-1">Room VNum</label>
            <div className="text-white text-sm font-mono bg-slate-900 rounded px-3 py-2 border border-slate-600">
              {shop.room_vnum ?? 'Unknown'}
            </div>
          </div>
        </div>

        <div className="space-y-4">
          {/* Profit Buy */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Profit Buy <span className="text-xs text-slate-500">(e.g. 1.20 = 120%)</span>
            </label>
            <input
              type="number"
              value={profitBuy}
              onChange={(e) => setProfitBuy(Number(e.target.value))}
              step={0.01}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Profit Sell */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Profit Sell <span className="text-xs text-slate-500">(e.g. 0.80 = 80%)</span>
            </label>
            <input
              type="number"
              value={profitSell}
              onChange={(e) => setProfitSell(Number(e.target.value))}
              step={0.01}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Buy Types */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Buy Types (item type integers)</label>
            <div className="flex flex-wrap gap-2 mb-2">
              {buyTypes.map((t) => (
                <span
                  key={t}
                  className="inline-flex items-center gap-1 px-2 py-1 bg-slate-700 rounded text-xs text-slate-200"
                >
                  {t}
                  <button
                    onClick={() => removeBuyType(t)}
                    className="text-red-400 hover:text-red-300 ml-1"
                  >
                    x
                  </button>
                </span>
              ))}
              {buyTypes.length === 0 && (
                <span className="text-xs text-slate-500 italic">None</span>
              )}
            </div>
            <div className="flex gap-2">
              <input
                type="number"
                value={newBuyType}
                onChange={(e) => setNewBuyType(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addBuyType(); } }}
                placeholder="Type number..."
                className="flex-1 bg-slate-900 border border-slate-600 rounded px-2 py-1 text-white text-sm focus:outline-none focus:border-amber-500"
              />
              <button
                onClick={addBuyType}
                className="text-xs text-amber-400 hover:text-amber-300 px-2 py-1 border border-amber-600/50 rounded"
              >
                Add
              </button>
            </div>
          </div>

          {/* Sell Types */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Sell Types (item type integers)</label>
            <div className="flex flex-wrap gap-2 mb-2">
              {sellTypes.map((t) => (
                <span
                  key={t}
                  className="inline-flex items-center gap-1 px-2 py-1 bg-slate-700 rounded text-xs text-slate-200"
                >
                  {t}
                  <button
                    onClick={() => removeSellType(t)}
                    className="text-red-400 hover:text-red-300 ml-1"
                  >
                    x
                  </button>
                </span>
              ))}
              {sellTypes.length === 0 && (
                <span className="text-xs text-slate-500 italic">None</span>
              )}
            </div>
            <div className="flex gap-2">
              <input
                type="number"
                value={newSellType}
                onChange={(e) => setNewSellType(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); addSellType(); } }}
                placeholder="Type number..."
                className="flex-1 bg-slate-900 border border-slate-600 rounded px-2 py-1 text-white text-sm focus:outline-none focus:border-amber-500"
              />
              <button
                onClick={addSellType}
                className="text-xs text-amber-400 hover:text-amber-300 px-2 py-1 border border-amber-600/50 rounded"
              >
                Add
              </button>
            </div>
          </div>

          {saveError && (
            <div className="bg-red-900/30 border border-red-700 rounded p-3 text-sm text-red-300">
              {saveError}
            </div>
          )}

          <div className="flex gap-3 pt-2">
            <button
              onClick={handleSave}
              disabled={saving}
              className="bg-amber-600 hover:bg-amber-500 disabled:opacity-50 disabled:cursor-not-allowed text-white px-4 py-2 rounded text-sm font-medium"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button
              onClick={() => navigate('/admin/game/mobs')}
              className="bg-slate-700 hover:bg-slate-600 text-white px-4 py-2 rounded text-sm"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
