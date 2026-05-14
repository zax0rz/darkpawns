import { useQuery } from '@tanstack/react-query';
import { useParams, Link } from 'react-router-dom';
import { api } from '../api/client';

const sectorLabels = [
  'Inside',
  'City',
  'Field',
  'Forest',
  'Hills',
  'Mountain',
  'Water (Swim)',
  'Water (No Swim)',
  'Underwater',
  'Flying',
];

function sectorLabel(sector: number): string {
  return sectorLabels[sector] || `Sector ${sector}`;
}

export function RoomDetailPage() {
  const { vnum } = useParams<{ vnum: string }>();

  const { data: room, isLoading, error } = useQuery({
    queryKey: ['room', vnum],
    queryFn: () => api.room(Number(vnum)),
    enabled: !!vnum,
  });

  if (isLoading) {
    return <div className="text-slate-400 animate-pulse">Loading room...</div>;
  }

  if (error || !room) {
    return (
      <div className="space-y-4">
        <Link
          to="/admin/game/zones"
          className="text-amber-400 hover:text-amber-300 text-sm"
        >
          ← Back to Zones
        </Link>
        <div className="bg-red-900/30 border border-red-700 rounded p-4 text-sm text-red-300">
          Room not found or failed to load.
          <div className="mt-1 text-red-400/70 text-xs">
            {(error as Error)?.message || 'Not found'}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <Link
        to="/admin/game/zones"
        className="text-amber-400 hover:text-amber-300 text-sm"
      >
        ← Back to Zones
      </Link>

      {/* Header */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-amber-400">#{room.vnum}</span>
          <h1 className="text-xl font-bold text-white">{room.name}</h1>
          <Link
            to={`/admin/game/rooms/${room.vnum}/edit`}
            className="bg-amber-600 hover:bg-amber-500 text-white px-3 py-1 rounded text-sm ml-auto"
          >
            Edit
          </Link>
        </div>
      </div>

      {/* Meta */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
        <h2 className="text-sm font-medium text-slate-300 mb-4">Properties</h2>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <StatBlock label="Zone" value={room.zone} />
          <StatBlock label="Sector" value={sectorLabel(room.sector)} />
          <StatBlock
            label="Flags"
            value={room.flags.join(', ') || 'None'}
          />
        </div>
      </div>

      {/* Description */}
      {room.description && (
        <div className="bg-slate-800 rounded-lg border border-slate-700 p-6">
          <h2 className="text-sm font-medium text-slate-300 mb-2">
            Description
          </h2>
          <p className="text-sm text-slate-200 italic whitespace-pre-wrap">
            {room.description}
          </p>
        </div>
      )}
    </div>
  );
}

function StatBlock({
  label,
  value,
}: {
  label: string;
  value: string | number;
}) {
  return (
    <div>
      <div className="text-xs text-slate-500 mb-1">{label}</div>
      <div className="text-sm text-white font-mono">{value}</div>
    </div>
  );
}
