import { useQuery } from '@tanstack/react-query';
import { olcApi } from '../../api/olc';
import { ServerProposal } from './ServerProposal';

interface VNumPickerProps {
  kind: 'room' | 'mob' | 'obj' | 'shop';
  value: number;
  identity: string;
  label: string;
  disabled?: boolean;
  min?: number;
  max?: number;
  showLabel?: boolean;
  onCommit: (value: string) => void;
}

const kindLabel: Record<VNumPickerProps['kind'], string> = {
  room: 'room',
  mob: 'mobile',
  obj: 'object',
  shop: 'shop',
};

export function VNumPicker({
  kind,
  value,
  identity,
  label,
  disabled = false,
  min,
  max,
  showLabel = true,
  onCommit,
}: VNumPickerProps) {
  const lookupQuery = useQuery({
    queryKey: ['olc-vnum-lookup', kind, value],
    queryFn: () => olcApi.lookup(kind, value),
    enabled: Number.isInteger(value) && value >= 0,
    staleTime: 30_000,
    retry: false,
  });

  return (
    <div>
      {showLabel && <label htmlFor={identity} className="mb-1 block text-[11px] font-semibold uppercase tracking-wider text-ink-muted">{label}</label>}
      <ServerProposal
        value={value}
        identity={identity}
        disabled={disabled}
        min={min}
        max={max}
        onCommit={onCommit}
      />
      {lookupQuery.isLoading && <p className="mt-1 text-[11px] text-ink-muted">Checking {kindLabel[kind]}…</p>}
      {!lookupQuery.isLoading && lookupQuery.data?.exists && (
        <p className="mt-1 truncate text-[11px] text-online" title={lookupQuery.data.name}>
          {lookupQuery.data.name || `Existing ${kindLabel[kind]}`}
        </p>
      )}
      {!lookupQuery.isLoading && lookupQuery.data && !lookupQuery.data.exists && (
        <p className="mt-1 text-[11px] text-accent">No {kindLabel[kind]} at this VNUM.</p>
      )}
    </div>
  );
}
