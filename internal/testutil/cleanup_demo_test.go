//go:build integration

package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ashebanow/rainbridge/internal/karakeep"
)

func TestCleanupDemonstration(t *testing.T) {
	// Create a mock server that responds to CRUD operations
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			// Create operations
			if r.URL.Path == "/v1/bookmarks" {
				w.WriteHeader(http.StatusCreated)
				fmt.Fprintln(w, `{"id": "test-bookmark-123", "title": "[Test] Demo Bookmark", "url": "https://example.com"}`)
			} else if r.URL.Path == "/v1/lists" {
				w.WriteHeader(http.StatusCreated)
				fmt.Fprintln(w, `{"id": "test-list-456", "name": "[Test] Demo List"}`)
			}
		case "GET":
			// Read operations
			if r.URL.Path == "/v1/bookmarks" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `[{"id": "test-bookmark-123", "title": "[Test] Demo Bookmark", "url": "https://example.com"}]`)
			} else if r.URL.Path == "/v1/lists" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `[{"id": "test-list-456", "name": "[Test] Demo List"}]`)
			}
		case "DELETE":
			// Delete operations
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"success": true}`)
		}
	}))
	defer server.Close()

	// Create a client pointed at our mock server
	client := karakeep.NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	// Set up cleanup using the testutil helper
	helper := NewCleanupHelper()
	if helper.ShouldSkipCleanup() {
		helper.LogCleanupSkipped()
		return
	}

	t.Cleanup(func() {
		helper.LogCleanupStart()
		
		// Simulate cleanup - in real tests this would be more complex
		t.Log("Cleaning up demo bookmarks...")
		bookmarks, err := client.GetAllBookmarks()
		if err != nil {
			t.Logf("Failed to get bookmarks for cleanup: %v", err)
		} else {
			for _, bookmark := range bookmarks {
				if bookmark.Title == "[Test] Demo Bookmark" {
					t.Logf("Would delete bookmark: %s", bookmark.Title)
					// In real cleanup, we would call client.DeleteBookmark(bookmark.ID)
				}
			}
		}

		t.Log("Cleaning up demo lists...")
		lists, err := client.GetAllLists()
		if err != nil {
			t.Logf("Failed to get lists for cleanup: %v", err)
		} else {
			for _, list := range lists {
				if list.Name == "[Test] Demo List" {
					t.Logf("Would delete list: %s", list.Name)
					// In real cleanup, we would call client.DeleteList(list.ID)
				}
			}
		}
		
		helper.LogCleanupComplete()
	})

	// Create some test data
	bookmark := &karakeep.Bookmark{
		URL:   "https://example.com",
		Title: "[Test] Demo Bookmark",
	}
	
	createdBookmark, err := client.CreateBookmark(bookmark)
	if err != nil {
		t.Fatalf("Failed to create bookmark: %v", err)
	}
	t.Logf("Created bookmark: %s", createdBookmark.Title)

	list := &karakeep.List{
		Name: "[Test] Demo List",
	}
	
	createdList, err := client.CreateList(list)
	if err != nil {
		t.Fatalf("Failed to create list: %v", err)
	}
	t.Logf("Created list: %s", createdList.Name)

	// Test passes, cleanup will run automatically
}