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
RUN go build -v -o keepalive ./cli/keepalive && chmod +x keepalive

CMD ["sh", "-c", "op run --env-file prod.env -- /app/keepalive -discord-token=\"$DISCORD_SECRET\" -discord-bot-url=\"${DISCORD_BOT_URL:-http://discord-bot:6060/healthz}\" -twitch-bot-url=\"${TWITCH_BOT_URL:-}\" -check-interval=${CHECK_INTERVAL:-60} -alert-interval=${ALERT_INTERVAL:-3600} -log-level=${LOG_LEVEL:-info}"]
