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

func TestKarakeepIntegration(t *testing.T) {
	_ = godotenv.Load("../../.env")

	karakeepToken := os.Getenv("KARAKEEP_API_TOKEN")
	if karakeepToken == "" {
		t.Skip("KARAKEEP_API_TOKEN not set, skipping integration test")
	}

	// Mock Raindrop.io server
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "[Test] Mocked Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 101, "title": "[Test] Mocked Bookmark", "link": "https://example.com/mock"}]}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer raindropServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")

	karakeepClient := karakeep.NewClient(karakeepToken)

	importer := NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}
}
