#!/bin/bash

# Build and push the embeddings sidecar image (pedro-embed).
#
# A CPU-only llama.cpp server serving nomic-embed-text (768-dim, mean pooling) on
# :8081. Runs as a sidecar in the twitch-bot pod so mem-palace / FAQ can embed text
# (the pedrogpt chat server runs MTP and no longer serves /v1/embeddings).
#
# Usage:
#   ./build-embed.sh [TAG]          # build + push to the registry (default tag: short SHA)
#   ./build-embed.sh --no-push TAG  # build only

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY="100.81.89.62:5000"
IMAGE="pedro-embed"
PUSH=true

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-push) PUSH=false; shift ;;
    -h|--help) head -11 "$0" | tail -8; exit 0 ;;
    *) TAG="$1"; shift ;;
  esac
done

TAG="${TAG:-$(git rev-parse --short HEAD)}"
REF="$REGISTRY/$IMAGE:$TAG"

echo "=== Building $REF (CPU llama.cpp + nomic-embed-text) ==="
# Build context is the embed dir (Dockerfile + downloads its own model).
docker build -f "$SCRIPT_DIR/Dockerfile" -t "$REF" "$SCRIPT_DIR"

if [[ "$PUSH" == "true" ]]; then
  echo "=== Pushing $REF ==="
  docker push "$REF"
  echo ""
  echo "Set twitchBot.embedSidecar.image.tag=$TAG in charts/pedro-bots/values.yaml and deploy."
else
  echo "Built $REF (not pushed)."
fi
