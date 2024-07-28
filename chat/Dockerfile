FROM alpine:3.20.2
RUN apk add --no-cache ca-certificates curl
COPY ./chat /usr/local/bin/chat
ENTRYPOINT ["/usr/local/bin/chat"]
