import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { useToast } from '../components/Toast';
import { PlayerDetailModal } from '../components/PlayerDetailModal';
import { MetricsCard } from '../components/MetricsCard';

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
      showToast(`Zone reset triggered: ${data.count} zones`, 'success');
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
      <h1 className="text-2xl font-bold text-white">Operations</h1>

      {/* Server Status + Online Players */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Server Status */}
        <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
          <h2 className="text-sm font-medium text-slate-300 mb-3">Server Status</h2>
          {serverLoading ? (
            <div className="text-sm text-slate-500 animate-pulse">Loading...</div>
          ) : serverError ? (
            <div className="text-sm text-red-400">Failed to load server info</div>
          ) : (
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-slate-400">Rooms</span>
                <span className="text-white font-mono">{server?.room_count}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Players Online</span>
                <span className="text-white font-mono">{server?.player_count}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-400">Zones</span>
                <span className="text-white font-mono">{server?.zone_count}</span>
              </div>
              {server?.uptime && (
                <div className="flex justify-between">
                  <span className="text-slate-400">Uptime</span>
                  <span className="text-white font-mono text-xs">{server.uptime}</span>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Online Players */}
        <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
          <h2 className="text-sm font-medium text-slate-300 mb-3">Online Players</h2>
          {playersLoading ? (
            <div className="text-sm text-slate-500 animate-pulse">Loading...</div>
          ) : !players || players.length === 0 ? (
            <div className="text-sm text-slate-500">No players online</div>
          ) : (
            <div className="space-y-1 max-h-48 overflow-y-auto">
              {players.map((p) => (
                <button
                  key={p.name}
                  onClick={() => setSelectedPlayer(p.name)}
                  className="w-full flex justify-between text-sm py-1 border-b border-slate-700/50 last:border-0 hover:bg-slate-700/50 transition-colors rounded px-1 text-left"
                >
                  <span className="text-amber-400 hover:text-amber-300">{p.name}</span>
                  <span className="text-slate-400">
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
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-medium text-slate-300">
            Server Log (last {logLines} lines)
          </h2>
          <div className="flex gap-2">
            <select
              value={logLines}
              onChange={(e) => setLogLines(Number(e.target.value))}
              className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1 border border-slate-600"
            >
              <option value={25}>25</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
              <option value={200}>200</option>
            </select>
            <button
              onClick={() => refetchLogs()}
              className="bg-slate-700 text-slate-300 text-xs rounded px-3 py-1 border border-slate-600 hover:bg-slate-600 transition-colors"
            >
              Refresh
            </button>
          </div>
        </div>
        <div className="bg-slate-900 rounded border border-slate-700 p-3 max-h-96 overflow-y-auto">
          {logsLoading ? (
            <div className="text-sm text-slate-500 animate-pulse">Loading logs...</div>
          ) : !logs || logs.length === 0 ? (
            <div className="text-sm text-slate-500 font-mono">No log entries yet</div>
          ) : (
            <pre className="text-xs text-slate-300 font-mono whitespace-pre-wrap">
              {logs.map((line, i) => (
                <div key={i} className="hover:bg-slate-800/50">
                  {line}
                </div>
              ))}
            </pre>
          )}
        </div>
      </div>

      {/* Quick Actions */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
        <h2 className="text-sm font-medium text-slate-300 mb-3">Quick Actions</h2>
        <div className="flex gap-3">
          <button
            onClick={handleResetZones}
            disabled={resetZonesMutation.isPending}
            className="bg-amber-700 hover:bg-amber-600 disabled:bg-slate-700 disabled:text-slate-500 text-white text-sm rounded px-4 py-2 border border-amber-600 disabled:border-slate-600 transition-colors"
          >
            {resetZonesMutation.isPending ? 'Resetting...' : 'Zone Reset All'}
          </button>
          <button
            onClick={handleSaveWorld}
            disabled={saveWorldMutation.isPending}
            className="bg-blue-700 hover:bg-blue-600 disabled:bg-slate-700 disabled:text-slate-500 text-white text-sm rounded px-4 py-2 border border-blue-600 disabled:border-slate-600 transition-colors"
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
