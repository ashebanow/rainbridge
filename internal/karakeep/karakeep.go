package karakeep

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the Karakeep API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient creates a new Karakeep API client.
func NewClient(token string) *Client {
	return &Client{
		baseURL:    "https://api.karakeep.app/v1",
		httpClient: &http.Client{},
		token:      token,
	}
}

// Bookmark represents a Karakeep bookmark.
type Bookmark struct {
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// CreateBookmark creates a new bookmark in Karakeep.
func (c *Client) CreateBookmark(bookmark *Bookmark) error {
	jsonPayload, err := json.Marshal(bookmark)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/bookmarks", c.baseURL), bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create bookmark: %s", resp.Status)
	}

	return nil
}
