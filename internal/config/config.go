package config

import (
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration.
type Config struct {
	RaindropToken string
	KarakeepToken string
}

// Load loads the configuration from environment variables or a .env file.
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		RaindropToken: os.Getenv("RAINDROP_API_TOKEN"),
		KarakeepToken: os.Getenv("KARAKEEP_API_TOKEN"),
	}

	return cfg, nil
}
