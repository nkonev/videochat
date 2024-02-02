// Plugins
import vue from '@vitejs/plugin-vue'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import anotherEntrypointIndexHtmlPlugin from "./vite/another-entrypoint-index-html-plugin";

// Utilities
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import { resolve } from 'path';

import GlobalsPolyfills from '@esbuild-plugins/node-globals-polyfill';

// https://vitejs.dev/config/
const base = "/";

export default defineConfig({
  base: base,
  build: {
    rollupOptions: {
      input: {
        appMain: resolve(__dirname, 'index.html'),
        appBlog: resolve(__dirname, 'blog', 'index.html'),
      },
    },
  },
  plugins: [
    anotherEntrypointIndexHtmlPlugin(null, "/blog"),
    vue({
      template: { transformAssetUrls }
    }),
    // https://github.com/vuetifyjs/vuetify-loader/tree/next/packages/vite-plugin
    vuetify({
      autoImport: true,
      styles: {
        configFile: 'src/styles/settings.scss',
      },
    }),
  ],
  optimizeDeps: {
    esbuildOptions: {
      define: {
        global: 'globalThis',
      },
      plugins: [
        GlobalsPolyfills({
          process: true,
          buffer: true,
        }),
      ],
    },
  },
  define: { 'process.env': {} },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      stream: 'stream-browserify',
      // https: 'agent-base',
      // comment above line and uncomment below line if it doesnot work
      http:'agent-base',
    },
    extensions: [
      '.js',
      '.json',
      '.jsx',
      '.mjs',
      '.ts',
      '.tsx',
      '.vue',
    ],
  },
  server: {
    port: 3000,
    strictPort: true,
  },
})
