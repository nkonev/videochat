// Plugins
import vue from '@vitejs/plugin-vue'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import anotherEntrypointIndexHtmlPlugin from "./vite/another-entrypoint-index-html-plugin";

// Utilities
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import { resolve } from 'path';

import { VitePWA } from 'vite-plugin-pwa';

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
    target: 'esnext' // Fix for Top-level await is not available in the configured target environment ("chrome87", "edge88", "es2020", "firefox78", "safari14" + 2 overrides)
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
    VitePWA({
      srcDir: "src",
      filename: "service-worker.js",
      strategies: "injectManifest",
      injectRegister: false,
      manifest: false,
      injectManifest: {
        injectionPoint: null,
      },
      devOptions: {
        enabled: true
      }
    })
  ],
  define: { 'process.env': {} },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
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
