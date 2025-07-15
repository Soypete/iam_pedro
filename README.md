# IAM_PEDRO

This is a twitch chat app build in go using [llama.cpp](https://github.com/ggerganov/llama.cpp), [langchain-go](https://github.com/tmc/langchaingo), and [supabase](https://supabase.com).

[![Actions Status](https://github.com/soypete/{}/workflows/build/badge.svg)](https://github.com/soypete/{}/actions/workflows/go.yml)
[![wakatime](https://wakatime.com/badge/user/953eeb5a-d347-44af-9d8b-a5b8a918cecf/project/018ef728-5089-4148-b326-592f7a744f7e.svg)](https://wakatime.com/badge/user/953eeb5a-d347-44af-9d8b-a5b8a918cecf/project/018ef728-5089-4148-b326-592f7a744f7e)

## To Use

install [lama.cpp](https://github.com/ggerganov/llama.cpp) and run there server on `127.0.0.1` and port `8080`

Them pull the docker container

```bash
docker pull ghcr.io/soypete/iam_pedro:latest
```

Then run the container with the following environment variables

```bash
docker run -e LLAMA_CPP_PATH="" -e POSTGRES_URL="" -e TWITCH_ID="" -e TWITCH_SECRET="" -e POSTGRES_VECTOR_URL=""
```

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

## Agents Flow

![Agents Architecture](docs/images/janky%20agents.png)

Building agents without MCP (Model Context Protocol) is challenging and often results in janky, fragile implementations. Our approach leverages web search capabilities to provide Pedro with real-time information access, enabling more reliable and informed responses. This allows Pedro to move beyond static knowledge and engage with current events, technical documentation, and evolving information landscapes.
# 🤖 Pedro Orchestrator: Step-Based Agent Planning

An experimental scaffold for bringing workflow-awareness and lightweight agentic planning to Pedro, the LLM-powered bot for Discord and Twitch.

This prototype focuses on enabling **structured reasoning** via **prescribed step sequences**, with live feedback in chat threads and potential future expansion into DAG-style orchestration.

---
# TODO:

##  Key Concept

Pedro doesn't just reply — it **plans and executes**.

Instead of returning a one-shot answer, Pedro will:
1. Generate a **structured plan** using a constrained set of known steps
2. Execute each step in order
3. Publish intermediate status (in Discord threads or Twitch overlays)
4. Complete with a final synthesized response

---

##  Prescribed Planning Actions (MVP)

Each workflow Pedro generates must use only these valid steps:

- ✅ `api_call` — Make a predefined API call (e.g. OpenAI, DuckDuckGo)
- ✅ `db_query` — Query known structured data sources (e.g. Postgres)
- ✅ `web_search` — Perform a live or cached web search
- ✅ `reference_check` — Check historical logs or frequent queries
- ✅ `return_response` — Compile and deliver final output

---

## ✅ To-Do (Pedro Workflow MVP)

### 1. Planning & Prompting

- [ ] Define a system prompt to instruct Pedro to **respond with a plan**
- [ ] Enforce plan schema (e.g. JSON list of steps)
- [ ] Tag questions that should trigger planning (`#plan`, `#multi`, etc.)

### 2. Plan Schema & Validation

- [ ] Define `Step` schema:
  ```json
  { "action": "api_call", "params": { "tool": "openai", "query": "..." } }
  ```
- [ ] Validate plan output against schema
- [ ] Allow users/devs to mark steps as failed or redundant

### 3. Step Execution Engine

- [ ] Build an executor to parse step list and run them in sequence
- [ ] Handle `await` or async steps cleanly
- [ ] Log each step's result and status (success/fail)

### 4. Discord/Twitch Integration

- [ ] Post plan as a threaded message on Discord
- [ ] Post step-by-step execution progress (✔️ / ❌)
- [ ] Post final result with reference to full workflow

### 5. Pedro Behavior Controls

- [ ] Allow toggling `pedro-agentic-mode: on|off`
- [ ] Add retry button or message for broken plans
- [ ] Log user question → plan → result for memory debugging

---

##  Stretch Goals

- Multi-path workflows (add branching)
- Error recovery or replan loop
- Integration with external orchestrators (LangGraph, Dagster)
- Push logs into vector DB for plan similarity and RAG

---

##  Goal

Make Pedro more than a chatbot — make it a visible, understandable, and semi-reliable **agentic planner**.


