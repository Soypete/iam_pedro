#!/bin/bash

set -e

# Prometheus setup script for 100.125.196.1
PROMETHEUS_VERSION="2.45.0"
PROMETHEUS_USER="prometheus"
PROMETHEUS_HOME="/opt/prometheus"
CONFIG_DIR="/etc/prometheus"

echo "Setting up Prometheus server..."

# Create prometheus user
sudo useradd --no-create-home --shell /bin/false $PROMETHEUS_USER || true

# Create directories
sudo mkdir -p $PROMETHEUS_HOME
sudo mkdir -p $CONFIG_DIR
sudo mkdir -p /var/lib/prometheus

# Download and install Prometheus
cd /tmp
wget https://github.com/prometheus/prometheus/releases/download/v$PROMETHEUS_VERSION/prometheus-$PROMETHEUS_VERSION.linux-amd64.tar.gz
tar xvf prometheus-$PROMETHEUS_VERSION.linux-amd64.tar.gz
cd prometheus-$PROMETHEUS_VERSION.linux-amd64

# Copy binaries
sudo cp prometheus promtool $PROMETHEUS_HOME/
sudo cp -r consoles console_libraries $PROMETHEUS_HOME/

# Set ownership
sudo chown -R $PROMETHEUS_USER:$PROMETHEUS_USER $PROMETHEUS_HOME
sudo chown -R $PROMETHEUS_USER:$PROMETHEUS_USER $CONFIG_DIR
sudo chown -R $PROMETHEUS_USER:$PROMETHEUS_USER /var/lib/prometheus

# Copy configuration
sudo cp prometheus.yml $CONFIG_DIR/prometheus.yml
sudo chown $PROMETHEUS_USER:$PROMETHEUS_USER $CONFIG_DIR/prometheus.yml

# Create systemd service
sudo tee /etc/systemd/system/prometheus.service > /dev/null <<EOF
[Unit]
Description=Prometheus Server
Wants=network-online.target
After=network-online.target

[Service]
User=$PROMETHEUS_USER
Group=$PROMETHEUS_USER
Type=simple
ExecStart=$PROMETHEUS_HOME/prometheus \\
    --config.file=$CONFIG_DIR/prometheus.yml \\
    --storage.tsdb.path=/var/lib/prometheus/ \\
    --web.console.templates=$PROMETHEUS_HOME/consoles \\
    --web.console.libraries=$PROMETHEUS_HOME/console_libraries \\
    --web.listen-address=0.0.0.0:9090 \\
    --web.external-url=http://100.125.196.1:9090

[Install]
WantedBy=multi-user.target
EOF

# Start and enable Prometheus
sudo systemctl daemon-reload
sudo systemctl enable prometheus
sudo systemctl start prometheus

# Wait and check status
sleep 5
sudo systemctl status prometheus --no-pager

echo ""
echo "Prometheus setup complete!"
echo "Web UI available at: http://100.125.196.1:9090"
echo "Configuration file: $CONFIG_DIR/prometheus.yml"
echo "Data directory: /var/lib/prometheus"
echo ""
echo "To check logs: sudo journalctl -u prometheus -f"
echo "To restart: sudo systemctl restart prometheus"