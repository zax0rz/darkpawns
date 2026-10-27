import { ConfirmDialog } from './ConfirmDialog';
import { Tabs, TabsList, TabsTrigger, TabsContent } from './ui/tabs';
import { Dialog, DialogContent, DialogTitle, DialogDescription } from './ui/dialog';
import type { ReactNode } from 'react';
import { Button } from './ui/button';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from './ui/table';
import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { useToast } from '../hooks/useToast';
import { sexLabel } from '../lib/gameLabels';

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


type Tab = 'stats' | 'inventory' | 'equipment';

interface PlayerDetailModalProps {
  playerName: string;
  onClose: () => void;
}

function ProgressBar({ current, max, label, color }: { current: number; max: number; label: string; color: string }) {
  const pct = max > 0 ? Math.round((current / max) * 100) : 0;
  return (
    <div>
      <div className="flex justify-between text-xs text-ink-muted mb-1">
        <span>{label}</span>
        <span>{current}/{max}</span>
      </div>
      <div className="w-full bg-paper rounded-none h-2">
        <div
          className={`h-2 rounded-none ${color}`}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>
    </div>
  );
}

export function PlayerDetailModal({ playerName, onClose }: PlayerDetailModalProps) {
  const [kickOpen, setKickOpen] = useState(false);
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
      <PlayerDialogFrame title={playerName} onClose={onClose}>
        <div className="bg-paper-deep rounded-none border border-rule p-6 w-full max-w-2xl max-h-[80vh]">
          <div className="text-sm text-ink-muted animate-pulse">Loading player data...</div>
        </div>
      </PlayerDialogFrame>
    );
  }

  if (error || !player) {
    return (
      <PlayerDialogFrame title={playerName} onClose={onClose}>
        <div className="bg-paper-deep rounded-none border border-rule p-6 w-full max-w-2xl max-h-[80vh]">
          <div className="flex justify-between items-center mb-4">
            <h2 className="text-sm font-medium text-ink-muted">Player: {playerName}</h2>
            <Button variant="ghost" size="plain" onClick={onClose} aria-label="Close player details" className="text-ink-muted hover:text-ink text-lg">&times;</Button>
          </div>
          <div className="text-sm text-accent">Failed to load player details.</div>
          <div className="mt-1 text-xs text-accent">{(error as Error)?.message || 'Not found'}</div>
        </div>
      </PlayerDialogFrame>
    );
  }

  const handleKick = () => setKickOpen(true);

  return (
    <PlayerDialogFrame title={playerName} onClose={onClose}>
      <ConfirmDialog open={kickOpen} onOpenChange={setKickOpen} title={`Kick ${playerName}?`} description="Disconnect this character from the game." action="Kick player" onConfirm={() => kickMutation.mutate()} />
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as Tab)}
        className="bg-paper-deep w-full max-w-2xl max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-rule">
          <div>
            <h2 className="text-lg font-bold text-ink">{playerName}</h2>
            <p className="text-xs text-ink-muted mt-0.5">
              Lv.{player.level} {classNames[player.class] || `Class ${player.class}`} · {raceNames[player.race] || `Race ${player.race}`} · {sexLabel(player.sex)}
            </p>
          </div>
          <Button variant="ghost" size="plain" onClick={onClose} aria-label="Close player details" className="text-ink-muted hover:text-ink text-xl leading-none">&times;</Button>
        </div>

        {/* Tabs */}
        <TabsList className="w-full justify-start" aria-label="Player details">
          {(['stats', 'inventory', 'equipment'] as Tab[]).map((tab) => (
            <TabsTrigger
              value={tab} key={tab}
              className={`px-4 py-2 text-sm font-medium capitalize transition-colors ${
                activeTab === tab
                  ? 'text-accent border-b-2 border-accent'
                  : 'text-ink-muted hover:text-ink'
              }`}
            >
              {tab}
            </TabsTrigger>
          ))}
        </TabsList>

        {/* Content */}
        <div className="overflow-y-auto flex-1 p-4 space-y-4">
          <TabsContent value="stats">
            <div className="space-y-4">
              {/* HP/Mana/Move Bars */}
              <div className="space-y-2">
                <ProgressBar current={player.health} max={player.max_health} label="HP" color="bg-accent" />
                <ProgressBar current={player.mana} max={player.max_mana} label="Mana" color="bg-paper-deep" />
                <ProgressBar current={player.move} max={player.max_move} label="Move" color="bg-online" />
              </div>

              {/* Alignment Bar */}
              <div>
                <div className="flex justify-between text-xs text-ink-muted mb-1">
                  <span>Alignment</span>
                  <span>{player.alignment}</span>
                </div>
                <div className="w-full bg-paper rounded-none h-2">
                  <div
                    className={`h-2 rounded-none ${
                      player.alignment > 100 ? 'bg-paper-deep' : player.alignment < -100 ? 'bg-accent' : 'bg-paper-deep'
                    }`}
                    style={{ width: `${Math.min(Math.max(((player.alignment + 1000) / 2000) * 100, 0), 100)}%` }}
                  />
                </div>
              </div>

              {/* Stats Grid */}
              <div>
                <h3 className="text-xs font-medium text-ink-muted mb-2 uppercase tracking-wide">Attributes</h3>
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
                <h3 className="text-xs font-medium text-ink-muted mb-2 uppercase tracking-wide">Combat</h3>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  <StatItem label="AC" value={player.ac} />
                  <StatItem label="THAC0" value={player.thac0} />
                  <StatItem label="Hitroll" value={player.hitroll} />
                  <StatItem label="Damroll" value={player.damroll} />
                </div>
              </div>

              {/* Economy */}
              <div>
                <h3 className="text-xs font-medium text-ink-muted mb-2 uppercase tracking-wide">Economy</h3>
                <div className="grid grid-cols-3 gap-3">
                  <StatItem label="Gold" value={player.gold.toLocaleString()} />
                  <StatItem label="Bank" value={player.bank_gold.toLocaleString()} />
                  <StatItem label="EXP" value={player.exp.toLocaleString()} />
                </div>
              </div>

              {/* Session Info */}
              <div>
                <h3 className="text-xs font-medium text-ink-muted mb-2 uppercase tracking-wide">Session</h3>
                <div className="grid grid-cols-2 gap-3">
                  <StatItem label="Connected" value={player.connected_at} />
                  <StatItem label="Last Active" value={player.last_active} />
                  <StatItem label="Room" value={`#${player.room}`} />
                  <StatItem label="Affects" value={player.affects} />
                </div>
              </div>

              {/* Actions */}
              <div className="flex gap-2 pt-2">
                <Button variant="ghost" size="plain"
                  onClick={() => saveMutation.mutate()}
                  disabled={saveMutation.isPending}
                  className="bg-paper-deep hover:bg-paper-deep disabled:bg-paper-deep disabled:text-ink-muted text-paper text-sm px-4 py-2 rounded-[2px] transition-colors"
                >
                  {saveMutation.isPending ? 'Saving...' : 'Save'}
                </Button>
                <Button variant="ghost" size="plain"
                  onClick={handleKick}
                  disabled={kickMutation.isPending}
                  className="bg-accent hover:bg-accent disabled:bg-paper-deep disabled:text-accent text-paper text-sm px-4 py-2 rounded-[2px] transition-colors"
                >
                  {kickMutation.isPending ? 'Kicking...' : 'Kick'}
                </Button>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="inventory">
            <div>
              {player.inventory && player.inventory.length > 0 ? (
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="text-xs text-ink-muted border-b border-rule">
                      <TableHead className="text-left py-1 pr-2">#</TableHead>
                      <TableHead className="text-left py-1 pr-2">Name</TableHead>
                      <TableHead className="text-left py-1">Description</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {player.inventory.map((item, i) => (
                      <TableRow key={i} className="border-b border-rule text-ink">
                        <TableCell className="py-1.5 pr-2 font-mono text-accent">{item.vnum}</TableCell>
                        <TableCell className="py-1.5 pr-2">{item.name}</TableCell>
                        <TableCell className="py-1.5 text-ink-muted">{item.name}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-ink-muted">No items in inventory.</p>
              )}
            </div>
          </TabsContent>

          <TabsContent value="equipment">
            <div>
              {player.equipment && player.equipment.length > 0 ? (
                <Table className="w-full text-sm">
                  <TableHeader>
                    <TableRow className="text-xs text-ink-muted border-b border-rule">
                      <TableHead className="text-left py-1 pr-3">Slot</TableHead>
                      <TableHead className="text-left py-1 pr-2">#</TableHead>
                      <TableHead className="text-left py-1 pr-2">Name</TableHead>
                      <TableHead className="text-left py-1">Description</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {player.equipment.map((item, i) => (
                      <TableRow key={i} className="border-b border-rule text-ink">
                        <TableCell className="py-1.5 pr-3 text-xs text-ink-muted font-mono">{item.slot || '—'}</TableCell>
                        <TableCell className="py-1.5 pr-2 font-mono text-accent">{item.vnum}</TableCell>
                        <TableCell className="py-1.5 pr-2">{item.name}</TableCell>
                        <TableCell className="py-1.5 text-ink-muted">{item.name}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              ) : (
                <p className="text-sm text-ink-muted">No equipment worn.</p>
              )}
            </div>
          </TabsContent>
        </div>
      </Tabs>
    </PlayerDialogFrame>
  );
}

function StatItem({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-paper rounded-[2px] p-2">
      <div className="text-xs text-ink-muted">{label}</div>
      <div className="text-sm text-ink font-mono">{value}</div>
    </div>
  );
}

function PlayerDialogFrame({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return <Dialog open onOpenChange={(open) => { if (!open) onClose(); }}>
    <DialogContent showCloseButton={false} className="block max-h-[85dvh] overflow-y-auto p-0 sm:max-w-2xl">
      <DialogTitle className="sr-only">Player: {title}</DialogTitle>
      <DialogDescription className="sr-only">Inspect online character statistics, inventory and equipment.</DialogDescription>
      {children}
    </DialogContent>
  </Dialog>;
}
