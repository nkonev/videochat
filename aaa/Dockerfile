FROM bellsoft/liberica-openjdk-alpine:21.0.2
ARG APP_HOME=/opt/aaa
RUN apk add --no-cache curl
RUN mkdir -p ${APP_HOME}
WORKDIR ${APP_HOME}
# HEALTHCHECK --interval=30s --timeout=3s CMD curl -f http://localhost:3010/actuator/health || exit 1
COPY ./*-exec.jar ${APP_HOME}/aaa.jar
ENTRYPOINT ["java", "-jar", "aaa.jar"]
