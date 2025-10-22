# Grafana Configuration

This directory contains Grafana dashboards and alert rules for monitoring Pedro bot services.

## Contents

### Dashboards

- **`grafana-vllm-dashboard.json`** - vLLM Performance Metrics
  - Request latency (TTFT, E2E)
  - Token throughput
  - KV cache usage
  - Memory usage
  - Request finish reason distribution

- **`grafana-twitch-dashboard.json`** - Twitch Bot Metrics
  - Connection status
  - Message rate (sent/received)
  - LLM response metrics (successful/failed/empty)
  - Web search success rate

- **`grafana-discord-dashboard.json`** - Discord Bot Metrics
  - Command metrics by type (`/ask_pedro`, `/stump_pedro`, `/help_pedro`)
  - Command latency tracking
  - Error rates by command
  - 20 Questions game statistics (started/won/lost)
  - Pedro win rate gauge

### Alert Rules

- **`grafana-alert-rules.yaml`** - Pre-configured alert rules
  - **Critical alerts**: Service outages, high error rates
  - **Warning alerts**: Performance degradation, high latency
  - **Info alerts**: Memory usage, low activity

### Documentation

- **`ALERTS-README.md`** - Complete guide for setting up alerts with Discord webhooks

## Quick Import

### Import Dashboards

1. In Grafana, go to **Dashboards** → **Import**
2. Click **Upload JSON file**
3. Select one of the dashboard JSON files
4. Choose your Prometheus datasource
5. Click **Import**

### Setup Alerts

See [`ALERTS-README.md`](./ALERTS-README.md) for complete setup instructions.

## Directory Structure

```
config/grafana/
├── README.md                          # This file
├── ALERTS-README.md                   # Alert setup guide
├── grafana-vllm-dashboard.json        # vLLM dashboard
├── grafana-twitch-dashboard.json      # Twitch bot dashboard
├── grafana-discord-dashboard.json     # Discord bot dashboard
└── grafana-alert-rules.yaml           # Alert rules configuration
```

## Prerequisites

- Grafana 10.0+
- Prometheus datasource configured
- Pedro services exposing metrics:
  - Twitch bot: http://localhost:6061/metrics
  - Discord bot: http://localhost:6060/metrics
  - vLLM: http://localhost:8000/metrics (or your vLLM endpoint)

## Prometheus Configuration

Ensure your `prometheus.yml` includes these scrape targets:

```yaml
scrape_configs:
  - job_name: 'pedro-twitch'
    static_configs:
      - targets: ['localhost:6061']

  - job_name: 'pedro-discord'
    static_configs:
      - targets: ['localhost:6060']

  - job_name: 'vllm'
    static_configs:
      - targets: ['localhost:8000']  # Adjust to your vLLM endpoint
```

## Metrics Exposed

### Twitch Bot (expvar + Go runtime)
- `twitch_connection_count` - Connection status
- `twitch_message_recieved_count` - Messages received
- `twitch_message_sent_count` - Messages sent
- `successful_llm_gen_count` - Successful LLM generations
- `failed_llm_gen_count` - Failed LLM generations
- `empty_llm_response_count` - Empty LLM responses
- `web_search_success_count` - Successful web searches
- `web_search_fail_count` - Failed web searches
- Go runtime metrics (memory, GC, goroutines, etc.)

### Discord Bot (Prometheus + expvar + Go runtime)
- `discord_command_total{command="<name>"}` - Commands executed (with labels)
- `discord_command_errors{command="<name>"}` - Command errors (with labels)
- `discord_command_duration_seconds` - Command latency histogram (with labels)
- `discord_stump_pedro_games_total{status="<status>"}` - 20Q game stats
- `discord_message_recieved` - Messages received (legacy)
- `discord_message_sent` - Messages sent (legacy)
- Go runtime metrics

### vLLM (Prometheus)
- `vllm:request_success_total{finished_reason="<reason>",model_name="<model>"}` - Request outcomes
- `vllm:time_to_first_token_seconds` - TTFT histogram
- `vllm:e2e_request_latency_seconds` - E2E latency histogram
- `vllm:prompt_tokens_total` - Prompt tokens processed
- `vllm:generation_tokens_total` - Generation tokens processed
- `vllm:kv_cache_usage_perc` - KV cache utilization
- `vllm:num_requests_running` - Active requests
- `vllm:num_requests_waiting` - Queued requests
- Process metrics (memory, CPU)

## Dashboard Features

### Command-Level Monitoring (Discord)
The Discord dashboard uses Prometheus labels to track metrics per command type:
- View usage patterns by command
- Compare latency across different commands
- Identify which commands are failing

Example queries:
```promql
# Command usage over time
rate(discord_command_total{command="ask_pedro"}[5m])

# Error rate by command
rate(discord_command_errors[5m]) / rate(discord_command_total[5m])

# p95 latency by command
histogram_quantile(0.95, sum(rate(discord_command_duration_seconds_bucket[5m])) by (command, le))
```

### vLLM Model Tracking
All vLLM metrics include the `model_name` label, allowing you to:
- Monitor multiple models simultaneously
- Compare performance across different models
- Track model-specific issues

## Updating Dashboards

After making changes to dashboards in Grafana:

1. Export the updated dashboard JSON
2. Replace the corresponding file in this directory
3. Commit to git for version control

## Related Documentation

- [Main README](../../README.md#setting-up-grafana-alerts-with-discord) - Grafana setup overview
- [Deployment Scripts](../../deployment/README.md) - Deployment automation
- [KeepAlive Service](../../keepalive/README.md) - Health monitoring with Discord alerts
