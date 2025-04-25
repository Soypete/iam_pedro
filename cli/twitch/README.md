# PEDRO twitch bot
---

The AI chat bot that interacts with Twitch chat. It uses Oauth to connect to a twitch account and listen to chat messages. It uses the OpenAI API to generate responses to chat messages. It can also send messages to the chat.


## Build  

We use go build command to build. It is packaged and distributed as a docker image. The docker image is built using the Dockerfile in the cli/twitch directory of the project. The docker image is built using the following command:

```
docker build -t pedro-twitch:latest -f cli/twitch/twitchbot.Dockerfile .
```
This is built using Github Actions.


## Run
The following need to be specified at Runtime:

```
docker run pedro-twitch -e OP_SERVICE_ACCOUNT ${1password_service_account} -e LLAMA_CPP_PATH ${tailscale_serve_address} -e TWITCH_ID ${twitch_clientID} -p 6060:6060
```

The twitch ID needs to be available in plain text as it is used in the redirect URL for the twitch oauth. So this cannot be read in via the 1password secret manager. The LLAMA_CPP_PATH is the path to the openAI endpoints. This is the local path of the self hosted server. The password_service_account is the credential used to read in all secrets from the secret manager and should passed in at run time. IT is the only secret in your environment and can be easily rotated if needed. 


```
```

