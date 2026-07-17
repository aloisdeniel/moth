import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'

// The SDK is aliased to its TypeScript source so editing sdk/react/src
// hot-reloads straight into the example (the file:.. dependency alone would
// serve the built dist and need a rebuild per change).
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@moth/react': fileURLToPath(new URL('../src/index.ts', import.meta.url)),
    },
  },
  server: {
    fs: { allow: ['..'] },
  },
})
