import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

// The Go API is normally on 4350. ADMIN_API_TARGET points the dev server at a
// fixture on another port, so two agents can each run a console against their
// own worktree without sharing cookies or state.
const api = process.env.ADMIN_API_TARGET || 'http://localhost:4350';

export default defineConfig({
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  plugins: [react(), tailwindcss()],
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
          if (req.method === 'GET' && req.headers.accept?.includes('text/html')) {
            return req.url || '/admin/';
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
