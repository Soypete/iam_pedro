# Pedro Deployment Guide

Pedro's bots (Discord, Twitch, Keepalive) run as pods in a k3s cluster administered via the `foundry` CLI.

## Cluster Architecture

| Node | Role | Tailscale IP |
|------|------|-------------|
| blue1 | control-plane + ZOT registry | `100.81.89.62` |
| blue2 | worker + Prometheus | `100.125.196.1` |
| refurb | worker | `100.70.90.12` |

- **ZOT Registry**: `100.81.89.62:5000` (HTTP, no auth)
- **LLM Service**: `http://100.121.229.114:8080` (llama.cpp / vLLM)
- **Namespace**: `chatbot`
- **Helm Release**: `pedro`
- **Helm Chart**: `charts/pedro-bots/`

## Prerequisites

- `kubectl` configured for the cluster
- `helm` installed
- `op` (1Password CLI) authenticated (`op whoami`)
- `podman` for building images (`podman machine start`)
- `foundry` CLI for cluster administration

## Deploying

### 1. Build and Push Images

Migrations are embedded in the binary — a new migration requires a new image build.

```bash
TAG=$(git rev-parse --short HEAD)

for SERVICE in discord twitch keepalive; do
    podman build --platform linux/amd64 \
        -f cli/${SERVICE}/${SERVICE}Bot.Dockerfile \
        -t 100.81.89.62:5000/pedro-${SERVICE}:${TAG} .
    podman push --tls-verify=false 100.81.89.62:5000/pedro-${SERVICE}:${TAG}

    podman tag 100.81.89.62:5000/pedro-${SERVICE}:${TAG} \
               100.81.89.62:5000/pedro-${SERVICE}:latest
    podman push --tls-verify=false 100.81.89.62:5000/pedro-${SERVICE}:latest
done
```

> **Note**: The ZOT registry must be running on blue1.
> Verify: `ssh 100.81.89.62 "docker ps | grep zot"`

### 2. Resolve Secrets from 1Password

```bash
cat > /tmp/pedro-secrets.yaml <<EOF
secrets:
  twitchId: "$(op read 'op://pedro/TWITCH_ID/credential')"
  twitchSecret: "$(op read 'op://pedro/TWITCH_SECRET/credential')"
  twitchToken: "$(op read 'op://pedro/TWITCH_TOKEN/credential')"
  postgresUrl: "$(op read 'op://pedro/POSTGRES_URL/credential')"
  postgresVectorUrl: "$(op read 'op://pedro/POSTGRES_VECTOR_URL/credential')"
  discordSecret: "$(op read 'op://pedro/DISCORD_SECRET/credential')"
  discordClientId: "$(op read 'op://pedro/DISCORD_CLIENT_ID/credential')"
  discordPublicKey: "$(op read 'op://pedro/DISCORD_PUBLIC_KEY/credential')"
  discordPermission: "$(op read 'op://pedro/DISCORD_PERMISSION/credential')"
  supabasePubKey: "$(op read 'op://pedro/SUPABASE_PUB_KEY/credential')"
  supabasePrivKey: "$(op read 'op://pedro/SUPABASE_PRIV_KEY/credential')"
  supabaseUrl: "$(op read 'op://pedro/SUPABASE_URL/credential')"
  supabaseJwt: "$(op read 'op://pedro/SUPABASE_JWT/credential')"
EOF
```

### 3. Deploy with Helm

**First install:**
```bash
kubectl create namespace chatbot --dry-run=client -o yaml | kubectl apply -f -
helm install pedro ./charts/pedro-bots \
  --namespace chatbot \
  --values /tmp/pedro-secrets.yaml
```

**Upgrade (code changes, config changes, new image tags):**
```bash
helm upgrade pedro ./charts/pedro-bots \
  --namespace chatbot \
  --values /tmp/pedro-secrets.yaml
```

After an upgrade, pods restart automatically. Monitor with:
```bash
kubectl rollout status deployment -n chatbot
```

## Cluster Status and Health

```bash
# Foundry cluster overview
foundry status

# Pod status in chatbot namespace
kubectl get pods -n chatbot -o wide

# All resources
kubectl get all -n chatbot

# Helm release history
helm history pedro -n chatbot
```

## Viewing Logs

```bash
# Stream logs by component
kubectl logs -n chatbot -l app.kubernetes.io/component=discord -f
kubectl logs -n chatbot -l app.kubernetes.io/component=twitch -f
kubectl logs -n chatbot -l app.kubernetes.io/component=keepalive -f

# Previous pod logs (after crash/restart)
kubectl logs -n chatbot -l app.kubernetes.io/component=twitch --previous

# All bots at once
kubectl logs -n chatbot --selector 'app.kubernetes.io/name=pedro-bots' --max-log-requests=10 -f
```

## Rollback

```bash
# Roll back to previous revision
helm rollback pedro -n chatbot

# Roll back to a specific revision
helm rollback pedro 5 -n chatbot

# List revision history
helm history pedro -n chatbot
```

## Twitch OAuth Token Renewal

The Twitch bot uses a pre-stored token from 1Password. When the keepalive service alerts that the token is expired:

1. Deploy with an empty token to trigger the OAuth flow:
    ```bash
    cat > /tmp/pedro-secrets-notoken.yaml <<EOF
    secrets:
      twitchId: "$(op read 'op://pedro/TWITCH_ID/credential')"
      twitchSecret: "$(op read 'op://pedro/TWITCH_SECRET/credential')"
      twitchToken: ""
      postgresUrl: "$(op read 'op://pedro/POSTGRES_URL/credential')"
      postgresVectorUrl: "$(op read 'op://pedro/POSTGRES_VECTOR_URL/credential')"
      discordSecret: "$(op read 'op://pedro/DISCORD_SECRET/credential')"
      discordClientId: "$(op read 'op://pedro/DISCORD_CLIENT_ID/credential')"
      discordPublicKey: "$(op read 'op://pedro/DISCORD_PUBLIC_KEY/credential')"
      discordPermission: "$(op read 'op://pedro/DISCORD_PERMISSION/credential')"
      supabasePubKey: "$(op read 'op://pedro/SUPABASE_PUB_KEY/credential')"
      supabasePrivKey: "$(op read 'op://pedro/SUPABASE_PRIV_KEY/credential')"
      supabaseUrl: "$(op read 'op://pedro/SUPABASE_URL/credential')"
      supabaseJwt: "$(op read 'op://pedro/SUPABASE_JWT/credential')"
    EOF

    helm upgrade pedro ./charts/pedro-bots \
      --namespace chatbot \
      --values /tmp/pedro-secrets-notoken.yaml
    kubectl rollout restart deployment pedro-twitch -n chatbot
    ```

2. Watch logs for the OAuth URL:
    ```bash
    kubectl logs -n chatbot -l app.kubernetes.io/component=twitch -f
    # Look for: "Visit the URL for the auth dialog: https://id.twitch.tv/oauth2/authorize?..."
    ```

3. Open the URL in a browser while logged in as the **bot account** (not the streamer account).

4. The logs will print the new token:
    ```
    Token received: <token>
    IMPORTANT: Save this token to 1Password as TWITCH_TOKEN...
    ```

5. Save to 1Password:
    ```bash
    op item edit "TWITCH_TOKEN" "credential=<token_from_logs>"
    ```

6. Redeploy with the new token (steps 2–3 above with `twitchToken` populated).

## Debugging Common Issues

### Pod not starting / CrashLoopBackOff
```bash
kubectl describe pod -n chatbot <pod-name>
kubectl logs -n chatbot <pod-name> --previous
```

### LLM model not found (400 error in logs)
The model name in `charts/pedro-bots/values.yaml` must match the INI section name registered on the llama.cpp server (not the HuggingFace repo path).

```bash
# Check what model the server reports
curl http://100.121.229.114:8080/v1/models
```

Then update `values.yaml` and do a `helm upgrade`.

### Database migration failure
Migrations run automatically at pod startup (embedded via `go:embed`). If a migration fails:
```bash
kubectl logs -n chatbot -l app.kubernetes.io/component=discord
# Look for: "error running migrations"
```

Check the migration files in `database/migrations/` and the current DB state manually.

### Image pull errors
Verify the ZOT registry is running on blue1:
```bash
ssh 100.81.89.62 "docker ps | grep zot"
```

Verify the image was pushed:
```bash
curl http://100.81.89.62:5000/v2/pedro-discord/tags/list
curl http://100.81.89.62:5000/v2/pedro-twitch/tags/list
```

### Keepalive alerts flooding Discord
The keepalive service checks bot health on `CHECK_INTERVAL` (default 60s) and alerts on `ALERT_INTERVAL` (default 3600s). If the underlying issue is resolved, pods will stop alerting once they restart and pass health checks.

## Monitoring

| Service | Port | URL |
|---------|------|-----|
| Discord bot metrics | 6060 | `http://pedro-discord:6060/metrics` |
| Twitch bot metrics | 6061 | `http://pedro-twitch:6061/metrics` |
| Health check | — | `http://pedro-discord:6060/healthz` |
| Prometheus | 9090 | `http://100.125.196.1:9090` |
