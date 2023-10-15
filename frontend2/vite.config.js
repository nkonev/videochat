// Plugins
import vue from '@vitejs/plugin-vue'
import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'
import forwardToTrailingSlashPlugin from './vite/forward-to-trailing-slash-plugin.js'
import anotherEntrypointIndexHtmlPlugin from "./vite/another-entrypoint-index-html-plugin";

// Utilities
import { defineConfig } from 'vite'
import { fileURLToPath, URL } from 'node:url'
import { resolve } from 'path';

// https://vitejs.dev/config/
const base = "/front2";

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
    forwardToTrailingSlashPlugin(base),
    anotherEntrypointIndexHtmlPlugin(base, "/blog"),
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
