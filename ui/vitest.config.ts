import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitest/config'
import { openApiSpec } from './openapi-plugin'

export default defineConfig({
  plugins: [openApiSpec()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
})
