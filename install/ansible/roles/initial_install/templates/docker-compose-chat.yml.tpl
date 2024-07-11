version: '3.7'

services:
  chat:
    image: nkonev/chat:latest
    networks:
      backend:
    deploy:
      replicas: 1
#      update_config:
#        parallelism: 1
#        delay: 20s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.chat-service.loadbalancer.server.port=1235"
        - "traefik.http.routers.chat-router.rule=PathPrefix(`/api/chat`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.chat-router.entrypoints=https"
        - "traefik.http.routers.chat-router.middlewares=auth-middleware@file,api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.chat-router.tls=true"
        - "traefik.http.routers.chat-router.tls.certresolver=myresolver"

        - "traefik.http.routers.blog-router.rule=PathPrefix(`/api/blog`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.blog-router.entrypoints=https"
        - "traefik.http.routers.blog-router.middlewares=api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.blog-router.tls=true"
        - "traefik.http.routers.blog-router.tls.certresolver=myresolver"

        - "traefik.http.routers.chat-public-router.rule=PathPrefix(`/api/chat/public`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.chat-public-router.entrypoints=https"
        - "traefik.http.routers.chat-public-router.middlewares=api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.chat-public-router.tls=true"
        - "traefik.http.routers.chat-public-router.tls.certresolver=myresolver"

        - "traefik.http.middlewares.chat-stripprefix-middleware.stripprefix.prefixes=/chat"
        - "traefik.http.routers.chat-version-router.rule=Path(`/chat/git.json`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.chat-version-router.entrypoints=https"
        - "traefik.http.routers.chat-version-router.tls=true"
        - "traefik.http.routers.chat-version-router.tls.certresolver=myresolver"
        - "traefik.http.routers.chat-version-router.middlewares=chat-stripprefix-middleware"

    environment:
      - CHAT_POSTGRESQL.URL=postgres://chat:chatPazZw0rd@postgresql:5432/chat?sslmode=disable&application_name=chat-app
      - CHAT_SERVER.BODY.LIMIT=100G
      - CHAT_AAA.URL.BASE=http://aaa:8060
      - CHAT_RABBITMQ.URL=amqp://videoChat:videoChatPazZw0rd@rabbitmq:5672
      - CHAT_REDIS.ADDRESS=redis:6379
      - CHAT_REDIS.DB=5
      - CHAT_OTLP.ENDPOINT=jaeger:4317
      - CHAT_FRONTENDURL=https://{{ domain }}
      - CHAT_MESSAGE.ALLOWEDMEDIAURLS=https://{{ domain }},
      - CHAT_MESSAGE.ALLOWEDIFRAMEURLS=https://www.youtube.com,https://coub.com,https://vk.com,https://rutube.ru
      - CHAT_ONLYADMINCANCREATEBLOG={{ chat_only_role_admin_can_create_blog }}

    logging:
      driver: "journald"
      options:
        tag: chat

networks:
  backend:
    driver: overlay
