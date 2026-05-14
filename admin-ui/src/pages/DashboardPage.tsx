import { useQuery } from '@tanstack/react-query';
import { api, type AgentStatus, type Finding } from '../api/client';
import { StatCardSkeleton } from '../components/Skeleton';

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
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white">Dashboard</h1>

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
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4">
          <span className="text-sm text-slate-500 dark:text-slate-400">Uptime: </span>
          <span className="text-sm text-slate-900 dark:text-white font-mono">{server.uptime}</span>
        </div>
      )}

      {/* Live Data Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Agent Status Card */}
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4">
          <div className="flex items-center gap-2 mb-3">
            <span className="text-xl">🤖</span>
            <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300">Agent Status</h3>
          </div>
          {agentsLoading ? (
            <div className="text-xs text-slate-400 dark:text-slate-500 animate-pulse">Loading agents...</div>
          ) : !agents || agents.length === 0 ? (
            <div className="text-xs text-slate-400 dark:text-slate-500">No agents reporting</div>
          ) : (
            <div className="space-y-2">
              {agents.map((agent) => (
                <AgentRow key={agent.agent_id} agent={agent} />
              ))}
            </div>
          )}
        </div>

        {/* Recent Findings Card */}
        <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4">
          <div className="flex items-center gap-2 mb-3">
            <span className="text-xl">🔍</span>
            <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300">Recent Findings</h3>
          </div>
          {findingsLoading ? (
            <div className="text-xs text-slate-400 dark:text-slate-500 animate-pulse">Loading findings...</div>
          ) : latestFindings.length === 0 ? (
            <div className="text-xs text-slate-400 dark:text-slate-500">No findings yet</div>
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
      <div className="bg-white/50 dark:bg-slate-800/50 rounded border border-dashed border-slate-300 dark:border-slate-600 p-4 text-sm text-slate-500 dark:text-slate-400">
        💡 If server stats show "...", the Go server may not be running on port
        4350. Start it with{' '}
        <code className="bg-slate-100 dark:bg-slate-700 px-1 rounded">go run ./cmd/server</code>
      </div>
    </div>
  );
}

function AgentRow({ agent }: { agent: AgentStatus }) {
  const dotColor = agent.status === 'active'
    ? 'bg-green-500'
    : agent.status === 'error'
      ? 'bg-red-500'
      : 'bg-yellow-500';

  return (
    <div className="flex items-center gap-2 py-1.5 border-b border-slate-100 dark:border-slate-700/50 last:border-0">
      <span className={`w-2 h-2 rounded-full ${dotColor} shrink-0`} />
      <span className="text-sm text-slate-900 dark:text-white font-medium">{agent.name}</span>
      <span className="text-xs text-slate-500 dark:text-slate-400 capitalize">{agent.status}</span>
      <span className="text-xs text-slate-400 dark:text-slate-500 ml-auto font-mono truncate max-w-[120px]">{agent.model}</span>
    </div>
  );
}

const severityBadgeColors: Record<string, string> = {
  critical: 'bg-red-100 dark:bg-red-900 text-red-700 dark:text-red-300',
  high: 'bg-orange-100 dark:bg-orange-900 text-orange-700 dark:text-orange-300',
  medium: 'bg-yellow-100 dark:bg-yellow-900 text-yellow-700 dark:text-yellow-300',
  low: 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-300',
};

const statusBadgeColors: Record<string, string> = {
  open: 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300',
  confirmed: 'bg-orange-100 dark:bg-orange-900 text-orange-700 dark:text-orange-300',
  rejected: 'bg-slate-100 dark:bg-slate-700 text-slate-500 dark:text-slate-400',
  fixed: 'bg-green-100 dark:bg-green-900 text-green-700 dark:text-green-300',
};

function FindingRow({ finding }: { finding: Finding }) {
  return (
    <div className="flex items-center gap-2 py-1.5 border-b border-slate-100 dark:border-slate-700/50 last:border-0">
      <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] font-medium leading-tight ${severityBadgeColors[finding.severity] || severityBadgeColors.low}`}>
        {finding.severity.toUpperCase()}
      </span>
      <span className="text-sm text-slate-900 dark:text-white truncate flex-1 min-w-0">{finding.title}</span>
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
    <div className="bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 p-4">
      <div className="text-xs text-slate-500 dark:text-slate-400 mb-1">{label}</div>
      <div
        className={`text-2xl font-bold ${
          error
            ? 'text-red-500 dark:text-red-400'
            : color === 'green'
              ? 'text-green-600 dark:text-green-400'
              : 'text-slate-900 dark:text-white'
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
    slate: 'bg-slate-100 dark:bg-slate-800 text-slate-700 dark:text-slate-300 border-slate-200 dark:border-slate-700',
    blue: 'bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800',
    orange: 'bg-orange-100 dark:bg-orange-900/40 text-orange-700 dark:text-orange-300 border-orange-200 dark:border-orange-800',
    green: 'bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300 border-green-200 dark:border-green-800',
    red: 'bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300 border-red-200 dark:border-red-800',
  };

  return (
    <div className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full border text-xs font-medium ${colorClasses[color]}`}>
      <span>{label}:</span>
      <span className="font-bold">{value}</span>
    </div>
  );
}
