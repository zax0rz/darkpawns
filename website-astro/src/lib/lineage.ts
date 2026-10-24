import data from './lineage.json';

export const lineage = data;
const escape = (value: string) => value.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' }[c]!));
export const nodeName = (id: string) => data.nodes.find((node) => node.id === id)!.label;
export const sourceFor = (id: string | null) => data.sources.find((source) => source.id === id);

export function lineageSVG(standalone = false): string {
  const highlighted = new Set(['diku:c1', 'c1:c2', 'c2:c3', 'c3:dp', 'dp:go']);
  const paths = data.edges.map((edge) => {
    const parent = data.nodes.find((node) => node.id === edge.parent)!;
    const child = data.nodes.find((node) => node.id === edge.child)!;
    const middle = (parent.y + child.y) / 2;
    const path = edge.parent === 'merc1' && edge.child === 'rom1'
      ? `M ${parent.x - 79} ${parent.y} H 972 V 775 H ${child.x} V ${child.y - 26}`
      : `M ${parent.x} ${parent.y + 26} V ${middle} H ${child.x} V ${child.y - 26}`;
    return `<path class="edge ${edge.state} ${highlighted.has(`${edge.parent}:${edge.child}`) ? 'our-line' : ''}" d="${path}"/>`;
  }).join('');
  const nodes = data.nodes.map((node) => {
    const edge = data.edges.find((item) => item.child === node.id);
    const source = edge && sourceFor(edge.source);
    const href = standalone ? (source?.url?.startsWith('/') ? `https://darkpawns.org${source.url}` : source?.url) : (edge || node.id === 'socket' ? `#lineage-${node.id}` : undefined);
    const label = `${node.label}${edge ? `: ${edge.state === 'unconfirmed' ? 'unconfirmed connection from' : edge.state === 'influence' ? 'influenced by' : 'descends from'} ${nodeName(edge.parent)}` : ''}`;
    const inside = `<title>${escape(label)}</title><rect x="${node.x - 79}" y="${node.y - 26}" width="158" height="52"/><text x="${node.x}" y="${node.y + 7}">${escape(node.label)}</text>`;
    const classes = ['node', ['dp', 'go'].includes(node.id) ? 'ours' : '', node.id === 'diku' ? 'root' : ''].join(' ');
    return href ? `<a class="${classes}" href="${escape(href)}" aria-label="${escape(label)}">${inside}</a>` : `<g class="${classes}">${inside}</g>`;
  }).join('');
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${standalone ? '0 -140 1810 1500' : '0 0 1810 1220'}" width="1810" height="${standalone ? 1500 : 1220}" role="img" aria-labelledby="family-tree-title family-tree-desc"><title id="family-tree-title">The DikuMUD family tree, with Dark Pawns highlighted</title><desc id="family-tree-desc">A working reconstruction. Solid lines indicate supported ancestry, dashed lines remain unconfirmed, and a dotted line marks AberMUD’s design influence. Dark Pawns descends from CircleMUD 3.0, following earlier bases on 1.7 and 2.2. SocketMUD is shown separately because its proposed Merc connection is unconfirmed. A complete text list follows the chart on the About page.</desc>${standalone ? `<style>svg{background:#EFE7D6;color:#1A1614}text{font:22px Georgia,serif;text-anchor:middle;fill:#1A1614}.edge{fill:none;stroke:#1A1614;stroke-width:1.3}.unconfirmed{stroke-dasharray:7 5}.influence{stroke-dasharray:2 5}.our-line{stroke:#A8201A;stroke-width:2.5}.node rect{fill:#EFE7D6;stroke:#1A1614;stroke-width:1.2}.ours rect{stroke:#A8201A;stroke-width:2}.ours text{fill:#A8201A}.root text,.ours text{font-weight:bold}.export-title{font-size:42px}.export-note{font-size:22px;fill:#56504A}</style><text class="export-title" x="905" y="-75">The DikuMUD family tree</text><text class="export-note" x="905" y="-25">Solid: supported ancestry. Dashed: unconfirmed. Dotted: design influence.</text>` : ''}<g>${paths}</g><g>${nodes}</g>${standalone ? '<a href="https://darkpawns.org/about/#lineage"><text class="export-note" x="905" y="1280">Working reconstruction, October 23, 2026. Sources and limitations: darkpawns.org/about/#lineage</text></a>' : ''}<text class="influence-label" x="1060" y="90">design influence</text><text class="separate-label" x="1530" y="1010">separate family</text></svg>`;
}
