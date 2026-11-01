import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// The same offline page policy as the packaged app (httpapi/offline_policy.go, ADR-0133):
// nothing loads from the internet. Development adds only Vite's own needs: its inline
// React-refresh preamble and the reload connection to this dev server.
const offlinePolicy = [
  "default-src 'self'", "script-src 'self' 'unsafe-inline'", "style-src 'self' 'unsafe-inline'",
  "img-src 'self' data: blob:", "font-src 'self' data:", "connect-src 'self' ws://127.0.0.1:* ws://localhost:*",
  "object-src 'none'", "frame-src 'none'", "worker-src 'self' blob:", "base-uri 'self'",
  "form-action 'self'", "frame-ancestors 'none'",
].join('; ');

export default defineConfig({
  plugins: [react()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: {
    proxy: { '/api': 'http://127.0.0.1:8765' }, strictPort: true,
    headers: { 'Content-Security-Policy': offlinePolicy },
  },
});
