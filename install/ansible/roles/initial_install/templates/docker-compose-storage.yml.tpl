version: '3.7'

services:
  storage:
    image: nkonev/chat-storage:latest
    networks:
      backend:
    deploy:
      replicas: 1
#      update_config:
#        parallelism: 1
#        delay: 20s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.storage-service.loadbalancer.server.port=1236"
        - "traefik.http.routers.storage-router.rule=PathPrefix(`/api/storage`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.storage-router.entrypoints=https"
        - "traefik.http.routers.storage-router.middlewares=auth-middleware@file,api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.storage-router.tls=true"
        - "traefik.http.routers.storage-router.tls.certresolver=myresolver"

        - "traefik.http.routers.storage-public-router.rule=PathPrefix(`/api/storage/public`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.storage-public-router.entrypoints=https"
        - "traefik.http.routers.storage-public-router.middlewares=api-strip-prefix-middleware@file,retry-middleware@file"
        - "traefik.http.routers.storage-public-router.tls=true"
        - "traefik.http.routers.storage-public-router.tls.certresolver=myresolver"

        - "traefik.http.middlewares.storage-stripprefix-middleware.stripprefix.prefixes=/storage"
        - "traefik.http.routers.storage-version-router.rule=Path(`/storage/git.json`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.storage-version-router.entrypoints=https"
        - "traefik.http.routers.storage-version-router.tls=true"
        - "traefik.http.routers.storage-version-router.tls.certresolver=myresolver"
        - "traefik.http.routers.storage-version-router.middlewares=storage-stripprefix-middleware"
    environment:
        - STORAGE_MINIO.INTERNALENDPOINT=minio:9000
        - STORAGE_MINIO.INTERCONTAINERURL=http://minio:9000
        - STORAGE_MINIO.PUBLICDOWNLOADTTL=24h
        - STORAGE_MINIO.PUBLICUPLOADTTL=24h
        - STORAGE_MINIO.ACCESSKEYID={{ minio_user }}
        - STORAGE_MINIO.SECRETACCESSKEY={{ minio_password }}
        - STORAGE_CHAT.URL.BASE=http://chat:1235
        - STORAGE_AAA.URL.BASE=http://aaa:8060
        - STORAGE_LIMITS.STAT.DIR=/data
        - STORAGE_REDIS.ADDRESS=redis:6379
        - STORAGE_REDIS.DB=3
        - STORAGE_LIMITS.ENABLED=true
        - STORAGE_RABBITMQ.URL=amqp://videoChat:videoChatPazZw0rd@rabbitmq:5672
        - STORAGE_SCHEDULERS.CLEANFILESOFDELETEDCHATTASK.ENABLED=true
        - STORAGE_SCHEDULERS.ACTUALIZEPREVIEWSTASK.ENABLED=true
        - STORAGE_OTLP.ENDPOINT=jaeger:4317

    logging:
      driver: "journald"
      options:
        tag: chat-storage
    volumes:
      - /mnt/chat-minio/data:/data
      # use temp dir for uploading large files
      - /mnt/chat-storage-tmp:/tmp

networks:
  backend:
    driver: overlay
