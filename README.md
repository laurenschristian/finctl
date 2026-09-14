# finctl

<img src="assets/icon.png" alt="finctl icon" width="96" align="right">

[![CI](https://github.com/laurenschristian/finctl/actions/workflows/ci.yml/badge.svg)](https://github.com/laurenschristian/finctl/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

One static binary: a CLI and an [MCP](https://modelcontextprotocol.io) server for free market, macro and research data. No cgo, no paid keys for the core.

```console
finctl quote AAPL MSFT           # delayed quote, change, day range (Cboe, Yahoo fallback)
finctl chart NVDA --range 6m     # price history + a sparkline
finctl fund MSFT                 # revenue, capex, shares by fiscal period (SEC XBRL)
finctl capex --quarters 4        # hyperscaler capex, YoY (MSFT/GOOGL/AMZN/META/ORCL)
finctl crypto bitcoin ethereum   # crypto spot (CoinGecko; args are coin ids)
finctl fx USD                    # spot FX from a base (ECB via Frankfurter)

finctl macro brief               # rates, Fed odds, and risk on one screen
finctl rates                     # 3m/2y/10y/30y + 2s10s, 3m10y (Treasury, keyless)
finctl curve                     # full Treasury par yield curve
finctl fedodds                   # market-implied FOMC target-rate odds (Kalshi)
finctl cot --market ES           # CFTC net non-commercial positioning
finctl energy                    # WTI, Brent, natural gas, gasoline (Yahoo)
finctl fiscal                    # US debt to the penny (Treasury FiscalData)
finctl series DGS10 --n 6        # any FRED series (needs FINCTL_FRED_KEY)

finctl tw-revenue                # TWSE monthly revenue, AI-server ODM basket (MoM/YoY)
finctl gpu-rent --trend          # vast.ai H100/B200 $/hr low + median, stored over time

finctl cache clear               # drop the on-disk cache
finctl doctor                    # config, cache, provider reachability
finctl mcp                       # MCP server over stdio
```

Add `--json` to any command for machine-readable output.

## Install

```console
brew install laurenschristian/tap/finctl
go install github.com/laurenschristian/finctl@latest
```

Or grab a binary from [Releases](https://github.com/laurenschristian/finctl/releases).

## SEC contact User-Agent (required for `fund` and `capex`)

SEC EDGAR rejects requests without a contact in the User-Agent (HTTP 403). Set one once:

```console
export FINCTL_USER_AGENT="finctl/0.1 (you@example.com)"
```

Or put `user_agent: finctl/0.1 (you@example.com)` in the config file. The keyless market commands (`quote`, `chart`, `crypto`, `fx`) work without it.

> `fund` reads SEC XBRL companyfacts and works for off-calendar fiscal-year filers (NVDA, AAPL) too. Rows are labeled by the calendar quarter the fiscal period ended in (CY2026Q2). Cash-flow items (capex) are de-cumulated from year-to-date filings. The fiscal-Q4 quarter has no separate quarterly filing (only the annual 10-K), so it appears as a gap, by design.

## Configure

Precedence is flags, then environment (`FINCTL_USER_AGENT`, `FINCTL_CACHE_DIR`, `FINCTL_CONFIG`), then the config file
(`~/Library/Application Support/finctl/config.yaml` on macOS, `~/.config/finctl/config.yaml` on Linux).
Provider keys are optional and off the keyless path; set them later with `FINCTL_<PROVIDER>_KEY` or a `*_key_cmd` that prints the secret from a keychain, `op read`, `pass` or sops.

## MCP

```console
claude mcp add fin -- finctl mcp
```

Fourteen tools: `fin_quote`, `fin_chart`, `fin_fund`, `fin_capex`, `fin_crypto`, `fin_fx`, `fin_rates`, `fin_fedodds`, `fin_cot`, `fin_series`, `fin_fiscal`, `fin_energy`, `fin_tw_revenue`, `fin_gpu_rent`. Any stdio MCP client (Cursor, Claude Desktop, Zed) works the same: command `finctl`, args `["mcp"]`.

## Data sources

- Quotes and charts: Cboe delayed, Yahoo fallback
- Fundamentals and capex: SEC EDGAR XBRL (needs a contact User-Agent)
- Crypto: CoinGecko
- FX: Frankfurter (ECB reference rates)
- Rates and curve: US Treasury par yield curve XML
- Fed odds: Kalshi KXFED prediction markets
- Positioning: CFTC Commitments of Traders (Socrata)
- Fiscal: Treasury FiscalData (debt to the penny)
- Macro series: FRED (free key via `FINCTL_FRED_KEY`)
- Taiwan ODM revenue: TWSE OpenAPI monthly revenue
- GPU rental spot: vast.ai on-demand offers

Responses are cached on disk (pure-Go SQLite) with per-provider TTLs and rate limits. See [docs/data-sources.md](docs/data-sources.md) for the full catalog and [PLAN.md](PLAN.md) for the roadmap (macro, filings, portfolio, research).

## Development

```console
make hooks    # gofmt, dash check, gitleaks, build, lint on commit; tests + coverage floor on push
make test
make cover    # floor in scripts/coverage.sh
make lint     # golangci-lint, config in .golangci.yml
make sec      # gosec
make docs     # regenerate man/ and docs/cli/
```

See [CONTRIBUTING.md](CONTRIBUTING.md). MIT.
