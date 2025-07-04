//go:build !integration

package karakeep

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateBookmarkWithEmptyFields tests creating bookmarks with empty/missing fields
func TestCreateBookmarkWithEmptyFields(t *testing.T) {
	testCases := []struct {
		name     string
		bookmark *Bookmark
		expectError bool
	}{
		{
			name: "Empty URL",
			bookmark: &Bookmark{
				URL:   "",
				Title: "Test",
			},
			expectError: false, // API should handle validation
		},
		{
			name: "Empty Title",
			bookmark: &Bookmark{
				URL:   "https://example.com",
				Title: "",
			},
			expectError: false,
		},
		{
			name: "Nil Tags",
			bookmark: &Bookmark{
				URL:   "https://example.com",
				Title: "Test",
				Tags:  nil,
			},
			expectError: false,
		},
		{
			name: "Empty Tags Array",
			bookmark: &Bookmark{
				URL:   "https://example.com",
				Title: "Test",
				Tags:  []string{},
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.URL.Path != "/v1/bookmarks" {
					t.Errorf("Unexpected request: %s %s", r.Method, r.URL.Path)
				}

				var received Bookmark
				json.NewDecoder(r.Body).Decode(&received)

				// Server accepts the request but may return error
				if tc.expectError {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, `{"error": "Invalid bookmark data"}`)
				} else {
					w.WriteHeader(http.StatusCreated)
					received.ID = "bookmark-123"
					json.NewEncoder(w).Encode(received)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.SetBaseURL(server.URL + "/v1")

			_, err := client.CreateBookmark(tc.bookmark)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCreateBookmarkWithSpecialCharacters tests handling of special characters
func TestCreateBookmarkWithSpecialCharacters(t *testing.T) {
	specialBookmark := &Bookmark{
		URL:         "https://example.com/path?query=<>&\"'",
		Title:       `Special <>&"' chars \ / | ? * : ; , . ! @ # $ % ^ & ( ) [ ] { }`,
		Description: "Line 1\nLine 2\tTabbed\rCarriage return",
		Tags:        []string{"tag<>", "tag&amp;", "tag\"quote\""},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received Bookmark
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)

		// Verify special characters are preserved in JSON
		if received.Title != specialBookmark.Title {
			t.Errorf("Special characters not preserved in title. Expected %q, got %q", 
				specialBookmark.Title, received.Title)
		}

		w.WriteHeader(http.StatusCreated)
		received.ID = "bookmark-123"
		json.NewEncoder(w).Encode(received)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateBookmark(specialBookmark)
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}

	if created.Title != specialBookmark.Title {
		t.Errorf("Special characters not preserved. Expected %q, got %q", 
			specialBookmark.Title, created.Title)
	}
}

// TestCreateBookmarkWithUnicode tests handling of Unicode content
func TestCreateBookmarkWithUnicode(t *testing.T) {
	unicodeBookmark := &Bookmark{
		URL:         "https://example.com/世界/مرحبا",
		Title:       "Hello 世界 🌍 مرحبا мир ñoño",
		Description: "Unicode: 🎉 emoji, 中文 Chinese, العربية Arabic, Русский Russian",
		Tags:        []string{"emoji-🏷️", "中文标签", "тег", "العلامة"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received Bookmark
		json.NewDecoder(r.Body).Decode(&received)

		// Verify Unicode is preserved
		if received.Title != unicodeBookmark.Title {
			t.Errorf("Unicode not preserved in title")
		}
		if len(received.Tags) != len(unicodeBookmark.Tags) {
			t.Errorf("Unicode tags not preserved")
		}

		w.WriteHeader(http.StatusCreated)
		received.ID = "bookmark-123"
		json.NewEncoder(w).Encode(received)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateBookmark(unicodeBookmark)
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}

	if created.Title != unicodeBookmark.Title {
		t.Errorf("Unicode title not preserved")
	}
}

// TestCreateBookmarkWithVeryLongContent tests handling of extremely long content
func TestCreateBookmarkWithVeryLongContent(t *testing.T) {
	longBookmark := &Bookmark{
		URL:         "https://example.com/" + strings.Repeat("a", 2000),
		Title:       strings.Repeat("Title ", 500), // 3000 chars
		Description: strings.Repeat("Description ", 1000), // 12000 chars
		Tags:        make([]string, 100),
	}
	
	// Fill tags with long strings
	for i := range longBookmark.Tags {
		longBookmark.Tags[i] = fmt.Sprintf("tag-%d-%s", i, strings.Repeat("x", 50))
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received Bookmark
		body, _ := io.ReadAll(r.Body)
		
		// Check that large payload was received
		if len(body) < 10000 {
			t.Errorf("Expected large payload, got %d bytes", len(body))
		}
		
		json.Unmarshal(body, &received)

		w.WriteHeader(http.StatusCreated)
		received.ID = "bookmark-123"
		json.NewEncoder(w).Encode(received)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateBookmark(longBookmark)
	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}

	if len(created.URL) != len(longBookmark.URL) {
		t.Errorf("Long URL not preserved. Expected length %d, got %d", 
			len(longBookmark.URL), len(created.URL))
	}
}

// TestCreateListWithEmptyName tests creating a list with empty name
func TestCreateListWithEmptyName(t *testing.T) {
	emptyList := &List{
		Name: "",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received List
		json.NewDecoder(r.Body).Decode(&received)

		if received.Name != "" {
			t.Errorf("Expected empty name, got %q", received.Name)
		}

		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, `{"id": "list-123", "name": ""}`)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateList(emptyList)
	if err != nil {
		t.Fatalf("CreateList failed: %v", err)
	}

	if created.Name != "" {
		t.Errorf("Expected empty name, got %q", created.Name)
	}
}

// TestCreateListWithSpecialCharacters tests lists with special characters
func TestCreateListWithSpecialCharacters(t *testing.T) {
	specialList := &List{
		Name: `Special <>&"' List \ / | ? * : ; , . ! @ # $ % ^ & ( ) [ ] { }`,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received List
		json.NewDecoder(r.Body).Decode(&received)

		if received.Name != specialList.Name {
			t.Errorf("Special characters not preserved")
		}

		w.WriteHeader(http.StatusCreated)
		received.ID = "list-123"
		json.NewEncoder(w).Encode(received)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateList(specialList)
	if err != nil {
		t.Fatalf("CreateList failed: %v", err)
	}

	if created.Name != specialList.Name {
		t.Errorf("Special characters not preserved. Expected %q, got %q", 
			specialList.Name, created.Name)
	}
}

// TestCreateListWithUnicode tests lists with Unicode names
func TestCreateListWithUnicode(t *testing.T) {
	unicodeList := &List{
		Name: "List 世界 🌍 مرحبا мир ñoño",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var received List
		json.NewDecoder(r.Body).Decode(&received)

		if received.Name != unicodeList.Name {
			t.Errorf("Unicode not preserved")
		}

		w.WriteHeader(http.StatusCreated)
		received.ID = "list-123"
		json.NewEncoder(w).Encode(received)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateList(unicodeList)
	if err != nil {
		t.Fatalf("CreateList failed: %v", err)
	}

	if created.Name != unicodeList.Name {
		t.Errorf("Unicode not preserved. Expected %q, got %q", 
			unicodeList.Name, created.Name)
	}
}

// TestAddBookmarkToListWithInvalidIDs tests error handling for invalid IDs
func TestAddBookmarkToListWithInvalidIDs(t *testing.T) {
	testCases := []struct {
		name       string
		bookmarkID string
		listID     string
		statusCode int
		expectError bool
	}{
		{
			name:       "Empty bookmark ID",
			bookmarkID: "",
			listID:     "list-123",
			statusCode: http.StatusBadRequest,
			expectError: true,
		},
		{
			name:       "Empty list ID",
			bookmarkID: "bookmark-123",
			listID:     "",
			statusCode: http.StatusBadRequest,
			expectError: true,
		},
		{
			name:       "Non-existent bookmark",
			bookmarkID: "non-existent",
			listID:     "list-123",
			statusCode: http.StatusNotFound,
			expectError: true,
		},
		{
			name:       "Non-existent list",
			bookmarkID: "bookmark-123",
			listID:     "non-existent",
			statusCode: http.StatusNotFound,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				expectedPath := fmt.Sprintf("/v1/lists/%s/bookmarks/%s", tc.listID, tc.bookmarkID)
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tc.statusCode)
				if tc.statusCode != http.StatusOK {
					fmt.Fprintf(w, `{"error": "Error for test case %s"}`, tc.name)
				} else {
					fmt.Fprintln(w, `{}`)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.SetBaseURL(server.URL + "/v1")

			err := client.AddBookmarkToList(tc.bookmarkID, tc.listID)
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestCreateBookmarkWithMalformedResponse tests handling of malformed API responses
func TestCreateBookmarkWithMalformedResponse(t *testing.T) {
	bookmark := &Bookmark{
		URL:   "https://example.com",
		Title: "Test",
	}

	testCases := []struct {
		name     string
		response string
		expectError bool
	}{
		{
			name:     "Invalid JSON",
			response: `{"id": "bookmark-123", "title": "Unclosed JSON}`,
			expectError: true,
		},
		{
			name:     "Empty response",
			response: ``,
			expectError: true,
		},
		{
			name:     "HTML response instead of JSON",
			response: `<html><body>Error</body></html>`,
			expectError: true,
		},
		{
			name:     "Null response",
			response: `null`,
			expectError: false, // json.Unmarshal handles null
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
				fmt.Fprint(w, tc.response)
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.SetBaseURL(server.URL + "/v1")

			_, err := client.CreateBookmark(bookmark)
			if tc.expectError && err == nil {
				t.Error("Expected error for malformed response, got nil")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestClientWithInvalidAuthentication tests behavior with invalid tokens
func TestClientWithInvalidAuthentication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, `{"error": "Unauthorized"}`)
			return
		}
		
		// Valid response for authorized requests
		if r.Method == "POST" && r.URL.Path == "/v1/bookmarks" {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, `{"id": "bookmark-123", "title": "Test"}`)
		}
	}))
	defer server.Close()

	// Test with invalid token
	client := NewClient("invalid-token")
	client.SetBaseURL(server.URL + "/v1")

	bookmark := &Bookmark{URL: "https://example.com", Title: "Test"}
	_, err := client.CreateBookmark(bookmark)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected 401 error, got: %v", err)
	}

	// Test with valid token
	client = NewClient("valid-token")
	client.SetBaseURL(server.URL + "/v1")

	created, err := client.CreateBookmark(bookmark)
	if err != nil {
		t.Errorf("Expected success with valid token, got error: %v", err)
	}
	if created.ID != "bookmark-123" {
		t.Errorf("Expected bookmark ID 'bookmark-123', got %q", created.ID)
	}
}