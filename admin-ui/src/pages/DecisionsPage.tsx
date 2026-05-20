import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api, type DecisionRow } from '../api/client';

function timeAgo(dateStr: string): string {
  const now = new Date();
  const date = new Date(dateStr);
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

const outcomeStyles: Record<string, string> = {
  movement: 'bg-blue-900 text-blue-300',
  combat_hit_taken: 'bg-red-900 text-red-300',
  combat_started: 'bg-orange-900 text-orange-300',
  combat_ended: 'bg-slate-700 text-slate-300',
  healed: 'bg-green-900 text-green-300',
  agent_died: 'bg-red-900 text-red-400 font-bold',
  level_up: 'bg-yellow-900 text-yellow-300',
  item_acquired: 'bg-cyan-900 text-cyan-300',
  item_dropped: 'bg-slate-700 text-slate-400',
  error: 'bg-red-950 text-red-400',
  no_change: 'bg-slate-800 text-slate-500',
};

const classStyles: Record<string, string> = {
  movement: 'bg-blue-800 text-blue-200',
  combat: 'bg-red-800 text-red-200',
  inventory: 'bg-cyan-800 text-cyan-200',
  social: 'bg-purple-800 text-purple-200',
  info: 'bg-slate-700 text-slate-300',
  magic: 'bg-violet-800 text-violet-200',
  system: 'bg-amber-800 text-amber-200',
  other: 'bg-slate-700 text-slate-400',
};

function DecisionRowView({ row }: { row: DecisionRow }) {
  const hpChanged = row.pre_health !== row.post_health;
  const roomChanged = row.pre_room !== row.post_room;

  return (
    <tr className="border-b border-slate-700/50 hover:bg-slate-700/30 transition-colors">
      <td className="px-3 py-2 text-slate-500 text-xs font-mono">{row.id}</td>
      <td className="px-3 py-2 text-slate-400 text-xs">{timeAgo(row.ts)}</td>
      <td className="px-3 py-2">
        <span className={`inline-block px-1.5 py-0.5 rounded text-xs font-medium ${classStyles[row.command_class || 'other'] || 'bg-slate-700 text-slate-300'}`}>
          {row.command_class || '—'}
        </span>
      </td>
      <td className="px-3 py-2 text-white font-mono text-sm">{row.command}</td>
      <td className="px-3 py-2">
        <span className={`inline-block px-1.5 py-0.5 rounded text-xs font-medium ${outcomeStyles[row.outcome_category] || 'bg-slate-700 text-slate-300'}`}>
          {row.outcome_category}
        </span>
      </td>
      <td className="px-3 py-2 text-xs">
        {row.pre_room !== null && (
          <span className={roomChanged ? 'text-yellow-400' : 'text-slate-400'}>
            Room {row.pre_room}{roomChanged ? ` → ${row.post_room}` : ''}
          </span>
        )}
      </td>
      <td className="px-3 py-2 text-xs">
        {row.pre_health !== null && (
          <span className={hpChanged ? (row.post_health !== null && row.post_health < (row.pre_health || 0) ? 'text-red-400' : 'text-green-400') : 'text-slate-400'}>
            {row.pre_health}/{row.pre_max_health}{hpChanged ? ` → ${row.post_health}` : ''}
          </span>
        )}
      </td>
      <td className="px-3 py-2">
        {row.is_agent ? (
          <span className="text-xs font-mono text-green-400">{row.agent_harness}/{row.agent_model}</span>
        ) : (
          <span className="text-xs text-slate-500">{row.player_name}</span>
        )}
      </td>
      <td className="px-3 py-2 text-slate-500 text-xs">{row.turn_number}</td>
      <td className="px-3 py-2 text-slate-500 text-xs">{row.duration_ms !== null ? `${row.duration_ms.toFixed(1)}ms` : '—'}</td>
    </tr>
  );
}

export function DecisionsPage() {
  const [sessionId, setSessionId] = useState('');
  const [playerName, setPlayerName] = useState('');
  const [isAgent, setIsAgent] = useState('');
  const [commandClass, setCommandClass] = useState('');
  const [outcome, setOutcome] = useState('');
  const [harness, setHarness] = useState('');
  const [page, setPage] = useState(0);
  const limit = 50;

  const { data, isLoading } = useQuery({
    queryKey: ['decisions', sessionId, playerName, isAgent, commandClass, outcome, harness, page],
    queryFn: () => api.decisions({
      session_id: sessionId || undefined,
      player_name: playerName || undefined,
      is_agent: isAgent || undefined,
      command_class: commandClass || undefined,
      outcome: outcome || undefined,
      harness: harness || undefined,
      limit,
      offset: page * limit,
    }),
    refetchInterval: 10000,
  });

  const totalPages = data ? Math.ceil(data.total / limit) : 0;

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold text-white">Decision Log</h1>

      {/* Filters */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <input
            type="text"
            placeholder="Session ID"
            value={sessionId}
            onChange={(e) => { setSessionId(e.target.value); setPage(0); }}
            className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1.5 border border-slate-600"
          />
          <input
            type="text"
            placeholder="Player name"
            value={playerName}
            onChange={(e) => { setPlayerName(e.target.value); setPage(0); }}
            className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1.5 border border-slate-600"
          />
          <select
            value={isAgent}
            onChange={(e) => { setIsAgent(e.target.value); setPage(0); }}
            className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1.5 border border-slate-600"
          >
            <option value="">All players</option>
            <option value="true">Agents only</option>
            <option value="false">Humans only</option>
          </select>
          <select
            value={commandClass}
            onChange={(e) => { setCommandClass(e.target.value); setPage(0); }}
            className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1.5 border border-slate-600"
          >
            <option value="">All commands</option>
            <option value="movement">Movement</option>
            <option value="combat">Combat</option>
            <option value="inventory">Inventory</option>
            <option value="social">Social</option>
            <option value="info">Info</option>
            <option value="magic">Magic</option>
            <option value="system">System</option>
          </select>
          <select
            value={outcome}
            onChange={(e) => { setOutcome(e.target.value); setPage(0); }}
            className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1.5 border border-slate-600"
          >
            <option value="">All outcomes</option>
            <option value="movement">Movement</option>
            <option value="combat_hit_taken">Combat hit taken</option>
            <option value="combat_started">Combat started</option>
            <option value="agent_died">Agent died</option>
            <option value="level_up">Level up</option>
            <option value="item_acquired">Item acquired</option>
            <option value="error">Error</option>
            <option value="no_change">No change</option>
          </select>
          <select
            value={harness}
            onChange={(e) => { setHarness(e.target.value); setPage(0); }}
            className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1.5 border border-slate-600"
          >
            <option value="">All harnesses</option>
            <option value="openclaw">OpenClaw</option>
            <option value="claude-code">Claude Code</option>
            <option value="gemini-cli">Gemini CLI</option>
            <option value="dp-agent">dp-agent</option>
          </select>
        </div>
      </div>

      {/* Results */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
        {isLoading ? (
          <div className="p-6 text-center text-slate-500 animate-pulse">Loading decisions...</div>
        ) : data && data.data.length > 0 ? (
          <>
            <div className="px-4 py-2 border-b border-slate-700 text-xs text-slate-400">
              {data.total.toLocaleString()} decisions {data.total > limit ? `(showing ${page * limit + 1}–${Math.min((page + 1) * limit, data.total)} of ${data.total.toLocaleString()})` : ''}
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-slate-700 text-xs text-slate-400 uppercase tracking-wider">
                    <th className="text-left px-3 py-2">ID</th>
                    <th className="text-left px-3 py-2">Time</th>
                    <th className="text-left px-3 py-2">Class</th>
                    <th className="text-left px-3 py-2">Command</th>
                    <th className="text-left px-3 py-2">Outcome</th>
                    <th className="text-left px-3 py-2">Room</th>
                    <th className="text-left px-3 py-2">HP</th>
                    <th className="text-left px-3 py-2">Player</th>
                    <th className="text-left px-3 py-2">Turn</th>
                    <th className="text-left px-3 py-2">ms</th>
                  </tr>
                </thead>
                <tbody>
                  {data.data.map((row) => (
                    <DecisionRowView key={row.id} row={row} />
                  ))}
                </tbody>
              </table>
            </div>
            {/* Pagination */}
            <div className="px-4 py-3 border-t border-slate-700 flex justify-between items-center">
              <button
                onClick={() => setPage(Math.max(0, page - 1))}
                disabled={page === 0}
                className="px-3 py-1 text-xs rounded bg-slate-700 text-slate-300 hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                ← Previous
              </button>
              <span className="text-xs text-slate-400">
                Page {page + 1} of {totalPages}
              </span>
              <button
                onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
                disabled={page >= totalPages - 1}
                className="px-3 py-1 text-xs rounded bg-slate-700 text-slate-300 hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Next →
              </button>
            </div>
          </>
        ) : (
          <div className="p-6 text-center text-slate-500 text-sm">
            No decisions logged yet. Decision capture activates when agents connect.
          </div>
        )}
      </div>
    </div>
  );
}
