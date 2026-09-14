// Package cli wires the cobra commands.
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/finctl/internal/config"
	"github.com/laurenschristian/finctl/internal/fin"
)

var (
	Version = "dev"

	flagJSON bool
	cfg      *config.Config
	client   *fin.Client
)

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "finctl",
		Short:         "CLI and MCP server for markets and macro data",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			if cfg, err = config.Load(); err != nil {
				return err
			}
			client = fin.New(cfg.URL)
			return nil
		},
	}
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "print raw JSON")
	root.AddCommand(doctorCmd(), mcpCmd())
	return root
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config and reachability",
		RunE: func(_ *cobra.Command, _ []string) error {
			state := map[string]any{"config": config.Path(), "url": client.BaseURL}
			if flagJSON {
				return emit(state)
			}
			fmt.Printf("config  %s\nurl     %s\n", config.Path(), client.BaseURL)
			return nil
		},
	}
}
