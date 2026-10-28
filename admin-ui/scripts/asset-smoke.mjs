// Use the disposable real-handler fixture from pkg/admin/console_preview_test.go.
// Run once against ADMIN_UI_DIR serving and once against Vite with its API proxy.
import { chromium, expect } from '@playwright/test';
import { readFile, readdir, mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const origin = process.env.ADMIN_ASSET_ORIGIN || 'http://127.0.0.1:4350';
const mode = process.env.ADMIN_ASSET_MODE || 'build';
if (!['build', 'dev'].includes(mode)) throw new Error('ADMIN_ASSET_MODE must be build or dev');
const dist = resolve(process.env.ADMIN_UI_DIST || 'dist');
const output = resolve(process.env.ADMIN_EVIDENCE_DIR || '../.local/admin-base-evidence', mode);
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
const report = { mode, origin, assets: [], network: [], errors: [] };
try {
  const context = await browser.newContext({ colorScheme: 'light', extraHTTPHeaders: { 'X-Requested-With': 'asset-smoke' } });
  const index = await context.request.get(origin + '/admin/');
  expect(index.status()).toBe(200);
  expect(await index.text(), 'HTML must not reference root assets').not.toMatch(/(?:src|href)=["']\/assets\//);
  // Disable browser caching so reload checks still prove asset status 200.
  await context.route('**/*', route => route.continue());
  if (mode === 'build') {
    // Check every emitted file, including lazy chunks and fonts. Compare bytes:
    // a SPA fallback returning index.html with status 200 must fail this check.
    for (const entry of await readdir(dist, { recursive: true, withFileTypes: true })) {
      if (!entry.isFile() || entry.name.endsWith('.html')) continue;
      const file = resolve(entry.parentPath, entry.name);
      const path = '/admin/' + file.slice(dist.length + 1).replaceAll('\\', '/');
      const response = await context.request.get(origin + path);
      expect(response.status(), path).toBe(200);
      expect(Buffer.compare(await response.body(), await readFile(file)), path + ' bytes').toBe(0);
      report.assets.push({ path, status: response.status() });
    }
  } else {
    for (const path of ['/admin/@vite/client', '/admin/src/main.tsx', '/admin/favicon.svg', '/admin/icons.svg']) {
      const response = await context.request.get(origin + path);
      expect(response.status(), path).toBe(200);
      expect(response.headers()['content-type'], path).not.toContain('text/html');
      report.assets.push({ path, status: response.status() });
    }
  }
  expect((await context.request.get(origin + '/preview-session')).ok()).toBeTruthy();
  for (const path of ['/health', '/admin/session']) {
    const response = await context.request.get(origin + path);
    expect(response.status(), path + ' API proxy').toBe(200);
    if (path === '/admin/session') await response.json();
  }
  const page = await context.newPage();
  const bad = [];
  let hmrConnected = false;
  page.on('websocket', socket => socket.on('framereceived', ({ payload }) => {
    if (typeof payload === 'string' && payload.includes('"type":"connected"')) hmrConnected = true;
  }));
  page.on('request', request => {
    if (new URL(request.url()).pathname.startsWith('/assets/')) bad.push(request.url());
  });
  page.on('response', response => {
    if (['script', 'stylesheet', 'font', 'image'].includes(response.request().resourceType())) {
      report.network.push({ url: response.url(), status: response.status() });
      if (response.status() !== 200) bad.push(`${response.status()} ${response.url()}`);
    }
  });
  page.on('pageerror', error => report.errors.push(error.message));
  for (const [name, path] of [['dashboard', '/admin/'], ['lazy-editor', '/admin/workshop/scripts']]) {
    await page.goto(origin + path);
    await expect(page.locator('main').first()).toBeVisible();
    await expect(page.getByText('Something went wrong', { exact: true })).toHaveCount(0);
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path: resolve(output, name + '.png'), fullPage: true });
  }
  if (mode === 'dev') {
    await page.goto(origin + '/admin/component-preview.html');
    await expect(page.locator('.recharts-line-curve')).toHaveCount(2);
    await expect.poll(() => hmrConnected, { message: 'Vite HMR connects' }).toBe(true);
  }
  // The existing public login entry must also survive a direct visit and reload.
  await context.clearCookies();
  await page.goto(origin + '/admin/');
  await expect(page).toHaveURL(/\/login$/);
  if (mode === 'dev') {
    await page.reload();
    await expect(page.locator('form')).toBeVisible();
  }
  expect(report.errors).toEqual([]);
  expect(bad, 'all assets 200; no root /assets requests').toEqual([]);
  expect(report.network.some(r => new URL(r.url).pathname.startsWith('/admin/'))).toBeTruthy();
  console.log(`PASS ${mode}: ${report.assets.length} asset checks; ${report.network.length} browser asset responses; no root /assets requests`);
} finally {
  await writeFile(resolve(output, 'report.json'), JSON.stringify(report, null, 2) + '\n');
  await browser.close();
}
