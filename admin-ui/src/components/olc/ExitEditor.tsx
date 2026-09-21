import type { OlcExit, OlcRoom, RoomPatchOperation } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface ExitEditorProps {
  room: OlcRoom;
  dirty: string[];
  disabled?: boolean;
  onOperation: (operation: RoomPatchOperation) => void;
}

const EXIT_DIRECTIONS = ['north', 'east', 'south', 'west', 'up', 'down'];

function dirtyFor(dirty: string[], direction: string, suffix?: string): boolean {
  return dirty.includes(`exit.${direction}${suffix ? `.${suffix}` : ''}`) || dirty.includes(`exit.${direction}`);
}

function doorIsSet(exit: OlcExit, bit: number): boolean {
  return bit === 0 ? exit.doorState > 0 : exit.doorState === 2;
}

function exitTitle(direction: string): string {
  return direction.charAt(0).toUpperCase() + direction.slice(1);
}

export function ExitEditor({ room, dirty, disabled = false, onOperation }: ExitEditorProps) {
  return (
    <section className="border-t border-rule pt-5">
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <h2 className="text-lg text-ink">Exits</h2>
        <span className="font-mono text-[11px] uppercase tracking-wider text-ink-muted">six directions</span>
      </div>
      <div className="space-y-2">
        {EXIT_DIRECTIONS.map((direction) => {
          const exit = room.exits[direction];
          return (
            <details key={`${direction}:${Boolean(exit)}`} open={Boolean(exit)} className="border border-rule bg-paper">
              <summary className="flex cursor-pointer list-none items-center justify-between gap-3 px-4 py-3 text-sm font-semibold text-ink [&::-webkit-details-marker]:hidden">
                <span>{exitTitle(direction)}</span>
                <span className="font-mono text-[10px] uppercase tracking-wider text-ink-muted">
                  {exit ? 'present' : 'empty'}
                </span>
              </summary>
              <div className="border-t border-rule px-4 py-4">
                {!exit ? (
                  <button
                    type="button"
                    disabled={disabled}
                    onClick={() => onOperation({ kind: 'ensure_exit', direction, value: -1 })}
                    className="border border-rule bg-paper-deep px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Add exit
                  </button>
                ) : (
                  <ExitFields exit={exit} direction={direction} dirty={dirty} disabled={disabled} onOperation={onOperation} />
                )}
              </div>
            </details>
          );
        })}
      </div>
    </section>
  );
}

function ExitFields({
  exit,
  direction,
  dirty,
  disabled,
  onOperation,
}: {
  exit: OlcExit;
  direction: string;
  dirty: string[];
  disabled: boolean;
  onOperation: (operation: RoomPatchOperation) => void;
}) {
  const commit = (kind: string, value: string) => {
    if (kind === 'set_exit_target' || kind === 'set_exit_key') {
      onOperation({ kind, direction, value: Number(value) });
    } else {
      onOperation({ kind, direction, text: value });
    }
  };

  return (
    <div className="space-y-4">
      <div className="grid gap-4 md:grid-cols-2">
        <div>
          <label htmlFor={`exit-${direction}-target`} className="mb-2 block text-sm font-semibold text-ink">
            Target room
          </label>
          <ServerProposal
            value={exit.toRoom}
            identity={`exit-${direction}-target`}
            disabled={disabled}
            min={-1}
            onCommit={(value) => commit('set_exit_target', value)}
          />
        </div>
        <div>
          <label htmlFor={`exit-${direction}-key`} className="mb-2 block text-sm font-semibold text-ink">
            Key vnum
          </label>
          <ServerProposal
            value={exit.key}
            identity={`exit-${direction}-key`}
            disabled={disabled}
            min={-1}
            onCommit={(value) => commit('set_exit_key', value)}
          />
        </div>
        <div>
          <label htmlFor={`exit-${direction}-keywords`} className="mb-2 block text-sm font-semibold text-ink">
            Keywords
          </label>
          <ServerProposal
            value={exit.keywords}
            identity={`exit-${direction}-keywords`}
            disabled={disabled}
            onCommit={(value) => commit('set_exit_keywords', value)}
          />
        </div>
        <div>
          <label htmlFor={`exit-${direction}-description`} className="mb-2 block text-sm font-semibold text-ink">
            Description
          </label>
          <ServerProposal
            value={exit.description}
            identity={`exit-${direction}-description`}
            disabled={disabled}
            multiline
            onCommit={(value) => commit('set_exit_description', value)}
          />
        </div>
      </div>

      <fieldset>
        <legend className="mb-2 text-sm font-semibold text-ink">Door flags</legend>
        <div className="flex flex-wrap gap-2">
          {[{ bit: 0, label: 'Door' }, { bit: 1, label: 'Pickproof' }].map((flag) => (
            <label key={flag.bit} className="flex min-h-10 items-center gap-2 border border-rule bg-paper px-3 py-2 text-sm text-ink hover:bg-paper-deep">
              <input
                type="checkbox"
                checked={doorIsSet(exit, flag.bit)}
                disabled={disabled}
                onChange={(event) => {
                  const nextDoor = flag.bit === 0 ? event.currentTarget.checked : doorIsSet(exit, 0);
                  const nextPickproof = flag.bit === 1 ? event.currentTarget.checked : doorIsSet(exit, 1);
                  onOperation({ kind: 'set_exit_door_flags', direction, value: nextDoor ? (nextPickproof ? 2 : 1) : 0 });
                }}
                className="h-4 w-4 accent-accent"
              />
              <span>{flag.label}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <div className="flex items-center justify-between gap-3 border-t border-rule pt-3">
        <span className="font-mono text-[11px] uppercase tracking-wider text-ink-muted">
          {[dirtyFor(dirty, direction, 'target') && 'target', dirtyFor(dirty, direction, 'description') && 'description', dirtyFor(dirty, direction, 'keywords') && 'keywords', dirtyFor(dirty, direction, 'key') && 'key', dirtyFor(dirty, direction, 'door_flags') && 'door flags'].filter(Boolean).join(' · ') || 'unchanged'}
        </span>
        <button
          type="button"
          disabled={disabled}
          onClick={() => {
            if (window.confirm(`Purge the ${direction} exit?`)) onOperation({ kind: 'purge_exit', direction });
          }}
          className="border border-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-accent hover:bg-accent hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
        >
          Purge exit
        </button>
      </div>
    </div>
  );
}
