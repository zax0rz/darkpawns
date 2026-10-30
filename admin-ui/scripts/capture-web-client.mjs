import { chromium, expect } from '@playwright/test';
import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromiumPath, crop, highlight } from './media-capture.mjs';

const root = resolve(fileURLToPath(new URL('../..', import.meta.url)));
const output = resolve(root, 'docs/images/generated/web-client-first-steps.png');
const base = process.env.MEDIA_CAPTURE_SERVER_URL || 'http://127.0.0.1:4360';
const browser = await chromium.launch({ executablePath: await chromiumPath(), args: ['--deterministic-mode', '--disable-gpu', '--disable-skia-runtime-opts', '--force-color-profile=srgb'] });
try {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1200 }, deviceScaleFactor: 1, colorScheme: 'light', reducedMotion: 'reduce' });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  await page.goto(base, { waitUntil: 'networkidle' });
  await expect(page.locator('#status-text')).toHaveText('This server is running');
  await page.getByRole('link', { name: 'Play in this browser' }).click();
  const terminalInput = page.locator('#terminal textarea').first();
  await terminalInput.focus();
  await page.keyboard.type('Guest', { delay: 120 });
  await page.keyboard.press('Enter');
  await expect(page.locator('.xterm-rows')).toContainText('Welcome to Dark Pawns!', { timeout: 15000 });
  await expect(page.locator('.xterm-rows')).toContainText('At the Temple Altar', { timeout: 15000 });
  const target = page.locator('.terminal-frame');
  await expect(target).toBeVisible();
  await page.evaluate(() => document.fonts.ready);
  // The anchor starts a smooth scroll. Settle it before measuring the crop;
  // otherwise screenshot height depends on which animation frame we sample.
  await target.evaluate(element => element.scrollIntoView({ block: 'center', behavior: 'instant' }));
  await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
  await highlight(page, target);
  const clip = await crop(page, target, 28);
  await page.screenshot({ path: output, clip, animations: 'disabled' });
  expect(errors).toEqual([]);
} finally {
  await browser.close();
}
