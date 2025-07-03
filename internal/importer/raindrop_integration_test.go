//go:build integration

package importer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/thiswillbeyourgithub/rainbridge/internal/karakeep"
	"github.com/thiswillbeyourgithub/rainbridge/internal/raindrop"
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

	importer := NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}
}
