//go:build !integration

package importer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ashebanow/rainbridge/internal/karakeep"
	"github.com/ashebanow/rainbridge/internal/raindrop"
)

// TestRunImportWithEmptyData tests importing with no bookmarks or collections
func TestRunImportWithEmptyData(t *testing.T) {
	// Mock servers returning empty data
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Should not receive any requests
		t.Errorf("Unexpected request to Karakeep: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, `{}`)
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed with empty data: %v", err)
	}
}

// TestRunImportWithNilBookmarks tests handling of nil/empty bookmarks
func TestRunImportWithNilBookmarks(t *testing.T) {
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Test Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			// Return empty items for bookmarks
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	listCreated := false
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" && r.Method == "POST" {
			listCreated = true
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if !listCreated {
		t.Error("Expected list to be created even with no bookmarks")
	}
}

// TestRunImportWithMissingURLs tests bookmarks with missing URLs
func TestRunImportWithMissingURLs(t *testing.T) {
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Test Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				w.WriteHeader(http.StatusOK)
				// Bookmark with empty URL
				fmt.Fprintln(w, `{"items": [{"_id": 101, "title": "No URL Bookmark", "link": ""}]}`)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	bookmarkRequests := 0
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
		} else if r.URL.Path == "/v1/bookmarks" {
			bookmarkRequests++
			// Should still try to create bookmark even with empty URL
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "bookmark-456", "title": "No URL Bookmark"}`)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if bookmarkRequests != 1 {
		t.Errorf("Expected 1 bookmark creation attempt, got %d", bookmarkRequests)
	}
}

// TestRunImportWithVeryLongURLs tests bookmarks with URLs exceeding limits
func TestRunImportWithVeryLongURLs(t *testing.T) {
	longURL := "https://example.com/" + generateLongString(2100) // Over 2000 char limit
	
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Test Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				w.WriteHeader(http.StatusOK)
				response := fmt.Sprintf(`{"items": [{"_id": 101, "title": "Long URL Bookmark", "link": "%s"}]}`, longURL)
				fmt.Fprintln(w, response)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	var receivedURL string
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
		} else if r.URL.Path == "/v1/bookmarks" {
			var bookmark karakeep.Bookmark
			json.NewDecoder(r.Body).Decode(&bookmark)
			receivedURL = bookmark.URL
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "bookmark-456", "title": "Long URL Bookmark"}`)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if receivedURL != longURL {
		t.Errorf("Expected long URL to be passed as-is, got different URL")
	}
}

// TestRunImportWithSpecialCharacters tests handling of special characters
func TestRunImportWithSpecialCharacters(t *testing.T) {
	specialTitle := `Special <>&"' chars \ / | ? * : ; , . ! @ # $ % ^ & ( ) [ ] { }`
	specialDesc := "Description with\nnewlines\tand\rspecial\bchars"
	
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"items": [{"_id": 1, "title": %q}]}`, specialTitle)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `{"items": [{"_id": 101, "title": %q, "excerpt": %q, "link": "https://example.com"}]}`, 
					specialTitle, specialDesc)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	var receivedListName, receivedBookmarkTitle, receivedBookmarkDesc string
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			var list karakeep.List
			json.NewDecoder(r.Body).Decode(&list)
			receivedListName = list.Name
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "list-123", "name": %q}`, list.Name)
		} else if r.URL.Path == "/v1/bookmarks" {
			var bookmark karakeep.Bookmark
			json.NewDecoder(r.Body).Decode(&bookmark)
			receivedBookmarkTitle = bookmark.Title
			receivedBookmarkDesc = bookmark.Description
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "bookmark-456", "title": %q}`, bookmark.Title)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if receivedListName != specialTitle {
		t.Errorf("Special characters in list name not preserved. Expected %q, got %q", specialTitle, receivedListName)
	}
	if receivedBookmarkTitle != specialTitle {
		t.Errorf("Special characters in bookmark title not preserved. Expected %q, got %q", specialTitle, receivedBookmarkTitle)
	}
	if receivedBookmarkDesc != specialDesc {
		t.Errorf("Special characters in bookmark description not preserved. Expected %q, got %q", specialDesc, receivedBookmarkDesc)
	}
}

// TestRunImportWithUnicode tests handling of Unicode characters
func TestRunImportWithUnicode(t *testing.T) {
	unicodeTitle := "Hello 世界 🌍 مرحبا мир ñoño"
	unicodeDesc := "Unicode: 🎉 emoji, 中文 Chinese, العربية Arabic"
	unicodeTags := []string{"emoji-🏷️", "中文标签", "тег"}
	
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"items": [{"_id": 1, "title": %q}]}`, unicodeTitle)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				w.WriteHeader(http.StatusOK)
				tagsJSON, _ := json.Marshal(unicodeTags)
				fmt.Fprintf(w, `{"items": [{"_id": 101, "title": %q, "excerpt": %q, "link": "https://example.com", "tags": %s}]}`, 
					unicodeTitle, unicodeDesc, string(tagsJSON))
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	var receivedBookmark karakeep.Bookmark
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "list-123", "name": %q}`, unicodeTitle)
		} else if r.URL.Path == "/v1/bookmarks" {
			json.NewDecoder(r.Body).Decode(&receivedBookmark)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "bookmark-456", "title": %q}`, receivedBookmark.Title)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if receivedBookmark.Title != unicodeTitle {
		t.Errorf("Unicode title not preserved. Expected %q, got %q", unicodeTitle, receivedBookmark.Title)
	}
	if receivedBookmark.Description != unicodeDesc {
		t.Errorf("Unicode description not preserved. Expected %q, got %q", unicodeDesc, receivedBookmark.Description)
	}
	if len(receivedBookmark.Tags) != len(unicodeTags) {
		t.Errorf("Unicode tags not preserved. Expected %d tags, got %d", len(unicodeTags), len(receivedBookmark.Tags))
	}
}

// TestRunImportWithInvalidURLs tests handling of invalid URLs
func TestRunImportWithInvalidURLs(t *testing.T) {
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Test Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": [
					{"_id": 101, "title": "Invalid URL 1", "link": "not-a-url"},
					{"_id": 102, "title": "Invalid URL 2", "link": "javascript:alert('test')"},
					{"_id": 103, "title": "Invalid URL 3", "link": "file:///etc/passwd"}
				]}`)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	bookmarkAttempts := 0
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
		} else if r.URL.Path == "/v1/bookmarks" {
			bookmarkAttempts++
			// Still try to create bookmarks with invalid URLs
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "bookmark-%d", "title": "Invalid URL"}`, bookmarkAttempts)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if bookmarkAttempts != 3 {
		t.Errorf("Expected 3 bookmark creation attempts, got %d", bookmarkAttempts)
	}
}

// TestRunImportWithDuplicateBookmarks tests handling of duplicate bookmarks
func TestRunImportWithDuplicateBookmarks(t *testing.T) {
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Test Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			if page == "0" {
				w.WriteHeader(http.StatusOK)
				// Return same bookmark multiple times
				fmt.Fprintln(w, `{"items": [
					{"_id": 101, "title": "Duplicate Bookmark", "link": "https://example.com/same"},
					{"_id": 102, "title": "Duplicate Bookmark", "link": "https://example.com/same"},
					{"_id": 103, "title": "Duplicate Bookmark", "link": "https://example.com/same"}
				]}`)
			} else {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	bookmarkCreations := 0
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
		} else if r.URL.Path == "/v1/bookmarks" {
			bookmarkCreations++
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "bookmark-%d", "title": "Duplicate Bookmark"}`, bookmarkCreations)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	// Should create all bookmarks even if duplicates
	if bookmarkCreations != 3 {
		t.Errorf("Expected 3 bookmark creations, got %d", bookmarkCreations)
	}
}

// TestRunImportWithDuplicateCollections tests handling of collections with identical names
func TestRunImportWithDuplicateCollections(t *testing.T) {
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [
				{"_id": 1, "title": "Duplicate Name"},
				{"_id": 2, "title": "Duplicate Name"},
				{"_id": 3, "title": "Duplicate Name"}
			]}`)
		} else if strings.HasPrefix(r.URL.Path, "/rest/v1/raindrops/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	listCreations := 0
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			listCreations++
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "list-%d", "name": "Duplicate Name"}`, listCreations)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	// Should create all lists even with duplicate names
	if listCreations != 3 {
		t.Errorf("Expected 3 list creations, got %d", listCreations)
	}
}

// TestRunImportWithLargeDataset tests handling of large datasets
func TestRunImportWithLargeDataset(t *testing.T) {
	const bookmarkCount = 1000
	
	raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/v1/collections" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Large Collection"}]}`)
		} else if r.URL.Path == "/rest/v1/raindrops/1" {
			page := r.URL.Query().Get("page")
			pageNum := 0
			if page != "" {
				fmt.Sscanf(page, "%d", &pageNum)
			}
			
			perPage := 50
			start := pageNum * perPage
			end := start + perPage
			if end > bookmarkCount {
				end = bookmarkCount
			}
			
			w.WriteHeader(http.StatusOK)
			if start >= bookmarkCount {
				fmt.Fprintln(w, `{"items": []}`)
			} else {
				items := make([]string, 0, end-start)
				for i := start; i < end; i++ {
					item := fmt.Sprintf(`{"_id": %d, "title": "Bookmark %d", "link": "https://example.com/bookmark/%d"}`, 
						i+1000, i, i)
					items = append(items, item)
				}
				fmt.Fprintf(w, `{"items": [%s]}`, strings.Join(items, ","))
			}
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer raindropServer.Close()

	var mu sync.Mutex
	bookmarksCreated := 0
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Large Collection"}`)
		} else if r.URL.Path == "/v1/bookmarks" {
			mu.Lock()
			bookmarksCreated++
			currentCount := bookmarksCreated
			mu.Unlock()
			
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, `{"id": "bookmark-%d", "title": "Bookmark"}`, currentCount)
		} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`)
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	
	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

	importer := NewImporter(raindropClient, karakeepClient)

	err := importer.RunImport()
	if err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}

	if bookmarksCreated != bookmarkCount {
		t.Errorf("Expected %d bookmarks created, got %d", bookmarkCount, bookmarksCreated)
	}
}

// TestRunImportWithAPIErrors tests error handling when APIs return errors
func TestRunImportWithAPIErrors(t *testing.T) {
	testCases := []struct {
		name string
		karakeepListError bool
		karakeepBookmarkError bool
		expectError bool
	}{
		{
			name: "List creation fails",
			karakeepListError: true,
			expectError: false, // Should continue despite list creation failure
		},
		{
			name: "Bookmark creation fails",
			karakeepBookmarkError: true,
			expectError: false, // Should continue despite bookmark creation failure
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			raindropServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/rest/v1/collections" {
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Test Collection"}]}`)
				} else if r.URL.Path == "/rest/v1/raindrops/1" {
					page := r.URL.Query().Get("page")
					if page == "0" {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, `{"items": [{"_id": 101, "title": "Test Bookmark", "link": "https://example.com"}]}`)
					} else {
						w.WriteHeader(http.StatusOK)
						fmt.Fprintln(w, `{"items": []}`)
					}
				} else {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintln(w, `{}`)
				}
			}))
			defer raindropServer.Close()

			karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v1/lists" && tc.karakeepListError {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintln(w, `{"error": "Internal server error"}`)
				} else if r.URL.Path == "/v1/lists" {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
				} else if r.URL.Path == "/v1/bookmarks" && tc.karakeepBookmarkError {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, `{"error": "Invalid bookmark data"}`)
				} else if r.URL.Path == "/v1/bookmarks" {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprintln(w, `{"id": "bookmark-456", "title": "Test Bookmark"}`)
				} else if strings.HasPrefix(r.URL.Path, "/v1/lists/list-123/bookmarks/") {
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, `{}`)
				} else {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintln(w, `{}`)
				}
			}))
			defer karakeepServer.Close()

			raindropClient := raindrop.NewClient("test-token")
			raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
			
			karakeepClient := karakeep.NewClient("test-token")
			karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")

			importer := NewImporter(raindropClient, karakeepClient)

			err := importer.RunImport()
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}