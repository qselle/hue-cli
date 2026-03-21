package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	DiscoveryURL = "https://discovery.meethue.com"
	AppName      = "hue-cli"
	DeviceType   = "hue-cli#cli"
)

type DiscoveredBridge struct {
	ID                string `json:"id"`
	InternalIPAddress string `json:"internalipaddress"`
	Port              int    `json:"port"`
}

// DiscoverBridges finds Hue bridges on the network via the Philips discovery endpoint.
func DiscoverBridges(ctx context.Context) ([]DiscoveredBridge, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", DiscoveryURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bridge discovery failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading discovery response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery API returned %d: %s", resp.StatusCode, string(body))
	}

	var bridges []DiscoveredBridge
	if err := json.Unmarshal(body, &bridges); err != nil {
		return nil, fmt.Errorf("parsing discovery response: %w", err)
	}

	return bridges, nil
}

type pairResponse struct {
	Success *struct {
		Username string `json:"username"`
	} `json:"success,omitempty"`
	Error *struct {
		Type        int    `json:"type"`
		Address     string `json:"address"`
		Description string `json:"description"`
	} `json:"error,omitempty"`
}

// Pair attempts to create an application key on the bridge.
// The user must press the link button on the bridge before calling this.
func Pair(ctx context.Context, bridgeIP string) (string, error) {
	body := fmt.Sprintf(`{"devicetype":"%s","generateclientkey":true}`, DeviceType)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	url := fmt.Sprintf("https://%s/api", bridgeIP)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("pairing request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading pairing response: %w", err)
	}

	var results []pairResponse
	if err := json.Unmarshal(respBody, &results); err != nil {
		return "", fmt.Errorf("parsing pairing response: %w", err)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("empty pairing response")
	}

	if results[0].Error != nil {
		return "", fmt.Errorf("pairing failed: %s", results[0].Error.Description)
	}

	if results[0].Success == nil || results[0].Success.Username == "" {
		return "", fmt.Errorf("unexpected pairing response")
	}

	return results[0].Success.Username, nil
}

// PairWithRetry polls the bridge for pairing, waiting for the user to press the link button.
func PairWithRetry(ctx context.Context, bridgeIP string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		appKey, err := Pair(ctx, bridgeIP)
		if err == nil {
			return appKey, nil
		}

		// Error type 101 = link button not pressed
		if !strings.Contains(err.Error(), "link button") {
			return "", err
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}

	return "", fmt.Errorf("timed out waiting for link button press")
}
