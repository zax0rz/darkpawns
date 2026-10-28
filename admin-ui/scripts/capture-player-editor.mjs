// Capture the v0.1.7 player editor from the disposable admin preview server.
// Start it with ADMIN_PREVIEW=1 go test -tags adminpreview -run '^TestConsolePreview$' -timeout 0 ./pkg/admin
// and run Vite on port 5187 before invoking this script.
import { chromium, expect } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

const ui = process.env.ADMIN_UI_URL || 'http://127.0.0.1:5187';
const api = process.env.ADMIN_PREVIEW_URL || 'http://127.0.0.1:4350';
const repo = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const out = resolve(repo, 'website-astro/src/content/docs/server/images');
await mkdir(out, { recursive: true });
const blurActiveElement = (page) => page.evaluate(() => {
  if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
});

const browser = await chromium.launch({ executablePath: process.env.CHROMIUM_PATH || undefined });
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1600 }, colorScheme: 'light' });
  await context.request.get(`${api}/preview-session`);
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));

  await page.goto(`${ui}/admin/staff/players`);
  await expect(page.locator('main').first()).toBeVisible();
  await page.waitForLoadState('networkidle');
  await page.getByRole('button', { name: 'Returner', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Returner', exact: true })).toBeVisible();
  await page.waitForLoadState('networkidle');

  await page.getByRole('tab', { name: 'Flags' }).click();
  await expect(page.getByRole('tab', { name: 'Flags' })).toHaveAttribute('data-state', 'active');
  await expect(page.getByRole('switch', { name: 'Brief descriptions' })).toBeVisible();
  await blurActiveElement(page);
  await page.waitForTimeout(250);
  await page.screenshot({ path: resolve(out, 'player-management-flags.png'), fullPage: true });

  await page.getByRole('tab', { name: 'Identity' }).click();
  await expect(page.getByRole('tab', { name: 'Identity' })).toHaveAttribute('data-state', 'active');
  await page.getByLabel('New name').fill('Newname');
  await blurActiveElement(page);
  await page.waitForTimeout(250);
  await page.screenshot({ path: resolve(out, 'player-management-identity.png'), fullPage: true });
  await page.getByRole('button', { name: 'Delete character…' }).click();
  await expect(page.getByRole('alertdialog')).toBeVisible();
  const deleteButton = page.getByRole('button', { name: 'Delete character', exact: true });
  await page.getByLabel('Type Returner').fill('returner');
  await expect(deleteButton).toBeDisabled();
  await page.getByLabel('Type Returner').fill('Returner');
  await expect(deleteButton).toBeEnabled();
  await expect(deleteButton).toHaveCSS('opacity', '1');
  await blurActiveElement(page);
  await page.screenshot({ path: resolve(out, 'player-management-delete-confirm.png'), fullPage: true });

  expect(errors).toEqual([]);
  console.log(`Saved player editor screenshots to ${out}`);
} finally {
  await browser.close();
}
