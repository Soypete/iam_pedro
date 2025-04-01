FROM 1password/op:2

FROM golang:1.23-alpine

RUN apk add --no-cache git

COPY --from=1password/op:2 /usr/local/bin/op /usr/local/bin/op

WORKDIR /app
COPY go.* ./
RUN go mod download

EXPOSE 6060

COPY . ./
RUN go build -v -o main .

CMD ["op", "run", "--", "--env-file", "prod.env", "/app/main", "-model", "meta-llama3.1", "-discordMode", "-twitchMode"]
