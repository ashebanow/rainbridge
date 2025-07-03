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

// SetBaseURL sets the base URL for the Karakeep API client.
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// SetHTTPClient sets the HTTP client for the Karakeep API client.
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
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

	resp, err := c.httpClient.Do(req)
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

	resp, err := c.httpClient.Do(req)
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add bookmark to list: %s", resp.Status)
	}

	return nil
}
