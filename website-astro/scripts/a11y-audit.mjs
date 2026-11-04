// A manual accessibility and layout probe for the built site, run by hand
// (not wired into the build). It checks the things the SEO probe cannot:
// colour contrast, landmark and heading structure, image alt text, the
// accessible name of every control, touch-target sizes, and a local lab
// proxy for Largest Contentful Paint and Cumulative Layout Shift.
//
//   npm run build
//   npx playwright install chromium   # once, if the browser is not cached
//   npx astro preview                 # serves ./dist on http://localhost:4321
//   node scripts/a11y-audit.mjs http://localhost:4321
//
// Playwright is already a dependency of this package (the build's Mermaid
// renderer uses it too), so only the Chromium binary may need installing.
// Optional extra arguments are paths to probe instead of the default set.
//
// Contrast is measured on the desktop pass; touch targets on a 390px
// coarse-pointer pass, so the site's own mobile rules are the ones in play.
// Screen-reader-only text and inputs named by an associated <label> are not
// defects. Inline links are exempt from the touch-target check, as WCAG 2.2
// target-size-minimum exempts them.
import { chromium } from 'playwright';

const [rawBase, ...argPaths] = process.argv.slice(2);
const base = (rawBase ?? 'http://localhost:4321').replace(/\/$/, '');
const paths = argPaths.length ? argPaths : [
  '/', '/play/', '/world/', '/map/', '/database/', '/archive/', '/archive/logs/',
  '/blog/', '/docs/', '/docs/host/install/', '/about/', '/connect/', '/download/',
  '/credits/', '/privacy/', '/contact/', '/zones/',
];

const desktopAudit = () => {
  const parse = (value) => {
    const match = value.match(/rgba?\(([^)]+)\)/);
    if (!match) return null;
    const [r, g, b, a = '1'] = match[1].split(/[,/]/).map((part) => parseFloat(part.trim()));
    return { r, g, b, a: Number.isNaN(a) ? 1 : a };
  };
  const blend = (fg, bg) => ({
    r: fg.r * fg.a + bg.r * (1 - fg.a),
    g: fg.g * fg.a + bg.g * (1 - fg.a),
    b: fg.b * fg.a + bg.b * (1 - fg.a),
    a: 1,
  });
  const luminance = ({ r, g, b }) => {
    const channel = (c) => {
      const s = c / 255;
      return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
    };
    return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b);
  };
  const ratio = (a, b) => {
    const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
    return (hi + 0.05) / (lo + 0.05);
  };
  const backgroundOf = (el) => {
    let node = el;
    let acc = null;
    while (node && node !== document.documentElement) {
      const bg = parse(getComputedStyle(node).backgroundColor);
      if (bg && bg.a > 0) acc = acc ? blend(acc, bg) : bg;
      if (acc && acc.a === 1) break;
      node = node.parentElement;
    }
    return acc ?? { r: 255, g: 255, b: 255, a: 1 };
  };
  const srOnly = (el) => !!el.closest('.visually-hidden, .sr-only');

  const contrast = [];
  const selectors = 'main p, main li, main h1, main h2, main h3, main a, main td, main th, main span, main dd, main dt, main time, main small, main code';
  for (const el of document.querySelectorAll(selectors)) {
    const text = (el.textContent ?? '').trim();
    if (!text || el.children.length > 0 || srOnly(el)) continue;
    const style = getComputedStyle(el);
    if (style.visibility === 'hidden' || style.display === 'none') continue;
    if (!el.getClientRects().length) continue;
    const fg = parse(style.color);
    if (!fg) continue;
    const bg = backgroundOf(el);
    const value = ratio(fg.a === 1 ? fg : blend(fg, bg), bg);
    const size = parseFloat(style.fontSize);
    const bold = parseInt(style.fontWeight, 10) >= 700;
    const large = size >= 24 || (size >= 18.66 && bold);
    const min = large ? 3 : 4.5;
    if (value < min) contrast.push({ text: text.slice(0, 40), ratio: Math.round(value * 100) / 100, min });
  }

  const labelledBy = (el) => (el.getAttribute('aria-labelledby') ?? '').split(/\s+/).filter(Boolean)
    .map((id) => document.getElementById(id)?.textContent ?? '').join(' ');
  const nameOf = (el) => [
    el.getAttribute('aria-label'),
    labelledBy(el),
    [...(el.labels ?? [])].map((label) => label.textContent).join(' '),
    el.textContent,
    el.getAttribute('title'),
    el.getAttribute('alt'),
  ].map((part) => (part ?? '').trim()).find(Boolean) ?? '';
  const unnamed = [...document.querySelectorAll('a, button, input, select, textarea')]
    .filter((el) => !el.closest('[aria-hidden="true"]') && !nameOf(el)).length;

  const lcp = window.__lcp || null;
  const cls = window.__cls ?? 0;

  return {
    landmarks: {
      header: document.querySelectorAll('header').length,
      nav: document.querySelectorAll('nav').length,
      main: document.querySelectorAll('main').length,
      footer: document.querySelectorAll('footer').length,
    },
    h1: document.querySelectorAll('h1').length,
    images: [...document.querySelectorAll('img')].filter((img) => !img.hasAttribute('alt')).length,
    unnamed,
    contrast,
    lcp,
    cls,
  };
};

const touchAudit = () => [...document.querySelectorAll('button, select, input:not([type="hidden"]), .btn-primary, .btn-secondary, nav a, footer a')]
  .filter((el) => {
    const box = el.getBoundingClientRect();
    if (!box.width || !box.height) return false;
    return box.width < 24 || box.height < 24;
  }).length;

const initMetrics = () => {
  window.__cls = 0;
  window.__lcp = 0;
  try {
    new PerformanceObserver((list) => {
      for (const entry of list.getEntries()) if (!entry.hadRecentInput) window.__cls += entry.value;
    }).observe({ type: 'layout-shift', buffered: true });
  } catch { /* layout-shift unsupported */ }
  try {
    new PerformanceObserver((list) => {
      window.__lcp = list.getEntries().at(-1).startTime;
    }).observe({ type: 'largest-contentful-paint', buffered: true });
  } catch { /* LCP unsupported */ }
};

const browser = await chromium.launch();
const desktop = await browser.newContext({ viewport: { width: 1280, height: 900 } });
await desktop.addInitScript(initMetrics);
const page = await desktop.newPage();
const mobile = await browser.newContext({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });
const mobilePage = await mobile.newPage();

let failures = 0;
console.log('path\tstatus\th1\tlandmarks(h/n/main/f)\timg-no-alt\tunnamed\t<24px\tcontrast\tLCP(ms)\tCLS');
for (const path of paths) {
  const response = await page.goto(base + path, { waitUntil: 'load' });
  const status = response?.status() ?? 0;
  if (status >= 400) { console.log(`${path}\t${status}\t-`); continue; }
  await page.waitForTimeout(250);
  const r = await page.evaluate(desktopAudit);
  await mobilePage.goto(base + path, { waitUntil: 'load' });
  const small = await mobilePage.evaluate(touchAudit);
  const hard = r.landmarks.main !== 1 || r.h1 !== 1 || r.images > 0 || r.unnamed > 0 || r.contrast.length > 0;
  if (hard) failures++;
  console.log([
    path, status, r.h1,
    `${r.landmarks.header}/${r.landmarks.nav}/${r.landmarks.main}/${r.landmarks.footer}`,
    r.images, r.unnamed, small, r.contrast.length,
    r.lcp === null ? '-' : Math.round(r.lcp),
    Math.round(r.cls * 1000) / 1000,
  ].join('\t'));
  for (const c of r.contrast.slice(0, 4)) console.log(`\tcontrast ${c.ratio} (< ${c.min}): "${c.text}"`);
}
await browser.close();
console.log(failures ? `\n${failures} page(s) with a hard failure` : '\nall pages pass the hard checks');
process.exit(failures ? 1 : 0);
