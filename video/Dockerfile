FROM alpine:3.16.2
RUN apk add --no-cache ca-certificates curl
COPY ./video /usr/local/bin/video
ENTRYPOINT ["/usr/local/bin/video"]
