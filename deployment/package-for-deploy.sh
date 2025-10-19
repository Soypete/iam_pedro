#!/bin/bash

set -e

# Package Pedro deployment for Taildrop transfer
# Creates a tarball with everything needed to deploy on remote machine

TAG=${1:-$(git rev-parse --short HEAD)}
OUTPUT_DIR="pedro-deploy-$TAG"
TARBALL="pedro-deploy-$TAG.tar.gz"

echo "=== Packaging Pedro Deployment ==="
echo "Tag: $TAG"
echo "Output: $TARBALL"
echo ""

# Clean up old package
rm -rf "$OUTPUT_DIR" "$TARBALL"

# Create package directory
mkdir -p "$OUTPUT_DIR"

# Copy deployment files
echo "Copying deployment files..."
cp deployment/deploy-docker.sh "$OUTPUT_DIR/"
cp deployment/README.md "$OUTPUT_DIR/"
cp deployment/DEPLOYMENT_FIXES.md "$OUTPUT_DIR/"

# Copy Dockerfiles and code
echo "Copying application code..."
cp -r cli "$OUTPUT_DIR/"
cp -r ai "$OUTPUT_DIR/"
cp -r database "$OUTPUT_DIR/"
cp -r discord "$OUTPUT_DIR/"
cp -r twitch "$OUTPUT_DIR/"
cp -r logging "$OUTPUT_DIR/"
cp -r metrics "$OUTPUT_DIR/"
cp -r types "$OUTPUT_DIR/"
cp go.mod go.sum "$OUTPUT_DIR/"

# Create setup script
cat > "$OUTPUT_DIR/setup.sh" << 'EOF'
#!/bin/bash

set -e

echo "=== Pedro Bot Setup ==="
echo ""

# Check for Docker/Podman
if ! command -v docker &> /dev/null; then
    echo "Docker/Podman not found. Attempting to install Podman..."

    if command -v apt-get &> /dev/null; then
        sudo apt-get update
        sudo apt-get install -y podman
        sudo ln -sf /usr/bin/podman /usr/bin/docker
    elif command -v dnf &> /dev/null; then
        sudo dnf install -y podman
        sudo ln -sf /usr/bin/podman /usr/bin/docker
    else
        echo "ERROR: Could not install Podman. Please install Docker or Podman manually."
        exit 1
    fi
fi

# Enable Podman auto-start
if command -v podman &> /dev/null; then
    echo "Enabling Podman auto-start..."
    sudo systemctl enable --now podman.socket 2>/dev/null || true
    sudo systemctl enable --now podman-restart.service 2>/dev/null || true
fi

# Configure registries for Podman
if [ -f /etc/containers/registries.conf ]; then
    if ! grep -q "unqualified-search-registries.*docker.io" /etc/containers/registries.conf; then
        echo "Configuring container registries..."
        sudo sed -i '1i unqualified-search-registries = ["docker.io"]' /etc/containers/registries.conf
    fi
fi

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Configure environment files:"
echo "   sudo nano /opt/pedro/service.env"
echo "   sudo nano /opt/pedro/prod.env"
echo ""
echo "2. Deploy bots:"
echo "   ./deploy-docker.sh discord    # Deploy Discord bot"
echo "   ./deploy-docker.sh twitch     # Deploy Twitch bot"
echo "   ./deploy-docker.sh both       # Deploy both bots"
echo ""
echo "See README.md for detailed configuration instructions."
EOF

chmod +x "$OUTPUT_DIR/setup.sh"
chmod +x "$OUTPUT_DIR/deploy-docker.sh"

# Create README for package
cat > "$OUTPUT_DIR/QUICK_START.md" << 'EOF'
# Pedro Bot Quick Start

## 1. Run Setup

```bash
./setup.sh
```

This installs Docker/Podman and configures the system.

## 2. Configure Secrets

Edit the configuration files:

```bash
sudo nano /opt/pedro/service.env
sudo nano /opt/pedro/prod.env
```

See `README.md` for detailed configuration instructions.

## 3. Deploy

```bash
# Deploy Discord bot
./deploy-docker.sh discord

# Deploy Twitch bot
./deploy-docker.sh twitch

# Deploy both
./deploy-docker.sh both
```

## 4. Monitor

```bash
# View running containers
docker ps

# View logs
docker logs -f pedro-discord
docker logs -f pedro-twitch

# Check metrics
curl http://localhost:6060/metrics  # Discord
curl http://localhost:6061/metrics  # Twitch
```

## Troubleshooting

See `DEPLOYMENT_FIXES.md` for common issues and solutions.
EOF

# Create tarball
echo "Creating tarball..."
tar czf "$TARBALL" "$OUTPUT_DIR"

# Cleanup
rm -rf "$OUTPUT_DIR"

echo ""
echo "✅ Package created: $TARBALL"
echo ""
echo "Transfer to remote machine via Taildrop:"
echo "1. Open Tailscale on remote machine"
echo "2. Drag $TARBALL to remote machine in Taildrop"
echo "3. On remote machine:"
echo "   tar xzf $TARBALL"
echo "   cd $OUTPUT_DIR"
echo "   ./setup.sh"
echo "   ./deploy-docker.sh both"
