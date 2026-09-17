# Vendored third-party code

The splash page is served straight off disk by the Go binary with no build
step, so xterm.js and its fit addon are committed here as built files rather
than installed. They are byte-for-byte the published npm artifacts.

| file | package | version | licence |
|------|---------|---------|---------|
| `../xterm.js` | `@xterm/xterm` | 5.5.0 | MIT — `xterm.LICENSE` |
| `../xterm.css` | `@xterm/xterm` | 5.5.0 | MIT — `xterm.LICENSE` |
| `../addon-fit.js` | `@xterm/addon-fit` | 0.10.0 | MIT — `addon-fit.LICENSE` |

The MIT licence requires its copyright and permission notice to travel with
every copy of the software. Minification strips comments, so the notice does
not survive inside `xterm.js` or `addon-fit.js` — it is preserved here instead,
and these files are served alongside them.

To update, copy the built files out of `website-astro/node_modules` again and
bring the matching `LICENSE` with them:

    cp node_modules/@xterm/xterm/lib/xterm.js        ../web/public/xterm.js
    cp node_modules/@xterm/xterm/css/xterm.css       ../web/public/xterm.css
    cp node_modules/@xterm/addon-fit/lib/addon-fit.js ../web/public/addon-fit.js
    cp node_modules/@xterm/xterm/LICENSE             ../web/public/vendor/xterm.LICENSE
    cp node_modules/@xterm/addon-fit/LICENSE         ../web/public/vendor/addon-fit.LICENSE
