# finctl PRD 04: Supply-chain edge commands

Status: draft, 2026-09-14. Depends on PRD 00, PRD 01. Providers: edgar (capex), twse, siliconanalysts, vastai. These are the differentiated, high-signal reads the Serenity thesis keys off; none of them are in a normal terminal.

## Purpose

Turn the AI/semiconductor supply chain into commands: hyperscaler capex, Taiwan ODM revenue, wafer/HBM pricing, and GPU rental spot. Filing-accurate or official-stat where possible, all free, all cached and trended in SQLite.

## Commands

- `capex` *: quarterly capex for MSFT, GOOGL, AMZN, META, ORCL from SEC XBRL (`PaymentsToAcquirePropertyPlantAndEquipment`), with QoQ and YoY, and a combined hyperscaler total. `--since 2023`. The master-thesis input. `--json` = per-company series.
- `tw-revenue [2330 2454 6669 ...]` *: monthly revenue + MoM/YoY for TSMC, MediaTek, Wiwynn, Quanta, Hon Hai, ASE from TWSE OpenAPI `t187ap05_L`. Default set = the AI-server ODM basket. One JSON pull, filtered by company id. AI-server demand read in one command.
- `silicon` *: wafer price by node, HBM/DRAM pricing, advanced-packaging from SiliconAnalysts market-data (3 latest points per series, with citations), plus market-pulse headlines (category + impact). `--json` = series + citations.
- `gpu-rent` *: H100 SXM and B200 lowest ask + median from vast.ai bundles (keyless), stored each run so `--trend` shows the spot-price curve over time. Compute-cost proxy.

## Data model

`CapexPoint{Company, FiscalPeriod, Capex, QoQ, YoY}`. `MonthlyRevenue{CompanyID, Name, Month, RevenueTWD, MoM, YoY}`. `PriceSeries{Series, Points []{Date, Value, Unit}, Citations []string}`. `GPURent{GPU, LowAsk, Median, AsOf}`.

## Trending

`gpu-rent` and optionally `silicon` append each fetch to a `history` table (`series, ts, value`) so `--trend` renders a sparkline over stored points. This is the one place finctl stores time series itself rather than caching a single latest response.

## Output

Tables with YoY/MoM as signed percents; `capex` shows the combined hyperscaler total row. `silicon` prints each series with its citation footnote. `--json` everywhere.

## MCP tools

`fin_capex`, `fin_tw_revenue`, `fin_silicon`, `fin_gpu_rent`. Read-only.

## Edge cases

- TWSE `t187ap05_L` is ~600 KB; cache 24h and index by company id in memory for the filtered view.
- SiliconAnalysts free tier is 3 points per series; state that in the output so a reader does not mistake it for a full history.
- vast.ai bundle availability swings intraday; `gpu-rent` reports the lowest ask and the median of the current offer set, with the fetch timestamp.
- Hyperscaler capex tag varies; reuse the edgar tag-list fallback from PRD 02.

## Out of scope for v1 (documented for later)

Korea 20-day semiconductor exports (JS portal / Korea Customs PDF / data.go.kr free key), Japan chip-equipment exports (e-Stat free key), DRAM/NAND spot beyond SiliconAnalysts (TrendForce/KITA). Each needs a scrape or a key; parked in PRD 01 TBD.

## Acceptance criteria

1. `finctl capex` shows the five hyperscalers' latest quarter capex + YoY and a combined total, from XBRL.
2. `finctl tw-revenue` defaults to the ODM basket and matches TWSE's latest month (TSMC 2330 sanity check).
3. `finctl gpu-rent --trend` shows a sparkline after several stored runs.
4. Provider fixture tests; no network in tests.
