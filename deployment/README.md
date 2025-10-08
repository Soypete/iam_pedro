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

The deployment uses two configuration files:
1. **`/opt/pedro/service.env`** - Non-secret systemd environment variables
2. **`/opt/pedro/prod.env`** - 1Password secret references (injected at container runtime)

### Step 1: Create service.env (Non-Secrets)

This file contains the 1Password service account token and non-secret values:

```bash
sudo mkdir -p /opt/pedro
sudo tee /opt/pedro/service.env > /dev/null <<EOF
# 1Password Service Account Token (required for op CLI in container)
OP_SERVICE_ACCOUNT_TOKEN=ops_your_service_account_token_here

# Twitch Client ID (not a secret, safe to store in plain text)
TWITCH_ID=your_twitch_client_id

# OAuth redirect host for remote authentication
# Use Tailscale hostname for remote access, or IP address
OAUTH_REDIRECT_HOST=100.81.89.62:3000
EOF

sudo chmod 600 /opt/pedro/service.env
```

### Step 2: Create prod.env (1Password Secret References)

This file uses 1Password secret references that will be injected at container runtime:

```bash
sudo tee /opt/pedro/prod.env > /dev/null <<EOF
# Discord Bot
DISCORD_TOKEN=op://vault/discord-bot/token

# Twitch Bot - use TWITCH_SECRET for OAuth flow
TWITCH_SECRET=op://vault/twitch-bot/client-secret

# Twitch Bot - OPTIONAL: use TWITCH_TOKEN to skip OAuth flow
# If set, bot will use this token directly instead of running OAuth
# TWITCH_TOKEN=op://vault/twitch-bot/access-token

# Database
DATABASE_URL=op://vault/postgres/connection-url

# LLM Service
LLAMA_CPP_PATH=https://pedro-gpu.tail6fbc5.ts.net
EOF

sudo chmod 600 /opt/pedro/prod.env
```

**Important Notes:**
- The containers use `op run --env-file=/app/prod.env` to inject secrets at runtime
- `OP_SERVICE_ACCOUNT_TOKEN` must be set in `service.env` for the container's `op` CLI to authenticate
- `TWITCH_ID` is not a secret and can be stored as plain text
- For Twitch: either use `TWITCH_SECRET` (requires OAuth) or `TWITCH_TOKEN` (pre-generated token)

## Twitch OAuth Setup

The Twitch bot supports two authentication methods:

### Method 1: Pre-generated Token (Recommended for Production)

If you have a `TWITCH_TOKEN` in your 1Password vault, the bot will use it directly and skip the OAuth flow. This is best for headless deployments.

To get a token initially, use Method 2 once, then save the token to 1Password.

### Method 2: Remote OAuth Flow (For Initial Setup)

If `TWITCH_TOKEN` is not set, the bot will initiate an OAuth flow on startup.

**Prerequisites:**
1. Update Twitch Developer Portal with your redirect URL:
   - Go to https://dev.twitch.tv/console/apps
   - Edit your application
   - Add OAuth Redirect URL: `http://100.81.89.62:3000/oauth/redirect` (or your Tailscale hostname)
   - Save changes

**During First Deployment:**
1. Start the Twitch bot service
2. Watch the logs for the OAuth URL:
   ```bash
   sudo journalctl -u pedro-twitch -f
   ```
3. You'll see output like:
   ```
   Visit the URL for the auth dialog: https://id.twitch.tv/oauth2/authorize?...
   OAuth redirect configured for: http://100.81.89.62:3000/oauth/redirect
   ```
4. Open that URL in a browser (from any device on the network/Tailscale)
5. Authorize the application
6. The bot will receive the token and print:
   ```
   Token received: abc123...
   IMPORTANT: Save this token to 1Password as TWITCH_TOKEN to avoid OAuth flow on restart
   ```
7. Save the token to 1Password:
   ```bash
   op item create --category=password --title="twitch-bot" \
     --vault=vault \
     access-token=<the_token_from_logs>
   ```
8. Update `/opt/pedro/prod.env` to uncomment `TWITCH_TOKEN`:
   ```bash
   TWITCH_TOKEN=op://vault/twitch-bot/access-token
   ```
9. Restart the service - it will now use the saved token

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