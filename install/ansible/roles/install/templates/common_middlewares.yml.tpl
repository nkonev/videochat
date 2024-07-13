http:
  middlewares:
    api-strip-prefix-middleware:
      stripPrefix:
        prefixes:
          - "/api"
    retry-middleware:
      retry:
        attempts: 4
    auth-middleware:
      forwardAuth:
        address: "http://aaa:8060/internal/profile/auth"
        authRequestHeaders:
          - "Cookie"
          - "uber-trace-id"
        authResponseHeadersRegex: "^X-Auth-"
    redirect-to-https:
      redirectScheme:
        scheme: https

{% if old_domain != None %}
    redirect-from-old-blog-to-public-blog-post:
      redirectRegex:
        regex: "^http.*://{{ old_domain }}/(.*)"
        replacement: "https://{{ domain }}/public/blog/${1}"
{% endif %}

    redirect-from-old-frontend-to-public-blog-post:
      redirectRegex:
        regex: "^http.*://{{ domain }}/blog/post/(.*)"
        replacement: "https://{{ domain }}/public/blog/post/${1}"

