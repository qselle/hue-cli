package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/qselle/hue-cli/internal/auth"
)

var (
	bridgeIP   string
	authRemote bool
	authManual bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Pair with a Hue Bridge",
	Long: `Authenticate with a Philips Hue Bridge.

By default, discovers and pairs with a bridge on your local network (link button).
Use --remote to authenticate via the Hue Cloud API (OAuth2) for remote access.

Remote mode requires HUE_CLIENT_ID, HUE_CLIENT_SECRET, and HUE_APP_ID environment variables.
Get them at https://developers.meethue.com/my-apps/`,
	RunE: runAuth,
}

var forgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Remove stored credentials",
	RunE:  runForget,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	RunE:  runAuthStatus,
}

func init() {
	authCmd.Flags().StringVar(&bridgeIP, "bridge-ip", "", "Bridge IP address (skips discovery, local mode only)")
	authCmd.Flags().BoolVar(&authRemote, "remote", false, "Use Hue Cloud API (OAuth2) for remote access")
	authCmd.Flags().BoolVar(&authManual, "manual", false, "Manually paste the authorization code (for headless servers, remote mode only)")
	authCmd.AddCommand(forgetCmd)
	authCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if authRemote {
		return runAuthRemote(cmd, args)
	}

	// Local mode
	ip := bridgeIP
	if ip == "" {
		fmt.Println("Discovering Hue bridges...")
		bridges, err := auth.DiscoverBridges(ctx)
		if err != nil {
			return fmt.Errorf("discovery failed: %w", err)
		}

		if len(bridges) == 0 {
			return fmt.Errorf("no Hue bridges found on your network")
		}

		ip = bridges[0].InternalIPAddress
		if len(bridges) == 1 {
			fmt.Printf("Found bridge: %s\n", ip)
		} else {
			fmt.Printf("Found %d bridges, using first: %s\n", len(bridges), ip)
		}
	}

	fmt.Println()
	fmt.Println("Press the link button on your Hue Bridge, then press Enter...")
	fmt.Scanln()

	fmt.Println("Pairing...")
	appKey, err := auth.PairWithRetry(ctx, ip, 30*time.Second)
	if err != nil {
		return fmt.Errorf("pairing failed: %w", err)
	}

	cfg := &auth.Config{
		Mode:     "local",
		BridgeIP: ip,
		AppKey:   appKey,
	}

	if err := auth.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if jsonOutput {
		out := map[string]any{"status": "paired", "mode": "local", "bridge_ip": ip}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println("Paired successfully!")
	return nil
}

func runAuthRemote(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	oauthCfg := getRemoteOAuthConfig()

	var cfg *auth.Config
	var err error
	if authManual {
		cfg, err = auth.LoginRemoteManual(ctx, oauthCfg)
	} else {
		cfg, err = auth.LoginRemoteBrowser(ctx, oauthCfg)
	}
	if err != nil {
		return fmt.Errorf("remote authentication failed: %w", err)
	}

	if err := auth.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if jsonOutput {
		out := map[string]any{"status": "authenticated", "mode": "remote"}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println("Authenticated with Hue Cloud successfully!")
	return nil
}

func runForget(cmd *cobra.Command, args []string) error {
	if err := auth.ClearConfig(); err != nil {
		return fmt.Errorf("forget failed: %w", err)
	}

	if jsonOutput {
		out := map[string]any{"status": "forgotten"}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println("Credentials removed.")
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := auth.LoadConfig()
	if err != nil {
		if jsonOutput {
			out := map[string]any{"authenticated": false}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		fmt.Println("Not authenticated. Run 'hue-cli auth' to get started.")
		return nil
	}

	if jsonOutput {
		out := map[string]any{
			"authenticated": true,
			"mode":          cfg.Mode,
		}
		if cfg.IsRemote() {
			out["token_expired"] = time.Now().Unix() >= cfg.ExpiresAt
		} else {
			out["bridge_ip"] = cfg.BridgeIP
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	if cfg.IsRemote() {
		status := "valid"
		if time.Now().Unix() >= cfg.ExpiresAt {
			status = "expired (will auto-refresh)"
		}
		fmt.Printf("Authenticated via Hue Cloud (token: %s)\n", status)
	} else {
		fmt.Printf("Paired with bridge at %s (local)\n", cfg.BridgeIP)
	}
	return nil
}

func getRemoteOAuthConfig() auth.OAuthConfig {
	clientID := os.Getenv("HUE_CLIENT_ID")
	clientSecret := os.Getenv("HUE_CLIENT_SECRET")
	appID := os.Getenv("HUE_APP_ID")

	if clientID == "" || clientSecret == "" || appID == "" {
		fmt.Fprintln(os.Stderr, "Error: HUE_CLIENT_ID, HUE_CLIENT_SECRET, and HUE_APP_ID environment variables are required.")
		fmt.Fprintln(os.Stderr, "Register an app at https://developers.meethue.com/my-apps/")
		os.Exit(1)
	}

	return auth.OAuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AppID:        appID,
	}
}
