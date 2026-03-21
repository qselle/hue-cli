package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/qselle/hue-cli/internal/auth"
)

var bridgeIP string

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Pair with a Hue Bridge",
	Long:  "Discover and pair with a Philips Hue Bridge on your local network.\nYou will need to press the link button on the bridge during pairing.",
	RunE:  runAuth,
}

var forgetCmd = &cobra.Command{
	Use:   "forget",
	Short: "Remove stored bridge credentials",
	RunE:  runForget,
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show pairing status",
	RunE:  runAuthStatus,
}

func init() {
	authCmd.Flags().StringVar(&bridgeIP, "bridge-ip", "", "Bridge IP address (skips discovery)")
	authCmd.AddCommand(forgetCmd)
	authCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(authCmd)
}

func runAuth(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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
		BridgeIP: ip,
		AppKey:   appKey,
	}

	if err := auth.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	if jsonOutput {
		out := map[string]any{"status": "paired", "bridge_ip": ip}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println("Paired successfully!")
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

	fmt.Println("Bridge credentials removed.")
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := auth.LoadConfig()
	if err != nil {
		if jsonOutput {
			out := map[string]any{"paired": false}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}
		fmt.Println("Not paired. Run 'hue-cli auth' to pair with a bridge.")
		return nil
	}

	if jsonOutput {
		out := map[string]any{
			"paired":    true,
			"bridge_ip": cfg.BridgeIP,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Printf("Paired with bridge at %s\n", cfg.BridgeIP)
	return nil
}
