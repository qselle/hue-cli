package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	HueAuthURL  = "https://api.meethue.com/v2/oauth2/authorize"
	HueTokenURL = "https://api.meethue.com/v2/oauth2/token"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	AppID        string
}

type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// LoginRemoteBrowser starts a local callback server and opens the browser for Hue OAuth2.
func LoginRemoteBrowser(ctx context.Context, cfg OAuthConfig) (*Config, error) {
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	deviceID, err := randomDeviceID()
	if err != nil {
		return nil, fmt.Errorf("generating device ID: %w", err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting callback server: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	authURL := buildHueAuthURL(cfg.ClientID, cfg.AppID, deviceID, redirectURI, state)
	fmt.Printf("Opening browser for Hue authorization...\n")
	fmt.Printf("If it doesn't open, visit:\n%s\n\n", authURL)
	openBrowser(authURL)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("invalid state parameter")
			http.Error(w, "Invalid state", http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			http.Error(w, "No code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		fmt.Fprintf(w, "<html><body><h1>Authorized!</h1><p>You can close this window and return to the terminal.</p></body></html>")
	})

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(ctx)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authorization timed out after 5 minutes")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return exchangeHueCode(ctx, cfg, code)
}

// LoginRemoteManual prints the auth URL and waits for the user to paste the code.
func LoginRemoteManual(ctx context.Context, cfg OAuthConfig) (*Config, error) {
	state, err := randomState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	deviceID, err := randomDeviceID()
	if err != nil {
		return nil, fmt.Errorf("generating device ID: %w", err)
	}

	redirectURI := "http://localhost"
	authURL := buildHueAuthURL(cfg.ClientID, cfg.AppID, deviceID, redirectURI, state)

	fmt.Println("Open this URL in your browser:")
	fmt.Println()
	fmt.Println(authURL)
	fmt.Println()
	fmt.Println("Authorize the app, then copy the 'code' parameter from the redirect URL.")
	fmt.Print("Paste the code here: ")

	var code string
	fmt.Scanln(&code)
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("empty code")
	}

	return exchangeHueCode(ctx, cfg, code)
}

// RefreshRemoteToken refreshes an expired access token.
func RefreshRemoteToken(ctx context.Context, cfg *Config) error {
	basicAuth := base64.StdEncoding.EncodeToString(
		[]byte(cfg.ClientID + ":" + cfg.ClientSecret),
	)

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {cfg.RefreshToken},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", HueTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	cfg.AccessToken = tokenResp.AccessToken
	cfg.RefreshToken = tokenResp.RefreshToken
	cfg.ExpiresAt = time.Now().Unix() + tokenResp.ExpiresIn

	return SaveConfig(cfg)
}

// GetValidRemoteConfig loads the config and refreshes the token if expired.
func GetValidRemoteConfig(ctx context.Context) (*Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	if !cfg.IsRemote() {
		return cfg, nil
	}

	if time.Now().Unix() < cfg.ExpiresAt {
		return cfg, nil
	}

	if err := RefreshRemoteToken(ctx, cfg); err != nil {
		return nil, fmt.Errorf("token expired and refresh failed: %w", err)
	}

	return cfg, nil
}

func exchangeHueCode(ctx context.Context, cfg OAuthConfig, code string) (*Config, error) {
	basicAuth := base64.StdEncoding.EncodeToString(
		[]byte(cfg.ClientID + ":" + cfg.ClientSecret),
	)

	data := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", HueTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+basicAuth)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, string(body))
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}

	return &Config{
		Mode:         "remote",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Unix() + tokenResp.ExpiresIn,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
	}, nil
}

func buildHueAuthURL(clientID, appID, deviceID, redirectURI, state string) string {
	params := url.Values{
		"clientid":      {clientID},
		"appid":         {appID},
		"deviceid":      {deviceID},
		"devicename":    {"hue-cli"},
		"state":         {state},
		"response_type": {"code"},
	}
	return HueAuthURL + "?" + params.Encode()
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomDeviceID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
