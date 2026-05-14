import { useQuery } from '@tanstack/react-query';
import { api } from '../api/client';

function formatBytes(bytes: number): string {
  return (bytes / 1024 / 1024).toFixed(1) + ' MB';
}

function formatDuration(ns: number): string {
  if (ns < 1_000) return `${ns}ns`;
  if (ns < 1_000_000) return `${(ns / 1_000).toFixed(1)}µs`;
  if (ns < 1_000_000_000) return `${(ns / 1_000_000).toFixed(1)}ms`;
  return `${(ns / 1_000_000_000).toFixed(2)}s`;
}

export function MetricsCard() {
  const { data: metrics, isLoading, error } = useQuery({
    queryKey: ['metrics'],
    queryFn: api.metrics,
    refetchInterval: 15000,
  });

  if (isLoading) {
    return (
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
        <h2 className="text-sm font-medium text-slate-300 mb-3">Server Metrics</h2>
        <div className="text-sm text-slate-500 animate-pulse">Loading metrics...</div>
      </div>
    );
  }

  if (error || !metrics) {
    return (
      <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
        <h2 className="text-sm font-medium text-slate-300 mb-3">Server Metrics</h2>
        <div className="text-sm text-red-400">Failed to load metrics</div>
      </div>
    );
  }

  const heapPct = metrics.memory_sys > 0
    ? Math.round((metrics.memory_heap / metrics.memory_sys) * 100)
    : 0;

  return (
    <div className="bg-slate-800 rounded-lg border border-slate-700 p-4">
      <h2 className="text-sm font-medium text-slate-300 mb-3">Server Metrics</h2>

      {/* Memory */}
      <div className="mb-3">
        <div className="flex justify-between text-xs text-slate-400 mb-1">
          <span>Heap / Sys</span>
          <span>{formatBytes(metrics.memory_heap)} / {formatBytes(metrics.memory_sys)}</span>
        </div>
        <div className="w-full bg-slate-700 rounded-full h-2">
          <div
            className="h-2 rounded-full bg-cyan-500"
            style={{ width: `${Math.min(heapPct, 100)}%` }}
          />
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-sm">
        <MetricItem label="Alloc" value={formatBytes(metrics.memory_alloc)} />
        <MetricItem label="Sys" value={formatBytes(metrics.memory_sys)} />
        <MetricItem label="Heap" value={formatBytes(metrics.memory_heap)} />

        <MetricItem
          label="Goroutines"
          value={String(metrics.goroutines)}
          icon={
            <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          }
        />

        <MetricItem label="GC Cycles" value={String(metrics.gc_cycles)} />
        <MetricItem label="GC Pause" value={formatDuration(metrics.pause_total_ns)} />
        <MetricItem label="Last GC" value={metrics.last_gc} />
        <MetricItem label="Uptime" value={metrics.uptime} />
      </div>
    </div>
  );
}

function MetricItem({ label, value, icon }: { label: string; value: string; icon?: React.ReactNode }) {
  return (
    <div>
      <div className="flex items-center gap-1 text-xs text-slate-400 mb-0.5">
        {icon}
        <span>{label}</span>
      </div>
      <div className="text-sm text-white font-mono">{value}</div>
    </div>
  );
}
