# K8s-optimized multi-stage build for Keepalive service
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/pedro-keepalive ./cli/keepalive

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/pedro-keepalive /usr/local/bin/pedro-keepalive

USER nobody:nobody
ENTRYPOINT ["pedro-keepalive"]
