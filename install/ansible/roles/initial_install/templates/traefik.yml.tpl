# https://docs.traefik.io/reference/static-configuration/file/
pilot:
  dashboard: false
global:
  checkNewVersion: false
  sendAnonymousUsage: false
entryPoints:
  http:
    address: :80
  https:
    address: :443
  traefik:
    address: :8080
providers:
  providersThrottleDuration: 2s
  docker:
    swarmMode: true
    watch: true
    exposedByDefault: false
  file:
    directory: /etc/traefik/dynamic

api:
  insecure: true
  dashboard: true
log:
  level: WARN
accessLog:
  format: json
  fields:
    defaultMode: keep
    headers:
      defaultMode: drop
      names:
        User-Agent: keep
        uber-trace-id: keep
tracing:
  spanNameLimit: 150
  jaeger:
    samplingType: const
    samplingParam: 1
    localAgentHostPort: jaeger:6831
    propagation: jaeger
    disableAttemptReconnecting: false
    gen128Bit: true

certificatesResolvers:
  myresolver:
    acme:
      email: {{ acme_email }}
      storage: /etc/traefik/acme.json
      httpChallenge:
        # used during the challenge
        entryPoint: http
