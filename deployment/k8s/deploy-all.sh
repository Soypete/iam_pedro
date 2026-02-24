#!/usr/bin/env bash
set -euo pipefail

# Deploy all Pedro services to k3s via Helm
# Usage: ./deploy-all.sh [IMAGE_TAG]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/charts/pedro"
NAMESPACE="pedro"
RELEASE_NAME="pedro"
IMAGE_TAG="${1:-latest}"

echo "=== Pedro Full Stack Deploy ==="
echo "Image tag: ${IMAGE_TAG}"
echo "Namespace: ${NAMESPACE}"
echo "Services:  discord, twitch, keepalive"
echo ""

# Check prerequisites
command -v helm >/dev/null 2>&1 || { echo "ERROR: helm is not installed"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "ERROR: kubectl is not installed"; exit 1; }

# Verify cluster access
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "ERROR: Cannot connect to Kubernetes cluster. Check your kubeconfig."
    exit 1
fi

# Check if pedro-secrets exists
if ! kubectl get secret pedro-secrets -n "${NAMESPACE}" >/dev/null 2>&1; then
    echo "Secret 'pedro-secrets' not found in namespace '${NAMESPACE}'."
    echo ""
    echo "Create it now? You'll need the following values:"
    echo "  - LLAMA_CPP_PATH (llama.cpp server URL)"
    echo "  - POSTGRES_URL"
    echo "  - POSTGRES_VECTOR_URL"
    echo "  - DISCORD_SECRET"
    echo "  - TWITCH_SECRET"
    echo "  - TWITCH_ID"
    echo "  - MODEL (model name for llama.cpp)"
    echo ""
    read -rp "Create secret interactively? (y/N): " response
    if [[ "${response}" == "y" || "${response}" == "Y" ]]; then
        # Create namespace first
        kubectl create namespace "${NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

        read -rp "LLAMA_CPP_PATH: " LLAMA_CPP_PATH
        read -rp "POSTGRES_URL: " POSTGRES_URL
        read -rp "POSTGRES_VECTOR_URL (blank if none): " POSTGRES_VECTOR_URL
        read -rsp "DISCORD_SECRET: " DISCORD_SECRET; echo
        read -rsp "TWITCH_SECRET: " TWITCH_SECRET; echo
        read -rp "TWITCH_ID: " TWITCH_ID
        read -rp "MODEL: " MODEL

        kubectl create secret generic pedro-secrets -n "${NAMESPACE}" \
            --from-literal=llama-cpp-path="${LLAMA_CPP_PATH}" \
            --from-literal=postgres-url="${POSTGRES_URL}" \
            --from-literal=postgres-vector-url="${POSTGRES_VECTOR_URL}" \
            --from-literal=discord-secret="${DISCORD_SECRET}" \
            --from-literal=twitch-secret="${TWITCH_SECRET}" \
            --from-literal=twitch-id="${TWITCH_ID}" \
            --from-literal=model="${MODEL}"

        echo "Secret created."
        SECRETS_FLAG="--set secrets.create=false"
    else
        echo ""
        echo "Create it manually:"
        echo ""
        echo "  kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -"
        echo ""
        echo "  kubectl create secret generic pedro-secrets -n ${NAMESPACE} \\"
        echo "    --from-literal=llama-cpp-path='http://your-llama-cpp:8080' \\"
        echo "    --from-literal=postgres-url='postgres://...' \\"
        echo "    --from-literal=postgres-vector-url='' \\"
        echo "    --from-literal=discord-secret='your-discord-token' \\"
        echo "    --from-literal=twitch-secret='your-twitch-secret' \\"
        echo "    --from-literal=twitch-id='your-twitch-id' \\"
        echo "    --from-literal=model='your-model-name'"
        echo ""
        exit 1
    fi
else
    echo "Found existing pedro-secrets in namespace ${NAMESPACE}"
    SECRETS_FLAG="--set secrets.create=false"
fi

echo ""
echo "Deploying all services..."

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
    --namespace "${NAMESPACE}" \
    --create-namespace \
    --set discord.enabled=true \
    --set discord.image.tag="${IMAGE_TAG}" \
    --set twitch.enabled=true \
    --set twitch.image.tag="${IMAGE_TAG}" \
    --set keepalive.enabled=true \
    --set keepalive.image.tag="${IMAGE_TAG}" \
    ${SECRETS_FLAG} \
    "$@"

echo ""
echo "Waiting for deployments..."
kubectl rollout status deployment/pedro-discord -n "${NAMESPACE}" --timeout=120s
kubectl rollout status deployment/pedro-twitch -n "${NAMESPACE}" --timeout=120s

echo ""
echo "=== All services deployed ==="
echo ""
echo "Pods:"
kubectl get pods -n "${NAMESPACE}"
echo ""
echo "CronJobs:"
kubectl get cronjob -n "${NAMESPACE}"
echo ""
echo "Services:"
kubectl get svc -n "${NAMESPACE}"
echo ""
echo "--- Quick reference ---"
echo "Discord logs:   kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/component=discord -f"
echo "Twitch logs:    kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/component=twitch -f"
echo "Keepalive logs: kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/component=keepalive --tail=50"
