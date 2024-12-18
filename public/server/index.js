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
import "./instrumentation.js";

import express from 'express'
import compression from 'compression'
import { renderPage } from 'vike/server'
import { root } from './root.js'
import {blog, blog_post, path_prefix} from "../common/router/routes.js"
import { SitemapStream } from 'sitemap'
import {getChatApiUrl, getFrontendUrl, getHttpClientTimeout, getPort} from "../common/config.js";
import axios from "axios";
import opentelemetry from '@opentelemetry/api';
import * as api from '@opentelemetry/api';

axios.defaults.timeout = getHttpClientTimeout();

const isProduction = process.env.NODE_ENV === 'production'

const tracer = opentelemetry.trace.getTracer(
    'public-handlers',
    '0.0.0',
);

startServer()

const pathPrefixAndBlog = path_prefix + blog;

async function startServer() {
  const app = express()
  app.disable('x-powered-by')
  app.use(compression())

  const traceHeader = function (req, res, next) {
        // https://opentelemetry.io/docs/languages/js/context/
        const ctx = api.context.active();
        // https://opentelemetry.io/docs/languages/js/instrumentation/#get-a-span-from-context
        const span = opentelemetry.trace.getSpan(ctx);
        if (span) {
            const traceId = span.spanContext().traceId;
            // console.log("processing traceId", traceId);
            res.header('trace-id', traceId);
        }
        next()
  }

  app.use(traceHeader)

  // Vite integration
  if (isProduction) {
    app.get('/*',function (req, res, next) {
        if (req.url.startsWith(path_prefix)) { // patched by me for sitemap and resources. use production package (you can use docker image built by make)
            const newUrl = req.url.slice(path_prefix.length);
            // console.log("Patching url from ", req.url, "to ", newUrl);
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

  const sitemapHandler = async function(req, res) {
      tracer.startActiveSpan('sitemapXmlHandler', async (span) => {
          res.header('Content-Type', 'application/xml');

          try {
              const smStream = new SitemapStream({hostname: getFrontendUrl()});

              // index page
              smStream.write({url: pathPrefixAndBlog + "/", lastmod: new Date()})

              const apiHost = getChatApiUrl();
              const PAGE_SIZE = 40;
              for (let page = 0; ; page++) {
                  const response = await axios.get(apiHost + `/internal/blog/seo?page=${page}&size=${PAGE_SIZE}`);
                  const data = response.data;
                  if (data.length == 0) {
                      break
                  }
                  for (const item of data) {
                      smStream.write({url: path_prefix + blog_post + `/${item.chatId}`, lastmod: item.lastModified})
                  }
              }

              // stream write the response
              smStream.pipe(res).on('error', (e) => {
                  throw e
              })

              // make sure to attach a write stream such as streamToPromise before ending
              smStream.end()
          } catch (e) {
              console.error(e)
              res.status(500).end()
          } finally {
              span.end();
          }
      })
  }

  app.get('/sitemap.xml', sitemapHandler);
  app.get('/blog/sitemap.xml', sitemapHandler); // for google, url like http://localhost:8081/public/blog/sitemap.xml

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
      tracer.startActiveSpan('ssrHandler', async (span) => {
        try {
          const pageContextInit = {
              urlOriginal: req.originalUrl,
              userAgent: req.headers["user-agent"]
          }
          const pageContext = await renderPage(pageContextInit)
          if (pageContext.errorWhileRendering) {
              // Install error tracking here, see https://vike.dev/errors
          }
          const {httpResponse} = pageContext
          let overrideStatus = null;
          if (pageContext.httpStatus) {
              overrideStatus = pageContext.httpStatus;
          }
          if (!httpResponse) {
              return next()
          } else {
              const {body, statusCode, headers, earlyHints} = httpResponse
              // to help YandexBot to get the page
              // if (res.writeEarlyHints) res.writeEarlyHints({ link: earlyHints.map((e) => e.earlyHintLink) })
              headers.forEach(([name, value]) => res.setHeader(name, value))
              res.status(overrideStatus ? overrideStatus : statusCode)
              // For HTTP streams use httpResponse.pipe() instead, see https://vike.dev/streaming
              res.send(body)
          }
        } finally {
            span.end()
        }
      })

  })

  const port = getPort()
  app.listen(port)
  console.log(`Server running at http://localhost:${port}`)
}
