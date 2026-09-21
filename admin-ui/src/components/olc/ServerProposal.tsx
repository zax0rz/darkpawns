interface ServerProposalProps {
  value: string | number;
  identity: string;
  disabled?: boolean;
  multiline?: boolean;
  min?: number;
  max?: number;
  onCommit: (value: string) => void;
}

export function ServerProposal({
  value,
  identity,
  disabled = false,
  multiline = false,
  min,
  max,
  onCommit,
}: ServerProposalProps) {
  const commonProps = {
    id: identity,
    name: identity,
    disabled,
    'aria-label': identity,
    className:
      'w-full border border-rule bg-paper px-3 py-2 text-sm text-ink outline-none transition-colors focus:border-accent focus:ring-1 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50',
    defaultValue: value,
    key: `${identity}:${value}`,
    onBlur: (event: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      onCommit(event.currentTarget.value),
  };

  if (multiline) return <textarea {...commonProps} rows={6} />;

  return (
    <input
      {...commonProps}
      type={typeof value === 'number' ? 'number' : 'text'}
      min={min}
      max={max}
      inputMode={typeof value === 'number' ? 'numeric' : undefined}
    />
  );
}
