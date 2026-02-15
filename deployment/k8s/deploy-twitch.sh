#!/usr/bin/env bash
set -euo pipefail

# Deploy Pedro Twitch bot to k3s via Helm
# Usage: ./deploy-twitch.sh [IMAGE_TAG]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/charts/pedro"
NAMESPACE="pedro"
RELEASE_NAME="pedro"
IMAGE_TAG="${1:-latest}"

echo "=== Pedro Twitch Bot Deploy ==="
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

# Check if pedro-secrets exists; prompt to create if not
if ! kubectl get secret pedro-secrets -n "${NAMESPACE}" >/dev/null 2>&1; then
    echo "Secret 'pedro-secrets' not found in namespace '${NAMESPACE}'."
    echo ""
    echo "You can create it by running:"
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
    echo "Or let Helm manage it by passing --set secrets.create=true with the values."
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
echo "Deploying Twitch bot..."

helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
    --namespace "${NAMESPACE}" \
    --create-namespace \
    --set twitch.enabled=true \
    --set twitch.image.tag="${IMAGE_TAG}" \
    --set discord.enabled=false \
    --set keepalive.enabled=false \
    ${SECRETS_FLAG} \
    "$@"

echo ""
echo "Waiting for Twitch deployment to roll out..."
kubectl rollout status deployment/pedro-twitch -n "${NAMESPACE}" --timeout=120s

echo ""
echo "=== Twitch bot deployed ==="
kubectl get pods -n "${NAMESPACE}" -l app.kubernetes.io/component=twitch
echo ""
echo "Metrics: kubectl port-forward -n ${NAMESPACE} svc/pedro-twitch 6060:6060"
echo "Logs:    kubectl logs -n ${NAMESPACE} -l app.kubernetes.io/component=twitch -f"
