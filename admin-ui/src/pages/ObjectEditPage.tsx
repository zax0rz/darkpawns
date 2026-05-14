import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

export function ObjectEditPage() {
  const { vnum } = useParams<{ vnum: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: obj, isLoading, error } = useQuery({
    queryKey: ['object', vnum],
    queryFn: () => api.object(Number(vnum)),
    enabled: !!vnum,
  });

  const [shortDesc, setShortDesc] = useState('');
  const [longDesc, setLongDesc] = useState('');
  const [weight, setWeight] = useState(0);
  const [cost, setCost] = useState(0);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (obj && !initialized) {
      setShortDesc(obj.short_desc);
      setLongDesc(obj.long_desc);
      setWeight(obj.weight);
      setCost(obj.cost);
      setInitialized(true);
    }
  }, [obj, initialized]);

  const handleSave = async () => {
    if (!vnum) return;
    setSaving(true);
    setSaveError('');
    try {
      const data: Record<string, unknown> = {};
      if (shortDesc) data.short_desc = shortDesc;
      if (longDesc) data.long_desc = longDesc;
      data.weight = weight;
      data.cost = cost;

      await api.updateObject(Number(vnum), data);
      queryClient.invalidateQueries({ queryKey: ['object', vnum] });
      navigate(`/admin/game/objects/${vnum}`);
    } catch (err) {
      setSaveError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (isLoading) {
    return <div className="text-slate-400 animate-pulse">Loading object...</div>;
  }

  if (error || !obj) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/objects" className="text-amber-400 hover:text-amber-300 text-sm">
          ← Back to Objects
        </Link>
        <div className="bg-red-900/30 border border-red-700 rounded p-4 text-sm text-red-300">
          Object not found or failed to load.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to={`/admin/game/objects/${vnum}`} className="text-amber-400 hover:text-amber-300 text-sm">
        ← Back to Object
      </Link>

      <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
        <div className="flex items-baseline gap-3 mb-6">
          <span className="text-lg font-mono text-amber-400">#{obj.vnum}</span>
          <h1 className="text-xl font-bold text-white">Edit Object</h1>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Short Description</label>
            <input
              type="text"
              value={shortDesc}
              onChange={(e) => setShortDesc(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Long Description</label>
            <textarea
              value={longDesc}
              onChange={(e) => setLongDesc(e.target.value)}
              rows={4}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500 resize-y"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">Weight</label>
              <input
                type="number"
                value={weight}
                onChange={(e) => setWeight(Number(e.target.value))}
                className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">Cost</label>
              <input
                type="number"
                value={cost}
                onChange={(e) => setCost(Number(e.target.value))}
                className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
              />
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
              onClick={() => navigate(`/admin/game/objects/${vnum}`)}
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
