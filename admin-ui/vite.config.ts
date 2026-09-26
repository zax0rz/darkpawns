import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/admin': {
        target: 'http://localhost:4350',
        changeOrigin: true,
        // Browser navigation belongs to Vite's SPA fallback. Fetch requests
        // for /admin/* still reach the Go API on port 4350.
        bypass(req) {
          if (req.method === 'GET' && req.headers.accept?.includes('text/html')) {
            return req.url || '/admin/';
          }
        },
      },
      '/ws': {
        target: 'ws://localhost:4350',
        ws: true,
      },
    },
  },
});
