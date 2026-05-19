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
  },
  server: {
    port: 3000,
    proxy: {
      '/api': 'http://localhost:3001',
      '/socket.io': { target: 'http://localhost:3001', ws: true },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    // The app still depends on legacy CommonJS packages (moment,
    // material-ui 0.20, react-s-alert, ...). Pre-bundle deps so Vitest
    // resolves CJS/ESM interop the same way the dev server and build do.
    deps: {
      optimizer: {
        web: { enabled: true },
      },
    },
  },
})
