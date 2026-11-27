import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// Serwer deweloperski przekazuje /api i /s do działającego skrytka.exe (port HTTP z konfiguracji).
export default defineConfig({
  plugins: [svelte()],
  build: { outDir: 'dist', emptyOutDir: true, assetsInlineLimit: 0 },
  server: { proxy: { '/api': 'http://127.0.0.1:8080', '/s/': 'http://127.0.0.1:8080' } },
})
