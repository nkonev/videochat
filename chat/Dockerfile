FROM alpine:3.16.2
RUN apk add --no-cache ca-certificates curl
COPY ./chat /usr/local/bin/chat
ENTRYPOINT ["/usr/local/bin/chat"]
