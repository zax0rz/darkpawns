import { useState } from 'react';
import type { OlcExit, OlcRoom, RoomPatchOperation } from '../../api/olc';
import type { OlcSchema } from '../../api/olc';
import { ServerProposal } from './ServerProposal';
import { VNumPicker } from './VNumPicker';

interface ExitEditorProps {
  schema: OlcSchema;
  room: OlcRoom;
  dirty: string[];
  disabled?: boolean;
  onOperation: (operation: RoomPatchOperation) => void;
}

function dirtyFor(dirty: string[], direction: string, suffix?: string): boolean {
  return dirty.includes(`exit.${direction}${suffix ? `.${suffix}` : ''}`) || dirty.includes(`exit.${direction}`);
}

function exitTitle(direction: string): string {
  return direction.charAt(0).toUpperCase() + direction.slice(1);
}

export function ExitEditor({ schema, room, dirty, disabled = false, onOperation }: ExitEditorProps) {
  const [openDirections, setOpenDirections] = useState<Record<string, boolean>>({});
  const directions = schema.exits?.directions || [];
  const doorOptions = schema.exits?.doorOptions || [];
  return (
    <section className="border-t border-rule pt-5">
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <h2 className="text-lg text-ink">Exits</h2>
        <span className="font-mono text-[11px] uppercase tracking-wider text-ink-muted">{directions.length} directions</span>
      </div>
      <div className="space-y-2">
        {directions.map((direction) => {
          const exit = room.exits[direction];
          const open = Object.prototype.hasOwnProperty.call(openDirections, direction)
            ? openDirections[direction]
            : Boolean(exit);
          return (
            <details
              key={direction}
              open={open}
              onToggle={(event) =>
                setOpenDirections((current) => ({
                  ...current,
                  [direction]: event.currentTarget.open,
                }))
              }
              className="border border-rule bg-paper"
            >
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
                  <ExitFields
                    exit={exit}
                    direction={direction}
                    doorOptions={doorOptions}
                    dirty={dirty}
                    disabled={disabled}
                    onOperation={onOperation}
                  />
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
  doorOptions,
  dirty,
  disabled,
  onOperation,
}: {
  exit: OlcExit;
  direction: string;
  doorOptions: { value: number; label: string }[];
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
        <VNumPicker
          kind="room"
          label="Target room"
          identity={`exit-${direction}-target`}
          value={exit.toRoom}
          disabled={disabled}
          min={-1}
          onCommit={(value) => commit('set_exit_target', value)}
        />
        <VNumPicker
          kind="obj"
          label="Key vnum"
          identity={`exit-${direction}-key`}
          value={exit.key}
          disabled={disabled}
          min={-1}
          onCommit={(value) => commit('set_exit_key', value)}
        />
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

      <div>
        <label htmlFor={`exit-${direction}-door`} className="mb-2 block text-sm font-semibold text-ink">Door flags</label>
        <select
          id={`exit-${direction}-door`}
          value={exit.doorState}
          disabled={disabled}
          onChange={(event) => onOperation({ kind: 'set_exit_door_flags', direction, value: Number(event.currentTarget.value) })}
          className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
        >
          {doorOptions.map((option) => (
            <option key={option.value} value={option.value}>{option.label}</option>
          ))}
        </select>
      </div>

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
