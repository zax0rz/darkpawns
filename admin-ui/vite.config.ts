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
      },
      '/ws': {
        target: 'ws://localhost:4350',
        ws: true,
      },
    },
  },
});
