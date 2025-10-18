# IAM_PEDRO

This is a twitch chat app build in go using [llama.cpp](https://github.com/ggerganov/llama.cpp), [langchain-go](https://github.com/tmc/langchaingo), and [supabase](https://supabase.com).

[![Actions Status](https://github.com/soypete/{}/workflows/build/badge.svg)](https://github.com/soypete/{}/actions/workflows/go.yml)
[![wakatime](https://wakatime.com/badge/user/953eeb5a-d347-44af-9d8b-a5b8a918cecf/project/018ef728-5089-4148-b326-592f7a744f7e.svg)](https://wakatime.com/badge/user/953eeb5a-d347-44af-9d8b-a5b8a918cecf/project/018ef728-5089-4148-b326-592f7a744f7e)

## Quick Start

### Local Development (Recommended for Testing)

Run Pedro locally with Ollama - no GPU or production access needed!

```bash
cd local-dev
./setup.sh    # One-time setup: installs model, creates configs
./run.sh      # Start Discord bot

# In another terminal
ollama serve  # Keep this running
```

**See [local-dev/README.md](local-dev/README.md) for complete local development guide.**

### Production Deployment

Deploy to production servers with Docker and systemd. See [deployment/README.md](deployment/README.md) for complete guide.

## Chat Experience

The bot should record all chat in a vector db with emdeddings and then use that to generate responses. The bot should also be able to generate content based on the chat history.
The bot should also have a table of helpful links that it can provide to the chat.
The bot should also know what the stream title is as well as history of the stream titles.
The bot should respond to questions, to its name, and to types of prompts that it has been trained on.

## Notes:

So far the longest that the bot has taken to respond is 5 minutes, so we need to account for that in the tmeout the api call.

## Discord integration

- two slash commands:
    - /askPedro <question>
    - /stumpPedro <thing for 20 questions>
    - /helpPedro

things to do:
- [ ] connect to discord
- [ ] add slash commands
- [ ] leverage threads for 20 questions
- [ ] instructions for how to use the bot

## Production Deployment

### Manual Build and Deploy (Recommended)

Build and deploy Pedro containers to production servers with monitoring:

```bash
# Build both Discord and Twitch containers with current git commit as tag
./deployment/build-and-deploy.sh

# Build specific service with custom tag
./deployment/build-and-deploy.sh v1.2.3 discord
./deployment/build-and-deploy.sh v1.2.3 twitch

# Deploy to target host (100.81.89.62)
scp pedro-discord-<TAG>.tar.gz deployment/deploy-*.sh remote-deploy.sh user@100.81.89.62:~/
ssh user@100.81.89.62
./remote-deploy.sh
```

### Services and Ports

- **Discord Bot**: Port 6060 (`http://100.81.89.62:6060/metrics`)
- **Twitch Bot**: Port 6061 (`http://100.81.89.62:6061/metrics`)
- **Prometheus**: Port 9090 (`http://100.125.196.1:9090`)

### Environment Setup

Create `/opt/pedro/prod.env` on the target host with:

```bash
DISCORD_TOKEN=your_discord_token
TWITCH_TOKEN=your_twitch_token
TWITCH_CHANNEL=your_twitch_channel
DATABASE_URL=your_database_url
OPENAI_API_KEY=your_openai_key
OP_CONNECT_HOST=your_1password_connect_host
OP_CONNECT_TOKEN=your_1password_connect_token
```

### Prometheus Monitoring Setup

Set up Prometheus on 100.125.196.1:

```bash
# Copy configuration and setup script
scp prometheus/prometheus.yml prometheus/setup-prometheus.sh user@100.125.196.1:~/
ssh user@100.125.196.1
chmod +x setup-prometheus.sh
./setup-prometheus.sh
```

This will monitor:
- Pedro Discord Bot metrics
- Pedro Twitch Bot metrics  
- Pedro LLM service at `https://pedro-gpu.tail6fbc5.ts.net`

### Service Management

```bash
# Check service status
sudo systemctl status pedro-discord
sudo systemctl status pedro-twitch

# View logs
sudo journalctl -u pedro-discord -f
sudo journalctl -u pedro-twitch -f

# Restart services
sudo systemctl restart pedro-discord
sudo systemctl restart pedro-twitch
```

## TODO

* change bot name
* git bot moderator permissions
* add more tokens to llm in llama cpp
* batch twitch chat to set via the langchain [GenerateContent](https://github.com/tmc/langchaingo/blob/3a36972919a83b119825de4ea6216e175ae20cb3/examples/openai-chat-example/openai_chat_example.go#L25C19-L25C34)
* Add embeddings -> we need to select a permenant model for it
* add config for managing the bot [channel commands, prompts, links, stream title etc]
* integrate twitch api for getting stream title
* integrate a classifier for the chat messages -> lable history for training
* make things like twitch channel, bot name, etc configurable
