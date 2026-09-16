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
    return <div className="text-ink-muted animate-pulse">Loading room...</div>;
  }

  if (error || !room) {
    return (
      <div className="space-y-4">
        <Link
          to="/admin/game/zones"
          className="text-accent hover:text-accent text-sm"
        >
          ← Back to Zones
        </Link>
        <div className="bg-paper-deep border border-accent rounded p-4 text-sm text-accent">
          Room not found or failed to load.
          <div className="mt-1 text-accent text-xs">
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
        className="text-accent hover:text-accent text-sm"
      >
        ← Back to Zones
      </Link>

      {/* Header */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <div className="flex items-baseline gap-3">
          <span className="text-lg font-mono text-accent">#{room.vnum}</span>
          <h1 className="text-xl font-bold text-ink">{room.name}</h1>
        </div>
      </div>

      {/* Meta */}
      <div className="bg-paper-deep rounded-none border border-rule p-6">
        <h2 className="text-sm font-medium text-ink-muted mb-4">Properties</h2>
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
        <div className="bg-paper-deep rounded-none border border-rule p-6">
          <h2 className="text-sm font-medium text-ink-muted mb-2">
            Description
          </h2>
          <p className="text-sm text-ink italic whitespace-pre-wrap">
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
      <div className="text-xs text-ink-muted mb-1">{label}</div>
      <div className="text-sm text-ink font-mono">{value}</div>
    </div>
  );
}
