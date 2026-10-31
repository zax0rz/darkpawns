// Review evidence only; published docs imagery comes from make media.
import { mkdirSync, writeFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { chromium } from 'playwright';
const base = process.env.DOCS_REVIEW_URL ?? 'http://127.0.0.1:4389';
const output = resolve(process.env.DOCS_REVIEW_DIR ?? '../docs/briefs/2026-10-30-docs-pass-2/screenshots');
mkdirSync(output, { recursive: true });
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH });
const routes = [['index','/docs/'],['first-hour','/docs/play/first-hour/'],['install','/docs/host/install/'],['connect-client','/docs/play/connect-client/'],['edit-rooms','/docs/build/edit-rooms/'],['combat','/docs/reference/commands/combat/']];
const results = [];
try {
  for (const [size, width] of [['desktop',1440],['mobile',375]]) {
    const page = await browser.newPage({ viewport: { width, height: 1000 }, reducedMotion: 'reduce' });
    for (const [name, route] of routes) {
      await page.goto(base + route);
      await page.evaluate(() => document.fonts.ready);
      await page.locator('img').evaluateAll(images => Promise.all(images.map(image => image.decode())));
      if (await page.locator('img[src$=".gif"]').count()) await page.waitForTimeout(8000);
      await page.screenshot({ path: resolve(output, `${name}-${size}.png`), fullPage: true });
      const details = await page.evaluate(() => ({ overflow: document.documentElement.scrollWidth > innerWidth, title: document.querySelector('h1')?.textContent, edit: [...document.querySelectorAll('a[href]')].find((a) => a.href.includes('/edit/main/'))?.href }));
      if (details.overflow || !details.title || !details.edit) throw new Error(`${size} ${route}: invalid rendered page`);
      results.push({ size, width, route, ...details });
    }
    await page.close();
  }
} finally { await browser.close(); }
writeFileSync(resolve(output, 'checks.json'), JSON.stringify(results, null, 2) + '\n');
console.log(`docs-review: ${results.length} captures at 1440px and 375px, no horizontal overflow`);
