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

// Raindrop represents a Raindrop.io bookmark.
type Raindrop struct {
	ID      int64    `json:"_id"`
	Title   string   `json:"title"`
	Excerpt string   `json:"excerpt"`
	Link    string   `json:"link"`
	Tags    []string `json:"tags"`
}

// GetRaindrops fetches all bookmarks from Raindrop.io.
func (c *Client) GetRaindrops() ([]Raindrop, error) {
	var allRaindrops []Raindrop
	page := 0
	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/raindrops/0?page=%d&perpage=50", c.baseURL, page), nil)
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
