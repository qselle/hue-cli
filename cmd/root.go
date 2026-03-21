package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "hue-cli",
	Short: "Hue CLI — control your Philips Hue lights",
	Long:  "A command-line interface and MCP server for the Philips Hue Bridge.\nControl lights, scenes, and rooms from your terminal or through AI agents.",
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}
