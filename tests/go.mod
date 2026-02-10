module github.com/Soypete/twitch-llm-bot/tests

go 1.24

replace github.com/Soypete/twitch-llm-bot => ../

require (
	github.com/Soypete/twitch-llm-bot v0.0.0-00010101000000-000000000000
	github.com/bwmarrin/discordgo v0.28.1
	github.com/gempir/go-twitch-irc/v2 v2.8.1
)