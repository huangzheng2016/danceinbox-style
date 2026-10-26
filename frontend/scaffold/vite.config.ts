import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) } },
  // ECharts is one ~600 kB chunk, loaded only by pages that draw a chart
  build: { chunkSizeWarningLimit: 700 },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8080',
      '/papi': 'http://127.0.0.1:8080',
    },
  },
})
