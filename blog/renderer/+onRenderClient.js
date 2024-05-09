// https://vike.dev/onRenderClient
export { onRenderClient }

import { createApp } from './app'
import { getPageTitle } from './getPageTitle'

let app
async function onRenderClient(pageContext) {
  // This onRenderClient() hook only supports SSR, see https://vike.dev/render-modes for how to modify onRenderClient()
  // to support SPA
  if (!pageContext.Page) throw new Error('My onRenderClient() hook expects pageContext.Page to be defined')

  if (!app) {
    app = createApp(pageContext)
    app.mount('#app')
  } else {
    app.changePage(pageContext)
  }
  document.title = getPageTitle(pageContext)
}
