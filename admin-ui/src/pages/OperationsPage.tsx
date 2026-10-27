import { ConfirmDialog } from '../components/ConfirmDialog';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '../components/ui/select';
import { Button } from '../components/ui/button';
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
  const [resetOpen, setResetOpen] = useState(false);
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

  const handleResetZones = () => setResetOpen(true);

  return (
    <div className="space-y-6">
      <ConfirmDialog open={resetOpen} onOpenChange={setResetOpen} title="Reset all zones?" description="This will respawn all mobs and objects." action="Reset all zones" onConfirm={() => resetZonesMutation.mutate()} />
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
                <Button variant="ghost" size="plain"
                  key={p.name}
                  onClick={() => setSelectedPlayer(p.name)}
                  className="w-full flex justify-between text-sm py-1 border-b border-rule last:border-0 hover:bg-paper transition-colors rounded-[2px] px-1 text-left"
                >
                  <span className="text-accent hover:text-accent">{p.name}</span>
                  <span className="text-ink-muted">
                    Lv.{p.level} · Room {p.room}
                  </span>
                </Button>
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
            <Select value={String(logLines)} onValueChange={(value) => setLogLines(Number(value))}><SelectTrigger aria-label="Log lines"><SelectValue /></SelectTrigger><SelectContent><SelectItem value="25">25</SelectItem><SelectItem value="50">50</SelectItem><SelectItem value="100">100</SelectItem><SelectItem value="200">200</SelectItem></SelectContent></Select>
            <Button variant="ghost" size="plain"
              onClick={() => refetchLogs()}
              className="bg-paper text-ink-muted text-xs rounded-[2px] px-3 py-1 border border-rule hover:bg-paper transition-colors"
            >
              Refresh
            </Button>
          </div>
        </div>
        <div className="bg-paper rounded-[2px] border border-rule p-3 max-h-96 overflow-y-auto">
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
          <Button variant="ghost" size="plain"
            onClick={handleResetZones}
            disabled={resetZonesMutation.isPending}
            className="bg-accent hover:bg-accent-deep disabled:bg-paper disabled:text-ink-muted text-paper text-sm rounded-[2px] px-4 py-2 border border-accent disabled:border-rule transition-colors"
          >
            {resetZonesMutation.isPending ? 'Resetting...' : 'Zone Reset All'}
          </Button>
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
