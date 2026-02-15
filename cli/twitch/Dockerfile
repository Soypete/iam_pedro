# K8s-optimized multi-stage build for Twitch bot
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/pedro-twitch ./cli/twitch

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/pedro-twitch /usr/local/bin/pedro-twitch

EXPOSE 6060
EXPOSE 3000

USER nobody:nobody
ENTRYPOINT ["pedro-twitch"]
