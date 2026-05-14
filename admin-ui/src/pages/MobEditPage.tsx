import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

const POSITION_LABELS: Record<number, string> = {
  0: 'Dead',
  1: 'Mortally Wounded',
  2: 'Incapacitated',
  3: 'Stunned',
  4: 'Sleeping',
  5: 'Resting',
  6: 'Sitting',
  7: 'Fighting',
  8: 'Standing',
};

const SEX_LABELS: Record<number, string> = {
  0: 'Male',
  1: 'Female',
  2: 'Neutral',
};

function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (v: number) => void }) {
  return (
    <div>
      <label className="block text-sm font-medium text-slate-300 mb-1">{label}</label>
      <input
        type="number"
        value={value}
        onChange={(e) => onChange(Number(e.target.value))}
        className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
      />
    </div>
  );
}

export function MobEditPage() {
  const { vnum } = useParams<{ vnum: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: mob, isLoading, error } = useQuery({
    queryKey: ['mob', vnum],
    queryFn: () => api.mob(Number(vnum)),
    enabled: !!vnum,
  });

  const [shortDesc, setShortDesc] = useState('');
  const [longDesc, setLongDesc] = useState('');
  const [keywords, setKeywords] = useState('');
  const [level, setLevel] = useState(0);
  const [ac, setAc] = useState(0);
  const [hpNumDice, setHpNumDice] = useState(0);
  const [hpSizeDice, setHpSizeDice] = useState(0);
  const [hpAdd, setHpAdd] = useState(0);
  const [gold, setGold] = useState(0);
  const [exp, setExp] = useState(0);
  const [alignment, setAlignment] = useState(0);

  // New fields
  const [actionFlags, setActionFlags] = useState('');
  const [affectFlags, setAffectFlags] = useState('');
  const [str, setStr] = useState(0);
  const [int, setInt] = useState(0);
  const [wis, setWis] = useState(0);
  const [dex, setDex] = useState(0);
  const [con, setCon] = useState(0);
  const [cha, setCha] = useState(0);
  const [thac0, setThac0] = useState(0);
  const [dmgNumDice, setDmgNumDice] = useState(0);
  const [dmgSizeDice, setDmgSizeDice] = useState(0);
  const [dmgAdd, setDmgAdd] = useState(0);
  const [position, setPosition] = useState(8);
  const [defaultPos, setDefaultPos] = useState(8);
  const [sex, setSex] = useState(0);
  const [race, setRace] = useState(0);

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (mob && !initialized) {
      setShortDesc(mob.short_desc);
      setLongDesc(mob.long_desc);
      setKeywords(mob.keywords ?? '');
      setLevel(mob.level);
      setAc(mob.ac);
      // Parse HP dice: "3d8+10" → numDice=3, sizeDice=8, add=10
      const hpMatch = mob.hp.match(/(\d+)d(\d+)([+-]\d+)?/);
      if (hpMatch) {
        setHpNumDice(Number(hpMatch[1]));
        setHpSizeDice(Number(hpMatch[2]));
        setHpAdd(Number(hpMatch[3]));
      }
      setGold(mob.gold);
      setExp(mob.exp);
      setAlignment(mob.alignment);

      // New fields
      setActionFlags(Array.isArray(mob.action_flags) ? mob.action_flags.join(', ') : '');
      setAffectFlags(Array.isArray(mob.affect_flags) ? mob.affect_flags.join(', ') : '');
      setStr(mob.str ?? 0);
      setInt(mob.int ?? 0);
      setWis(mob.wis ?? 0);
      setDex(mob.dex ?? 0);
      setCon(mob.con ?? 0);
      setCha(mob.cha ?? 0);
      setPosition(mob.position ?? 8);
      setDefaultPos(mob.default_pos ?? 8);
      setSex(mob.sex ?? 0);
      setRace(mob.race ?? 0);

      setInitialized(true);
    }
  }, [mob, initialized]);

  const handleSave = async () => {
    if (!vnum) return;
    setSaving(true);
    setSaveError('');
    try {
      const data: Record<string, unknown> = {};
      if (shortDesc) data.short_desc = shortDesc;
      if (longDesc) data.long_desc = longDesc;
      if (keywords) data.keywords = keywords;
      data.level = level;
      data.ac = ac;
      data.hp_num_dice = hpNumDice;
      data.hp_size_dice = hpSizeDice;
      data.hp_add = hpAdd;
      data.gold = gold;
      data.exp = exp;
      data.alignment = alignment;

      // New fields
      if (actionFlags) {
        data.action_flags = actionFlags.split(',').map((s) => s.trim()).filter(Boolean);
      }
      if (affectFlags) {
        data.affect_flags = affectFlags.split(',').map((s) => s.trim()).filter(Boolean);
      }
      data.str = str;
      data.int = int;
      data.wis = wis;
      data.dex = dex;
      data.con = con;
      data.cha = cha;
      data.thac0 = thac0;
      data.dmg_num_dice = dmgNumDice;
      data.dmg_size_dice = dmgSizeDice;
      data.dmg_add = dmgAdd;
      data.position = position;
      data.default_pos = defaultPos;
      data.sex = sex;
      data.race = race;

      await api.updateMob(Number(vnum), data);
      queryClient.invalidateQueries({ queryKey: ['mob', vnum] });
      navigate(`/admin/game/mobs/${vnum}`);
    } catch (err) {
      setSaveError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (isLoading) {
    return <div className="text-slate-400 animate-pulse">Loading mob...</div>;
  }

  if (error || !mob) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/mobs" className="text-amber-400 hover:text-amber-300 text-sm">
          ← Back to Mobs
        </Link>
        <div className="bg-red-900/30 border border-red-700 rounded p-4 text-sm text-red-300">
          Mob not found or failed to load.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to={`/admin/game/mobs/${vnum}`} className="text-amber-400 hover:text-amber-300 text-sm">
        ← Back to Mob
      </Link>

      <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
        <div className="flex items-baseline gap-3 mb-6">
          <span className="text-lg font-mono text-amber-400">#{mob.vnum}</span>
          <h1 className="text-xl font-bold text-white">Edit Mob</h1>
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

          {/* Core Stats Grid */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <NumberField label="Level" value={level} onChange={setLevel} />
            <NumberField label="AC" value={ac} onChange={setAc} />
            <NumberField label="Gold" value={gold} onChange={setGold} />
            <NumberField label="EXP" value={exp} onChange={setExp} />
          </div>

          {/* HP Dice */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Hit Points (Dice)</label>
            <div className="grid grid-cols-3 gap-4">
              <NumberField label="Num Dice" value={hpNumDice} onChange={setHpNumDice} />
              <NumberField label="Size Dice" value={hpSizeDice} onChange={setHpSizeDice} />
              <NumberField label="Add HP" value={hpAdd} onChange={setHpAdd} />
            </div>
            <div className="text-xs text-slate-400 mt-1 font-mono">
              {hpNumDice}d{hpSizeDice}+{hpAdd}
            </div>
          </div>

          {/* Damage Dice */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Damage Dice</label>
            <div className="grid grid-cols-3 gap-4">
              <NumberField label="Num Dice" value={dmgNumDice} onChange={setDmgNumDice} />
              <NumberField label="Size Dice" value={dmgSizeDice} onChange={setDmgSizeDice} />
              <NumberField label="Add HP" value={dmgAdd} onChange={setDmgAdd} />
            </div>
            <div className="text-xs text-slate-400 mt-1 font-mono">
              {dmgNumDice}d{dmgSizeDice}+{dmgAdd}
            </div>
          </div>

          {/* Stats Grid */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Stats</label>
            <div className="grid grid-cols-3 md:grid-cols-6 gap-4">
              <NumberField label="Str" value={str} onChange={setStr} />
              <NumberField label="Int" value={int} onChange={setInt} />
              <NumberField label="Wis" value={wis} onChange={setWis} />
              <NumberField label="Dex" value={dex} onChange={setDex} />
              <NumberField label="Con" value={con} onChange={setCon} />
              <NumberField label="Cha" value={cha} onChange={setCha} />
            </div>
          </div>

          {/* THAC0 */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <NumberField label="THAC0" value={thac0} onChange={setThac0} />
          </div>

          {/* Position */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Position</label>
            <select
              value={position}
              onChange={(e) => setPosition(Number(e.target.value))}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            >
              {Object.entries(POSITION_LABELS).map(([val, label]) => (
                <option key={val} value={val}>
                  {val}: {label}
                </option>
              ))}
            </select>
          </div>

          {/* Default Position */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Default Position</label>
            <select
              value={defaultPos}
              onChange={(e) => setDefaultPos(Number(e.target.value))}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            >
              {Object.entries(POSITION_LABELS).map(([val, label]) => (
                <option key={val} value={val}>
                  {val}: {label}
                </option>
              ))}
            </select>
          </div>

          {/* Sex + Race */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">Sex</label>
              <select
                value={sex}
                onChange={(e) => setSex(Number(e.target.value))}
                className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
              >
                {Object.entries(SEX_LABELS).map(([val, label]) => (
                  <option key={val} value={val}>
                    {val}: {label}
                  </option>
                ))}
              </select>
            </div>
            <NumberField label="Race" value={race} onChange={setRace} />
          </div>

          {/* Alignment */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">Alignment</label>
              <input
                type="number"
                value={alignment}
                onChange={(e) => setAlignment(Math.max(-1000, Math.min(1000, Number(e.target.value))))}
                min={-1000}
                max={1000}
                className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
              />
              <div className="text-xs text-slate-500 mt-1">
                {alignment === 0 ? 'Neutral' : alignment > 0 ? 'Good' : 'Evil'} ({alignment})
              </div>
            </div>
          </div>

          {/* Action Flags */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Action Flags <span className="text-xs text-slate-500">(comma-separated hex, e.g. 0x00000280)</span>
            </label>
            <input
              type="text"
              value={actionFlags}
              onChange={(e) => setActionFlags(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Affect Flags */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Affect Flags <span className="text-xs text-slate-500">(comma-separated hex)</span>
            </label>
            <input
              type="text"
              value={affectFlags}
              onChange={(e) => setAffectFlags(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-amber-500"
            />
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
              onClick={() => navigate(`/admin/game/mobs/${vnum}`)}
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
