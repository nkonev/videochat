FROM alpine:3.16.2
RUN apk add --no-cache ca-certificates
ARG BINARY
COPY ./$BINARY /usr/local/bin/storage
ENTRYPOINT ["/usr/local/bin/storage"]