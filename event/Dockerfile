FROM alpine:3.20.2
RUN apk add --no-cache ca-certificates curl
COPY ./event /usr/local/bin/event
ENTRYPOINT ["/usr/local/bin/event"]
