import type { OlcExtraDescription, OlcRoom, RoomPatchOperation } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface ExtraDescriptionListProps {
  room: OlcRoom;
  dirty: string[];
  disabled?: boolean;
  onOperation: (operation: RoomPatchOperation) => void;
}

function extraDirty(dirty: string[], index: number, suffix: string): boolean {
  return dirty.includes(`extra.${index}.${suffix}`) || dirty.includes(`extra.${index}`);
}

export function ExtraDescriptionList({ room, dirty, disabled = false, onOperation }: ExtraDescriptionListProps) {
  return (
    <section className="border-t border-rule pt-5">
      <div className="mb-4 flex items-baseline justify-between gap-3">
        <h2 className="text-lg text-ink">Extra descriptions</h2>
        <button
          type="button"
          disabled={disabled}
          onClick={() => onOperation({ kind: 'add_extra_description', keywords: '', text: '' })}
          className="border border-rule bg-paper-deep px-3 py-2 text-xs font-semibold uppercase tracking-wider text-ink hover:bg-ink hover:text-paper disabled:cursor-not-allowed disabled:opacity-50"
        >
          Add block
        </button>
      </div>
      {room.extraDescs.length === 0 ? (
        <p className="border border-dashed border-rule px-4 py-5 text-sm text-ink-muted">
          No keyed text blocks yet.
        </p>
      ) : (
        <div className="space-y-3">
          {room.extraDescs.map((extra, index) => (
            <ExtraDescription
              key={`${index}:${extra.keywords}:${extra.description}`}
              extra={extra}
              index={index}
              dirty={dirty}
              disabled={disabled}
              onOperation={onOperation}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function ExtraDescription({
  extra,
  index,
  dirty,
  disabled,
  onOperation,
}: {
  extra: OlcExtraDescription;
  index: number;
  dirty: string[];
  disabled: boolean;
  onOperation: (operation: RoomPatchOperation) => void;
}) {
  return (
    <article className="border border-rule bg-paper p-4">
      <div className="mb-3 flex items-baseline justify-between gap-3">
        <h3 className="text-base text-ink">Block {index + 1}</h3>
        <button
          type="button"
          disabled={disabled}
          onClick={() => onOperation({ kind: 'remove_extra_description', index })}
          className="text-xs font-semibold uppercase tracking-wider text-accent hover:text-accent-deep disabled:cursor-not-allowed disabled:opacity-50"
        >
          Remove
        </button>
      </div>
      <div className="grid gap-4">
        <div>
          <div className="mb-2 flex items-baseline justify-between gap-3">
            <label htmlFor={`extra-${index}-keywords`} className="text-sm font-semibold text-ink">Keywords</label>
            {extraDirty(dirty, index, 'keywords') && <span className="font-mono text-[10px] uppercase tracking-wider text-accent">changed</span>}
          </div>
          <ServerProposal
            value={extra.keywords}
            identity={`extra-${index}-keywords`}
            disabled={disabled}
            onCommit={(value) => onOperation({ kind: 'set_extra_keyword', index, keywords: value })}
          />
        </div>
        <div>
          <div className="mb-2 flex items-baseline justify-between gap-3">
            <label htmlFor={`extra-${index}-description`} className="text-sm font-semibold text-ink">Description</label>
            {extraDirty(dirty, index, 'description') && <span className="font-mono text-[10px] uppercase tracking-wider text-accent">changed</span>}
          </div>
          <ServerProposal
            value={extra.description}
            identity={`extra-${index}-description`}
            multiline
            disabled={disabled}
            onCommit={(value) => onOperation({ kind: 'set_extra_description', index, text: value })}
          />
        </div>
      </div>
    </article>
  );
}
