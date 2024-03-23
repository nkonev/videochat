import vue from '@vitejs/plugin-vue'
import vike from 'vike/plugin'
import vuetify from 'vite-plugin-vuetify'

export default {
  plugins: [
      vue(),
      vike(),
      vuetify({
          autoImport: true,
      }),
  ],
  ssr: {
    // https://github.com/vuetifyjs/vuetify/issues/15700
    noExternal: [ /\.css$/, /^vuetify/ ],
  },
}
