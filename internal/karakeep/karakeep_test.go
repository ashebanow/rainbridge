//go:build !integration

package karakeep

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// MockSleeper implements Sleeper interface for testing
type MockSleeper struct {
	sleeps []time.Duration
	mu     sync.Mutex
}

func (m *MockSleeper) Sleep(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sleeps = append(m.sleeps, duration)
	// Don't actually sleep, just record the duration
}

func (m *MockSleeper) GetSleeps() []time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]time.Duration, len(m.sleeps))
	copy(result, m.sleeps)
	return result
}

func (m *MockSleeper) TotalSleepTime() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	var total time.Duration
	for _, sleep := range m.sleeps {
		total += sleep
	}
	return total
}

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

func TestCreateBookmarkRateLimitingWithRetry(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		
		// Return 429 for the first 2 attempts, then success
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, `{"id": "bookmark-123", "url": "https://example.com", "title": "Test"}`)
	}))
	defer server.Close()

	mockSleeper := &MockSleeper{}
	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    mockSleeper,
	}

	bookmark := &Bookmark{
		URL:   "https://example.com",
		Title: "Test Bookmark",
	}

	start := time.Now()
	createdBookmark, err := client.CreateBookmark(bookmark)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("CreateBookmark failed: %v", err)
	}

	if createdBookmark.ID != "bookmark-123" {
		t.Errorf("Expected bookmark ID 'bookmark-123', got '%s'", createdBookmark.ID)
	}

	// Verify we made 3 attempts
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Verify that we recorded the proper sleep delays
	sleeps := mockSleeper.GetSleeps()
	if len(sleeps) != 2 {
		t.Errorf("Expected 2 sleep calls, got %d", len(sleeps))
	}

	// Verify exponential backoff pattern (approximately 1s, 2s with jitter)
	if len(sleeps) > 0 {
		// First delay should be around 1 second (+/- 10% jitter)
		if sleeps[0] < 900*time.Millisecond || sleeps[0] > 1100*time.Millisecond {
			t.Errorf("First delay should be ~1s, got %v", sleeps[0])
		}
	}
	if len(sleeps) > 1 {
		// Second delay should be around 2 seconds (+/- 10% jitter)
		if sleeps[1] < 1800*time.Millisecond || sleeps[1] > 2200*time.Millisecond {
			t.Errorf("Second delay should be ~2s, got %v", sleeps[1])
		}
	}

	// Test should complete quickly since we're not actually sleeping
	if elapsed > 100*time.Millisecond {
		t.Errorf("Test took too long without actual sleeping: %v", elapsed)
	}
}

func TestCreateBookmarkRateLimitingMaxRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 429
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	mockSleeper := &MockSleeper{}
	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    mockSleeper,
	}

	bookmark := &Bookmark{
		URL:   "https://example.com",
		Title: "Test Bookmark",
	}

	start := time.Now()
	_, err := client.CreateBookmark(bookmark)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	expectedError := "rate limited after 5 retries"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error containing '%s', got '%s'", expectedError, err.Error())
	}

	// Verify we made 5 retry attempts (6 total requests)
	sleeps := mockSleeper.GetSleeps()
	if len(sleeps) != 5 {
		t.Errorf("Expected 5 sleep calls (max retries), got %d", len(sleeps))
	}

	// Test should complete quickly since we're not actually sleeping
	if elapsed > 100*time.Millisecond {
		t.Errorf("Test took too long without actual sleeping: %v", elapsed)
	}
}

func TestCreateListRateLimitingWithRetry(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		
		// Return 429 for the first 2 attempts, then success
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintln(w, `{"id": "list-123", "name": "Test List"}`)
	}))
	defer server.Close()

	mockSleeper := &MockSleeper{}
	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    mockSleeper,
	}

	list := &List{Name: "Test List"}

	start := time.Now()
	createdList, err := client.CreateList(list)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("CreateList failed: %v", err)
	}

	if createdList.ID != "list-123" {
		t.Errorf("Expected list ID 'list-123', got '%s'", createdList.ID)
	}

	// Verify we made 3 attempts
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Verify that we recorded the proper sleep delays
	sleeps := mockSleeper.GetSleeps()
	if len(sleeps) != 2 {
		t.Errorf("Expected 2 sleep calls, got %d", len(sleeps))
	}

	// Verify exponential backoff pattern (approximately 1s, 2s with jitter)
	if len(sleeps) > 0 {
		// First delay should be around 1 second (+/- 10% jitter)
		if sleeps[0] < 900*time.Millisecond || sleeps[0] > 1100*time.Millisecond {
			t.Errorf("First delay should be ~1s, got %v", sleeps[0])
		}
	}
	if len(sleeps) > 1 {
		// Second delay should be around 2 seconds (+/- 10% jitter)
		if sleeps[1] < 1800*time.Millisecond || sleeps[1] > 2200*time.Millisecond {
			t.Errorf("Second delay should be ~2s, got %v", sleeps[1])
		}
	}

	// Test should complete quickly since we're not actually sleeping
	if elapsed > 100*time.Millisecond {
		t.Errorf("Test took too long without actual sleeping: %v", elapsed)
	}
}

func TestAddBookmarkToListRateLimitingWithRetry(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		
		// Return 429 for the first 2 attempts, then success
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockSleeper := &MockSleeper{}
	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    mockSleeper,
	}

	start := time.Now()
	err := client.AddBookmarkToList("bookmark-456", "list-123")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("AddBookmarkToList failed: %v", err)
	}

	// Verify we made 3 attempts
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", attemptCount)
	}

	// Verify that we recorded the proper sleep delays
	sleeps := mockSleeper.GetSleeps()
	if len(sleeps) != 2 {
		t.Errorf("Expected 2 sleep calls, got %d", len(sleeps))
	}

	// Verify exponential backoff pattern (approximately 1s, 2s with jitter)
	if len(sleeps) > 0 {
		// First delay should be around 1 second (+/- 10% jitter)
		if sleeps[0] < 900*time.Millisecond || sleeps[0] > 1100*time.Millisecond {
			t.Errorf("First delay should be ~1s, got %v", sleeps[0])
		}
	}
	if len(sleeps) > 1 {
		// Second delay should be around 2 seconds (+/- 10% jitter)
		if sleeps[1] < 1800*time.Millisecond || sleeps[1] > 2200*time.Millisecond {
			t.Errorf("Second delay should be ~2s, got %v", sleeps[1])
		}
	}

	// Test should complete quickly since we're not actually sleeping
	if elapsed > 100*time.Millisecond {
		t.Errorf("Test took too long without actual sleeping: %v", elapsed)
	}
}

func TestNoRetryOnNon429Errors(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		// Return 500 Internal Server Error
		w.WriteHeader(http.StatusInternalServerError)
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

	_, err := client.CreateBookmark(bookmark)
	if err == nil {
		t.Fatal("Expected error for 500 status, got nil")
	}

	// Verify we only made 1 attempt (no retry for non-429 errors)
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attemptCount)
	}
}

// =============================================================================
// COMPREHENSIVE ERROR HANDLING TESTS
// =============================================================================

// Test helper to create a test client with a mock server
func createTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := &Client{
		baseURL:    server.URL + "/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    &MockSleeper{}, // Use mock sleeper by default for faster tests
	}
	return client, server
}

// Test helper to create a timeout client
func createTimeoutClient(timeout time.Duration) *Client {
	return &Client{
		baseURL: "https://api.karakeep.app/v1",
		httpClient: &http.Client{
			Timeout: timeout,
		},
		token:   "test-token",
		sleeper: &MockSleeper{},
	}
}

func TestCreateBookmarkHTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedError  string
		shouldRetry    bool
		expectedCalls  int
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedError:  "failed to create bookmark: 404 Not Found",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedError:  "failed to create bookmark: 401 Unauthorized",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			expectedError:  "failed to create bookmark: 400 Bad Request",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			expectedError:  "failed to create bookmark: 403 Forbidden",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedError:  "failed to create bookmark: 500 Internal Server Error",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "502 Bad Gateway",
			statusCode:     http.StatusBadGateway,
			expectedError:  "failed to create bookmark: 502 Bad Gateway",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedError:  "failed to create bookmark: 503 Service Unavailable",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "504 Gateway Timeout",
			statusCode:     http.StatusGatewayTimeout,
			expectedError:  "failed to create bookmark: 504 Gateway Timeout",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "429 Rate Limited",
			statusCode:     http.StatusTooManyRequests,
			expectedError:  "rate limited after 5 retries",
			shouldRetry:    true,
			expectedCalls:  6, // 1 initial + 5 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&callCount, 1)
				w.WriteHeader(tt.statusCode)
			})
			defer server.Close()

			bookmark := &Bookmark{
				URL:   "https://example.com",
				Title: "Test Bookmark",
			}

			_, err := client.CreateBookmark(bookmark)
			if err == nil {
				t.Fatalf("Expected error for status %d, got nil", tt.statusCode)
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}

			if int(callCount) != tt.expectedCalls {
				t.Errorf("Expected %d calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

func TestCreateListHTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedError  string
		shouldRetry    bool
		expectedCalls  int
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedError:  "failed to create list: 404 Not Found",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedError:  "failed to create list: 401 Unauthorized",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			expectedError:  "failed to create list: 400 Bad Request",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			expectedError:  "failed to create list: 403 Forbidden",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedError:  "failed to create list: 500 Internal Server Error",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "502 Bad Gateway",
			statusCode:     http.StatusBadGateway,
			expectedError:  "failed to create list: 502 Bad Gateway",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedError:  "failed to create list: 503 Service Unavailable",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "504 Gateway Timeout",
			statusCode:     http.StatusGatewayTimeout,
			expectedError:  "failed to create list: 504 Gateway Timeout",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "429 Rate Limited",
			statusCode:     http.StatusTooManyRequests,
			expectedError:  "rate limited after 5 retries",
			shouldRetry:    true,
			expectedCalls:  6, // 1 initial + 5 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&callCount, 1)
				w.WriteHeader(tt.statusCode)
			})
			defer server.Close()

			list := &List{Name: "Test List"}

			_, err := client.CreateList(list)
			if err == nil {
				t.Fatalf("Expected error for status %d, got nil", tt.statusCode)
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}

			if int(callCount) != tt.expectedCalls {
				t.Errorf("Expected %d calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

func TestAddBookmarkToListHTTPErrors(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectedError  string
		shouldRetry    bool
		expectedCalls  int
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			expectedError:  "failed to add bookmark to list: 404 Not Found",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedError:  "failed to add bookmark to list: 401 Unauthorized",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "400 Bad Request",
			statusCode:     http.StatusBadRequest,
			expectedError:  "failed to add bookmark to list: 400 Bad Request",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			expectedError:  "failed to add bookmark to list: 403 Forbidden",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedError:  "failed to add bookmark to list: 500 Internal Server Error",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "502 Bad Gateway",
			statusCode:     http.StatusBadGateway,
			expectedError:  "failed to add bookmark to list: 502 Bad Gateway",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedError:  "failed to add bookmark to list: 503 Service Unavailable",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "504 Gateway Timeout",
			statusCode:     http.StatusGatewayTimeout,
			expectedError:  "failed to add bookmark to list: 504 Gateway Timeout",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "429 Rate Limited",
			statusCode:     http.StatusTooManyRequests,
			expectedError:  "rate limited after 5 retries",
			shouldRetry:    true,
			expectedCalls:  6, // 1 initial + 5 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&callCount, 1)
				w.WriteHeader(tt.statusCode)
			})
			defer server.Close()

			err := client.AddBookmarkToList("bookmark-123", "list-456")
			if err == nil {
				t.Fatalf("Expected error for status %d, got nil", tt.statusCode)
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}

			if int(callCount) != tt.expectedCalls {
				t.Errorf("Expected %d calls, got %d", tt.expectedCalls, callCount)
			}
		})
	}
}

func TestNetworkTimeoutErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		setupFunc  func(client *Client) error
	}{
		{
			name:   "CreateBookmark Timeout",
			method: "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:   "CreateList Timeout",
			method: "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:   "AddBookmarkToList Timeout",
			method: "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a server that hangs
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second) // Longer than client timeout
			}))
			defer server.Close()

			// Create client with short timeout
			client := createTimeoutClient(100 * time.Millisecond)
			client.SetBaseURL(server.URL + "/v1")

			err := tt.setupFunc(client)
			if err == nil {
				t.Fatal("Expected timeout error, got nil")
			}

			// Check that it's a timeout error
			if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
				t.Errorf("Expected timeout error, got: %s", err.Error())
			}
		})
	}
}

func TestInvalidJSONResponses(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		expectedErr  string
		method       string
		setupFunc    func(client *Client) error
	}{
		{
			name:         "Invalid JSON CreateBookmark",
			response:     `{"id": "bookmark-123", "url": "https://example.com"`,
			expectedErr:  "unexpected EOF",
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:         "Invalid JSON CreateList",
			response:     `{"id": "list-123", "name": "Test List",}`,
			expectedErr:  "invalid character",
			method:       "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:         "Non-JSON Response CreateBookmark",
			response:     `<html><body>Error</body></html>`,
			expectedErr:  "invalid character",
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:         "Non-JSON Response CreateList",
			response:     `plain text response`,
			expectedErr:  "invalid character",
			method:       "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:         "Empty Response CreateBookmark",
			response:     ``,
			expectedErr:  "EOF",
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:         "Empty Response CreateList",
			response:     ``,
			expectedErr:  "EOF",
			method:       "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				// Return appropriate status for method
				if tt.method == "CreateBookmark" || tt.method == "CreateList" {
					w.WriteHeader(http.StatusCreated)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				fmt.Fprint(w, tt.response)
			})
			defer server.Close()

			err := tt.setupFunc(client)
			if err == nil {
				t.Fatal("Expected JSON parsing error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestConnectionErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupClient   func() *Client
		expectedError string
		method        string
		setupFunc     func(client *Client) error
	}{
		{
			name: "Connection Refused CreateBookmark",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://localhost:0", // Port 0 should refuse connections
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			expectedError: "connect:",
			method:        "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name: "Connection Refused CreateList",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://localhost:0", // Port 0 should refuse connections
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			expectedError: "connect:",
			method:        "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name: "Connection Refused AddBookmarkToList",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://localhost:0", // Port 0 should refuse connections
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			expectedError: "connect:",
			method:        "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
		{
			name: "Invalid Host CreateBookmark",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://invalid-host-that-does-not-exist.local",
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			expectedError: "deadline exceeded",
			method:        "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name: "Invalid Port CreateList",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://localhost:99999", // Invalid port
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			expectedError: "invalid port",
			method:        "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			
			err := tt.setupFunc(client)
			if err == nil {
				t.Fatal("Expected connection error, got nil")
			}
			
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestRequestCreationFailures(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *Client
		method      string
		setupFunc   func(client *Client) error
	}{
		{
			name: "Invalid URL CreateBookmark",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "://invalid-url",
					httpClient: &http.Client{},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			method: "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name: "Invalid URL CreateList",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "://invalid-url",
					httpClient: &http.Client{},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			method: "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name: "Invalid URL AddBookmarkToList",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "://invalid-url",
					httpClient: &http.Client{},
					token:      "test-token",
					sleeper:    &MockSleeper{},
				}
			},
			method: "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			
			err := tt.setupFunc(client)
			if err == nil {
				t.Fatal("Expected error for invalid URL, got nil")
			}
			
			// Should be a URL parsing error
			if !strings.Contains(err.Error(), "missing protocol scheme") {
				t.Errorf("Expected URL parsing error, got: %s", err.Error())
			}
		})
	}
}

func TestAuthenticationFailures(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
		method         string
		setupFunc      func(client *Client) error
	}{
		{
			name:           "401 Unauthorized CreateBookmark",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Unauthorized"}`,
			expectedError:  "failed to create bookmark: 401 Unauthorized",
			method:         "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:           "403 Forbidden CreateList",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"error": "Forbidden"}`,
			expectedError:  "failed to create list: 403 Forbidden",
			method:         "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:           "401 Invalid Token AddBookmarkToList",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid access token"}`,
			expectedError:  "failed to add bookmark to list: 401 Unauthorized",
			method:         "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				// Verify Authorization header is present
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer test-token" {
					t.Errorf("Expected Authorization header 'Bearer test-token', got '%s'", authHeader)
				}

				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.responseBody)
			})
			defer server.Close()

			err := tt.setupFunc(client)
			if err == nil {
				t.Fatal("Expected authentication error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestMalformedAPIResponses(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectError    bool
		errorContains  string
		method         string
		setupFunc      func(client *Client) error
	}{
		{
			name:           "Wrong Data Types CreateBookmark",
			response:       `{"id": 123, "url": "https://example.com", "title": "Test"}`,
			expectError:    true,
			errorContains:  "cannot unmarshal",
			method:         "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:           "Wrong Data Types CreateList",
			response:       `{"id": 123, "name": "Test List"}`,
			expectError:    true,
			errorContains:  "cannot unmarshal",
			method:         "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:           "Array Instead of Object CreateBookmark",
			response:       `[{"id": "bookmark-123", "url": "https://example.com", "title": "Test"}]`,
			expectError:    true,
			errorContains:  "cannot unmarshal",
			method:         "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:           "String Instead of Object CreateList",
			response:       `"not an object"`,
			expectError:    true,
			errorContains:  "cannot unmarshal",
			method:         "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:           "Missing Required Fields CreateBookmark",
			response:       `{"url": "https://example.com"}`,
			expectError:    false, // JSON decoding will use zero values
			method:         "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:           "Missing Required Fields CreateList",
			response:       `{"name": "Test List"}`,
			expectError:    false, // JSON decoding will use zero values
			method:         "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				// Return appropriate status for method
				if tt.method == "CreateBookmark" || tt.method == "CreateList" {
					w.WriteHeader(http.StatusCreated)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				fmt.Fprint(w, tt.response)
			})
			defer server.Close()

			err := tt.setupFunc(client)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestLargeResponseHandling(t *testing.T) {
	tests := []struct {
		name          string
		responseSize  int
		method        string
		setupFunc     func(client *Client) error
		generateFunc  func(size int) string
	}{
		{
			name:         "Large CreateBookmark Response",
			responseSize: 1024 * 1024, // 1MB
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
			generateFunc: func(size int) string {
				// Create a large description to fill the response
				largeDesc := strings.Repeat("a", size-100)
				return fmt.Sprintf(`{"id": "bookmark-123", "url": "https://example.com", "title": "Test", "description": "%s"}`, largeDesc)
			},
		},
		{
			name:         "Large CreateList Response",
			responseSize: 1024 * 1024, // 1MB
			method:       "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
			generateFunc: func(size int) string {
				// Create a large name to fill the response
				largeName := strings.Repeat("a", size-50)
				return fmt.Sprintf(`{"id": "list-123", "name": "%s"}`, largeName)
			},
		},
		{
			name:         "Very Large Response",
			responseSize: 10 * 1024 * 1024, // 10MB
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
			generateFunc: func(size int) string {
				// Create a very large description to fill the response
				largeDesc := strings.Repeat("a", size-100)
				return fmt.Sprintf(`{"id": "bookmark-123", "url": "https://example.com", "title": "Test", "description": "%s"}`, largeDesc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				// Return appropriate status for method
				if tt.method == "CreateBookmark" || tt.method == "CreateList" {
					w.WriteHeader(http.StatusCreated)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				
				// Generate large response
				response := tt.generateFunc(tt.responseSize)
				fmt.Fprint(w, response)
			})
			defer server.Close()

			start := time.Now()
			
			err := tt.setupFunc(client)
			
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Large responses should still complete in reasonable time
			if elapsed > 30*time.Second {
				t.Errorf("Large response took too long: %v", elapsed)
			}
		})
	}
}

func TestConcurrentRequests(t *testing.T) {
	tests := []struct {
		name      string
		method    string
		setupFunc func(client *Client) error
	}{
		{
			name:   "Concurrent CreateBookmark",
			method: "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:   "Concurrent CreateList",
			method: "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:   "Concurrent AddBookmarkToList",
			method: "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requestCount, 1)
				// Add small delay to increase chance of race conditions
				time.Sleep(10 * time.Millisecond)
				
				// Return appropriate status and response for method
				if tt.method == "CreateBookmark" {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"id": "bookmark-123", "url": "https://example.com", "title": "Test"}`)
				} else if tt.method == "CreateList" {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"id": "list-123", "name": "Test List"}`)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			})
			defer server.Close()

			const numGoroutines = 10
			errChan := make(chan error, numGoroutines)
			
			// Start multiple goroutines making requests
			for i := 0; i < numGoroutines; i++ {
				go func() {
					err := tt.setupFunc(client)
					errChan <- err
				}()
			}
			
			// Wait for all goroutines to complete
			for i := 0; i < numGoroutines; i++ {
				err := <-errChan
				if err != nil {
					t.Errorf("Concurrent request failed: %v", err)
				}
			}
			
			// Verify all requests were made
			if requestCount != numGoroutines {
				t.Errorf("Expected %d requests, got %d", numGoroutines, requestCount)
			}
		})
	}
}

func TestTruncatedResponseHandling(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		closeEarly   bool
		expectedErr  string
		method       string
		setupFunc    func(client *Client) error
	}{
		{
			name:         "Truncated JSON CreateBookmark",
			response:     `{"id": "bookmark-123", "url": "https://example.com", "title": "Test"`,
			closeEarly:   false,
			expectedErr:  "unexpected EOF",
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:         "Truncated JSON CreateList",
			response:     `{"id": "list-123", "name": "Test List"`,
			closeEarly:   false,
			expectedErr:  "unexpected EOF",
			method:       "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:         "Connection Closed Early CreateBookmark",
			response:     `{"id": "bookmark-123", "url": "https://example.com", "title": "Test"}`,
			closeEarly:   true,
			expectedErr:  "EOF",
			method:       "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:         "Connection Closed Early CreateList",
			response:     `{"id": "list-123", "name": "Test List"}`,
			closeEarly:   true,
			expectedErr:  "EOF",
			method:       "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				// Return appropriate status for method
				if tt.method == "CreateBookmark" || tt.method == "CreateList" {
					w.WriteHeader(http.StatusCreated)
				} else {
					w.WriteHeader(http.StatusOK)
				}
				
				if tt.closeEarly {
					// Write partial response then close connection
					w.Write([]byte(tt.response[:len(tt.response)/2]))
					if hijacker, ok := w.(http.Hijacker); ok {
						conn, _, _ := hijacker.Hijack()
						conn.Close()
					}
				} else {
					fmt.Fprint(w, tt.response)
				}
			})
			defer server.Close()

			err := tt.setupFunc(client)

			if err == nil {
				t.Fatal("Expected truncated response error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestConcurrentErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedError string
		method        string
		setupFunc     func(client *Client) error
	}{
		{
			name:          "Concurrent 404 Errors CreateBookmark",
			statusCode:    http.StatusNotFound,
			expectedError: "404 Not Found",
			method:        "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:          "Concurrent 500 Errors CreateList",
			statusCode:    http.StatusInternalServerError,
			expectedError: "500 Internal Server Error",
			method:        "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:          "Concurrent Rate Limits AddBookmarkToList",
			statusCode:    http.StatusTooManyRequests,
			expectedError: "rate limited after 5 retries",
			method:        "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requestCount, 1)
				w.WriteHeader(tt.statusCode)
			})
			defer server.Close()

			const numGoroutines = 5
			errChan := make(chan error, numGoroutines)
			
			// Start multiple goroutines making requests
			for i := 0; i < numGoroutines; i++ {
				go func() {
					err := tt.setupFunc(client)
					errChan <- err
				}()
			}
			
			// Wait for all goroutines to complete
			errorCount := 0
			for i := 0; i < numGoroutines; i++ {
				err := <-errChan
				if err != nil {
					errorCount++
					if !strings.Contains(err.Error(), tt.expectedError) {
						t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
					}
				}
			}
			
			// All requests should have failed
			if errorCount != numGoroutines {
				t.Errorf("Expected %d errors, got %d", numGoroutines, errorCount)
			}
		})
	}
}

func TestRequestHeaderValidation(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectError   bool
		expectedAuth  string
		method        string
		setupFunc     func(client *Client) error
	}{
		{
			name:          "Valid Token CreateBookmark",
			token:         "valid-token-123",
			expectError:   false,
			expectedAuth:  "Bearer valid-token-123",
			method:        "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:          "Valid Token CreateList",
			token:         "valid-token-456",
			expectError:   false,
			expectedAuth:  "Bearer valid-token-456",
			method:        "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:          "Valid Token AddBookmarkToList",
			token:         "valid-token-789",
			expectError:   false,
			expectedAuth:  "Bearer valid-token-789",
			method:        "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
		{
			name:          "Empty Token CreateBookmark",
			token:         "",
			expectError:   false,
			expectedAuth:  "Bearer",
			method:        "CreateBookmark",
			setupFunc: func(client *Client) error {
				bookmark := &Bookmark{
					URL:   "https://example.com",
					Title: "Test Bookmark",
				}
				_, err := client.CreateBookmark(bookmark)
				return err
			},
		},
		{
			name:          "Special Characters Token CreateList",
			token:         "token-with-special-chars!@#$%^&*()",
			expectError:   false,
			expectedAuth:  "Bearer token-with-special-chars!@#$%^&*()",
			method:        "CreateList",
			setupFunc: func(client *Client) error {
				list := &List{Name: "Test List"}
				_, err := client.CreateList(list)
				return err
			},
		},
		{
			name:          "Unicode Token AddBookmarkToList",
			token:         "token-with-unicode-∑∆∫",
			expectError:   false,
			expectedAuth:  "Bearer token-with-unicode-∑∆∫",
			method:        "AddBookmarkToList",
			setupFunc: func(client *Client) error {
				return client.AddBookmarkToList("bookmark-123", "list-456")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedAuth string
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				receivedAuth = r.Header.Get("Authorization")
				
				// Return appropriate status and response for method
				if tt.method == "CreateBookmark" {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"id": "bookmark-123", "url": "https://example.com", "title": "Test"}`)
				} else if tt.method == "CreateList" {
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"id": "list-123", "name": "Test List"}`)
				} else {
					w.WriteHeader(http.StatusOK)
				}
			})
			defer server.Close()

			// Set the token
			client.token = tt.token

			err := tt.setupFunc(client)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if receivedAuth != tt.expectedAuth {
				t.Errorf("Expected Authorization header '%s', got '%s'", tt.expectedAuth, receivedAuth)
			}
		})
	}
}
