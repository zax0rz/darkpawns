// The findings/agent vocabulary, as documented on pkg/admin/agent_store.go's
// Finding and AgentStatus structs. The server stores these as free strings
// (agents post them directly), so this is the console's single copy of the
// values it offers; filters, forms, and inline editors all read it.

export const FINDING_SOURCES = [
  { value: 'reek', label: 'Reek' },
  { value: 'daeron', label: 'Daeron' },
] as const;

export const FINDING_SEVERITIES = [
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
] as const;

export const FINDING_STATUSES = [
  { value: 'open', label: 'Open' },
  { value: 'confirmed', label: 'Confirmed' },
  { value: 'rejected', label: 'Rejected' },
  { value: 'fixed', label: 'Fixed' },
] as const;

export const AGENT_STATUSES = [
  { value: 'active', label: 'Active' },
  { value: 'idle', label: 'Idle' },
  { value: 'error', label: 'Error' },
] as const;
