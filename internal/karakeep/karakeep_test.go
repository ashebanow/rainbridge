//go:build !integration

package karakeep

import (
	"encoding/json"
	"fmt"
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
		fmt.Fprintln(w, `{"id": "bookmark-123"}`)
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

	if _, err := client.CreateBookmark(bookmark); err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}
}

func TestCreateList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/lists" {
			t.Errorf("Expected path /v1/lists, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, `{"id": "list-123"}`)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
	}

	list := &List{Name: "Test List"}

	if _, err := client.CreateList(list); err != nil {
		t.Fatalf("CreateList failed: %v", err)
	}
}

func TestAddBookmarkToList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/v1/lists/list-123/bookmarks/bookmark-456" {
			t.Errorf("Expected path /v1/lists/list-123/bookmarks/bookmark-456, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
	}

	if err := client.AddBookmarkToList("bookmark-456", "list-123"); err != nil {
		t.Fatalf("AddBookmarkToList failed: %v", err)
	}
}
