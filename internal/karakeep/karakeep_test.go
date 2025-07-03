//go:build !integration

package karakeep

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateBookmark(t *testing.T) {
	

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/bookmarks" {
			t.Errorf("Expected path /v1/bookmarks, got %s", r.URL.Path)
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", authHeader)
		}

		var bookmark Bookmark
		if err := json.NewDecoder(r.Body).Decode(&bookmark); err != nil {
			t.Fatal(err)
		}

		if bookmark.URL != "https://example.com" {
			t.Errorf("Expected URL 'https://example.com', got '%s'", bookmark.URL)
		}

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
	}

	bookmark := &Bookmark{
		URL:   "https://example.com",
		Title: "Test Bookmark",
	}

	if err := client.CreateBookmark(bookmark); err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}
}
