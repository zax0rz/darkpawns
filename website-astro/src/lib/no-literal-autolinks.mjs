// GFM turns bare URLs, www. hostnames and email addresses into live links.
// Almost every Markdown file on this site is verbatim text somebody else wrote
// (see the smart punctuation note in astro.config.mjs), so a hostname a player
// typed in 2004 would ship as a link to whoever owns that domain now.
//
// A link an author wrote, [text](url) or <url>, starts at its bracket. A GFM
// autolink literal starts at its first letter, so that is the one this plugin
// turns back into plain text. Sätteri 0.10 reports JavaScript string offsets;
// positions must be enabled explicitly for this visitor.
const OPEN_BRACKET = '[';
const OPEN_ANGLE = '<';

export const noLiteralAutolinks = {
  name: 'no-literal-autolinks',
  options: { position: true },
  link(node, ctx) {
    const start = node.position?.start?.offset;
    if (start === undefined) return;
    const first = ctx.source[start];
    if (first === OPEN_BRACKET || first === OPEN_ANGLE) return;
    ctx.replaceNode(node, {
      type: 'text',
      value: node.children.map((child) => child.value ?? '').join(''),
    });
  },
};
