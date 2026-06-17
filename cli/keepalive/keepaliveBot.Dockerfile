FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY bin/keepalive /app/keepalive-service

CMD ["/app/keepalive-service"]
