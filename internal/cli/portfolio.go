package cli

import (
	"context"
	"fmt"
	"sort"

	"github.com/laurenschristian/finctl/internal/ibkr"
	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
	"github.com/laurenschristian/finctl/internal/vault"
	"github.com/spf13/cobra"
)

// portPositions fetches positions from the gateway and annotates them with vault
// target weights and drift.
func portPositions(ctx context.Context) ([]model.Position, map[string]float64, error) {
	cl := ibkr.New(cfg.IBKRURL)
	ok, err := cl.Authenticated(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, nil, fmt.Errorf("IBKR gateway not authenticated: run `ibkrctl login` (2FA), then retry")
	}
	pos, err := cl.Positions(ctx)
	if err != nil {
		return nil, nil, err
	}
	targets := map[string]float64{}
	for _, t := range vault.Targets(cfg.VaultDir) {
		targets[t.Ticker] = t.Weight
	}
	for i := range pos {
		if tw, ok := targets[pos[i].Symbol]; ok {
			pos[i].TargetWeight = tw
			pos[i].Drift = pos[i].Weight - tw
		}
	}
	return pos, targets, nil
}

// watchRows loads the vault watchlist and adds live prices and distance to buy.
func watchRows(ctx context.Context) ([]model.WatchRow, error) {
	items := vault.Watchlist(cfg.VaultDir)
	if len(items) == 0 {
		return nil, fmt.Errorf("no watchlist found: set vault_dir and a watchlist note with a ticker + buy-zone table")
	}
	var rows []model.WatchRow
	for _, it := range items {
		r := model.WatchRow{Symbol: it.Ticker, BuyLow: it.BuyLow, BuyHigh: it.BuyHigh, Thesis: it.Thesis}
		q, err := provider.CboeQuote(ctx, hx, it.Ticker)
		if err != nil || q == nil || q.Last == 0 {
			q, _ = provider.YahooQuote(ctx, hx, it.Ticker)
		}
		if q != nil {
			r.Last = q.Last
			switch {
			case it.BuyHigh <= 0:
				// single-point or missing zone: nothing to compare against
			case it.BuyLow > 0 && r.Last < it.BuyLow:
				r.BelowZone = true
				r.DistanceToBuyPct = (r.Last - it.BuyLow) / it.BuyLow * 100
			case r.Last <= it.BuyHigh:
				r.InZone = true
			default:
				r.DistanceToBuyPct = (r.Last - it.BuyHigh) / it.BuyHigh * 100
			}
		}
		rows = append(rows, r)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].DistanceToBuyPct < rows[j].DistanceToBuyPct })
	return rows, nil
}

func portCmd() *cobra.Command {
	var driftFlag float64
	c := &cobra.Command{
		Use:   "port",
		Short: "Positions and P&L from the IBKR gateway, vs vault target weights",
		RunE: func(cmd *cobra.Command, _ []string) error {
			pos, targets, err := portPositions(cmd.Context())
			if err != nil {
				return err
			}
			return show(pos, func() string {
				b, w := newTab()
				hasTargets := len(targets) > 0
				if hasTargets {
					fmt.Fprintln(w, "SYMBOL\tQTY\tVALUE\tWEIGHT\tTARGET\tDRIFT")
				} else {
					fmt.Fprintln(w, "SYMBOL\tQTY\tVALUE\tWEIGHT\tP&L")
				}
				for _, p := range pos {
					if hasTargets {
						drift := ""
						if p.TargetWeight > 0 {
							drift = pctStr(p.Drift * 100)
							if p.Drift*100 > driftFlag || p.Drift*100 < -driftFlag {
								drift += " *"
							}
						}
						fmt.Fprintf(w, "%s\t%.0f\t%s\t%.1f%%\t%.1f%%\t%s\n",
							p.Symbol, p.Qty, money(p.Value), p.Weight*100, p.TargetWeight*100, drift)
					} else {
						fmt.Fprintf(w, "%s\t%.0f\t%s\t%.1f%%\t%s\n",
							p.Symbol, p.Qty, money(p.Value), p.Weight*100, money(p.UnrealPnL))
					}
				}
				_ = w.Flush()
				out := b.String()
				if !hasTargets {
					out += "no vault targets found (set vault_dir and a Dashboard table for drift)\n"
				}
				return out
			})
		},
	}
	c.Flags().Float64Var(&driftFlag, "drift", 3, "flag drift beyond this many percentage points")
	return c
}

func watchlistCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watchlist",
		Short: "Vault watchlist with live price and distance to buy-zone",
		RunE: func(cmd *cobra.Command, _ []string) error {
			rows, err := watchRows(cmd.Context())
			if err != nil {
				return err
			}
			return show(rows, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "SYMBOL\tLAST\tBUY ZONE\tTO BUY")
				for _, r := range rows {
					zone := ""
					if r.BuyHigh > 0 {
						zone = fmt.Sprintf("%.0f-%.0f", r.BuyLow, r.BuyHigh)
					}
					tobuy := ""
					switch {
					case r.InZone:
						tobuy = "in zone"
					case r.BelowZone:
						tobuy = fmt.Sprintf("%.1f%% below", r.DistanceToBuyPct)
					case r.BuyHigh > 0:
						tobuy = fmt.Sprintf("+%.1f%%", r.DistanceToBuyPct)
					}
					fmt.Fprintf(w, "%s\t%.2f\t%s\t%s\n", r.Symbol, r.Last, zone, tobuy)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}
