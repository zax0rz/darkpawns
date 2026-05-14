import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api, type PlayerDetail } from '../api/client';
import { useToast } from './Toast';

const classNames: Record<number, string> = {
  0: 'Magic User', 1: 'Cleric', 2: 'Thief', 3: 'Warrior',
  4: 'Anti-Paladin', 5: 'Paladin', 6: 'Ranger', 7: 'Bard',
  8: 'Monk', 9: 'Barbarian', 10: 'Sorcerer', 11: 'Assassin', 12: 'Necromancer',
};

const raceNames: Record<number, string> = {
  0: 'Human', 1: 'Elf', 2: 'Dwarf', 3: 'Hobbit', 4: 'Pixie',
  5: 'Orc', 6: 'Gnome', 7: 'Troll', 8: 'Half-Orc', 9: 'Giant',
  10: 'Gnoll', 11: 'Bugbear', 12: 'Kender', 13: 'Vampire', 14: 'Werewolf', 15: 'Faerie',
};

const sexLabels: Record<number, string> = {
  0: 'Male', 1: 'Female', 2: 'Neutral',
};

type Tab = 'stats' | 'inventory' | 'equipment';

interface PlayerDetailModalProps {
  playerName: string;
  onClose: () => void;
}

function ProgressBar({ current, max, label, color }: { current: number; max: number; label: string; color: string }) {
  const pct = max > 0 ? Math.round((current / max) * 100) : 0;
  return (
    <div>
      <div className="flex justify-between text-xs text-slate-400 mb-1">
        <span>{label}</span>
        <span>{current}/{max}</span>
      </div>
      <div className="w-full bg-slate-700 rounded-full h-2">
        <div
          className={`h-2 rounded-full ${color}`}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>
    </div>
  );
}

export function PlayerDetailModal({ playerName, onClose }: PlayerDetailModalProps) {
  const [activeTab, setActiveTab] = useState<Tab>('stats');
  const queryClient = useQueryClient();
  const { showToast } = useToast();

  const { data: player, isLoading, error } = useQuery({
    queryKey: ['player-detail', playerName],
    queryFn: () => api.playerDetail(playerName),
  });

  const saveMutation = useMutation({
    mutationFn: () => api.savePlayer(playerName),
    onSuccess: () => {
      showToast(`Player ${playerName} saved`, 'success');
    },
    onError: (err: Error) => {
      showToast(`Save failed: ${err.message}`, 'error');
    },
  });

  const kickMutation = useMutation({
    mutationFn: () => api.kickPlayer(playerName),
    onSuccess: () => {
      showToast(`Player ${playerName} kicked`, 'success');
      queryClient.invalidateQueries({ queryKey: ['players'] });
      onClose();
    },
    onError: (err: Error) => {
      showToast(`Kick failed: ${err.message}`, 'error');
    },
  });

  if (isLoading) {
    return (
      <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
        <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 w-full max-w-2xl max-h-[80vh]">
          <div className="text-sm text-slate-500 animate-pulse">Loading player data...</div>
        </div>
      </div>
    );
  }

  if (error || !player) {
    return (
      <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
        <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 w-full max-w-2xl max-h-[80vh]">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-sm font-medium text-slate-300">Player: {playerName}</h2>
            <button onClick={onClose} className="text-slate-400 hover:text-white text-lg">&times;</button>
          </div>
          <div className="text-sm text-red-400">Failed to load player details.</div>
          <div className="mt-1 text-xs text-red-400/70">{(error as Error)?.message || 'Not found'}</div>
        </div>
      </div>
    );
  }

  const handleKick = () => {
    if (window.confirm(`Are you sure you want to kick ${playerName}?`)) {
      kickMutation.mutate();
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50" onClick={onClose}>
      <div
        className="bg-slate-800 rounded-lg border border-slate-700 w-full max-w-2xl max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div>
            <h2 className="text-lg font-bold text-white">{playerName}</h2>
            <p className="text-xs text-slate-400 mt-0.5">
              Lv.{player.level} {classNames[player.class] || `Class ${player.class}`} · {raceNames[player.race] || `Race ${player.race}`} · {sexLabels[player.sex] || 'Unknown'}
            </p>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white text-xl leading-none">&times;</button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-slate-700">
          {(['stats', 'inventory', 'equipment'] as Tab[]).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-sm font-medium capitalize transition-colors ${
                activeTab === tab
                  ? 'text-amber-400 border-b-2 border-amber-400'
                  : 'text-slate-400 hover:text-slate-200'
              }`}
            >
              {tab}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="overflow-y-auto flex-1 p-4 space-y-4">
          {activeTab === 'stats' && (
            <div className="space-y-4">
              {/* HP/Mana/Move Bars */}
              <div className="space-y-2">
                <ProgressBar current={player.health} max={player.max_health} label="HP" color="bg-red-500" />
                <ProgressBar current={player.mana} max={player.max_mana} label="Mana" color="bg-blue-500" />
                <ProgressBar current={player.move} max={player.max_move} label="Move" color="bg-green-500" />
              </div>

              {/* Alignment Bar */}
              <div>
                <div className="flex justify-between text-xs text-slate-400 mb-1">
                  <span>Alignment</span>
                  <span>{player.alignment}</span>
                </div>
                <div className="w-full bg-slate-700 rounded-full h-2">
                  <div
                    className={`h-2 rounded-full ${
                      player.alignment > 100 ? 'bg-blue-500' : player.alignment < -100 ? 'bg-red-500' : 'bg-slate-400'
                    }`}
                    style={{ width: `${Math.min(Math.max(((player.alignment + 1000) / 2000) * 100, 0), 100)}%` }}
                  />
                </div>
              </div>

              {/* Stats Grid */}
              <div>
                <h3 className="text-xs font-medium text-slate-400 mb-2 uppercase tracking-wide">Attributes</h3>
                <div className="grid grid-cols-3 gap-3">
                  <StatItem label="STR" value={player.stats?.str ?? 0} />
                  <StatItem label="INT" value={player.stats?.int ?? 0} />
                  <StatItem label="WIS" value={player.stats?.wis ?? 0} />
                  <StatItem label="DEX" value={player.stats?.dex ?? 0} />
                  <StatItem label="CON" value={player.stats?.con ?? 0} />
                  <StatItem label="CHA" value={player.stats?.cha ?? 0} />
                </div>
              </div>

              {/* Combat */}
              <div>
                <h3 className="text-xs font-medium text-slate-400 mb-2 uppercase tracking-wide">Combat</h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  <StatItem label="AC" value={player.ac} />
                  <StatItem label="THAC0" value={player.thac0} />
                  <StatItem label="Hitroll" value={player.hitroll} />
                  <StatItem label="Damroll" value={player.damroll} />
                </div>
              </div>

              {/* Economy */}
              <div>
                <h3 className="text-xs font-medium text-slate-400 mb-2 uppercase tracking-wide">Economy</h3>
                <div className="grid grid-cols-3 gap-3">
                  <StatItem label="Gold" value={player.gold.toLocaleString()} />
                  <StatItem label="Bank" value={player.bank_gold.toLocaleString()} />
                  <StatItem label="EXP" value={player.exp.toLocaleString()} />
                </div>
              </div>

              {/* Session Info */}
              <div>
                <h3 className="text-xs font-medium text-slate-400 mb-2 uppercase tracking-wide">Session</h3>
                <div className="grid grid-cols-2 gap-3">
                  <StatItem label="Connected" value={player.connected_at} />
                  <StatItem label="Last Active" value={player.last_active} />
                  <StatItem label="Room" value={`#${player.room}`} />
                  <StatItem label="Affects" value={player.affects} />
                </div>
              </div>

              {/* Actions */}
              <div className="flex gap-2 pt-2">
                <button
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending}
                  className="bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 disabled:text-blue-400 text-white text-sm px-4 py-2 rounded transition-colors"
                >
                  {saveMutation.isPending ? 'Saving...' : 'Save'}
                </button>
                <button
                  onClick={handleKick}
                  disabled={kickMutation.isPending}
                  className="bg-red-700 hover:bg-red-600 disabled:bg-red-900 disabled:text-red-400 text-white text-sm px-4 py-2 rounded transition-colors"
                >
                  {kickMutation.isPending ? 'Kicking...' : 'Kick'}
                </button>
              </div>
            </div>
          )}

          {activeTab === 'inventory' && (
            <div>
              {player.inventory && player.inventory.length > 0 ? (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-xs text-slate-400 border-b border-slate-700">
                      <th className="text-left py-1 pr-2">#</th>
                      <th className="text-left py-1 pr-2">Name</th>
                      <th className="text-left py-1">Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {player.inventory.map((item, i) => (
                      <tr key={i} className="border-b border-slate-700/50 text-slate-200">
                        <td className="py-1.5 pr-2 font-mono text-amber-400">{item.vnum}</td>
                        <td className="py-1.5 pr-2">{item.name}</td>
                        <td className="py-1.5 text-slate-400">{item.short_desc}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="text-sm text-slate-500">No items in inventory.</p>
              )}
            </div>
          )}

          {activeTab === 'equipment' && (
            <div>
              {player.equipment && player.equipment.length > 0 ? (
                <table className="w-full text-sm">
                  <thead>
                    <tr className="text-xs text-slate-400 border-b border-slate-700">
                      <th className="text-left py-1 pr-3">Slot</th>
                      <th className="text-left py-1 pr-2">#</th>
                      <th className="text-left py-1 pr-2">Name</th>
                      <th className="text-left py-1">Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {player.equipment.map((item, i) => (
                      <tr key={i} className="border-b border-slate-700/50 text-slate-200">
                        <td className="py-1.5 pr-3 text-xs text-slate-400 font-mono">{item.wear_location}</td>
                        <td className="py-1.5 pr-2 font-mono text-amber-400">{item.vnum}</td>
                        <td className="py-1.5 pr-2">{item.name}</td>
                        <td className="py-1.5 text-slate-400">{item.short_desc}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              ) : (
                <p className="text-sm text-slate-500">No equipment worn.</p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function StatItem({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-slate-700/50 rounded p-2">
      <div className="text-xs text-slate-400">{label}</div>
      <div className="text-sm text-white font-mono">{value}</div>
    </div>
  );
}
