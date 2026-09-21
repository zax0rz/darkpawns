import type { OlcSchema } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface ScriptTarget {
  scriptName: string;
  scriptFunctions: number;
}

interface LiveScriptControlsProps {
  schema: OlcSchema;
  value: ScriptTarget;
  disabled?: boolean;
  onScriptName: (value: string) => void;
  onScriptFlag: (bit: number, enabled: boolean) => void;
}

function isFlagSet(flags: number, bit: number): boolean {
  return (flags & 2 ** bit) !== 0;
}

export function LiveScriptControls({
  schema,
  value,
  disabled = false,
  onScriptName,
  onScriptFlag,
}: LiveScriptControlsProps) {
  const scriptName = schema.fields.find((field) => field.key === 'script_name');
  const scriptFlags = schema.fields.find((field) => field.key === 'script_flags');
  if (!scriptName && !scriptFlags) return null;

  return (
    <section className="border-t border-rule pt-5">
      <div className="mb-4 border border-accent bg-paper-deep px-4 py-3">
        <p className="text-sm font-semibold text-accent">Live script controls</p>
        <p className="mt-1 text-sm text-ink">
          Applies immediately; Discard will not undo it.
        </p>
      </div>
      <div className="grid gap-5 lg:grid-cols-2">
        {scriptName && (
          <div>
            <label htmlFor="olc-script-name" className="mb-2 block text-sm font-semibold text-ink">
              {scriptName.label}
            </label>
            <ServerProposal
              value={value.scriptName}
              identity="olc-script-name"
              disabled={disabled}
              onCommit={onScriptName}
            />
          </div>
        )}
        {scriptFlags && (
          <fieldset>
            <legend className="mb-2 text-sm font-semibold text-ink">{scriptFlags.label}</legend>
            <div className="grid gap-2 sm:grid-cols-2">
              {(scriptFlags.options || []).map((option) => (
                <label
                  key={option.value}
                  className="flex min-h-10 items-center gap-2 border border-rule bg-paper px-3 py-2 text-sm text-ink hover:bg-paper-deep"
                >
                  <input
                    type="checkbox"
                    checked={isFlagSet(value.scriptFunctions, option.value)}
                    disabled={disabled}
                    onChange={(event) => onScriptFlag(option.value, event.currentTarget.checked)}
                    className="h-4 w-4 accent-accent"
                  />
                  <span>{option.label}</span>
                </label>
              ))}
            </div>
          </fieldset>
        )}
      </div>
    </section>
  );
}
