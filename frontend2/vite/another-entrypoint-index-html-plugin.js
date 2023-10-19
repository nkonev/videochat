// fixes the first load the appropriate index.html for /blog/<whatever>

export default (base, subroute) => ({
  name: "another-entrypoint-index-html",
  configureServer(server) {
    server.middlewares.use(
      (req, res, next) => {
        const { url, headers } = req;
        const realUrl = new URL(
          url ?? '.',
          `${headers[':scheme'] ?? 'http'}://${headers[':authority'] ?? headers.host}`,
        );

        const normalizedBase = base ? base : "";
        if (realUrl.pathname.startsWith(`${normalizedBase}${subroute}`)) {
          realUrl.pathname = `${normalizedBase}${subroute}/index.html`;
          req.url = `${realUrl.pathname}${realUrl.search}`;
        }

        return next();
      }
    )
  }
})
