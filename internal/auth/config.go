package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LocalConfig struct {
	BridgeIP string `json:"bridge_ip"`
	AppKey   string `json:"app_key"`
}

type RemoteConfig struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AppKey       string `json:"app_key"` // whitelist username on bridge, needed for CLIP v2
}

type Config struct {
	Active string        `json:"active"` // "local" or "remote"
	Local  *LocalConfig  `json:"local,omitempty"`
	Remote *RemoteConfig `json:"remote,omitempty"`
}

func (c *Config) IsRemote() bool {
	return c.Active == "remote"
}

func (c *Config) HasLocal() bool {
	return c.Local != nil && c.Local.BridgeIP != "" && c.Local.AppKey != ""
}

func (c *Config) HasRemote() bool {
	return c.Remote != nil && c.Remote.AccessToken != ""
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "hue-cli")
	return dir, os.MkdirAll(dir, 0700)
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func SaveConfig(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, fmt.Errorf("getting config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not authenticated — run 'hue-cli auth' first")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

// LoadOrCreate loads existing config or returns a new empty one.
func LoadOrCreate() *Config {
	cfg, err := LoadConfig()
	if err != nil {
		return &Config{}
	}
	return cfg
}

// SetLocal updates the local credentials, preserving remote credentials.
func (c *Config) SetLocal(bridgeIP, appKey string) {
	c.Local = &LocalConfig{
		BridgeIP: bridgeIP,
		AppKey:   appKey,
	}
	c.Active = "local"
}

// SetRemote updates the remote credentials, preserving local credentials.
func (c *Config) SetRemote(remote *RemoteConfig) {
	c.Remote = remote
	c.Active = "remote"
}

func ClearConfig() error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
