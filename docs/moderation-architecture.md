# Twitch Chat Moderation System Architecture

## Overview

The Pedro bot includes an LLM-powered moderation system that monitors Twitch chat in parallel with the main chat response functionality. The system uses tool calling to make moderation decisions and executes actions via the Twitch Helix API.

## Architecture Diagram

```
                    ┌─────────────────────────────────────────────────────────┐
                    │                    Twitch IRC                           │
                    │                  (go-twitch-irc)                        │
                    └─────────────────────┬───────────────────────────────────┘
                                          │
                              OnPrivateMessage callback
                                          │
                    ┌─────────────────────▼───────────────────────────────────┐
                    │               IRC Message Handler                        │
                    │                (connection.go)                           │
                    └───────┬─────────────────────────────────┬───────────────┘
                            │                                 │
              Regular Chat Path                    Moderation Path (parallel)
                            │                                 │
                    ┌───────▼───────┐                 ┌───────▼───────┐
                    │  HandleChat() │                 │ modMonitor.   │
                    │               │                 │ MessageCh     │
                    └───────┬───────┘                 └───────┬───────┘
                            │                                 │
                    ┌───────▼───────┐                 ┌───────▼───────┐
                    │   TwitchLLM   │                 │   Quick       │
                    │  (Response)   │                 │   Filter      │
                    └───────────────┘                 └───────┬───────┘
                                                              │
                                              ┌───────────────▼───────────────┐
                                              │     LLM Tool Calling          │
                                              │   (Moderation Decision)       │
                                              │                               │
                                              │  Tools:                       │
                                              │  - no_action                  │
                                              │  - warn_user                  │
                                              │  - timeout_user               │
                                              │  - ban_user                   │
                                              │  - delete_message             │
                                              │  - enable_slow_mode           │
                                              │  - enable_subscribers_only    │
                                              │  - etc.                       │
                                              └───────────────┬───────────────┘
                                                              │
                                              ┌───────────────▼───────────────┐
                                              │     Action Executor           │
                                              │                               │
                                              │  - Rate limiting              │
                                              │  - Dry run mode               │
                                              │  - Helix API calls            │
                                              └───────────────┬───────────────┘
                                                              │
                         ┌────────────────────────────────────┼────────────────┐
                         │                                    │                │
                 ┌───────▼───────┐                   ┌────────▼────────┐ ┌─────▼─────┐
                 │ Twitch Helix  │                   │   PostgreSQL    │ │ Prometheus│
                 │     API       │                   │  (mod_actions)  │ │  Metrics  │
                 │               │                   │                 │ │           │
                 │ - Ban user    │                   │ - Trigger msg   │ │ - Counters│
                 │ - Timeout     │                   │ - LLM decision  │ │ - Duration│
                 │ - Delete msg  │                   │ - Action taken  │ │ - By tool │
                 │ - Chat modes  │                   │ - Result        │ │           │
                 └───────────────┘                   └─────────────────┘ └───────────┘
```

## Components

### 1. Moderation Monitor (`twitch/moderation/monitor.go`)

The central component that runs as a goroutine, processing messages from a buffered channel.

**Key features:**
- Buffered message channel (100 messages) to prevent blocking the main IRC handler
- Quick content filtering before LLM evaluation (checks for spam patterns, links, etc.)
- Configurable rate limiting
- Support for dry-run mode

### 2. Moderation Tools (`ai/twitchchat/agent/moderation_tools.go`)

Defines the available moderation actions as LLM tool definitions:

| Tool | Description |
|------|-------------|
| `no_action` | Take no action (message is acceptable) |
| `warn_user` | Send a warning message to chat |
| `timeout_user` | Temporarily ban a user (1-1209600 seconds) |
| `ban_user` | Permanently ban a user |
| `delete_message` | Delete a specific message |
| `enable_slow_mode` | Enable slow mode (wait between messages) |
| `disable_slow_mode` | Disable slow mode |
| `enable_followers_only` | Require followers to chat |
| `disable_followers_only` | Disable followers-only mode |
| `enable_subscribers_only` | Require subscribers to chat |
| `disable_subscribers_only` | Disable subscribers-only mode |
| `enable_emote_only` | Restrict chat to emotes only |
| `disable_emote_only` | Disable emote-only mode |
| `create_poll` | Create a channel poll |
| `end_poll` | End an active poll |
| `create_prediction` | Create a channel prediction |
| `resolve_prediction` | Resolve/cancel a prediction |
| `send_announcement` | Send a channel announcement |
| `add_vip` | Grant VIP status to a user |
| `remove_vip` | Remove VIP status from a user |
| `start_raid` | Start a raid to another channel |
| `cancel_raid` | Cancel an outgoing raid |

### 3. Twitch Helix Client (`twitch/helix/client.go`)

REST client for Twitch's Helix API that implements all moderation endpoints:

- Authentication via OAuth2 access token
- Automatic retry with exponential backoff
- Proper error handling and logging

### 4. Configuration (`ai/moderation_config.go`)

YAML-based configuration supporting:

```yaml
enabled: true
channels:
  - soypetetech
sensitivity_level: moderate  # conservative, moderate, aggressive
allowed_tools:
  - no_action
  - warn_user
  - timeout_user
  - delete_message
rate_limits:
  actions_per_minute: 10
  bans_per_hour: 5
  timeouts_per_user_per_hour: 3
channel_rules:
  - Be respectful and kind
  - No spam or self-promotion
  - Keep discussions on-topic
dry_run: false  # Log actions without executing
escalation:
  warnings_before_timeout: 2
  timeouts_before_ban: 3
  timeout_multiplier: 2.0
```

### 5. Database Schema

```sql
CREATE TABLE mod_actions (
    id SERIAL PRIMARY KEY,
    channel_id TEXT NOT NULL,
    trigger_message TEXT NOT NULL,
    trigger_user_id TEXT NOT NULL,
    trigger_username TEXT NOT NULL,
    llm_decision TEXT NOT NULL,
    tool_call_name TEXT NOT NULL,
    tool_call_args JSONB,
    action_result TEXT NOT NULL,
    target_user_id TEXT,
    target_username TEXT,
    duration_seconds INTEGER,
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### 6. Metrics

**Expvar counters:**
- `mod_action_total` - Total moderation actions attempted
- `mod_action_success` - Successfully executed actions
- `mod_action_failed` - Failed action attempts
- `mod_action_no_action` - Messages where LLM decided no action needed

**Prometheus metrics:**
- `moderation_actions_total{tool, success}` - Actions by tool type and success status
- `moderation_evaluations_total` - Messages evaluated by LLM
- `moderation_decision_duration_seconds` - Histogram of LLM decision time

## Message Flow

1. **IRC Reception**: Message arrives via Twitch IRC
2. **Parallel Dispatch**: Message sent to both chat handler and moderation channel
3. **Quick Filter**: Fast checks for obvious spam/violations (regex patterns)
4. **LLM Evaluation**: Message context sent to LLM with moderation tools
5. **Tool Parsing**: LLM response parsed for tool calls
6. **Rate Check**: Verify action doesn't exceed rate limits
7. **Action Execution**: Call Helix API (or log in dry-run mode)
8. **Database Logging**: Record action in mod_actions table
9. **Metrics Update**: Increment relevant counters

## Configuration Options

### Sensitivity Levels

| Level | Description |
|-------|-------------|
| `conservative` | Only act on clear violations, prefer warnings |
| `moderate` | Balanced approach, reasonable enforcement |
| `aggressive` | Strict enforcement, quick escalation |

### Rate Limits

Prevent the bot from taking too many actions:

- `actions_per_minute`: Maximum total actions per minute
- `bans_per_hour`: Maximum bans per hour
- `timeouts_per_user_per_hour`: Maximum timeouts for any single user per hour

### Escalation

Progressive enforcement:

1. First offense: Warning (if `warnings_before_timeout > 0`)
2. Repeated offense: Timeout with increasing duration
3. Continued violations: Ban (after `timeouts_before_ban` timeouts)

## CLI Flags

```bash
# Enable moderation with default config
./twitch -enableModeration

# Use custom config file
./twitch -modConfig configs/moderation.yaml

# Run in dry-run mode (log only)
./twitch -enableModeration -modDryRun
```

## Grafana Dashboard

The moderation dashboard (`grafana-moderation-dashboard.json`) provides:

- Total messages evaluated
- Actions taken (success/failed/no-action)
- Action rate over time
- Decision latency percentiles
- Actions by tool type breakdown
- Success rate gauge

## Alerts

Configured alerts in `grafana-alert-rules.yaml`:

| Alert | Severity | Description |
|-------|----------|-------------|
| Moderation High Failure Rate | Critical | >20% of actions failing |
| Unusual Moderation Activity | Warning | >30 actions per minute |
| Moderation Decisions Slow | Warning | p95 latency >5 seconds |
| High Ban Rate | Warning | >10 bans per hour |

## Security Considerations

1. **Dry Run Mode**: Always test with `dry_run: true` first
2. **Tool Restrictions**: Limit `allowed_tools` to what's needed
3. **Rate Limits**: Prevent runaway moderation
4. **Audit Trail**: All actions logged to database
5. **OAuth Scopes**: Bot needs `moderator:manage:*` and `channel:manage:*` scopes
