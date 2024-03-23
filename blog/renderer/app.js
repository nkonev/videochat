export { createApp }

import { createSSRApp, defineComponent, h, markRaw, reactive } from 'vue'
import PageShell from './PageShell.vue'
import { setPageContext } from './usePageContext'

// Vuetify
import 'vuetify/styles'
import { createVuetify } from 'vuetify'
import * as components from 'vuetify/components'
import * as directives from 'vuetify/directives'

const vuetify = createVuetify({
    components,
    directives,
    ssr: true,
})

function createApp(pageContext) {
  const { Page } = pageContext

  let rootComponent
  const PageWithShell = defineComponent({
    data: () => ({
      Page: markRaw(Page)
    }),
    created() {
      rootComponent = this
    },
    render() {
      return h(
        PageShell,
        {},
        {
          default: () => {
            return h(this.Page)
          }
        }
      )
    }
  })

  const app = createSSRApp(PageWithShell).use(vuetify)

  // We use `app.changePage()` to do Client Routing, see `+onRenderClient.ts`
  Object.assign(app, {
    changePage: (pageContext) => {
      Object.assign(pageContextReactive, pageContext)
      rootComponent.Page = markRaw(pageContext.Page)
    }
  })

  // When doing Client Routing, we mutate pageContext (see usage of `app.changePage()` in `+onRenderClient.ts`).
  // We therefore use a reactive pageContext.
  const pageContextReactive = reactive(pageContext)

  // Make pageContext available from any Vue component
  setPageContext(app, pageContextReactive)

  return app
}
