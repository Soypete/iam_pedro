# Pedro Deployment Guide

This guide covers deploying Pedro's Discord and Twitch bots with Prometheus monitoring.

## Architecture

- **Bot Host**: 100.81.89.62 (Discord + Twitch containers)
- **Monitoring Host**: 100.125.196.1 (Prometheus)
- **LLM Service**: pedro-gpu.tail6fbc5.ts.net (existing)

## Quick Start

### 1. Deploy Discord Bot to 100.81.89.62

```bash
# Copy repo to target host
git clone <repo-url>
cd iam_pedro

# Build and deploy Discord bot
./deployment/remote-build-deploy.sh $(git rev-parse --short HEAD) discord

# Edit environment file
sudo nano /opt/pedro/prod.env

# Restart service after configuration
sudo systemctl restart pedro-discord
```

### 2. Deploy Twitch Bot (Optional)

```bash
# Deploy Twitch bot on same host (different port)
./deployment/remote-build-deploy.sh $(git rev-parse --short HEAD) twitch

# Restart service after configuration  
sudo systemctl restart pedro-twitch
```

### 3. Set up Prometheus on 100.125.196.1

```bash
# Copy prometheus files to monitoring host
scp -r prometheus/ soypete@100.125.196.1:~/

# SSH to monitoring host and setup
ssh soypete@100.125.196.1
cd prometheus/
chmod +x setup-prometheus.sh
./setup-prometheus.sh
```

## Environment Configuration

### Option 1: Using 1Password Service Account (Recommended)

The Docker containers use 1Password CLI (`op`) for secrets management. Set up a 1Password Service Account on the VM:

```bash
# Install 1Password CLI
curl -sSfo op.zip https://cache.agilebits.com/dist/1P/op2/pkg/v2.30.0/op_linux_amd64_v2.30.0.zip
unzip op.zip
sudo mv op /usr/local/bin/
sudo chmod +x /usr/local/bin/op

# Authenticate with Service Account token
export OP_SERVICE_ACCOUNT_TOKEN="your_service_account_token_here"
echo 'export OP_SERVICE_ACCOUNT_TOKEN="your_service_account_token_here"' >> ~/.bashrc

# Verify connection
op vault list

# Create the prod.env file with 1Password secret references
sudo mkdir -p /opt/pedro
sudo tee /opt/pedro/prod.env > /dev/null <<EOF
# Reference format: op://<vault>/<item>/<field>
DISCORD_TOKEN=op://vault/discord-bot/token
TWITCH_TOKEN=op://vault/twitch-bot/token
TWITCH_CHANNEL=op://vault/twitch-bot/channel
DATABASE_URL=op://vault/postgres/connection-url
OPENAI_API_KEY=op://vault/openai/api-key
EOF
```

**Note:** The containers use `op run --env-file prod.env` to inject secrets at runtime. Make sure your 1Password vault contains the referenced items.

### Option 2: Plain Environment Variables

Alternatively, create `/opt/pedro/prod.env` with plain values:

```bash
sudo tee /opt/pedro/prod.env > /dev/null <<EOF
DISCORD_TOKEN=your_discord_bot_token
TWITCH_TOKEN=your_twitch_oauth_token
TWITCH_CHANNEL=your_twitch_channel_name
DATABASE_URL=postgresql://user:pass@host:port/dbname
OPENAI_API_KEY=your_openai_api_key
EOF

sudo chmod 600 /opt/pedro/prod.env
```

**Note:** If using plain values, you'll need to modify the Dockerfile CMD to remove the `op run --` wrapper.

## Service Management

### Check Status
```bash
# Discord bot
sudo systemctl status pedro-discord

# Twitch bot  
sudo systemctl status pedro-twitch

# Prometheus
sudo systemctl status prometheus
```

### View Logs
```bash
# Discord logs
sudo journalctl -u pedro-discord -f

# Twitch logs
sudo journalctl -u pedro-twitch -f

# Prometheus logs
sudo journalctl -u prometheus -f
```

### Restart Services
```bash
# After config changes
sudo systemctl restart pedro-discord
sudo systemctl restart pedro-twitch
sudo systemctl restart prometheus
```

## Monitoring Endpoints

| Service | Host | Port | URL |
|---------|------|------|-----|
| Discord Bot | 100.81.89.62 | 6060 | http://100.81.89.62:6060/metrics |
| Twitch Bot | 100.81.89.62 | 6061 | http://100.81.89.62:6061/metrics |
| Prometheus | 100.125.196.1 | 9090 | http://100.125.196.1:9090 |
| LLM Service | pedro-gpu.tail6fbc5.ts.net | 443 | https://pedro-gpu.tail6fbc5.ts.net/metrics |

## Available Metrics

The bots expose these metrics:
- `twitch_connection_count` - Twitch connections established
- `twitch_message_recieved_count` - Messages received from Twitch
- `twitch_message_sent_count` - Messages sent to Twitch  
- `discord_message_recieved` - Messages received from Discord
- `discord_message_sent` - Messages sent to Discord
- `empty_llm_response` - Empty responses from LLM
- `successfull_llm_gen` - Successful LLM generations
- `failed_llm_gen` - Failed LLM generations

## Troubleshooting

### Container Issues
```bash
# Check if containers are running
docker ps

# Check container logs
docker logs pedro-discord
docker logs pedro-twitch

# Rebuild container
./deployment/remote-build-deploy.sh new-tag discord
```

### Network Issues
```bash
# Test metrics endpoints
curl http://localhost:6060/metrics
curl http://localhost:6061/metrics

# Check if ports are listening
sudo netstat -tlnp | grep -E '(6060|6061|9090)'
```

### Environment Issues
```bash
# Verify environment file
sudo cat /opt/pedro/prod.env

# Test 1Password connection (if using)
op whoami
```

## Security Notes

- Ensure firewall rules allow access to required ports
- Keep environment variables secure and never commit to version control
- Use 1Password or similar secrets management for production
- Consider setting up reverse proxy with TLS for external access