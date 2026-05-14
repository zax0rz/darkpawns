import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';

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
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState('');
  const [initialized, setInitialized] = useState(false);

  useEffect(() => {
    if (room && !initialized) {
      setName(room.name);
      setDescription(room.description);
      setInitialized(true);
    }
  }, [room, initialized]);

  const handleSave = async () => {
    if (!vnum) return;
    setSaving(true);
    setSaveError('');
    try {
      await api.updateRoom(Number(vnum), { name, description });
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
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={8}
              className="w-full bg-slate-900 border border-slate-600 rounded px-3 py-2 text-white text-sm focus:outline-none focus:border-amber-500 resize-y"
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
