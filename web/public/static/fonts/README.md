# Vendored fonts

`dm-serif-display-latin-400.woff2` is DM Serif Display (Colophon Foundry),
licensed under the SIL Open Font License 1.1, which permits redistribution.

It ships with the server rather than being fetched from Google Fonts because
`web/security.go` sets `font-src 'self'`: a remote font request is blocked
before it starts. Vendoring also keeps a self-hosted install working with no
outbound network at render time.

Source: https://fonts.google.com/specimen/DM+Serif+Display
