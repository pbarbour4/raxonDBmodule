package config

import "os"

const (
	defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/raxon?sslmode=disable"
	defaultChannelName = "tokenization-channel"
	defaultPort        = "8080"
	// SessionSecret must be 32+ bytes in production; override via RAXON_SESSION_SECRET.
	defaultSessionSecret = "change-this-to-a-32-byte-secret!"
)

type Config struct {
	DatabaseURL      string
	ChannelName      string
	Port             string
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCRedirectURL  string
	SessionSecret    string
	StaticDir        string
	AuthMode         string
	Environment      string
	SessionTTL       string
}

func Load() Config {
	return Config{
		DatabaseURL:      getEnv("RAXON_DATABASE_URL", defaultDatabaseURL),
		ChannelName:      getEnv("RAXON_CHANNEL_NAME", defaultChannelName),
		Port:             getEnv("RAXON_PORT", defaultPort),
		OIDCIssuerURL:    getEnv("RAXON_OIDC_ISSUER_URL", ""),
		OIDCClientID:     getEnv("RAXON_OIDC_CLIENT_ID", ""),
		OIDCClientSecret: getEnv("RAXON_OIDC_CLIENT_SECRET", ""),
		OIDCRedirectURL:  getEnv("RAXON_OIDC_REDIRECT_URL", "http://localhost:8080/auth/callback"),
		SessionSecret:    getEnv("RAXON_SESSION_SECRET", defaultSessionSecret),
		StaticDir:        getEnv("RAXON_STATIC_DIR", "Stitch-files"),
		AuthMode:         getEnv("RAXON_AUTH_MODE", "oidc"),
		Environment:      getEnv("RAXON_ENV", "development"),
		SessionTTL:       getEnv("RAXON_SESSION_TTL", "8h"),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
