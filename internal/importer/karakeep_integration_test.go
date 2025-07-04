//go:build integration

package importer

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
	"github.com/ashebanow/rainbridge/internal/testutil"
	"github.com/joho/godotenv"
)

// setupCleanup sets up test data cleanup
func setupCleanup(t *testing.T, client *karakeep.Client, testPrefix string) {
	helper := testutil.NewCleanupHelper()
	if helper.ShouldSkipCleanup() {
		helper.LogCleanupSkipped()
		return
	}
	
	t.Cleanup(func() {
		helper.LogCleanupStart()
		
		// Clean up bookmarks
		if err := cleanupBookmarks(client, testPrefix); err != nil {
			log.Printf("Failed to cleanup bookmarks: %v", err)
		}
		
		// Clean up lists
		if err := cleanupLists(client, testPrefix); err != nil {
			log.Printf("Failed to cleanup lists: %v", err)
		}
		
		helper.LogCleanupComplete()
	})
}

// cleanupBookmarks removes all test bookmarks
func cleanupBookmarks(client *karakeep.Client, testPrefix string) error {
	log.Println("Cleaning up test bookmarks...")
	
	bookmarks, err := client.GetAllBookmarks()
	if err != nil {
		return err
	}
	
	deletedCount := 0
	for _, bookmark := range bookmarks {
		// Only delete bookmarks that match our test prefix
		if strings.HasPrefix(bookmark.Title, testPrefix) {
			log.Printf("Deleting test bookmark: %s (ID: %s)", bookmark.Title, bookmark.ID)
			if err := client.DeleteBookmark(bookmark.ID); err != nil {
				log.Printf("Failed to delete bookmark %s: %v", bookmark.ID, err)
			} else {
				deletedCount++
			}
		}
	}
	
	log.Printf("Deleted %d test bookmarks", deletedCount)
	return nil
}

// cleanupLists removes all test lists
func cleanupLists(client *karakeep.Client, testPrefix string) error {
	log.Println("Cleaning up test lists...")
	
	lists, err := client.GetAllLists()
	if err != nil {
		return err
	}
	
	deletedCount := 0
	for _, list := range lists {
		// Only delete lists that match our test prefix
		if strings.HasPrefix(list.Name, testPrefix) {
			log.Printf("Deleting test list: %s (ID: %s)", list.Name, list.ID)
			if err := client.DeleteList(list.ID); err != nil {
				log.Printf("Failed to delete list %s: %v", list.ID, err)
			} else {
				deletedCount++
			}
		}
	}
	
	log.Printf("Deleted %d test lists", deletedCount)
	return nil
}

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

	// Set up cleanup
	setupCleanup(t, karakeepClient, "[Test]")

	importer := NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}
}
