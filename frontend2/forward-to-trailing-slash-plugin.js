const assetExtensions = new Set([
  'cjs',
  'css',
  'graphql',
  'ico',
  'jpeg',
  'jpg',
  'js',
  'json',
  'map',
  'mjs',
  'png',
  'sass',
  'scss',
  'svg',
  'styl',
  'stylus',
  'ts',
  'tsx',
  'woff',
  'woff2',
]);

export default (base) => ({
  name: 'forward-to-trailing-slash',
  configureServer(server) {
    server.middlewares.use((req, res, next) => {
      const { url, headers } = req;

      const normalizedBase = base ? base : "";
      const startsWithAt = url?.startsWith(`${normalizedBase}/@`);
      if (startsWithAt) {
        return next();
      }

      // needed for dynamic routing components in vue
      const startsWithSrc = url?.startsWith(`${normalizedBase}/src`);
      if (startsWithSrc) {
        return next();
      }

      const startsNodeModules = url?.startsWith(`${normalizedBase}/node_modules`);
      if (startsNodeModules) {
        return next();
      }

      const realUrl = new URL(
        url ?? '.',
        `${headers[':scheme'] ?? 'http'}://${headers[':authority'] ?? headers.host}`,
      );

      const endsWithSlash = realUrl.pathname.endsWith('/');
      if (!endsWithSlash) {
        const ext = realUrl.pathname.split('.').pop();
        if (!ext || !assetExtensions.has(ext)) {
          realUrl.pathname = `${realUrl.pathname}/`;
          req.url = `${realUrl.pathname}${realUrl.search}`;
        }
      }

      return next();
    });
  },
});
