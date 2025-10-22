#!/bin/bash

set -e

# Grafana deployment script for Pedro metrics monitoring
# Deploys standalone Grafana container on blue2 (100.125.196.1)

echo "=== Grafana Deployment ==="
echo ""

# Configuration file paths
SERVICE_ENV="/opt/pedro/service.env"
GRAFANA_ENV="/opt/pedro/grafana.env"
PROVISIONING_DIR="/opt/pedro/grafana-provisioning"

# Check prerequisites
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker not found. Please install docker."
    exit 1
fi

# Check for 1Password CLI
if ! command -v op &> /dev/null; then
    echo "ERROR: 1Password CLI (op) not found. Please install it."
    echo "See: https://developer.1password.com/docs/cli/get-started/"
    exit 1
fi

# Ensure config directory exists
sudo mkdir -p /opt/pedro

# Load service account token
if [ -f "$SERVICE_ENV" ]; then
    echo "Loading 1Password service account token..."
    source "$SERVICE_ENV"
    export OP_SERVICE_ACCOUNT_TOKEN
else
    echo "ERROR: $SERVICE_ENV not found!"
    echo "This file should contain your OP_SERVICE_ACCOUNT_TOKEN"
    exit 1
fi

# Check for grafana.env
if [ ! -f "$GRAFANA_ENV" ]; then
    echo "Creating template $GRAFANA_ENV..."
    sudo tee "$GRAFANA_ENV" > /dev/null <<'EOF'
# Grafana Admin Password (1Password reference)
GF_SECURITY_ADMIN_PASSWORD=op://pedro/k8s-grafana-password/password
EOF
    echo "✅ Created $GRAFANA_ENV"
    echo ""
fi

# Copy provisioning configuration if available
if [ -d "config/grafana/provisioning" ]; then
    echo "Copying Grafana provisioning configuration..."
    sudo mkdir -p "$PROVISIONING_DIR"
    sudo cp -r config/grafana/provisioning/* "$PROVISIONING_DIR/"
    echo "✅ Provisioning configuration copied to $PROVISIONING_DIR"
    echo ""
fi

# Stop and remove existing container
echo "Stopping existing Grafana container..."
docker stop grafana 2>/dev/null || true
docker rm grafana 2>/dev/null || true

# Create volume for Grafana data persistence
echo "Creating Grafana data volume..."
docker volume create grafana-data 2>/dev/null || true

# Load the password from 1Password reference
echo "Loading Grafana admin password from 1Password..."
source "$GRAFANA_ENV"

# Resolve the 1Password reference using op CLI
ADMIN_PASSWORD=$(op read "$GF_SECURITY_ADMIN_PASSWORD")

if [ -z "$ADMIN_PASSWORD" ]; then
    echo "ERROR: Failed to read password from 1Password"
    echo "Reference: $GF_SECURITY_ADMIN_PASSWORD"
    exit 1
fi

echo "✅ Successfully retrieved password from 1Password"

# Run Grafana container
echo "Starting Grafana container..."

# Build docker run command with conditional provisioning mount
DOCKER_CMD="docker run -d \
    --name grafana \
    --restart unless-stopped \
    -p 3000:3000 \
    -v grafana-data:/var/lib/grafana"

# Add provisioning mount if directory exists
if [ -d "$PROVISIONING_DIR" ]; then
    DOCKER_CMD="$DOCKER_CMD \
    -v $PROVISIONING_DIR:/etc/grafana/provisioning"
    echo "✅ Mounting provisioning configuration"
fi

DOCKER_CMD="$DOCKER_CMD \
    -e GF_SECURITY_ADMIN_USER=admin \
    -e GF_SECURITY_ADMIN_PASSWORD=$ADMIN_PASSWORD \
    -e GF_SERVER_ROOT_URL=http://100.125.196.1:3000 \
    grafana/grafana:latest"

eval $DOCKER_CMD

echo ""
echo "✅ Grafana deployed!"
echo "   Container: grafana"
echo "   URL: http://100.125.196.1:3000"
echo "   Username: admin"
echo "   Password: (from 1Password: op://pedro/k8s-grafana-password/password)"
echo ""

if [ -d "$PROVISIONING_DIR" ]; then
    echo "✅ Auto-provisioned resources:"
    echo "   - Prometheus datasource (http://100.125.196.1:9090)"
    echo "   - vLLM Performance Dashboard"
    echo "   - Twitch Bot Metrics Dashboard"
    echo "   - Discord Bot Metrics Dashboard"
    echo "   - Alert rules (requires contact point setup)"
    echo ""
    echo "Next steps:"
    echo "  1. Access Grafana at http://100.125.196.1:3000"
    echo "  2. Check 'Pedro Bot' folder for pre-loaded dashboards"
    echo "  3. (Optional) Configure Discord webhook for alerts"
    echo "     See: config/grafana/ALERTS-README.md"
else
    echo "Next steps:"
    echo "  1. Access Grafana at http://100.125.196.1:3000"
    echo "  2. Add Prometheus data source: http://100.125.196.1:9090"
    echo "  3. Import dashboards from config/grafana/"
    echo "     - grafana-vllm-dashboard.json"
    echo "     - grafana-twitch-dashboard.json"
    echo "     - grafana-discord-dashboard.json"
    echo "  4. (Optional) Import alert rules: grafana-alert-rules.yaml"
fi
echo ""
echo "Manage container:"
echo "  docker logs -f grafana          # View logs"
echo "  docker stop grafana             # Stop container"
echo "  docker start grafana            # Start container"
echo "  docker restart grafana          # Restart container"
echo ""
echo "Container will automatically restart on failure and system reboot."
