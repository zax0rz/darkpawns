import { markdownResponse } from '../../lib/markdown';
import { lineage, nodeName, sourceFor } from '../../lib/lineage';

export function GET() {
  const relationships = lineage.edges.map((edge) => {
    const source = sourceFor(edge.source);
    return `- **${nodeName(edge.parent)} to ${nodeName(edge.child)}** (${edge.state === 'confirmed' ? 'supported ancestry' : edge.state === 'influence' ? 'design influence' : 'unconfirmed connection'}). ${source ? `[${source.title}](${source.url.startsWith('/') ? `https://darkpawns.org${source.url}` : source.url}): ${source.note}` : 'Reported in the earlier chart; not established from a primary source in this review.'} ${edge.note}`;
  }).join('\n');
  return markdownResponse(`
# The DikuMUD family tree

Working reconstruction, reviewed ${lineage.reviewed}. This is a selection of the family, not every game or contribution. Supported ancestry can omit intermediate releases; an unconfirmed connection needs more evidence and is not necessarily false. Design influence does not assert shared source code.

Dark Pawns began on CircleMUD 1.7 in September 1994, according to Frontline's preserved account. It moved to 2.2 in December 1995, 3.0pl9 in March 1996, and 3.0pl11 in June 1996. The chart places the surviving C game under CircleMUD 3.x. DPReturns and the Go restoration are separate descendants of the original Dark Pawns.

Merc's credits acknowledge Copper, Alfa, and Silly code; ROM 2.4 uses Merc 2.1. A single-parent tree leaves those exchanges out. SocketMUD is shown separately because the earlier chart's Merc connection remains unconfirmed.

## Relationships and sources

${relationships}

[View the diagram](https://darkpawns.org/about/#lineage). [Open the full-size SVG](https://darkpawns.org/about/lineage.svg).
`, { title: 'The DikuMUD family tree', textKind: 'reconstruction', voiceLayer: 'mythic-admin', source: 'Primary source credits and histories linked per relationship' }, '/about/#lineage');
}
