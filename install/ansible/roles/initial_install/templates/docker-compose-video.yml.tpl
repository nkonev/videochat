version: '3.7'

services:
  video:
    image: nkonev/chat-video:latest
    networks:
      backend:
    deploy:
      replicas: 1
      update_config:
        parallelism: 1
        delay: 20s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.video-service.loadbalancer.server.port=1237"

        - "traefik.http.routers.video-router.rule=PathPrefix(`/api/video`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.video-router.entrypoints=https"
        - "traefik.http.routers.video-router.middlewares=auth-middleware@file,api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.video-router.tls=true"
        - "traefik.http.routers.video-router.tls.certresolver=myresolver"

        - "traefik.http.middlewares.video-stripprefix-middleware.stripprefix.prefixes=/video"
        - "traefik.http.routers.video-version-router.rule=Path(`/video/git.json`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.video-version-router.entrypoints=https"
        - "traefik.http.routers.video-version-router.tls=true"
        - "traefik.http.routers.video-version-router.tls.certresolver=myresolver"
        - "traefik.http.routers.video-version-router.middlewares=video-stripprefix-middleware"
    environment:
      - VIDEO_AAA.URL.BASE=http://aaa:8060
      - VIDEO_CHAT.URL.BASE=http://chat:1235
      - VIDEO_STORAGE.URL.BASE=http://storage:1236
      - VIDEO_LIVEKIT.URL=http://livekit:7880
      - VIDEO_RABBITMQ.URL=amqp://videoChat:videoChatPazZw0rd@rabbitmq:5672
      - VIDEO_REDIS.ADDRESS=redis:6379
      - VIDEO_REDIS.DB=4
      - VIDEO_ONLYROLEADMINRECORDING={{ video_only_role_admin_can_record }}
      - VIDEO_OTLP.ENDPOINT=jaeger:4317

    logging:
      driver: "journald"
      options:
        tag: chat-video

networks:
  backend:
    driver: overlay
