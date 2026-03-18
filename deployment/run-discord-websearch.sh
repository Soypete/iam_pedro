#!/bin/bash
# Run Discord bot with agent websearch feature
# Branch: discord-agent-websearch

set -e

echo "🚀 Starting pedro-discord (agent websearch)..."

docker run -d \
    --name pedro-discord \
    --restart unless-stopped \
    -p 6060:6060 \
    -v /opt/pedro/prod.env:/app/prod.env:ro \
    -e OP_SERVICE_ACCOUNT_TOKEN="$(grep OP_SERVICE_ACCOUNT_TOKEN /opt/pedro/service.env | cut -d= -f2)" \
    -e TWITCH_ID="$(grep TWITCH_ID /opt/pedro/service.env | cut -d= -f2)" \
    -e LLAMA_CPP_PATH="http://100.121.229.114:8000/v1" \
    localhost/pedro-discord:discord-agent-websearch \
    op run --env-file=/app/prod.env -- /app/main -model hugging-quants/Meta-Llama-3.1-8B-Instruct-AWQ-INT4

echo "✅ pedro-discord started"
echo "📊 Health check: curl http://localhost:6060/healthz"
echo "📝 View logs: docker logs -f pedro-discord"
