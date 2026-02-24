#!/usr/bin/env bash
set -euo pipefail

# Deploy Pedro Keepalive CronJob to k3s via Helm
# Usage: ./deploy-keepalive.sh [IMAGE_TAG]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/charts/pedro"
NAMESPACE="pedro"
RELEASE_NAME="pedro"
IMAGE_TAG="${1:-latest}"

echo "=== Pedro Keepalive CronJob Deploy ==="
echo "Image tag: ${IMAGE_TAG}"
echo "Namespace: ${NAMESPACE}"
echo ""

# Check prerequisites
command -v helm >/dev/null 2>&1 || { echo "ERROR: helm is not installed"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "ERROR: kubectl is not installed"; exit 1; }

# Verify cluster access
if ! kubectl cluster-info >/dev/null 2>&1; then
    echo "ERROR: Cannot connect to Kubernetes cluster. Check your kubeconfig."
    exit 1
fi

# Check if pedro-secrets exists (keepalive needs DISCORD_SECRET for alerting)
if ! kubectl get secret pedro-secrets -n "${NAMESPACE}" >/dev/null 2>&1; then
    echo "Secret 'pedro-secrets' not found in namespace '${NAMESPACE}'."
    echo "The keepalive service needs DISCORD_SECRET for sending alerts."
    echo ""
    echo "Create the secret first:"
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
    read -rp "Continue with Helm-managed secrets? (y/N): " response
    if [[ "${response}" != "y" && "${response}" != "Y" ]]; then
        echo "Aborted. Create the secret first, then re-run."
        exit 1
    fi
    SECRETS_FLAG="--set secrets.create=true"
else
    echo "Found existing pedro-secrets in namespace ${NAMESPACE}"
    SECRETS_FLAG="--set secrets.create=false"
fi

echo ""
echo "Deploying Keepalive CronJob..."

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
    --namespace "${NAMESPACE}" \
    --create-namespace \
    --set keepalive.enabled=true \
    --set keepalive.image.tag="${IMAGE_TAG}" \
    --set discord.enabled=false \
    --set twitch.enabled=false \
    ${SECRETS_FLAG} \
    "$@"

echo ""
echo "=== Keepalive CronJob deployed ==="
kubectl get cronjob -n "${NAMESPACE}" pedro-keepalive
echo ""
echo "Next run:    kubectl get cronjob -n ${NAMESPACE} pedro-keepalive"
echo "Job history: kubectl get jobs -n ${NAMESPACE} -l app.kubernetes.io/component=keepalive"
echo "Logs:        kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/component=keepalive --tail=50"
