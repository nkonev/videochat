FROM alpine:3.13.0
RUN apk add --no-cache ca-certificates
ARG BINARY
COPY ./$BINARY /sbin/app
ENTRYPOINT ["/sbin/app"]