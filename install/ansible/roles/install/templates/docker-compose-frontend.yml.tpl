version: '3.7'

services:
  frontend:
    image: nkonev/chat-frontend:{{ tag }}
    networks:
      backend:
    deploy:
      replicas: 1
#      update_config:
#        parallelism: 1
#        delay: 20s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.frontend-service.loadbalancer.server.port=8082"
        - "traefik.http.routers.frontend-router.rule=PathPrefix(`/`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.frontend-router.entrypoints=https"
        - "traefik.http.routers.frontend-router.middlewares=redirect-from-old-frontend-to-public-blog-post@file,retry-middleware@file"
        - "traefik.http.routers.frontend-router.tls=true"
        - "traefik.http.routers.frontend-router.tls.certresolver=myresolver"

        - "traefik.http.routers.https-redirect-router.rule=PathPrefix(`/`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.https-redirect-router.entrypoints=http"
        - "traefik.http.routers.https-redirect-router.middlewares=redirect-to-https@file"
{% if old_domain != None %}
        - "traefik.http.routers.blog-https-redirect-router.rule=PathPrefix(`/`) && Host(`{{ old_domain }}`)"
        - "traefik.http.routers.blog-https-redirect-router.entrypoints=http"
        - "traefik.http.routers.blog-https-redirect-router.middlewares=redirect-to-https@file"
{% endif %}

    logging:
      driver: "journald"
      options:
        tag: chat-frontend

networks:
  backend:
    driver: overlay
