import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api, type NarrativeRow } from '../api/client';

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

function getValenceStyle(valence: number): string {
  if (valence <= -2) {
    return 'text-accent dark:text-accent font-extrabold border-l-2 border-accent pl-2'; // Oxblood red dread/warning
  }
  if (valence >= 2) {
    return 'text-emerald-700 dark:text-emerald-400 font-bold border-l-2 border-emerald-600 pl-2'; // Forest green triumph/loot
  }
  return 'text-ink opacity-85'; // Charcoal default ink
}

function getEventIcon(eventType: string): string {
  switch (eventType) {
    case 'mob_kill': return '⚔️';
    case 'mob_death': return '💀';
    case 'item_loot': return '💎';
    case 'player_encounter': return '👥';
    case 'room_visit': return '🗺️';
    case 'session_summary': return '📖';
    default: return '📜';
  }
}

export function MindReaderPage() {
  const [selectedAgent, setSelectedAgent] = useState('');
  const [page, setPage] = useState(0);
  const limit = 50;

  const { data, isLoading } = useQuery({
    queryKey: ['narrative', selectedAgent, page],
    queryFn: () => api.narrative({
      agent_name: selectedAgent || undefined,
      limit,
      offset: page * limit,
    }),
    refetchInterval: 10000, // Dynamic 10-second polling
  });

  const totalPages = data ? Math.ceil(data.total / limit) : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-extrabold text-accent tracking-widest font-sans">
          AI AGENT MIND READER
        </h1>
        <p className="text-xs text-ink-muted uppercase tracking-widest font-sans mt-1">
          Real-Time Narrative Telemetry Feed &amp; Cognitive Logs
        </p>
      </div>

      {/* Filter Ledger */}
      <div className="bg-paper-deep border-2 border-rule p-4 rounded-none relative">
        <div className="absolute top-1 left-1 w-1.5 h-1.5 border-t border-l border-rule/25" />
        <div className="absolute top-1 right-1 w-1.5 h-1.5 border-t border-r border-rule/25" />
        <div className="absolute bottom-1 left-1 w-1.5 h-1.5 border-b border-l border-rule/25" />
        <div className="absolute bottom-1 right-1 w-1.5 h-1.5 border-b border-r border-rule/25" />
        
        <div className="flex flex-col sm:flex-row gap-4 items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="text-xs font-sans uppercase font-bold tracking-wider text-ink-muted">
              FILTER OPERATOR:
            </span>
            <select
              value={selectedAgent}
              onChange={(e) => { setSelectedAgent(e.target.value); setPage(0); }}
              className="bg-paper text-ink text-xs font-sans uppercase tracking-wider rounded-none px-3 py-2 border-2 border-rule focus:outline-none focus:border-accent"
            >
              <option value="">All Autonomous Entities</option>
              <option value="Daeron">Daeron (Triage Master)</option>
              <option value="Reek">Reek (Code Crawler)</option>
            </select>
          </div>
          <div className="text-right text-[10px] font-mono text-ink-muted uppercase tracking-wider">
            SYSTEM STATUS: POLLING ACTIVE [10s]
          </div>
        </div>
      </div>

      {/* Scroll Sheet (Aged Typewriter Scroll) */}
      <div className="bg-paper border-2 border-rule p-6 md:p-8 rounded-none relative min-h-[400px] shadow-[4px_4px_0px_0px_rgba(26,22,20,0.1)]">
        {/* Horizontal Red Margin Rule lines (classic paper design) */}
        <div className="absolute left-8 md:left-12 top-0 bottom-0 border-l border-accent/20" />
        
        {isLoading ? (
          <div className="p-12 text-center text-ink-muted font-sans uppercase tracking-widest animate-pulse">
            Establishing mental interface with active agents...
          </div>
        ) : data && data.data.length > 0 ? (
          <div className="pl-6 md:pl-10 space-y-6">
            <div className="text-[10px] font-mono text-ink-muted border-b border-rule/20 pb-2 uppercase tracking-widest">
              TELEMETRY BATCH: {data.total.toLocaleString()} MEMORY FRAGMENTS LOGGED
            </div>
            
            <div className="space-y-6 font-mono text-sm leading-relaxed">
              {data.data.map((row: NarrativeRow) => (
                <div 
                  key={row.id} 
                  className={`group relative hover:bg-paper-deep/30 transition-colors p-2 -mx-2 flex flex-col sm:flex-row sm:items-start gap-2 border-b border-rule/5 border-dashed ${getValenceStyle(row.valence)}`}
                >
                  {/* Time badge */}
                  <span className="text-[10px] text-ink-muted uppercase tracking-wider shrink-0 w-24">
                    [{timeAgo(row.created_at)}]
                  </span>

                  {/* Icon & Subject */}
                  <span className="text-xs uppercase font-extrabold tracking-wider shrink-0 text-accent font-sans w-28 truncate">
                    {getEventIcon(row.event_type)} {row.agent_name}
                  </span>

                  {/* Summary Narrative */}
                  <div className="flex-1 font-serif text-sm">
                    {row.summary}
                    
                    {row.room_name && (
                      <span className="block text-[11px] font-sans uppercase tracking-wide text-ink-muted mt-0.5">
                        LOCATION: {row.room_name} [vnum {row.room_vnum}]
                      </span>
                    )}
                  </div>

                  {/* Context Info */}
                  <div className="shrink-0 text-right text-[10px] text-ink-muted font-sans uppercase tracking-wider self-end sm:self-auto">
                    {row.session_id ? `session: ${row.session_id.slice(0, 8)}...` : ''}
                  </div>
                </div>
              ))}
            </div>

            {/* Pagination Controls (Parchment styles) */}
            <div className="border-t border-rule/20 pt-6 flex justify-between items-center font-sans">
              <button
                onClick={() => setPage(Math.max(0, page - 1))}
                disabled={page === 0}
                className="px-4 py-2 text-xs uppercase tracking-widest font-bold border-2 border-rule bg-paper hover:bg-paper-deep disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              >
                ← PREVIOUS RECORD
              </button>
              <span className="text-xs text-ink-muted uppercase tracking-wider font-bold">
                PAGE {page + 1} OF {totalPages || 1}
              </span>
              <button
                onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
                disabled={page >= totalPages - 1}
                className="px-4 py-2 text-xs uppercase tracking-widest font-bold border-2 border-rule bg-paper hover:bg-paper-deep disabled:opacity-40 disabled:cursor-not-allowed transition-all"
              >
                NEXT RECORD →
              </button>
            </div>
          </div>
        ) : (
          <div className="pl-6 md:pl-10 p-12 text-center text-ink-muted font-sans uppercase tracking-widest">
            No memories cataloged inside the ledger. Memory logging activates when autonomous agents connect and explore.
          </div>
        )}
      </div>
    </div>
  );
}
