FROM 1password/op:2

FROM golang:1.23-alpine

RUN apk add --no-cache git ca-certificates

COPY --from=1password/op:2 /usr/local/bin/op /usr/local/bin/op

WORKDIR /app
# This comes from the root directory
COPY go.* ./
RUN go mod download

# pull in all modules from the repo
COPY . ./
RUN go build -v -o keepalive-service ./cli/keepalive

CMD ["op", "run", "--env-file", "prod.env", "--", "/app/keepalive-service"]
