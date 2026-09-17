# Vendored fonts

DM Serif Display, Source Serif 4 and JetBrains Mono, latin subsets, all under
the SIL Open Font License 1.1, which permits redistribution.

They ship with the console rather than being fetched from Google Fonts because
`web/security.go` sets `style-src 'self'` and `font-src 'self'`. A remote font
link is blocked before it starts: served from the Go binary this console loaded
**zero** font faces and rendered entirely in fallbacks, while the vite dev
server — which sets no CSP — showed the intended faces. The two surfaces
disagreed, and only the dev one was ever looked at.

`index.css` declares them with relative urls so vite fingerprints them into
`dist/assets/`, which the server already serves. No new route, and dev and
production load the same bytes.

DM Serif Display is also vendored for the self-hosted front door at
`web/public/static/fonts/`; the two copies are the same file.
