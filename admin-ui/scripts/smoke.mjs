// Real-handler fixture: ADMIN_PREVIEW=1 go test -tags adminpreview ./pkg/admin
// -run '^TestConsolePreview$' -timeout 0. Run Vite separately on port 5187.
// Override ADMIN_UI_URL to capture an unchanged baseline on a different port.
import { chromium, expect } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const base = process.env.ADMIN_UI_URL || 'http://127.0.0.1:5187';
// The fixture server is separate from the UI origin in the Vite workflow, so it
// has its own override. Serving the built SPA from the fixture (one origin) works
// too: point both variables at the same address.
const fixture = process.env.ADMIN_PREVIEW_URL || 'http://127.0.0.1:4350';
const out = resolve(process.env.ADMIN_EVIDENCE_DIR || '../.local/admin-evidence/after');
await mkdir(out, { recursive: true });
const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, colorScheme: 'light' });
  await context.request.get(fixture + '/preview-session');
  const page = await context.newPage();
  const errors = []; page.on('pageerror', error => errors.push(error.message));
  const routes = [
    ['dashboard', '/admin/'], ['zones', '/admin/game/zones'], ['zone-detail', '/admin/game/zones/1'], ['zone-editor', '/admin/game/zones/1/edit'],
    ['mobs', '/admin/game/mobs'], ['mob-detail', '/admin/game/mobs/2001'], ['mob-editor', '/admin/game/mobs/2001/edit'],
    ['objects', '/admin/game/objects'], ['object-detail', '/admin/game/objects/3001'], ['object-editor', '/admin/game/objects/3001/edit'],
    ['room-detail', '/admin/game/rooms/1001'], ['room-editor', '/admin/game/rooms/1001/edit'],
    ['shops', '/admin/game/shops'], ['shop-editor', '/admin/game/shops/2001/edit'],
    ['players', '/admin/staff/players'], ['notifications', '/admin/staff/notifications'], ['operations', '/admin/operations'],
    ['workshop', '/admin/workshop'], ['builder-guide', '/admin/workshop/help'],
    ['scripts', '/admin/workshop/scripts?file=guard.lua'], ['text-files', '/admin/workshop/text?file=news'], ['terminal', '/admin/webclient'],
  ];
  const report = [];
  for (const [name, path] of routes) {
    expect((await context.request.get(fixture + '/preview-reset')).ok()).toBeTruthy();
    await page.goto(base + path);
    await expect(page.locator('main').first()).toBeVisible();
    await expect(page.locator('main').first()).not.toBeEmpty();
    // Finish API queries and font loading before retaining visual evidence.
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => document.fonts.ready);
    if (name === 'dashboard') await expect(page.locator('main .animate-pulse')).toHaveCount(0, {timeout:10000});
    if (name === 'players') { await page.getByRole('button', { name: 'Returner', exact: true }).click(); await expect(page.getByRole('heading', {name:'Returner',exact:true})).toBeVisible(); }
    await expect(page.getByText('Something went wrong', { exact: true })).toHaveCount(0);
    expect(errors.splice(0), name + ' runtime errors').toEqual([]);
    await page.screenshot({ path: resolve(out, name + '.png'), fullPage: true });
    report.push({ name, path, headings: await page.locator('main h1, main h2').allTextContents(), alerts: await page.locator('[role=alert]').allTextContents() });
  }
  const publicContext = await browser.newContext({ viewport: { width: 1440, height: 1000 }, colorScheme: 'light' });
  const publicPage = await publicContext.newPage();
  for (const [name, path] of [['login','/login'],['not-found','/admin/missing-page']]) { await publicPage.goto(base+path); await expect(publicPage.locator('#root')).not.toBeEmpty(); await publicPage.evaluate(()=>document.fonts.ready); await publicPage.screenshot({path:resolve(out,name+'.png'),fullPage:true}); }
  await page.setViewportSize({width:390,height:844});
  for (const [name,path] of routes.filter(([name])=>['dashboard','players','notifications','mobs'].includes(name))) { await context.request.get(fixture + '/preview-reset'); await page.goto(base+path); await expect(page.locator('main').first()).not.toBeEmpty(); await page.waitForLoadState('networkidle'); await page.screenshot({path:resolve(out,name+'-mobile.png'),fullPage:true}); }
  if (process.env.ADMIN_COMPONENT_PREVIEW === '1') {
    await page.setViewportSize({width:1440,height:1000});
    await page.goto(base+'/admin/component-preview.html');
    await expect(page.locator('.recharts-line-curve')).toHaveCount(2);
    await page.screenshot({path:resolve(out,'chart-foundation.png'),fullPage:true});
    await page.setViewportSize({width:2123,height:790});
    await page.goto(base+'/admin/'); await page.waitForLoadState('networkidle');
    await page.screenshot({path:resolve(out,'dashboard-user-2123.png'),fullPage:true});
  }
  await writeFile(resolve(out,'report.json'),JSON.stringify(report,null,2));
  console.log(`PASS: ${routes.length+2} routed pages; 4 mobile captures. Synthetic local world, real API handlers.`);
} finally { await browser.close(); }
