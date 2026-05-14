import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

const SECTOR_LABELS: Record<number, string> = {
  0: 'Inside',
  1: 'City',
  2: 'Field',
  3: 'Forest',
  4: 'Hills',
  5: 'Mountain',
  6: 'Water (swim)',
  7: 'Water (noswim)',
  8: 'Underwater',
  9: 'Flying',
};

const FLAG_BITS: { bit: number; label: string }[] = [
  { bit: 4, label: 'PEACEFUL' },
  { bit: 9, label: 'PRIVATE' },
  { bit: 17, label: 'BFR' },
  { bit: 20, label: 'NOMAGIC' },
  { bit: 25, label: 'NO_MAGIC' },
];

function hexArrayToFlags(hexArr: string[]): Set<number> {
  const bits = new Set<number>();
  if (!hexArr || hexArr.length === 0) return bits;
  for (const hex of hexArr) {
    const num = parseInt(hex, 16);
    if (isNaN(num)) continue;
    for (let b = 0; b < 32; b++) {
      if (num & (1 << b)) {
        bits.add(b + hexArr.indexOf(hex) * 32);
      }
    }
  }
  return bits;
}

function flagsToHexArray(bits: Set<number>): string[] {
  const nums = [0, 0, 0, 0];
  for (const b of bits) {
    const idx = Math.floor(b / 32);
    const bit = b % 32;
    if (idx < 4) nums[idx] |= 1 << bit;
  }
  return nums.map((n) => '0x' + n.toString(16).padStart(8, '0'));
}

export function RoomEditPage() {
  const { vnum } = useParams<{ vnum: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: room, isLoading, error } = useQuery({
    queryKey: ['room', vnum],
    queryFn: () => api.room(Number(vnum)),
    enabled: !!vnum,
  });

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [sector, setSector] = useState(0);

  // Flags as a hex string (join of 4 hex strings)
  const [flagsRaw, setFlagsRaw] = useState('0x00000000, 0x00000000, 0x00000000, 0x00000000');
  const [flagBits, setFlagBits] = useState<Set<number>>(new Set());

  // Extra descs
  interface ExtraDesc {
    keywords: string;
    description: string;
  }
  const [extraDescs, setExtraDescs] = useState<ExtraDesc[]>([]);

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (room && !initialized) {
      setName(room.name);
      setDescription(room.description);
      setSector(room.sector ?? 0);

      if (room.flags && Array.isArray(room.flags)) {
        setFlagsRaw(room.flags.join(', '));
        const bits = hexArrayToFlags(room.flags);
        setFlagBits(bits);
      }

      setInitialized(true);
    }
  }, [room, initialized]);

  const toggleFlag = (bit: number) => {
    setFlagBits((prev) => {
      const next = new Set(prev);
      if (next.has(bit)) {
        next.delete(bit);
      } else {
        next.add(bit);
      }
      // Sync raw hex
      const hexArr = flagsToHexArray(next);
      setFlagsRaw(hexArr.join(', '));
      return next;
    });
  };

  const handleFlagsRawChange = (val: string) => {
    setFlagsRaw(val);
    // Parse and update bits from the raw string
    const parts = val.split(',').map((s) => s.trim()).filter(Boolean);
    const bits = hexArrayToFlags(parts);
    setFlagBits(bits);
  };

  const addExtraDesc = () => {
    setExtraDescs((prev) => [...prev, { keywords: '', description: '' }]);
  };

  const removeExtraDesc = (idx: number) => {
    setExtraDescs((prev) => prev.filter((_, i) => i !== idx));
  };

  const updateExtraDesc = (idx: number, field: 'keywords' | 'description', val: string) => {
    setExtraDescs((prev) => prev.map((ed, i) => (i === idx ? { ...ed, [field]: val } : ed)));
  };

  const handleSave = async () => {
    if (!vnum) return;
    setSaving(true);
    setSaveError('');
    try {
      const flagsParts = flagsRaw.split(',').map((s) => s.trim()).filter(Boolean);
      // Pad to 4 elements if needed
      while (flagsParts.length < 4) flagsParts.push('0x00000000');

      const data: Record<string, unknown> = {};
      if (name) data.name = name;
      if (description) data.description = description;
      data.sector = sector;
      data.flags = flagsParts;
      if (extraDescs.length > 0) {
        data.extra_descs = extraDescs;
      }

      await api.updateRoom(Number(vnum), data);
      queryClient.invalidateQueries({ queryKey: ['room', vnum] });
      navigate(`/admin/game/rooms/${vnum}`);
    } catch (err) {
      setSaveError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (isLoading) {
    return <div className="text-slate-400 animate-pulse">Loading room...</div>;
  }

  if (error || !room) {
    return (
      <div className="space-y-4">
        <Link to="/admin/game/zones" className="text-amber-400 hover:text-amber-300 text-sm">
          ← Back to Zones
        </Link>
        <div className="bg-red-900/30 border border-red-700 rounded p-4 text-sm text-red-300">
          Room not found or failed to load.
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link to={`/admin/game/rooms/${vnum}`} className="text-amber-400 hover:text-amber-300 text-sm">
        ← Back to Room
      </Link>

      <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
        <div className="flex items-baseline gap-3 mb-6">
          <span className="text-lg font-mono text-amber-400">#{room.vnum}</span>
          <h1 className="text-xl font-bold text-white">Edit Room</h1>
        </div>

        <div className="space-y-4">
          {/* Name */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={8}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500 resize-y"
            />
          </div>

          {/* Sector */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Sector Type</label>
            <select
              value={sector}
              onChange={(e) => setSector(Number(e.target.value))}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            >
              {Object.entries(SECTOR_LABELS).map(([val, label]) => (
                <option key={val} value={val}>
                  {val}: {label}
                </option>
              ))}
            </select>
          </div>

          {/* Flags */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-2">Flags</label>
            <div className="flex flex-wrap gap-3 mb-3">
              {FLAG_BITS.map((fb) => (
                <label key={fb.bit} className="flex items-center gap-2 text-sm text-slate-200 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={flagBits.has(fb.bit)}
                    onChange={() => toggleFlag(fb.bit)}
                    className="rounded border-slate-500 bg-slate-700 text-amber-500 focus:ring-amber-500"
                  />
                  {fb.label}
                </label>
              ))}
            </div>
            <label className="block text-xs text-slate-400 mb-1">Raw hex array (4-element)</label>
            <input
              type="text"
              value={flagsRaw}
              onChange={(e) => handleFlagsRawChange(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm font-mono focus:outline-none focus:border-amber-500"
            />
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
                No extra descriptions added. (Data may be read-only if server hasn't returned them.)
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
              onClick={() => navigate(`/admin/game/rooms/${vnum}`)}
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
