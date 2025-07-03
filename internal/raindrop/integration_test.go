//go:build integration

package raindrop

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestIntegrationGetRaindrops(t *testing.T) {
	_ = godotenv.Load("../../.env")

	token := os.Getenv("RAINDROP_API_TOKEN")
	if token == "" {
		t.Skip("RAINDROP_API_TOKEN not set, skipping integration test")
	}

	client := NewClient(token)

	_, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("Failed to get raindrops: %v", err)
	}
}
