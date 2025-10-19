#!/bin/bash

set -e

# Configuration
TAG=${1:-$(git rev-parse --short HEAD)}
TARGET_HOST="100.81.89.62"
SERVICE=${2:-"both"}  # discord, twitch, or both

echo "Building and deploying Pedro containers with tag: $TAG"
echo "Target host: $TARGET_HOST"
echo "Service: $SERVICE"

# Build Discord container
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    echo "Building Discord container..."
    docker build -f cli/discord/discordBot.Dockerfile -t pedro-discord:$TAG .
    
    echo "Saving Discord container to tarball..."
    docker save pedro-discord:$TAG | gzip > pedro-discord-$TAG.tar.gz
fi

# Build Twitch container  
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "Building Twitch container..."
    docker build -f cli/twitch/twitchBot.Dockerfile -t pedro-twitch:$TAG .
    
    echo "Saving Twitch container to tarball..."
    docker save pedro-twitch:$TAG | gzip > pedro-twitch-$TAG.tar.gz
fi

# Create remote deployment script
cat > remote-deploy.sh << EOF
#!/bin/bash
set -e

TAG="$TAG"
SERVICE="$SERVICE"

echo "Loading containers on remote host..."

# Load Discord container if needed
if [[ "\$SERVICE" == "discord" || "\$SERVICE" == "both" ]]; then
    if [ -f pedro-discord-\$TAG.tar.gz ]; then
        echo "Loading Discord container..."
        docker load < pedro-discord-\$TAG.tar.gz
        
        # Deploy Discord service
        chmod +x deploy-discord.sh
        ./deploy-discord.sh \$TAG pedro-discord
    fi
fi

# Load Twitch container if needed
if [[ "\$SERVICE" == "twitch" || "\$SERVICE" == "both" ]]; then
    if [ -f pedro-twitch-\$TAG.tar.gz ]; then
        echo "Loading Twitch container..."
        docker load < pedro-twitch-\$TAG.tar.gz
        
        # Deploy Twitch service
        chmod +x deploy-twitch.sh
        ./deploy-twitch.sh \$TAG pedro-twitch
    fi
fi

echo "Deployment complete!"
echo "Check service status with:"
if [[ "\$SERVICE" == "discord" || "\$SERVICE" == "both" ]]; then
    echo "  sudo systemctl status pedro-discord"
    echo "  Discord metrics: http://localhost:6060/metrics"
fi
if [[ "\$SERVICE" == "twitch" || "\$SERVICE" == "both" ]]; then
    echo "  sudo systemctl status pedro-twitch"
    echo "  Twitch metrics: http://localhost:6061/metrics"
fi
EOF

chmod +x remote-deploy.sh

echo ""
echo "Build complete! Files created:"
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    echo "  - pedro-discord-$TAG.tar.gz"
fi
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    echo "  - pedro-twitch-$TAG.tar.gz"
fi
echo "  - remote-deploy.sh"
echo ""
echo "To deploy to $TARGET_HOST:"
echo "1. Copy files to remote host:"

COPY_CMD="scp"
if [[ "$SERVICE" == "discord" || "$SERVICE" == "both" ]]; then
    COPY_CMD="$COPY_CMD pedro-discord-$TAG.tar.gz"
fi
if [[ "$SERVICE" == "twitch" || "$SERVICE" == "both" ]]; then
    COPY_CMD="$COPY_CMD pedro-twitch-$TAG.tar.gz"
fi
COPY_CMD="$COPY_CMD deployment/deploy-*.sh remote-deploy.sh soypete@$TARGET_HOST:~/"

echo "   $COPY_CMD"
echo "2. SSH to remote host and run:"
echo "   ssh soypete@$TARGET_HOST"
echo "   ./remote-deploy.sh"