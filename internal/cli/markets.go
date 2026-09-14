package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
)

func quoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "quote <symbol> [symbol...]",
		Short: "Delayed quote (Cboe, Yahoo fallback)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var quotes []model.Quote
			for _, sym := range args {
				q, err := provider.CboeQuote(ctx, hx, sym)
				if err != nil || q.Last == 0 {
					q, err = provider.YahooQuote(ctx, hx, sym)
					if err != nil {
						return err
					}
				}
				quotes = append(quotes, *q)
			}
			return show(quotes, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "SYMBOL\tLAST\tCHG\tCHG%\tDAY LOW\tDAY HIGH\tVOLUME")
				for _, q := range quotes {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						q.Symbol, money(q.Last), money(q.Change), pctStr(q.ChangePct),
						money(q.DayLow), money(q.DayHigh), abbr(q.Volume))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
}

func chartCmd() *cobra.Command {
	var rng, interval string
	c := &cobra.Command{
		Use:   "chart <symbol>",
		Short: "Price history with a sparkline (Yahoo)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ch, err := provider.YahooChart(cmd.Context(), hx, args[0], rng, interval)
			if err != nil {
				return err
			}
			return show(ch, func() string {
				if len(ch.Bars) == 0 {
					return "no data\n"
				}
				closes := make([]float64, len(ch.Bars))
				for i, bar := range ch.Bars {
					closes[i] = bar.Close
				}
				first, last := closes[0], closes[len(closes)-1]
				chg := 0.0
				if first != 0 {
					chg = (last - first) / first * 100
				}
				return fmt.Sprintf("%s  %s  %s -> %s  %+.2f%%\n%s\n",
					ch.Symbol, rng, money(first), money(last), chg, sparkline(closes))
			})
		},
	}
	c.Flags().StringVar(&rng, "range", "6m", "range: 1d,5d,1m,3m,6m,1y,2y,5y,ytd,max")
	c.Flags().StringVar(&interval, "interval", "1d", "bar interval: 1d,1wk,1mo")
	return c
}

func cryptoCmd() *cobra.Command {
	var vs string
	c := &cobra.Command{
		Use:   "crypto <coin> [coin...]",
		Short: "Crypto spot prices (CoinGecko); coins are ids e.g. bitcoin ethereum",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ps, err := provider.Crypto(cmd.Context(), hx, args, vs)
			if err != nil {
				return err
			}
			return show(ps, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "COIN\tPRICE\t24H%")
				for _, p := range ps {
					fmt.Fprintf(w, "%s\t%s %s\t%s\n", p.Coin, money(p.Price), strings.ToUpper(p.VS), pctStr(p.Change24))
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().StringVar(&vs, "vs", "usd", "quote currency")
	return c
}

func fxCmd() *cobra.Command {
	var from string
	c := &cobra.Command{
		Use:   "fx <currency> [currency...]",
		Short: "Spot FX rates from a base (ECB via Frankfurter)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rates, err := provider.FX(cmd.Context(), hx, from, args)
			if err != nil {
				return err
			}
			return show(rates, func() string {
				b, w := newTab()
				fmt.Fprintln(w, "PAIR\tRATE\tDATE")
				for _, r := range rates {
					fmt.Fprintf(w, "%s/%s\t%s\t%s\n", r.From, r.To, money(r.Rate), r.Date)
				}
				_ = w.Flush()
				return b.String()
			})
		},
	}
	c.Flags().StringVar(&from, "from", "USD", "base currency")
	return c
}
