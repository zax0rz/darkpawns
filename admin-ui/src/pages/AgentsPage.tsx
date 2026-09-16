import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api, type AgentStatus, type Finding, type LiveAgentSession } from '../api/client';

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
      ? 'bg-online'
      : agent.status === 'error'
        ? 'bg-accent'
        : 'bg-paper-deep';

  return (
    <div className="bg-paper-deep rounded-none border border-rule p-5">
      <div className="flex items-center gap-3 mb-3">
        <div className={`w-3 h-3 rounded-none ${statusColor}`} />
        <h3 className="text-lg font-bold text-ink">{agent.name}</h3>
      </div>
      <div className="space-y-1 text-sm">
        <div className="text-ink-muted">
          <span className="text-ink-muted">Status:</span>{' '}
          <span className={agent.status === 'active' ? 'text-online' : agent.status === 'error' ? 'text-accent' : 'text-ink-muted'}>
            {agent.status}
          </span>
        </div>
        <div className="text-ink-muted">
          <span className="text-ink-muted">Model:</span>{' '}
          <span className="text-ink font-mono text-xs">{agent.model}</span>
        </div>
        <div className="text-ink-muted">
          <span className="text-ink-muted">Role:</span>{' '}
          <span className="text-ink-muted">{agent.description}</span>
        </div>
        <div className="text-ink-muted">
          <span className="text-ink-muted">Last run:</span>{' '}
          <span className="text-ink-muted">{timeAgo(agent.last_run)}</span>
        </div>
      </div>
    </div>
  );
}

const severityStyles: Record<string, string> = {
  critical: 'bg-paper-deep text-accent border-accent',
  high: 'bg-paper-deep text-ink-muted border-rule',
  medium: 'bg-paper-deep text-ink-muted border-rule',
  low: 'bg-paper text-ink-muted border-rule',
};

const statusStyles: Record<string, string> = {
  open: 'bg-paper-deep text-ink-muted',
  confirmed: 'bg-paper-deep text-ink-muted',
  rejected: 'bg-paper text-ink-muted',
  fixed: 'bg-paper-deep text-ink',
};

function FindingRow({ finding }: { finding: Finding }) {
  return (
    <tr className="border-b border-rule hover:bg-paper transition-colors">
      <td className="px-4 py-3">
        <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium border ${severityStyles[finding.severity] || 'bg-paper text-ink-muted'}`}>
          {finding.severity.toUpperCase()}
        </span>
      </td>
      <td className="px-4 py-3 text-ink font-mono text-sm">{finding.title}</td>
      <td className="px-4 py-3 text-ink-muted font-mono text-xs">{finding.file}:{finding.line}</td>
      <td className="px-4 py-3">
        <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium ${statusStyles[finding.status] || 'bg-paper text-ink-muted'}`}>
          {finding.status}
        </span>
      </td>
      <td className="px-4 py-3 text-ink-muted text-xs">{finding.source}</td>
    </tr>
  );
}

function LiveAgentRow({ session }: { session: LiveAgentSession }) {
  const connectedAgo = timeAgo(session.connected_at);
  return (
    <tr className="border-b border-rule hover:bg-paper transition-colors">
      <td className="px-4 py-3 text-ink font-mono text-sm">{session.player_name}</td>
      <td className="px-4 py-3">
        <span className="inline-block px-2 py-0.5 rounded text-xs font-medium bg-paper-deep text-ink border border-rule">
          connected
        </span>
      </td>
      <td className="px-4 py-3 text-ink-muted font-mono text-xs">{session.harness}</td>
      <td className="px-4 py-3 text-ink-muted font-mono text-xs">{session.model}</td>
      <td className="px-4 py-3 text-ink-muted text-xs">{session.level > 0 ? `Lvl ${session.level}` : '—'}</td>
      <td className="px-4 py-3 text-ink-muted text-xs">Room {session.room_vnum}</td>
      <td className="px-4 py-3 text-ink-muted text-xs">{connectedAgo}</td>
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

  const { data: liveSessions, isLoading: liveLoading } = useQuery({
    queryKey: ['liveAgentSessions'],
    queryFn: api.liveAgentSessions,
    refetchInterval: 5000,
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
      <h1 className="text-2xl font-bold text-ink">AI Agents</h1>

      {/* Live Game Agent Sessions */}
      <div className="bg-paper-deep rounded-none border border-rule overflow-hidden">
        <div className="px-4 py-3 border-b border-rule">
          <h2 className="text-sm font-medium text-ink-muted">Live Game Sessions</h2>
        </div>
        {liveLoading ? (
          <div className="p-6 text-center text-ink-muted animate-pulse">Loading live sessions...</div>
        ) : liveSessions && liveSessions.length > 0 ? (
          <table className="w-full">
            <thead>
              <tr className="border-b border-rule text-xs text-ink-muted uppercase tracking-wider">
                <th className="text-left px-4 py-3">Player</th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-left px-4 py-3">Harness</th>
                <th className="text-left px-4 py-3">Model</th>
                <th className="text-left px-4 py-3">Level</th>
                <th className="text-left px-4 py-3">Room</th>
                <th className="text-left px-4 py-3">Connected</th>
              </tr>
            </thead>
            <tbody>
              {liveSessions.map((session) => (
                <LiveAgentRow key={session.player_name} session={session} />
              ))}
            </tbody>
          </table>
        ) : (
          <div className="p-6 text-center text-ink-muted text-sm">
            No agents currently connected to the game server.
          </div>
        )}
      </div>

      {/* Agent Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {agentsLoading ? (
          <>
            <div className="bg-paper-deep rounded-none border border-rule p-5 animate-pulse">
              <div className="h-6 w-32 bg-paper rounded mb-3" />
              <div className="space-y-2">
                <div className="h-4 w-48 bg-paper rounded" />
                <div className="h-4 w-40 bg-paper rounded" />
              </div>
            </div>
            <div className="bg-paper-deep rounded-none border border-rule p-5 animate-pulse">
              <div className="h-6 w-24 bg-paper rounded mb-3" />
              <div className="space-y-2">
                <div className="h-4 w-44 bg-paper rounded" />
                <div className="h-4 w-36 bg-paper rounded" />
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
      <div className="bg-paper-deep rounded-none border border-rule overflow-hidden">
        <div className="px-4 py-3 border-b border-rule flex items-center justify-between">
          <h2 className="text-sm font-medium text-ink-muted">Findings Feed</h2>
          <div className="flex gap-2">
            <select
              value={filterSource}
              onChange={(e) => setFilterSource(e.target.value)}
              className="bg-paper text-ink-muted text-xs rounded px-2 py-1 border border-rule"
            >
              <option value="">Source: All</option>
              <option value="reek">Reek</option>
              <option value="daeron">Daeron</option>
            </select>
            <select
              value={filterSeverity}
              onChange={(e) => setFilterSeverity(e.target.value)}
              className="bg-paper text-ink-muted text-xs rounded px-2 py-1 border border-rule"
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
              className="bg-paper text-ink-muted text-xs rounded px-2 py-1 border border-rule"
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
          <div className="p-6 text-center text-ink-muted animate-pulse">Loading findings...</div>
        ) : findingsError ? (
          <div className="p-6 text-center text-accent text-sm">
            Failed to load findings
          </div>
        ) : findings && findings.length > 0 ? (
          <table className="w-full">
            <thead>
              <tr className="border-b border-rule text-xs text-ink-muted uppercase tracking-wider">
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
          <div className="p-6 text-center text-ink-muted text-sm">
            No findings yet. Reek and Daeron will populate these via API.
          </div>
        )}
      </div>

      {/* Triage Summaries */}
      <div className="bg-paper-deep rounded-none border border-rule overflow-hidden">
        <div className="px-4 py-3 border-b border-rule">
          <h2 className="text-sm font-medium text-ink-muted">Triage Summaries</h2>
        </div>

        {triagesLoading ? (
          <div className="p-6 text-center text-ink-muted animate-pulse">Loading triage summaries...</div>
        ) : triages && triages.length > 0 ? (
          <div className="divide-y divide-rule">
            {[...triages].reverse().map((triage) => (
              <div key={triage.id} className="px-4 py-3 hover:bg-paper transition-colors">
                <div className="flex items-center gap-3 mb-1">
                  <span className="text-sm font-mono text-ink">{triage.date}</span>
                  <span className="text-xs text-online">{triage.confirmed} confirmed</span>
                  <span className="text-xs text-accent">{triage.rejected} rejected</span>
                  {triage.pending > 0 && (
                    <span className="text-xs text-ink-muted">{triage.pending} pending</span>
                  )}
                </div>
                {triage.summary && (
                  <div className="text-xs text-ink-muted">{triage.summary}</div>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="p-6 text-center text-ink-muted text-sm">
            No triage summaries yet. Daeron will post daily summaries here.
          </div>
        )}
      </div>
    </div>
  );
}
