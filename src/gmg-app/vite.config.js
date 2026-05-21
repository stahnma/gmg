/// <reference types="vitest" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// This is a legacy Create React App codebase: components live in `.js` files
// but contain JSX, and React 16.6 has no automatic JSX runtime. So we run the
// React Babel transform over `.js`/`.jsx` with the classic runtime, and tell
// esbuild's dependency pre-bundler to treat `.js` as JSX as well.
export default defineConfig({
  plugins: [
    react({ jsxRuntime: 'classic' }),
  ],
  // Dev: @vitejs/plugin-react runs Babel over `.js` (covers JSX). Production:
  // Vite transpiles with esbuild, which only treats `.jsx` as JSX by default,
  // so widen the jsx loader to `.js` under src/. Deps get the same treatment
  // via the pre-bundler.
  esbuild: {
    loader: 'jsx',
    include: /src\/.*\.jsx?$/,
    exclude: [],
  },
  optimizeDeps: {
    esbuildOptions: {
      loader: { '.js': 'jsx' },
    },
  },
  base: '/',
  build: {
    // gmg-server serves this directory statically at the site root.
    outDir: '../gmg-server/public/app',
    emptyOutDir: true,
    // Vite's default 500 KB chunk-size warning is unrealistic for an SPA
    // that eagerly loads MUI + Chart.js + react-toastify; current bundle
    // is ~800 KB. 1000 KB still flags genuine future bloat regressions
    // without nagging on every build.
    chunkSizeWarningLimit: 1000,
  },
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:3001',
      '/socket.io': { target: 'http://localhost:3001', ws: true },
    },
  },
  test: {
    // happy-dom instead of jsdom: jsdom still pulls in the deprecated
    // 'whatwg-encoding' package (no replacement upstream yet), and
    // happy-dom is enough for our single smoke test while keeping
    // `npm install` warning-free.
    environment: 'happy-dom',
    globals: true,
    // Pre-bundle deps so Vitest resolves CJS/ESM interop the same way the
    // dev server and build do (kept as a guardrail even after the Phase 2/3
    // CJS cleanups, since legacy CJS can still creep in via transitive deps).
    deps: {
      optimizer: {
        web: { enabled: true },
      },
    },
  },
})
