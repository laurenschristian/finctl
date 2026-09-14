// Package cli wires the cobra commands over the provider layer.
package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/finctl/internal/cache"
	"github.com/laurenschristian/finctl/internal/config"
	"github.com/laurenschristian/finctl/internal/httpx"
	"github.com/laurenschristian/finctl/internal/ibkr"
)

var (
	Version = "dev"

	flagJSON bool
	cfg      *config.Config
	store    *cache.Store
	hx       *httpx.Client
)

// noSetup lists commands that run without opening the cache/HTTP layer.
func noSetup(cmd *cobra.Command) bool {
	switch cmd.Name() {
	case "completion", "help", "__complete", "__completeNoDesc", "keys":
		return true
	}
	if p := cmd.Parent(); p != nil && (p.Name() == "completion" || p.Name() == "help") {
		return true
	}
	return false
}

func Root() *cobra.Command {
	root := &cobra.Command{
		Use:           "finctl",
		Short:         "CLI and MCP server for markets, macro, and research data",
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			if cfg, err = config.Load(); err != nil {
				return err
			}
			if noSetup(cmd) {
				return nil
			}
			if store, err = cache.Open(cfg.CacheFile()); err != nil {
				return err
			}
			hx = httpx.New(store, cfg.UA())
			return nil
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			if store != nil {
				_ = store.Close()
			}
		},
	}
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "print raw JSON")
	root.AddCommand(
		quoteCmd(),
		chartCmd(),
		fundCmd(),
		capexCmd(),
		cryptoCmd(),
		fxCmd(),
		ratesCmd(),
		curveCmd(),
		seriesCmd(),
		fedoddsCmd(),
		cotCmd(),
		fiscalCmd(),
		energyCmd(),
		macroCmd(),
		twRevenueCmd(),
		gpuRentCmd(),
		shortCmd(),
		optionsCmd(),
		filingsCmd(),
		insiderCmd(),
		dilutionCmd(),
		researchCmd(),
		lensCmd(),
		portCmd(),
		watchlistCmd(),
		voicesCmd(),
		voiceCmd(),
		dailyCmd(),
		cacheCmd(),
		keysCmd(),
		doctorCmd(),
		mcpCmd(),
	)
	return root
}

func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// show prints a table by default, or JSON with --json.
func show(v any, render func() string) error {
	if flagJSON {
		return emit(v)
	}
	fmt.Print(render())
	return nil
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check config, cache size, keys, and IBKR gateway reachability",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cacheSize := "0"
			if fi, err := os.Stat(cfg.CacheFile()); err == nil {
				cacheSize = abbr(float64(fi.Size())) + "B"
			}
			gateway := "not authenticated"
			if ok, err := ibkr.New(cfg.IBKRURL).Authenticated(cmd.Context()); err != nil {
				gateway = "unreachable"
			} else if ok {
				gateway = "authenticated"
			}
			state := map[string]any{
				"config":     config.Path(),
				"cache":      cfg.CacheFile(),
				"cacheSize":  cacheSize,
				"ibkr_url":   cfg.IBKRURL,
				"gateway":    gateway,
				"user_agent": cfg.UA(),
				"keys":       cfg.Configured(),
			}
			if flagJSON {
				return emit(state)
			}
			fmt.Printf("config      %s\n", config.Path())
			fmt.Printf("cache       %s (%s)\n", cfg.CacheFile(), cacheSize)
			fmt.Printf("ibkr_url    %s (%s)\n", cfg.IBKRURL, gateway)
			fmt.Printf("user_agent  %s\n", cfg.UA())
			fmt.Printf("keys        %v\n", cfg.Configured())
			return nil
		},
	}
}

func keysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keys",
		Short: "Show which provider API keys are configured (no secrets printed)",
		RunE: func(_ *cobra.Command, _ []string) error {
			conf := cfg.Configured()
			if flagJSON {
				return emit(conf)
			}
			for _, p := range []string{"fred", "eia", "bls", "finnhub"} {
				status := "missing"
				if conf[p] {
					status = "set"
				}
				fmt.Printf("%-8s %s\n", p, status)
			}
			return nil
		},
	}
}

func cacheCmd() *cobra.Command {
	c := &cobra.Command{Use: "cache", Short: "Manage the on-disk cache"}
	c.AddCommand(&cobra.Command{
		Use:   "clear [provider]",
		Short: "Clear the cache (all, or one provider)",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := ""
			if len(args) > 0 {
				p = args[0]
			}
			if err := store.Clear(cmd.Context(), p); err != nil {
				return err
			}
			fmt.Println("cache cleared")
			return nil
		},
	})
	return c
}
