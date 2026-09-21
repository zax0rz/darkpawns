import { useMemo, useState } from 'react';
import type { OlcPatchOperation, OlcSchema, OlcZoneCommand, OlcZoneCommandArgument } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface ZoneCommandTimelineProps {
  schema: OlcSchema;
  commands: OlcZoneCommand[];
  disabled?: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}

interface ReorderRequest {
  from: number;
  to: number;
}

function descriptorFor(schema: OlcSchema, command: string) {
  return schema.zoneCommands.find((entry) => entry.command === command) || schema.zoneCommands[0];
}

function argumentValue(command: OlcZoneCommand, index: number): number {
  return [command.arg1, command.arg2, command.arg3][index] || 0;
}

function commandWith(command: OlcZoneCommand, index: number, value: number): OlcPatchOperation {
  return {
    kind: 'modify_command',
    index: command.position,
    command: command.command,
    if_flag: command.ifFlag,
    arg1: index === 0 ? value : command.arg1,
    arg2: index === 1 ? value : command.arg2,
    arg3: index === 2 ? value : command.arg3,
  };
}

function edgeLabels(commands: OlcZoneCommand[]): string[] {
  return commands.flatMap((command, index) => {
    if (index === 0 || command.ifFlag === 0) return [];
    const previous = commands[index - 1];
    return [`#${previous.position} ${previous.command} → #${command.position} ${command.command}`];
  });
}

function moved(commands: OlcZoneCommand[], reorder: ReorderRequest): OlcZoneCommand[] {
  const result = [...commands];
  const [command] = result.splice(reorder.from, 1);
  result.splice(reorder.to, 0, command);
  return result;
}

function ReorderConfirmation({
  commands,
  request,
  disabled,
  onCancel,
  onConfirm,
}: {
  commands: OlcZoneCommand[];
  request: ReorderRequest;
  disabled: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const before = edgeLabels(commands);
  const after = edgeLabels(moved(commands, request));
  const removed = before.filter((edge) => !after.includes(edge));
  const added = after.filter((edge) => !before.includes(edge));
  return (
    <div className="border border-accent bg-paper-deep px-4 py-4" role="dialog" aria-label="Confirm command reorder">
      <p className="font-semibold text-ink">Reorder changes dependency edges</p>
      <p className="mt-1 text-sm text-ink-muted">IfFlag follows the command it is attached to, but its previous command changes when the order moves.</p>
      <div className="mt-3 grid gap-3 text-xs md:grid-cols-2">
        <div>
          <p className="font-semibold uppercase tracking-wider text-ink-muted">Edges removed</p>
          {removed.length ? removed.map((edge) => <p key={edge} className="mt-1 font-mono text-accent">− {edge}</p>) : <p className="mt-1 text-ink-muted">None</p>}
        </div>
        <div>
          <p className="font-semibold uppercase tracking-wider text-ink-muted">Edges added</p>
          {added.length ? added.map((edge) => <p key={edge} className="mt-1 font-mono text-online">+ {edge}</p>) : <p className="mt-1 text-ink-muted">None</p>}
        </div>
      </div>
      <div className="mt-4 flex gap-2">
        <button type="button" disabled={disabled} onClick={onCancel} className="border border-rule px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:opacity-50">Cancel</button>
        <button type="button" disabled={disabled} onClick={onConfirm} className="border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:opacity-50">Apply reorder</button>
      </div>
    </div>
  );
}

function CommandArgument({
  argument,
  value,
  disabled,
  onValue,
}: {
  argument: OlcZoneCommandArgument;
  value: number;
  disabled: boolean;
  onValue: (value: number) => void;
}) {
  if (!argument.visible) return null;
  if (argument.control === 'select') {
    return (
      <label className="block">
        <span className="mb-1 block text-[11px] font-semibold uppercase tracking-wider text-ink-muted">{argument.label}</span>
        <select value={String(value)} disabled={disabled} onChange={(event) => onValue(Number(event.currentTarget.value))} className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink">
          {argument.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
      </label>
    );
  }
  return (
    <label className="block">
      <span className="mb-1 block text-[11px] font-semibold uppercase tracking-wider text-ink-muted">{argument.label}</span>
      <ServerProposal
        value={value}
        identity={`zone-arg-${argument.key}-${value}`}
        disabled={disabled}
        min={argument.bounds?.min}
        max={argument.bounds?.max}
        onCommit={(raw) => onValue(Number(raw))}
      />
    </label>
  );
}

export function ZoneCommandTimeline({ schema, commands, disabled = false, onOperation }: ZoneCommandTimelineProps) {
  const [reorder, setReorder] = useState<ReorderRequest | null>(null);
  const [newCommandType, setNewCommandType] = useState(schema.zoneCommands[0]?.command || '');
  const descriptors = useMemo(() => schema.zoneCommands, [schema.zoneCommands]);
  const addDescriptor = descriptorFor(schema, newCommandType);

  const addCommand = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const form = event.currentTarget;
    const data = new FormData(form);
    const command = String(data.get('command') || descriptors[0]?.command || 'M');
    onOperation([{
      kind: 'add_command',
      index: -1,
      command,
      if_flag: Number(data.get('if_flag') || 0),
      arg1: Number(data.get('arg1') || 0),
      arg2: Number(data.get('arg2') || 0),
      arg3: Number(data.get('arg3') || 0),
    }]);
    form.reset();
  };

  return (
    <section className="border-t border-rule pt-5">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <h2 className="text-lg text-ink">Reset command timeline</h2>
          <p className="mt-1 text-sm text-ink-muted">IfFlag depends on the previous command. Reordering is an explicit, edge-changing operation.</p>
        </div>
        <span className="font-mono text-xs text-ink-muted">{commands.length} commands</span>
      </div>

      <div className="mt-4 space-y-3">
        {commands.map((command, index) => {
          const descriptor = descriptorFor(schema, command.command);
          return (
            <div key={`${command.position}-${command.command}`} className="border border-rule bg-paper px-4 py-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <span className="font-mono text-sm text-accent">#{index + 1}</span>
                  <span className="font-semibold text-ink">{descriptor?.label || command.command}</span>
                  {command.ifFlag !== 0 && <span className="border border-accent px-2 py-1 font-mono text-[10px] uppercase tracking-wider text-accent">if previous succeeds</span>}
                </div>
                <div className="flex gap-2">
                  {index > 0 && <button type="button" disabled={disabled} onClick={() => setReorder({ from: index, to: index - 1 })} className="border border-rule px-2 py-1 text-xs text-ink hover:bg-paper-deep disabled:opacity-50">↑ Move</button>}
                  {index < commands.length - 1 && <button type="button" disabled={disabled} onClick={() => setReorder({ from: index, to: index + 1 })} className="border border-rule px-2 py-1 text-xs text-ink hover:bg-paper-deep disabled:opacity-50">↓ Move</button>}
                  <button type="button" disabled={disabled} onClick={() => onOperation([{ kind: 'remove_command', index: command.position }])} className="px-2 py-1 text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep disabled:opacity-50">Remove</button>
                </div>
              </div>
              <div className="mt-4 grid gap-3 md:grid-cols-3">
                {descriptor?.arguments.map((argument, argumentIndex) => (
                  <CommandArgument
                    key={argument.key}
                    argument={argument}
                    value={argumentValue(command, argumentIndex)}
                    disabled={disabled}
                    onValue={(value) => onOperation([commandWith(command, argumentIndex, value)])}
                  />
                ))}
              </div>
              <label className="mt-3 flex items-center gap-2 text-sm text-ink">
                <input
                  type="checkbox"
                  checked={command.ifFlag !== 0}
                  disabled={disabled || index === 0}
                  onChange={(event) => onOperation([{ kind: 'modify_command', index: command.position, command: command.command, if_flag: event.currentTarget.checked ? 1 : 0, arg1: command.arg1, arg2: command.arg2, arg3: command.arg3 }])}
                  className="h-4 w-4 accent-accent"
                />
                Run only if the previous command succeeds
              </label>
            </div>
          );
        })}
      </div>

      <form className="mt-4 border border-rule bg-paper-deep px-4 py-4" onSubmit={addCommand}>
        <h3 className="text-sm font-semibold uppercase tracking-wider text-ink">Add command</h3>
        <div className="mt-3 grid gap-3 md:grid-cols-4">
          <label className="block"><span className="mb-1 block text-[11px] font-semibold uppercase tracking-wider text-ink-muted">Command type</span><select name="command" value={newCommandType} onChange={(event) => setNewCommandType(event.currentTarget.value)} disabled={disabled} className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink">{descriptors.map((entry) => <option key={entry.command} value={entry.command}>{entry.command} — {entry.label}</option>)}</select></label>
          {addDescriptor?.arguments.map((argument, index) => (
            <label key={argument.key} className={argument.visible ? 'block' : 'hidden'}>
              <span className="mb-1 block text-[11px] font-semibold uppercase tracking-wider text-ink-muted">{argument.label}</span>
              {argument.control === 'select' ? (
                <select name={`arg${index + 1}`} disabled={disabled} defaultValue={argument.options[0]?.value ?? 0} className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink">
                  {argument.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                </select>
              ) : (
                <input name={`arg${index + 1}`} type="number" min={argument.bounds?.min} max={argument.bounds?.max} disabled={disabled} className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink" />
              )}
            </label>
          ))}
        </div>
        <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-ink"><input name="if_flag" type="checkbox" value="1" disabled={disabled} className="h-4 w-4 accent-accent" /> Depends on previous success</label>
          <button type="submit" disabled={disabled || descriptors.length === 0} className="border border-accent bg-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-paper hover:bg-accent-deep disabled:opacity-50">Add command</button>
        </div>
      </form>

      {reorder && <div className="mt-4"><ReorderConfirmation commands={commands} request={reorder} disabled={disabled} onCancel={() => setReorder(null)} onConfirm={() => { onOperation([{ kind: 'reorder_command', index: commands[reorder.from].position, to_index: reorder.to }]); setReorder(null); }} /></div>}
    </section>
  );
}
