// This file isn't processed by Vite, see https://github.com/vikejs/vike/issues/562
// Consequently:
//  - When changing this file, you needed to manually restart your server for your changes to take effect.
//  - To use your environment variables defined in your .env files, you need to install dotenv, see https://vike.dev/env
//  - To use your path aliases defined in your vite.config.js, you need to tell Node.js about them, see https://vike.dev/path-aliases

// If you want Vite to process your server code then use one of these:
//  - vavite (https://github.com/cyco130/vavite)
//     - See vavite + Vike examples at https://github.com/cyco130/vavite/tree/main/examples
//  - vite-node (https://github.com/antfu/vite-node)
//  - HatTip (https://github.com/hattipjs/hattip)
//    - You can use Bati (https://batijs.dev/) to scaffold a Vike + HatTip app. Note that Bati generates apps that use the V1 design (https://vike.dev/migration/v1-design) and Vike packages (https://vike.dev/vike-packages)

import express from 'express'
import compression from 'compression'
import { renderPage } from 'vike/server'
import { root } from './root.js'
import { blog } from "../common/router/routes.js"
import { SitemapStream } from 'sitemap'
import { getChatApiUrl, getFrontendUrl } from "../common/config.js";
import axios from "axios";

const isProduction = process.env.NODE_ENV === 'production'

startServer()

async function startServer() {
  const app = express()

  app.use(compression())

  // Vite integration
  if (isProduction) {
    app.get('/*',function (req, res, next) {
        if (req.url.startsWith(blog)) {
            const newUrl = req.url.slice(blog.length);
            req.url = newUrl;
        }
        next();
    });

    // In production, we need to serve our static assets ourselves.
    // (In dev, Vite's middleware serves our static assets.)
    const sirv = (await import('sirv')).default
    app.use(sirv(`${root}/dist/client`))
  } else {
    // We instantiate Vite's development server and integrate its middleware to our server.
    // ⚠️ We instantiate it only in development. (It isn't needed in production and it
    // would unnecessarily bloat our production server.)
    const vite = await import('vite')
    const viteDevMiddleware = (
      await vite.createServer({
        root,
        server: { middlewareMode: true }
      })
    ).middlewares
    app.use(viteDevMiddleware)
  }

  app.get('/sitemap.xml', async function(req, res) {
    res.header('Content-Type', 'application/xml');

    try {
        const smStream = new SitemapStream({ hostname: getFrontendUrl() });

        // index page
        smStream.write({url: `/blog/`, lastmod: new Date()})

        const apiHost = getChatApiUrl();
        const PAGE_SIZE = 40;
        for (let page = 0; ; page++) {
            const response = await axios.get(apiHost + `/internal/blog/seo?page=${page}&size=${PAGE_SIZE}`);
            const data = response.data;
            if (data.length == 0) {
                break
            }
            for (const item of data) {
                smStream.write({url: `/blog/post/${item.chatId}`, lastmod: item.lastModified})
            }
        }

        // stream write the response
        smStream.pipe(res).on('error', (e) => {throw e})

        // make sure to attach a write stream such as streamToPromise before ending
        smStream.end()
    } catch (e) {
        console.error(e)
        res.status(500).end()
    }
  })

  app.get('/robots.txt', function (req, res) {
    res.type('text/plain');
    const sitemapUrl = getFrontendUrl() + '/sitemap.xml';
    res.send(`User-agent: *
Sitemap: ${sitemapUrl}`);
  });
  // ...
  // Other middlewares (e.g. some RPC middleware such as Telefunc)
  // ...

  // Vike middleware. It should always be our last middleware (because it's a
  // catch-all middleware superseding any middleware placed after it).
  app.get('*', async (req, res, next) => {
    const pageContextInit = {
      urlOriginal: req.originalUrl
    }
    const pageContext = await renderPage(pageContextInit)
    if (pageContext.errorWhileRendering) {
      // Install error tracking here, see https://vike.dev/errors
    }
    const { httpResponse } = pageContext
    if (!httpResponse) {
      return next()
    } else {
      const { body, statusCode, headers, earlyHints } = httpResponse
      // to help YandexBot to get the page
      // if (res.writeEarlyHints) res.writeEarlyHints({ link: earlyHints.map((e) => e.earlyHintLink) })
      headers.forEach(([name, value]) => res.setHeader(name, value))
      res.status(statusCode)
      // For HTTP streams use httpResponse.pipe() instead, see https://vike.dev/streaming
      res.send(body)
    }
  })

  const port = process.env.PORT || 3100
  app.listen(port)
  console.log(`Server running at http://localhost:${port}`)
}
