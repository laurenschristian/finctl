# finctl PRD 02: Research commands

Status: draft, 2026-09-14. Depends on PRD 00 (arch), PRD 01 (providers). Providers: edgar, cboe, yahoo, finnhub, finra, cftc, openinsider, disclosedcapitol, stocktwits.

## Purpose

Single-command answers to "what is this stock" so the agent (and I) stop stitching five MCPs per question. A ticker in, a normalized dossier out, cached.

## Commands

- `quote <T...>` *: last, change, day range, volume, market cap, 52w range. Provider: cboe (quote + VIX for `^VIX`), yahoo fallback. `--json` returns `model.Quote[]`.
- `chart <T> --range 6m --interval 1d` *: ASCII/unicode sparkline of close, plus first/last/%chg (reuse the sparkline from ibkrctl). Provider: yahoo.
- `fund <T>` *: revenue, gross margin, opex, FCF, shares-out trend (last 8 quarters + last 3 FY) from SEC XBRL companyfacts; finnhub metrics as overlay. Provider: edgar (source of truth), finnhub (overlay). Table by quarter; `--json` = `model.Fundamentals`.
- `research <T>` *: one-screen dossier = quote + fund summary + next earnings (finnhub) + insider (Form 4, 90d) + short interest + days-to-cover + options summary (IV, put/call, max pain from cboe chain) + StockTwits sentiment + news + a 10-K risk-factor customer-concentration snippet (edgar fts). Composes the other commands; one cached object.
- `lens <T>` *: the Serenity checklist with the computable items filled in (dilution, GAAP vs non-GAAP margin gap, customer concentration, short interest, IV rank) and the manual items listed. Provider: edgar + cboe + finra. See `internal/lens`.
- `compare <T1> <T2> --metric revenue|gm|fcf|short`: side-by-side of one metric across names.
- `earnings --week | <T>` *: earnings this week (finnhub calendar) or next date for a ticker.
- `insider <T> | --clusters` *: Form 4 transactions for a ticker (edgar), or cluster buys screen (openinsider). `--json` = `model.InsiderTx[]`.
- `short <T>` *: FINRA consolidated short interest, current vs previous, change%, days-to-cover. `--json` = `model.ShortInterest`.
- `options <T>`: cboe delayed chain summary: IV (front + skew), put/call ratio, open-interest walls, max pain. `--json` = `model.OptionsSummary`.
- `filings <T> --form 8-K|10-K|424B`: recent filings list (edgar submissions feed).
- `edgar search "<phrase>" --form 10-K`: full-text search across filings (edgar fts, UA required).
- `congress <T> | --recent` *: DisclosedCapitol trades per ticker or latest, with 30d alpha per trade.
- `dilution <T>` *: shares-outstanding trend from XBRL + ATM/shelf detection (8-K/424B via edgar fts). `--json` = shares series + flags.
- `margins <T>` *: GAAP vs non-GAAP gap from XBRL + press-release 8-K. Serenity input.

## Data model (internal/model)

`Quote{Symbol, Last, Change, ChangePct, DayLow, DayHigh, Volume, MarketCap, Week52Low, Week52High, Time}`. `Fundamentals{Symbol, Periods []Period}` where `Period{FiscalPeriod, Revenue, GrossMargin, Opex, FCF, SharesOut, GAAPNetMargin, NonGAAPNetMargin}`. `InsiderTx{Filer, Role, Date, Type, Shares, Price, Value}`. `ShortInterest{Symbol, SettlementDate, Current, Previous, ChangePct, DaysToCover}`. `OptionsSummary{Symbol, FrontIV, IVSkew, PutCallRatio, MaxPain, OIWalls []Strike}`.

## Output

Tables by default (reuse ibkrctl's tabwriter + number/money/sparkline helpers). `--json` emits the model type. `research` prints sectioned blocks (Quote / Fundamentals / Earnings / Insider / Short / Options / Sentiment / Risk) like ibkrctl's `review`.

## MCP tools

`fin_quote`, `fin_fund`, `fin_research`, `fin_lens`, `fin_earnings`, `fin_insider`, `fin_short`, `fin_options`, `fin_congress`, `fin_dilution`, `fin_margins`. All read-only, all wrap the same command functions, all return the `--json` model.

## Edge cases

- Ticker to CIK: cache `company_tickers.json` (7d); a miss is a clear "unknown ticker" error, not a crash.
- XBRL tag drift: companies report capex under `PaymentsToAcquirePropertyPlantAndEquipment` mostly, but some use `PaymentsToAcquireProductiveAssets`; the edgar provider tries a tag list and records which matched (study Equibles' normalization).
- cboe options chain for a large name is multi-MB; cache 60s and stream-parse only the fields the summary needs.
- SEC 10 req/s hard cap: `research`/`lens` fan several edgar calls; the httpx token bucket serializes them under the cap.

## Acceptance criteria

1. `finctl fund NVDA` shows 8 quarters of revenue/GM/FCF/shares from XBRL within one call after warm cache; `--json` matches `model.Fundamentals`.
2. `finctl research NVDA` returns every section or a per-section error string (never a total failure because one provider is down), like ibkrctl `review`.
3. `finctl short NVDA` matches the latest FINRA settlement date.
4. Each provider has one Go test against a saved fixture (no network in tests).

## Known limitation (v0.1) and the v0.2 fix

`fund` v0.1 uses SEC "frames" (facts SEC tags with a calendar-quarter `CYyyyyQn`). Companies whose fiscal year aligns to calendar quarters (MSFT, GOOGL, META) render a clean revenue/capex/shares trend. Companies with off-calendar fiscal years (NVDA Jan, AAPL Sep) rarely get frame tags, so `fund` shows few or no rows for them (honest, never stale). `capex` is unaffected for the hyperscalers because they report on calendar-ish quarters.

v0.2 first task: reimplement `Fundamentals` on `companyfacts` (all facts for a CIK in one call), selecting quarterly duration facts by `form` (10-Q/10-K) and `fp`/`fy` and deduping by period end (latest `filed` wins), instead of relying on SEC frames. This fills off-calendar names. The provider signature and the `fund` command stay the same; only the internals change.
