#!/bin/bash

set -e

echo "=== Starting Pedro Local Development Environment ==="
echo ""

# Parse arguments
SERVICE=""
PERFORMANCE_TIER=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        discord|twitch|both)
            SERVICE="$1"
            shift
            ;;
        --tier=*)
            PERFORMANCE_TIER="${1#*=}"
            shift
            ;;
        --tier)
            PERFORMANCE_TIER="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [SERVICE] [OPTIONS]"
            echo ""
            echo "SERVICE:"
            echo "  discord     Start Discord bot only (default)"
            echo "  twitch      Start Twitch bot only"
            echo "  both        Start both Discord and Twitch bots"
            echo ""
            echo "OPTIONS:"
            echo "  --tier=TIER    Performance tier: laptop, studio, or auto (default: auto)"
            echo ""
            echo "Performance Tiers:"
            echo "  laptop    M1 Max 64GB - 32B model, 32 ctx, 4 parallel"
            echo "  studio    M3 Ultra 96GB - 72B model, 128k ctx, 8 parallel"
            echo "  auto      Auto-detect based on system RAM"
            echo ""
            echo "Environment Variables:"
            echo "  PEDRO_PERFORMANCE_TIER    Override performance tier"
            echo "  PEDRO_MODEL              Override model name"
            echo "  PEDRO_NUM_CTX            Override context window"
            echo "  PEDRO_NUM_PARALLEL       Override parallel requests"
            echo ""
            echo "Examples:"
            echo "  ./run.sh discord                    # Auto-detect hardware"
            echo "  ./run.sh discord --tier=laptop      # Optimize for M1 Max"
            echo "  ./run.sh both --tier=studio         # Optimize for M3 Ultra"
            echo "  PEDRO_MODEL=qwen2.5-coder:14b ./run.sh discord"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Run with -h for help"
            exit 1
            ;;
    esac
done

# Default to discord if no service specified
SERVICE=${SERVICE:-discord}

# Determine performance tier
if [ -n "$PEDRO_PERFORMANCE_TIER" ]; then
    PERFORMANCE_TIER="$PEDRO_PERFORMANCE_TIER"
elif [ -z "$PERFORMANCE_TIER" ] || [ "$PERFORMANCE_TIER" = "auto" ]; then
    # Auto-detect based on system memory
    if [[ "$OSTYPE" == "darwin"* ]]; then
        TOTAL_RAM_GB=$(sysctl -n hw.memsize | awk '{print int($1/1024/1024/1024)}')
    else
        TOTAL_RAM_GB=$(free -g | awk '/^Mem:/{print $2}')
    fi

    if [ "$TOTAL_RAM_GB" -ge 80 ]; then
        PERFORMANCE_TIER="studio"
    elif [ "$TOTAL_RAM_GB" -ge 48 ]; then
        PERFORMANCE_TIER="laptop"
    else
        PERFORMANCE_TIER="basic"
    fi
    echo "🔍 Auto-detected: ${TOTAL_RAM_GB}GB RAM → ${PERFORMANCE_TIER} tier"
fi

# Set model and parameters based on tier
case "$PERFORMANCE_TIER" in
    studio)
        MODEL=${PEDRO_MODEL:-"qwen2.5-coder:72b-instruct"}
        NUM_CTX=${PEDRO_NUM_CTX:-131072}  # 128k context
        NUM_PARALLEL=${PEDRO_NUM_PARALLEL:-8}
        echo "🚀 Performance: M3 Ultra / Mac Studio (96GB)"
        ;;
    laptop)
        MODEL=${PEDRO_MODEL:-"qwen2.5-coder:32b-instruct"}
        NUM_CTX=${PEDRO_NUM_CTX:-32768}   # 32k context
        NUM_PARALLEL=${PEDRO_NUM_PARALLEL:-4}
        echo "💻 Performance: M1 Max / MacBook Pro (64GB)"
        ;;
    basic)
        MODEL=${PEDRO_MODEL:-"qwen2.5-coder:7b-instruct"}
        NUM_CTX=${PEDRO_NUM_CTX:-8192}    # 8k context
        NUM_PARALLEL=${PEDRO_NUM_PARALLEL:-2}
        echo "⚡ Performance: Basic (< 48GB RAM)"
        ;;
    *)
        echo "❌ Unknown performance tier: $PERFORMANCE_TIER"
        echo "Valid tiers: studio, laptop, basic, auto"
        exit 1
        ;;
esac

echo "   Model: $MODEL"
echo "   Context: $NUM_CTX tokens"
echo "   Parallel: $NUM_PARALLEL requests"
echo ""

# Export for docker-compose
export PEDRO_MODEL="$MODEL"
export PEDRO_NUM_CTX="$NUM_CTX"
export PEDRO_NUM_PARALLEL="$NUM_PARALLEL"

# Check if Ollama is running
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "❌ Ollama is not running!"
    echo ""
    echo "Please start Ollama in another terminal:"
    echo "  ollama serve"
    echo ""
    exit 1
fi

echo "✅ Ollama is running"

# Check if model exists, offer to pull if not
if ! ollama list | grep -q "$MODEL"; then
    echo ""
    echo "⚠️  Model $MODEL not found locally"
    echo ""
    read -p "Pull model now? This may take a while. (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Pulling $MODEL..."
        ollama pull "$MODEL"
    else
        echo "Skipping model pull. You can pull it manually:"
        echo "  ollama pull $MODEL"
        echo ""
        echo "Continuing with existing models..."
    fi
fi
echo ""

case "$SERVICE" in
    discord)
        echo "Starting Discord bot with PostgreSQL..."
        docker compose up -d postgres pedro-discord
        ;;
    twitch)
        echo "Starting Twitch bot with PostgreSQL..."
        docker compose --profile twitch up -d postgres pedro-twitch
        ;;
    both)
        echo "Starting both Discord and Twitch bots with PostgreSQL..."
        docker compose --profile twitch up -d
        ;;
    *)
        echo "Usage: $0 [discord|twitch|both]"
        echo ""
        echo "Examples:"
        echo "  ./run.sh discord  # Start Discord bot only (default)"
        echo "  ./run.sh twitch   # Start Twitch bot only"
        echo "  ./run.sh both     # Start both bots"
        exit 1
        ;;
esac

echo ""
echo "Waiting for services to start..."
sleep 3

echo ""
echo "=== Services Status ==="
docker compose ps

echo ""
echo "=== View Logs ==="
echo "  All services:     docker compose logs -f"
echo "  Discord only:     docker compose logs -f pedro-discord"
echo "  Twitch only:      docker compose logs -f pedro-twitch"
echo "  Database only:    docker compose logs -f postgres"
echo ""
echo "=== Metrics Endpoints ==="
echo "  Discord:          http://localhost:6060/metrics"
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "  Twitch:           http://localhost:6061/metrics"
    echo "  Twitch OAuth:     http://localhost:3000/oauth/redirect"
fi
echo ""
echo "=== Ollama ==="
echo "  API:              http://localhost:11434"
echo "  Models:           ollama list"
echo "  Pull model:       ollama pull qwen2.5-coder:7b-instruct"
echo ""
echo "=== Stop Services ==="
echo "  Run: ./stop.sh"
