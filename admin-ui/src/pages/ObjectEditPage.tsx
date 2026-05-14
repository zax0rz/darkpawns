import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

const TYPE_FLAG_LABELS: Record<number, string> = {
  0: 'Unused',
  1: 'Light',
  2: 'Scroll',
  3: 'Wand',
  4: 'Staff',
  5: 'Weapon',
  6: 'Fire Weapon',
  7: 'Missile',
  8: 'Treasure',
  9: 'Potion',
  10: 'Worn',
  11: 'Trash',
  12: 'Container',
  13: 'Note',
  14: 'Drink Container',
  15: 'Key',
  16: 'Food',
  17: 'Money',
  18: 'Pen',
  19: 'Boat',
  20: 'Fountain',
  21: 'Climbable',
  22: 'Item1',
  23: 'Item2',
  24: 'Generator',
  25: 'Altar',
  26: 'Planar',
  27: 'Portal',
};

interface Affect {
  location: number;
  modifier: number;
}

interface ExtraDesc {
  keywords: string;
  description: string;
}

function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (v: number) => void }) {
  return (
    <div>
      <label className="block text-xs text-slate-400 mb-1">{label}</label>
      <input
        type="number"
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full bg-slate-900 border border-slate-600 rounded px-2 py-1 text-white text-sm focus:outline-none focus:border-amber-500"
      />
    </div>
  );
}

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
  const [keywords, setKeywords] = useState('');
  const [weight, setWeight] = useState(0);
  const [cost, setCost] = useState(0);
  const [typeFlag, setTypeFlag] = useState(0);
  const [values, setValues] = useState<number[]>([0, 0, 0, 0]);
  const [wearFlagsStr, setWearFlagsStr] = useState('');
  const [extraFlagsStr, setExtraFlagsStr] = useState('');
  const [affects, setAffects] = useState<Affect[]>([]);
  const [extraDescs, setExtraDescs] = useState<ExtraDesc[]>([]);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (obj && !initialized) {
      setShortDesc(obj.short_desc);
      setLongDesc(obj.long_desc);
      setKeywords(obj.keywords ?? '');
      setWeight(obj.weight);
      setCost(obj.cost);
      setTypeFlag(obj.type_flag ?? 0);
      setValues(obj.values ?? [0, 0, 0, 0]);
      setWearFlagsStr(Array.isArray(obj.wear_flags) ? obj.wear_flags.join(', ') : '');
      setExtraFlagsStr(Array.isArray(obj.extra_flags) ? obj.extra_flags.join(', ') : '');
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
      if (keywords) data.keywords = keywords;
      data.weight = weight;
      data.cost = cost;
      data.type_flag = typeFlag;
      data.values = values;

      if (wearFlagsStr) {
        data.wear_flags = wearFlagsStr.split(',').map((s) => s.trim()).filter(Boolean).map(Number);
      }
      if (extraFlagsStr) {
        data.extra_flags = extraFlagsStr.split(',').map((s) => s.trim()).filter(Boolean).map(Number);
      }
      if (affects.length > 0) {
        data.affects = affects;
      }
      if (extraDescs.length > 0) {
        data.extra_descs = extraDescs;
      }

      await api.updateObject(Number(vnum), data);
      queryClient.invalidateQueries({ queryKey: ['object', vnum] });
      navigate(`/admin/game/objects/${vnum}`);
    } catch (err) {
      setSaveError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const updateValue = (idx: number, val: number) => {
    setValues((prev) => prev.map((v, i) => (i === idx ? val : v)));
  };

  const addAffect = () => setAffects((prev) => [...prev, { location: 0, modifier: 0 }]);
  const removeAffect = (idx: number) => setAffects((prev) => prev.filter((_, i) => i !== idx));
  const updateAffect = (idx: number, field: 'location' | 'modifier', val: number) => {
    setAffects((prev) => prev.map((a, i) => (i === idx ? { ...a, [field]: val } : a)));
  };

  const addExtraDesc = () => setExtraDescs((prev) => [...prev, { keywords: '', description: '' }]);
  const removeExtraDesc = (idx: number) => setExtraDescs((prev) => prev.filter((_, i) => i !== idx));
  const updateExtraDesc = (idx: number, field: 'keywords' | 'description', val: string) => {
    setExtraDescs((prev) => prev.map((ed, i) => (i === idx ? { ...ed, [field]: val } : ed)));
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
          {/* Keywords */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Keywords</label>
            <input
              type="text"
              value={keywords}
              onChange={(e) => setKeywords(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Short Description */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Short Description</label>
            <input
              type="text"
              value={shortDesc}
              onChange={(e) => setShortDesc(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Long Description */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Long Description</label>
            <textarea
              value={longDesc}
              onChange={(e) => setLongDesc(e.target.value)}
              rows={4}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500 resize-y"
            />
          </div>

          {/* Type Flag */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Type Flag</label>
            <select
              value={typeFlag}
              onChange={(e) => setTypeFlag(Number(e.target.value))}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            >
              {Object.entries(TYPE_FLAG_LABELS).map(([val, label]) => (
                <option key={val} value={val}>
                  {val}: {label}
                </option>
              ))}
            </select>
          </div>

          {/* Weight & Cost */}
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

          {/* Values[4] */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">
              Values <span className="text-xs text-slate-500">(meaning depends on Type Flag)</span>
            </label>
            <div className="grid grid-cols-4 gap-3">
              {[0, 1, 2, 3].map((idx) => (
                <NumberField key={idx} label={`Value ${idx}`} value={values[idx] ?? 0} onChange={(v) => updateValue(idx, v)} />
              ))}
            </div>
          </div>

          {/* Wear Flags */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Wear Flags <span className="text-xs text-slate-500">(comma-separated [4]int)</span>
            </label>
            <input
              type="text"
              value={wearFlagsStr}
              onChange={(e) => setWearFlagsStr(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Extra Flags */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Extra Flags <span className="text-xs text-slate-500">(comma-separated [4]int)</span>
            </label>
            <input
              type="text"
              value={extraFlagsStr}
              onChange={(e) => setExtraFlagsStr(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Affects */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="block text-sm font-medium text-slate-300">Affects</label>
              <button
                onClick={addAffect}
                className="text-xs text-amber-400 hover:text-amber-300 px-2 py-1 border border-amber-600/50 rounded"
              >
                + Add Affect
              </button>
            </div>
            {affects.map((aff, idx) => (
              <div key={idx} className="bg-slate-900/50 border border-slate-600 rounded p-3 mb-2">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-slate-400 font-mono">Affect #{idx + 1}</span>
                  <button
                    onClick={() => removeAffect(idx)}
                    className="text-xs text-red-400 hover:text-red-300"
                  >
                    Remove
                  </button>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <NumberField label="Location" value={aff.location} onChange={(v) => updateAffect(idx, 'location', v)} />
                  <NumberField label="Modifier" value={aff.modifier} onChange={(v) => updateAffect(idx, 'modifier', v)} />
                </div>
              </div>
            ))}
          </div>

          {/* Extra Descs */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="block text-sm font-medium text-slate-300">Extra Descriptions</label>
              <button
                onClick={addExtraDesc}
                className="text-xs text-amber-400 hover:text-amber-300 px-2 py-1 border border-amber-600/50 rounded"
              >
                + Add Extra Desc
              </button>
            </div>
            {extraDescs.length === 0 && (
              <p className="text-xs text-slate-500 italic">
                No extra descriptions added.
              </p>
            )}
            {extraDescs.map((ed, idx) => (
              <div key={idx} className="bg-slate-900/50 border border-slate-600 rounded p-3 mb-2">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-xs text-slate-400 font-mono">#{idx + 1}</span>
                  <button
                    onClick={() => removeExtraDesc(idx)}
                    className="text-xs text-red-400 hover:text-red-300"
                  >
                    Remove
                  </button>
                </div>
                <div className="space-y-2">
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Keywords</label>
                    <input
                      type="text"
                      value={ed.keywords}
                      onChange={(e) => updateExtraDesc(idx, 'keywords', e.target.value)}
                      className="w-full bg-slate-950 border border-slate-600 rounded px-2 py-1 text-white text-sm focus:outline-none focus:border-amber-500"
                    />
                  </div>
                  <div>
                    <label className="block text-xs text-slate-400 mb-1">Description</label>
                    <textarea
                      value={ed.description}
                      onChange={(e) => updateExtraDesc(idx, 'description', e.target.value)}
                      rows={3}
                      className="w-full bg-slate-950 border border-slate-600 rounded px-2 py-1 text-white text-sm focus:outline-none focus:border-amber-500 resize-y"
                    />
                  </div>
                </div>
              </div>
            ))}
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
