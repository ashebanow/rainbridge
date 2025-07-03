//go:build !integration

package raindrop

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetRaindrops(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/raindrops/0" {
			t.Errorf("Expected path /rest/v1/raindrops/0, got %s", r.URL.Path)
		}

		page := r.URL.Query().Get("page")
		if page == "0" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Page 1"}]}`)
		} else if page == "1" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": [{"_id": 2, "title": "Page 2"}]}`)
		} else {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{"items": []}`)
		}
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/rest/v1",
		httpClient: server.Client(),
		token:      "test-token",
	}

	raindrops, err := client.GetRaindrops()
	if err != nil {
		t.Fatalf("GetRaindrops failed: %v", err)
	}

	if len(raindrops) != 2 {
		t.Errorf("Expected 2 raindrops, got %d", len(raindrops))
	}

	if raindrops[0].Title != "Page 1" {
		t.Errorf("Expected first raindrop title 'Page 1', got '%s'", raindrops[0].Title)
	}
}

func TestGetCollections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/v1/collections" {
			t.Errorf("Expected path /rest/v1/collections, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"items": [{"_id": 123, "title": "Test Collection"}]}`)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/rest/v1",
		httpClient: server.Client(),
		token:      "test-token",
	}

	collections, err := client.GetCollections()
	if err != nil {
		t.Fatalf("GetCollections failed: %v", err)
	}

	if len(collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(collections))
	}

	if collections[0].Title != "Test Collection" {
		t.Errorf("Expected collection title 'Test Collection', got '%s'", collections[0].Title)
	}
}
