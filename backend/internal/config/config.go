// Package config loads runtime configuration from environment variables (and a
// local .env file in development, via godotenv).
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every setting the app needs at startup.
type Config struct {
	Port string

	MongoURI string
	MongoDB  string

	MFAtlasBaseURL      string
	MFAtlasClientID     string
	MFAtlasClientSecret string

	// PublicWebhookURL is the https URL (e.g. an ngrok tunnel) that mf-atlas
	// should deliver webhooks to. Optional — the app works via polling alone
	// if this is left empty.
	PublicWebhookURL string

	// WebhookSecret is the `secret` returned once by
	// POST /api/webhooks/v1/subscriptions — set it here after registering a
	// subscription so incoming deliveries can be signature-verified.
	WebhookSecret string

	ReconcileInterval time.Duration
}

// Load reads configuration from the process environment, loading a .env file
// first if one is present (missing .env is not an error).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:             getEnv("MONGO_DB", "mf_demo"),
		MFAtlasBaseURL:      getEnv("MF_ATLAS_BASE_URL", ""),
		MFAtlasClientID:     getEnv("MF_ATLAS_CLIENT_ID", ""),
		MFAtlasClientSecret: getEnv("MF_ATLAS_CLIENT_SECRET", ""),
		PublicWebhookURL:    getEnv("PUBLIC_WEBHOOK_URL", ""),
		WebhookSecret:       getEnv("MF_ATLAS_WEBHOOK_SECRET", ""),
	}

	interval := getEnv("RECONCILE_INTERVAL_SECONDS", "20")
	d, err := time.ParseDuration(interval + "s")
	if err != nil {
		return nil, fmt.Errorf("invalid RECONCILE_INTERVAL_SECONDS: %w", err)
	}
	cfg.ReconcileInterval = d

	if cfg.MFAtlasBaseURL == "" || cfg.MFAtlasClientID == "" || cfg.MFAtlasClientSecret == "" {
		fmt.Println("WARNING: MF_ATLAS_BASE_URL / MF_ATLAS_CLIENT_ID / MF_ATLAS_CLIENT_SECRET are not fully set.")
		fmt.Println("The app will start, but every call to mf-atlas will fail until these are configured in .env.")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
