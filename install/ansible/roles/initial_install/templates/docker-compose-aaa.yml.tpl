version: '3.7'

services:
  aaa:
    image: nkonev/chat-aaa:latest
    networks:
      backend:
    deploy:
      replicas: 1
#      update_config:
#        parallelism: 1
#        delay: 120s
      labels:
        - "traefik.enable=true"
        - "traefik.http.services.aaa-service.loadbalancer.server.port=8060"
        - "traefik.http.routers.aaa-router.rule=PathPrefix(`/api/aaa`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.aaa-router.entrypoints=https"
        - "traefik.http.routers.aaa-router.tls=true"
        - "traefik.http.routers.aaa-router.tls.certresolver=myresolver"

        - "traefik.http.routers.aaa-router.middlewares=retry-middleware@file"

        - "traefik.http.middlewares.aaa-stripprefix-middleware.stripprefix.prefixes=/aaa"
        - "traefik.http.routers.aaa-version-router.rule=Path(`/aaa/git.json`) && Host(`{{ domain }}`)"
        - "traefik.http.routers.aaa-version-router.entrypoints=https"
        - "traefik.http.routers.aaa-version-router.tls=true"
        - "traefik.http.routers.aaa-version-router.tls.certresolver=myresolver"
        - "traefik.http.routers.aaa-version-router.middlewares=aaa-stripprefix-middleware,retry-middleware@file"

    environment:
      - _JAVA_OPTIONS=-Djava.security.egd=file:/dev/./urandom -Xms256m -Xmx512m -XX:MetaspaceSize=128M -XX:MaxMetaspaceSize=256M -XX:OnOutOfMemoryError="kill -9 %p" -Dnetworkaddress.cache.ttl=0 -XX:+HeapDumpOnOutOfMemoryError -XX:HeapDumpPath=/opt/aaa
      - SPRING_DATASOURCE_URL=jdbc:postgresql://postgresql:5432/aaa?connectTimeout=10&socketTimeout=40&ApplicationName=aaa-app
      - SPRING_RABBITMQ_ADDRESSES=rabbitmq:5672
      - SPRING_RABBITMQ_USERNAME=videoChat
      - SPRING_RABBITMQ_PASSWORD=videoChatPazZw0rd
      - SPRING_MAIL_HOST={{ mail_host }}
      - SPRING_MAIL_PORT={{ mail_port }}
      - CUSTOM_EMAIL_FROM={{ mail_from }}
      - SPRING_MAIL_USERNAME={{ mail_username }}
      - SPRING_MAIL_PASSWORD={{ mail_password }}
      - CUSTOM_API-URL=https://{{ domain }}/api/aaa
      - CUSTOM_FRONTEND-URL=https://{{ domain }}
      - MANAGEMENT_HEALTH_MAIL_ENABLED=false
{% if google_client_id != None %}
      - spring.security.oauth2.client.registration.google.client-id={{ google_client_id }}
      - spring.security.oauth2.client.registration.google.client-secret={{ google_client_secret }}
      - spring.security.oauth2.client.registration.google.redirect-uri=https://{{ domain }}/api/aaa/login/oauth2/code/google
{% endif %}
{% if vkontakte_client_id != None %}
      - spring.security.oauth2.client.registration.vkontakte.client-id={{ vkontakte_client_id }}
      - spring.security.oauth2.client.registration.vkontakte.client-secret={{ vkontakte_client_secret }}
      - spring.security.oauth2.client.registration.vkontakte.redirect-uri=https://{{ domain }}/api/aaa/login/oauth2/code/vkontakte
      - spring.security.oauth2.client.registration.vkontakte.authorization-grant-type=authorization_code
      - spring.security.oauth2.client.registration.vkontakte.client-authentication-method=client_secret_post
      - spring.security.oauth2.client.provider.vkontakte.authorization-uri=https://oauth.vk.com/authorize
      - spring.security.oauth2.client.provider.vkontakte.token-uri=https://oauth.vk.com/access_token
      - spring.security.oauth2.client.provider.vkontakte.user-info-uri=https://api.vk.com/method/users.get?v=5.92
      - spring.security.oauth2.client.provider.vkontakte.user-info-authentication-method=form
      - spring.security.oauth2.client.provider.vkontakte.user-name-attribute=response
{% endif %}
      - CUSTOM_CSRF_COOKIE_SECURE=true
      - CUSTOM_CSRF_COOKIE_SAME-SITE=Strict
      # Lax needed to have possibility to go by confirmation link in change email confirmation and to take the existing cookie
      - server.servlet.session.cookie.same-site=Lax
      - server.servlet.session.cookie.secure=true
      - management.otlp.tracing.endpoint=http://jaeger:4318/v1/traces
      - spring.data.redis.url=redis://redis:6379/0
      - CUSTOM_INITIAL_ADMIN_PASSWORD={{ initial_admin_password | password_hash(hashtype='bcrypt', salt=initial_admin_password_salt, rounds=10) | replace('$', '$$') }}
      - CUSTOM_ACCESS_LOG_ENABLE={{ aaa_access_log }}
    logging:
      driver: "journald"
      options:
        tag: chat-aaa

networks:
  backend:
    driver: overlay
