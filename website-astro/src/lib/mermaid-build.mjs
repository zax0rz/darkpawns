import { createHash } from 'node:crypto';
import { existsSync, readdirSync, readFileSync, mkdirSync, writeFileSync } from 'node:fs';
import { homedir } from 'node:os';
import { join } from 'node:path';
import { chromium } from 'playwright';

const cache = new URL('../../.generated/diagrams/', import.meta.url);
const diagramTheme = { startOnLoad: false, securityLevel: 'strict', theme: 'base', look: 'classic', deterministicIds: true, themeCSS: '* { filter: none !important; }',
  themeVariables: { fontFamily: 'Georgia, serif', primaryColor: '#E5DAC1', primaryTextColor: '#1A1614', primaryBorderColor: '#1A1614', lineColor: '#56504A', secondaryColor: '#EFE7D6', tertiaryColor: '#EFE7D6', dropShadow: 'none' } };
const mermaidVersion = JSON.parse(readFileSync(new URL('../../node_modules/mermaid/package.json', import.meta.url), 'utf8')).version;
function executable() {
  if (process.env.CHROMIUM_PATH) return process.env.CHROMIUM_PATH;
  if (existsSync(chromium.executablePath())) return chromium.executablePath();
  const root = join(homedir(), '.cache/ms-playwright');
  for (const dir of existsSync(root) ? readdirSync(root).sort().reverse() : []) {
    for (const suffix of ['chrome-linux/chrome', 'chrome-linux64/chrome']) {
      const path = join(root, dir, suffix);
      if (dir.startsWith('chromium-') && existsSync(path)) return path;
    }
  }
  throw new Error('Mermaid build requires Chromium. Run: npx playwright install chromium');
}
export default function mermaidBuild() {
  return {
    name: 'darkpawns-mermaid',
    async code(node) {
      if (node.lang !== 'mermaid') return;
      mkdirSync(cache, { recursive: true });
      const id = 'diagram-' + createHash('sha256').update(mermaidVersion + JSON.stringify(diagramTheme) + node.value).digest('hex').slice(0, 16);
      const file = new URL(`${id}.svg`, cache);
      let svg;
      if (existsSync(file)) svg = readFileSync(file, 'utf8');
      else {
        const browser = await chromium.launch({ executablePath: executable(), headless: true });
        try {
          const page = await browser.newPage();
          await page.setContent('<!doctype html><html><body></body></html>');
          await page.addScriptTag({ path: new URL('../../node_modules/mermaid/dist/mermaid.min.js', import.meta.url).pathname });
          svg = await page.evaluate(async ({ source, id, theme }) => {
            mermaid.initialize({ ...theme, deterministicIDSeed: id });
            return (await mermaid.render(id, source)).svg;
          }, { source: node.value, id, theme: diagramTheme });
          writeFileSync(file, svg);
        } finally { await browser.close(); }
      }
      return { type: 'html', value: `<figure class="mermaid-diagram">${svg}</figure>` };
    },
  };
}
