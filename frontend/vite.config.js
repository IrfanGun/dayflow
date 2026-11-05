import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

const port = Number(process.env.WAILS_VITE_PORT || 34116);

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  build: { outDir: 'dist', emptyOutDir: true },
  server: { host: '127.0.0.1', port, strictPort: false }
});
