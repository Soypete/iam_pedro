#!/bin/bash
# Run Twitch bot with auth health check feature
# Branch: twitch-auth-health-check

set -e

echo "🚀 Starting pedro-twitch (auth health check)..."

docker run -d \
    --name pedro-twitch \
    --restart unless-stopped \
    -p 6061:6061 \
    -v /opt/pedro/prod.env:/app/prod.env:ro \
    -e OP_SERVICE_ACCOUNT_TOKEN="$(grep OP_SERVICE_ACCOUNT_TOKEN /opt/pedro/service.env | cut -d= -f2)" \
    -e TWITCH_ID="$(grep TWITCH_ID /opt/pedro/service.env | cut -d= -f2)" \
    -e LLAMA_CPP_PATH="http://100.112.230.20:8000/v1" \
    localhost/pedro-twitch:twitch-auth-health-check \
    op run --env-file=/app/prod.env -- /app/main -model hugging-quants/Meta-Llama-3.1-8B-Instruct-AWQ-INT4

echo "✅ pedro-twitch started"
echo "📊 Health check: curl http://localhost:6061/healthz"
echo "📝 View logs: docker logs -f pedro-twitch"
