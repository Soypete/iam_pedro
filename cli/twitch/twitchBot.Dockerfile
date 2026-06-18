FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o twitch ./cli/twitch

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

EXPOSE 6060

COPY --from=builder /build/twitch /app/main

CMD ["/app/main"]