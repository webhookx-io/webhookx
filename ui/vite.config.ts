import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { openApiSpec } from './openapi-plugin'

export default defineConfig({
  plugins: [react(), tailwindcss(), openApiSpec()],
  resolve: { alias: { '@': new URL('./src', import.meta.url).pathname } },
  build: {
    // Keep framework and UI primitives stable across route-level deployments.
    // Swagger remains lazy-loaded with the OpenAPI route.
    chunkSizeWarningLimit: 1_400,
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'query-vendor': ['@tanstack/react-query'],
          'ui-vendor': ['radix-ui', 'class-variance-authority', 'sonner'],
        },
      },
    },
  },
  server: {
    port: 5173,
    host: 'localhost',
    strictPort: true,
    proxy: {
      '/config': {
        target: 'http://localhost:9605',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:9601',
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
