# finctl: one finance CLI + MCP that replaces the invest/serenity/finnhub patchwork

Status: planning (2026-09-13). Build after eero + dsmctl + ibkrctl, or before if it hurts more.
Recipe: same as adgctl (Go 1.27, cobra, official MCP go-sdk, goreleaser, MIT, public). One binary, `--json` everywhere, `finctl mcp`.

## What it replaces

| Today | Tomorrow |
| --- | --- |
| `/invest` skill: 300 lines of prose telling the agent which MCP to call and which vault file to edit | `finctl port review`, `finctl watchlist`, `finctl research NVDA`, `finctl daily`. The skill shrinks to "run these, then write the note". |
| `serenity` skill: 15-point checklist run from memory + a tweet archive refreshed every 30 min | `finctl lens NVDA`: the checklist items that are data (dilution, GAAP vs non-GAAP margin, customer concentration, short interest, IV) come computed. The archive stays a reference doc. |
| Finnhub MCP + IBKR MCP + WebSearch for prices | Providers behind one interface with a SQLite cache. Finnhub stays as a provider, not a separate tool. |
| financial-advisor agent pulling Monarch + IBKR by hand | `finctl networth` (Monarch + IBKR + crypto) feeds it. |

Vault stays the source of truth for targets, theses, notes. finctl reads the Dashboard target table and the watchlist note; it writes nothing to the vault by default (`--write` flag later).

## Commands (v0.1 = marked *)

Portfolio
- `port` *: positions, P&L, allocation vs target (targets parsed from vault Dashboard table), drift flags, cash. Needs ibkrctl or the gateway on :5001.
- `port review` *: same plus earnings-this-week for holdings, insider/short changes since last review, and a paste-ready review block.
- `watchlist` *: vault watchlist with live price, distance to buy zone, earnings date, IV rank.
- `networth`: Monarch + IBKR + crypto rollup.

Research
- `quote T...` *, `chart T --range 6m` (ASCII sparkline)
- `fund T` *: revenue/GM/opex/FCF/shares-out trend from SEC XBRL (free, filing-accurate), Finnhub metrics as overlay.
- `research T` *: one-screen dossier = quote, fund, next earnings, insider (Form 4, 90d), short interest + days-to-cover, options (IV, put/call, max pain), StockTwits sentiment, news, 10-K risk-factor customer-concentration snippet.
- `lens T` *: Serenity checklist with the computable items filled in and the manual ones listed.
- `compare T1 T2 --metric`
- `earnings --week | T` *
- `insider T | --clusters` * (EDGAR Form 4 feed + OpenInsider cluster buys)
- `short T` * (FINRA bi-monthly), `options T` (Cboe delayed chain: IV, put/call, OI walls)
- `filings T --form 8-K`, `edgar search "indium phosphide"` (full-text; needs UA header, 403 on naked curl, works with proper UA)
- `congress --recent` (House/Senate PTR disclosures)

Macro
- `macro brief` *: fed funds, 2y/10y/30y + 2s10s, latest CPI/PPI/NFP/JOLTS/GDPNow prints with dates, VIX, DXY, oil, gold, BTC, Fed-cut odds from Kalshi + Polymarket + CME.
- `rates` *, `curve [--date]`, `series FRED_ID` * (any FRED series), `calendar --week` *
- `fedodds` * (Kalshi KXFED + Polymarket), `cot --market ES|NQ|GC|CL`, `sentiment` (Michigan, AAII if reachable, put/call, VIX term)
- `energy` (EIA WTI/Brent/HH, rig count)

Edge (the stuff the Serenity lens actually keys off)
- `capex` *: hyperscaler capex per quarter straight from XBRL (MSFT, GOOGL, AMZN, META, ORCL) + YoY. Master-thesis input, free, filing-accurate.
- `korea-exports`: first-20-days semiconductor exports (Korea Customs). Behind a JS portal; scrape or use Bloomberg-free mirror. TBD.
- `tsmc-revenue`: monthly revenue (investor.tsmc.com is Cloudflare-gated to curl; fetch with browser UA or from TWSE MOPS).
- `memory-prices`: DRAM/NAND spot (TrendForce free tables). TBD.
- `gpu-rent`: H100/B200 hourly rental index (public price pages). TBD.
- `dilution T` *: shares outstanding trend + ATM/shelf 8-K/424B detection from EDGAR.
- `margins T` *: GAAP vs non-GAAP gap from XBRL + press-release 8-K.

Ops
- `daily` *: morning brief = port + watchlist buy-zones + earnings this week + macro prints today + fedodds + insider clusters in holdings. One command, one screen, `--json` for the agent.
- `cache clear`, `keys` (which providers are configured), `mcp` *.

## Architecture
- `internal/provider/<name>`: fred, bls, treasury, edgar (xbrl + full-text + form4 atom), cboe, finra, cftc, finnhub, yahoo (chart only; options need crumb), stocktwits, kalshi, polymarket, coingecko, frankfurter, eia, monarch (via existing MCP? no: Monarch GraphQL with token), ibkr (gateway :5001).
- `internal/model`: normalized types (Quote, Fundamentals, InsiderTx, ShortInterest, OptionsSummary, MacroPrint, Position).
- `internal/cache`: SQLite (modernc, no cgo). TTL per provider. Rate-limit guard per provider.
- `internal/vault`: read-only parsers for Dashboard targets table + watchlist note + tracker CSV.
- `internal/lens`: Serenity checklist scoring from model data.
- `internal/cli`, `internal/mcp`: thin.
- Keys via config `password_cmd` pattern (Keychain), env `FINCTL_FRED_KEY` etc.

## Keys to obtain (all free)
FRED (fred.stlouisfed.org/docs/api/api_key.html), EIA (eia.gov/opendata), BLS v2 (higher quota), Finnhub (have). Everything else in docs/data-sources.md is keyless.

## Not in scope
Backtesting, live streaming, order placement (ibkrctl owns that), anything paid.

## Build order
1. providers: edgar (xbrl, form4), treasury+fed h15, fred, cboe, finra, kalshi/polymarket, finnhub, yahoo chart, stocktwits + cache
2. commands: quote, fund, series, rates, macro brief, fedodds, insider, short, earnings, capex
3. vault parsers + port/watchlist (needs gateway)
4. research, lens, daily
5. mcp, release, rewrite /invest skill to call finctl
