package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qselle/hue-cli/internal/api"
)

var scenesCmd = &cobra.Command{
	Use:   "scenes",
	Short: "List all scenes",
	RunE:  runScenes,
}

var sceneActivateCmd = &cobra.Command{
	Use:   "activate <name-or-id>",
	Short: "Activate a scene",
	Args:  cobra.ExactArgs(1),
	RunE:  runSceneActivate,
}

func init() {
	scenesCmd.AddCommand(sceneActivateCmd)
	rootCmd.AddCommand(scenesCmd)
}

func runScenes(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	scenes, err := client.ListScenes(ctx)
	if err != nil {
		return fmt.Errorf("fetching scenes: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(scenes)
	}

	if len(scenes) == 0 {
		fmt.Println("No scenes found.")
		return nil
	}

	for _, s := range scenes {
		status := ""
		if s.Status.Active == "active" {
			status = " (active)"
		}
		fmt.Printf("  %-30s  %s%s\n", s.Metadata.Name, s.ID[:8], status)
	}

	return nil
}

func runSceneActivate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	target := args[0]
	sceneID, err := resolveSceneID(ctx, client, target)
	if err != nil {
		return err
	}

	if err := client.ActivateScene(ctx, sceneID); err != nil {
		return fmt.Errorf("activating scene: %w", err)
	}

	if jsonOutput {
		out := map[string]any{"status": "ok", "scene_id": sceneID}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}

	fmt.Println("Scene activated.")
	return nil
}

func resolveSceneID(ctx context.Context, client *api.Client, nameOrID string) (string, error) {
	scenes, err := client.ListScenes(ctx)
	if err != nil {
		return "", fmt.Errorf("listing scenes: %w", err)
	}

	for _, s := range scenes {
		if s.ID == nameOrID {
			return s.ID, nil
		}
	}

	for _, s := range scenes {
		if strings.EqualFold(s.Metadata.Name, nameOrID) {
			return s.ID, nil
		}
	}

	return "", fmt.Errorf("scene not found: %s", nameOrID)
}
