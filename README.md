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

things to do:
- [ ] connect to discord
- [ ] add slash commands
- [ ] leverage threads for 20 questions
- [ ] instructions for how to use the bot

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
