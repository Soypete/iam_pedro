FROM golang:1.25-alpine

RUN apk add --no-cache git ca-certificates

WORKDIR /app
COPY go.* ./
RUN go mod download

COPY . ./
RUN go build -v -o keepalive-service ./cli/keepalive

CMD ["/app/keepalive-service"]
