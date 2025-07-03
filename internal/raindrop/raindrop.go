package raindrop

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the Raindrop.io API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient creates a new Raindrop.io API client.
func NewClient(token string) *Client {
	return &Client{
		baseURL:    "https://api.raindrop.io/rest/v1",
		httpClient: &http.Client{},
		token:      token,
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

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to get raindrops: %s", resp.Status)
		}

		var response struct {
			Items []Raindrop `json:"items"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
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

	resp, err := c.httpClient.Do(req)
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
