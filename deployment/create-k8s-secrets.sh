#!/bin/bash

# Create Kubernetes secrets for the pedro chatbot namespace from 1Password.
#
# This script creates:
#   - Secret "pedro-service-account"  — OP_SERVICE_ACCOUNT_TOKEN for op run
#   - ConfigMap "pedro-prod-env"      — prod.env with op:// refs (not real secrets)
#
# The pedro bot containers use `op run --env-file=/app/prod.env` so the ConfigMap
# only needs to contain the op:// reference strings. The actual secret resolution
# happens at container startup via the 1Password CLI.
#
# Prerequisites:
#   - kubectl configured and connected to the cluster
#   - op CLI installed and authenticated (or OP_SERVICE_ACCOUNT_TOKEN set)
#
# Usage: ./create-k8s-secrets.sh

set -euo pipefail

NAMESPACE="chatbot"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROD_ENV_FILE="$REPO_ROOT/prod.env"

echo "=== Pedro k8s Secret Setup ==="
echo "Namespace: $NAMESPACE"
echo ""

# ---------------------------------------------------------------------------
# Preflight checks
# ---------------------------------------------------------------------------
if ! command -v kubectl &>/dev/null; then
  echo "ERROR: kubectl not found"
  exit 1
fi

if ! kubectl cluster-info &>/dev/null; then
  echo "ERROR: kubectl cannot reach the cluster"
  exit 1
fi

if [[ ! -f "$PROD_ENV_FILE" ]]; then
  echo "ERROR: $PROD_ENV_FILE not found"
  exit 1
fi

# Get OP_SERVICE_ACCOUNT_TOKEN — either already in env or prompt
if [[ -z "${OP_SERVICE_ACCOUNT_TOKEN:-}" ]]; then
  read -rsp "Enter OP_SERVICE_ACCOUNT_TOKEN: " OP_SERVICE_ACCOUNT_TOKEN
  echo ""
fi

if [[ -z "$OP_SERVICE_ACCOUNT_TOKEN" ]]; then
  echo "ERROR: OP_SERVICE_ACCOUNT_TOKEN is required"
  exit 1
fi

# ---------------------------------------------------------------------------
# Namespace
# ---------------------------------------------------------------------------
if kubectl get namespace "$NAMESPACE" &>/dev/null; then
  echo "Namespace '$NAMESPACE' already exists"
else
  echo "Creating namespace '$NAMESPACE'..."
  kubectl create namespace "$NAMESPACE"
fi

# ---------------------------------------------------------------------------
# Secret: pedro-service-account (OP token for op run inside containers)
# ---------------------------------------------------------------------------
echo ""
echo "--- Creating Secret 'pedro-service-account' ---"
kubectl create secret generic pedro-service-account \
  --namespace="$NAMESPACE" \
  --from-literal=OP_SERVICE_ACCOUNT_TOKEN="$OP_SERVICE_ACCOUNT_TOKEN" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secret 'pedro-service-account' applied"

# ---------------------------------------------------------------------------
# ConfigMap: pedro-prod-env (prod.env with op:// refs)
# ---------------------------------------------------------------------------
echo ""
echo "--- Creating ConfigMap 'pedro-prod-env' ---"
kubectl create configmap pedro-prod-env \
  --namespace="$NAMESPACE" \
  --from-file=prod.env="$PROD_ENV_FILE" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "ConfigMap 'pedro-prod-env' applied"

# ---------------------------------------------------------------------------
# Done
# ---------------------------------------------------------------------------
echo ""
echo "=== Secrets created ==="
echo ""
echo "Resources in namespace '$NAMESPACE':"
kubectl get secret,configmap -n "$NAMESPACE" | grep -E "pedro|NAME"
echo ""
echo "Now run Helm to deploy the bots:"
echo "  helm upgrade --install pedro-bots ./charts/twitch-llm-bot \\"
echo "    --namespace $NAMESPACE \\"
echo "    --set discord.twitchId=<id> \\"
echo "    --set discord.model=<model-name> \\"
echo "    --set twitch.twitchId=<id> \\"
echo "    --set twitch.model=<model-name>"
