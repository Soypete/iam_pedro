#!/bin/bash

set -e

# Prometheus Docker deployment script for blue2
# This script sets up Prometheus in a Docker container

PROMETHEUS_VERSION="v2.54.1"
PROMETHEUS_DATA_DIR="$HOME/prometheus-data"
PROMETHEUS_CONFIG_DIR="$HOME/prometheus-config"

echo "Setting up Prometheus in Docker on blue2..."

# Create directories for persistent data and config
mkdir -p "$PROMETHEUS_DATA_DIR"
mkdir -p "$PROMETHEUS_CONFIG_DIR"

# Set proper permissions for Prometheus container (runs as user 65534:65534 or nobody)
# This allows the container to write to the data directory
echo "Setting permissions on data directory..."
sudo chown -R 65534:65534 "$PROMETHEUS_DATA_DIR"

# Copy prometheus.yml to the config directory
echo "Copying prometheus.yml to $PROMETHEUS_CONFIG_DIR/"
cp prometheus.yml "$PROMETHEUS_CONFIG_DIR/"

# Stop and remove existing Prometheus container if it exists
echo "Stopping existing Prometheus container (if any)..."
sudo docker stop prometheus 2>/dev/null || true
sudo docker rm prometheus 2>/dev/null || true

# Run Prometheus in Docker
echo "Starting Prometheus container..."
sudo docker run -d \
  --name prometheus \
  --restart unless-stopped \
  -p 9090:9090 \
  -v "$PROMETHEUS_CONFIG_DIR/prometheus.yml:/etc/prometheus/prometheus.yml" \
  -v "$PROMETHEUS_DATA_DIR:/prometheus" \
  prom/prometheus:$PROMETHEUS_VERSION \
  --config.file=/etc/prometheus/prometheus.yml \
  --storage.tsdb.path=/prometheus \
  --web.console.libraries=/usr/share/prometheus/console_libraries \
  --web.console.templates=/usr/share/prometheus/consoles

# Wait for container to start
sleep 3

# Check if container is running
if sudo docker ps | grep -q prometheus; then
  echo ""
  echo "Prometheus deployment complete!"
  echo "Web UI available at: http://blue2:9090 or http://<blue2-ip>:9090"
  echo ""
  echo "Useful commands:"
  echo "  View logs:    sudo docker logs -f prometheus"
  echo "  Restart:      sudo docker restart prometheus"
  echo "  Stop:         sudo docker stop prometheus"
  echo "  Start:        sudo docker start prometheus"
  echo "  Update config: Edit $PROMETHEUS_CONFIG_DIR/prometheus.yml then run: sudo docker restart prometheus"
  echo ""
  echo "Data stored in: $PROMETHEUS_DATA_DIR"
  echo "Config stored in: $PROMETHEUS_CONFIG_DIR"
else
  echo "ERROR: Prometheus container failed to start!"
  echo "Check logs with: sudo docker logs prometheus"
  exit 1
fi
