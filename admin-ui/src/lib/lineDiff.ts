export type DiffLine = { kind: 'same' | 'server' | 'draft'; text: string };

// Past this many LCS cells the diff is too costly to compute in the browser;
// the caller falls back to showing both copies whole.
const MAX_CELLS = 4_000_000;

// lineDiff is a plain LCS line diff between the server copy and the local
// draft. It is display-only: the conflict view never merges.
export function lineDiff(server: string, draft: string): DiffLine[] | null {
  const a = server.split('\n');
  const b = draft.split('\n');
  const n = a.length;
  const m = b.length;
  if (n * m > MAX_CELLS) return null;
  const width = m + 1;
  const lcs = new Uint32Array((n + 1) * width);
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i * width + j] = a[i] === b[j]
        ? lcs[(i + 1) * width + j + 1] + 1
        : Math.max(lcs[(i + 1) * width + j], lcs[i * width + j + 1]);
    }
  }
  const out: DiffLine[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ kind: 'same', text: a[i] });
      i++;
      j++;
    } else if (lcs[(i + 1) * width + j] >= lcs[i * width + j + 1]) {
      out.push({ kind: 'server', text: a[i++] });
    } else {
      out.push({ kind: 'draft', text: b[j++] });
    }
  }
  while (i < n) out.push({ kind: 'server', text: a[i++] });
  while (j < m) out.push({ kind: 'draft', text: b[j++] });
  return out;
}
