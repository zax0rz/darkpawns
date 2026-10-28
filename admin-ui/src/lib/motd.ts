// C: src/comm.c:1289 colour_expansion, mirrored by session.expandTerminalColors.
// Unknown markers and lone ampersands survive; && emits one literal ampersand.
export function motdTerminalText(text: string, colour = true): string {
  const codes = '&ndbgcrmywDBGCRMYW';
  const values = ['&', '\x1b[0m', ...[30, 34, 32, 36, 31, 35, 33, 37].map(n => `\x1b[0;${n}m`), ...[30, 34, 32, 36, 31, 35, 33, 37].map(n => `\x1b[1;${n}m`)];
  const expanded = colour ? text.replace(/&([\s\S])/g, (pair, code: string) => {
    const index = codes.indexOf(code);
    return index < 0 ? pair : values[index];
  }) : text;
  return expanded.replace(/\r\n|\n\r|\r|\n/g, pair => pair.length === 2 ? pair : '\r\n');
}

// tedit counts the bytes after CRLF expansion, even though it saves LF on disk.
export function motdBytes(text: string): number {
  return new TextEncoder().encode(text.replaceAll('\r', '').replaceAll('\n', '\r\n')).length;
}

export const gossipNotice = "What's said on gossip echoes to our Discord.";
export function discordSnippet(invite: string): string | null {
  try {
    const url = new URL(invite.trim());
    if (url.protocol !== 'https:' || url.username || url.password || url.port || url.search || url.hash) return null;
    if (!((url.hostname === 'discord.gg' && /^\/[\w-]+\/?$/.test(url.pathname)) ||
      (url.hostname === 'discord.com' && /^\/invite\/[\w-]+\/?$/.test(url.pathname)))) return null;
    return `&CJoin us on Discord&n\n${url.href}\n`;
  } catch { return null; }
}
