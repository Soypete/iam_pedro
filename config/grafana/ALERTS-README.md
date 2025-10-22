# Grafana Alert Rules Setup

This directory contains pre-configured Grafana alert rules for monitoring Pedro bot services.

## Quick Setup

### 1. Set Up Discord Webhook (One Channel)

1. Open your Discord server settings
2. Go to **Integrations** → **Webhooks**
3. Click **New Webhook**
4. Name it "Pedro Alerts" and select your target channel (e.g., #pedro-alerts)
5. Copy the webhook URL

### 2. Configure Grafana Contact Point

1. In Grafana, navigate to **Alerting** → **Contact points**
2. Click **Add contact point**
3. Set **Name**: `Pedro Discord Alerts`
4. Set **Integration**: `Discord`
5. Paste your webhook URL
6. (Optional) To get tagged, add this to **Message** field:
   ```
   <@YOUR_USER_ID> {{ template "default.message" . }}
   ```
   To find your Discord user ID: Right-click your name → Copy ID (requires Developer Mode enabled)
7. Click **Test** to verify
8. Click **Save contact point**

### 3. Import Alert Rules

**Option A: Manual Import (Recommended)**

1. In Grafana, navigate to **Alerting** → **Alert rules**
2. Click **New alert rule**
3. For each alert in `grafana-alert-rules.yaml`, create manually:
   - Copy the PromQL query from `expr` field
   - Set the threshold from `evaluator.params`
   - Set evaluation time from `for` field
   - Add annotations for summary/description
   - Select contact point: `Pedro Discord Alerts`

**Option B: Provisioning (Advanced)**

If you're running Grafana in Docker or Kubernetes:

1. Copy `grafana-alert-rules.yaml` to your Grafana provisioning directory:
   ```bash
   cp deployment/grafana-alert-rules.yaml /etc/grafana/provisioning/alerting/
   ```

2. Update the `datasourceUid` in the YAML file to match your Prometheus datasource UID:
   - In Grafana, go to **Connections** → **Data sources** → **Prometheus**
   - Copy the UID from the URL (e.g., `P1809F7CD0C75ACF3`)
   - Replace all instances of `prometheus` in the YAML with your actual UID

3. Restart Grafana

4. Configure the contact point for the alert group:
   - Navigate to **Alerting** → **Notification policies**
   - Click **+ New nested policy**
   - Match label: `severity = critical|warning|info`
   - Contact point: `Pedro Discord Alerts`
   - Save policy

### 4. Verify Alerts

1. Navigate to **Alerting** → **Alert rules**
2. You should see folders:
   - **pedro_critical_alerts** (4 rules)
   - **pedro_warning_alerts** (6 rules)
   - **pedro_info_alerts** (2 rules)
3. Check that all alerts show "Normal" or "Pending" state
4. Verify contact point is assigned

### 5. Test an Alert

Trigger a test alert to verify Discord notifications work:

1. Find an alert that's easy to trigger (e.g., "High Memory Usage")
2. Temporarily lower the threshold to trigger it
3. Wait for evaluation period
4. Check Discord channel for alert notification
5. Restore original threshold

## Alert Summary

### 🚨 Critical Alerts (1m check interval)
- **Twitch Bot Offline** - No active Twitch connection
- **Discord Bot High Error Rate** - >20% commands failing
- **vLLM Service Down** - vLLM not responding
- **High LLM Failure Rate** - >15% LLM calls failing

### ⚠️ Warning Alerts (2m check interval)
- **vLLM High TTFT** - p95 time-to-first-token >1s
- **vLLM High E2E Latency** - p95 end-to-end >3s
- **vLLM KV Cache High** - Cache usage >85%
- **vLLM Queue Backup** - >10 requests waiting
- **High Empty Response Rate** - >25% empty LLM responses
- **Web Search Failures** - >30% searches failing
- **Discord Commands Slow** - p95 latency >5s

### 📊 Info Alerts (5m check interval)
- **High Memory Usage** - Go bots >100MB heap
- **vLLM High Memory** - vLLM >2GB resident memory

## Customization

### Adjust Thresholds

Edit thresholds in the YAML file by changing the `params` values:

```yaml
evaluator:
  params:
    - 0.2  # Change this value (20% in this example)
  type: gt  # gt = greater than, lt = less than
```

### Add Custom Alerts

To add your own alert:

1. Copy an existing alert block from the YAML
2. Generate a new UID: `pedro_<descriptive_name>`
3. Modify the PromQL query in the `expr` field
4. Update thresholds, annotations, and labels
5. Re-import the file

### Silence Alerts

To temporarily disable an alert:

1. Navigate to **Alerting** → **Silences**
2. Click **Add silence**
3. Add matcher: `alertname = <alert_title>`
4. Set duration and comment
5. Click **Submit**

## Troubleshooting

### Alerts Not Firing

- Check Prometheus is scraping metrics correctly
- Verify datasource UID matches in alert rules
- Check alert evaluation logs in **Alerting** → **Alert rules** → (click alert) → **State history**

### Discord Notifications Not Sending

- Test the contact point: **Alerting** → **Contact points** → **Test**
- Verify webhook URL is correct
- Check Grafana logs for Discord API errors
- Ensure notification policy is routing alerts to your contact point

### Metrics Not Found

If you see "no data" errors:
- Verify the service is exposing metrics at the correct port
- Check Prometheus is configured to scrape the service
- Confirm metric names match between the alert rules and actual metrics

## Dashboard Files

- `grafana-vllm-dashboard.json` - vLLM performance metrics
- `grafana-twitch-dashboard.json` - Twitch bot metrics
- `grafana-discord-dashboard.json` - Discord bot metrics with command-level details

Import these dashboards to visualize the same metrics used in alerts.
