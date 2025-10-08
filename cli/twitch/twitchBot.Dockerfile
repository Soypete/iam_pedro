FROM 1password/op:2

FROM golang:1.23-alpine

RUN apk add --no-cache git

COPY --from=1password/op:2 /usr/local/bin/op /usr/local/bin/op

WORKDIR /app
# This comes from the root directory
COPY go.* ./
RUN go mod download

EXPOSE 6060

# pull in all modules from the repo
COPY . ./
RUN go build -v -o main ./cli/twitch

CMD ["op", "run", "--", "--env-file", "prod.env", "/app/main", "-model", "deepseek", "-discordMode", "-twitchMode"]
