package cli

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/laurenschristian/finctl/internal/model"
	"github.com/laurenschristian/finctl/internal/provider"
)

func mcpCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run as an MCP server over stdio (for Claude, Cursor, etc.)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return mcpServer().Run(cmd.Context(), &mcp.StdioTransport{})
		},
	}
}

type rawOut struct {
	Data any `json:"data,omitempty"`
}

func wrap(v any, err error) (*mcp.CallToolResult, rawOut, error) {
	if err != nil {
		return nil, rawOut{}, err
	}
	return nil, rawOut{Data: v}, nil
}

type symbolsArg struct {
	Symbols []string `json:"symbols"`
}
type symbolArg struct {
	Symbol string `json:"symbol"`
}
type chartArg struct {
	Symbol   string `json:"symbol"`
	Range    string `json:"range,omitempty"`
	Interval string `json:"interval,omitempty"`
}
type cryptoArg struct {
	Coins []string `json:"coins"`
	VS    string   `json:"vs,omitempty"`
}
type fxArg struct {
	From string   `json:"from,omitempty"`
	To   []string `json:"to"`
}
type capexArg struct {
	Tickers  []string `json:"tickers,omitempty"`
	Quarters int      `json:"quarters,omitempty"`
}

func mcpServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "finctl", Version: Version}, nil)

	mcp.AddTool(s, &mcp.Tool{Name: "fin_quote", Description: "Delayed quotes for symbols (Cboe, Yahoo fallback)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in symbolsArg) (*mcp.CallToolResult, rawOut, error) {
			var out []any
			for _, sym := range in.Symbols {
				q, err := provider.CboeQuote(ctx, hx, sym)
				if err != nil || q.Last == 0 {
					q, err = provider.YahooQuote(ctx, hx, sym)
					if err != nil {
						return nil, rawOut{}, err
					}
				}
				out = append(out, q)
			}
			return wrap(out, nil)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_chart", Description: "OHLCV price history for a symbol (Yahoo). range e.g. 6m, interval e.g. 1d."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in chartArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.YahooChart(ctx, hx, in.Symbol, orElse(in.Range, "6m"), orElse(in.Interval, "1d")))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_fund", Description: "Revenue, capex, and shares-outstanding trend from SEC XBRL."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in symbolArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.Fundamentals(ctx, hx, in.Symbol))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_capex", Description: "Hyperscaler quarterly capex with YoY (SEC XBRL). Omit tickers for MSFT/GOOGL/AMZN/META/ORCL."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in capexArg) (*mcp.CallToolResult, rawOut, error) {
			names := in.Tickers
			if len(names) == 0 {
				names = hyperscalers
			}
			return wrap(provider.HyperscalerCapex(ctx, hx, names, in.Quarters))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_crypto", Description: "Crypto spot prices (CoinGecko). coins are ids e.g. bitcoin, ethereum."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in cryptoArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.Crypto(ctx, hx, in.Coins, in.VS))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_fx", Description: "Spot FX rates from a base currency to targets (ECB via Frankfurter)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in fxArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.FX(ctx, hx, orElse(in.From, "USD"), in.To))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_rates", Description: "Treasury par yield curve and curve spreads (keyless)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.TreasuryCurve(ctx, hx))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_fedodds", Description: "Market-implied FOMC target-rate odds for the next meeting (Kalshi, keyless)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.FedOdds(ctx, hx))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_cot", Description: "CFTC Commitments of Traders net non-commercial positioning. market e.g. ES, NQ, GC, CL."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in cotArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.Cot(ctx, hx, orElse(in.Market, "ES")))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_series", Description: "Last N observations of a FRED series (needs FINCTL_FRED_KEY). id e.g. DGS10, CPIAUCSL."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in seriesArg) (*mcp.CallToolResult, rawOut, error) {
			key, _ := cfg.Key("fred")
			n := in.N
			if n == 0 {
				n = 12
			}
			return wrap(provider.FredSeries(ctx, hx, key, in.ID, n))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_fiscal", Description: "US debt to the penny (Treasury FiscalData, keyless)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.TreasuryDebt(ctx, hx))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_energy", Description: "Energy futures: WTI, Brent, natural gas, gasoline (Yahoo, keyless)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, rawOut, error) {
			out := map[string]*model.Quote{}
			for name, sym := range map[string]string{"wti": "CL=F", "brent": "BZ=F", "natgas": "NG=F", "gasoline": "RB=F"} {
				if q, err := provider.YahooQuote(ctx, hx, sym); err == nil {
					out[name] = q
				}
			}
			return wrap(out, nil)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_tw_revenue", Description: "TWSE monthly revenue with MoM/YoY. ids e.g. 2330 (TSMC); omit for the AI-server ODM basket."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in idsArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.TWRevenue(ctx, hx, in.IDs))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_gpu_rent", Description: "Lowest ask and median $/GPU-hour on vast.ai. gpus e.g. \"H100 SXM\"; omit for H100 SXM + B200."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in gpusArg) (*mcp.CallToolResult, rawOut, error) {
			gpus := in.GPUs
			if len(gpus) == 0 {
				gpus = provider.GPUDefaults
			}
			var out []any
			for _, g := range gpus {
				r, err := provider.GPURent(ctx, hx, g)
				if err != nil {
					return nil, rawOut{}, err
				}
				out = append(out, r)
			}
			return wrap(out, nil)
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_short", Description: "FINRA consolidated short interest, days-to-cover, change for a ticker."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in symbolArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.ShortInterest(ctx, hx, in.Symbol))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_options", Description: "Delayed Cboe option chain summary: 30d IV, put/call, max pain, OI walls."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in symbolArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.OptionsChain(ctx, hx, in.Symbol))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_filings", Description: "Recent SEC filings for a ticker. form filters to 8-K/10-K/10-Q/4."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in filingsArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.Filings(ctx, hx, in.Symbol, in.Form, in.N))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_insider", Description: "Recent Form 4 insider filings for a ticker (SEC submissions)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in symbolArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.Insider(ctx, hx, in.Symbol, 20))
		})
	mcp.AddTool(s, &mcp.Tool{Name: "fin_dilution", Description: "Shares-outstanding trend and dilution flag for a ticker (SEC XBRL)."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in symbolArg) (*mcp.CallToolResult, rawOut, error) {
			return wrap(provider.Dilution(ctx, hx, in.Symbol))
		})
	return s
}

type idsArg struct {
	IDs []string `json:"ids,omitempty"`
}
type filingsArg struct {
	Symbol string `json:"symbol"`
	Form   string `json:"form,omitempty"`
	N      int    `json:"n,omitempty"`
}
type gpusArg struct {
	GPUs []string `json:"gpus,omitempty"`
}

type cotArg struct {
	Market string `json:"market,omitempty"`
}
type seriesArg struct {
	ID string `json:"id"`
	N  int    `json:"n,omitempty"`
}

func orElse(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
