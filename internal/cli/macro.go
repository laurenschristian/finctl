package cli

import (
	"fmt"

	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
	"github.com/spf13/cobra"
)

func curveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "curve",
		Short: "Treasury par yield curve with 2s10s / 3m10y spreads (keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := provider.TreasuryCurve(cmd.Context(), hx)
			if err != nil {
				return err
			}
			return show(c, func() string { return renderCurve(c) })
		},
	}
}

func ratesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rates",
		Short: "Key Treasury tenors and curve spreads (keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := provider.TreasuryCurve(cmd.Context(), hx)
			if err != nil {
				return err
			}
			return show(c, func() string { return renderRates(c) })
		},
	}
}

func renderCurve(c []model.RatePoint) string {
	b, w := newTab()
	fmt.Fprintln(w, "TENOR\tYIELD")
	for _, p := range c {
		fmt.Fprintf(w, "%s\t%.2f%%\n", p.Tenor, p.Yield)
	}
	_ = w.Flush()
	out := b.String()
	return out + spreadsLine(c)
}

func renderRates(c []model.RatePoint) string {
	keep := map[string]bool{"3M": true, "2Y": true, "10Y": true, "30Y": true}
	b, w := newTab()
	fmt.Fprintln(w, "TENOR\tYIELD")
	var asOf string
	for _, p := range c {
		asOf = p.AsOf
		if keep[p.Tenor] {
			fmt.Fprintf(w, "%s\t%.2f%%\n", p.Tenor, p.Yield)
		}
	}
	_ = w.Flush()
	out := b.String() + spreadsLine(c)
	if asOf != "" {
		out += "as of " + asOf + "\n"
	}
	return out
}

func spreadsLine(c []model.RatePoint) string {
	s := ""
	if v, ok := spreadOf(c, "10Y", "2Y"); ok {
		s += fmt.Sprintf("2s10s %+.0fbp  ", v)
	}
	if v, ok := spreadOf(c, "10Y", "3M"); ok {
		s += fmt.Sprintf("3m10y %+.0fbp", v)
	}
	if s != "" {
		s += "\n"
	}
	return s
}

// spreadOf is a thin exported-free wrapper so cli need not import the helper.
func spreadOf(c []model.RatePoint, a, b string) (float64, bool) {
	var av, bv float64
	var ao, bo bool
	for _, p := range c {
		switch p.Tenor {
		case a:
			av, ao = p.Yield, true
		case b:
			bv, bo = p.Yield, true
		}
	}
	if !ao || !bo {
		return 0, false
	}
	return (av - bv) * 100, true
}

func fedoddsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fedodds",
		Short: "Market-implied FOMC target-rate odds (Kalshi, keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			o, err := provider.FedOdds(cmd.Context(), hx)
			if err != nil {
				return err
			}
			return show(o, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "meeting %s  (%s)\n", o.Meeting, o.Source)
				fmt.Fprintln(w, "TARGET BAND\tPROB")
				for _, k := range o.Buckets {
					if k.Prob < 0.005 {
						continue
					}
					fmt.Fprintf(w, "%s\t%.0f%%\n", k.Band, k.Prob*100)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func cotCmd() *cobra.Command {
	var market string
	c := &cobra.Command{
		Use:   "cot",
		Short: "CFTC Commitments of Traders net non-commercial positioning (keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			r, err := provider.Cot(cmd.Context(), hx, market)
			if err != nil {
				return err
			}
			return show(r, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "%s  (week of %s)\n", r.Market, r.Date)
				fmt.Fprintf(w, "long\t%s\n", money(r.Long))
				fmt.Fprintf(w, "short\t%s\n", money(r.Short))
				fmt.Fprintf(w, "net\t%s\n", money(r.Net))
				if r.NetPrior != 0 {
					fmt.Fprintf(w, "net prior\t%s\t(%s)\n", money(r.NetPrior), money(r.Net-r.NetPrior))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().StringVar(&market, "market", "ES", "market code: ES NQ GC CL NG ZN DX BTC ...")
	return c
}

func seriesCmd() *cobra.Command {
	var n int
	c := &cobra.Command{
		Use:   "series <FRED_ID>",
		Short: "Last N observations of a FRED series (needs FINCTL_FRED_KEY)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, _ := cfg.Key("fred")
			s, err := provider.FredSeries(cmd.Context(), hx, key, args[0], n)
			if err != nil {
				return err
			}
			return show(s, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "%s\n", s.ID)
				fmt.Fprintln(w, "DATE\tVALUE")
				for _, o := range s.Points {
					fmt.Fprintf(w, "%s\t%.2f\n", o.Date, o.Value)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().IntVar(&n, "n", 12, "number of observations")
	return c
}

func fiscalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fiscal",
		Short: "US debt to the penny (Treasury FiscalData, keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			d, err := provider.TreasuryDebt(cmd.Context(), hx)
			if err != nil {
				return err
			}
			return show(d, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "as of %s\n", d.Date)
				fmt.Fprintf(w, "total debt\t%s\n", abbr(d.TotalDebt))
				fmt.Fprintf(w, "held by public\t%s\n", abbr(d.HeldPublic))
				fmt.Fprintf(w, "intragov\t%s\n", abbr(d.Intragov))
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func energyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "energy",
		Short: "Energy futures: WTI, Brent, natural gas (Yahoo, keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			syms := []struct{ name, sym string }{
				{"WTI crude", "CL=F"}, {"Brent crude", "BZ=F"}, {"Natural gas", "NG=F"}, {"Gasoline", "RB=F"},
			}
			type row struct {
				Name  string       `json:"name"`
				Quote *model.Quote `json:"quote,omitempty"`
				Err   string       `json:"error,omitempty"`
			}
			var rows []row
			for _, s := range syms {
				q, err := provider.YahooQuote(cmd.Context(), hx, s.sym)
				r := row{Name: s.name, Quote: q}
				if err != nil {
					r.Err = err.Error()
				}
				rows = append(rows, r)
			}
			return show(rows, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "PRODUCT\tLAST\tCHG%")
				for _, r := range rows {
					if r.Quote == nil {
						fmt.Fprintf(w, "%s\t%s\t\n", r.Name, "n/a")
						continue
					}
					fmt.Fprintf(w, "%s\t%.2f\t%s\n", r.Name, r.Quote.Last, pctStr(r.Quote.ChangePct))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func macroCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "macro",
		Short: "Macro backdrop",
	}
	c.AddCommand(macroBriefCmd())
	return c
}

func macroBriefCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "brief",
		Short: "Rates, curve, Fed odds, and risk in one screen (keyless)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			brief := map[string]any{}

			curve, cErr := provider.TreasuryCurve(ctx, hx)
			if cErr == nil {
				brief["rates"] = curve
			}
			odds, oErr := provider.FedOdds(ctx, hx)
			if oErr == nil {
				brief["fedOdds"] = odds
			}
			risk := map[string]*model.Quote{}
			for name, sym := range map[string]string{"VIX": "^VIX", "DXY": "DX-Y.NYB", "WTI": "CL=F", "Gold": "GC=F"} {
				if q, err := provider.YahooQuote(ctx, hx, sym); err == nil {
					risk[name] = q
				}
			}
			if btc, err := provider.Crypto(ctx, hx, []string{"bitcoin"}, "usd"); err == nil && len(btc) > 0 {
				brief["btc"] = btc[0]
			}
			brief["risk"] = risk

			return show(brief, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "RATES")
				if cErr != nil {
					fmt.Fprintf(w, "  (rates unavailable: %v)\n", cErr)
				} else {
					for _, p := range curve {
						if p.Tenor == "3M" || p.Tenor == "2Y" || p.Tenor == "10Y" || p.Tenor == "30Y" {
							fmt.Fprintf(w, "  %s\t%.2f%%\n", p.Tenor, p.Yield)
						}
					}
					if v, ok := spreadOf(curve, "10Y", "2Y"); ok {
						fmt.Fprintf(w, "  2s10s\t%+.0fbp\n", v)
					}
				}
				fmt.Fprintln(w, "FED ODDS")
				if oErr != nil {
					fmt.Fprintf(w, "  (unavailable: %v)\n", oErr)
				} else {
					fmt.Fprintf(w, "  meeting\t%s\n", odds.Meeting)
					for i, k := range odds.Buckets {
						if i >= 3 || k.Prob < 0.01 {
							break
						}
						fmt.Fprintf(w, "  %s\t%.0f%%\n", k.Band, k.Prob*100)
					}
				}
				fmt.Fprintln(w, "RISK")
				for _, name := range []string{"VIX", "DXY", "WTI", "Gold"} {
					if q := risk[name]; q != nil {
						fmt.Fprintf(w, "  %s\t%.2f\t%s\n", name, q.Last, pctStr(q.ChangePct))
					}
				}
				if btc, ok := brief["btc"].(model.CryptoPrice); ok {
					fmt.Fprintf(w, "  BTC\t%s\t%s\n", money(btc.Price), pctStr(btc.Change24))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}
