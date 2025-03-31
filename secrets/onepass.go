package secrets

import (
	"context"
	"fmt"
	"os"

	"github.com/1password/onepassword-sdk-go"
)

// supabase creds are in env.
var (
	TwitchID           string
	TwitchSecret       string
	PostgresUrl        string
	PostgresVectorUrl  string
	DiscordSecret      string
	DiscordClientID    string
	DiscordPublicKey   string
	DiscordPermissions string
)

// Init pull secrets from 1password and set them to global vars.
func Init() error {
	err := getSecrets()
	if err != nil {
		return fmt.Errorf("error getting secrets: %v", err)
	}
	println("Secrets loaded")
	println("TwitchID: ", TwitchID)
	println("TwitchSecret: ", TwitchSecret)
	println("PostgresUrl: ", PostgresUrl)
	println("PostgresVectorUrl: ", PostgresVectorUrl)
	println("DiscordSecret: ", DiscordSecret)
	println("DiscordClientID: ", DiscordClientID)
	println("DiscordPublicKey: ", DiscordPublicKey)
	println("DiscordPermissions: ", DiscordPermissions)
	return nil
}

func getSecrets() error {
	token := os.Getenv("OP_SA")
	fmt.Println("Token: ", token)

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("Pedro Inegration", "v1.0.0"),
	)
	if err != nil {
		return fmt.Errorf("error creating 1password client: %v", err)
	}
	fmt.Println("Client: ", client)
	TwitchID, err = client.Secrets().Resolve(context.TODO(), "op://pedro/TWITCH_ID/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	TwitchSecret, err = client.Secrets().Resolve(context.TODO(), "op://pedro/TWITCH_SECRET/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	PostgresUrl, err = client.Secrets().Resolve(context.TODO(), "op://pedro/POSTGRES_URL/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	PostgresVectorUrl, err = client.Secrets().Resolve(context.TODO(), "op://pedro/POSTGRES_VECTOR_URL/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	DiscordSecret, err = client.Secrets().Resolve(context.TODO(), "op://pedro/DISCORD_SECRET/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	DiscordClientID, err = client.Secrets().Resolve(context.TODO(), "op://pedro/DISCORD_CLIENT_ID/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	DiscordPublicKey, err = client.Secrets().Resolve(context.TODO(), "op://pedro/DISCORD_PUBLIC_KEY/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	DiscordPermissions, err = client.Secrets().Resolve(context.TODO(), "op://pedro/DISCORD_PERMISSIONS/credential")
	if err != nil {
		return fmt.Errorf("error resolving secret: %v", err)
	}
	return nil
}
