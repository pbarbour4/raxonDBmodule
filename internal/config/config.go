package config

import "os"

const (
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/raxon?sslmode=disable"
	defaultChannelName = "tokenization-channel"
)

type Config struct {
	DatabaseURL string
	ChannelName string
}

func Load() Config {
	return Config{
		DatabaseURL: getEnv("RAXON_DATABASE_URL", defaultDatabaseURL),
		ChannelName: getEnv("RAXON_CHANNEL_NAME", defaultChannelName),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
