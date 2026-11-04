// Text about a log entry, built only from fields the capture itself supplies
// (scripts/vargus_logs.py). Shared by the page, its Markdown twin, llms.txt
// and the character pages, so every place says the same thing.

type LogDate = { text: string; iso?: string; basis: string };
type LogData = { title: string; lines: number; recordedBy: string[]; cast: string[]; date?: LogDate };

/** "on 12 December 2000", "between 15 and 27 October 2000", "about 1 September 2000". */
export const whenPlayed = (date: LogDate) =>
  /^\d/.test(date.text) ? `on ${date.text}` : date.text[0].toLowerCase() + date.text.slice(1);

export const recorders = (data: LogData) =>
  data.recordedBy.length ? data.recordedBy.join(' and ') : 'not shown in the capture';

/** Everyone on screen apart from whoever recorded it. */
export const others = (data: LogData) => data.cast.filter((name) => !data.recordedBy.includes(name));

export function describeLog(data: LogData, maxNames = 12) {
  const names = others(data);
  const shown = names.slice(0, maxNames).join(', ') + (names.length > maxNames ? ` and ${names.length - maxNames} more` : '');
  return `${data.title}, a ${data.lines.toLocaleString('en-US')}-line game log from the original Dark Pawns server`
    + (data.recordedBy.length ? `, recorded by ${recorders(data)}` : '')
    + (data.date ? `, played ${whenPlayed(data.date)}` : '')
    + '.'
    + (names.length ? ` On screen: ${shown}.` : '');
}

/** What was removed, said plainly: "12 private tells, 6 lines". */
export function redactionSummary(r: Record<string, number>) {
  const label: Record<string, [string, string]> = {
    tells: ['private tell', 'private tells'],
    mail: ['mail line', 'mail lines'],
    lines: ['line with a slur or sexual violence', 'lines with slurs or sexual violence'],
    emails: ['email address', 'email addresses'],
    ips: ['IP address', 'IP addresses'],
    passwords: ['password', 'passwords'],
  };
  return Object.entries(label)
    .filter(([key]) => r[key])
    .map(([key, [one, many]]) => `${r[key]} ${r[key] === 1 ? one : many}`)
    .join(', ');
}

type LogEntry = { id: string; data: LogData & { appearances: Record<string, number> } };

/** Every log a character appears in, with the line they first appear on and
 *  whether it was their own screen. Names match case-insensitively. */
export function appearancesOf<E extends LogEntry>(entries: E[], key: string) {
  return entries
    .map((entry) => {
      const name = entry.data.cast.find((n) => n.toLowerCase() === key);
      return name ? { entry, name, line: entry.data.appearances[name] ?? 1, recorded: entry.data.recordedBy.includes(name) } : null;
    })
    .filter((row): row is NonNullable<typeof row> => row !== null)
    .sort((a, b) => a.entry.data.title.localeCompare(b.entry.data.title, 'en'));
}

export const characterKeys = (entries: LogEntry[]) =>
  [...new Set(entries.flatMap((entry) => entry.data.cast.map((name) => name.toLowerCase())))];
