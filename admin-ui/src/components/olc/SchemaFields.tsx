import type { OlcRoom, OlcSchema, OlcSchemaField, RoomPatchOperation } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface SchemaFieldsProps {
  schema: OlcSchema;
  room: OlcRoom;
  dirty: string[];
  disabled?: boolean;
  onOperation: (operation: RoomPatchOperation) => void;
}

const FIELD_OPERATIONS: Record<string, string> = {
  name: 'set_room_name',
  description: 'set_room_description',
  flags: 'set_room_flag',
  sector: 'set_room_sector',
};

function isFlagSet(flags: string[], bit: number): boolean {
  const word = Math.floor(bit / 32);
  const parsed = Number(flags[word] || 0);
  return Number.isFinite(parsed) && (parsed & 2 ** (bit % 32)) !== 0;
}

function fieldValue(field: OlcSchemaField, room: OlcRoom): string | number {
  switch (field.key) {
    case 'name':
      return room.name;
    case 'description':
      return room.description;
    case 'sector':
      return room.sector;
    default:
      return '';
  }
}

function FieldLabel({ field, dirty }: { field: OlcSchemaField; dirty: boolean }) {
  return (
    <div className="mb-2 flex items-baseline justify-between gap-3">
      <label htmlFor={`olc-${field.key}`} className="text-sm font-semibold text-ink">
        {field.label}
      </label>
      {dirty && <span className="font-mono text-[10px] uppercase tracking-wider text-accent">changed</span>}
    </div>
  );
}

function renderScalarField(
  field: OlcSchemaField,
  room: OlcRoom,
  dirty: boolean,
  disabled: boolean,
  onOperation: (operation: RoomPatchOperation) => void,
) {
  const operation = FIELD_OPERATIONS[field.key];
  if (!operation) return null;
  const current = fieldValue(field, room);
  const commit = (raw: string) => {
    if (field.control === 'number') {
      onOperation({ kind: operation, value: Number(raw) });
    } else if (field.control === 'select') {
      onOperation({ kind: operation, value: Number(raw) });
    } else {
      onOperation({ kind: operation, text: raw });
    }
  };

  return (
    <div>
      <FieldLabel field={field} dirty={dirty} />
      <ServerProposal
        value={current}
        identity={`olc-${field.key}`}
        disabled={disabled}
        multiline={field.control === 'textarea'}
        min={field.bounds?.min}
        max={field.bounds?.max}
        onCommit={commit}
      />
      {field.bounds && (
        <p className="mt-1 font-mono text-[11px] text-ink-muted">
          {field.bounds.min}–{field.bounds.max}
        </p>
      )}
    </div>
  );
}

export function SchemaFields({ schema, room, dirty, disabled = false, onOperation }: SchemaFieldsProps) {
  return (
    <section className="border-t border-rule pt-5">
      <h2 className="mb-4 text-lg text-ink">Room fields</h2>
      <div className="grid gap-5 lg:grid-cols-2">
        {schema.fields.map((field) => {
          if (field.key === 'script_name' || field.key === 'script_flags') return null;
          if (field.control === 'checkbox_group') {
            return (
              <fieldset key={field.key} className="lg:col-span-2">
                <FieldLabel field={field} dirty={dirty.includes(field.key)} />
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                  {(field.options || []).map((option) => (
                    <label
                      key={option.value}
                      className="flex min-h-10 items-center gap-2 border border-rule bg-paper px-3 py-2 text-sm text-ink hover:bg-paper-deep"
                    >
                      <input
                        type="checkbox"
                        checked={isFlagSet(room.flags, option.value)}
                        disabled={disabled}
                        onChange={(event) =>
                          onOperation({
                            kind: FIELD_OPERATIONS[field.key] || 'set_room_flag',
                            bit: option.value,
                            enabled: event.currentTarget.checked,
                          })
                        }
                        className="h-4 w-4 accent-accent"
                      />
                      <span>{option.label}</span>
                    </label>
                  ))}
                </div>
              </fieldset>
            );
          }
          if (field.control === 'select') {
            const current = fieldValue(field, room);
            return (
              <div key={field.key}>
                <FieldLabel field={field} dirty={dirty.includes(field.key)} />
                <select
                  id={`olc-${field.key}`}
                  name={`olc-${field.key}`}
                  value={String(current)}
                  disabled={disabled}
                  onChange={(event) =>
                    onOperation({ kind: FIELD_OPERATIONS[field.key] || field.key, value: Number(event.currentTarget.value) })
                  }
                  className="w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {(field.options || []).map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
            );
          }
          return (
            <div key={field.key}>
              {renderScalarField(field, room, dirty.includes(field.key), disabled, onOperation)}
            </div>
          );
        })}
      </div>
    </section>
  );
}
