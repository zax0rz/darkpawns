# Eyeballs pass A: landing and console quick fixes

2026-10-28. Screenshots use Chrome through the installed extension. Desktop
uses the existing 2560 × 800 viewport; mobile uses 390 × 844. The temporary
mobile override was reset afterward.

## Preview data

All screenshots use the opt-in `adminpreview` test fixture. Its database,
credentials and world are disposable. Log entries and online players are
explicitly synthetic. No production server or player records were used.

The fixture serves the actual built admin bundle and landing files.
`ADMIN_PREVIEW_SETUP_STATE=fresh` supplies the empty-world response for fresh
landing captures. Running captures use the fixture database's saved characters.
The fixture has no game WebSocket listener, so its terminal says disconnected;
these captures demonstrate layout and focus, not a gameplay connection test.

## Screenshots

| Surface | Desktop | Mobile |
| --- | --- | --- |
| Fresh landing | [desktop](landing-fresh-desktop.png) | [mobile](landing-fresh-mobile.png) |
| Running landing | [desktop](landing-running-desktop.png) | [mobile](landing-running-mobile.png) |
| Heartbeat explanation | [desktop](notifications-desktop.png) | [mobile](notifications-mobile.png) |
| Checkout update guidance | [desktop](updates-desktop.png) | [mobile](updates-mobile.png) |
| Server log | [desktop](logs-server-desktop.png) | [mobile](logs-server-mobile.png) |
| Mudlog | [desktop](logs-mudlog-desktop.png) | [mobile](logs-mudlog-mobile.png) |

## Checks

- Chrome: fresh warning names level 40, ownership and restart; running world
  hides it. Both primary links are above the fold on mobile.
- Chrome: Play scrolls to the terminal and focuses Terminal input at both sizes.
- Chrome: Healthchecks link and explanation render; the release announcement
  card is absent. No notification settings were changed or tests sent.
- Chrome: checkout update explanation renders with a guide link and no update
  action; both log sources show their own explanation.
- Go setup endpoint tests exercise zero, one and several saved characters,
  both startup states, no-store caching, OpenAPI registration, and unavailable
  storage without exposing storage details.
- Node tests cover fresh → restart → running state and malformed/unavailable
  data; the existing terminal protocol tests pass.
- Go build, vet, all tests and golangci-lint pass, run separately. Formatting
  and the deadcode ratchet pass. Admin build, ESLint and type generation pass.
- Site checks pass: voice lint, voice and Markdown tests, content inventory,
  wordmark, palette regression tests, Astro build and redirect generation.

## Design and scope

Impeccable's scoped polish pass preserved Haunted Paperback. Its cream-palette
warning is intentional under the site's design system. Its terminal-header
contrast warning was fixed using the terminal's existing light ink color.
Copy follows brand-voice §8.5.

The site palette gate initially mistook the existing `outline-offset-4`
focus-ring spacing for a color. The checker now recognizes outline offsets
as spacing; regression tests still reject unapproved outline, background
and text colors.

The new read-only Huma operation exposes two booleans, with no names or counts.
It observes the existing first-character safeguard without changing any binds
or listener behavior. The deployment guide explains the existing restart step.
No game bytes, RNG draws, creation, death or login behavior changed (R1/R3).
No oracle census is needed for this presentation and read-only status change.
Dashboard recording and equipment work remain B/C; network/ports stay with GLM.
