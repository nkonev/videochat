
// Composables
import { createApp } from 'vue'

import vuetify from "@/plugins/vuetify";
import blogRouter from "@/router/blogRouter";
import BlogApp from './BlogApp.vue'
import {isMobileBrowser} from "@/utils";
import pinia from "@/store";

export function registerPlugins (app) {
  app
    .use(vuetify)
    .use(blogRouter)
    .use(pinia)
}

const app = createApp(BlogApp)

registerPlugins(app)

app.config.globalProperties.isMobile = () => {
  return isMobileBrowser()
}

app.mount('#app')
