# Pedro Deployment Guide

Both bots run as pods in a k3s (Kubernetes) cluster managed via the `pedro-ops` repo.
Images are pushed to the ZOT registry at `100.81.89.62:5000`. K8s manifests live in `pedro-ops/scripts/k8s/`.

## Architecture

- **k3s Cluster**: blue1 (control plane, 100.81.89.62), blue2 (worker, 100.125.196.1), refurb (worker, 100.70.90.12)
- **Image Registry**: ZOT at `100.81.89.62:5000`
- **LLM Service**: `pedro-gpu.tail6fbc5.ts.net`
- **Cluster status**: `foundry status`

## Building and Deploying

```bash
TAG=$(git rev-parse --short HEAD)

# Build and push to ZOT registry
podman build -f cli/twitch/twitchBot.Dockerfile -t 100.81.89.62:5000/pedro-twitch:$TAG .
podman push 100.81.89.62:5000/pedro-twitch:$TAG

podman build -f cli/discord/discordBot.Dockerfile -t 100.81.89.62:5000/pedro-discord:$TAG .
podman push 100.81.89.62:5000/pedro-discord:$TAG

# Apply k8s manifests (from pedro-ops repo)
kubectl apply -f scripts/k8s/chatbot/
```

## Twitch OAuth Setup

The Twitch bot supports two authentication methods:

### Method 1: Pre-generated Token (Recommended for Production)

If `TWITCH_TOKEN` is set in the k8s Secret/1Password, the bot uses it directly and skips the OAuth flow.

To get a token initially, use Method 2 once, then save it to 1Password.

### Method 2: OAuth Flow (For Initial Token Generation)

If `TWITCH_TOKEN` is not set, the bot initiates an OAuth flow on startup.

**OAuth redirect URL** (already registered in Twitch Developer Console):
```
https://chatbot-pedro-twitch-auth-ingress.tail6fc5.ts.net/oauth/redirect
```

This is a Tailscale ingress (`tag:k8s`, 100.77.49.108) created by the Tailscale k8s operator.
The k8s manifest is in `pedro-ops/scripts/k8s/chatbot/pedro-twitch-auth-ingress.yaml`.

The `OAUTH_REDIRECT_HOST` env var in the pod must be set to:
```
chatbot-pedro-twitch-auth-ingress.tail6fc5.ts.net
```

**During OAuth flow:**
1. Watch pod logs for the auth URL:
   ```bash
   kubectl logs -n chatbot -l app=pedro-twitch -f
   ```
2. You'll see:
   ```
   Visit the URL for the auth dialog: https://id.twitch.tv/oauth2/authorize?...
   OAuth redirect configured for: https://chatbot-pedro-twitch-auth-ingress.tail6fc5.ts.net/oauth/redirect
   ```
3. Open the URL, authorize, then save the printed token to 1Password as `TWITCH_TOKEN`.
4. Update the k8s Secret and restart the pod.

## Pod Management

```bash
# Check status
kubectl get pods -n chatbot

# View logs
kubectl logs -n chatbot -l app=pedro-twitch -f
kubectl logs -n chatbot -l app=pedro-discord -f

# Restart a pod
kubectl rollout restart deployment/pedro-twitch -n chatbot
kubectl rollout restart deployment/pedro-discord -n chatbot
```

## Monitoring Endpoints

| Service | URL |
|---------|-----|
| Discord Bot metrics | http://100.81.89.62:6060/metrics |
| Twitch Bot metrics | http://100.81.89.62:6061/metrics |
| Prometheus | http://100.125.196.1:9090 |
| LLM Service | https://pedro-gpu.tail6fbc5.ts.net/metrics |

## Available Metrics

- `twitch_connection_count` - Twitch connections established
- `twitch_message_recieved_count` - Messages received from Twitch
- `twitch_message_sent_count` - Messages sent to Twitch
- `discord_message_recieved` - Messages received from Discord
- `discord_message_sent` - Messages sent to Discord
- `empty_llm_response_count` - Empty responses from LLM
- `successful_llm_gen_count` - Successful LLM generations
- `failed_llm_gen_count` - Failed LLM generations

## Security Notes

- Secrets managed via 1Password, injected at runtime via `op run`
- Never commit secrets to version control
