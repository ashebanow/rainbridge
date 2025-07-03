//go:build !integration

package importer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thiswillbeyourgithub/rainbridge/internal/karakeep"
	"github.com/thiswillbeyourgithub/rainbridge/internal/raindrop"
)

func TestRunImport(t *testing.T) {
	// Mock Raindrop.io server
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
				// For subsequent pages, return an empty items array to terminate pagination
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(w, `{"items": []}`)
			}
		} else {
			// For any other unhandled path, return a 404 with an empty JSON object
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`) // Ensure a body is always written
		}
	}))
	defer raindropServer.Close()

	// Mock Karakeep server
	karakeepServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/lists" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "list-123", "name": "Test Collection"}`)
		} else if r.URL.Path == "/v1/bookmarks" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "bookmark-456", "title": "Test Bookmark"}`)
		} else if r.URL.Path == "/v1/lists/list-123/bookmarks/bookmark-456" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, `{}`) // Always write a body
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintln(w, `{}`) // Always write a body
		}
	}))
	defer karakeepServer.Close()

	raindropClient := raindrop.NewClient("test-token")
	raindropClient.SetBaseURL(raindropServer.URL + "/rest/v1")
	raindropClient.SetHTTPClient(&http.Client{Transport: raindropServer.Client().Transport})

	karakeepClient := karakeep.NewClient("test-token")
	karakeepClient.SetBaseURL(karakeepServer.URL + "/v1")
	karakeepClient.SetHTTPClient(&http.Client{Transport: karakeepServer.Client().Transport})

	importer := NewImporter(raindropClient, karakeepClient)

	if err := importer.RunImport(); err != nil {
		t.Fatalf("RunImport failed: %v", err)
	}
}