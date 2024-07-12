version: '3.7'

services:
  event:
    image: {{ image }}
    networks:
      backend:
    deploy:
      replicas: 1
#      update_config:
#        parallelism: 1
#        delay: 20s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.event-service.loadbalancer.server.port=1238"
        - "traefik.http.routers.event-router.rule=PathPrefix(`/api/event/graphql`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.event-router.entrypoints=https"
        - "traefik.http.routers.event-router.middlewares=auth-middleware@file,retry-middleware@file"
        - "traefik.http.routers.event-router.tls=true"
        - "traefik.http.routers.event-router.tls.certresolver=myresolver"

        - "traefik.http.routers.event-public-router.rule=PathPrefix(`/api/event/public`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.event-public-router.entrypoints=https"
        - "traefik.http.routers.event-public-router.middlewares=api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.event-public-router.tls=true"
        - "traefik.http.routers.event-public-router.tls.certresolver=myresolver"

        - "traefik.http.middlewares.event-stripprefix-middleware.stripprefix.prefixes=/event"
        - "traefik.http.routers.event-version-router.rule=Path(`/event/git.json`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.event-version-router.entrypoints=https"
        - "traefik.http.routers.event-version-router.tls=true"
        - "traefik.http.routers.event-version-router.tls.certresolver=myresolver"
        - "traefik.http.routers.event-version-router.middlewares=event-stripprefix-middleware"

    environment:
      - EVENT_CHAT.URL.BASE=http://chat:1235
      - EVENT_AAA.URL.BASE=http://aaa:8060
      - EVENT_RABBITMQ.URL=amqp://videoChat:videoChatPazZw0rd@rabbitmq:5672
      - EVENT_OTLP.ENDPOINT=jaeger:4317

    logging:
      driver: "journald"
      options:
        tag: chat-event

networks:
  backend:
    driver: overlay
