package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/finctl/internal/provider"
)

func fundCmd() *cobra.Command {
	var quarters int
	c := &cobra.Command{
		Use:   "fund <symbol>",
		Short: "Revenue, capex, and shares-outstanding trend from SEC XBRL",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := provider.Fundamentals(cmd.Context(), hx, args[0])
			if err != nil {
				return err
			}
			return show(f, func() string {
				b, w := newTab()
				fmt.Fprintf(w, "%s (%s)\n", f.Symbol, f.CIK)
				fmt.Fprintln(w, "PERIOD\tREVENUE\tCAPEX\tSHARES OUT")
				periods := f.Periods
				if quarters > 0 && len(periods) > quarters {
					periods = periods[len(periods)-quarters:]
				}
				for _, p := range periods {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Fiscal, abbr(p.Revenue), abbr(p.Capex), abbr(p.SharesOut))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().IntVar(&quarters, "quarters", 8, "how many recent periods to show (0 = all)")
	return c
}

var hyperscalers = []string{"MSFT", "GOOGL", "AMZN", "META", "ORCL"}

func capexCmd() *cobra.Command {
	var quarters int
	var tickers []string
	c := &cobra.Command{
		Use:   "capex",
		Short: "Hyperscaler quarterly capex with YoY (SEC XBRL)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			names := tickers
			if len(names) == 0 {
				names = hyperscalers
			}
			points, err := provider.HyperscalerCapex(cmd.Context(), hx, names, quarters)
			if err != nil {
				return err
			}
			return show(points, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "COMPANY\tPERIOD\tCAPEX\tYOY")
				for _, p := range points {
					yoy := ""
					if p.YoY != 0 {
						yoy = fmt.Sprintf("%+.1f%%", p.YoY)
					}
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Company, p.Fiscal, abbr(p.Capex), yoy)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().IntVar(&quarters, "quarters", 6, "recent quarters per company")
	c.Flags().StringSliceVar(&tickers, "tickers", nil, "override the hyperscaler set (comma-separated)")
	return c
}
