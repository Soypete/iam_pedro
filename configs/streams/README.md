# Stream Context Configurations

This directory contains YAML configuration files that provide Pedro with context about what's happening on the stream. These configs help Pedro give relevant, informed responses to viewers.

## Usage

### Command Line
```bash
# Run with stream config
go run ./cli/twitch \
  -model "your-model-name" \
  -streamConfig "configs/streams/golang-nov-2025.yaml"
```

### Docker

Mount the configs directory as a volume:

```bash
# Using docker run
docker run \
  -v $(pwd)/configs:/app/configs:ro \
  -e LLAMA_CPP_PATH="http://host.docker.internal:8080" \
  -e POSTGRES_URL="..." \
  -e TWITCH_ID="..." \
  -e TWITCH_SECRET="..." \
  pedro-twitch \
  -model "your-model" \
  -streamConfig "/app/configs/streams/golang-nov-2025.yaml"

# Using docker-compose
# Add to your docker-compose.yml:
services:
  pedro-twitch:
    image: pedro-twitch:latest
    volumes:
      - ./configs:/app/configs:ro
    environment:
      - LLAMA_CPP_PATH=http://llama-cpp:8080
      - POSTGRES_URL=...
      - TWITCH_ID=...
      - TWITCH_SECRET=...
    command: >
      -model "your-model"
      -streamConfig "/app/configs/streams/golang-nov-2025.yaml"
```

**Note**: The `:ro` flag mounts the directory as read-only for security.

### Kubernetes

Create a ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pedro-stream-configs
data:
  golang-nov-2025.yaml: |
    # ... your config content ...
---
apiVersion: v1
kind: Pod
metadata:
  name: pedro-twitch
spec:
  containers:
  - name: pedro
    image: pedro-twitch:latest
    args:
      - "-model"
      - "your-model"
      - "-streamConfig"
      - "/app/configs/streams/golang-nov-2025.yaml"
    volumeMounts:
    - name: stream-configs
      mountPath: /app/configs/streams
      readOnly: true
  volumes:
  - name: stream-configs
    configMap:
      name: pedro-stream-configs
```

## Config Types

### Meetup Events
Use for community meetups, talks, and presentations.

**Example**: `golang-nov-2025.yaml`

Key features:
- Speaker information
- Event schedule
- Community organization info (e.g., Forge Utah)
- Registration links
- Video call details

### Live Coding Sessions
Use for regular coding streams.

**Example**: `live-coding-template.yaml`

Key features:
- What you're building today
- Tech stack
- Learning objectives
- Links to repos/docs

### Conferences
Use when streaming conference talks or events.

**Example**: `conference-template.yaml`

Key features:
- Conference name and schedule
- Multiple speakers/sessions
- Sponsor information

## Creating a New Config

1. Copy an appropriate template:
```bash
cp configs/streams/live-coding-template.yaml configs/streams/my-stream-2025-11-10.yaml
```

2. Edit the file with your stream details

3. Run Pedro with your config:
```bash
go run ./cli/twitch -model "model" -streamConfig "configs/streams/my-stream-2025-11-10.yaml"
```

## Config Structure

All configs share this base structure:

```yaml
metadata:
  name: "Stream Title"
  date: "2025-11-10T18:00:00-07:00"  # ISO 8601 format

event_info:
  title: "What You're Doing"
  description: "Detailed description"

# ... type-specific fields ...

bot_instructions:
  response_style: "enthusiastic and helpful"
  key_points:
    - "Point 1 to emphasize"
    - "Point 2 to emphasize"
```

## Best Practices

1. **Update dates in ISO 8601 format**: `2025-11-10T18:30:00-07:00`
2. **Keep descriptions concise**: Pedro has a 500-character limit
3. **Include relevant links**: Registration, docs, repos
4. **Test your config**: Run with `-errorLevel debug` to see if it loads
5. **Use descriptive filenames**: Include date and topic

## Troubleshooting

### Config Not Loading

```bash
# Check the file exists
ls -la configs/streams/your-config.yaml

# Test with debug logging
go run ./cli/twitch \
  -model "model" \
  -streamConfig "configs/streams/your-config.yaml" \
  -errorLevel debug
```

### Docker Path Issues

Make sure paths in the container match your command:
- Host path: `./configs/streams/file.yaml`
- Container path: `/app/configs/streams/file.yaml`
- Volume mount: `./configs:/app/configs:ro`

### Validation Errors

Required fields:
- `metadata.name`
- `event_info.title`
- `metadata.date`

Check the logs for specific validation errors.

## Examples

See the template files in this directory:
- `golang-nov-2025.yaml` - Meetup event example
- `live-coding-template.yaml` - Regular stream template
- `conference-template.yaml` - Conference stream template
