import { useState } from 'react';
import type { OlcObject, OlcPatchOperation, OlcSchema } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface ObjectValueMatrixEditorProps {
  schema: OlcSchema;
  object: OlcObject;
  disabled?: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}

export function ObjectValueMatrixEditor({ schema, object, disabled = false, onOperation }: ObjectValueMatrixEditorProps) {
  const row = schema.valueMatrix.find((entry) => entry.itemType === object.typeFlag);
  if (!row) return null;
  return (
    <section className="border-t border-rule pt-5">
      <h2 className="mb-4 text-lg text-ink">Values</h2>
      <div className="grid gap-5 lg:grid-cols-2">
        {row.values.map((field, index) => {
          if (!field.visible) return null;
          return (
            <ValueSlot
              key={`${object.typeFlag}:${index}`}
              field={field}
              index={index}
              current={object.values[index] || 0}
              disabled={disabled}
              onOperation={onOperation}
            />
          );
        })}
      </div>
    </section>
  );
}

function ValueSlot({
  field,
  index,
  current,
  disabled,
  onOperation,
}: {
  field: { label: string; control: string; min: number; max: number; options?: { value: number; label: string }[] };
  index: number;
  current: number;
  disabled: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}) {
  const operation = `set_value${index + 1}`;
  const commit = (value: string) => onOperation([{ kind: operation, value: Number(value) }]);
  return (
    <div>
      <label className="mb-2 block text-sm font-semibold text-ink">{field.label}</label>
      {field.control === 'spell_picker' ? (
        <SpellPicker field={field} current={current} disabled={disabled} onCommit={commit} />
      ) : field.control === 'select' ? (
        <select
          value={String(current)}
          disabled={disabled}
          onChange={(event) => commit(event.currentTarget.value)}
          className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
        >
          {(field.options || []).map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
      ) : field.control === 'checkbox_group' ? (
        <div className="grid gap-2 sm:grid-cols-2">
          {(field.options || []).map((option) => (
            <label key={option.value} className="flex min-h-10 items-center gap-2 border border-rule bg-paper px-3 py-2 text-sm text-ink hover:bg-paper-deep">
              <input
                type="checkbox"
                checked={(current & 2 ** option.value) !== 0}
                disabled={disabled}
                onChange={() => onOperation([{ kind: 'toggle_container_flag', value: option.value }])}
                className="h-4 w-4 accent-accent"
              />
              <span>{option.label}</span>
            </label>
          ))}
        </div>
      ) : (
        <ServerProposal value={current} identity={`olc-value-${index}`} disabled={disabled} min={field.min} max={field.max} onCommit={commit} />
      )}
      {field.control !== 'checkbox_group' && field.control !== 'select' && (
        <p className="mt-1 font-mono text-[11px] text-ink-muted">{field.min}–{field.max}</p>
      )}
    </div>
  );
}

function SpellPicker({
  field,
  current,
  disabled,
  onCommit,
}: {
  field: { options?: { value: number; label: string }[] };
  current: number;
  disabled: boolean;
  onCommit: (value: string) => void;
}) {
  const [search, setSearch] = useState('');
  const options = (field.options || []).filter((option) => option.label.toLowerCase().includes(search.toLowerCase()));
  return (
    <div className="space-y-2">
      <input
        type="search"
        value={search}
        disabled={disabled}
        onChange={(event) => setSearch(event.currentTarget.value)}
        placeholder="Search spells…"
        aria-label="Search spells"
        className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
      />
      <select
        value={String(current)}
        disabled={disabled}
        onChange={(event) => onCommit(event.currentTarget.value)}
        className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
      >
        {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
    </div>
  );
}
