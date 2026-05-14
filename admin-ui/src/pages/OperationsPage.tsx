import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';

export function OperationsPage() {
  const [logLines, setLogLines] = useState(50);

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
                <div
                  key={p.name}
                  className="flex justify-between text-sm py-1 border-b border-slate-700/50 last:border-0"
                >
                  <span className="text-white">{p.name}</span>
                  <span className="text-slate-400">
                    Lv.{p.level} · Room {p.room}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

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
            disabled
            title="Zone reset trigger is not yet wired"
            className="bg-slate-700 text-slate-500 text-sm rounded px-4 py-2 border border-slate-600 cursor-not-allowed"
          >
            Zone Reset All
          </button>
          <button
            disabled
            title="Not yet implemented"
            className="bg-slate-700 text-slate-500 text-sm rounded px-4 py-2 border border-slate-600 cursor-not-allowed"
          >
            Save World State
          </button>
        </div>
      </div>
    </div>
  );
}
