//go:build integration

package karakeep

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestIntegrationCreateBookmark(t *testing.T) {
	_ = godotenv.Load("../../.env")

	token := os.Getenv("KARAKEEP_API_TOKEN")
	if token == "" {
		t.Skip("KARAKEEP_API_TOKEN not set, skipping integration test")
	}

	client := NewClient(token)

	bookmark := &Bookmark{
		URL:   "https://example.com/integration-test",
		Title: "[Test] Integration Test Bookmark",
		Tags:  []string{"test"},
	}

	if err := client.CreateBookmark(bookmark); err != nil {
		t.Fatalf("Failed to create bookmark: %v", err)
	}
}
