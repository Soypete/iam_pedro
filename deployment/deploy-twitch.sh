#!/bin/bash

set -e

# Default values
TAG=${1:-latest}
CONTAINER_NAME="pedro-twitch"
SERVICE_NAME="pedro-twitch"
IMAGE_NAME=${2:-pedro-twitch}  # Allow custom image name, default to local build
METRICS_PORT=6061  # Different port from Discord to avoid conflicts
TARGET_HOST="100.81.89.62"

echo "Deploying Pedro Twitch Bot with tag: $TAG"

# Create the systemd service file
cat > pedro-twitch.service << EOF
[Unit]
Description=Pedro Twitch Bot
After=docker.service
Requires=docker.service

[Service]
Type=forking
Restart=always
RestartSec=10
EnvironmentFile=/opt/pedro/service.env
ExecStartPre=-/usr/bin/docker stop $CONTAINER_NAME
ExecStartPre=-/usr/bin/docker rm $CONTAINER_NAME
ExecStartPre=-/usr/bin/docker pull $IMAGE_NAME:$TAG
ExecStart=/usr/bin/docker run -d \\
    --name $CONTAINER_NAME \\
    --restart unless-stopped \\
    -p $METRICS_PORT:6060 \\
    -p 3000:3000 \\
    -v /opt/pedro/prod.env:/app/prod.env:ro \\
    -e OP_SERVICE_ACCOUNT_TOKEN=\${OP_SERVICE_ACCOUNT_TOKEN} \\
    -e TWITCH_ID=\${TWITCH_ID} \\
    -e OAUTH_REDIRECT_HOST=\${OAUTH_REDIRECT_HOST} \\
    $IMAGE_NAME:$TAG
ExecStop=/usr/bin/docker stop $CONTAINER_NAME
ExecStopPost=/usr/bin/docker rm $CONTAINER_NAME

[Install]
WantedBy=multi-user.target
EOF

# Create deployment script
cat > deploy.sh << 'EOF'
#!/bin/bash
set -e

# Ensure directories exist
sudo mkdir -p /opt/pedro

# Check environment files exist
if [ ! -f /opt/pedro/prod.env ]; then
    echo "Warning: /opt/pedro/prod.env does not exist. Please create it with your 1Password secret references."
    echo "Example variables needed:"
    echo "TWITCH_SECRET=op://vault/twitch-bot/client-secret"
    echo "TWITCH_TOKEN=op://vault/twitch-bot/access-token"
    echo "DATABASE_URL=op://vault/postgres/connection-url"
    echo "LLAMA_CPP_PATH=http://pedro-gpu.tail6fbc5.ts.net"
    exit 1
fi

if [ ! -f /opt/pedro/service.env ]; then
    echo "Warning: /opt/pedro/service.env does not exist. Creating template..."
    sudo tee /opt/pedro/service.env > /dev/null <<ENVEOF
# 1Password Service Account Token (required for op CLI)
OP_SERVICE_ACCOUNT_TOKEN=your_service_account_token_here

# Twitch Client ID (not a secret, can be plain text)
TWITCH_ID=your_twitch_client_id

# OAuth redirect host (use Tailscale hostname or IP for remote OAuth)
# Options: localhost:3000, 100.81.89.62:3000, or blue1.tail6fbc5.ts.net:3000
OAUTH_REDIRECT_HOST=100.81.89.62:3000
ENVEOF
    echo "ERROR: Please edit /opt/pedro/service.env with actual values!"
    exit 1
fi

# Copy service file
sudo cp pedro-twitch.service /etc/systemd/system/

# Reload systemd and start service
sudo systemctl daemon-reload
sudo systemctl enable pedro-twitch
sudo systemctl restart pedro-twitch

# Check status
sleep 5
sudo systemctl status pedro-twitch --no-pager

echo "Twitch bot deployed successfully!"
echo "Metrics available at: http://localhost:$METRICS_PORT/metrics"
echo "Check logs with: sudo journalctl -u pedro-twitch -f"
EOF

chmod +x deploy.sh

echo "Deployment files created:"
echo "1. pedro-twitch.service - systemd service definition"
echo "2. deploy.sh - deployment script"
echo ""
echo "To deploy to $TARGET_HOST:"
echo "1. Copy these files to the target machine"
echo "2. Ensure /opt/pedro/prod.env exists with required environment variables"
echo "3. Run: ./deploy.sh"
echo ""
echo "The service will be available at http://$TARGET_HOST:$METRICS_PORT/metrics"