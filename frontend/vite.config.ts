import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Dev: API calls proxy to the Go backend so no CORS setup is needed.
      '/api': 'http://localhost:8080',
    },
  },
})
