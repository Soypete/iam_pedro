FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

EXPOSE 6060

COPY bin/twitch /app/main

CMD ["/app/main"]
