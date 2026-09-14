package cli

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

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
	return s
}

func orElse(v, d string) string {
	if v == "" {
		return d
	}
	return v
}
