FROM alpine:3.16.2
RUN apk add --no-cache ca-certificates ffmpeg
COPY ./storage /usr/local/bin/storage
ENTRYPOINT ["/usr/local/bin/storage"]
