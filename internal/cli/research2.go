package cli

import (
	"fmt"

	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
	"github.com/spf13/cobra"
)

func shortCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "short <ticker>",
		Short: "FINRA consolidated short interest, days-to-cover, change (keyless)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			si, err := provider.ShortInterest(cmd.Context(), hx, args[0])
			if err != nil {
				return err
			}
			return show(si, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "%s  (settlement %s)\n", si.Symbol, si.SettlementDate)
				fmt.Fprintf(w, "short interest\t%s\n", abbr(si.Current))
				fmt.Fprintf(w, "previous\t%s\t(%s)\n", abbr(si.Previous), pctStr(si.ChangePct))
				fmt.Fprintf(w, "days to cover\t%.2f\n", si.DaysToCover)
				fmt.Fprintf(w, "avg daily vol\t%s\n", abbr(si.AvgDailyVol))
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func optionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "options <ticker>",
		Short: "Delayed Cboe option chain summary: IV, put/call, max pain, OI walls",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := provider.OptionsChain(cmd.Context(), hx, args[0])
			if err != nil {
				return err
			}
			return show(o, func() string { return renderOptions(o) })
		},
	}
}

func renderOptions(o *model.OptionsSummary) string {
	b, w := newTab()
	fmt.Fprintf(w, "%s  underlying %.2f  (%d expiries)\n", o.Symbol, o.Underlying, o.Expiries)
	fmt.Fprintf(w, "30d IV\t%.1f%%\n", o.FrontIV)
	fmt.Fprintf(w, "put/call OI\t%.2f\n", o.PutCallRatio)
	fmt.Fprintf(w, "max pain\t%.2f\n", o.MaxPain)
	if len(o.OIWalls) > 0 {
		fmt.Fprintln(w, "OI walls:")
		for _, s := range o.OIWalls {
			fmt.Fprintf(w, "  %.2f\t%s\n", s.Strike, abbr(s.OI))
		}
	}
	_ = w.Flush()
	return b.String()
}

func filingsCmd() *cobra.Command {
	var form string
	var n int
	c := &cobra.Command{
		Use:   "filings <ticker>",
		Short: "Recent SEC filings (submissions feed). --form 8-K|10-K|10-Q|4",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs, err := provider.Filings(cmd.Context(), hx, args[0], form, n)
			if err != nil {
				return err
			}
			return show(fs, func() string { return renderFilings(fs) })
		},
	}
	c.Flags().StringVar(&form, "form", "", "filter to a form type")
	c.Flags().IntVar(&n, "n", 20, "max filings")
	return c
}

func insiderCmd() *cobra.Command {
	var n int
	c := &cobra.Command{
		Use:   "insider <ticker>",
		Short: "Recent Form 4 insider filings (SEC submissions)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fs, err := provider.Insider(cmd.Context(), hx, args[0], n)
			if err != nil {
				return err
			}
			return show(fs, func() string { return renderFilings(fs) })
		},
	}
	c.Flags().IntVar(&n, "n", 20, "max filings")
	return c
}

func renderFilings(fs []model.Filing) string {
	b, w := newTab()
	fmt.Fprintln(w, "FORM\tFILED\tDESCRIPTION")
	for _, f := range fs {
		d := f.Desc
		if len(d) > 44 {
			d = d[:44]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", f.Form, f.FilingDate, d)
	}
	_ = w.Flush()
	return b.String()
}

func dilutionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dilution <ticker>",
		Short: "Shares-outstanding trend and dilution flag (SEC XBRL)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := provider.Dilution(cmd.Context(), hx, args[0])
			if err != nil {
				return err
			}
			return show(d, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "%s  shares outstanding (%s, %s YoY)\n", d.Symbol, d.Flag, pctStr(d.ChangePctYr))
				fmt.Fprintln(w, "PERIOD\tSHARES")
				for _, p := range d.Points {
					fmt.Fprintf(w, "%s\t%s\n", p.Date, abbr(p.Value))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func researchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "research <ticker>",
		Short: "One-screen dossier: quote, fundamentals, short, options, insider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			t := args[0]
			d := map[string]any{"symbol": t}

			q, qErr := provider.CboeQuote(ctx, hx, t)
			if qErr != nil || q == nil || q.Last == 0 {
				q, qErr = provider.YahooQuote(ctx, hx, t)
			}
			if qErr == nil {
				d["quote"] = q
			}
			f, fErr := provider.Fundamentals(ctx, hx, t)
			if fErr == nil {
				d["fundamentals"] = f
			}
			si, sErr := provider.ShortInterest(ctx, hx, t)
			if sErr == nil {
				d["short"] = si
			}
			op, oErr := provider.OptionsChain(ctx, hx, t)
			if oErr == nil {
				d["options"] = op
			}
			ins, iErr := provider.Insider(ctx, hx, t, 5)
			if iErr == nil {
				d["insider"] = ins
			}
			return show(d, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "== %s ==\n", t)
				if qErr == nil {
					fmt.Fprintf(w, "quote\t%.2f\t%s\n", q.Last, pctStr(q.ChangePct))
				} else {
					fmt.Fprintf(w, "quote\t(%v)\n", qErr)
				}
				if fErr == nil && len(f.Periods) > 0 {
					last := f.Periods[len(f.Periods)-1]
					fmt.Fprintf(w, "revenue (%s)\t%s\n", last.Fiscal, abbr(last.Revenue))
					if last.Capex > 0 {
						fmt.Fprintf(w, "capex\t%s\n", abbr(last.Capex))
					}
				}
				if sErr == nil {
					fmt.Fprintf(w, "short interest\t%s\tdtc %.1f\n", abbr(si.Current), si.DaysToCover)
				}
				if oErr == nil {
					fmt.Fprintf(w, "30d IV\t%.1f%%\tp/c %.2f\tmax pain %.0f\n", op.FrontIV, op.PutCallRatio, op.MaxPain)
				}
				if iErr == nil {
					fmt.Fprintf(w, "insider\t%d recent Form 4\n", len(ins))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func lensCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lens <ticker>",
		Short: "Serenity checklist with the computable items filled in",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			t := args[0]
			lens := map[string]any{"symbol": t}
			if d, err := provider.Dilution(ctx, hx, t); err == nil {
				lens["dilution"] = d
			}
			if si, err := provider.ShortInterest(ctx, hx, t); err == nil {
				lens["short"] = si
			}
			if op, err := provider.OptionsChain(ctx, hx, t); err == nil {
				lens["options"] = op
			}
			return show(lens, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "Serenity lens: %s\n", t)
				if d, ok := lens["dilution"].(*model.DilutionReport); ok {
					fmt.Fprintf(w, "[auto] dilution\t%s (%s YoY)\n", d.Flag, pctStr(d.ChangePctYr))
				}
				if si, ok := lens["short"].(*model.ShortInterest); ok {
					fmt.Fprintf(w, "[auto] short / days-to-cover\t%s / %.1f\n", abbr(si.Current), si.DaysToCover)
				}
				if op, ok := lens["options"].(*model.OptionsSummary); ok {
					fmt.Fprintf(w, "[auto] 30d IV / put-call\t%.1f%% / %.2f\n", op.FrontIV, op.PutCallRatio)
				}
				fmt.Fprintln(w, "[manual] moat and switching costs")
				fmt.Fprintln(w, "[manual] customer concentration (read the 10-K risk factors)")
				fmt.Fprintln(w, "[manual] GAAP vs non-GAAP quality")
				fmt.Fprintln(w, "[manual] insider alignment and recent selling")
				_ = w.Flush()
				return b.String()
			})
		},
	}
}
