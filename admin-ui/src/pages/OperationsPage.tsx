import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { useToast } from '../hooks/useToast';
import { PlayerDetailModal } from '../components/PlayerDetailModal';
import { MetricsCard } from '../components/MetricsCard';

// The buffer writes the level first (pkg/admin/log_buffer.go writes
// record.Level.String() ahead of the timestamp), so a line reads
// "WARN 2026-09-17T12:25:31-04:00 cannot spawn mob: ...". Reading the level
// lets the log be scanned by weight without adding a column: errors take the
// accent, warnings full ink, the rest stays muted. Fifty zone-reset warnings
// at one weight is a wall, not a log.
function logLineClass(line: string): string {
  const level = /^\s*([A-Z]+)\b/.exec(line)?.[1];
  if (level === 'ERROR' || level === 'FATAL') return 'text-accent';
  if (level === 'WARN') return 'text-ink';
  return 'text-ink-muted';
}

export function OperationsPage() {
  const [logLines, setLogLines] = useState(50);
  const [selectedPlayer, setSelectedPlayer] = useState<string | null>(null);
  const { showToast } = useToast();
  const queryClient = useQueryClient();

  const {
    data: server,
    isLoading: serverLoading,
    error: serverError,
  } = useQuery({
    queryKey: ['server'],
    queryFn: api.server,
  });

  const {
    data: players,
    isLoading: playersLoading,
  } = useQuery({
    queryKey: ['players'],
    queryFn: api.players,
  });

  const {
    data: logs,
    isLoading: logsLoading,
    refetch: refetchLogs,
  } = useQuery({
    queryKey: ['logs', logLines],
    queryFn: () => api.logs(logLines),
    refetchInterval: 10000,
  });

  const saveWorldMutation = useMutation({
    mutationFn: () => api.saveWorld(),
    onSuccess: (data) => {
      showToast(`World state saved: ${data.status}`, 'success');
    },
    onError: (err: Error) => {
      showToast(`Save failed: ${err.message}`, 'error');
    },
  });

  const resetZonesMutation = useMutation({
    mutationFn: () => api.resetAllZones(),
    onSuccess: (data) => {
      showToast(`Zone reset triggered: ${data.zones_reset}/${data.zones_total} zones`, data.errors?.length ? 'error' : 'success');
      queryClient.invalidateQueries({ queryKey: ['server'] });
    },
    onError: (err: Error) => {
      showToast(`Zone reset failed: ${err.message}`, 'error');
    },
  });

  const handleSaveWorld = () => {
    if (window.confirm('Save the entire world state? This may take a moment.')) {
      saveWorldMutation.mutate();
    }
  };

  const handleResetZones = () => {
    if (window.confirm('Reset all zones? This will respawn all mobs and objects.')) {
      resetZonesMutation.mutate();
    }
  };

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-ink">Operations</h1>

      {/* Server Status + Online Players */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Server Status */}
        <div className="bg-paper-deep rounded-none border border-rule p-4">
          <h2 className="text-sm font-medium text-ink-muted mb-3">Server Status</h2>
          {serverLoading ? (
            <div className="text-sm text-ink-muted animate-pulse">Loading...</div>
          ) : serverError ? (
            <div className="text-sm text-accent">Failed to load server info</div>
          ) : (
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-ink-muted">Rooms</span>
                <span className="text-ink font-mono">{server?.room_count}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-muted">Players Online</span>
                <span className="text-ink font-mono">{server?.player_count}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-ink-muted">Zones</span>
                <span className="text-ink font-mono">{server?.zone_count}</span>
              </div>
              {server?.uptime && (
                <div className="flex justify-between">
                  <span className="text-ink-muted">Uptime</span>
                  <span className="text-ink font-mono text-xs">{server.uptime}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Online Players */}
        <div className="bg-paper-deep rounded-none border border-rule p-4">
          <h2 className="text-sm font-medium text-ink-muted mb-3">Online Players</h2>
          {playersLoading ? (
            <div className="text-sm text-ink-muted animate-pulse">Loading...</div>
          ) : !players || players.length === 0 ? (
            <div className="text-sm text-ink-muted">No players online</div>
          ) : (
            <div className="space-y-1 max-h-48 overflow-y-auto">
              {players.map((p) => (
                <button
                  key={p.name}
                  onClick={() => setSelectedPlayer(p.name)}
                  className="w-full flex justify-between text-sm py-1 border-b border-rule last:border-0 hover:bg-paper transition-colors rounded px-1 text-left"
                >
                  <span className="text-accent hover:text-accent">{p.name}</span>
                  <span className="text-ink-muted">
                    Lv.{p.level} · Room {p.room}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Server Metrics */}
      <MetricsCard />

      {/* Server Log */}
      <div className="bg-paper-deep rounded-none border border-rule p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-medium text-ink-muted">
            Server Log (last {logLines} lines)
          </h2>
          <div className="flex gap-2">
            <select
              value={logLines}
              onChange={(e) => setLogLines(Number(e.target.value))}
              className="bg-paper text-ink-muted text-xs rounded px-2 py-1 border border-rule"
            >
              <option value={25}>25</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
              <option value={200}>200</option>
            </select>
            <button
              onClick={() => refetchLogs()}
              className="bg-paper text-ink-muted text-xs rounded px-3 py-1 border border-rule hover:bg-paper transition-colors"
            >
              Refresh
            </button>
          </div>
        </div>
        <div className="bg-paper rounded border border-rule p-3 max-h-96 overflow-y-auto">
          {logsLoading ? (
            <div className="text-sm text-ink-muted animate-pulse">Loading logs...</div>
          ) : !logs || logs.length === 0 ? (
            <div className="text-sm text-ink-muted font-mono">No log entries yet</div>
          ) : (
            <pre className="text-xs font-mono whitespace-pre-wrap">
              {logs.map((line, i) => (
                <div key={i} className={`hover:bg-paper-deep ${logLineClass(line)}`}>
                  {line}
                </div>
              ))}
            </pre>
          )}
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-paper-deep rounded-none border border-rule p-4">
        <h2 className="text-sm font-medium text-ink-muted mb-3">Quick Actions</h2>
        <div className="flex gap-3">
          <button
            onClick={handleResetZones}
            disabled={resetZonesMutation.isPending}
            className="bg-accent hover:bg-accent disabled:bg-paper disabled:text-ink-muted text-ink text-sm rounded px-4 py-2 border border-accent disabled:border-rule transition-colors"
          >
            {resetZonesMutation.isPending ? 'Resetting...' : 'Zone Reset All'}
          </button>
          <button
            onClick={handleSaveWorld}
            disabled={saveWorldMutation.isPending}
            className="bg-paper-deep hover:bg-paper-deep disabled:bg-paper disabled:text-ink-muted text-ink text-sm rounded px-4 py-2 border border-rule disabled:border-rule transition-colors"
          >
            {saveWorldMutation.isPending ? 'Saving...' : 'Save World State'}
          </button>
        </div>
      </div>

      {/* Player Detail Modal */}
      {selectedPlayer && (
        <PlayerDetailModal
          playerName={selectedPlayer}
          onClose={() => setSelectedPlayer(null)}
        />
      )}
    </div>
  );
}
