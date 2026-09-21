import type { OlcObject, OlcPatchOperation, OlcSchema } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface ObjectAppliesEditorProps {
  schema: OlcSchema;
  object: OlcObject;
  disabled?: boolean;
  onOperation: (operations: OlcPatchOperation[]) => void;
}

export function ObjectAppliesEditor({ schema, object, disabled = false, onOperation }: ObjectAppliesEditorProps) {
  const descriptor = schema.applies;
  if (!descriptor) return null;
  const firstApply = descriptor.options.find((option) => option.value !== 0)?.value || 0;
  const replace = (index: number, location: number, modifier: number) => {
    if (location === 0) {
      onOperation([{ kind: descriptor.removeOperation, index }]);
      return;
    }
    onOperation([
      { kind: descriptor.removeOperation, index },
      { kind: descriptor.addOperation, index, location, modifier },
    ]);
  };

  return (
    <section className="border-t border-rule pt-5">
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <h2 className="text-lg text-ink">Applies</h2>
        <span className="font-mono text-[11px] uppercase tracking-wider text-ink-muted">{object.affects.length}/{descriptor.max} slots</span>
      </div>
      <div className="space-y-3">
        {object.affects.length === 0 && <p className="border border-dashed border-rule px-4 py-5 text-sm text-ink-muted">No applies yet.</p>}
        {object.affects.map((affect, index) => (
          <article key={`${index}:${affect.location}:${affect.modifier}`} className="grid gap-3 border border-rule bg-paper p-4 md:grid-cols-[1fr_10rem_auto] md:items-end">
            <div>
              <label htmlFor={`apply-${index}-location`} className="mb-2 block text-sm font-semibold text-ink">Apply type</label>
              <select
                id={`apply-${index}-location`}
                value={String(affect.location)}
                disabled={disabled}
                onChange={(event) => replace(index, Number(event.currentTarget.value), affect.modifier)}
                className="w-full border border-rule bg-paper-deep px-3 py-2 text-sm text-ink outline-none focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50"
              >
                {descriptor.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor={`apply-${index}-modifier`} className="mb-2 block text-sm font-semibold text-ink">Modifier</label>
              <ServerProposal
                value={affect.modifier}
                identity={`apply-${index}-modifier`}
                disabled={disabled}
                onCommit={(value) => replace(index, affect.location, Number(value))}
              />
            </div>
            <button
              type="button"
              disabled={disabled}
              onClick={() => onOperation([{ kind: descriptor.removeOperation, index }])}
              className="border border-accent px-3 py-2 text-xs font-semibold uppercase tracking-wider text-accent hover:bg-accent hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
            >
              Remove
            </button>
          </article>
        ))}
      </div>
      <button
        type="button"
        disabled={disabled}
        onClick={() => onOperation([{ kind: descriptor.addOperation, location: firstApply, modifier: 0 }])}
        className="mt-4 border border-rule bg-paper-deep px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
      >
        Add apply
      </button>
    </section>
  );
}
