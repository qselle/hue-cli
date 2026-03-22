package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const RemoteBaseURL = "https://api.meethue.com/route/clip/v2"

type Client struct {
	httpClient  *http.Client
	baseURL     string
	appKey      string // local mode
	bearerToken string // remote mode
}

// NewLocalClient creates a client for the local Hue Bridge API.
func NewLocalClient(bridgeIP, appKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		baseURL: fmt.Sprintf("https://%s/clip/v2", bridgeIP),
		appKey:  appKey,
	}
}

// NewRemoteClient creates a client for the Hue Remote API (cloud).
// Requires both the OAuth2 bearer token and the app key (whitelist username).
func NewRemoteClient(bearerToken, appKey string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		baseURL:     RemoteBaseURL,
		appKey:      appKey,
		bearerToken: bearerToken,
	}
}

func (c *Client) setAuth(req *http.Request) {
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}
	if c.appKey != "" {
		req.Header.Set("hue-application-key", c.appKey)
	}
}

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	url := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) put(ctx context.Context, path string, payload any) ([]byte, error) {
	url := c.baseURL + path

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (c *Client) getJSON(ctx context.Context, path string, v any) error {
	body, err := c.get(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
