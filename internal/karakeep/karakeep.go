package karakeep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// Sleeper interface for dependency injection of sleep functionality
type Sleeper interface {
	Sleep(duration time.Duration)
}

// RealSleeper implements Sleeper using time.Sleep
type RealSleeper struct{}

func (r RealSleeper) Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// Client is the Karakeep API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	sleeper    Sleeper
}

// NewClient creates a new Karakeep API client.
func NewClient(token string) *Client {
	return &Client{
		baseURL:    "https://api.karakeep.app/v1",
		httpClient: &http.Client{},
		token:      token,
		sleeper:    RealSleeper{},
	}
}

// SetBaseURL sets the base URL for the Karakeep API client.
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SetHTTPClient sets the HTTP client for the Karakeep API client.
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

// SetSleeper sets the sleeper for the Karakeep API client.
func (c *Client) SetSleeper(sleeper Sleeper) {
	c.sleeper = sleeper
}

// doRequestWithRetry performs an HTTP request with exponential backoff retry logic.
func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	const maxRetries = 5
	baseDelay := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		// If not rate limited, return the response
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		// Close the response body since we're retrying
		resp.Body.Close()

		// If this was the last attempt, return the rate limit error
		if attempt == maxRetries {
			return nil, fmt.Errorf("rate limited after %d retries: %s", maxRetries, resp.Status)
		}

		// Calculate delay with exponential backoff and jitter
		delay := baseDelay * time.Duration(1<<uint(attempt))
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		sleepDuration := delay + jitter

		log.Printf("Rate limited (429), retrying in %v (attempt %d/%d)", sleepDuration, attempt+1, maxRetries)
		c.sleeper.Sleep(sleepDuration)
	}

	// This should never be reached due to the loop structure
	return nil, fmt.Errorf("unexpected error in retry logic")
}

// Bookmark represents a Karakeep bookmark.
type Bookmark struct {
	ID          string   `json:"id,omitempty"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// List represents a Karakeep list.
type List struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}


// CreateBookmark creates a new bookmark in Karakeep and returns the created bookmark.
func (c *Client) CreateBookmark(bookmark *Bookmark) (*Bookmark, error) {
	jsonPayload, err := json.Marshal(bookmark)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/bookmarks", c.baseURL), bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create bookmark: %s", resp.Status)
	}

	var createdBookmark Bookmark
	if err := json.NewDecoder(resp.Body).Decode(&createdBookmark); err != nil {
		return nil, err
	}

	return &createdBookmark, nil
}

// CreateList creates a new list in Karakeep and returns the created list.
func (c *Client) CreateList(list *List) (*List, error) {
	jsonPayload, err := json.Marshal(list)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/lists", c.baseURL), bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create list: %s", resp.Status)
	}

	var createdList List
	if err := json.NewDecoder(resp.Body).Decode(&createdList); err != nil {
		return nil, err
	}

	return &createdList, nil
}

// AddBookmarkToList adds a bookmark to a list in Karakeep.
func (c *Client) AddBookmarkToList(bookmarkID, listID string) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/lists/%s/bookmarks/%s", c.baseURL, listID, bookmarkID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add bookmark to list: %s", resp.Status)
	}

	return nil
}

// GetAllBookmarks fetches all bookmarks from Karakeep.
func (c *Client) GetAllBookmarks() ([]*Bookmark, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/bookmarks", c.baseURL), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get bookmarks: %s", resp.Status)
	}

	var bookmarks []*Bookmark
	if err := json.NewDecoder(resp.Body).Decode(&bookmarks); err != nil {
		return nil, err
	}

	return bookmarks, nil
}

// GetAllLists fetches all lists from Karakeep.
func (c *Client) GetAllLists() ([]*List, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/lists", c.baseURL), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get lists: %s", resp.Status)
	}

	var lists []*List
	if err := json.NewDecoder(resp.Body).Decode(&lists); err != nil {
		return nil, err
	}

	return lists, nil
}

// DeleteBookmark deletes a bookmark from Karakeep.
func (c *Client) DeleteBookmark(bookmarkID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/bookmarks/%s", c.baseURL, bookmarkID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete bookmark: %s", resp.Status)
	}

	return nil
}

// DeleteList deletes a list from Karakeep.
func (c *Client) DeleteList(listID string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/lists/%s", c.baseURL, listID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete list: %s", resp.Status)
	}

	return nil
}
