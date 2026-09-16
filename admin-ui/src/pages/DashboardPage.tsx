import { useQuery } from '@tanstack/react-query';
import { Icon } from '../components/Icon';
import { api, type AgentStatus, type Finding } from '../api/client';
import { StatCardSkeleton } from '../components/Skeleton';


export function DashboardPage() {
  const {
    data: server,
    isLoading: serverLoading,
    error: serverError,
  } = useQuery({
    queryKey: ['server'],
    queryFn: api.server,
  });

  const {
    data: health,
    isLoading: healthLoading,
  } = useQuery({
    queryKey: ['health'],
    queryFn: api.health,
    refetchInterval: 30000,
  });

  const {
    data: agents,
    isLoading: agentsLoading,
  } = useQuery({
    queryKey: ['agents'],
    queryFn: api.agents,
    refetchInterval: 30000,
  });

  const {
    data: findings,
    isLoading: findingsLoading,
  } = useQuery({
    queryKey: ['findings'],
    queryFn: () => api.findings(),
    refetchInterval: 30000,
  });

  // Compute stats from findings
  const totalFindings = findings?.length || 0;
  const openCount = findings?.filter(f => f.status === 'open').length || 0;
  const confirmedCount = findings?.filter(f => f.status === 'confirmed').length || 0;
  const fixedCount = findings?.filter(f => f.status === 'fixed').length || 0;
  const criticalHighCount = findings?.filter(f => f.severity === 'critical' || f.severity === 'high').length || 0;

  // Limit to latest 5 findings for the card
  const latestFindings = findings?.slice(-5).reverse() || [];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-ink">Dashboard</h1>

      {/* Server Status */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {(serverLoading || healthLoading) ? (
          <>
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
            <StatCardSkeleton />
          </>
        ) : (
          <>
            <StatCard
              label="Status"
              value={health?.status || '...'}
              error={!!serverError}
              color={health?.status === 'ok' ? 'green' : 'slate'}
            />
            <StatCard
              label="Zones"
              value={server?.zone_count?.toString() || '...'}
              error={!!serverError}
            />
            <StatCard
              label="Rooms"
              value={server?.room_count?.toString() || '...'}
              error={!!serverError}
            />
            <StatCard
              label="Players"
              value={server?.player_count?.toString() || '...'}
              error={!!serverError}
            />
          </>
        )}
      </div>

      {/* Findings Stats Row */}
      <div className="flex flex-wrap gap-2">
        <StatPill label="Total" value={totalFindings} color="slate" />
        <StatPill label="Open" value={openCount} color="blue" />
        <StatPill label="Confirmed" value={confirmedCount} color="orange" />
        <StatPill label="Fixed" value={fixedCount} color="green" />
        <StatPill label="Critical/High" value={criticalHighCount} color="red" />
      </div>

      {/* Uptime */}
      {server?.uptime && (
        <div className="bg-paper-deep rounded-none border border-rule p-4">
          <span className="text-sm text-ink-muted">Uptime: </span>
          <span className="text-sm text-ink font-mono">{server.uptime}</span>
        </div>
      )}

      {/* Live Data Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Agent Status Card */}
        <div className="bg-paper-deep rounded-none border border-rule p-4">
          <div className="flex items-center gap-2 mb-3">
            <Icon name="agents" className="h-4 w-4 text-ink-muted" />
            <h3 className="text-sm font-medium text-ink-muted">Agent Status</h3>
          </div>
          {agentsLoading ? (
            <div className="text-xs text-ink-muted animate-pulse">Loading agents...</div>
          ) : !agents || agents.length === 0 ? (
            <div className="text-xs text-ink-muted">No agents reporting</div>
          ) : (
            <div className="space-y-2">
              {agents.map((agent) => (
                <AgentRow key={agent.agent_id} agent={agent} />
              ))}
            </div>
          )}
        </div>

        {/* Recent Findings Card */}
        <div className="bg-paper-deep rounded-none border border-rule p-4">
          <div className="flex items-center gap-2 mb-3">
            <Icon name="search" className="h-4 w-4 text-ink-muted" />
            <h3 className="text-sm font-medium text-ink-muted">Recent Findings</h3>
          </div>
          {findingsLoading ? (
            <div className="text-xs text-ink-muted animate-pulse">Loading findings...</div>
          ) : latestFindings.length === 0 ? (
            <div className="text-xs text-ink-muted">No findings yet</div>
          ) : (
            <div className="space-y-1">
              {latestFindings.map((finding) => (
                <FindingRow key={finding.id} finding={finding} />
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Dev note */}
      <div className="bg-paper-deep rounded border border-dashed border-rule p-4 text-sm text-ink-muted">
        If server stats show "...", the Go server may not be running on port
        4350. Start it with{' '}
        <code className="bg-paper px-1 rounded">go run ./cmd/server</code>
      </div>
    </div>
  );
}

function AgentRow({ agent }: { agent: AgentStatus }) {
  const dotColor = agent.status === 'active'
    ? 'bg-online'
    : agent.status === 'error'
      ? 'bg-accent'
      : 'bg-paper-deep';

  return (
    <div className="flex items-center gap-2 py-1.5 border-b border-rule last:border-0">
      <span className={`w-2 h-2 rounded-none ${dotColor} shrink-0`} />
      <span className="text-sm text-ink font-medium">{agent.name}</span>
      <span className="text-xs text-ink-muted capitalize">{agent.status}</span>
      <span className="text-xs text-ink-muted ml-auto font-mono truncate max-w-[120px]">{agent.model}</span>
    </div>
  );
}

const severityBadgeColors: Record<string, string> = {
  critical: 'bg-paper-deep text-accent',
  high: 'bg-paper-deep text-ink-muted',
  medium: 'bg-paper-deep text-ink-muted',
  low: 'bg-paper text-ink-muted',
};

const statusBadgeColors: Record<string, string> = {
  open: 'bg-paper-deep text-ink-muted',
  confirmed: 'bg-paper-deep text-ink-muted',
  rejected: 'bg-paper text-ink-muted',
  fixed: 'bg-paper-deep text-ink',
};

function FindingRow({ finding }: { finding: Finding }) {
  return (
    <div className="flex items-center gap-2 py-1.5 border-b border-rule last:border-0">
      <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium leading-tight ${severityBadgeColors[finding.severity] || severityBadgeColors.low}`}>
        {finding.severity.toUpperCase()}
      </span>
      <span className="text-sm text-ink truncate flex-1 min-w-0">{finding.title}</span>
      <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium leading-tight shrink-0 ${statusBadgeColors[finding.status] || statusBadgeColors.open}`}>
        {finding.status}
      </span>
    </div>
  );
}

function StatCard({
  label,
  value,
  error,
  color,
}: {
  label: string;
  value: string;
  error?: boolean;
  color?: 'green' | 'slate';
}) {
  return (
    <div className="bg-paper-deep rounded-none border border-rule p-4">
      <div className="text-xs text-ink-muted mb-1">{label}</div>
      <div
        className={`text-2xl font-bold ${
          error
            ? 'text-accent'
            : color === 'green'
              ? 'text-online'
              : 'text-ink'
        }`}
      >
        {error ? '—' : value}
      </div>
    </div>
  );
}

function StatPill({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: 'slate' | 'blue' | 'orange' | 'green' | 'red';
}) {
  const colorClasses: Record<string, string> = {
    slate: 'bg-paper text-ink-muted border-rule',
    blue: 'bg-paper-deep text-ink-muted border-rule',
    orange: 'bg-paper-deep text-ink-muted border-rule',
    green: 'bg-paper-deep text-ink border-rule',
    red: 'bg-paper-deep text-accent border-accent',
  };

  return (
    <div className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-none border text-xs font-medium ${colorClasses[color]}`}>
      <span>{label}:</span>
      <span className="font-bold">{value}</span>
    </div>
  );
}
