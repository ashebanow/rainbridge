//go:build !integration

package raindrop

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGetRaindropsWithEmptyResponse tests handling of empty API responses
func TestGetRaindropsWithEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("GetRaindrops failed: %v", err)
	}

	if len(raindrops) != 0 {
		t.Errorf("Expected 0 raindrops, got %d", len(raindrops))
	}
}

// TestGetRaindropsWithMalformedJSON tests handling of malformed JSON responses
func TestGetRaindropsWithMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Unclosed JSON}`)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	_, err := client.GetRaindrops()
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	}
}

// TestGetRaindropsWithUnicodeData tests handling of Unicode data
func TestGetRaindropsWithUnicodeData(t *testing.T) {
	unicodeTitle := "Hello 世界 🌍 مرحبا мир"
	unicodeTags := []string{"emoji-🏷️", "中文标签", "тег"}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "0" || page == "" {
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"_id":     1,
						"title":   unicodeTitle,
						"link":    "https://example.com/unicode/世界",
						"excerpt": "Unicode description: 🎉",
						"tags":    unicodeTags,
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			// Return empty items for subsequent pages
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("GetRaindrops failed: %v", err)
	}

	if len(raindrops) != 1 {
		t.Fatalf("Expected 1 raindrop, got %d", len(raindrops))
	}

	if raindrops[0].Title != unicodeTitle {
		t.Errorf("Unicode title not preserved. Expected %q, got %q", unicodeTitle, raindrops[0].Title)
	}

	if len(raindrops[0].Tags) != len(unicodeTags) {
		t.Errorf("Expected %d tags, got %d", len(unicodeTags), len(raindrops[0].Tags))
	}
}

// TestGetRaindropsWithLargePageSize tests handling of large datasets with pagination
func TestGetRaindropsWithLargePageSize(t *testing.T) {
	totalItems := 250 // More than typical page size
	itemsServed := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		perPage := 50
		
		pageNum := 0
		if page != "" {
			fmt.Sscanf(page, "%d", &pageNum)
		}
		
		start := pageNum * perPage
		end := start + perPage
		if end > totalItems {
			end = totalItems
		}
		
		items := []map[string]interface{}{}
		for i := start; i < end; i++ {
			items = append(items, map[string]interface{}{
				"_id":   i + 1,
				"title": fmt.Sprintf("Bookmark %d", i),
				"link":  fmt.Sprintf("https://example.com/bookmark/%d", i),
			})
			itemsServed++
		}
		
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{"items": items}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("GetRaindrops failed: %v", err)
	}

	if len(raindrops) != totalItems {
		t.Errorf("Expected %d raindrops, got %d", totalItems, len(raindrops))
	}

	// Verify all items are unique and properly ordered
	seen := make(map[int64]bool)
	for i, raindrop := range raindrops {
		expectedID := int64(i + 1)
		if raindrop.ID != expectedID {
			t.Errorf("Expected ID %d at position %d, got %d", expectedID, i, raindrop.ID)
		}
		if seen[raindrop.ID] {
			t.Errorf("Duplicate ID found: %d", raindrop.ID)
		}
		seen[raindrop.ID] = true
	}
}

// TestGetCollectionsWithEmptyResponse tests handling of no collections
func TestGetCollectionsWithEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	collections, err := client.GetCollections()
	if err != nil {
		t.Fatalf("GetCollections failed: %v", err)
	}

	if len(collections) != 0 {
		t.Errorf("Expected 0 collections, got %d", len(collections))
	}
}

// TestGetCollectionsWithSpecialCharacters tests collections with special characters
func TestGetCollectionsWithSpecialCharacters(t *testing.T) {
	specialTitle := `Special <>&"' Collection \ / | ? *`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"_id":   1,
					"title": specialTitle,
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	collections, err := client.GetCollections()
	if err != nil {
		t.Fatalf("GetCollections failed: %v", err)
	}

	if len(collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(collections))
	}

	if collections[0].Title != specialTitle {
		t.Errorf("Special characters not preserved. Expected %q, got %q", specialTitle, collections[0].Title)
	}
}

// TestGetRaindropsWithNullFields tests handling of null/missing fields
func TestGetRaindropsWithNullFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "0" || page == "" {
			w.WriteHeader(http.StatusOK)
			// JSON with various null/missing fields
			fmt.Fprintln(w, `{
				"items": [
					{
						"_id": 1,
						"title": null,
						"link": "https://example.com",
						"excerpt": null,
						"tags": null
					},
					{
						"_id": 2,
						"title": "Valid Title",
						"link": null,
						"excerpt": "Valid excerpt"
					},
					{
						"_id": 3,
						"title": "",
						"link": "",
						"excerpt": "",
						"tags": []
					}
				]
			}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("GetRaindrops failed: %v", err)
	}

	if len(raindrops) != 3 {
		t.Fatalf("Expected 3 raindrops, got %d", len(raindrops))
	}

	// Check handling of null values
	if raindrops[0].Title != "" {
		t.Errorf("Expected empty title for null, got %q", raindrops[0].Title)
	}
	if raindrops[0].Tags != nil {
		t.Errorf("Expected nil tags for null, got %v", raindrops[0].Tags)
	}

	if raindrops[1].Link != "" {
		t.Errorf("Expected empty link for null, got %q", raindrops[1].Link)
	}

	// Check empty strings are preserved
	if raindrops[2].Title != "" || raindrops[2].Link != "" || raindrops[2].Excerpt != "" {
		t.Error("Expected empty strings to be preserved")
	}
	if len(raindrops[2].Tags) != 0 {
		t.Errorf("Expected empty tags array, got %v", raindrops[2].Tags)
	}
}

// TestGetRaindropsWithExtremelyLongContent tests handling of very long content
func TestGetRaindropsWithExtremelyLongContent(t *testing.T) {
	longTitle := strings.Repeat("a", 5000)
	longExcerpt := strings.Repeat("b", 10000)
	longURL := "https://example.com/" + strings.Repeat("c", 3000)
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "0" || page == "" {
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"_id":     1,
						"title":   longTitle,
						"link":    longURL,
						"excerpt": longExcerpt,
						"tags":    []string{strings.Repeat("tag", 1000)},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		}
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL)

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("GetRaindrops failed: %v", err)
	}

	if len(raindrops) != 1 {
		t.Fatalf("Expected 1 raindrop, got %d", len(raindrops))
	}

	// Verify long content is preserved
	if len(raindrops[0].Title) != 5000 {
		t.Errorf("Expected title length 5000, got %d", len(raindrops[0].Title))
	}
	if len(raindrops[0].Excerpt) != 10000 {
		t.Errorf("Expected excerpt length 10000, got %d", len(raindrops[0].Excerpt))
	}
	if len(raindrops[0].Link) != len("https://example.com/")+3000 {
		t.Errorf("Expected URL length %d, got %d", len("https://example.com/")+3000, len(raindrops[0].Link))
	}
}

// TestGetRaindropsWithInvalidToken tests behavior with invalid authentication
func TestGetRaindropsWithInvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, `{"error": "Invalid token"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer server.Close()

	// Test with invalid token
	client := NewClient("invalid-token")
	client.SetBaseURL(server.URL)

	_, err := client.GetRaindrops()
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 error, got: %v", err)
	}

	// Test with valid token
	client = NewClient("valid-token")
	client.SetBaseURL(server.URL)

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Errorf("Expected success with valid token, got error: %v", err)
	}
	if len(raindrops) != 0 {
		t.Errorf("Expected 0 raindrops, got %d", len(raindrops))
	}
}