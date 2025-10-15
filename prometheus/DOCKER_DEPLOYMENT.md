# Prometheus Docker Deployment Guide for blue2

## Overview

This guide shows how to deploy Prometheus on blue2 (your homelab machine) using Docker to scrape metrics from:
- Discord Pedro bot on blue1 (100.81.89.62:6060)
- Twitch Pedro bot on blue1 (100.81.89.62:6061)
- vLLM service (100.79.52.122:8000)

## Prerequisites

- Docker installed on blue2
- Network connectivity from blue2 to:
  - blue1 (100.81.89.62) on ports 6060 and 6061
  - vLLM server (100.79.52.122) on port 8080

## Port Requirements

### Ports that need to be exposed/accessible:

1. **On blue2 (Prometheus host)**:
   - Port 9090: Prometheus web UI (accessible from your network)
   - This is exposed by the Docker container

2. **On blue1 (100.81.89.62)**:
   - Port 6060: Discord bot metrics endpoint (already exposed)
   - Port 6061: Twitch bot metrics endpoint (already exposed)
   - These are already configured in the Go bots

3. **On vLLM server (100.79.52.122)**:
   - Port 8000: vLLM metrics endpoint at /metrics (enabled by default)
   - No additional configuration needed for vLLM

## Deployment Steps

### 1. Transfer files to blue2

From your local machine:

```bash
# Copy the prometheus directory to blue2
scp -r prometheus/ user@blue2:~/
```

Replace `user@blue2` with your actual username and hostname/IP.

### 2. SSH into blue2

```bash
ssh user@blue2
cd ~/prometheus
```

### 3. Review and customize prometheus.yml (optional)

The `prometheus.yml` is already configured with the correct targets. You can review it:

```bash
cat prometheus.yml
```

If you need to change scrape intervals or add additional targets, edit the file before deploying.

### 4. Make the deployment script executable

```bash
chmod +x deploy-prometheus-docker.sh
```

### 5. Deploy Prometheus

```bash
./deploy-prometheus-docker.sh
```

This script will:
- Create directories for Prometheus data and config
- Copy prometheus.yml to the config directory
- Stop any existing Prometheus container
- Start a new Prometheus container with the updated config

### 6. Verify deployment

Check the Prometheus web UI:

```bash
# Get blue2's IP address
ip addr show | grep "inet "

# Then visit in your browser:
# http://<blue2-ip>:9090
```

Or if you're on the same network, you might be able to use:
```
http://blue2:9090
```

### 7. Verify targets are being scraped

In the Prometheus web UI:
1. Navigate to Status > Targets
2. You should see all jobs (prometheus, pedro-discord, pedro-twitch, pedro-vllm) listed
3. State should be "UP" for each target if everything is configured correctly

## Troubleshooting

### Check if Prometheus container is running

```bash
docker ps | grep prometheus
```

### View Prometheus logs

```bash
docker logs -f prometheus
```

### Test connectivity to targets from blue2

```bash
# Test Discord bot metrics
curl http://100.81.89.62:6060/metrics

# Test Twitch bot metrics
curl http://100.81.89.62:6061/metrics

# Test vLLM metrics
curl http://100.79.52.122:8000/metrics
```

If any of these fail, you have a network connectivity issue.

### Restart Prometheus after config changes

```bash
# Edit the config
nano ~/prometheus-config/prometheus.yml

# Restart the container to pick up changes
docker restart prometheus
```

### Stop Prometheus

```bash
docker stop prometheus
```

### Start Prometheus

```bash
docker start prometheus
```

### Remove Prometheus (clean slate)

```bash
docker stop prometheus
docker rm prometheus
# Data is preserved in ~/prometheus-data and ~/prometheus-config
```

## Network Firewall Notes

If targets show as "DOWN" in Prometheus:

1. **Check firewall on blue1** - Ensure ports 6060 and 6061 are accessible from blue2:
   ```bash
   # On blue1, check if ports are listening
   sudo ss -tlnp | grep -E ':(6060|6061)'

   # If using UFW firewall on blue1
   sudo ufw allow 6060/tcp
   sudo ufw allow 6061/tcp
   ```

2. **Check firewall on vLLM server (100.79.52.122)** - Ensure port 8000 is accessible:
   ```bash
   # On the vLLM server
   sudo ss -tlnp | grep :8000
   ```

3. **Check firewall on blue2** - Ensure port 9090 is accessible for the web UI:
   ```bash
   # On blue2
   sudo ufw allow 9090/tcp
   ```

## Data Persistence

Prometheus data is stored in `~/prometheus-data` on blue2. This directory persists between container restarts and recreations.

To clear old data:
```bash
docker stop prometheus
rm -rf ~/prometheus-data/*
docker start prometheus
```

## Updating Prometheus

To update to a newer Prometheus version:

1. Edit `deploy-prometheus-docker.sh`
2. Change `PROMETHEUS_VERSION` to the desired version
3. Run `./deploy-prometheus-docker.sh` again

The script will automatically pull the new image and recreate the container.
