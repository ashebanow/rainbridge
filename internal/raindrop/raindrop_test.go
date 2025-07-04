//go:build !integration

package raindrop

import (
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

func TestRateLimitingWithRetry(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := atomic.AddInt32(&attemptCount, 1)
		
		// Return 429 for the first 2 attempts, then success
		if attempt <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"items": [{"_id": 1, "title": "Success after retry"}]}`)
	}))
	defer server.Close()

	mockSleeper := &MockSleeper{}
	client := &Client{
		baseURL:    server.URL + "/rest/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    mockSleeper,
	}

	start := time.Now()
	collections, err := client.GetCollections()
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("GetCollections failed: %v", err)
	}

	if len(collections) != 1 {
		t.Fatalf("Expected 1 collection, got %d", len(collections))
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

func TestRateLimitingMaxRetries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return 429
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	mockSleeper := &MockSleeper{}
	client := &Client{
		baseURL:    server.URL + "/rest/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    mockSleeper,
	}

	start := time.Now()
	_, err := client.GetCollections()
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

func TestNoRetryOnNon429Errors(t *testing.T) {
	var attemptCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attemptCount, 1)
		// Return 500 Internal Server Error
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &Client{
		baseURL:    server.URL + "/rest/v1",
		httpClient: server.Client(),
		token:      "test-token",
	}

	_, err := client.GetCollections()
	if err == nil {
		t.Fatal("Expected error for 500 status, got nil")
	}

	// Verify we only made 1 attempt (no retry for non-429 errors)
	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt (no retry), got %d", attemptCount)
	}
}

// Test helper to create a test client with a mock server
func createTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := &Client{
		baseURL:    server.URL + "/rest/v1",
		httpClient: server.Client(),
		token:      "test-token",
		sleeper:    &MockSleeper{}, // Use mock sleeper by default for faster tests
	}
	return client, server
}

// Test helper to create a timeout client
func createTimeoutClient(timeout time.Duration) *Client {
	return &Client{
		baseURL: "https://api.raindrop.io/rest/v1",
		httpClient: &http.Client{
			Timeout: timeout,
		},
		token: "test-token",
	}
}

func TestHTTPErrorHandling(t *testing.T) {
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
			expectedError:  "failed to get collections: 404 Not Found",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedError:  "failed to get collections: 401 Unauthorized",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedError:  "failed to get collections: 500 Internal Server Error",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "502 Bad Gateway",
			statusCode:     http.StatusBadGateway,
			expectedError:  "failed to get collections: 502 Bad Gateway",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "503 Service Unavailable",
			statusCode:     http.StatusServiceUnavailable,
			expectedError:  "failed to get collections: 503 Service Unavailable",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "504 Gateway Timeout",
			statusCode:     http.StatusGatewayTimeout,
			expectedError:  "failed to get collections: 504 Gateway Timeout",
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

			_, err := client.GetCollections()
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

func TestRaindropsHTTPErrorHandling(t *testing.T) {
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
			expectedError:  "failed to get raindrops: 404 Not Found",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			expectedError:  "failed to get raindrops: 401 Unauthorized",
			shouldRetry:    false,
			expectedCalls:  1,
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			expectedError:  "failed to get raindrops: 500 Internal Server Error",
			shouldRetry:    false,
			expectedCalls:  1,
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

			_, err := client.GetRaindrops()
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

func TestNetworkTimeout(t *testing.T) {
	// Create a server that hangs
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Longer than client timeout
	}))
	defer server.Close()

	// Create client with short timeout
	client := createTimeoutClient(100 * time.Millisecond)
	client.SetBaseURL(server.URL + "/rest/v1")

	_, err := client.GetCollections()
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// Check that it's a timeout error
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected timeout error, got: %s", err.Error())
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		expectedErr  string
		endpoint     string
	}{
		{
			name:         "Invalid JSON Collections",
			response:     `{"items": [{"_id": 123, "title": "Test"}`,
			expectedErr:  "unexpected EOF",
			endpoint:     "collections",
		},
		{
			name:         "Invalid JSON Raindrops",
			response:     `{"items": [{"_id": 1, "title": "Test",}]}`,
			expectedErr:  "invalid character",
			endpoint:     "raindrops",
		},
		{
			name:         "Non-JSON Response Collections",
			response:     `<html><body>Error</body></html>`,
			expectedErr:  "invalid character",
			endpoint:     "collections",
		},
		{
			name:         "Non-JSON Response Raindrops",
			response:     `plain text response`,
			expectedErr:  "invalid character",
			endpoint:     "raindrops",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, tt.response)
			})
			defer server.Close()

			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}

			if err == nil {
				t.Fatal("Expected JSON parsing error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestEmptyAndNilResponses(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		endpoint     string
		expectEmpty  bool
		expectError  bool
	}{
		{
			name:         "Empty Items Collections",
			response:     `{"items": []}`,
			endpoint:     "collections",
			expectEmpty:  true,
			expectError:  false,
		},
		{
			name:         "Empty Items Raindrops",
			response:     `{"items": []}`,
			endpoint:     "raindrops",
			expectEmpty:  true,
			expectError:  false,
		},
		{
			name:         "Missing Items Field Collections",
			response:     `{}`,
			endpoint:     "collections",
			expectEmpty:  true,
			expectError:  false,
		},
		{
			name:         "Missing Items Field Raindrops",
			response:     `{}`,
			endpoint:     "raindrops",
			expectEmpty:  true,
			expectError:  false,
		},
		{
			name:         "Null Items Collections",
			response:     `{"items": null}`,
			endpoint:     "collections",
			expectEmpty:  true,
			expectError:  false,
		},
		{
			name:         "Null Items Raindrops",
			response:     `{"items": null}`,
			endpoint:     "raindrops",
			expectEmpty:  true,
			expectError:  false,
		},
		{
			name:         "Empty Response Collections",
			response:     ``,
			endpoint:     "collections",
			expectEmpty:  false,
			expectError:  true,
		},
		{
			name:         "Empty Response Raindrops",
			response:     ``,
			endpoint:     "raindrops",
			expectEmpty:  false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, tt.response)
			})
			defer server.Close()

			if tt.endpoint == "collections" {
				collections, err := client.GetCollections()
				if tt.expectError {
					if err == nil {
						t.Fatal("Expected error, got nil")
					}
					return
				}
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tt.expectEmpty && len(collections) != 0 {
					t.Errorf("Expected empty collections, got %d items", len(collections))
				}
			} else {
				raindrops, err := client.GetRaindrops()
				if tt.expectError {
					if err == nil {
						t.Fatal("Expected error, got nil")
					}
					return
				}
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tt.expectEmpty && len(raindrops) != 0 {
					t.Errorf("Expected empty raindrops, got %d items", len(raindrops))
				}
			}
		})
	}
}

func TestPaginationEdgeCases(t *testing.T) {
	tests := []struct {
		name              string
		responses         []string
		expectedCount     int
		expectedCallCount int
	}{
		{
			name: "Zero Items First Page",
			responses: []string{
				`{"items": []}`,
			},
			expectedCount:     0,
			expectedCallCount: 1,
		},
		{
			name: "Multiple Empty Pages",
			responses: []string{
				`{"items": [{"_id": 1, "title": "Item 1"}]}`,
				`{"items": []}`,
			},
			expectedCount:     1,
			expectedCallCount: 2,
		},
		{
			name: "Single Item Multiple Pages",
			responses: []string{
				`{"items": [{"_id": 1, "title": "Item 1"}]}`,
				`{"items": [{"_id": 2, "title": "Item 2"}]}`,
				`{"items": []}`,
			},
			expectedCount:     2,
			expectedCallCount: 3,
		},
		{
			name: "Large Number of Pages",
			responses: func() []string {
				responses := make([]string, 10)
				for i := 0; i < 9; i++ {
					responses[i] = fmt.Sprintf(`{"items": [{"_id": %d, "title": "Item %d"}]}`, i+1, i+1)
				}
				responses[9] = `{"items": []}`
				return responses
			}(),
			expectedCount:     9,
			expectedCallCount: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				callIdx := atomic.AddInt32(&callCount, 1) - 1
				if int(callIdx) < len(tt.responses) {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, tt.responses[callIdx])
				} else {
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"items": []}`)
				}
			})
			defer server.Close()

			raindrops, err := client.GetRaindrops()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(raindrops) != tt.expectedCount {
				t.Errorf("Expected %d raindrops, got %d", tt.expectedCount, len(raindrops))
			}

			if int(callCount) != tt.expectedCallCount {
				t.Errorf("Expected %d API calls, got %d", tt.expectedCallCount, callCount)
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
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Unauthorized"}`,
			expectedError:  "failed to get collections: 401 Unauthorized",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   `{"error": "Forbidden"}`,
			expectedError:  "failed to get collections: 403 Forbidden",
		},
		{
			name:           "401 Invalid Token",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid access token"}`,
			expectedError:  "failed to get collections: 401 Unauthorized",
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

			_, err := client.GetCollections()
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
		name         string
		response     string
		endpoint     string
		expectError  bool
		errorContains string
	}{
		{
			name:         "Missing Required Fields Collections",
			response:     `{"items": [{"title": "No ID"}]}`,
			endpoint:     "collections",
			expectError:  false, // JSON decoding will use zero values
		},
		{
			name:         "Missing Required Fields Raindrops",
			response:     `{"items": [{"title": "No ID"}]}`,
			endpoint:     "raindrops",
			expectError:  false, // JSON decoding will use zero values
		},
		{
			name:         "Wrong Data Types Collections",
			response:     `{"items": [{"_id": "not-a-number", "title": "Test"}]}`,
			endpoint:     "collections",
			expectError:  true,
			errorContains: "cannot unmarshal",
		},
		{
			name:         "Wrong Data Types Raindrops",
			response:     `{"items": [{"_id": "not-a-number", "title": "Test"}]}`,
			endpoint:     "raindrops",
			expectError:  true,
			errorContains: "cannot unmarshal",
		},
		{
			name:         "Nested Object Instead of Array",
			response:     `{"items": {"_id": 1, "title": "Not an array"}}`,
			endpoint:     "collections",
			expectError:  true,
			errorContains: "cannot unmarshal",
		},
		{
			name:         "String Instead of Object",
			response:     `{"items": "not an array"}`,
			endpoint:     "raindrops",
			expectError:  true,
			errorContains: "cannot unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				count := atomic.AddInt32(&requestCount, 1)
				w.WriteHeader(http.StatusOK)
				
				// For raindrops, we need to handle pagination properly
				if tt.endpoint == "raindrops" {
					page := r.URL.Query().Get("page")
					if page == "" || page == "0" {
						// First page - return the test response
						fmt.Fprint(w, tt.response)
					} else {
						// Subsequent pages - return empty to stop pagination
						fmt.Fprint(w, `{"items": []}`)
					}
				} else {
					fmt.Fprint(w, tt.response)
				}
				
				// Prevent infinite loops by limiting requests
				if count > 10 {
					t.Errorf("Too many requests (%d), possible infinite loop", count)
				}
			})
			defer server.Close()

			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}

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
		name           string
		itemCount      int
		endpoint       string
		expectedCount  int
	}{
		{
			name:           "Large Collections Response",
			itemCount:      1000,
			endpoint:       "collections",
			expectedCount:  1000,
		},
		{
			name:           "Large Raindrops Response",
			itemCount:      1000,
			endpoint:       "raindrops",
			expectedCount:  250, // Limited by pagination (5 pages * 50 per page)
		},
		{
			name:           "Very Large Single Page",
			itemCount:      5000,
			endpoint:       "collections",
			expectedCount:  5000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				
				// Generate large response
				var items []string
				if tt.endpoint == "collections" {
					for i := 0; i < tt.itemCount; i++ {
						items = append(items, fmt.Sprintf(`{"_id": %d, "title": "Collection %d"}`, i+1, i+1))
					}
					response := fmt.Sprintf(`{"items": [%s]}`, strings.Join(items, ","))
					fmt.Fprint(w, response)
				} else {
					// For raindrops, handle pagination properly to avoid infinite loops
					page := r.URL.Query().Get("page")
					pageNum := 0
					if page != "" {
						fmt.Sscanf(page, "%d", &pageNum)
					}
					
					atomic.AddInt32(&requestCount, 1)
					
					// Limit to reasonable number of pages to prevent infinite loops
					const maxPages = 5
					if pageNum >= maxPages {
						fmt.Fprint(w, `{"items": []}`)
						return
					}
					
					// Generate items for this page (50 per page)
					perPage := 50
					startIdx := pageNum * perPage
					endIdx := startIdx + perPage
					if endIdx > tt.itemCount {
						endIdx = tt.itemCount
					}
					
					if startIdx >= tt.itemCount {
						fmt.Fprint(w, `{"items": []}`)
						return
					}
					
					for i := startIdx; i < endIdx; i++ {
						items = append(items, fmt.Sprintf(`{"_id": %d, "title": "Raindrop %d", "link": "https://example.com/%d", "excerpt": "Description %d", "tags": ["tag%d"]}`, i+1, i+1, i+1, i+1, i+1))
					}
					
					response := fmt.Sprintf(`{"items": [%s]}`, strings.Join(items, ","))
					fmt.Fprint(w, response)
				}
			})
			defer server.Close()

			start := time.Now()
			
			if tt.endpoint == "collections" {
				collections, err := client.GetCollections()
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if len(collections) != tt.expectedCount {
					t.Errorf("Expected %d collections, got %d", tt.expectedCount, len(collections))
				}
			} else {
				raindrops, err := client.GetRaindrops()
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				// For paginated raindrops, we expect fewer items due to pagination limits
				expectedForRaindrops := tt.expectedCount
				if tt.itemCount > 250 { // 5 pages * 50 per page
					expectedForRaindrops = 250
				}
				if len(raindrops) != expectedForRaindrops {
					t.Errorf("Expected %d raindrops, got %d", expectedForRaindrops, len(raindrops))
				}
			}
			
			elapsed := time.Since(start)
			// Large responses should still complete in reasonable time (less than 5 seconds)
			if elapsed > 5*time.Second {
				t.Errorf("Large response took too long: %v", elapsed)
			}
		})
	}
}

func TestConcurrentRequests(t *testing.T) {
	// Test that multiple concurrent requests don't interfere with each other
	var requestCount int32
	client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Add small delay to increase chance of race conditions
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"items": [{"_id": 1, "title": "Test"}]}`)
	})
	defer server.Close()

	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)
	
	// Start multiple goroutines making requests
	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := client.GetCollections()
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
}

// =============================================================================
// COMPREHENSIVE ERROR HANDLING TESTS
// =============================================================================
// The following tests provide comprehensive coverage of error handling scenarios
// for the raindrop client, including network errors, malformed responses,
// authentication failures, and edge cases.

func TestRequestCreationFailure(t *testing.T) {
	// Test with invalid URL to trigger request creation failure
	client := &Client{
		baseURL:    "://invalid-url",
		httpClient: &http.Client{},
		token:      "test-token",
	}
	
	_, err := client.GetCollections()
	if err == nil {
		t.Fatal("Expected error for invalid URL, got nil")
	}
	
	// Should be a URL parsing error
	if !strings.Contains(err.Error(), "missing protocol scheme") {
		t.Errorf("Expected URL parsing error, got: %s", err.Error())
	}
}

func TestConnectionErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupClient   func() *Client
		expectedError string
	}{
		{
			name: "Connection Refused",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://localhost:0", // Port 0 should refuse connections
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
				}
			},
			expectedError: "connect:", // More general connection error
		},
		{
			name: "Invalid Host",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://invalid-host-that-does-not-exist.local",
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
				}
			},
			expectedError: "deadline exceeded", // DNS resolution or connection timeout
		},
		{
			name: "Invalid Port",
			setupClient: func() *Client {
				return &Client{
					baseURL:    "http://localhost:99999", // Invalid port
					httpClient: &http.Client{Timeout: 1 * time.Second},
					token:      "test-token",
				}
			},
			expectedError: "invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			
			_, err := client.GetCollections()
			if err == nil {
				t.Fatal("Expected connection error, got nil")
			}
			
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestTruncatedResponseHandling(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		endpoint     string
		closeEarly   bool
		expectedErr  string
	}{
		{
			name:         "Truncated JSON Collections",
			response:     `{"items": [{"_id": 123, "title": "Test"`,
			endpoint:     "collections",
			closeEarly:   false,
			expectedErr:  "unexpected EOF",
		},
		{
			name:         "Truncated JSON Raindrops",
			response:     `{"items": [{"_id": 1, "title": "Test", "link": "https://example.com"`,
			endpoint:     "raindrops",
			closeEarly:   false,
			expectedErr:  "unexpected EOF",
		},
		{
			name:         "Connection Closed Early Collections",
			response:     `{"items": [{"_id": 123, "title": "Test"}]}`,
			endpoint:     "collections",
			closeEarly:   true,
			expectedErr:  "EOF",
		},
		{
			name:         "Connection Closed Early Raindrops",
			response:     `{"items": [{"_id": 1, "title": "Test", "link": "https://example.com"}]}`,
			endpoint:     "raindrops",
			closeEarly:   true,
			expectedErr:  "EOF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
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

			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}

			if err == nil {
				t.Fatal("Expected truncated response error, got nil")
			}

			if !strings.Contains(err.Error(), tt.expectedErr) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestRequestHeaderValidation(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		endpoint      string
		expectError   bool
		expectedAuth  string
	}{
		{
			name:          "Valid Token Collections",
			token:         "valid-token-123",
			endpoint:      "collections",
			expectError:   false,
			expectedAuth:  "Bearer valid-token-123",
		},
		{
			name:          "Valid Token Raindrops",
			token:         "valid-token-456",
			endpoint:      "raindrops",
			expectError:   false,
			expectedAuth:  "Bearer valid-token-456",
		},
		{
			name:          "Empty Token Collections",
			token:         "",
			endpoint:      "collections",
			expectError:   false,
			expectedAuth:  "Bearer",
		},
		{
			name:          "Special Characters Token",
			token:         "token-with-special-chars!@#$%^&*()",
			endpoint:      "collections",
			expectError:   false,
			expectedAuth:  "Bearer token-with-special-chars!@#$%^&*()",
		},
		{
			name:          "Unicode Token",
			token:         "token-with-unicode-∑∆∫",
			endpoint:      "collections",
			expectError:   false,
			expectedAuth:  "Bearer token-with-unicode-∑∆∫",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedAuth string
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				receivedAuth = r.Header.Get("Authorization")
				w.WriteHeader(http.StatusOK)
				if tt.endpoint == "collections" {
					fmt.Fprint(w, `{"items": [{"_id": 1, "title": "Test"}]}`)
				} else {
					// For raindrops, handle pagination properly
					page := r.URL.Query().Get("page")
					if page == "" || page == "0" {
						fmt.Fprint(w, `{"items": [{"_id": 1, "title": "Test", "link": "https://example.com"}]}`)
					} else {
						// Return empty for subsequent pages
						fmt.Fprint(w, `{"items": []}`)
					}
				}
			})
			defer server.Close()

			// Set the token
			client.token = tt.token

			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}

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

func TestRateLimitWithRetryAfterHeader(t *testing.T) {
	// Note: The current implementation doesn't actually handle Retry-After headers
	// This test verifies that the exponential backoff still works even when 
	// Retry-After headers are present (they are ignored)
	tests := []struct {
		name             string
		retryAfterHeader string
		expectError      bool
	}{
		{
			name:             "Retry-After Seconds",
			retryAfterHeader: "2",
			expectError:      false,
		},
		{
			name:             "Retry-After HTTP Date", 
			retryAfterHeader: time.Now().Add(2*time.Second).Format(time.RFC1123),
			expectError:      false,
		},
		{
			name:             "Invalid Retry-After",
			retryAfterHeader: "invalid",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var attemptCount int32
			mockSleeper := &MockSleeper{}
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				attempt := atomic.AddInt32(&attemptCount, 1)
				
				if attempt == 1 {
					// First attempt: return 429 with Retry-After header
					w.Header().Set("Retry-After", tt.retryAfterHeader)
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				
				// Second attempt: return success
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{"items": [{"_id": 1, "title": "Success"}]}`)
			})
			defer server.Close()

			// Replace the client's sleeper with our controlled mock
			client.sleeper = mockSleeper

			start := time.Now()
			_, err := client.GetCollections()
			elapsed := time.Since(start)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify we made 2 attempts
			if attemptCount != 2 {
				t.Errorf("Expected 2 attempts, got %d", attemptCount)
			}

			// Verify we attempted to sleep (exponential backoff, ignoring Retry-After)
			sleeps := mockSleeper.GetSleeps()
			if len(sleeps) != 1 {
				t.Errorf("Expected 1 sleep call, got %d", len(sleeps))
			}

			// Verify exponential backoff pattern (approximately 1s with jitter)
			if len(sleeps) > 0 {
				if sleeps[0] < 900*time.Millisecond || sleeps[0] > 1100*time.Millisecond {
					t.Errorf("Delay should be ~1s, got %v", sleeps[0])
				}
			}

			// Test should complete quickly since we're not actually sleeping
			if elapsed > 100*time.Millisecond {
				t.Errorf("Test took too long without actual sleeping: %v", elapsed)
			}
		})
	}
}

func TestMaxResponseSizeHandling(t *testing.T) {
	tests := []struct {
		name          string
		responseSize  int
		endpoint      string
		expectError   bool
	}{
		{
			name:          "Normal Size Response Collections",
			responseSize:  1024,
			endpoint:      "collections",
			expectError:   false,
		},
		{
			name:          "Large Response Collections",
			responseSize:  10 * 1024 * 1024, // 10MB
			endpoint:      "collections",
			expectError:   false,
		},
		{
			name:          "Large Response Raindrops",
			responseSize:  1024 * 1024, // 1MB
			endpoint:      "raindrops",
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				
				// Generate response of specified size
				if tt.endpoint == "collections" {
					itemsNeeded := (tt.responseSize - 100) / 50 // Approximate items needed
					if itemsNeeded < 1 {
						itemsNeeded = 1
					}
					
					w.Write([]byte(`{"items": [`))
					for i := 0; i < itemsNeeded; i++ {
						if i > 0 {
							w.Write([]byte(`,`))
						}
						item := fmt.Sprintf(`{"_id": %d, "title": "Collection %d"}`, i+1, i+1)
						w.Write([]byte(item))
					}
					w.Write([]byte(`]}`))
				} else {
					// For raindrops, we need to handle pagination
					page := r.URL.Query().Get("page")
					if page == "" || page == "0" {
						itemsNeeded := (tt.responseSize - 100) / 100 // Approximate items needed
						if itemsNeeded < 1 {
							itemsNeeded = 1
						}
						
						w.Write([]byte(`{"items": [`))
						for i := 0; i < itemsNeeded; i++ {
							if i > 0 {
								w.Write([]byte(`,`))
							}
							item := fmt.Sprintf(`{"_id": %d, "title": "Raindrop %d", "link": "https://example.com/%d", "excerpt": "Description %d", "tags": ["tag%d"]}`, i+1, i+1, i+1, i+1, i+1)
							w.Write([]byte(item))
						}
						w.Write([]byte(`]}`))
					} else {
						// Subsequent pages: return empty
						w.Write([]byte(`{"items": []}`))
					}
				}
			})
			defer server.Close()

			start := time.Now()
			
			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}
			
			elapsed := time.Since(start)

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

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

func TestConcurrentErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedError string
	}{
		{
			name:          "Concurrent 404 Errors",
			statusCode:    http.StatusNotFound,
			expectedError: "404 Not Found",
		},
		{
			name:          "Concurrent 500 Errors",
			statusCode:    http.StatusInternalServerError,
			expectedError: "500 Internal Server Error",
		},
		{
			name:          "Concurrent Rate Limits",
			statusCode:    http.StatusTooManyRequests,
			expectedError: "rate limited after 5 retries",
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
					_, err := client.GetCollections()
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

func TestPartialJSONDecoding(t *testing.T) {
	tests := []struct {
		name         string
		response     string
		endpoint     string
		expectError  bool
		errorContains string
	}{
		{
			name:         "Valid JSON with Extra Data Collections",
			response:     `{"items": [{"_id": 1, "title": "Test"}]}{"extra": "data"}`,
			endpoint:     "collections",
			expectError:  false, // JSON decoder stops at first valid object
		},
		{
			name:         "Valid JSON with Extra Data Raindrops",
			response:     `{"items": [{"_id": 1, "title": "Test", "link": "https://example.com"}]}{"extra": "data"}`,
			endpoint:     "raindrops",
			expectError:  false, // JSON decoder stops at first valid object
		},
		{
			name:         "Multiple JSON Objects Collections",
			response:     `{"items": [{"_id": 1, "title": "Test1"}]}{"items": [{"_id": 2, "title": "Test2"}]}`,
			endpoint:     "collections",
			expectError:  false, // Decoder reads first object only
		},
		{
			name:         "Nested JSON Structure Collections",
			response:     `{"items": [{"_id": 1, "title": "Test", "nested": {"key": "value"}}]}`,
			endpoint:     "collections",
			expectError:  false, // Should handle nested structures
		},
		{
			name:         "Array Instead of Object Collections",
			response:     `[{"_id": 1, "title": "Test"}]`,
			endpoint:     "collections",
			expectError:  true,
			errorContains: "cannot unmarshal",
		},
		{
			name:         "Number Instead of Object Raindrops",
			response:     `123`,
			endpoint:     "raindrops",
			expectError:  true,
			errorContains: "cannot unmarshal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				if tt.endpoint == "raindrops" {
					// Handle pagination for raindrops
					page := r.URL.Query().Get("page")
					if page == "" || page == "0" {
						fmt.Fprint(w, tt.response)
					} else {
						// Return empty for subsequent pages
						fmt.Fprint(w, `{"items": []}`)
					}
				} else {
					fmt.Fprint(w, tt.response)
				}
			})
			defer server.Close()

			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}

			if tt.expectError {
				if err == nil {
					t.Fatal("Expected JSON decoding error, got nil")
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

func TestRaindropsPaginationErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		pageResponses map[int]struct {
			statusCode int
			response   string
		}
		expectError   bool
		errorContains string
	}{
		{
			name: "Error on Second Page",
			pageResponses: map[int]struct {
				statusCode int
				response   string
			}{
				0: {http.StatusOK, `{"items": [{"_id": 1, "title": "Item 1", "link": "https://example.com/1"}]}`},
				1: {http.StatusInternalServerError, `{"error": "Server error"}`},
			},
			expectError:   true,
			errorContains: "500 Internal Server Error",
		},
		{
			name: "Invalid JSON on Third Page",
			pageResponses: map[int]struct {
				statusCode int
				response   string
			}{
				0: {http.StatusOK, `{"items": [{"_id": 1, "title": "Item 1", "link": "https://example.com/1"}]}`},
				1: {http.StatusOK, `{"items": [{"_id": 2, "title": "Item 2", "link": "https://example.com/2"}]}`},
				2: {http.StatusOK, `{"items": [{"_id": 3, "title": "Item 3"`},
			},
			expectError:   true,
			errorContains: "unexpected EOF",
		},
		{
			name: "Rate Limited During Pagination",
			pageResponses: map[int]struct {
				statusCode int
				response   string
			}{
				0: {http.StatusOK, `{"items": [{"_id": 1, "title": "Item 1", "link": "https://example.com/1"}]}`},
				1: {http.StatusTooManyRequests, ``},
			},
			expectError:   true,
			errorContains: "rate limited after 5 retries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pageRequests int32
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("page")
				pageNum := 0
				if page != "" {
					fmt.Sscanf(page, "%d", &pageNum)
				}
				
				atomic.AddInt32(&pageRequests, 1)
				
				if response, exists := tt.pageResponses[pageNum]; exists {
					w.WriteHeader(response.statusCode)
					fmt.Fprint(w, response.response)
				} else {
					// Default: empty page to stop pagination
					w.WriteHeader(http.StatusOK)
					fmt.Fprint(w, `{"items": []}`)
				}
			})
			defer server.Close()

			_, err := client.GetRaindrops()

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

func TestErrorMessageFormatting(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		statusText     string
		endpoint       string
		expectedFormat string
	}{
		{
			name:           "Collections Standard Error",
			statusCode:     http.StatusBadRequest,
			statusText:     "Bad Request",
			endpoint:       "collections",
			expectedFormat: "failed to get collections: 400 Bad Request",
		},
		{
			name:           "Raindrops Standard Error",
			statusCode:     http.StatusNotFound,
			statusText:     "Not Found",
			endpoint:       "raindrops",
			expectedFormat: "failed to get raindrops: 404 Not Found",
		},
		{
			name:           "Collections Custom Status",
			statusCode:     http.StatusTeapot,
			statusText:     "I'm a teapot",
			endpoint:       "collections",
			expectedFormat: "failed to get collections: 418 I'm a teapot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, server := createTestClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, `{"error": "Test error"}`)
			})
			defer server.Close()

			var err error
			if tt.endpoint == "collections" {
				_, err = client.GetCollections()
			} else {
				_, err = client.GetRaindrops()
			}

			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if err.Error() != tt.expectedFormat {
				t.Errorf("Expected error message '%s', got '%s'", tt.expectedFormat, err.Error())
			}
		})
	}
}
