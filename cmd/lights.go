package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qselle/hue-cli/internal/api"
	"github.com/qselle/hue-cli/internal/auth"
	"github.com/qselle/hue-cli/internal/format"
)

var lightsCmd = &cobra.Command{
	Use:   "lights",
	Short: "List all lights",
	RunE:  runLights,
}

var (
	lightOn         string
	lightBrightness float64
	lightColor      string
)

var lightSetCmd = &cobra.Command{
	Use:   "set <name-or-id>",
	Short: "Control a light",
	Long:  "Set the state of a light by name or ID.\nExamples:\n  hue-cli lights set \"Desk Lamp\" --on true --brightness 80\n  hue-cli lights set \"Desk Lamp\" --color ff0000",
	Args:  cobra.ExactArgs(1),
	RunE:  runLightSet,
}

func init() {
	lightSetCmd.Flags().StringVar(&lightOn, "on", "", "Turn light on or off (true/false)")
	lightSetCmd.Flags().Float64VarP(&lightBrightness, "brightness", "b", -1, "Brightness (0-100)")
	lightSetCmd.Flags().StringVarP(&lightColor, "color", "c", "", "Color as hex RGB (e.g. ff0000)")
	lightsCmd.AddCommand(lightSetCmd)
	rootCmd.AddCommand(lightsCmd)
}

func runLights(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	lights, err := client.ListLights(ctx)
	if err != nil {
		return fmt.Errorf("fetching lights: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(lights)
	}

	if len(lights) == 0 {
		fmt.Println("No lights found.")
		return nil
	}

	for _, l := range lights {
		status := format.OnOff(l.On.On)
		bri := ""
		if l.Dimming != nil {
			bri = format.Brightness(l.Dimming.Brightness)
		}
		color := ""
		if l.Color != nil && l.Dimming != nil {
			color = format.XYToRGBHex(l.Color.XY.X, l.Color.XY.Y, l.Dimming.Brightness)
		}
		fmt.Printf("  %-3s  %-6s  %-30s  %s  %s\n", status, bri, l.Metadata.Name, color, l.ID[:8])
	}

	return nil
}

func runLightSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	target := args[0]

	// Resolve name to ID
	lightID, err := resolveLightID(ctx, client, target)
	if err != nil {
		return err
	}

	var payload api.SetLightPayload

	if lightOn != "" {
		on := strings.ToLower(lightOn) == "true" || lightOn == "1"
		payload.On = &api.OnState{On: on}
	}

	if lightBrightness >= 0 {
		payload.Dimming = &api.Dimming{Brightness: lightBrightness}
	}

	if lightColor != "" {
		x, y, err := format.HexToXY(lightColor)
		if err != nil {
			return fmt.Errorf("invalid color: %w", err)
		}
		payload.Color = &api.Color{XY: api.XYColor{X: x, Y: y}}
	}

	if err := client.SetLight(ctx, lightID, payload); err != nil {
		return fmt.Errorf("setting light: %w", err)
	}

	if jsonOutput {
		out := map[string]any{"status": "ok", "light_id": lightID}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println("Light updated.")
	return nil
}

func resolveLightID(ctx context.Context, client *api.Client, nameOrID string) (string, error) {
	lights, err := client.ListLights(ctx)
	if err != nil {
		return "", fmt.Errorf("listing lights: %w", err)
	}

	// Check if it's a direct ID match
	for _, l := range lights {
		if l.ID == nameOrID {
			return l.ID, nil
		}
	}

	// Check by name (case-insensitive)
	for _, l := range lights {
		if strings.EqualFold(l.Metadata.Name, nameOrID) {
			return l.ID, nil
		}
	}

	return "", fmt.Errorf("light not found: %s", nameOrID)
}

func getAPIClient() (*api.Client, error) {
	cfg, err := auth.LoadConfig()
	if err != nil {
		return nil, err
	}
	return api.NewClient(cfg.BridgeIP, cfg.AppKey), nil
}
