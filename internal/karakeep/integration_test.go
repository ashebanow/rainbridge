//go:build integration

package karakeep

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/ashebanow/rainbridge/internal/testutil"
	"github.com/joho/godotenv"
)

// setupCleanup sets up test data cleanup
func setupCleanup(t *testing.T, client *Client, testPrefix string) {
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
func cleanupBookmarks(client *Client, testPrefix string) error {
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
func cleanupLists(client *Client, testPrefix string) error {
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

func TestIntegrationCreateBookmark(t *testing.T) {
	_ = godotenv.Load("../../.env")

	token := os.Getenv("KARAKEEP_API_TOKEN")
	if token == "" {
		t.Skip("KARAKEEP_API_TOKEN not set, skipping integration test")
	}

	client := NewClient(token)

	// Set up cleanup
	setupCleanup(t, client, "[Test]")

	bookmark := &Bookmark{
		URL:   "https://example.com/integration-test",
		Title: "[Test] Integration Test Bookmark",
		Tags:  []string{"test"},
	}

	createdBookmark, err := client.CreateBookmark(bookmark)
	if err != nil {
		t.Fatalf("Failed to create bookmark: %v", err)
	}

	t.Logf("Created bookmark: %s (ID: %s)", createdBookmark.Title, createdBookmark.ID)
}

func TestIntegrationCreateList(t *testing.T) {
	_ = godotenv.Load("../../.env")

	token := os.Getenv("KARAKEEP_API_TOKEN")
	if token == "" {
		t.Skip("KARAKEEP_API_TOKEN not set, skipping integration test")
	}

	client := NewClient(token)

	// Set up cleanup
	setupCleanup(t, client, "[Test]")

	list := &List{
		Name: "[Test] Integration Test List",
	}

	createdList, err := client.CreateList(list)
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}

	t.Logf("Created list: %s (ID: %s)", createdList.Name, createdList.ID)
}
