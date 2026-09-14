package cli

import (
	"context"
	"fmt"

	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
	"github.com/spf13/cobra"
)

// section is one block of the daily brief: either data or a one-line reason it
// is unavailable. daily never blanks because one input failed.
type section struct {
	Data any    `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

// insiderCluster is one holding's recent Form 4 count.
type insiderCluster struct {
	Symbol string `json:"symbol"`
	Form4  int    `json:"form4"`
}

func sec(v any, err error) section {
	if err != nil {
		return section{Err: err.Error()}
	}
	return section{Data: v}
}

func dailyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "daily",
		Short: "Morning brief: book, buy-zones, macro, Fed odds, in one screen",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			brief := dailyBrief(ctx)
			return show(brief, func() string { return renderDaily(brief) })
		},
	}
}

// dailyBrief computes each section independently so one failure never blanks
// the others (the ibkrctl review pattern).
func dailyBrief(ctx context.Context) map[string]section {
	out := map[string]section{}

	pos, _, perr := portPositions(ctx)
	out["book"] = sec(pos, perr)

	if rows, err := watchRows(ctx); err != nil {
		out["buyZones"] = sec(nil, err)
	} else {
		var inRange []model.WatchRow
		for _, r := range rows {
			if r.InZone || r.BelowZone {
				inRange = append(inRange, r)
			}
		}
		out["buyZones"] = section{Data: inRange}
	}

	out["earnings"] = section{Err: "earnings needs FINCTL_FINNHUB_KEY (not wired yet)"}

	out["macro"] = sec(provider.TreasuryCurve(ctx, hx))
	out["fedOdds"] = sec(provider.FedOdds(ctx, hx))

	// Insider clusters need holdings; skip cleanly when the book is unavailable.
	if perr != nil {
		out["insider"] = section{Err: "needs the IBKR gateway (run `ibkrctl login`)"}
	} else {
		var cs []insiderCluster
		for _, p := range pos {
			if fs, err := provider.Insider(ctx, hx, p.Symbol, 90); err == nil && len(fs) > 0 {
				cs = append(cs, insiderCluster{Symbol: p.Symbol, Form4: len(fs)})
			}
		}
		out["insider"] = section{Data: cs}
	}
	return out
}

func renderDaily(b map[string]section) string {
	sb, w := newTab()
	line := func(title string, s section, body func()) {
		fmt.Fprintf(w, "== %s ==\n", title)
		if s.Err != "" {
			fmt.Fprintf(w, "  section unavailable: %s\n", s.Err)
			return
		}
		body()
	}

	line("Book", b["book"], func() {
		pos, _ := b["book"].Data.([]model.Position)
		if len(pos) == 0 {
			fmt.Fprintln(w, "  no positions")
			return
		}
		for i, p := range pos {
			if i >= 8 {
				fmt.Fprintf(w, "  ... and %d more\n", len(pos)-8)
				break
			}
			drift := ""
			if p.TargetWeight > 0 {
				drift = "  drift " + pctStr(p.Drift*100)
			}
			fmt.Fprintf(w, "  %s\t%s\t%.1f%%%s\n", p.Symbol, money(p.Value), p.Weight*100, drift)
		}
	})
	line("Buy-zones in range", b["buyZones"], func() {
		rows, _ := b["buyZones"].Data.([]model.WatchRow)
		if len(rows) == 0 {
			fmt.Fprintln(w, "  none in range")
			return
		}
		for _, r := range rows {
			tag := "in zone"
			if r.BelowZone {
				tag = fmt.Sprintf("%.0f%% below", r.DistanceToBuyPct)
			}
			fmt.Fprintf(w, "  %s\t%.2f\t(zone %.0f-%.0f, %s)\n", r.Symbol, r.Last, r.BuyLow, r.BuyHigh, tag)
		}
	})
	line("Earnings this week", b["earnings"], func() {})
	line("Macro today", b["macro"], func() {
		c, _ := b["macro"].Data.([]model.RatePoint)
		for _, p := range c {
			if p.Tenor == "3M" || p.Tenor == "2Y" || p.Tenor == "10Y" || p.Tenor == "30Y" {
				fmt.Fprintf(w, "  %s\t%.2f%%\n", p.Tenor, p.Yield)
			}
		}
		if v, ok := spreadOf(c, "10Y", "2Y"); ok {
			fmt.Fprintf(w, "  2s10s\t%+.0fbp\n", v)
		}
	})
	line("Fed odds", b["fedOdds"], func() {
		o, _ := b["fedOdds"].Data.(*model.FedOdds)
		fmt.Fprintf(w, "  meeting\t%s\n", o.Meeting)
		for i, k := range o.Buckets {
			if i >= 3 || k.Prob < 0.01 {
				break
			}
			fmt.Fprintf(w, "  %s\t%.0f%%\n", k.Band, k.Prob*100)
		}
	})
	line("Insider clusters", b["insider"], func() {
		if cs, ok := b["insider"].Data.([]insiderCluster); ok && len(cs) > 0 {
			for _, c := range cs {
				fmt.Fprintf(w, "  %s\t%d Form 4 (90d)\n", c.Symbol, c.Form4)
			}
		} else {
			fmt.Fprintln(w, "  none")
		}
	})
	_ = w.Flush()
	return sb.String()
}
