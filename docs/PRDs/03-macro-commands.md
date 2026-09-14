# finctl PRD 03: Macro commands

Status: SHIPPED v0.3.0 (2026-09-14), keyless-first subset. Delivered: rates, curve, fedodds, cot, series (FRED key), fiscal, energy (Yahoo futures), macro brief. Deferred: calendar (needs a verified release-schedule source), sentiment (VIX term), full EIA energy + FRED-backed rates/prints. Providers: treasury (par-curve XML + FiscalData), cftc (Socrata legacy combined jun7-fc8e), kalshi (KXFED ladder -> per-band odds), fred (series). All keyless except series (FRED free key). 6 MCP tools added (fin_rates/fedodds/cot/series/fiscal/energy). Depends on PRD 00, PRD 01. Providers: fred, bls, eia, fed h15, treasury, fiscaldata, kalshi, polymarket, cftc, michigan, gdpnow.

## Purpose

The macro backdrop in one screen: rates, the curve, the latest inflation/jobs prints with dates, Fed-cut odds, and energy, without opening six sites. Keyless where possible; FRED/EIA/BLS keys unlock the rest.

## Commands

- `macro brief` *: fed funds, 2y/10y/30y + 2s10s spread, latest CPI/PPI/NFP/JOLTS/GDPNow prints with their dates, VIX, DXY, oil, gold, BTC, and Fed-cut odds. One screen, `--json` for the agent. Composes `rates`, `series`, `fedodds`, `energy`, and cboe VIX.
- `rates` *: current fed funds, SOFR, 3m/2y/10y/30y from FRED (DGS-series) or Fed H.15 CSV when no FRED key.
- `curve [--date YYYY-MM-DD]`: full Treasury par curve (Treasury XML) with 2s10s/3m10y spreads; historical with `--date`.
- `series <FRED_ID>` *: any FRED series, last N observations + change. `--id CPIAUCSL --n 12`.
- `calendar --week` *: this week's macro releases (dates + consensus where available; BLS schedule + FRED release dates).
- `fedodds` *: implied probability of the next FOMC decision from Kalshi (KXFED) and Polymarket (tag fed), shown side by side. Optionally ZQ futures via Yahoo.
- `cot --market ES|NQ|GC|CL`: CFTC Commitments of Traders positioning (net non-commercial), latest vs 4w ago, from the Socrata resources in PRD 01.
- `sentiment`: Michigan ICS, put/call ratio (cboe), VIX term (front vs 3m). AAII skipped (bot wall).
- `energy`: EIA WTI/Brent/Henry Hub spot + weekly inventories; rig count via EIA weekly.
- `fiscal`: FiscalData avg interest on the debt, latest auction results, debt to the penny.

## Data model

`MacroPrint{Name, Value, Unit, AsOf, Prior, Change}`. `RatePoint{Tenor, Yield, AsOf}`. `FedOdds{Meeting, Source, Cut25Pct, Cut50Pct, HoldPct, AsOf}` (one row per source). `Series{ID, Title, Points []Obs}`.

## Output

`macro brief` is a fixed dashboard: three short blocks (Rates / Prints / Risk) plus a Fed-odds line, tabwriter-aligned. `series`/`rates`/`cot` are tables. `--json` on any returns the model.

## MCP tools

`fin_macro_brief`, `fin_rates`, `fin_series`, `fin_fedodds`, `fin_cot`, `fin_energy`, `fin_calendar`. Read-only.

## Keys and fallbacks

- No FRED key: `rates`/`series`/`macro brief` degrade to Fed H.15 CSV (rates only) and print a one-line "set FINCTL_FRED_KEY for full macro" hint. `series <arbitrary id>` requires the key.
- No EIA key: `energy` errors with the register hint; `macro brief` shows oil/gold via Yahoo futures (`CL=F`, `GC=F`) instead.
- No BLS key: `calendar` uses the FRED release calendar; CPI/PPI prints still come from FRED series.
- `finctl keys` lists which of FRED/EIA/BLS/Finnhub are configured.

## Acceptance criteria

1. `finctl macro brief` renders every block with as-of dates; a missing key degrades one block, never blanks the screen.
2. `finctl fedodds` shows Kalshi and Polymarket side by side with the meeting date.
3. `finctl series DGS10 --n 5` matches FRED's last five observations.
4. `finctl cot --market ES` returns the latest weekly report date from CFTC.
5. Each provider has a fixture test; no network in tests.
