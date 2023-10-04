
// Composables
import { createApp } from 'vue'

import vuetify from "@/plugins/vuetify";
import blogRouter from "@/router/blogRouter";
import BlogApp from './BlogApp.vue'

export function registerPlugins (app) {
  app
    .use(vuetify)
    .use(blogRouter)
}

const app = createApp(BlogApp)

registerPlugins(app)

app.mount('#app')
