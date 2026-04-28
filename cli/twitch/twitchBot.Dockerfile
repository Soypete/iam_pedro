FROM golang:1.26-alpine

RUN apk add --no-cache ca-certificates

WORKDIR /app

EXPOSE 6060

COPY bin/twitch /app/main

CMD ["/app/main"]