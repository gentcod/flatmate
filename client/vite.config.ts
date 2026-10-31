import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      // Same-origin cookies: the dev server forwards API calls to Go,
      // so session + CSRF cookies behave exactly like production.
      '/api': 'http://localhost:8080',
    },
  },
})
