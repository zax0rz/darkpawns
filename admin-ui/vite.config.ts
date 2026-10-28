import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// The Go API is normally on 4350. ADMIN_API_TARGET points the dev server at a
// fixture on another port, so two agents can each run a console against their
// own worktree without sharing cookies or state.
const base = '/admin/';
const api = process.env.ADMIN_API_TARGET || 'http://localhost:4350';

export default defineConfig({
  base,
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  plugins: [react(), tailwindcss(), {
    name: 'admin-login-entry',
    configureServer(server) {
      // /login is also a public entry in the Go server. Keep direct visits and
      // refreshes working in development, outside Vite's asset base.
      server.middlewares.use((req, _res, next) => {
        if (req.method === 'GET' && req.url?.split('?')[0] === '/login' &&
          req.headers.accept?.includes('text/html')) {
          req.url = base + req.url.slice('/login'.length);
        }
        next();
      });
    },
  }],
  server: {
    port: 5173,
    proxy: {
      '/health': { target: api, changeOrigin: true },
      '/admin': {
        target: api,
        changeOrigin: true,
        // Browser navigation belongs to Vite's SPA fallback. Fetch requests
        // for /admin/* still reach the Go API on the configured target.
        bypass(req) {
          const path = req.url?.split('?')[0].slice(base.length) || '';
          // Vite's modules and public files share the console's base path.
          // Leave those on Vite; only API fetches belong on the Go server.
          const viteFile = path.startsWith('@') || path.startsWith('src/') ||
            path.startsWith('node_modules/') || path === 'favicon.svg' || path === 'icons.svg';
          if (req.method === 'GET' && (viteFile || req.headers.accept?.includes('text/html'))) {
            return req.url || base;
          }
        },
      },
      '/ws': {
        target: api.replace(/^http/, 'ws'),
        ws: true,
      },
      // The preview fixture's session and reset helpers, so a browser hitting the
      // dev server gets them from the same origin as the console.
      '/preview-session': { target: api, changeOrigin: true },
      '/preview-reset': { target: api, changeOrigin: true },
    },
  },
});
