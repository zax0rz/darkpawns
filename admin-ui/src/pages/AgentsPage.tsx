import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api, type AgentStatus, type Finding } from '../api/client';

function timeAgo(dateStr: string): string {
  const now = new Date();
  const date = new Date(dateStr);
  const diffMs = now.getTime() - date.getTime();
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return 'just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

function AgentCard({ agent }: { agent: AgentStatus }) {
  const statusColor =
    agent.status === 'active'
      ? 'bg-green-400'
      : agent.status === 'error'
        ? 'bg-red-400'
        : 'bg-yellow-400';

  return (
    <div className="bg-slate-800 rounded-lg border border-slate-700 p-5">
      <div className="flex items-center gap-3 mb-3">
        <div className={`w-3 h-3 rounded-full ${statusColor}`} />
        <h3 className="text-lg font-bold text-white">{agent.name}</h3>
      </div>
      <div className="space-y-1 text-sm">
        <div className="text-slate-400">
          <span className="text-slate-500">Status:</span>{' '}
          <span className={agent.status === 'active' ? 'text-green-400' : agent.status === 'error' ? 'text-red-400' : 'text-yellow-400'}>
            {agent.status}
          </span>
        </div>
        <div className="text-slate-400">
          <span className="text-slate-500">Model:</span>{' '}
          <span className="text-white font-mono text-xs">{agent.model}</span>
        </div>
        <div className="text-slate-400">
          <span className="text-slate-500">Role:</span>{' '}
          <span className="text-slate-300">{agent.description}</span>
        </div>
        <div className="text-slate-400">
          <span className="text-slate-500">Last run:</span>{' '}
          <span className="text-slate-300">{timeAgo(agent.last_run)}</span>
        </div>
      </div>
    </div>
  );
}

const severityStyles: Record<string, string> = {
  critical: 'bg-red-900 text-red-300 border-red-700',
  high: 'bg-orange-900 text-orange-300 border-orange-700',
  medium: 'bg-yellow-900 text-yellow-300 border-yellow-700',
  low: 'bg-slate-700 text-slate-300 border-slate-600',
};

const statusStyles: Record<string, string> = {
  open: 'bg-blue-900 text-blue-300',
  confirmed: 'bg-orange-900 text-orange-300',
  rejected: 'bg-slate-700 text-slate-400',
  fixed: 'bg-green-900 text-green-300',
};

function FindingRow({ finding }: { finding: Finding }) {
  return (
    <tr className="border-b border-slate-700/50 hover:bg-slate-700/30 transition-colors">
      <td className="px-4 py-3">
        <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium border ${severityStyles[finding.severity] || 'bg-slate-700 text-slate-300'}`}>
          {finding.severity.toUpperCase()}
        </span>
      </td>
      <td className="px-4 py-3 text-white font-mono text-sm">{finding.title}</td>
      <td className="px-4 py-3 text-slate-300 font-mono text-xs">{finding.file}:{finding.line}</td>
      <td className="px-4 py-3">
        <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${statusStyles[finding.status] || 'bg-slate-700 text-slate-300'}`}>
          {finding.status}
        </span>
      </td>
      <td className="px-4 py-3 text-slate-400 text-xs">{finding.source}</td>
    </tr>
  );
}

export function AgentsPage() {
  const [filterSource, setFilterSource] = useState('');
  const [filterStatus, setFilterStatus] = useState('');
  const [filterSeverity, setFilterSeverity] = useState('');

  const { data: agents, isLoading: agentsLoading } = useQuery({
    queryKey: ['agents'],
    queryFn: api.agents,
    refetchInterval: 30000,
  });

  const { data: findings, isLoading: findingsLoading, error: findingsError } = useQuery({
    queryKey: ['findings', filterStatus, filterSeverity, filterSource],
    queryFn: () => api.findings({ status: filterStatus || undefined, severity: filterSeverity || undefined, source: filterSource || undefined }),
    refetchInterval: 30000,
  });

  const { data: triages, isLoading: triagesLoading } = useQuery({
    queryKey: ['triageSummaries'],
    queryFn: api.triageSummaries,
  });

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-white">AI Agents</h1>

      {/* Agent Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {agentsLoading ? (
          <>
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-5 animate-pulse">
              <div className="h-6 w-32 bg-slate-700 rounded mb-3" />
              <div className="space-y-2">
                <div className="h-4 w-48 bg-slate-700 rounded" />
                <div className="h-4 w-40 bg-slate-700 rounded" />
              </div>
            </div>
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-5 animate-pulse">
              <div className="h-6 w-24 bg-slate-700 rounded mb-3" />
              <div className="space-y-2">
                <div className="h-4 w-44 bg-slate-700 rounded" />
                <div className="h-4 w-36 bg-slate-700 rounded" />
              </div>
            </div>
          </>
        ) : (
          agents?.map((agent) => (
            <AgentCard key={agent.agent_id} agent={agent} />
          ))
        )}
      </div>

      {/* Findings Feed */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-700 flex items-center justify-between">
          <h2 className="text-sm font-medium text-slate-300">Findings Feed</h2>
          <div className="flex gap-2">
            <select
              value={filterSource}
              onChange={(e) => setFilterSource(e.target.value)}
              className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1 border border-slate-600"
            >
              <option value="">Source: All</option>
              <option value="reek">Reek</option>
              <option value="daeron">Daeron</option>
            </select>
            <select
              value={filterSeverity}
              onChange={(e) => setFilterSeverity(e.target.value)}
              className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1 border border-slate-600"
            >
              <option value="">Severity: All</option>
              <option value="critical">Critical</option>
              <option value="high">High</option>
              <option value="medium">Medium</option>
              <option value="low">Low</option>
            </select>
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value)}
              className="bg-slate-700 text-slate-300 text-xs rounded px-2 py-1 border border-slate-600"
            >
              <option value="">Status: All</option>
              <option value="open">Open</option>
              <option value="confirmed">Confirmed</option>
              <option value="rejected">Rejected</option>
              <option value="fixed">Fixed</option>
            </select>
          </div>
        </div>

        {findingsLoading ? (
          <div className="p-6 text-center text-slate-500 animate-pulse">Loading findings...</div>
        ) : findingsError ? (
          <div className="p-6 text-center text-red-400 text-sm">
            Failed to load findings
          </div>
        ) : findings && findings.length > 0 ? (
          <table className="w-full">
            <thead>
              <tr className="border-b border-slate-700 text-xs text-slate-400 uppercase tracking-wider">
                <th className="text-left px-4 py-3">Severity</th>
                <th className="text-left px-4 py-3">Title</th>
                <th className="text-left px-4 py-3">Location</th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-left px-4 py-3">Source</th>
              </tr>
            </thead>
            <tbody>
              {findings.map((finding) => (
                <FindingRow key={finding.id} finding={finding} />
              ))}
            </tbody>
          </table>
        ) : (
          <div className="p-6 text-center text-slate-500 text-sm">
            No findings yet. Reek and Daeron will populate these via API.
          </div>
        )}
      </div>

      {/* Triage Summaries */}
      <div className="bg-slate-800 rounded-lg border border-slate-700 overflow-hidden">
        <div className="px-4 py-3 border-b border-slate-700">
          <h2 className="text-sm font-medium text-slate-300">Triage Summaries</h2>
        </div>

        {triagesLoading ? (
          <div className="p-6 text-center text-slate-500 animate-pulse">Loading triage summaries...</div>
        ) : triages && triages.length > 0 ? (
          <div className="divide-y divide-slate-700/50">
            {[...triages].reverse().map((triage) => (
              <div key={triage.id} className="px-4 py-3 hover:bg-slate-700/30 transition-colors">
                <div className="flex items-center gap-3 mb-1">
                  <span className="text-sm font-mono text-white">{triage.date}</span>
                  <span className="text-xs text-green-400">{triage.confirmed} confirmed</span>
                  <span className="text-xs text-red-400">{triage.rejected} rejected</span>
                  {triage.pending > 0 && (
                    <span className="text-xs text-yellow-400">{triage.pending} pending</span>
                  )}
                </div>
                {triage.summary && (
                  <div className="text-xs text-slate-400">{triage.summary}</div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="p-6 text-center text-slate-500 text-sm">
            No triage summaries yet. Daeron will post daily summaries here.
          </div>
        )}
      </div>
    </div>
  );
}
