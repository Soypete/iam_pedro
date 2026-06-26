#!/bin/bash

# Deploy the embeddings sidecar end-to-end:
#   1. build + push the pedro-embed image (CPU llama.cpp + nomic-embed-text)
#   2. build + push the pedro-twitch image (carries the EMBEDDINGS_PATH code changes)
#   3. pin both image tags in charts/pedro-bots/values.yaml
#   4. helm upgrade the release (migration 0010 auto-runs at twitch-bot startup via goose)
#   5. verify the rollout + the embeddings endpoint
#
# No secrets are handled here — twitch/postgres creds come from OpenBAO injection in
# the chart, and the embed sidecar needs none (public model, no auth).
#
# Prereqs: docker, helm, kubectl (pointed at the K3s cluster), push access to the
# registry. Run from the repo root or this dir.
#
# Usage:
#   ./deploy-embed.sh [TAG]            # full build+push+deploy (default tag: short SHA)
#   ./deploy-embed.sh --no-deploy TAG  # build+push images + bump values, skip helm
#   ./deploy-embed.sh --embed-only TAG # only (re)build+push the embed image

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
REGISTRY="100.81.89.62:5000"
NAMESPACE="chatbot"
RELEASE="pedro"
CHART_DIR="$REPO_ROOT/charts/pedro-bots"
VALUES="$CHART_DIR/values.yaml"

DEPLOY=true
EMBED_ONLY=false
TAG=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-deploy)  DEPLOY=false; shift ;;
    --embed-only) EMBED_ONLY=true; shift ;;
    -h|--help)    sed -n '3,20p' "$0"; exit 0 ;;
    *)            TAG="$1"; shift ;;
  esac
done

TAG="${TAG:-$(git -C "$REPO_ROOT" rev-parse --short HEAD)}"
EMBED_REF="$REGISTRY/pedro-embed:$TAG"
TWITCH_REF="$REGISTRY/pedro-twitch:$TAG"

require() { command -v "$1" >/dev/null 2>&1 || { echo "ERROR: '$1' not found in PATH"; exit 1; }; }
require docker

# ---------------------------------------------------------------------------
# 1. embed image
# ---------------------------------------------------------------------------
echo "=== Building + pushing $EMBED_REF ==="
docker build -f "$SCRIPT_DIR/Dockerfile" -t "$EMBED_REF" "$SCRIPT_DIR"
docker push "$EMBED_REF"

if [[ "$EMBED_ONLY" == "true" ]]; then
  echo "embed image pushed: $EMBED_REF (--embed-only; nothing else changed)"
  exit 0
fi

# ---------------------------------------------------------------------------
# 2. twitch image (contains the EMBEDDINGS_PATH/MODEL code)
# ---------------------------------------------------------------------------
echo "=== Building + pushing $TWITCH_REF ==="
docker build -f "$REPO_ROOT/cli/twitch/twitchBot.Dockerfile" -t "$TWITCH_REF" "$REPO_ROOT"
docker push "$TWITCH_REF"

# ---------------------------------------------------------------------------
# 3. pin image tags in values.yaml (twitchBot.image.tag + embedSidecar.image.tag)
# ---------------------------------------------------------------------------
echo "=== Pinning image tags ($TAG) in $VALUES ==="
# twitchBot.image.tag — the line after 'repository: .../pedro-twitch'
perl -0pi -e "s{(repository:\s*\Q$REGISTRY\E/pedro-twitch\s*\n\s*tag:\s*)\S+}{\${1}$TAG}g" "$VALUES"
# embedSidecar.image.tag — the line after 'repository: .../pedro-embed'
perl -0pi -e "s{(repository:\s*\Q$REGISTRY\E/pedro-embed\s*\n\s*tag:\s*)\S+}{\${1}$TAG}g" "$VALUES"
echo "  twitch + embed tags set to $TAG"
grep -nE "pedro-(twitch|embed)|tag:" "$VALUES" | grep -A1 -E 'pedro-(twitch|embed)' || true

if [[ "$DEPLOY" == "false" ]]; then
  echo "--no-deploy: images pushed and $VALUES updated. Run 'helm upgrade $RELEASE $CHART_DIR -n $NAMESPACE' when ready."
  exit 0
fi

# ---------------------------------------------------------------------------
# 4. helm upgrade (migration 0010 runs automatically at twitch startup via goose)
# ---------------------------------------------------------------------------
require helm
require kubectl
echo "=== helm upgrade $RELEASE -n $NAMESPACE ==="
helm upgrade "$RELEASE" "$CHART_DIR" -n "$NAMESPACE"

# ---------------------------------------------------------------------------
# 5. verify
# ---------------------------------------------------------------------------
echo "=== Waiting for twitch rollout ==="
# Deployment name is "<release>-twitch" (release: pedro) -> pedro-twitch.
kubectl rollout status deployment/${RELEASE}-twitch -n "$NAMESPACE" --timeout=180s || \
  kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=twitch

POD="$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/component=twitch -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
if [[ -n "$POD" ]]; then
  echo "=== Smoke-testing embeddings sidecar in $POD ==="
  kubectl exec -n "$NAMESPACE" "$POD" -c embed -- \
    curl -s http://localhost:8081/v1/embeddings \
      -H 'Content-Type: application/json' \
      -d '{"model":"nomic-embed-text","input":"hello"}' \
    | head -c 200 || echo "(embed curl failed — check 'kubectl logs -n $NAMESPACE $POD -c embed')"
  echo ""
  echo "Logs: kubectl logs -n $NAMESPACE $POD -c embed       # embeddings server"
  echo "      kubectl logs -n $NAMESPACE $POD -c twitch-bot  # mem-palace ontology load + migration 0010"
fi

echo ""
echo "=== Done ==="
