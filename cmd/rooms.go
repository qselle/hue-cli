package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var roomsCmd = &cobra.Command{
	Use:   "rooms",
	Short: "List all rooms",
	RunE:  runRooms,
}

func init() {
	rootCmd.AddCommand(roomsCmd)
}

func runRooms(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	client, err := getAPIClient()
	if err != nil {
		return err
	}

	rooms, err := client.ListRooms(ctx)
	if err != nil {
		return fmt.Errorf("fetching rooms: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rooms)
	}

	if len(rooms) == 0 {
		fmt.Println("No rooms found.")
		return nil
	}

	for _, r := range rooms {
		devices := len(r.Children)
		fmt.Printf("  %-30s  %d devices  %s\n", r.Metadata.Name, devices, r.ID[:8])
	}

	return nil
}
