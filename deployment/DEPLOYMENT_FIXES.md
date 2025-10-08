# Deployment Fixes Summary

This document summarizes the fixes applied to resolve Docker secrets injection and Twitch OAuth issues.

## Problems Solved

### 1. Docker Secrets Injection
**Problem:** The Dockerfile CMD had incorrect syntax for `op run`, preventing 1Password secrets from being injected properly.

**Solution:**
- Fixed Dockerfile CMD syntax: `op run --env-file=/app/prod.env -- /app/main ...`
- Added volume mount in systemd: `-v /opt/pedro/prod.env:/app/prod.env:ro`
- Added `OP_SERVICE_ACCOUNT_TOKEN` environment variable to container
- Split configuration into two files:
  - `/opt/pedro/service.env` - systemd environment (token, non-secrets)
  - `/opt/pedro/prod.env` - 1Password secret references

### 2. Twitch OAuth on Headless Server
**Problem:** OAuth flow required browser on localhost:3000, which doesn't work on headless deployment.

**Solution:**
- Added `TWITCH_TOKEN` environment variable support to skip OAuth
- Modified `twitch/auth.go` to check for pre-existing token first
- Added `OAUTH_REDIRECT_HOST` environment variable for remote OAuth
- Exposed port 3000 from Twitch container for initial OAuth setup
- Updated code to support remote OAuth via Tailscale or IP address

## Files Changed

### Docker Configuration
- `cli/discord/discordBot.Dockerfile` - Fixed CMD syntax, removed twitch mode
- `cli/twitch/twitchBot.Dockerfile` - Fixed CMD syntax, removed discord mode

### Deployment Scripts
- `deployment/deploy-discord.sh` - Added volume mount, env vars, service.env check
- `deployment/deploy-twitch.sh` - Added volume mount, env vars, port 3000, OAuth host

### Application Code
- `twitch/auth.go` - Added TWITCH_TOKEN check, OAUTH_REDIRECT_HOST support, better logging

### Documentation
- `deployment/README.md` - Comprehensive guide for both config files and OAuth setup

## Configuration Files Required

### /opt/pedro/service.env (systemd environment)
```bash
OP_SERVICE_ACCOUNT_TOKEN=ops_your_token_here
TWITCH_ID=your_twitch_client_id
OAUTH_REDIRECT_HOST=100.81.89.62:3000
```

### /opt/pedro/prod.env (1Password references)
```bash
DISCORD_TOKEN=op://vault/discord-bot/token
TWITCH_SECRET=op://vault/twitch-bot/client-secret
TWITCH_TOKEN=op://vault/twitch-bot/access-token  # Optional: skip OAuth
DATABASE_URL=op://vault/postgres/connection-url
LLAMA_CPP_PATH=https://pedro-gpu.tail6fbc5.ts.net
```

## Deployment Flow

### First Time Twitch Deployment:
1. Deploy without `TWITCH_TOKEN` in prod.env
2. Watch logs: `sudo journalctl -u pedro-twitch -f`
3. Open OAuth URL in browser from any device
4. Copy token from logs
5. Save token to 1Password
6. Update prod.env with `TWITCH_TOKEN` reference
7. Restart service - no more OAuth needed!

### Subsequent Deployments:
- Bot automatically uses `TWITCH_TOKEN` from 1Password
- No browser interaction required
- Fully headless operation

## Testing Checklist

- [ ] 1Password CLI works in container (`op` binary present)
- [ ] OP_SERVICE_ACCOUNT_TOKEN authenticates successfully
- [ ] Secrets are injected from prod.env at runtime
- [ ] Discord bot starts without OAuth (uses DISCORD_TOKEN)
- [ ] Twitch bot uses TWITCH_TOKEN when available
- [ ] Twitch bot falls back to OAuth when TWITCH_TOKEN missing
- [ ] OAuth redirect works via remote host (100.81.89.62:3000)
- [ ] Port 3000 is accessible for OAuth callback
- [ ] Metrics exposed on 6060 (discord) and 6061 (twitch)
- [ ] Services restart automatically on failure
- [ ] Logs show successful secret injection

## Security Notes

- `service.env` contains OP_SERVICE_ACCOUNT_TOKEN - protect with chmod 600
- `prod.env` contains only 1Password references - safe even if exposed
- TWITCH_ID is not a secret - safe to store in plain text
- Never commit actual tokens/secrets to version control
- Use 1Password for all sensitive values