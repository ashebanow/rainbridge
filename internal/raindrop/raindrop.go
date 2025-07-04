package raindrop

import (
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

// Client is the Raindrop.io API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
	sleeper    Sleeper
}

// NewClient creates a new Raindrop.io API client.
func NewClient(token string) *Client {
	return &Client{
		baseURL:    "https://api.raindrop.io/rest/v1",
		httpClient: &http.Client{},
		token:      token,
		sleeper:    RealSleeper{},
	}
}

// SetBaseURL sets the base URL for the Raindrop.io API client.
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SetHTTPClient sets the HTTP client for the Raindrop.io API client.
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

// SetSleeper sets the sleeper for the Raindrop.io API client.
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

// Raindrop represents a Raindrop.io bookmark.
type Raindrop struct {
	ID      int64    `json:"_id"`
	Title   string   `json:"title"`
	Excerpt string   `json:"excerpt"`
	Link    string   `json:"link"`
	Tags    []string `json:"tags"`
}

// Collection represents a Raindrop.io collection.
type Collection struct {
	ID    int64  `json:"_id"`
	Title string `json:"title"`
}

// GetRaindrops fetches all bookmarks from Raindrop.io.
func (c *Client) GetRaindrops() ([]Raindrop, error) {
	return c.GetRaindropsByCollection(0)
}

// GetRaindropsByCollection fetches all bookmarks from a specific collection in Raindrop.io.
func (c *Client) GetRaindropsByCollection(collectionID int64) ([]Raindrop, error) {
	var allRaindrops []Raindrop
	page := 0
	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/raindrops/%d?page=%d&perpage=50", c.baseURL, collectionID, page), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Authorization", "Bearer "+c.token)

		resp, err := c.doRequestWithRetry(req)
		if err != nil {
			return nil, err
		}
		
		// Process response and ensure body is closed immediately
		response, err := func() (struct{ Items []Raindrop `json:"items"` }, error) {
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				return struct{ Items []Raindrop `json:"items"` }{}, fmt.Errorf("failed to get raindrops: %s", resp.Status)
			}

			var response struct {
				Items []Raindrop `json:"items"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
				return response, err
			}

			return response, nil
		}()
		
		if err != nil {
			return nil, err
		}

		if len(response.Items) == 0 {
			break
		}

		allRaindrops = append(allRaindrops, response.Items...)
		page++
	}

	return allRaindrops, nil
}

// GetCollections fetches all collections from Raindrop.io.
func (c *Client) GetCollections() ([]Collection, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/collections", c.baseURL), nil)
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
		return nil, fmt.Errorf("failed to get collections: %s", resp.Status)
	}

	var response struct {
		Items []Collection `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Items, nil
}
