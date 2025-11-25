package swaggerhub

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	if baseURL == "" {
		baseURL = "https://api.swaggerhub.com"
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

type APIListing struct {
	APIs []APISummary `json:"apis"`
}

type APISummary struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Properties  []Property `json:"properties"`
}

type Property struct {
	Type  string `json:"type"`
	URL   string `json:"url,omitempty"`
	Value string `json:"value,omitempty"`
}

func (c *Client) ListAPIs(owner string) ([]APISummary, error) {
	u, err := url.Parse(fmt.Sprintf("%s/apis/%s", c.baseURL, owner))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("swaggerhub API returned status %d", resp.StatusCode)
	}

	var listing APIListing
	if err := json.NewDecoder(resp.Body).Decode(&listing); err != nil {
		return nil, err
	}

	return listing.APIs, nil
}

func (c *Client) GetAPISpec(specURL string) ([]byte, error) {
	log.Printf("Fetching spec: %s", specURL)
	req, err := http.NewRequest("GET", specURL, nil)
	if err != nil {
		return nil, err
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", c.apiKey)
	}

	req.Header.Set("Accept", "text/yaml") // Get the spec in YAML format

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch spec: status %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
