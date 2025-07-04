//go:build integration

package importer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
	"github.com/joho/godotenv"
)

func TestRaindropIntegration(t *testing.T) {
	_ = godotenv.Load("../../.env")

	raindropToken := os.Getenv("RAINDROP_API_TOKEN")
	if raindropToken == "" {
		t.Skip("RAINDROP_API_TOKEN not set, skipping integration test")
	}

	// Mock Karakeep server
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, `{"id": "mock-id"}`)
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient(raindropToken)
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL)

	// Note: Since we're using a mock Karakeep server, we can't use the real cleanup
	// as it would try to fetch/delete from the mock server.
	// In a real Raindrop integration test that creates real data in Karakeep,
	// you would use:
	// cleaner := testutil.NewTestDataCleaner(karakeepClient, "[Test]")
	// cleaner.RegisterCleanup(t)

	importer := NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}
}
