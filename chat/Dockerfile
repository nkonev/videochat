FROM alpine:3.14.2
RUN apk add --no-cache ca-certificates
ARG BINARY
COPY ./$BINARY /sbin/app
ENTRYPOINT ["/sbin/app"]