version: '3.7'

services:
  notification:
    image: nkonev/chat-notification:{{ tag }}
    networks:
      backend:
    deploy:
      replicas: 1
#      update_config:
#        parallelism: 1
#        delay: 20s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.notification-service.loadbalancer.server.port=1230"
        - "traefik.http.routers.notification-router.rule=PathPrefix(`/api/notification`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.notification-router.entrypoints=https"
        - "traefik.http.routers.notification-router.middlewares=auth-middleware@file,api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.notification-router.tls=true"
        - "traefik.http.routers.notification-router.tls.certresolver=myresolver"

        - "traefik.http.routers.notification-public-router.rule=PathPrefix(`/api/notification/public`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.notification-public-router.entrypoints=https"
        - "traefik.http.routers.notification-public-router.middlewares=api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.notification-public-router.tls=true"
        - "traefik.http.routers.notification-public-router.tls.certresolver=myresolver"

        - "traefik.http.middlewares.notification-stripprefix-middleware.stripprefix.prefixes=/notification"
        - "traefik.http.routers.notification-version-router.rule=Path(`/notification/git.json`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.notification-version-router.entrypoints=https"
        - "traefik.http.routers.notification-version-router.tls=true"
        - "traefik.http.routers.notification-version-router.tls.certresolver=myresolver"
        - "traefik.http.routers.notification-version-router.middlewares=notification-stripprefix-middleware"

    environment:
      - NOTIFICATION_RABBITMQ.URL=amqp://videoChat:videoChatPazZw0rd@rabbitmq:5672
      - NOTIFICATION_POSTGRESQL.URL=postgres://notification:notificationPazZw0rd@postgresql:5432/notification?sslmode=disable&application_name=notification-app
      - NOTIFICATION_OTLP.ENDPOINT=jaeger:4317

    logging:
      driver: "journald"
      options:
        tag: chat-notification

networks:
  backend:
    driver: overlay
