// Canonical README and operator-doc captures. Every descriptor owns its alt
// text and source list so the manifest can warn when the UI drifts.
import { chromium, expect } from '@playwright/test';
import { access, mkdir, readdir, writeFile } from 'node:fs/promises';
import { homedir } from 'node:os';
import { tmpdir } from 'node:os';
import { dirname, resolve } from 'node:path';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import { mkdtemp, rm } from 'node:fs/promises';

const repo = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
const out = resolve(repo, 'docs/images/generated');
const ui = process.env.ADMIN_UI_URL || 'http://127.0.0.1:5187';
const release = process.env.MEDIA_RELEASE || 'unreleased';
const oxblood = '#A8201A'; // website-astro/DESIGN.md colors.oxblood
await mkdir(out, { recursive: true });

export async function chromiumPath() {
  if (process.env.CHROMIUM_PATH) return process.env.CHROMIUM_PATH;
  const bundled = chromium.executablePath();
  try { await access(bundled); return bundled; } catch { /* cached browser may predate lockfile */ }
  const cache = resolve(homedir(), '.cache/ms-playwright');
  const candidates = (await readdir(cache)).filter(name => /^chromium-\d+$/.test(name)).sort((a, b) => Number(b.slice(9)) - Number(a.slice(9)));
  for (const name of candidates) {
    const candidate = resolve(cache, name, 'chrome-linux64/chrome');
    try { await access(candidate); return candidate; } catch { /* try older cache */ }
  }
  return undefined;
}

export async function crop(page, locator, padding = 24, maxHeight) {
  const box = await locator.boundingBox();
  if (!box) throw new Error('capture crop target is not visible');
  const viewport = page.viewportSize();
  const x = Math.max(0, Math.floor(box.x - padding));
  const y = Math.max(0, Math.floor(box.y - padding));
  const right = Math.min(viewport.width, Math.ceil(box.x + box.width + padding));
  const bottom = Math.min(viewport.height, maxHeight ? y + maxHeight : Math.ceil(box.y + box.height + padding));
  return { x, y, width: right - x, height: bottom - y };
}

export async function highlight(page, locator, inset = -3) {
  await locator.evaluate((element, { accent, inset }) => {
    document.querySelector('[data-media-ring]')?.remove();
    const rect = element.getBoundingClientRect();
    const ring = document.createElement('div');
    ring.dataset.mediaRing = '';
    Object.assign(ring.style, { position: 'fixed', left: `${rect.left + inset}px`, top: `${rect.top + inset}px`, width: `${rect.width - 2 * inset}px`, height: `${rect.height - 2 * inset}px`, outline: `3px solid ${accent}`, boxShadow: '0 0 0 100vmax rgba(26,22,20,.22)', zIndex: '2147483646', pointerEvents: 'none', boxSizing: 'border-box' });
    document.body.append(ring);
  }, { accent: oxblood, inset });
}

export async function badge(page, locator, number) {
  await locator.evaluate((element, args) => {
    const rect = element.getBoundingClientRect();
    const mark = document.createElement('span');
    mark.textContent = String(args.number);
    Object.assign(mark.style, { position: 'fixed', left: `${Math.max(4, rect.left - 15)}px`, top: `${Math.max(4, rect.top - 15)}px`, zIndex: '2147483647', width: '28px', height: '28px', display: 'grid', placeItems: 'center', borderRadius: '50%', background: args.accent, color: '#EFE7D6', border: '2px solid #1A1614', font: 'bold 15px Georgia,serif', boxSizing: 'border-box' });
    document.body.append(mark);
  }, { number, accent: oxblood });
}

export async function gif(page, output, steps, target) {
  const frames = await mkdtemp(resolve(tmpdir(), 'dp-media-frames-'));
  try {
    for (let index = 0; index < steps.length; index += 1) {
      await steps[index]();
      await page.mouse.move(0, 0);
      await page.waitForTimeout(100);
      const clip = await crop(page, target, 24);
      await page.screenshot({ path: resolve(frames, `frame-${String(index).padStart(3, '0')}.png`), clip, animations: 'disabled' });
    }
    const first = resolve(frames, 'frame-000.png');
    const dimensions = spawnSync('ffprobe', ['-v', 'error', '-select_streams', 'v:0', '-show_entries', 'stream=width,height', '-of', 'csv=s=x:p=0', first], { encoding: 'utf8' });
    if (dimensions.status !== 0) throw new Error(`ffprobe image dimensions failed: ${dimensions.stderr}`);
    const [width, height] = dimensions.stdout.trim().split('x');
    const palette = resolve(frames, 'palette.png');
    const sourcePattern = resolve(frames, 'frame-%03d.png');
    const zoomPattern = resolve(frames, 'zoom-%03d.png');
    const zoom = `zoompan=z='if(lte(on,5),1+on*0.03,if(lte(on,11),1.15,max(1,1.15-(on-11)*0.03)))':d=6:s=${width}x${height}:fps=6`;
    const focused = spawnSync('ffmpeg', ['-v', 'error', '-framerate', '1', '-i', sourcePattern, '-vf', zoom, '-y', zoomPattern], { encoding: 'utf8' });
    if (focused.status !== 0) throw new Error(`ffmpeg zoom failed: ${focused.stderr}`);
    const makePalette = spawnSync('ffmpeg', ['-v', 'error', '-framerate', '6', '-i', zoomPattern, '-vf', 'palettegen=stats_mode=full', '-y', palette], { encoding: 'utf8' });
    if (makePalette.status !== 0) throw new Error(`ffmpeg palette failed: ${makePalette.stderr}`);
    const encoded = spawnSync('ffmpeg', ['-v', 'error', '-framerate', '6', '-i', zoomPattern, '-i', palette, '-lavfi', 'paletteuse=dither=none', '-map_metadata', '-1', '-loop', '0', '-y', output], { encoding: 'utf8' });
    if (encoded.status !== 0) throw new Error(`ffmpeg GIF failed: ${encoded.stderr}`);
  } finally {
    await rm(frames, { recursive: true, force: true });
  }
}

async function captureAdmin() {
const captures = [
  { padding: -2, highlightInset: 6, output: 'admin-dashboard.png', alt: 'The dashboard charts show synthetic players online and daily arrivals.', route: '/admin/', target: 'main > div > div:nth-of-type(4)', sources: ['admin-ui/src/pages/DashboardPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-player-editor.png', alt: 'The player editor showing a synthetic character’s level, vitals and attributes.', route: '/admin/staff/players', action: async page => { const row = page.getByRole('button', { name: 'Returner', exact: true }); if (await row.count() && await row.getAttribute('aria-pressed') !== 'true') await row.click(); await expect(page.getByRole('heading', { name: 'Returner', exact: true })).toBeVisible(); }, target: 'section[aria-label="Character editor"]', sources: ['admin-ui/src/pages/PlayersPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-room-editor.png', alt: 'The room editor for a synthetic test room, with its description and room flags.', route: '/admin/game/rooms/1001/edit', scrollTarget: true, releaseDraft: true, target: 'section:has(> h2:text-is("Room fields"))', sources: ['admin-ui/src/pages/RoomEditorPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-object-editor.png', alt: 'Object fields for a synthetic steel sword, with Item type highlighted.', route: '/admin/game/objects/3001/edit', releaseDraft: true, cropTarget: 'section:has(> h2:text-is("Object fields"))', cropHeight: 620, target: 'section:has(> h2:text-is("Object fields")) > div.grid > div:nth-child(5)', sources: ['admin-ui/src/pages/EntityEditorPage.tsx', 'admin-ui/src/components/olc/EntitySchemaFields.tsx', 'admin-ui/src/components/olc/ClaimSaveFrame.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-mob-editor.png', alt: 'Mob fields for a synthetic guard, with its alias highlighted.', route: '/admin/game/mobs/2001/edit', releaseDraft: true, cropTarget: 'section:has(> h2:text-is("Mob fields"))', cropHeight: 620, target: 'section:has(> h2:text-is("Mob fields")) > div.grid > div:nth-child(1)', sources: ['admin-ui/src/pages/EntityEditorPage.tsx', 'admin-ui/src/components/olc/EntitySchemaFields.tsx', 'admin-ui/src/components/olc/ClaimSaveFrame.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-zone-editor.png', alt: 'Zone name, lifespan, reset mode, and top-room controls in the zone editor.', route: '/admin/game/zones/1/edit', releaseDraft: true, target: 'section:has(> h2:text-is("Zone settings"))', sources: ['admin-ui/src/pages/ZoneEditorPage.tsx', 'admin-ui/src/components/olc/ClaimSaveFrame.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-lua-scripts.png', alt: 'The selected guard.lua file with Save, Delete, and the live-apply notice.', route: '/admin/workshop/scripts?file=guard.lua', target: 'section[aria-label="File editor"]', sources: ['admin-ui/src/pages/FileEditorPage.tsx', 'admin-ui/src/components/olc/CodeEditor.tsx', 'admin-ui/src/components/Layout.tsx', 'admin-ui/src/index.css', 'pkg/admin/fileedit.go', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-notifications.png', alt: 'Notification settings with event rows and disabled delivery channels in the local preview.', route: '/admin/staff/notifications', target: 'main', sources: ['admin-ui/src/pages/NotificationsPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-configuration.png', alt: 'The Sessions & limits configuration card in the local checkout preview, with editable login and connection limits.', route: '/admin/staff/settings', target: 'section:has(> header > h2:text-is("Sessions & limits"))', verify: async page => { await expect(page.getByText('Read-only on this server', { exact: true })).toBeVisible(); await expect(page.getByLabel('Login idle timeout (s)')).toBeEnabled(); }, sources: ['admin-ui/src/pages/SettingsPage.tsx', 'admin-ui/src/components/NetworkCard.tsx', 'admin-ui/src/components/ui/input.tsx', 'admin-ui/src/components/Layout.tsx', 'admin-ui/src/index.css', 'pkg/admin/settings.go', 'pkg/admin/console_preview_test.go', 'pkg/settings/settings.go'] },
  { padding: 8, output: 'admin-operations.png', alt: 'The operations log with synthetic warning and error entries from the local preview.', route: '/admin/operations', target: 'main > div > div:nth-of-type(3)', sources: ['admin-ui/src/pages/OperationsPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-zones.png', alt: 'The zones list for the synthetic world used by the admin console preview.', route: '/admin/game/zones', padding: 8, scrollTarget: true, target: 'table', sources: ['admin-ui/src/pages/ZonesPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-updates.png', alt: 'The Updates page showing the available local release controls.', route: '/admin/staff/updates', target: 'main', sources: ['admin-ui/src/pages/UpdatesPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
  { output: 'admin-motd.png', alt: 'The welcome message editor with synthetic login text and its preview.', route: '/admin/staff/motd', target: 'section[aria-label="Message editor"]', sources: ['admin-ui/src/pages/MotdPage.tsx', 'admin-ui/src/components/MotdPreview.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go'] },
];

const browser = await chromium.launch({ executablePath: await chromiumPath(), args: ['--deterministic-mode', '--disable-gpu', '--disable-skia-runtime-opts', '--force-color-profile=srgb'] });
try {
  const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, colorScheme: 'light', deviceScaleFactor: 1, reducedMotion: 'reduce' });
  await context.request.get(`${ui}/preview-session`);
  const reset = await context.request.post(`${ui}/preview-reset`);
  expect(reset.ok()).toBeTruthy();
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  const manifest = [];
  for (const capture of captures) {
    await page.goto(`${ui}${capture.route}`);
    await expect(page.locator('main').first()).toBeVisible();
    await page.waitForLoadState('networkidle');
    await page.evaluate(() => document.fonts.ready);
    await page.evaluate(() => new Promise(resolve => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    if (capture.verify) await capture.verify(page);
    if (capture.action) await capture.action(page);
    const target = page.locator(capture.target).first();
    await expect(target).toBeVisible();
    if (capture.scrollTarget) await target.evaluate(element => element.scrollIntoView({ block: 'center', behavior: 'instant' }));
    await highlight(page, target, capture.highlightInset ?? -3);
    const cropTarget = capture.cropTarget ? page.locator(capture.cropTarget).first() : target;
    const clip = await crop(page, cropTarget, capture.padding ?? 28, capture.cropHeight);
    await page.screenshot({ path: resolve(out, capture.output), clip, animations: 'disabled' });
    await page.locator('[data-media-ring]').evaluate(el => el.remove());
    if (capture.releaseDraft) {
      const closed = page.waitForURL(url => !url.pathname.endsWith('/edit'));
      await page.getByRole('button', { name: 'Discard', exact: true }).click();
      await closed;
      const reset = await context.request.post(`${ui}/preview-reset`);
      expect(reset.ok()).toBeTruthy();
    }
  manifest.push({ path: `docs/images/generated/${capture.output}`, alt: capture.alt, sources: [...capture.sources, 'admin-ui/scripts/media-capture.mjs', 'website-astro/DESIGN.md'], release });
  }
  await page.goto(`${ui}/admin/staff/players`);
  await page.waitForLoadState('networkidle');
  const returner = page.getByRole('button', { name: 'Returner', exact: true });
  if (await returner.count() && await returner.getAttribute('aria-pressed') !== 'true') await returner.click();
  await expect(page.getByRole('heading', { name: 'Returner', exact: true })).toBeVisible();
  const target = page.locator('section[aria-label="Character editor"]');
  await gif(page, resolve(out, 'admin-player-editor.gif'), [
    async () => {},
    async () => { await page.getByRole('tab', { name: 'Equipment & inventory' }).click(); },
    async () => { await page.getByRole('tab', { name: 'Flags' }).click(); },
  ], target);
  manifest.push({ path: 'docs/images/generated/admin-player-editor.gif', alt: 'The player editor moves from character stats to equipment and flags.', sources: ['admin-ui/src/pages/PlayersPage.tsx', 'admin-ui/src/index.css', 'admin-ui/src/components/Layout.tsx', 'pkg/admin/console_preview_test.go', 'admin-ui/scripts/media-capture.mjs', 'website-astro/DESIGN.md'], release });
  await writeFile(resolve(out, 'manifest.json'), `${JSON.stringify({ schema: 1, release, media: manifest }, null, 2)}\n`);
  expect(errors).toEqual([]);
} finally {
  await browser.close();
}
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  await captureAdmin();
}
