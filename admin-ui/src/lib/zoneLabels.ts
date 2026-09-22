import type { OlcSchema } from '../api/olc';

// resetModeLabel reads the zone schema's reset_mode vocabulary (zedit's own
// strings), so every zone view says the same thing. Until the schema loads it
// shows an ellipsis rather than a raw "Mode 0" that looks like a real label;
// a value the schema does not name falls back to its number.
export function resetModeLabel(mode: number, schema?: Pick<OlcSchema, 'fields'>): string {
  if (!schema) return '…';
  const field = schema.fields.find((entry) => entry.key === 'reset_mode');
  return field?.options?.find((option) => option.value === mode)?.label ?? String(mode);
}
